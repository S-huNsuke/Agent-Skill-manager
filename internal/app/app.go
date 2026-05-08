package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/agents/claudecode"
	"github.com/caojun/agent-skills-manager/internal/agents/codex"
	"github.com/caojun/agent-skills-manager/internal/agents/geminicli"
	"github.com/caojun/agent-skills-manager/internal/agents/openclaw"
	"github.com/caojun/agent-skills-manager/internal/ai"
	"github.com/caojun/agent-skills-manager/internal/storage/sqlite"
)

const (
	defaultAppName    = "Agent Skills Manager"
	defaultAppVersion = "0.1.0"
)

type AgentRegistry interface {
	DiscoverAll(ctx context.Context) ([]agents.AgentInstall, error)
	ListInstalledSkills(ctx context.Context, install agents.AgentInstall) ([]string, error)
	AdapterFor(agentID string) (agents.Adapter, bool)
	InstallSkill(ctx context.Context, install agents.AgentInstall, mutation agents.SkillMutation) error
	UninstallSkill(ctx context.Context, install agents.AgentInstall, skillName string) error
	UpdateSkill(ctx context.Context, install agents.AgentInstall, mutation agents.SkillMutation) error
	ValidateSkillInstall(ctx context.Context, install agents.AgentInstall, skillName string) error
}

type App struct {
	Name      string
	Version   string
	Bootstrap BootstrapConfig
	logger    *slog.Logger
	startedAt time.Time
	ctx       context.Context
	registry  AgentRegistry

	db               *sql.DB
	projectsRepo     *sqlite.ProjectRepository
	skillGroupsRepo  *sqlite.SkillGroupRepository
	catalogSrcRepo   *sqlite.CatalogSourceRepository
	catalogSkillRepo *sqlite.CatalogSkillRepository
	settingsRepo     *sqlite.SettingsRepository
	taskRepo         *sqlite.TaskRepository
	bridge           ai.Bridge

	catalogMu      sync.RWMutex
	catalogSources []CatalogSourceViewModel
	catalogItems   []StoreItemViewModel

	projectsMu  sync.RWMutex
	projects    []ProjectViewModel
	skillGroups []SkillGroupViewModel

	assistantMu sync.Mutex
	activeTask  *AssistantTaskViewModel

	scheduler *Scheduler
}

type AppInfo struct {
	Name               string `json:"name"`
	Version            string `json:"version"`
	FrontendReady      bool   `json:"frontendReady"`
	FrontendDistDir    string `json:"frontendDistDir"`
	WailsConfigPath    string `json:"wailsConfigPath"`
	UsesEmbeddedAssets bool   `json:"usesEmbeddedAssets"`
	StartedAtRFC3339   string `json:"startedAt"`
}

/** 创建应用实例，初始化所有代理适配器和 SQLite 持久化 */
func New(repoRoot string, log *slog.Logger) (*App, error) {
	bootstrap, err := LoadBootstrapConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	registry := agents.NewRegistry(
		codex.NewAdapter(codex.Config{}),
		claudecode.NewAdapter(claudecode.Config{}),
		geminicli.NewAdapter(geminicli.Config{}),
		openclaw.NewAdapter(openclaw.Config{}),
	)

	db, err := initDatabase()
	if err != nil {
		if log != nil {
			log.Warn("sqlite init failed, running without persistence", slog.Any("error", err))
		}
		db = nil
	}

	app := &App{
		Name:      defaultAppName,
		Version:   defaultAppVersion,
		Bootstrap: bootstrap,
		logger:    log,
		registry:  registry,
		db:        db,
	}

	if db != nil {
		app.projectsRepo = sqlite.NewProjectRepository(db)
		app.skillGroupsRepo = sqlite.NewSkillGroupRepository(db)
		app.catalogSrcRepo = sqlite.NewCatalogSourceRepository(db)
		app.catalogSkillRepo = sqlite.NewCatalogSkillRepository(db)
		app.settingsRepo = sqlite.NewSettingsRepository(db)
		app.taskRepo = sqlite.NewTaskRepository(db)

		app.loadFromDatabase()
	} else {
		app.catalogSources = defaultCatalogSources()
		app.catalogItems = make([]StoreItemViewModel, 0)
		app.projects = scanLocalProjects()
		app.skillGroups = make([]SkillGroupViewModel, 0)
	}

	// 创建 AI Bridge，设置正确的工作目录
	bridge := ai.NewLocalBridge("python3", "none", "")
	// 设置 Python worker 的工作目录为 python，这样可以使用 `-m worker.main`
	workerDir := filepath.Join(bootstrap.RepoRoot, "python")
	bridge.WorkerDir = workerDir
	aiSettings := app.GetAISettings()
	bridge.UpdateConfig(aiSettings.Provider, aiSettings.Model, aiSettings.APIKey, aiSettings.BaseURL)
	app.bridge = bridge

	return app, nil
}

/** 初始化 SQLite 数据库连接和迁移 */
func initDatabase() (*sql.DB, error) {
	dataDir, err := ensureDataDir()
	if err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "data.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := sqlite.Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

/** 确保数据目录存在 */
func ensureDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir := filepath.Join(home, "Library", "Application Support", "agent-skills-manager")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	return dataDir, nil
}

/** 从数据库加载已有数据到内存缓存 */
func (a *App) loadFromDatabase() {
	ctx := context.Background()

	sources, err := a.catalogSrcRepo.List(ctx)
	if err != nil || len(sources) == 0 {
		a.catalogSources = defaultCatalogSources()
		for _, src := range a.catalogSources {
			_ = a.catalogSrcRepo.Put(ctx, catalogSourceVMToDomain(src))
		}
	} else {
		a.catalogSources = make([]CatalogSourceViewModel, 0, len(sources))
		for _, src := range sources {
			a.catalogSources = append(a.catalogSources, catalogSourceDomainToVM(src))
		}
	}

	skills, err := a.catalogSkillRepo.ListAll(ctx)
	if err == nil {
		a.catalogItems = make([]StoreItemViewModel, 0, len(skills))
		for _, s := range skills {
			a.catalogItems = append(a.catalogItems, catalogSkillDomainToVM(s))
		}
	} else {
		a.catalogItems = make([]StoreItemViewModel, 0)
	}

	projects, err := a.projectsRepo.List(ctx)
	if err == nil && len(projects) > 0 {
		a.projects = make([]ProjectViewModel, 0, len(projects))
		for _, p := range projects {
			a.projects = append(a.projects, projectDomainToVM(p))
		}
	} else {
		a.projects = scanLocalProjects()
		for _, p := range a.projects {
			_ = a.projectsRepo.Put(ctx, projectVMToDomain(p))
		}
	}

	groups, err := a.skillGroupsRepo.List(ctx)
	if err == nil {
		a.skillGroups = make([]SkillGroupViewModel, 0, len(groups))
		for _, g := range groups {
			vm := skillGroupDomainToVM(g)

			skillLinks, linkErr := a.skillGroupsRepo.ListSkillGroupSkills(ctx, g.ID)
			if linkErr == nil {
				names := make([]string, 0, len(skillLinks))
				for _, link := range skillLinks {
					names = append(names, link.SkillID)
				}
				vm.SkillNames = names
				vm.SkillCount = len(names)
			}

			a.skillGroups = append(a.skillGroups, vm)
		}
	} else {
		a.skillGroups = make([]SkillGroupViewModel, 0)
	}

	// 不再自动恢复 AI 助手任务，避免显示已完成的旧任务
	// AI 助手任务应该是临时的，每次启动都从空闲状态开始
}

/** 默认内置商店源 */
func defaultCatalogSources() []CatalogSourceViewModel {
	return []CatalogSourceViewModel{
		{
			ID:             "anthropics-skills",
			Name:           "Anthropic 官方技能",
			URL:            "https://github.com/anthropics/skills",
			IsBuiltin:      true,
			Enabled:        true,
			LastSyncedAt:   "",
			LastSyncStatus: "",
			SkillCount:     0,
		},
		{
			ID:             "composio-awesome-skills",
			Name:           "Awesome Claude Skills",
			URL:            "https://github.com/ComposioHQ/awesome-claude-skills",
			IsBuiltin:      true,
			Enabled:        true,
			LastSyncedAt:   "",
			LastSyncStatus: "",
			SkillCount:     0,
		},
		{
			ID:             "vercel-labs-skills",
			Name:           "Vercel Skills",
			URL:            "https://github.com/vercel-labs/skills",
			IsBuiltin:      true,
			Enabled:        true,
			LastSyncedAt:   "",
			LastSyncStatus: "",
			SkillCount:     0,
		},
	}
}

/** 扫描本地目录发现 Git 项目 */
func scanLocalProjects() []ProjectViewModel {
	home, err := os.UserHomeDir()
	if err != nil {
		return make([]ProjectViewModel, 0)
	}

	scanDirs := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "workspace"),
	}

	result := make([]ProjectViewModel, 0)
	for _, scanDir := range scanDirs {
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			gitDir := filepath.Join(scanDir, entry.Name(), ".git")
			if _, err := os.Stat(gitDir); err != nil {
				continue
			}

			projectPath := filepath.Join(scanDir, entry.Name())
			modTime := ""
			if info, statErr := os.Stat(projectPath); statErr == nil {
				modTime = info.ModTime().Format("2006-01-02")
			}

			result = append(result, ProjectViewModel{
				ID:              fmt.Sprintf("proj-%s", entry.Name()),
				Name:            entry.Name(),
				Path:            projectPath,
				Stage:           "已识别",
				BoundSkillGroup: "",
				BoundAgentID:    "",
				BoundAgentName:  "",
				SkillNames:      make([]string, 0),
				Summary:         fmt.Sprintf("路径: %s", projectPath),
				Needs:           make([]string, 0),
				LocalAgents:     make([]string, 0),
				Recent:          []string{fmt.Sprintf("最近修改: %s", modTime)},
				CreatedAt:       modTime,
			})
		}
	}

	return result
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.startedAt = time.Now().UTC()

	a.scheduler = NewScheduler(a)
	autoSettings := a.GetAutomationSettings()
	a.scheduler.ApplySettings(autoSettings)

	if a.logger != nil {
		a.logger.Info("desktop app startup complete", slog.String("frontendDistDir", a.Bootstrap.FrontendDistDir))
	}
}

func (a *App) Shutdown(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.StopAll()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.logger != nil {
		a.logger.Info("desktop app shutdown", slog.Bool("hadContext", ctx != nil || a.ctx != nil))
	}
}

func (a *App) GetAppInfo() AppInfo {
	started := ""
	if !a.startedAt.IsZero() {
		started = a.startedAt.Format(time.RFC3339)
	}

	return AppInfo{
		Name:               a.Name,
		Version:            a.Version,
		FrontendReady:      a.Bootstrap.FrontendDistDir != "",
		FrontendDistDir:    a.Bootstrap.FrontendDistDir,
		WailsConfigPath:    a.Bootstrap.WailsConfigPath,
		UsesEmbeddedAssets: a.Bootstrap.UsesEmbeddedAssets,
		StartedAtRFC3339:   started,
	}
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/caojun/agent-skills-manager/internal/ai"
	"github.com/caojun/agent-skills-manager/internal/domain"
)

func (a *App) GetStoreItems() []StoreItemViewModel {
	a.catalogMu.RLock()
	defer a.catalogMu.RUnlock()

	result := make([]StoreItemViewModel, 0, len(a.catalogItems))
	result = append(result, a.catalogItems...)
	return result
}

/** 返回项目列表 */
func (a *App) GetCatalogSources() []CatalogSourceViewModel {
	a.catalogMu.RLock()
	defer a.catalogMu.RUnlock()

	result := make([]CatalogSourceViewModel, 0, len(a.catalogSources))
	result = append(result, a.catalogSources...)
	return result
}

/** 同步商店源，从 GitHub 仓库获取技能列表，耗时操作在锁外执行 */
func (a *App) SyncCatalogSource(sourceID string) SyncResultViewModel {
	a.catalogMu.RLock()
	var target *CatalogSourceViewModel
	var targetCopy CatalogSourceViewModel
	for i := range a.catalogSources {
		if a.catalogSources[i].ID == sourceID {
			target = &a.catalogSources[i]
			targetCopy = a.catalogSources[i]
			break
		}
	}

	existingMap := make(map[string]bool)
	if target != nil {
		for _, item := range a.catalogItems {
			if item.Source == target.Name {
				existingMap[item.Name] = true
			}
		}
	}
	a.catalogMu.RUnlock()

	if target == nil {
		return SyncResultViewModel{
			SourceID:      sourceID,
			Success:       false,
			NewSkills:     0,
			UpdatedSkills: 0,
			Errors:        []string{"来源不存在"},
		}
	}

	skills, err := fetchGitHubSkills(targetCopy.URL)
	if err != nil {
		a.catalogMu.Lock()
		for i := range a.catalogSources {
			if a.catalogSources[i].ID == sourceID {
				a.catalogSources[i].LastSyncStatus = "failed"
				if a.catalogSrcRepo != nil {
					_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(a.catalogSources[i]))
				}
				break
			}
		}
		a.catalogMu.Unlock()
		a.logger.Error("商店同步失败", "sourceID", sourceID, "url", targetCopy.URL, "error", err)

		return SyncResultViewModel{
			SourceID:      sourceID,
			Success:       false,
			NewSkills:     0,
			UpdatedSkills: 0,
			Errors:        []string{translateSyncError(err)},
		}
	}

	type skillInstallInfo struct {
		skill     gitHubSkill
		installed bool
	}

	skillInfos := make([]skillInstallInfo, 0, len(skills))
	for _, skill := range skills {
		installed := a.isSkillInstalled(skill.Name)
		skillInfos = append(skillInfos, skillInstallInfo{skill: skill, installed: installed})
	}

	now := time.Now().Format("2006-01-02 15:04")

	a.catalogMu.Lock()
	for i := range a.catalogSources {
		if a.catalogSources[i].ID == sourceID {
			a.catalogSources[i].LastSyncedAt = now
			a.catalogSources[i].LastSyncStatus = "success"
			a.catalogSources[i].SkillCount = len(skills)
			break
		}
	}

	newCount := 0
	updatedCount := 0
	for _, info := range skillInfos {
		status := "available"
		if info.installed {
			status = "installed"
		}
		nextItem := StoreItemViewModel{
			ID:             fmt.Sprintf("%s-%s", sourceID, info.skill.Name),
			Name:           info.skill.Name,
			Author:         info.skill.Author,
			Source:         targetCopy.Name,
			Status:         status,
			Summary:        info.skill.Description,
			Installs:       fmt.Sprintf("来自 %s", targetCopy.Name),
			Impact:         "技能将安装到指定代理的技能目录",
			Compatibility:  info.skill.SupportedAgents,
			Homepage:       info.skill.Homepage,
			LocalCachePath: info.skill.CachePath,
		}

		if existingMap[info.skill.Name] {
			for i := range a.catalogItems {
				if a.catalogItems[i].Source == targetCopy.Name && a.catalogItems[i].Name == info.skill.Name {
					nextItem.Status = a.catalogItems[i].Status
					if info.installed {
						nextItem.Status = "installed"
					}
					if catalogItemChanged(a.catalogItems[i], nextItem) {
						updatedCount++
					}
					a.catalogItems[i] = nextItem
					break
				}
			}
			continue
		}

		newCount++
		a.catalogItems = append(a.catalogItems, nextItem)
	}

	if a.catalogSrcRepo != nil {
		for i := range a.catalogSources {
			if a.catalogSources[i].ID == sourceID {
				_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(a.catalogSources[i]))

				ctx := context.Background()
				domainSkills := make([]domain.CatalogSkill, 0, len(skills))
				for _, skill := range skills {
					domainSkills = append(domainSkills, domain.CatalogSkill{
						ID:              fmt.Sprintf("%s-%s", sourceID, skill.Name),
						SourceID:        sourceID,
						Name:            skill.Name,
						Version:         "latest",
						Author:          skill.Author,
						Description:     skill.Description,
						Homepage:        skill.Homepage,
						SupportedAgents: skill.SupportedAgents,
					})
				}
				_ = a.catalogSkillRepo.ReplaceBySource(ctx, sourceID, domainSkills)
				break
			}
		}
	}
	a.catalogMu.Unlock()

	return SyncResultViewModel{
		SourceID:      sourceID,
		Success:       true,
		NewSkills:     newCount,
		UpdatedSkills: updatedCount,
		Errors:        make([]string, 0),
	}
}

/** 同步所有已启用的商店源 */
func (a *App) SyncAllSources() []SyncResultViewModel {
	a.catalogMu.RLock()
	sourceIDs := make([]string, 0)
	for _, src := range a.catalogSources {
		if src.Enabled {
			sourceIDs = append(sourceIDs, src.ID)
		}
	}
	a.catalogMu.RUnlock()

	results := make([]SyncResultViewModel, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		results = append(results, a.SyncCatalogSource(id))
	}
	return results
}

/** 判断商店条目是否发生用户可见变化 */
func catalogItemChanged(current StoreItemViewModel, next StoreItemViewModel) bool {
	return current.ID != next.ID ||
		current.Name != next.Name ||
		current.Author != next.Author ||
		current.Source != next.Source ||
		current.Status != next.Status ||
		current.Summary != next.Summary ||
		current.Installs != next.Installs ||
		current.Impact != next.Impact ||
		current.Homepage != next.Homepage ||
		current.LocalCachePath != next.LocalCachePath ||
		strings.Join(current.Compatibility, "\x00") != strings.Join(next.Compatibility, "\x00")
}

/** 添加自定义商店源 */
func (a *App) AddCatalogSource(name string, url string) CatalogSourceViewModel {
	a.catalogMu.Lock()
	defer a.catalogMu.Unlock()

	id := fmt.Sprintf("custom-%d", time.Now().UnixMilli())
	source := CatalogSourceViewModel{
		ID:             id,
		Name:           name,
		URL:            url,
		IsBuiltin:      false,
		Enabled:        true,
		LastSyncedAt:   "",
		LastSyncStatus: "",
		SkillCount:     0,
	}
	a.catalogSources = append(a.catalogSources, source)

	if a.catalogSrcRepo != nil {
		_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(source))
	}

	return source
}

/** 移除商店源（内置源不可移除） */
func (a *App) RemoveCatalogSource(sourceID string) string {
	a.catalogMu.Lock()
	defer a.catalogMu.Unlock()

	for i, src := range a.catalogSources {
		if src.ID == sourceID {
			if src.IsBuiltin {
				return "error: 内置商店源不可移除"
			}
			a.catalogSources = append(a.catalogSources[:i], a.catalogSources[i+1:]...)

			filtered := make([]StoreItemViewModel, 0)
			for _, item := range a.catalogItems {
				if item.Source != src.Name {
					filtered = append(filtered, item)
				}
			}
			a.catalogItems = filtered

			if a.catalogSrcRepo != nil {
				_ = a.catalogSkillRepo.DeleteBySource(context.Background(), sourceID)
				_ = a.catalogSrcRepo.Delete(context.Background(), sourceID)
			}

			return "ok"
		}
	}
	return "error: 商店源不存在"
}

/** 检查技能是否已安装 */
func (a *App) isSkillInstalled(skillName string) bool {
	if a.registry == nil {
		return false
	}
	installs := a.getCachedInstalls()
	for _, install := range installs {
		skillNames, err := a.registry.ListInstalledSkills(context.Background(), install)
		if err != nil {
			continue
		}
		for _, name := range skillNames {
			if name == skillName {
				return true
			}
		}
	}
	return false
}

/** 解释商店技能的用途，从远程仓库获取 README 内容 */
func (a *App) ExplainStoreSkill(sourceName string, skillName string) SkillExplanationViewModel {
	result := SkillExplanationViewModel{
		AgentID:   "store",
		SkillName: skillName,
		Found:     false,
		Files:     make([]string, 0),
	}

	a.catalogMu.RLock()
	var targetItem *StoreItemViewModel
	for i := range a.catalogItems {
		if a.catalogItems[i].Source == sourceName && a.catalogItems[i].Name == skillName {
			targetItem = &a.catalogItems[i]
			break
		}
	}
	a.catalogMu.RUnlock()

	if targetItem == nil {
		return result
	}

	result.Found = true
	result.AgentName = sourceName
	result.SkillPath = targetItem.Homepage

	if targetItem.Homepage != "" {
		repo := parseGitHubRepo(targetItem.Homepage)
		if repo != "" {
			branches := []string{"main", "master"}
			filenames := []string{"SKILL.md", "README.md", "readme.md"}
		outer:
			for _, branch := range branches {
				for _, filename := range filenames {
					rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, filename)
					if content, err := fetchRawContent(rawURL); err == nil && content != "" {
						if len(content) > 8000 {
							content = content[:8000] + "\n...(内容过长，已截断)"
						}
						result.ReadmeContent = content
						result.ReadmeFile = filename
						break outer
					}
				}
			}
		}
	}

	if result.ReadmeContent == "" {
		result.ReadmeContent = targetItem.Summary
		result.ReadmeFile = "summary"
	}

	// 调用 AI 生成通俗解释
	cacheKey := fmt.Sprintf("store:%s:%s", sourceName, skillName)
	a.explainCacheMu.RLock()
	if cached, ok := a.explainCache[cacheKey]; ok {
		result.AiExplanation = cached
		a.explainCacheMu.RUnlock()
		return result
	}
	a.explainCacheMu.RUnlock()

	if a.bridge != nil && result.ReadmeContent != "" {
		content := result.ReadmeContent
		if len(content) > 2000 {
			content = content[:2000] + "\n...(已截断)"
		}

		prompt := fmt.Sprintf(
			"根据下面的技能文档，用1-3句话解释这个技能能帮用户做什么。语言口语化、具体，避免技术术语。直接说用途，举一个使用场景，不超过80字。\n\n技能名称：%s\n\n文档：\n%s",
			skillName, content,
		)
		ctx := context.Background()
		resp, err := a.bridge.Run(ctx, ai.WorkerRequest{
			Action: "chat",
			Payload: map[string]any{
				"message": prompt,
				"history": []any{},
			},
		})
		if err == nil && resp.Status == "ok" {
			if reply, ok := resp.Data["reply"].(string); ok && reply != "" {
				result.AiExplanation = reply
				a.explainCacheMu.Lock()
				a.explainCache[cacheKey] = reply
				a.explainCacheMu.Unlock()
			}
		}
	}

	return result
}

/** 获取远程文件内容 */
func fetchRawContent(rawURL string) (string, error) {
	resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type gitHubSkill struct {
	Name            string
	Author          string
	Description     string
	Homepage        string
	SupportedAgents []string
	CachePath       string
}

type gitHubContentEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

/** 从 GitHub 仓库获取技能列表，自动检测仓库类型 */
func fetchGitHubSkills(repoURL string) ([]gitHubSkill, error) {
	repo := parseGitHubRepo(repoURL)
	if repo == "" {
		return nil, fmt.Errorf("invalid GitHub URL: %s", repoURL)
	}

	skills, err := fetchSkillsFromDirectory(repo)
	if err == nil && len(skills) > 0 {
		return skills, nil
	}

	skills, err = fetchSkillsFromReadme(repo)
	if err == nil && len(skills) > 0 {
		return skills, nil
	}

	if err != nil {
		return nil, err
	}

	return make([]gitHubSkill, 0), nil
}

/** 从仓库的 skills/ 目录获取技能列表 */
func fetchSkillsFromDirectory(repo string) ([]gitHubSkill, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/skills", repo)
	resp, err := httpGetWithTimeout(apiURL, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills directory: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API 速率限制，请稍后重试")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("no skills/ directory (status %d)", resp.StatusCode)
	}

	var entries []gitHubContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse directory listing: %v", err)
	}

	skills := make([]gitHubSkill, 0)
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		skill := gitHubSkill{
			Name:            entry.Name,
			Author:          repo,
			Description:     fmt.Sprintf("来自 %s 的技能", entry.Name),
			Homepage:        fmt.Sprintf("https://github.com/%s/tree/main/skills/%s", repo, entry.Name),
			SupportedAgents: []string{"Claude Code", "Codex", "Trae"},
		}

		desc := fetchSkillDescription(repo, entry.Name)
		if desc != "" {
			skill.Description = desc
		}

		cachePath := cacheSkillFiles(repo, entry.Name)
		if cachePath != "" {
			skill.CachePath = cachePath
		}

		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

/** 获取技能描述，依次尝试多个分支和文件名 */
func fetchSkillDescription(repo string, skillName string) string {
	branches := []string{"main", "master"}
	filenames := []string{"SKILL.md", "README.md", "readme.md"}

	for _, branch := range branches {
		for _, filename := range filenames {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, filename)
			if desc, err := fetchReadmeDescription(rawURL); err == nil && desc != "" {
				return desc
			}
		}
	}
	return ""
}

/** 从仓库 README 解析 awesome-list 格式的技能列表，自动尝试多个分支名 */
func fetchSkillsFromReadme(repo string) ([]gitHubSkill, error) {
	branches := []string{"main", "master"}
	filenames := []string{"README.md", "readme.md", "Readme.md"}

	for _, branch := range branches {
		for _, filename := range filenames {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, branch, filename)
			resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
			if err != nil {
				continue
			}

			if resp.StatusCode == 200 {
				body, readErr := io.ReadAll(io.LimitReader(resp.Body, 65536))
				resp.Body.Close()
				if readErr != nil {
					continue
				}

				skills := parseAwesomeListReadme(string(body), repo)
				if len(skills) > 0 {
					return skills, nil
				}
				continue
			}

			resp.Body.Close()
		}
	}

	return nil, fmt.Errorf("no README found in %s (tried main/master branches)", repo)
}

/** 解析 awesome-list 格式的 README，提取技能条目 */
func parseAwesomeListReadme(content string, repo string) []gitHubSkill {
	skills := make([]gitHubSkill, 0)
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
			continue
		}

		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimSpace(trimmed)

		name, desc, link := parseListItem(trimmed)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		homepage := link
		if homepage == "" {
			homepage = fmt.Sprintf("https://github.com/%s", repo)
		}

		if desc == "" {
			desc = fmt.Sprintf("来自 %s 的技能", repo)
		}

		supportedAgents := []string{"Claude Code"}
		if strings.Contains(strings.ToLower(name), "codex") || strings.Contains(strings.ToLower(desc), "codex") {
			supportedAgents = append(supportedAgents, "Codex")
		}
		if strings.Contains(strings.ToLower(name), "trae") || strings.Contains(strings.ToLower(desc), "trae") {
			supportedAgents = append(supportedAgents, "Trae")
		}
		if len(supportedAgents) == 1 {
			supportedAgents = []string{"Claude Code", "Codex", "Trae"}
		}

		skills = append(skills, gitHubSkill{
			Name:            name,
			Author:          repo,
			Description:     desc,
			Homepage:        homepage,
			SupportedAgents: supportedAgents,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}

/** 解析 Markdown 列表项，提取名称、描述和链接 */
func parseListItem(item string) (name string, desc string, link string) {
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := linkRegex.FindStringSubmatch(item)
	if len(matches) >= 3 {
		name = matches[1]
		link = matches[2]
		rest := strings.TrimSpace(item[len(matches[0]):])
		if strings.HasPrefix(rest, " - ") {
			desc = strings.TrimPrefix(rest, " - ")
		} else if strings.HasPrefix(rest, " – ") {
			desc = strings.TrimPrefix(rest, " – ")
		} else if strings.HasPrefix(rest, ": ") {
			desc = strings.TrimPrefix(rest, ": ")
		} else if rest != "" {
			desc = rest
		}
	} else {
		if strings.Contains(item, " - ") {
			parts := strings.SplitN(item, " - ", 2)
			name = strings.TrimSpace(parts[0])
			desc = strings.TrimSpace(parts[1])
		} else if strings.Contains(item, ": ") {
			parts := strings.SplitN(item, ": ", 2)
			name = strings.TrimSpace(parts[0])
			desc = strings.TrimSpace(parts[1])
		} else {
			name = item
		}
	}

	name = strings.TrimSpace(strings.Trim(name, "**"))
	desc = strings.TrimSpace(desc)
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return name, desc, link
}

/** 从 GitHub URL 解析仓库路径 (owner/repo) */
func parseGitHubRepo(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "https://github.com/") {
		path := strings.TrimPrefix(url, "https://github.com/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return ""
}

/** 从 README/SKILL.md 获取技能描述（取第一段非标题文本） */
func fetchReadmeDescription(rawURL string) (string, error) {
	resp, err := httpGetWithTimeout(rawURL, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(body), "\n")
	var descLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(descLines) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		descLines = append(descLines, trimmed)
		if len(descLines) >= 3 {
			break
		}
	}

	if len(descLines) == 0 {
		return "", fmt.Errorf("no description found")
	}

	return strings.Join(descLines, " "), nil
}

/** 带超时的 HTTP GET 请求 */
func httpGetWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	return client.Do(req)
}

/** 获取技能缓存目录路径 */
func getSkillCacheDir(repo string, skillName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	safeRepo := strings.ReplaceAll(repo, "/", "-")
	return filepath.Join(home, "Library", "Application Support", "agent-skills-manager", "skill-cache", safeRepo, skillName)
}

/** 从 GitHub 下载技能文件并缓存到本地目录，返回缓存路径 */
func cacheSkillFiles(repo string, skillName string) string {
	cacheDir := getSkillCacheDir(repo, skillName)
	if cacheDir == "" {
		return ""
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return ""
	}

	branches := []string{"main", "master"}
	var dirEntries []gitHubContentEntry

	for _, branch := range branches {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/skills/%s?ref=%s", repo, skillName, branch)
		resp, err := httpGetWithTimeout(apiURL, 10*time.Second)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&dirEntries); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		break
	}

	if len(dirEntries) == 0 {
		return ""
	}

	for _, entry := range dirEntries {
		if entry.Type == "dir" {
			continue
		}

		for _, branch := range branches {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, entry.Name)
			resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
			if err != nil {
				continue
			}
			if resp.StatusCode != 200 {
				resp.Body.Close()
				continue
			}

			filePath := filepath.Join(cacheDir, entry.Name)
			f, err := os.Create(filePath)
			if err != nil {
				resp.Body.Close()
				break
			}
			if _, err := io.Copy(f, io.LimitReader(resp.Body, 512*1024)); err != nil {
				_ = err // write failure results in an incomplete cache file; non-fatal
			}
			f.Close()
			resp.Body.Close()
			break
		}
	}

	return cacheDir
}

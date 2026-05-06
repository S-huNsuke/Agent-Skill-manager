package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

/** 插入或更新项目 */
func (r *ProjectRepository) Put(ctx context.Context, project domain.Project) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO projects (
			id, name, type, path, description, skill_group_id, auto_apply_enabled, bound_agent_id, bound_agent_name, last_active_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			path = excluded.path,
			description = excluded.description,
			skill_group_id = excluded.skill_group_id,
			auto_apply_enabled = excluded.auto_apply_enabled,
			bound_agent_id = excluded.bound_agent_id,
			bound_agent_name = excluded.bound_agent_name,
			last_active_at = excluded.last_active_at,
			updated_at = excluded.updated_at`,
		project.ID,
		project.Name,
		project.Type,
		project.Path,
		project.Description,
		nullableString(project.SkillGroupID),
		boolToInt(project.AutoApplyEnabled),
		project.BoundAgentID,
		project.BoundAgentName,
		formatNullableTime(project.LastActiveAt),
		project.CreatedAt.UTC().Format(time.RFC3339Nano),
		project.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("put project %s: %w", project.ID, err)
	}

	return nil
}

/** 按 ID 查询项目 */
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (domain.Project, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, name, type, path, description, skill_group_id, auto_apply_enabled, bound_agent_id, bound_agent_name, last_active_at, created_at, updated_at
		FROM projects WHERE id = ?`,
		id,
	)

	return scanProject(row)
}

/** 列出所有项目 */
func (r *ProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, name, type, path, description, skill_group_id, auto_apply_enabled, bound_agent_id, bound_agent_name, last_active_at, created_at, updated_at
		FROM projects ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var project domain.Project
		var skillGroupID sql.NullString
		var autoApplyEnabled int
		var boundAgentID string
		var boundAgentName string
		var lastActiveAt sql.NullString
		var createdAt string
		var updatedAt string

		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Type,
			&project.Path,
			&project.Description,
			&skillGroupID,
			&autoApplyEnabled,
			&boundAgentID,
			&boundAgentName,
			&lastActiveAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}

		project.SkillGroupID = skillGroupID.String
		project.AutoApplyEnabled = autoApplyEnabled != 0
		project.BoundAgentID = boundAgentID
		project.BoundAgentName = boundAgentName

		project.LastActiveAt, err = parseNullableTime(lastActiveAt)
		if err != nil {
			return nil, fmt.Errorf("parse project %s last_active_at: %w", project.ID, err)
		}
		project.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse project %s created_at: %w", project.ID, err)
		}
		project.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse project %s updated_at: %w", project.ID, err)
		}

		projects = append(projects, project)
	}

	return projects, nil
}

/** 删除项目 */
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project %s: %w", id, err)
	}
	return nil
}

/** 扫描 project 行到 domain.Project */
func scanProject(row *sql.Row) (domain.Project, error) {
	var project domain.Project
	var skillGroupID sql.NullString
	var autoApplyEnabled int
	var boundAgentID string
	var boundAgentName string
	var lastActiveAt sql.NullString
	var createdAt string
	var updatedAt string

	if err := row.Scan(
		&project.ID,
		&project.Name,
		&project.Type,
		&project.Path,
		&project.Description,
		&skillGroupID,
		&autoApplyEnabled,
		&boundAgentID,
		&boundAgentName,
		&lastActiveAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Project{}, fmt.Errorf("scan project: %w", err)
	}

	project.SkillGroupID = skillGroupID.String
	project.AutoApplyEnabled = autoApplyEnabled != 0
	project.BoundAgentID = boundAgentID
	project.BoundAgentName = boundAgentName

	var err error
	project.LastActiveAt, err = parseNullableTime(lastActiveAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("parse project last_active_at: %w", err)
	}
	project.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("parse project created_at: %w", err)
	}
	project.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("parse project updated_at: %w", err)
	}

	return project, nil
}

// --- SkillGroup Repository ---

type SkillGroupRepository struct {
	db *sql.DB
}

func NewSkillGroupRepository(db *sql.DB) *SkillGroupRepository {
	return &SkillGroupRepository{db: db}
}

/** 插入或更新技能组 */
func (r *SkillGroupRepository) Put(ctx context.Context, group domain.SkillGroup) error {
	prefAgentsJSON, _ := json.Marshal(group.PreferredAgents)
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO skill_groups (
			id, name, description, source_type, version, goal_prompt, preferred_agents, bound_agent_id, bound_agent_name, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			source_type = excluded.source_type,
			version = excluded.version,
			goal_prompt = excluded.goal_prompt,
			preferred_agents = excluded.preferred_agents,
			bound_agent_id = excluded.bound_agent_id,
			bound_agent_name = excluded.bound_agent_name,
			updated_at = excluded.updated_at`,
		group.ID,
		group.Name,
		group.Description,
		group.SourceType,
		group.Version,
		group.GoalPrompt,
		string(prefAgentsJSON),
		group.BoundAgentID,
		group.BoundAgentName,
		group.CreatedAt.UTC().Format(time.RFC3339Nano),
		group.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("put skill group %s: %w", group.ID, err)
	}
	return nil
}

/** 按 ID 查询技能组 */
func (r *SkillGroupRepository) GetByID(ctx context.Context, id string) (domain.SkillGroup, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, name, description, source_type, version, goal_prompt, preferred_agents, bound_agent_id, bound_agent_name, created_at, updated_at
		FROM skill_groups WHERE id = ?`,
		id,
	)
	return scanSkillGroup(row)
}

/** 列出所有技能组 */
func (r *SkillGroupRepository) List(ctx context.Context) ([]domain.SkillGroup, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, name, description, source_type, version, goal_prompt, preferred_agents, bound_agent_id, bound_agent_name, created_at, updated_at
		FROM skill_groups ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list skill groups: %w", err)
	}
	defer rows.Close()

	var groups []domain.SkillGroup
	for rows.Next() {
		var group domain.SkillGroup
		var prefAgentsJSON string
		var boundAgentID string
		var boundAgentName string
		var createdAt string
		var updatedAt string

		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.SourceType,
			&group.Version,
			&group.GoalPrompt,
			&prefAgentsJSON,
			&boundAgentID,
			&boundAgentName,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan skill group: %w", err)
		}

		_ = json.Unmarshal([]byte(prefAgentsJSON), &group.PreferredAgents)
		group.BoundAgentID = boundAgentID
		group.BoundAgentName = boundAgentName
		group.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		group.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

		groups = append(groups, group)
	}

	return groups, nil
}

/** 删除技能组 */
func (r *SkillGroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM skill_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete skill group %s: %w", id, err)
	}
	return nil
}

/** 添加技能到技能组 */
func (r *SkillGroupRepository) PutSkillGroupSkill(ctx context.Context, sgs domain.SkillGroupSkill) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO skill_group_skills (skill_group_id, skill_id, required, priority, reason)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(skill_group_id, skill_id) DO UPDATE SET
			required = excluded.required,
			priority = excluded.priority,
			reason = excluded.reason`,
		sgs.SkillGroupID,
		sgs.SkillID,
		boolToInt(sgs.Required),
		sgs.Priority,
		sgs.Reason,
	)
	if err != nil {
		return fmt.Errorf("put skill group skill %s/%s: %w", sgs.SkillGroupID, sgs.SkillID, err)
	}
	return nil
}

/** 从技能组移除技能 */
func (r *SkillGroupRepository) DeleteSkillGroupSkill(ctx context.Context, groupID string, skillID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM skill_group_skills WHERE skill_group_id = ? AND skill_id = ?`, groupID, skillID)
	if err != nil {
		return fmt.Errorf("delete skill group skill %s/%s: %w", groupID, skillID, err)
	}
	return nil
}

/** 列出技能组的所有技能 */
func (r *SkillGroupRepository) ListSkillGroupSkills(ctx context.Context, groupID string) ([]domain.SkillGroupSkill, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT skill_group_id, skill_id, required, priority, reason
		FROM skill_group_skills WHERE skill_group_id = ? ORDER BY priority`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list skill group skills for %s: %w", groupID, err)
	}
	defer rows.Close()

	var skills []domain.SkillGroupSkill
	for rows.Next() {
		var sgs domain.SkillGroupSkill
		var required int
		if err := rows.Scan(&sgs.SkillGroupID, &sgs.SkillID, &required, &sgs.Priority, &sgs.Reason); err != nil {
			return nil, fmt.Errorf("scan skill group skill: %w", err)
		}
		sgs.Required = required != 0
		skills = append(skills, sgs)
	}

	return skills, nil
}

/** 扫描 skill_group 行到 domain.SkillGroup */
func scanSkillGroup(row *sql.Row) (domain.SkillGroup, error) {
	var group domain.SkillGroup
	var prefAgentsJSON string
	var boundAgentID string
	var boundAgentName string
	var createdAt string
	var updatedAt string

	if err := row.Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.SourceType,
		&group.Version,
		&group.GoalPrompt,
		&prefAgentsJSON,
		&boundAgentID,
		&boundAgentName,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.SkillGroup{}, fmt.Errorf("scan skill group: %w", err)
	}

	_ = json.Unmarshal([]byte(prefAgentsJSON), &group.PreferredAgents)
	group.BoundAgentID = boundAgentID
	group.BoundAgentName = boundAgentName
	group.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	group.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return group, nil
}

// --- CatalogSource Repository ---

type CatalogSourceRepository struct {
	db *sql.DB
}

func NewCatalogSourceRepository(db *sql.DB) *CatalogSourceRepository {
	return &CatalogSourceRepository{db: db}
}

/** 插入或更新商店源 */
func (r *CatalogSourceRepository) Put(ctx context.Context, source domain.CatalogSource) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO catalog_sources (
			id, name, url, is_builtin, enabled, last_synced_at, last_sync_status, last_sync_error, cache_expires_at, min_supported_client_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			url = excluded.url,
			is_builtin = excluded.is_builtin,
			enabled = excluded.enabled,
			last_synced_at = excluded.last_synced_at,
			last_sync_status = excluded.last_sync_status,
			last_sync_error = excluded.last_sync_error,
			cache_expires_at = excluded.cache_expires_at,
			min_supported_client_version = excluded.min_supported_client_version`,
		source.ID,
		source.Name,
		source.URL,
		boolToInt(source.IsBuiltin),
		boolToInt(source.Enabled),
		formatNullableTime(source.LastSyncedAt),
		source.LastSyncStatus,
		source.LastSyncError,
		formatNullableTime(source.CacheExpiresAt),
		source.MinSupportedClientVersion,
	)
	if err != nil {
		return fmt.Errorf("put catalog source %s: %w", source.ID, err)
	}
	return nil
}

/** 列出所有商店源 */
func (r *CatalogSourceRepository) List(ctx context.Context) ([]domain.CatalogSource, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, name, url, is_builtin, enabled, last_synced_at, last_sync_status, last_sync_error, cache_expires_at, min_supported_client_version
		FROM catalog_sources ORDER BY is_builtin DESC, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog sources: %w", err)
	}
	defer rows.Close()

	var sources []domain.CatalogSource
	for rows.Next() {
		var source domain.CatalogSource
		var isBuiltin int
		var enabled int
		var lastSyncedAt sql.NullString
		var cacheExpiresAt sql.NullString

		if err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.URL,
			&isBuiltin,
			&enabled,
			&lastSyncedAt,
			&source.LastSyncStatus,
			&source.LastSyncError,
			&cacheExpiresAt,
			&source.MinSupportedClientVersion,
		); err != nil {
			return nil, fmt.Errorf("scan catalog source: %w", err)
		}

		source.IsBuiltin = isBuiltin != 0
		source.Enabled = enabled != 0
		source.LastSyncedAt, _ = parseNullableTime(lastSyncedAt)
		source.CacheExpiresAt, _ = parseNullableTime(cacheExpiresAt)

		sources = append(sources, source)
	}

	return sources, nil
}

/** 删除商店源 */
func (r *CatalogSourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM catalog_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete catalog source %s: %w", id, err)
	}
	return nil
}

// --- CatalogSkill Repository ---

type CatalogSkillRepository struct {
	db *sql.DB
}

func NewCatalogSkillRepository(db *sql.DB) *CatalogSkillRepository {
	return &CatalogSkillRepository{db: db}
}

/** 插入或更新商店技能 */
func (r *CatalogSkillRepository) Put(ctx context.Context, skill domain.CatalogSkill) error {
	supportedAgentsJSON, _ := json.Marshal(skill.SupportedAgents)
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO catalog_skills (
			id, source_id, name, version, author, description, homepage, package_url, checksum_sha256, supported_agents, schema_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_id = excluded.source_id,
			name = excluded.name,
			version = excluded.version,
			author = excluded.author,
			description = excluded.description,
			homepage = excluded.homepage,
			package_url = excluded.package_url,
			checksum_sha256 = excluded.checksum_sha256,
			supported_agents = excluded.supported_agents,
			schema_version = excluded.schema_version`,
		skill.ID,
		skill.SourceID,
		skill.Name,
		skill.Version,
		skill.Author,
		skill.Description,
		skill.Homepage,
		skill.PackageURL,
		skill.ChecksumSHA256,
		string(supportedAgentsJSON),
		skill.SchemaVersion,
	)
	if err != nil {
		return fmt.Errorf("put catalog skill %s: %w", skill.ID, err)
	}
	return nil
}

/** 按来源列出技能 */
func (r *CatalogSkillRepository) ListBySource(ctx context.Context, sourceID string) ([]domain.CatalogSkill, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, source_id, name, version, author, description, homepage, package_url, checksum_sha256, supported_agents, schema_version
		FROM catalog_skills WHERE source_id = ? ORDER BY name`,
		sourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog skills by source %s: %w", sourceID, err)
	}
	defer rows.Close()

	return scanCatalogSkills(rows)
}

/** 列出所有商店技能 */
func (r *CatalogSkillRepository) ListAll(ctx context.Context) ([]domain.CatalogSkill, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, source_id, name, version, author, description, homepage, package_url, checksum_sha256, supported_agents, schema_version
		FROM catalog_skills ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all catalog skills: %w", err)
	}
	defer rows.Close()

	return scanCatalogSkills(rows)
}

/** 删除某来源的所有技能 */
func (r *CatalogSkillRepository) DeleteBySource(ctx context.Context, sourceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM catalog_skills WHERE source_id = ?`, sourceID)
	if err != nil {
		return fmt.Errorf("delete catalog skills by source %s: %w", sourceID, err)
	}
	return nil
}

/** 批量插入商店技能（先删除旧数据再插入） */
func (r *CatalogSkillRepository) ReplaceBySource(ctx context.Context, sourceID string, skills []domain.CatalogSkill) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for replace catalog skills: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM catalog_skills WHERE source_id = ?`, sourceID); err != nil {
		return fmt.Errorf("delete old catalog skills for source %s: %w", sourceID, err)
	}

	for _, skill := range skills {
		supportedAgentsJSON, _ := json.Marshal(skill.SupportedAgents)
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO catalog_skills (
				id, source_id, name, version, author, description, homepage, package_url, checksum_sha256, supported_agents, schema_version
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			skill.ID,
			skill.SourceID,
			skill.Name,
			skill.Version,
			skill.Author,
			skill.Description,
			skill.Homepage,
			skill.PackageURL,
			skill.ChecksumSHA256,
			string(supportedAgentsJSON),
			skill.SchemaVersion,
		); err != nil {
			return fmt.Errorf("insert catalog skill %s: %w", skill.ID, err)
		}
	}

	return tx.Commit()
}

/** 扫描 catalog_skills 行 */
func scanCatalogSkills(rows *sql.Rows) ([]domain.CatalogSkill, error) {
	var skills []domain.CatalogSkill
	for rows.Next() {
		var skill domain.CatalogSkill
		var supportedAgentsJSON string
		if err := rows.Scan(
			&skill.ID,
			&skill.SourceID,
			&skill.Name,
			&skill.Version,
			&skill.Author,
			&skill.Description,
			&skill.Homepage,
			&skill.PackageURL,
			&skill.ChecksumSHA256,
			&supportedAgentsJSON,
			&skill.SchemaVersion,
		); err != nil {
			return nil, fmt.Errorf("scan catalog skill: %w", err)
		}
		_ = json.Unmarshal([]byte(supportedAgentsJSON), &skill.SupportedAgents)
		skills = append(skills, skill)
	}
	return skills, nil
}

// --- Settings Repository ---

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

/** 保存设置项 */
func (r *SettingsRepository) Put(ctx context.Context, key string, value string) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at`,
		key,
		value,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("put setting %s: %w", key, err)
	}
	return nil
}

/** 读取设置项 */
func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %s: %w", key, err)
	}
	return value, nil
}

/** 列出所有设置项 */
func (r *SettingsRepository) List(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		result[key] = value
	}
	return result, nil
}

// --- Task Repository ---

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

/** 插入或更新任务 */
func (r *TaskRepository) Put(ctx context.Context, task domain.Task) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, task_type, trigger_source, project_id, skill_group_id, status, status_reason, plan_json, action_log, result_summary, retry_count, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			task_type = excluded.task_type,
			trigger_source = excluded.trigger_source,
			project_id = excluded.project_id,
			skill_group_id = excluded.skill_group_id,
			status = excluded.status,
			status_reason = excluded.status_reason,
			plan_json = excluded.plan_json,
			action_log = excluded.action_log,
			result_summary = excluded.result_summary,
			retry_count = excluded.retry_count,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at`,
		task.ID,
		task.TaskType,
		task.TriggerSource,
		task.ProjectID,
		nullableString(task.SkillGroupID),
		task.Status,
		task.StatusReason,
		task.PlanJSON,
		task.ActionLog,
		task.ResultSummary,
		task.RetryCount,
		formatNullableTime(task.StartedAt),
		formatNullableTime(task.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("put task %s: %w", task.ID, err)
	}
	return nil
}

/** 按 ID 查询任务 */
func (r *TaskRepository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, task_type, trigger_source, project_id, skill_group_id, status, status_reason, plan_json, action_log, result_summary, retry_count, started_at, finished_at
		FROM tasks WHERE id = ?`,
		id,
	)

	var task domain.Task
	var skillGroupID sql.NullString
	var startedAt sql.NullString
	var finishedAt sql.NullString

	if err := row.Scan(
		&task.ID,
		&task.TaskType,
		&task.TriggerSource,
		&task.ProjectID,
		&skillGroupID,
		&task.Status,
		&task.StatusReason,
		&task.PlanJSON,
		&task.ActionLog,
		&task.ResultSummary,
		&task.RetryCount,
		&startedAt,
		&finishedAt,
	); err != nil {
		return domain.Task{}, fmt.Errorf("get task %s: %w", id, err)
	}

	task.SkillGroupID = skillGroupID.String

	var err error
	task.StartedAt, err = parseNullableTime(startedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("parse task %s started_at: %w", id, err)
	}
	task.FinishedAt, err = parseNullableTime(finishedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("parse task %s finished_at: %w", id, err)
	}

	return task, nil
}

/** 查询最近的任务列表，按 started_at 降序排列 */
func (r *TaskRepository) ListRecent(ctx context.Context, limit int) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, task_type, trigger_source, project_id, skill_group_id, status, status_reason, plan_json, action_log, result_summary, retry_count, started_at, finished_at
		FROM tasks ORDER BY started_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		var skillGroupID sql.NullString
		var startedAt sql.NullString
		var finishedAt sql.NullString

		if err := rows.Scan(
			&task.ID,
			&task.TaskType,
			&task.TriggerSource,
			&task.ProjectID,
			&skillGroupID,
			&task.Status,
			&task.StatusReason,
			&task.PlanJSON,
			&task.ActionLog,
			&task.ResultSummary,
			&task.RetryCount,
			&startedAt,
			&finishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.SkillGroupID = skillGroupID.String
		task.StartedAt, _ = parseNullableTime(startedAt)
		task.FinishedAt, _ = parseNullableTime(finishedAt)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// --- Shared helpers ---

func formatNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return value
}

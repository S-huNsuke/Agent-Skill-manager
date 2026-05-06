package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

func TestProjectAndTaskRepositoriesPersistRequiredFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	assertTablesExist(t, db,
		"agents",
		"skills",
		"catalog_sources",
		"catalog_skills",
		"installed_skills",
		"skill_groups",
		"skill_group_skills",
		"projects",
		"tasks",
	)

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)

	projectLastActiveAt := time.Date(2026, 5, 2, 13, 14, 0, 0, time.UTC)
	projectCreatedAt := projectLastActiveAt.Add(-2 * time.Hour)
	projectUpdatedAt := projectLastActiveAt.Add(-time.Hour)

	project := domain.Project{
		ID:               "project-1",
		Name:             "Agent Skills Manager",
		Type:             "desktop",
		Path:             "/tmp/agent-skills-manager",
		Description:      "Task 2 project fixture",
		AutoApplyEnabled: true,
		LastActiveAt:     &projectLastActiveAt,
		CreatedAt:        projectCreatedAt,
		UpdatedAt:        projectUpdatedAt,
	}

	if err := projectRepo.Put(ctx, project); err != nil {
		t.Fatalf("put project: %v", err)
	}

	storedProject, err := projectRepo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}

	if storedProject.ID != project.ID {
		t.Fatalf("project id mismatch: got %q want %q", storedProject.ID, project.ID)
	}
	if storedProject.SkillGroupID != "" {
		t.Fatalf("project skill_group_id mismatch: got %q want empty", storedProject.SkillGroupID)
	}
	if storedProject.LastActiveAt == nil || !storedProject.LastActiveAt.Equal(projectLastActiveAt) {
		t.Fatalf("project last_active_at mismatch: got %v want %v", storedProject.LastActiveAt, projectLastActiveAt)
	}

	taskStartedAt := projectLastActiveAt.Add(15 * time.Minute)
	taskFinishedAt := taskStartedAt.Add(3 * time.Minute)

	task := domain.Task{
		ID:            "task-1",
		TaskType:      "apply_skill_group",
		TriggerSource: "manual",
		ProjectID:     project.ID,
		Status:        "failed",
		StatusReason:  "catalog source unavailable",
		PlanJSON:      `{"steps":["sync","install"]}`,
		ActionLog:     "sync started\nsync failed",
		ResultSummary: "no skills applied",
		RetryCount:    2,
		StartedAt:     &taskStartedAt,
		FinishedAt:    &taskFinishedAt,
	}

	if err := taskRepo.Put(ctx, task); err != nil {
		t.Fatalf("put task: %v", err)
	}

	storedTask, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	if storedTask.ID != task.ID {
		t.Fatalf("task id mismatch: got %q want %q", storedTask.ID, task.ID)
	}
	if storedTask.ProjectID != project.ID {
		t.Fatalf("task project_id mismatch: got %q want %q", storedTask.ProjectID, project.ID)
	}
	if storedTask.SkillGroupID != "" {
		t.Fatalf("task skill_group_id mismatch: got %q want empty", storedTask.SkillGroupID)
	}
	if storedTask.Status != task.Status {
		t.Fatalf("task status mismatch: got %q want %q", storedTask.Status, task.Status)
	}
	if storedTask.StatusReason != task.StatusReason {
		t.Fatalf("task status_reason mismatch: got %q want %q", storedTask.StatusReason, task.StatusReason)
	}
	if storedTask.RetryCount != task.RetryCount {
		t.Fatalf("task retry_count mismatch: got %d want %d", storedTask.RetryCount, task.RetryCount)
	}
}

func TestProjectRepositoryPutPreservesCreatedAtOnUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewProjectRepository(db)

	createdAt := time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC)
	firstUpdatedAt := createdAt.Add(30 * time.Minute)
	secondUpdatedAt := createdAt.Add(2 * time.Hour)
	lastActiveAt := createdAt.Add(90 * time.Minute)

	project := domain.Project{
		ID:               "project-created-at",
		Name:             "Original",
		Type:             "desktop",
		Path:             "/tmp/project-created-at",
		Description:      "first insert",
		AutoApplyEnabled: true,
		LastActiveAt:     &lastActiveAt,
		CreatedAt:        createdAt,
		UpdatedAt:        firstUpdatedAt,
	}

	if err := repo.Put(ctx, project); err != nil {
		t.Fatalf("put initial project: %v", err)
	}

	project.Name = "Updated"
	project.Description = "second insert"
	project.CreatedAt = createdAt.Add(24 * time.Hour)
	project.UpdatedAt = secondUpdatedAt

	if err := repo.Put(ctx, project); err != nil {
		t.Fatalf("put updated project: %v", err)
	}

	storedProject, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}

	if !storedProject.CreatedAt.Equal(createdAt) {
		t.Fatalf("project created_at mutated: got %v want %v", storedProject.CreatedAt, createdAt)
	}
	if !storedProject.UpdatedAt.Equal(secondUpdatedAt) {
		t.Fatalf("project updated_at mismatch: got %v want %v", storedProject.UpdatedAt, secondUpdatedAt)
	}
}

func TestForeignKeysAreEnforcedForOptionalSkillGroupRelationships(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(t.TempDir() + "/agent-skills-manager.sqlite")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)

	now := time.Date(2026, 5, 2, 13, 14, 0, 0, time.UTC)

	projectWithMissingGroup := domain.Project{
		ID:           "project-invalid-group",
		Name:         "Invalid project group",
		Type:         "desktop",
		Path:         "/tmp/project-invalid-group",
		SkillGroupID: "missing-group",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := projectRepo.Put(ctx, projectWithMissingGroup); err == nil {
		t.Fatal("expected project put with missing skill_group_id to fail")
	}

	project := domain.Project{
		ID:        "project-valid",
		Name:      "Valid project",
		Type:      "desktop",
		Path:      "/tmp/project-valid",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := projectRepo.Put(ctx, project); err != nil {
		t.Fatalf("put project without skill group: %v", err)
	}

	taskWithMissingGroup := domain.Task{
		ID:            "task-invalid-group",
		TaskType:      "apply_skill_group",
		TriggerSource: "manual",
		ProjectID:     project.ID,
		SkillGroupID:  "missing-group",
		Status:        "pending",
		RetryCount:    0,
	}

	if err := taskRepo.Put(ctx, taskWithMissingGroup); err == nil {
		t.Fatal("expected task put with missing skill_group_id to fail")
	}
}

func TestSkillGroupRepositoryPersistRequiredFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewSkillGroupRepository(db)

	now := time.Date(2026, 5, 2, 13, 14, 0, 0, time.UTC)

	skillGroup := domain.SkillGroup{
		ID:              "group-1",
		Name:            "Core Skills",
		Version:         "v1.0.0",
		BoundAgentID:    "claude-code",
		BoundAgentName:  "Claude Code",
		PreferredAgents: []string{"claude-code", "codex"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := repo.Put(ctx, skillGroup); err != nil {
		t.Fatalf("put skill group: %v", err)
	}

	storedGroup, err := repo.GetByID(ctx, skillGroup.ID)
	if err != nil {
		t.Fatalf("get skill group: %v", err)
	}

	if storedGroup.ID != skillGroup.ID {
		t.Fatalf("skill group id mismatch: got %q want %q", storedGroup.ID, skillGroup.ID)
	}
	if storedGroup.Name != skillGroup.Name {
		t.Fatalf("skill group name mismatch: got %q want %q", storedGroup.Name, skillGroup.Name)
	}
	if storedGroup.Version != skillGroup.Version {
		t.Fatalf("skill group version mismatch: got %q want %q", storedGroup.Version, skillGroup.Version)
	}
	if storedGroup.BoundAgentID != skillGroup.BoundAgentID {
		t.Fatalf("skill group bound_agent_id mismatch: got %q want %q", storedGroup.BoundAgentID, skillGroup.BoundAgentID)
	}
	if storedGroup.BoundAgentName != skillGroup.BoundAgentName {
		t.Fatalf("skill group bound_agent_name mismatch: got %q want %q", storedGroup.BoundAgentName, skillGroup.BoundAgentName)
	}
	if len(storedGroup.PreferredAgents) != 2 {
		t.Fatalf("skill group preferred_agents mismatch: got %d want 2", len(storedGroup.PreferredAgents))
	}
	if !storedGroup.CreatedAt.Equal(skillGroup.CreatedAt) {
		t.Fatalf("skill group created_at mismatch: got %v want %v", storedGroup.CreatedAt, skillGroup.CreatedAt)
	}
}

func TestSkillGroupRepositorySkillGroupSkills(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	groupRepo := NewSkillGroupRepository(db)

	now := time.Date(2026, 5, 2, 13, 14, 0, 0, time.UTC)

	group := domain.SkillGroup{
		ID:        "group-skills-1",
		Name:      "Test Group",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := groupRepo.Put(ctx, group); err != nil {
		t.Fatalf("put skill group: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO skills (id, name, description, tags) VALUES (?, ?, ?, ?)`, "skill-1", "Skill One", "test skill", "[]"); err != nil {
		t.Fatalf("insert skill: %v", err)
	}

	sgs := domain.SkillGroupSkill{
		SkillGroupID: group.ID,
		SkillID:      "skill-1",
		Required:     true,
		Priority:     1,
		Reason:       "core dependency",
	}
	if err := groupRepo.PutSkillGroupSkill(ctx, sgs); err != nil {
		t.Fatalf("put skill group skill: %v", err)
	}

	skills, err := groupRepo.ListSkillGroupSkills(ctx, group.ID)
	if err != nil {
		t.Fatalf("list skill group skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skill group skills count mismatch: got %d want 1", len(skills))
	}
	if skills[0].SkillID != "skill-1" {
		t.Fatalf("skill group skill id mismatch: got %q want %q", skills[0].SkillID, "skill-1")
	}
	if !skills[0].Required {
		t.Fatalf("skill group skill required mismatch: got false want true")
	}

	if err := groupRepo.DeleteSkillGroupSkill(ctx, group.ID, "skill-1"); err != nil {
		t.Fatalf("delete skill group skill: %v", err)
	}

	skillsAfterDelete, err := groupRepo.ListSkillGroupSkills(ctx, group.ID)
	if err != nil {
		t.Fatalf("list skill group skills after delete: %v", err)
	}
	if len(skillsAfterDelete) != 0 {
		t.Fatalf("skill group skills count after delete mismatch: got %d want 0", len(skillsAfterDelete))
	}
}

func TestCatalogSourceRepositoryPutAndList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewCatalogSourceRepository(db)

	source := domain.CatalogSource{
		ID:       "source-1",
		Name:     "Official Catalog",
		URL:      "https://catalog.example.com/skills.json",
		IsBuiltin: true,
		Enabled:  true,
	}

	if err := repo.Put(ctx, source); err != nil {
		t.Fatalf("put catalog source: %v", err)
	}

	sources, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list catalog sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("catalog sources count mismatch: got %d want 1", len(sources))
	}
	if sources[0].ID != source.ID {
		t.Fatalf("catalog source id mismatch: got %q want %q", sources[0].ID, source.ID)
	}
	if sources[0].Name != source.Name {
		t.Fatalf("catalog source name mismatch: got %q want %q", sources[0].Name, source.Name)
	}
	if !sources[0].IsBuiltin {
		t.Fatalf("catalog source is_builtin mismatch: got false want true")
	}
	if !sources[0].Enabled {
		t.Fatalf("catalog source enabled mismatch: got false want true")
	}

	if err := repo.Delete(ctx, source.ID); err != nil {
		t.Fatalf("delete catalog source: %v", err)
	}

	sourcesAfterDelete, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list catalog sources after delete: %v", err)
	}
	if len(sourcesAfterDelete) != 0 {
		t.Fatalf("catalog sources count after delete mismatch: got %d want 0", len(sourcesAfterDelete))
	}
}

func TestCatalogSkillRepositoryReplaceBySource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewCatalogSkillRepository(db)

	srcRepo := NewCatalogSourceRepository(db)
	if err := srcRepo.Put(ctx, domain.CatalogSource{ID: "source-1", Name: "Test Source", URL: "https://example.com", Enabled: true}); err != nil {
		t.Fatalf("put catalog source: %v", err)
	}

	skills := []domain.CatalogSkill{
		{ID: "skill-1", SourceID: "source-1", Name: "Skill One", Author: "Author A"},
		{ID: "skill-2", SourceID: "source-1", Name: "Skill Two", Author: "Author B"},
	}

	if err := repo.ReplaceBySource(ctx, "source-1", skills); err != nil {
		t.Fatalf("replace by source: %v", err)
	}

	listed, err := repo.ListBySource(ctx, "source-1")
	if err != nil {
		t.Fatalf("list by source: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("catalog skills count mismatch: got %d want 2", len(listed))
	}

	updatedSkills := []domain.CatalogSkill{
		{ID: "skill-3", SourceID: "source-1", Name: "Skill Three", Author: "Author C"},
	}
	if err := repo.ReplaceBySource(ctx, "source-1", updatedSkills); err != nil {
		t.Fatalf("replace by source (update): %v", err)
	}

	listedAfterUpdate, err := repo.ListBySource(ctx, "source-1")
	if err != nil {
		t.Fatalf("list by source after update: %v", err)
	}
	if len(listedAfterUpdate) != 1 {
		t.Fatalf("catalog skills count after update mismatch: got %d want 1", len(listedAfterUpdate))
	}
	if listedAfterUpdate[0].Name != "Skill Three" {
		t.Fatalf("catalog skill name after update mismatch: got %q want %q", listedAfterUpdate[0].Name, "Skill Three")
	}
}

func TestSettingsRepositoryPutAndGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewSettingsRepository(db)

	value, err := repo.Get(ctx, "general")
	if err != nil {
		t.Fatalf("get non-existent setting: %v", err)
	}
	if value != "" {
		t.Fatalf("non-existent setting value mismatch: got %q want empty", value)
	}

	if err := repo.Put(ctx, "general", `{"theme":"dark"}`); err != nil {
		t.Fatalf("put setting: %v", err)
	}

	storedValue, err := repo.Get(ctx, "general")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if storedValue != `{"theme":"dark"}` {
		t.Fatalf("setting value mismatch: got %q want %q", storedValue, `{"theme":"dark"}`)
	}

	if err := repo.Put(ctx, "general", `{"theme":"light"}`); err != nil {
		t.Fatalf("put setting (update): %v", err)
	}

	updatedValue, err := repo.Get(ctx, "general")
	if err != nil {
		t.Fatalf("get setting after update: %v", err)
	}
	if updatedValue != `{"theme":"light"}` {
		t.Fatalf("setting value after update mismatch: got %q want %q", updatedValue, `{"theme":"light"}`)
	}
}

func TestProjectRepositoryBoundAgentFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	repo := NewProjectRepository(db)

	now := time.Date(2026, 5, 2, 13, 14, 0, 0, time.UTC)

	project := domain.Project{
		ID:             "project-agent-1",
		Name:           "Agent Bound Project",
		Type:           "desktop",
		Path:           "/tmp/project-agent-1",
		BoundAgentID:   "claude-code",
		BoundAgentName: "Claude Code",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := repo.Put(ctx, project); err != nil {
		t.Fatalf("put project with agent: %v", err)
	}

	stored, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}

	if stored.BoundAgentID != "claude-code" {
		t.Fatalf("project bound_agent_id mismatch: got %q want %q", stored.BoundAgentID, "claude-code")
	}
	if stored.BoundAgentName != "Claude Code" {
		t.Fatalf("project bound_agent_name mismatch: got %q want %q", stored.BoundAgentName, "Claude Code")
	}
}

func assertTablesExist(t *testing.T, db *sql.DB, tableNames ...string) {
	t.Helper()

	for _, tableName := range tableNames {
		var actualName string

		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			tableName,
		).Scan(&actualName)
		if err != nil {
			if err == sql.ErrNoRows {
				t.Fatalf("expected table %q to exist", tableName)
			}
			t.Fatalf("query sqlite_master for %q: %v", tableName, err)
		}

		if actualName != tableName {
			t.Fatalf("table name mismatch: got %q want %q", actualName, tableName)
		}
	}
}

func TestMigrationExposesRequiredColumns(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	assertColumnsExist(t, db, "installed_skills", "error_message", "conflict_group", "is_managed_by_app")
	assertColumnsExist(t, db, "projects", "last_active_at")
	assertColumnsExist(t, db, "skill_groups", "version")
	assertColumnsExist(t, db, "tasks", "status", "status_reason", "retry_count")
	assertColumnsExist(t, db, "schema_migrations", "name", "applied_at")
}

func TestMigrateRecordsAppliedMigrationsAndSkipsReapplying(t *testing.T) {
	t.Parallel()

	db, err := Open(t.TempDir() + "/migrations.sqlite")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	firstCount := countRows(t, db, "schema_migrations")
	if firstCount != 3 {
		t.Fatalf("expected 3 recorded migrations after first run, got %d", firstCount)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	secondCount := countRows(t, db, "schema_migrations")
	if secondCount != 3 {
		t.Fatalf("expected 3 recorded migrations after second run, got %d", secondCount)
	}
}

func TestApplyMigrationIsAtomic(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := ensureMigrationTable(db); err != nil {
		t.Fatalf("ensure migration table: %v", err)
	}

	err = applyMigration(db, "999_broken.sql", []byte(`
CREATE TABLE broken_table (
    id TEXT PRIMARY KEY
);
THIS IS NOT VALID SQL;
`))
	if err == nil {
		t.Fatal("expected broken migration to fail")
	}

	if tableExists(t, db, "broken_table") {
		t.Fatal("expected broken_table creation to roll back on failed migration")
	}

	if countRows(t, db, "schema_migrations") != 0 {
		t.Fatal("expected failed migration not to be recorded")
	}
}

func assertColumnsExist(t *testing.T, db *sql.DB, tableName string, columnNames ...string) {
	t.Helper()

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", tableName, err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan pragma row for %s: %v", tableName, err)
		}

		columns[name] = true
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pragma rows for %s: %v", tableName, err)
	}

	for _, columnName := range columnNames {
		if !columns[columnName] {
			t.Fatalf("expected column %q on table %q", columnName, tableName)
		}
	}
}

func countRows(t *testing.T, db *sql.DB, tableName string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", tableName, err)
	}

	return count
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var actualName string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		tableName,
	).Scan(&actualName)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("query sqlite_master for %q: %v", tableName, err)
	}

	return actualName == tableName
}

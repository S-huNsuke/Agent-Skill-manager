CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    install_path TEXT NOT NULL,
    skills_path TEXT NOT NULL,
    last_seen_at TEXT,
    last_error_code TEXT NOT NULL DEFAULT '',
    last_error_message TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS catalog_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    is_builtin INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_synced_at TEXT,
    last_sync_status TEXT NOT NULL DEFAULT '',
    last_sync_error TEXT NOT NULL DEFAULT '',
    cache_expires_at TEXT,
    min_supported_client_version TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS catalog_skills (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    homepage TEXT NOT NULL DEFAULT '',
    package_url TEXT NOT NULL DEFAULT '',
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    supported_agents TEXT NOT NULL DEFAULT '[]',
    schema_version TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (source_id) REFERENCES catalog_sources(id)
);

CREATE TABLE IF NOT EXISTS installed_skills (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    version TEXT NOT NULL,
    install_state TEXT NOT NULL,
    install_path TEXT NOT NULL,
    source_id TEXT NOT NULL,
    last_checked_at TEXT,
    error_message TEXT NOT NULL DEFAULT '',
    conflict_group TEXT NOT NULL DEFAULT '',
    is_managed_by_app INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (skill_id) REFERENCES skills(id),
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (source_id) REFERENCES catalog_sources(id)
);

CREATE TABLE IF NOT EXISTS skill_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    goal_prompt TEXT NOT NULL DEFAULT '',
    preferred_agents TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS skill_group_skills (
    skill_group_id TEXT NOT NULL,
    skill_id TEXT NOT NULL,
    required INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (skill_group_id, skill_id),
    FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    path TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    skill_group_id TEXT,
    auto_apply_enabled INTEGER NOT NULL DEFAULT 0,
    last_active_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id)
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    trigger_source TEXT NOT NULL,
    project_id TEXT NOT NULL,
    skill_group_id TEXT,
    status TEXT NOT NULL,
    status_reason TEXT NOT NULL DEFAULT '',
    plan_json TEXT NOT NULL DEFAULT '',
    action_log TEXT NOT NULL DEFAULT '',
    result_summary TEXT NOT NULL DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at TEXT,
    finished_at TEXT,
    FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

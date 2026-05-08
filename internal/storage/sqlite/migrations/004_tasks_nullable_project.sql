CREATE TABLE IF NOT EXISTS tasks_new (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    trigger_source TEXT NOT NULL,
    project_id TEXT,
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

INSERT INTO tasks_new (
    id, task_type, trigger_source, project_id, skill_group_id, status, status_reason,
    plan_json, action_log, result_summary, retry_count, started_at, finished_at
)
SELECT
    id, task_type, trigger_source, NULLIF(project_id, ''), skill_group_id, status, status_reason,
    plan_json, action_log, result_summary, retry_count, started_at, finished_at
FROM tasks;

DROP TABLE tasks;

ALTER TABLE tasks_new RENAME TO tasks;

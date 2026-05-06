ALTER TABLE projects ADD COLUMN bound_agent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN bound_agent_name TEXT NOT NULL DEFAULT '';
ALTER TABLE skill_groups ADD COLUMN bound_agent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE skill_groups ADD COLUMN bound_agent_name TEXT NOT NULL DEFAULT '';

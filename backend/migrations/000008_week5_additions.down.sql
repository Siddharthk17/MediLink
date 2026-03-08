ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('patient', 'physician', 'admin'));

DROP INDEX IF EXISTS idx_search_queries_actor;
DROP TABLE IF EXISTS search_queries;

DROP INDEX IF EXISTS idx_research_exports_status;
DROP INDEX IF EXISTS idx_research_exports_requester;
DROP TABLE IF EXISTS research_exports;

DROP INDEX IF EXISTS idx_notif_prefs_user;
DROP TABLE IF EXISTS notification_preferences;

DROP INDEX IF EXISTS idx_alerts_priority;
DROP INDEX IF EXISTS idx_alerts_transaction;
DROP INDEX IF EXISTS idx_alerts_assigned;
DROP INDEX IF EXISTS idx_alerts_status_created;
DROP TABLE IF EXISTS alerts CASCADE;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS alert_priority;

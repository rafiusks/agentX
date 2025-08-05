-- Down migration for initial schema
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;
DROP TYPE IF EXISTS message_role;
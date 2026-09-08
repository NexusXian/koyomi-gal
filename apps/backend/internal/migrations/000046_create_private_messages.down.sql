DROP INDEX IF EXISTS idx_user_blocks_blocked_id;
DROP TABLE IF EXISTS user_blocks;

DROP TRIGGER IF EXISTS users_delete_direct_conversations ON users;
DROP FUNCTION IF EXISTS delete_user_direct_conversations();

ALTER TABLE conversation_members
    DROP CONSTRAINT IF EXISTS conversation_members_last_read_message_fk;
ALTER TABLE conversations
    DROP CONSTRAINT IF EXISTS conversations_last_message_fk;

DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS direct_conversations;
DROP TABLE IF EXISTS conversation_members;
DROP TABLE IF EXISTS conversations;

ALTER TABLE user_privacy_settings
    DROP CONSTRAINT IF EXISTS user_privacy_message_permission_check,
    DROP COLUMN IF EXISTS message_permission;

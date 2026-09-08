ALTER TABLE user_privacy_settings
    ADD COLUMN message_permission VARCHAR(20) NOT NULL DEFAULT 'everyone',
    ADD CONSTRAINT user_privacy_message_permission_check
        CHECK (message_permission IN ('everyone', 'none'));

CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    conversation_type VARCHAR(20) NOT NULL DEFAULT 'direct',
    last_message_id BIGINT,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT conversations_type_check CHECK (conversation_type IN ('direct'))
);

CREATE TABLE conversation_members (
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id BIGINT,
    unread_count BIGINT NOT NULL DEFAULT 0,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (conversation_id, user_id),
    CONSTRAINT conversation_members_unread_count_check CHECK (unread_count >= 0)
);

CREATE TABLE direct_conversations (
    conversation_id BIGINT PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    user_low_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_high_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT direct_conversations_canonical_pair_check CHECK (user_low_id < user_high_id),
    CONSTRAINT direct_conversations_unique_pair UNIQUE (user_low_id, user_high_id)
);

CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reply_to_message_id BIGINT REFERENCES messages(id) ON DELETE SET NULL,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT messages_distinct_users_check CHECK (sender_id <> receiver_id),
    CONSTRAINT messages_type_check CHECK (message_type IN ('text', 'image', 'system')),
    CONSTRAINT messages_content_check CHECK (
        deleted_at IS NOT NULL OR (CHAR_LENGTH(BTRIM(content)) BETWEEN 1 AND 2000)
    )
);

ALTER TABLE conversations
    ADD CONSTRAINT conversations_last_message_fk
        FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL;

ALTER TABLE conversation_members
    ADD CONSTRAINT conversation_members_last_read_message_fk
        FOREIGN KEY (last_read_message_id) REFERENCES messages(id) ON DELETE SET NULL;

CREATE TABLE user_blocks (
    blocker_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id),
    CONSTRAINT user_blocks_distinct_users_check CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_conversation_members_user_visible
    ON conversation_members (user_id, conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_conversation_members_user_unread
    ON conversation_members (user_id) WHERE deleted_at IS NULL AND unread_count > 0;
CREATE INDEX idx_conversations_message_sort
    ON conversations ((COALESCE(last_message_at, updated_at, created_at)) DESC, id DESC);
CREATE INDEX idx_messages_conversation_history
    ON messages (conversation_id, id DESC);
CREATE INDEX idx_messages_sender_id ON messages (sender_id);
CREATE INDEX idx_messages_receiver_id ON messages (receiver_id);
CREATE INDEX idx_messages_reply_to_message_id
    ON messages (reply_to_message_id) WHERE reply_to_message_id IS NOT NULL;
CREATE INDEX idx_user_blocks_blocked_id ON user_blocks (blocked_id, blocker_id);

CREATE FUNCTION delete_user_direct_conversations() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM conversations
    WHERE id IN (
        SELECT conversation_id
        FROM direct_conversations
        WHERE user_low_id = OLD.id OR user_high_id = OLD.id
    );
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_delete_direct_conversations
BEFORE DELETE ON users
FOR EACH ROW EXECUTE FUNCTION delete_user_direct_conversations();

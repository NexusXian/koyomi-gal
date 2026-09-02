ALTER SEQUENCE users_id_seq START WITH 1001;

SELECT setval(
    pg_get_serial_sequence('users', 'id'),
    GREATEST(
        COALESCE((SELECT MAX(id) FROM users), 0),
        (SELECT last_value FROM users_id_seq),
        1000
    ),
    true
);

ALTER TABLE audit_logs
    ADD COLUMN actor_name_snapshot VARCHAR(191) NOT NULL DEFAULT '' AFTER actor_user_id,
    ADD COLUMN actor_email_snapshot VARCHAR(191) NOT NULL DEFAULT '' AFTER actor_name_snapshot;

UPDATE audit_logs l
JOIN users u ON u.id = l.actor_user_id
SET
    l.actor_name_snapshot = u.name,
    l.actor_email_snapshot = u.email;

ALTER TABLE audit_logs
    DROP FOREIGN KEY fk_audit_logs_actor_user;

ALTER TABLE audit_logs
    MODIFY COLUMN actor_user_id BIGINT NULL;

ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_actor_user
        FOREIGN KEY (actor_user_id) REFERENCES users(id)
        ON DELETE SET NULL;

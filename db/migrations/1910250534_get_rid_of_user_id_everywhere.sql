-- +migrate Up
SET @saved_sql_mode       = @@sql_mode;
SET sql_mode              = 'NO_ENGINE_SUBSTITUTION';
UPDATE `users` SET `birth_date` = NULL WHERE `birth_date` = '0000-00-00';
SET sql_mode              = @saved_sql_mode;

UPDATE `groups` JOIN `users` ON `users`.`self_group_id` = `groups`.`id` SET `groups`.`type` = 'UserSelf';

DELETE `groups` FROM `groups` LEFT JOIN `users` ON `users`.`self_group_id` = `groups`.`id`
    WHERE `groups`.`type` = 'UserSelf' AND `users`.`id` IS NULL;

UPDATE `groups_attempts`
    LEFT JOIN `users` ON `users`.`id` = `groups_attempts`.`creator_user_id`
SET `groups_attempts`.`creator_user_id` = NULL
WHERE `groups_attempts`.`creator_user_id` IS NOT NULL AND `users`.id IS NULL;

DELETE `filters` FROM `filters` LEFT JOIN `users` ON `users`.`id` = `filters`.`user_id` WHERE `users`.`id` IS NULL;

ALTER TABLE `users`
    MODIFY COLUMN `creator_id` bigint(20) DEFAULT NULL COMMENT 'User who created a given login with the login generation tool',
    DROP INDEX `self_group_id`,
    CHANGE COLUMN `self_group_id` `group_id` bigint(20) NOT NULL COMMENT 'Group that represents this user' FIRST,
    DROP INDEX `godfather_user_id`,
    DROP COLUMN `godfather_user_id`;

UPDATE `users` LEFT JOIN `users` AS `creators` ON `creators`.`id` = `users`.`creator_id`
SET `users`.`creator_id` = `creators`.`group_id`;

ALTER TABLE `users` ADD UNIQUE INDEX `group_id` (`group_id`);

UPDATE `badges` LEFT JOIN `users` ON `users`.`id` = `badges`.`user_id` SET `badges`.`user_id` = `users`.`group_id`;
ALTER TABLE `badges`
    ADD CONSTRAINT `fk_badges_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE;

UPDATE `filters` LEFT JOIN `users` ON `users`.`id` = `filters`.`user_id` SET `filters`.`user_id` = `users`.`group_id`;
ALTER TABLE `filters`
    ADD CONSTRAINT `fk_filters_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE;

ALTER TABLE `groups_attempts` ADD COLUMN `creator_id` bigint(20) DEFAULT NULL AFTER `creator_user_id`;
UPDATE `groups_attempts` LEFT JOIN `users` ON `users`.`id` = `groups_attempts`.`creator_user_id` SET `groups_attempts`.`creator_id` = `users`.`group_id`;
ALTER TABLE `groups_attempts`
    ADD CONSTRAINT `fk_groups_attempts_creator_id_users_group_id` FOREIGN KEY (`creator_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL,
    DROP COLUMN `creator_user_id`;

ALTER TABLE `groups_groups`
    MODIFY COLUMN `inviting_user_id` bigint(20) DEFAULT NULL
        COMMENT 'User (one of the admins of the parent group) who initiated the invitation or accepted the request';
UPDATE `groups_groups` LEFT JOIN `users` ON `users`.`id` = `groups_groups`.`inviting_user_id` SET `groups_groups`.`inviting_user_id` = `users`.`group_id`;
ALTER TABLE `groups_groups`
    ADD CONSTRAINT `fk_groups_groups_inviting_user_id_users_group_id` FOREIGN KEY (`inviting_user_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL;

ALTER TABLE `groups_items`
    ADD COLUMN `creator_id` bigint(20) DEFAULT NULL COMMENT 'User who created the entry' AFTER `creator_user_id` ;
UPDATE `groups_items` LEFT JOIN `users` ON `users`.`id` = `groups_items`.`creator_user_id` SET `groups_items`.`creator_id` = `users`.`group_id`;
ALTER TABLE `groups_items`
    ADD CONSTRAINT `fk_groups_items_creator_id_users_group_id` FOREIGN KEY (`creator_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL;

UPDATE `messages` LEFT JOIN `users` ON `users`.`id` = `messages`.`user_id` SET `messages`.`user_id` = `users`.`group_id`;
ALTER TABLE `messages`
    ADD CONSTRAINT `fk_messages_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL;

UPDATE `refresh_tokens` LEFT JOIN `users` ON `users`.`id` = `refresh_tokens`.`user_id` SET `refresh_tokens`.`user_id` = `users`.`group_id`;
ALTER TABLE `refresh_tokens`
    ADD CONSTRAINT `fk_refresh_tokens_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE;

UPDATE `sessions` LEFT JOIN `users` ON `users`.`id` = `sessions`.`user_id` SET `sessions`.`user_id` = `users`.`group_id`;
ALTER TABLE `sessions`
    ADD CONSTRAINT `fk_sessions_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE;

ALTER TABLE `threads` ADD COLUMN `creator_id` bigint(20) DEFAULT NULL AFTER `creator_user_id`;
UPDATE `threads` LEFT JOIN `users` ON `users`.`id` = `threads`.`creator_user_id` SET `threads`.`creator_id` = `users`.`group_id`;
ALTER TABLE `threads`
    MODIFY COLUMN `creator_id` bigint(20) NOT NULL,
    DROP COLUMN `creator_user_id`;

UPDATE `users_answers` LEFT JOIN `users` ON `users`.`id` = `users_answers`.`user_id` SET `users_answers`.`user_id` = `users`.`group_id`;

DELETE FROM users_answers ua WHERE ua.user_id NOT IN (SELECT group_id FROM users);

ALTER TABLE `users_answers`
    ADD CONSTRAINT `fk_users_answers_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE,
    DROP COLUMN `grader_user_id`;

ALTER TABLE `users_items`
    DROP PRIMARY KEY,
    DROP FOREIGN KEY `fk_user_id`,
    DROP FOREIGN KEY `fk_item_id`,
    DROP FOREIGN KEY `fk_active_attempt_id`;
UPDATE `users_items` LEFT JOIN `users` ON `users`.`id` = `users_items`.`user_id` SET `users_items`.`user_id` = `users`.`group_id`;
ALTER TABLE `users_items`
    ADD PRIMARY KEY (`user_id`, `item_id`),
    ADD CONSTRAINT `fk_users_items_active_attempt_id_groups_attempts_id` FOREIGN KEY (`active_attempt_id`) REFERENCES `groups_attempts` (`id`) ON DELETE CASCADE,
    ADD CONSTRAINT `fk_users_items_item_id_items_id` FOREIGN KEY (`item_id`) REFERENCES `items` (`id`) ON DELETE CASCADE,
    ADD CONSTRAINT `fk_users_items_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE;

ALTER TABLE `users_threads`
    DROP INDEX `user_thread`,
    DROP INDEX `users_idx`;
UPDATE `users_threads` LEFT JOIN `users` ON `users`.`id` = `users_threads`.`user_id` SET `users_threads`.`user_id` = `users`.`group_id`;
ALTER TABLE `users_threads`
    ADD UNIQUE KEY `user_id_thread_id` (`user_id`, `thread_id`),
    ADD INDEX `user_id` (`user_id`);

ALTER TABLE `users`
    DROP PRIMARY KEY,
    DROP INDEX `group_id`,
    ADD PRIMARY KEY (`group_id`),
    ADD CONSTRAINT `fk_users_group_id_groups_id` FOREIGN KEY (`group_id`) REFERENCES `groups`(`id`) ON DELETE CASCADE,
    ADD CONSTRAINT `fk_users_creator_id_users_group_id` FOREIGN KEY (`creator_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL,
    DROP COLUMN `id`;

DROP TRIGGER `before_insert_users`;

DROP VIEW IF EXISTS groups_groups_active;
CREATE VIEW groups_groups_active AS SELECT * FROM groups_groups WHERE NOW() < expires_at;

-- +migrate Down
ALTER TABLE `users`
    DROP FOREIGN KEY `fk_users_group_id_groups_id`,
    DROP FOREIGN KEY `fk_users_creator_id_users_group_id`,
    MODIFY COLUMN `creator_id` bigint(20) DEFAULT NULL COMMENT 'Which user created a given login with the login generation tool',
    ADD COLUMN `id` bigint(20) DEFAULT NULL FIRST;
UPDATE `users` SET `id` = FLOOR(RAND() * 1000000000) + FLOOR(RAND() * 1000000000) * 1000000000;
ALTER TABLE `users`
    MODIFY COLUMN `id` bigint(20) NOT NULL,
    ADD UNIQUE INDEX `id` (`id`);
-- +migrate StatementBegin
CREATE TRIGGER `before_insert_users` BEFORE INSERT ON `users` FOR EACH ROW BEGIN IF (NEW.id IS NULL OR NEW.id = 0) THEN SET NEW.id = FLOOR(RAND() * 1000000000) + FLOOR(RAND() * 1000000000) * 1000000000; END IF; END
-- +migrate StatementEnd

UPDATE `users` LEFT JOIN `users` AS `creators` ON `creators`.`group_id` = `users`.`creator_id`
SET `users`.`creator_id` = `creators`.`id`
WHERE `users`.`creator_id` IS NOT NULL;

ALTER TABLE `badges`
    DROP FOREIGN KEY `fk_badges_user_id_users_group_id`;
UPDATE `badges` LEFT JOIN `users` ON `users`.`group_id` = `badges`.`user_id` SET `badges`.`user_id` = `users`.`id`;

ALTER TABLE `filters`
    DROP FOREIGN KEY `fk_filters_user_id_users_group_id`;
UPDATE `filters` LEFT JOIN `users` ON `users`.`group_id` = `filters`.`user_id` SET `filters`.`user_id` = `users`.`id`;

ALTER TABLE `groups_attempts` ADD COLUMN `creator_user_id` bigint(20) DEFAULT NULL AFTER `creator_id`,
                              DROP FOREIGN KEY `fk_groups_attempts_creator_id_users_group_id`;
UPDATE `groups_attempts` LEFT JOIN `users` ON `users`.`group_id` = `groups_attempts`.`creator_id` SET `groups_attempts`.`creator_user_id` = `users`.`id`;
ALTER TABLE `groups_attempts` DROP COLUMN `creator_id`;

ALTER TABLE `groups_groups`
    DROP FOREIGN KEY `fk_groups_groups_inviting_user_id_users_group_id`;
UPDATE `groups_groups` LEFT JOIN `users` ON `users`.`group_id` = `groups_groups`.`inviting_user_id`
SET `groups_groups`.`inviting_user_id` = IF(`users`.`id` <= 2147483647, `users`.`id`, NULL);
ALTER TABLE `groups_groups`
    MODIFY COLUMN `inviting_user_id` int(11) DEFAULT NULL
        COMMENT 'User (one of the admins of the parent group) who initiated the invitation or accepted the request';

ALTER TABLE `groups_items`
    DROP FOREIGN KEY `fk_groups_items_creator_id_users_group_id`,
    DROP COLUMN `creator_id`;

ALTER TABLE `messages`
    DROP FOREIGN KEY `fk_messages_user_id_users_group_id`;
UPDATE `messages` LEFT JOIN `users` ON `users`.`group_id` = `messages`.`user_id` SET `messages`.`user_id` = `users`.`id`;

ALTER TABLE `refresh_tokens`
    DROP FOREIGN KEY `fk_refresh_tokens_user_id_users_group_id`;
UPDATE `refresh_tokens` LEFT JOIN `users` ON `users`.`group_id` = `refresh_tokens`.`user_id` SET `refresh_tokens`.`user_id` = `users`.`id`;

ALTER TABLE `sessions`
    DROP FOREIGN KEY `fk_sessions_user_id_users_group_id`;
UPDATE `sessions` LEFT JOIN `users` ON `users`.`group_id` = `sessions`.`user_id` SET `sessions`.`user_id` = `users`.`id`;

ALTER TABLE `threads` ADD COLUMN `creator_user_id` bigint(20) DEFAULT NULL AFTER `creator_id`;
UPDATE `threads` LEFT JOIN `users` ON `users`.`group_id` = `threads`.`creator_id` SET `threads`.`creator_user_id` = `users`.`id`;
ALTER TABLE `threads`
    MODIFY COLUMN `creator_user_id` bigint(20) NOT NULL,
    DROP COLUMN `creator_id`;

ALTER TABLE `users_answers`
    DROP FOREIGN KEY `fk_users_answers_user_id_users_group_id`,
    ADD COLUMN `grader_user_id` int(11) DEFAULT NULL COMMENT 'Who did the last grading' AFTER `graded_at`;
UPDATE `users_answers` LEFT JOIN `users` ON `users`.`group_id` = `users_answers`.`user_id` SET `users_answers`.`user_id` = `users`.`id`;

ALTER TABLE `users_items`
    DROP FOREIGN KEY `fk_users_items_user_id_users_group_id`,
    DROP FOREIGN KEY `fk_users_items_item_id_items_id`,
    DROP FOREIGN KEY `fk_users_items_active_attempt_id_groups_attempts_id`,
    DROP PRIMARY KEY;
UPDATE `users_items` LEFT JOIN `users` ON `users`.`group_id` = `users_items`.`user_id` SET `users_items`.`user_id` = `users`.`id`;
ALTER TABLE `users_items`
    ADD PRIMARY KEY (`user_id`, `item_id`),
    ADD CONSTRAINT `fk_active_attempt_id` FOREIGN KEY (`active_attempt_id`) REFERENCES `groups_attempts` (`id`) ON DELETE CASCADE,
    ADD CONSTRAINT `fk_item_id` FOREIGN KEY (`item_id`) REFERENCES `items` (`id`) ON DELETE CASCADE,
    ADD CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;

ALTER TABLE `users_threads`
    DROP INDEX `user_id`,
    DROP INDEX `user_id_thread_id`;
UPDATE `users_threads` LEFT JOIN `users` ON `users`.`group_id` = `users_threads`.`user_id` SET `users_threads`.`user_id` = `users`.`id`;
ALTER TABLE `users_threads`
    ADD INDEX `users_idx` (`user_id`),
    ADD UNIQUE INDEX `user_thread` (`user_id`, `thread_id`);

ALTER TABLE `users`
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`id`),
    DROP INDEX `id`,
    CHANGE COLUMN `group_id` `self_group_id` bigint(20) DEFAULT NULL COMMENT 'Group that represents this user' AFTER `help_given`,
    ADD UNIQUE INDEX `self_group_id` (`self_group_id`),
    ADD COLUMN `godfather_user_id` int(11) DEFAULT NULL AFTER `member_state`,
    ADD INDEX `godfather_user_id` (`godfather_user_id`);

DROP VIEW IF EXISTS groups_groups_active;
CREATE VIEW groups_groups_active AS SELECT * FROM groups_groups WHERE NOW() < expires_at;

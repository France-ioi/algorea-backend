-- +migrate Up
UPDATE `groups_login_prefixes`
    SET `prefix` = CONCAT(`prefix`, '_')
WHERE `prefix` NOT LIKE '%\_';

CREATE TABLE `user_batch_prefixes` (
    `group_prefix` VARCHAR(13) NOT NULL PRIMARY KEY COMMENT 'Prefix used in front of all batches',
    `group_id` BIGINT(20) DEFAULT NULL COMMENT 'Group (and its descendants) in which managers can create users in batch. NULL if the group was deleted, in which case batches should be cleaned manually.',
    `max_users` MEDIUMINT UNSIGNED NOT NULL DEFAULT 1000 COMMENT 'Maximum number of users that can be created under this prefix',
    `allow_new` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether this prefix can be used for new user batches',
    CONSTRAINT `fk_user_batch_prefixes_group_id_groups_id`
        FOREIGN KEY (`group_id`) REFERENCES `groups`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB CHARSET=utf8
    COMMENT='Authorized login prefixes for user batch creation. A prefix cannot be deleted without deleting batches using it.';

CREATE TABLE `user_batches` (
    `group_prefix` VARCHAR(13) NOT NULL COMMENT 'Authorized (first) part of the full login prefix',
    `custom_prefix` VARCHAR(14) NOT NULL
        COMMENT 'Second part of the full login prefix, given by the user that created the batch',
    `size` MEDIUMINT UNSIGNED NOT NULL COMMENT 'Number of users created in this batch',
    `creator_id` BIGINT(20) DEFAULT NULL,
    `created_at` DATETIME NOT NULL DEFAULT NOW(),
    PRIMARY KEY (`group_prefix`, `custom_prefix`),
    CONSTRAINT `ck_user_batches_custom_prefix` CHECK (CAST(`custom_prefix` AS BINARY) REGEXP BINARY '^[a-z0-9-]+$'),
    CONSTRAINT `fk_user_batches_group_prefix_user_batch_prefixes_group_prefix`
        FOREIGN KEY (`group_prefix`) REFERENCES `user_batch_prefixes`(`group_prefix`) ON DELETE RESTRICT,
    CONSTRAINT `fk_user_batches_creator_id_users_group_id`
        FOREIGN KEY (`creator_id`) REFERENCES `users`(`group_id`) ON DELETE SET NULL
) ENGINE=InnoDB CHARSET=utf8
    COMMENT='Batches of users that were created';

INSERT INTO `user_batch_prefixes` (`group_prefix`, `group_id`, `max_users`)
SELECT LEFT(LEFT(`prefix`, CHAR_LENGTH(`prefix`) - LOCATE('_', REVERSE(`prefix`), 2)), 13) AS group_prefix,
       MAX(`group_id`) AS `group_id`,
       1000
FROM `groups_login_prefixes`
GROUP BY group_prefix;

INSERT INTO `user_batches` (`group_prefix`, `custom_prefix`, `size`, `creator_id`, `created_at`)
SELECT LEFT(LEFT(`prefix`, CHAR_LENGTH(`prefix`) - LOCATE('_', REVERSE(`prefix`), 2)), 13) AS `group_prefix`,
       LEFT(REVERSE(SUBSTR(REVERSE(SUBSTRING_INDEX(`prefix`, '_', -2)), 2)), 14) AS `custom_prefix`,
       (SELECT COUNT(*) FROM `users` WHERE BINARY `login` LIKE REPLACE(CONCAT(`prefix`, '%'), '_', '\_') LIMIT 1) AS `size`,
       (SELECT `creator_id` FROM `users` WHERE `creator_id` IS NOT NULL AND BINARY `login` LIKE REPLACE(CONCAT(`prefix`, '%'), '_', '\_') LIMIT 1) AS `creator_id`,
       (SELECT IFNULL(MIN(`created_at`), NOW()) FROM `groups` WHERE BINARY `name` LIKE REPLACE(CONCAT(`prefix`, '%'), '_', '\_')) AS `created_at`
FROM `groups_login_prefixes`;

ALTER TABLE `users` DROP COLUMN `login_module_prefix`;
DROP TRIGGER `before_insert_groups_login_prefixes`;
DROP TABLE `groups_login_prefixes`;

-- +migrate Down
CREATE TABLE `groups_login_prefixes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `group_id` bigint(20) NOT NULL,
  `prefix` varchar(100) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `prefix` (`prefix`),
  KEY `group_id` (`group_id`)
) ENGINE=InnoDB CHARSET=utf8
    COMMENT='Used to keep track of prefixes of logins that were generated in batch and attached to this group. Only the prefix is stored, not the actual usernames.';

-- +migrate StatementBegin
CREATE TRIGGER `before_insert_groups_login_prefixes` BEFORE INSERT ON `groups_login_prefixes` FOR EACH ROW BEGIN IF (NEW.id IS NULL OR NEW.id = 0) THEN SET NEW.id = FLOOR(RAND() * 1000000000) + FLOOR(RAND() * 1000000000) * 1000000000; END IF; END
-- +migrate StatementEnd

INSERT INTO `groups_login_prefixes` (`group_id`, `prefix`)
SELECT
    user_batch_prefixes.group_id,
    CONCAT(`group_prefix`, '_', `custom_prefix`) AS `prefix`
FROM `user_batches` LEFT JOIN `user_batch_prefixes` USING (`group_prefix`)
GROUP BY group_prefix, custom_prefix;

ALTER TABLE `users`
    ADD COLUMN `login_module_prefix` varchar(100) DEFAULT NULL
        COMMENT 'Set to enable login module accounts manager'
            AFTER `notifications_read_at`;

DROP TABLE `user_batches`;
DROP TABLE `user_batch_prefixes`;

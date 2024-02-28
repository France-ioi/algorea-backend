-- +migrate Up

# We first populate the new_sessions table (1),
# then the access_tokens table (2).
# After, we drop the old tables (3),
# and rename new_sessions to sessions (4).
# Finally, we remove the auto increment from sessions.session_id (5)
# because we want to use random uint63 generated by the backend.

# Data to test the migration (from empty tables)
#
# INSERT INTO `groups` (`id`, `name`, `type`) VALUES (1, 'group 1', 'User');
# INSERT INTO `groups` (`id`, `name`, `type`) VALUES (2, 'group 2', 'User');
# INSERT INTO `groups` (`id`, `name`, `type`) VALUES (3, 'group 3', 'User');
# INSERT INTO `groups` (`id`, `name`, `type`) VALUES (4, 'group 4', 'User');
# INSERT INTO `users` (`group_id`, `temp_user`, `login`) VALUES (1, 0, 'user_1');
# INSERT INTO `users` (`group_id`, `temp_user`, `login`) VALUES (2, 0, 'user_2');
# INSERT INTO `users` (`group_id`, `temp_user`, `login`) VALUES (3, 1, 'temp_user_3');
# INSERT INTO `users` (`group_id`, `temp_user`, `login`) VALUES (4, 1, 'temp_user_4');
# INSERT INTO `refresh_tokens` (`user_id`, `refresh_token`) VALUES (1, 'refresh_token_1');
# INSERT INTO `refresh_tokens` (`user_id`, `refresh_token`) VALUES (2, 'refresh_token_2');
# INSERT INTO `sessions` (`access_token`, `user_id`, `expires_at`, `issued_at`, `issuer`) VALUES ('access_token_1a', 1, '2024-01-01 01:00:00', '2024-01-01 00:00:00', 'login-module');
# INSERT INTO `sessions` (`access_token`, `user_id`, `expires_at`, `issued_at`, `issuer`) VALUES ('access_token_1b', 1, '2024-01-01 02:00:00', '2024-01-01 01:00:00', 'login-module');
# INSERT INTO `sessions` (`access_token`, `user_id`, `expires_at`, `issued_at`, `issuer`) VALUES ('access_token_2', 2, '2024-01-01 02:00:00', '2024-01-01 01:00:00', 'login-module');
# INSERT INTO `sessions` (`access_token`, `user_id`, `expires_at`, `issued_at`, `issuer`) VALUES ('access_token_3', 3, '2024-01-01 02:00:00', '2024-01-01 01:00:00', 'backend');
# INSERT INTO `sessions` (`access_token`, `user_id`, `expires_at`, `issued_at`, `issuer`) VALUES ('access_token_4', 4, '2024-01-01 02:00:00', '2024-01-01 01:00:00', 'backend');
#
# The result should be:
#
# SELECT * FROM `sessions`;
# +------------+---------+----------------+
# | session_id | user_id | refresh_token  |
# +------------+---------+----------------+
# | 1          | 1       | refresh_token_1|
# | 2          | 2       | refresh_token_2|
# | 3          | 3       | NULL           |
# | 4          | 4       | NULL           |
# +------------+---------+----------------+
#
# SELECT * FROM `access_tokens`;
# +-----------------+------------+---------------------+---------------------+
# | token           | session_id | expires_at          | issued_at           |
# +-----------------+------------+---------------------+---------------------+
# | access_token_1a | 1          | 2024-01-01 01:00:00 | 2024-01-01 00:00:00 |
# | access_token_1b | 1          | 2024-01-01 02:00:00 | 2024-01-01 01:00:00 |
# | access_token_2  | 2          | 2024-01-01 00:00:00 | 2024-01-01 01:00:00 |
# | access_token_3  | 3          | 2024-01-01 00:00:00 | 2024-01-01 01:00:00 |
# | access_token_4  | 4          | 2024-01-01 00:00:00 | 2024-01-01 01:00:00 |
# +----------------+------------+---------------------+---------------------+
#


CREATE TABLE `new_sessions` (
  `session_id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  CONSTRAINT `fk_sessions_users_user_id_group_id` FOREIGN KEY (`user_id`) REFERENCES `users`(`group_id`) ON DELETE CASCADE,
  `refresh_token` VARBINARY(2000) NULL COMMENT 'Refresh tokens (unlimited lifetime) used by the backend to request fresh access tokens from the auth module',
  PRIMARY KEY (`session_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Sessions represent a logged in user, on a specific device.';

ALTER TABLE `new_sessions` ADD INDEX `refresh_token` (`refresh_token`(767));

CREATE TABLE `access_tokens` (
  `token` VARBINARY(2000) NOT NULL COMMENT 'The access token.',
  `session_id` BIGINT NOT NULL,
  CONSTRAINT `fk_access_tokens_sessions_session_id` FOREIGN KEY (`session_id`) REFERENCES `new_sessions`(`session_id`) ON DELETE CASCADE,
  `expires_at` DATETIME NOT NULL COMMENT 'The time the token expires and becomes invalid. It should be deleted after this time.',
  `issued_at` DATETIME NOT NULL DEFAULT NOW() COMMENT 'The time the token was issued.'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Access tokens (short lifetime) distributed to users, to access a specific session.';

ALTER TABLE `access_tokens` ADD INDEX `expires_at` (`expires_at`);
ALTER TABLE `access_tokens` ADD INDEX `token` (`token`(767));

# 1a.
# For each row in the current refresh_tokens table [create rows for real users in sessions]:
# - Create an entry in the new sessions table with user_id, refresh_token and session_id (auto incremented for the migration)
INSERT INTO `new_sessions` (`user_id`, `refresh_token`) SELECT `user_id`, `refresh_token` FROM `refresh_tokens`;

# 1b.
# For each row in the current sessions table linked to a temp user [create rows for temp users in sessions]:
# - Create an entry in the new sessions table with user_id, refresh_token=NULL and session_id (auto incremented for the migration).
INSERT INTO `new_sessions` (`user_id`) SELECT `user_id` FROM `sessions` JOIN `users` ON `sessions`.`user_id` = `users`.`group_id` WHERE `users`.`temp_user` = 1;

# 2.
# For each row in the current sessions table [create rows for both real and temp users in access_tokens]:
# - Create an entry in the new access_tokens table with the token, expires_at, issued_at, and the session_id retrieved with the user_id in the new table sessions.
INSERT INTO `access_tokens` (`token`, `expires_at`, `issued_at`, `session_id`) SELECT `sessions`.`access_token`, `sessions`.`expires_at`, `sessions`.`issued_at`, `new_sessions`.`session_id` FROM `sessions` JOIN `new_sessions` ON `sessions`.`user_id` = `new_sessions`.`user_id`;

# 3. Drop the old tables table.
DROP TABLE `sessions`;
DROP TABLE `refresh_tokens`;

# 4. Rename the new_sessions table to sessions.
RENAME TABLE `new_sessions` TO `sessions`;

# 5. Remove the auto increment from sessions.session_id.
#
# We need to lock the table to avoid concurrent writes while the foreign key is being modified.
LOCK TABLES
  `sessions` WRITE,
  `access_tokens` WRITE;

ALTER TABLE `access_tokens` DROP FOREIGN KEY `fk_access_tokens_sessions_session_id`;
ALTER TABLE `sessions` MODIFY COLUMN `session_id` BIGINT NOT NULL;
ALTER TABLE `access_tokens` ADD CONSTRAINT `fk_access_tokens_sessions_session_id` FOREIGN KEY (`session_id`) REFERENCES `sessions`(`session_id`) ON DELETE CASCADE;

UNLOCK TABLES;


-- +migrate Down
DROP TABLE `access_tokens`;
DROP TABLE `sessions`;

CREATE TABLE `sessions` (
  `access_token` VARBINARY(2000) NULL DEFAULT NULL,
  `user_id` BIGINT(19) NOT NULL,
  `expires_at` DATETIME NOT NULL,
  `issued_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `issuer` ENUM('backend','login-module') NULL DEFAULT NULL COLLATE 'utf8_general_ci',
  INDEX `expires_at` (`expires_at`) USING BTREE,
  INDEX `access_token_prefix` (`access_token`(767)) USING BTREE,
  INDEX `user_id` (`user_id`) USING BTREE,
  CONSTRAINT `fk_sessions_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`group_id`) ON UPDATE NO ACTION ON DELETE CASCADE
)
  COMMENT='Access tokens (short lifetime) distributed to users'
  COLLATE='utf8_general_ci'
  ENGINE=InnoDB
;

CREATE TABLE `refresh_tokens` (
  `user_id` BIGINT(19) NOT NULL,
  `refresh_token` VARBINARY(2000) NOT NULL,
  PRIMARY KEY (`user_id`) USING BTREE,
  INDEX `refresh_token_prefix` (`refresh_token`(767)) USING BTREE,
  CONSTRAINT `fk_refresh_tokens_user_id_users_group_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`group_id`) ON UPDATE NO ACTION ON DELETE CASCADE
)
  COMMENT='Refresh tokens (unlimited lifetime) used by the backend to request fresh access tokens from the auth module'
  COLLATE='utf8_general_ci'
  ENGINE=InnoDB
;

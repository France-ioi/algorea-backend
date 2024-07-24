-- +migrate Up

ALTER TABLE `groups_login_prefixes` DROP COLUMN `idUserCreator`;
ALTER TABLE `items_items` DROP COLUMN `iWeight`;

ALTER TABLE `groups` DROP KEY `sPassword`;
ALTER TABLE `groups` DROP KEY `sType`;
ALTER TABLE `groups` DROP KEY `sTextId`;
ALTER TABLE `groups_groups` DROP KEY `idUserInviting`;
ALTER TABLE `history_groups_groups` DROP KEY `idGroupParent`;
ALTER TABLE `history_groups_groups` DROP KEY `idGroupChild`;
ALTER TABLE `history_groups_groups` DROP KEY `idUserInviting`;
ALTER TABLE `users_items` DROP KEY `UserAttempt`;

-- +migrate Down

-- +migrate Up
ALTER TABLE `groups_groups`
  COMMENT 'Parent-child (N-N) relationships between groups (acyclic graph). It includes potential relationships such as invitations or requests to join groups, as well as past relationships.',
  MODIFY COLUMN `iChildOrder` int(11) NOT NULL DEFAULT '0' COMMENT 'Position of this child among its siblings.',
  MODIFY COLUMN `sRole` enum('manager','owner','member','observer') NOT NULL DEFAULT 'member' COMMENT 'Role that the child has relative to the parent.',
  MODIFY COLUMN `idUserInviting` int(20) DEFAULT NULL COMMENT 'User (one of the admins of the parent group) who initiated the invitation or accepted the request',
  MODIFY COLUMN `sStatusDate` datetime DEFAULT NULL COMMENT 'When was the type last Changed.';

-- +migrate Down
ALTER TABLE `groups_groups`
  COMMENT '',
  MODIFY COLUMN `iChildOrder` int(11) NOT NULL DEFAULT '0' COMMENT '',
  MODIFY COLUMN `sRole` enum('manager','owner','member','observer') NOT NULL DEFAULT 'member' COMMENT '',
  MODIFY COLUMN `idUserInviting` int(20) DEFAULT NULL COMMENT '',
  MODIFY COLUMN `sStatusDate` datetime DEFAULT NULL COMMENT '';

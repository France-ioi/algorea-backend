-- +migrate Up
ALTER TABLE `groups`
    MODIFY COLUMN `type` enum('Class','Team','Club','Friends','Other','User','Session','Base','ContestParticipants') NOT NULL
        AFTER `name`,
    MODIFY COLUMN `open_contest` TINYINT(1) NOT NULL DEFAULT '0'
        COMMENT 'If true and the group is associated through activity_id with an item that is a contest, the contest should be started for this user as soon as he joins the group',
    ADD COLUMN `activity_id` BIGINT(20) DEFAULT NULL
        COMMENT 'Root activity (chapter, task, or course) associated with this group'
            AFTER `redirect_path`,
    ADD CONSTRAINT `fk_groups_activity_id_items_id` FOREIGN KEY (`activity_id`) REFERENCES `items`(`id`)
        ON DELETE SET NULL,
    ADD COLUMN `require_members_to_join_parent` TINYINT(1) NOT NULL DEFAULT 0
        COMMENT 'For sessions, whether the user joining this group should join the parent group as well'
            AFTER `require_watch_approval`,
    ADD COLUMN `organizer` VARCHAR(255) DEFAULT NULL
        COMMENT 'For sessions, a teacher/animator in charge of the organization',
    ADD COLUMN `address_line1` VARCHAR(255) DEFAULT NULL COMMENT 'For sessions or schools',
    ADD COLUMN `address_line2` VARCHAR(255) DEFAULT NULL COMMENT 'For sessions or schools',
    ADD COLUMN `address_postcode` VARCHAR(25) DEFAULT NULL COMMENT 'For sessions or schools',
    ADD COLUMN `address_city` VARCHAR(255) DEFAULT NULL COMMENT 'For sessions or schools',
    ADD COLUMN `address_country` VARCHAR(255) DEFAULT NULL COMMENT 'For sessions or schools',
    ADD COLUMN `expected_start` DATETIME DEFAULT NULL COMMENT 'For sessions, time at which the session is expected to start';

UPDATE `groups` SET `activity_id` = REVERSE(SUBSTRING_INDEX(REVERSE(REVERSE(SUBSTRING_INDEX(REVERSE(redirect_path), '/', 1))), '-', 1))
WHERE `redirect_path` IS NOT NULL AND `redirect_path` != '';

ALTER TABLE `groups` DROP COLUMN `redirect_path`;

-- +migrate Down
DELETE FROM `groups` WHERE `type` = 'Session';

ALTER TABLE `groups`
    MODIFY COLUMN `open_contest` TINYINT(1) NOT NULL DEFAULT '0'
        COMMENT 'If true and the group is associated through redirect_path with an item that is a contest, the contest should be started for this user as soon as he joins the group.',
    MODIFY COLUMN `type` enum('Class','Team','Club','Friends','Other','User','Base','ContestParticipants') NOT NULL
        AFTER `open_contest`,
    DROP COLUMN `require_members_to_join_parent`,
    DROP COLUMN `organizer`,
    DROP COLUMN `address_line1`,
    DROP COLUMN `address_line2`,
    DROP COLUMN `address_postcode`,
    DROP COLUMN `address_city`,
    DROP COLUMN `address_country`,
    DROP COLUMN `expected_start`,
    ADD COLUMN `redirect_path` TEXT
        COMMENT 'Where the user should be sent when joining this group. For now it is a path to be used in the url.'
            AFTER `code_expires_at`;

UPDATE `groups` SET `redirect_path` = `activity_id` WHERE `activity_id` IS NOT NULL;

ALTER TABLE `groups`
    DROP FOREIGN KEY `fk_groups_activity_id_items_id`,
    DROP COLUMN `activity_id`;

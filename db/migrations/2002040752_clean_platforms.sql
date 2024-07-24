-- +migrate Up
ALTER TABLE `platforms`
    MODIFY COLUMN `public_key` VARCHAR(512) DEFAULT NULL COMMENT 'Public key of this platform';

UPDATE `platforms` SET `public_key` = NULL WHERE NOT uses_tokens OR `public_key` = '';

UPDATE `platforms` p1
  JOIN (SELECT id, `priority`*10 + row_number() OVER (PARTITION BY `priority` ORDER BY `priority`) -1 as `new_priority` FROM platforms) p2 ON p1.id = p2.id
  SET p1.`priority` = p2.`new_priority`;

ALTER TABLE `platforms`
    DROP COLUMN `uses_tokens`,
    ADD UNIQUE KEY `priority` (`priority` DESC),
    MODIFY COLUMN `base_url` VARCHAR(200) DEFAULT NULL
        COMMENT 'Base URL for calling the API of the platform (for GDPR services)',
    MODIFY COLUMN `regexp` TEXT
        COMMENT 'Regexp matching the urls, to automatically detect content from this platform. It is the only way to specify which items are from which platform. Recomputation of items.platform_id is triggered when changed.',
    MODIFY COLUMN `priority` INT(11) NOT NULL DEFAULT '0'
        COMMENT 'Priority of the regexp compared to others (higher value is tried first). Recomputation of items.platform_id is triggered when changed.';

UPDATE `items` SET `url` = NULL WHERE `type` = 'Chapter' OR `url` = '';

DROP TRIGGER IF EXISTS `before_insert_items`;
-- +migrate StatementBegin
CREATE TRIGGER `before_insert_items` BEFORE INSERT ON `items` FOR EACH ROW BEGIN
    IF (NEW.id IS NULL OR NEW.id = 0) THEN
        SET NEW.id = FLOOR(RAND() * 1000000000) + FLOOR(RAND() * 1000000000) * 1000000000;
    END IF;

    IF NEW.url IS NOT NULL THEN
        SELECT platforms.id INTO @platformID FROM platforms
        WHERE NEW.url REGEXP platforms.regexp
        ORDER BY platforms.priority DESC LIMIT 1;

        SET NEW.platform_id=@platformID;
    ELSE
        SET NEW.platform_id = NULL;
    END IF;
END
-- +migrate StatementEnd

DROP TRIGGER IF EXISTS `before_update_items`;
-- +migrate StatementBegin
CREATE TRIGGER `before_update_items` BEFORE UPDATE ON `items` FOR EACH ROW BEGIN
    IF NOT OLD.url <=> NEW.url THEN
        IF NEW.url IS NOT NULL THEN
            SELECT platforms.id INTO @platformID FROM platforms
            WHERE NEW.url REGEXP platforms.regexp
            ORDER BY platforms.priority DESC LIMIT 1;

            SET NEW.platform_id=@platformID;
        ELSE
            SET NEW.platform_id = NULL;
        END IF;
    END IF;
END
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER `after_insert_platforms` AFTER INSERT ON `platforms` FOR EACH ROW BEGIN
    UPDATE `items`
        LEFT JOIN `platforms` AS `old_platform` ON `old_platform`.`id` = `items`.`platform_id`
    SET `items`.`platform_id` = (
        SELECT `platforms`.`id` FROM `platforms`
        WHERE `items`.`url` REGEXP `platforms`.`regexp`
        ORDER BY `platforms`.`priority` DESC
        LIMIT 1
    )
    WHERE `old_platform`.`priority` < NEW.`priority` OR (`items`.`url` IS NOT NULL AND `old_platform`.`id` IS NULL);
END
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER `after_update_platforms` AFTER UPDATE ON `platforms` FOR EACH ROW BEGIN
    IF OLD.`priority` != NEW.`priority` OR NOT OLD.`regexp` <=> NEW.`regexp` THEN
        UPDATE `items`
            LEFT JOIN `platforms` AS `old_platform` ON `old_platform`.`id` = `items`.`platform_id`
        SET `items`.`platform_id` = (
            SELECT `platforms`.`id` FROM `platforms`
            WHERE `items`.`url` REGEXP `platforms`.`regexp`
            ORDER BY `platforms`.`priority` DESC
            LIMIT 1
        )
        WHERE `old_platform`.`priority` < NEW.`priority` OR
              (`items`.`url` IS NOT NULL AND `old_platform`.`id` IS NULL) OR
              `old_platform`.`id` = NEW.`id`;
    END IF;
END
-- +migrate StatementEnd

UPDATE `items` SET `platform_id` = NULL;

ALTER TABLE `items`
    MODIFY COLUMN `platform_id` INT(11) DEFAULT NULL COMMENT 'Platform that hosts the item content. Auto-generated from `url` by triggers.',
    ADD CONSTRAINT `fk_items_platform_id_platforms_id` FOREIGN KEY (`platform_id`) REFERENCES `platforms`(`id`)
        ON DELETE RESTRICT;

UPDATE `items`
SET `items`.`platform_id` = (
    SELECT `platforms`.`id` FROM `platforms`
    WHERE `items`.`url` REGEXP `platforms`.`regexp`
    ORDER BY `platforms`.`priority` DESC
    LIMIT 1
)
WHERE `items`.`url` IS NOT NULL;


-- +migrate Down
ALTER TABLE `platforms`
    ADD COLUMN `uses_tokens` TINYINT(1) NOT NULL
        COMMENT 'Whether this platform supports tokens. If true, data such as the score sent to the platform will be sent as JWT, and the data sent by the platform must be signed with the key as well.'
        AFTER `public_key`,
    DROP KEY `priority`,
    MODIFY COLUMN `base_url` VARCHAR(200) DEFAULT NULL COMMENT '',
    MODIFY COLUMN `regexp` TEXT
        COMMENT 'Regexp matching the urls, to automatically detect content from this platform. It is the only way to specify which items are from which platform.',
    MODIFY COLUMN `priority` INT(11) NOT NULL DEFAULT '0'
        COMMENT 'Priority of the regexp compared to others (higher value is tried first).';

UPDATE `platforms` SET `uses_tokens` = `public_key` IS NULL, `public_key` = IFNULL(`public_key`, '');

ALTER TABLE `platforms`
    MODIFY COLUMN `public_key` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Public key of this platform';

DROP TRIGGER IF EXISTS `before_insert_items`;
-- +migrate StatementBegin
CREATE TRIGGER `before_insert_items` BEFORE INSERT ON `items` FOR EACH ROW BEGIN IF (NEW.id IS NULL OR NEW.id = 0) THEN SET NEW.id = FLOOR(RAND() * 1000000000) + FLOOR(RAND() * 1000000000) * 1000000000; END IF ; SELECT platforms.id INTO @platformID FROM platforms WHERE NEW.url REGEXP platforms.regexp ORDER BY platforms.priority DESC LIMIT 1 ; SET NEW.platform_id=@platformID ; END
-- +migrate StatementEnd

DROP TRIGGER IF EXISTS `before_update_items`;
-- +migrate StatementBegin
CREATE TRIGGER `before_update_items` BEFORE UPDATE ON `items` FOR EACH ROW BEGIN SELECT platforms.id INTO @platformID FROM platforms WHERE NEW.url REGEXP platforms.regexp ORDER BY platforms.priority DESC LIMIT 1 ; SET NEW.platform_id=@platformID ; END
-- +migrate StatementEnd

DROP TRIGGER `after_insert_platforms`;
DROP TRIGGER `after_update_platforms`;

ALTER TABLE `items`
    MODIFY COLUMN `platform_id` INT(11) DEFAULT NULL COMMENT 'Platform that hosts this item',
    DROP FOREIGN KEY `fk_items_platform_id_platforms_id`;

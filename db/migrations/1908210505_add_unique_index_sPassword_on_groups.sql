-- +migrate Up
UPDATE `groups` SET `sPassword` = NULL WHERE `sPassword` = '';

# Deduplicate groups.sPassword: appends a suffix to make all sPassword unique.
UPDATE `groups`
  JOIN (
    SELECT `ID`, ROW_NUMBER() OVER (PARTITION BY `groups`.`sPassword` ORDER BY `ID`) AS `order`
    FROM `groups`
           JOIN (
      SELECT `ID`, `sPassword` FROM `groups` AS `groups1`
      WHERE EXISTS(
        SELECT `ID` from `groups` AS `groups2`
        WHERE `groups1`.`sPassword`=`groups2`.`sPassword` AND `groups1`.`ID` > `groups2`.`ID`
      )
      GROUP BY `ID`
    ) AS `duplicates` USING (`ID`)
  ) AS `orders` USING (`ID`)
SET `groups`.`sPassword` = CONCAT(`groups`.`sPassword`, '@dup', `orders`.`order`);

ALTER TABLE `groups` ADD UNIQUE INDEX `sPassword` (`sPassword`);

-- +migrate Down
ALTER TABLE `groups` DROP INDEX `sPassword`;

DROP TABLE IF EXISTS `user_withdrawal`;

DELETE FROM `system`
WHERE `category` = 'invite'
  AND `key` IN ('WithdrawalMethod', 'WithdrawalMethods', 'WithdrawalMinAmount');

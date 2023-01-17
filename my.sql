CREATE TABLE `user` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(10) NOT NULL DEFAULT '' COMMENT 'user name',
  `nickname` varchar(10) DEFAULT '' COMMENT 'user nick',
  `avatar` varchar(128) DEFAULT '' COMMENT 'user avatar uri',
  `is_del` tinyint(1) DEFAULT '0' COMMENT 'user is del',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
  `update_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'last update time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8
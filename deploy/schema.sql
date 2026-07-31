-- QinBlog Go 版数据库结构（对照 Laravel migrations 生成，表结构与现库一致）
-- 用法：mysql -u root -p qin-blog < schema.sql
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `users` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `avatar` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `email_verified_at` timestamp NULL DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `openid` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `remember_token` varchar(100) DEFAULT NULL,
  `is_admin` tinyint(1) NOT NULL DEFAULT 0,
  `notification_count` int unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `users_email_unique` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `password_resets` (
  `email` varchar(255) NOT NULL,
  `token` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  KEY `password_resets_email_index` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `posts` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `body` text NOT NULL,
  `user_id` int unsigned NOT NULL,
  `category_id` int unsigned NOT NULL,
  `topic_id` int unsigned NOT NULL DEFAULT 0 COMMENT '所属专题,0代表无专题',
  `sort` int unsigned NOT NULL DEFAULT 1 COMMENT '用于专题排序',
  `comment_count` int unsigned NOT NULL DEFAULT 0 COMMENT '评论数量',
  `view_count` int unsigned NOT NULL DEFAULT 0 COMMENT '查看总数',
  `order` int unsigned NOT NULL DEFAULT 0 COMMENT '排序',
  `is_show` tinyint NOT NULL DEFAULT 1 COMMENT '是否显示',
  `excerpt` text COMMENT '文章摘要，SEO 优化时使用',
  `slug` varchar(255) DEFAULT NULL COMMENT 'SEO 友好的 URI',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `posts_title_index` (`title`),
  KEY `posts_user_id_index` (`user_id`),
  KEY `posts_category_id_index` (`category_id`),
  KEY `posts_topic_id_index` (`topic_id`),
  KEY `posts_sort_index` (`sort`),
  KEY `posts_slug_index` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `tags` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(10) NOT NULL,
  `description` text,
  `post_count` int NOT NULL DEFAULT 0 COMMENT '标签下的文章总数',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tags_name_unique` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `topics` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '专题名称',
  `description` text COMMENT '专题描述',
  `post_count` int unsigned NOT NULL DEFAULT 0 COMMENT '文章数',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `topics_name_index` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `comments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `content` text NOT NULL,
  `parent_id` int unsigned DEFAULT NULL,
  `approved` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否审核通过',
  `commentable_id` int unsigned NOT NULL,
  `commentable_type` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `comments_user_id_index` (`user_id`),
  KEY `comments_parent_id_index` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `links` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '名称',
  `url` varchar(255) NOT NULL COMMENT '链接地址',
  `logo` varchar(255) DEFAULT NULL COMMENT '图片',
  `sort` int unsigned NOT NULL DEFAULT 0 COMMENT '排序',
  `status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否显示',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `links_sort_index` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `post_tag` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int unsigned NOT NULL,
  `tag_id` int unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `post_tag_post_id_index` (`post_id`),
  KEY `post_tag_tag_id_index` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `columns` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '名称',
  `link` varchar(255) NOT NULL COMMENT '地址',
  `description` text COMMENT '描述',
  PRIMARY KEY (`id`),
  KEY `columns_name_index` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `categories` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '名称',
  `icon` varchar(255) NOT NULL COMMENT '图标',
  `post_count` int NOT NULL DEFAULT 0 COMMENT '分类下的文章总数',
  `description` text COMMENT '描述',
  PRIMARY KEY (`id`),
  KEY `categories_name_index` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `notifications` (
  `id` char(36) NOT NULL,
  `type` varchar(255) NOT NULL,
  `notifiable_type` varchar(255) NOT NULL,
  `notifiable_id` bigint unsigned NOT NULL,
  `data` text NOT NULL,
  `read_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `notifications_notifiable_index` (`notifiable_type`,`notifiable_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- settings 表由 Go 服务启动时 AutoMigrate 创建，无需在此建表

-- ================= 种子数据（等价 seed_* migrations） =================

INSERT INTO `categories` (`name`, `icon`, `description`) VALUES
  ('About Blog', 'iconblog', ''),
  ('PHP', 'iconphpx', ''),
  ('Laravel', 'iconlaravel', ''),
  ('MySQL', 'iconshujukuleixingtubiao-kuozhan-', ''),
  ('Vue', 'iconVue', ''),
  ('Docker', 'icondocker', ''),
  ('Python', 'iconpython', ''),
  ('JavaScript', 'iconphpx', ''),
  ('Life', 'iconlife', ''),
  ('Other', 'iconother', '');

INSERT INTO `columns` (`name`, `link`, `description`) VALUES
  ('文章', 'posts', ''),
  ('专题', 'topics', ''),
  ('关于', 'about', '');

-- ================= 演示数据（可选，便于首次验收） =================
-- 密码均为 password（Laravel bcrypt 哈希，验证 $2y$ 兼容）

INSERT INTO `users` (`id`, `name`, `email`, `email_verified_at`, `password`, `is_admin`, `created_at`, `updated_at`) VALUES
  (1, 'admin', 'admin@example.com', NOW(), '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 1, NOW(), NOW()),
  (2, 'demo', 'demo@example.com', NOW(), '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 0, NOW(), NOW());

INSERT INTO `topics` (`id`, `name`, `description`, `post_count`, `created_at`, `updated_at`) VALUES
  (1, 'Go 重构实战', '把 Laravel 博客搬到 Go 的系列文章', 1, NOW(), NOW());

INSERT INTO `tags` (`id`, `name`, `description`, `post_count`, `created_at`, `updated_at`) VALUES
  (1, 'Golang', NULL, 2, NOW(), NOW()),
  (2, 'Gin', NULL, 1, NOW(), NOW());

INSERT INTO `posts` (`id`, `title`, `body`, `user_id`, `category_id`, `topic_id`, `sort`, `comment_count`, `view_count`, `is_show`, `excerpt`, `slug`, `created_at`, `updated_at`) VALUES
  (1, 'Hello QinBlog Go', '<h2>重构完成</h2><p>这是用 <strong>Gin + GORM</strong> 重写后的第一篇文章，支持 Markdown 编辑、全文搜索与评论通知。</p><pre><code class="language-go">package main\n\nfunc main() {}\n</code></pre>', 1, 3, 1, 1, 2, 42, 1, '这是用 Gin + GORM 重写后的第一篇文章，支持 Markdown 编辑、全文搜索与评论通知。', 'hello-qinblog-go', NOW(), NOW()),
  (2, 'Go 并发编程入门', '<h2>Goroutine</h2><p>Go 的并发模型基于 goroutine 与 channel，天然适合高并发 Web 服务。</p><h2>Channel</h2><p>通过通信来共享内存。</p>', 1, 7, 0, 1, 0, 7, 1, 'Go 的并发模型基于 goroutine 与 channel，天然适合高并发 Web 服务。', 'go-concurrency-guide', NOW(), NOW());

INSERT INTO `post_tag` (`post_id`, `tag_id`, `created_at`, `updated_at`) VALUES
  (1, 1, NOW(), NOW()),
  (1, 2, NOW(), NOW()),
  (2, 1, NOW(), NOW());

INSERT INTO `comments` (`id`, `user_id`, `content`, `parent_id`, `approved`, `commentable_id`, `commentable_type`, `created_at`, `updated_at`) VALUES
  (1, 2, '恭喜重构完成！', NULL, 1, 1, 'App\\Post', NOW(), NOW()),
  (2, 1, '谢谢支持~', 1, 1, 1, 'App\\Post', NOW(), NOW());

INSERT INTO `links` (`name`, `url`, `logo`, `sort`, `status`, `created_at`, `updated_at`) VALUES
  ('Go 官网', 'https://go.dev', NULL, 1, 1, NOW(), NOW());

-- 同步分类文章计数（等价 PostObserver 维护的 post_count）
UPDATE `categories` c SET c.`post_count` =
  (SELECT COUNT(*) FROM `posts` p WHERE p.`category_id` = c.`id` AND p.`is_show` = 1);

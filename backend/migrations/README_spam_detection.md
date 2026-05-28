# 垃圾邮件检测功能数据库迁移说明

## 迁移文件

- `012_add_spam_detection_tables.sql` - 垃圾邮件检测功能的数据库迁移

## 迁移内容

### 1. 扩展 emails 表
添加以下字段：
- `is_spam` - 是否为垃圾邮件
- `spam_score` - 垃圾邮件评分（0-100）
- `spam_confidence` - 置信度（0-1.0）
- `spam_reason` - 检测原因（JSON 数组）
- `spam_detected_at` - 检测时间
- `spam_detected_by` - 检测层级
- `user_marked_spam` - 用户是否手动标记为垃圾
- `user_marked_at` - 用户标记时间

### 2. 新增表

#### email_lists（白名单/黑名单）
存储用户的白名单和黑名单配置。

#### sender_reputations（发件人信誉）
跟踪发件人的历史行为和信誉评分。

#### spam_rules（垃圾邮件规则）
存储内置和自定义的垃圾邮件检测规则。

#### bayesian_trainings（贝叶斯训练数据）
存储用户标记的邮件，用于训练个性化贝叶斯模型。

#### spam_detection_logs（检测日志）
记录每次垃圾邮件检测的详细信息。

#### settings（系统设置）
如果不存在则创建，并插入垃圾邮件检测的默认配置。

## 运行迁移

该功能的正式 schema 变更位于根目录版本化迁移 `012_add_spam_detection_tables.sql`。按 `backend/migrations/README.md` 的规则执行显式迁移；不要把 AutoMigrate 当成生产迁移方案。

```bash
# 需要手工执行 SQL 时，明确指定版本化迁移文件
psql -h localhost -U fusionmail -d fusionmail -f migrations/012_add_spam_detection_tables.sql
```

## 验证迁移

连接到数据库并验证表结构：

```sql
-- 检查 emails 表的新字段
\d emails

-- 检查新表是否创建
\dt email_lists
\dt sender_reputations
\dt spam_rules
\dt bayesian_trainings
\dt spam_detection_logs

-- 检查系统设置
SELECT * FROM settings WHERE category = 'spam';
```

## 回滚迁移

如果需要回滚迁移：

```sql
-- 删除新增的表
DROP TABLE IF EXISTS spam_detection_logs;
DROP TABLE IF EXISTS bayesian_trainings;
DROP TABLE IF EXISTS spam_rules;
DROP TABLE IF EXISTS sender_reputations;
DROP TABLE IF EXISTS email_lists;

-- 删除 emails 表的新字段
ALTER TABLE emails DROP COLUMN IF EXISTS is_spam;
ALTER TABLE emails DROP COLUMN IF EXISTS spam_score;
ALTER TABLE emails DROP COLUMN IF EXISTS spam_confidence;
ALTER TABLE emails DROP COLUMN IF EXISTS spam_reason;
ALTER TABLE emails DROP COLUMN IF EXISTS spam_detected_at;
ALTER TABLE emails DROP COLUMN IF EXISTS spam_detected_by;
ALTER TABLE emails DROP COLUMN IF EXISTS user_marked_spam;
ALTER TABLE emails DROP COLUMN IF EXISTS user_marked_at;

-- 删除系统设置
DELETE FROM settings WHERE category = 'spam';
```

## 注意事项

1. **备份数据库**：在执行迁移前，请务必备份数据库
2. **测试环境**：建议先在测试环境中执行迁移
3. **AutoMigrate 边界**：模型仍可用于开发环境 AutoMigrate，但生产变更以版本化迁移文件为准
4. **索引创建**：迁移会自动创建必要的索引以优化查询性能
5. **默认值**：所有新字段都有合理的默认值，不会影响现有数据

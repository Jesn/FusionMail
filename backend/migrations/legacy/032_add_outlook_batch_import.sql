-- 032_add_outlook_batch_import.sql
-- 为 Outlook 提供商添加 batch_import 协议支持

-- 更新 Outlook 提供商的 supported_protocols，添加 batch_import
UPDATE providers 
SET supported_protocols = '["oauth2","imap","batch_import"]'
WHERE name = 'outlook' 
  AND supported_protocols NOT LIKE '%batch_import%';

-- 验证更新结果
-- SELECT name, supported_protocols FROM providers WHERE name = 'outlook';

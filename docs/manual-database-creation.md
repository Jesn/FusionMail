# 手动创建开发数据库指南

如果自动化脚本无法使用，可以通过以下方式手动创建数据库。

## 方法 1: 使用数据库管理工具（推荐）

### 使用 pgAdmin

1. 打开 pgAdmin
2. 添加新服务器：
   - 名称: FusionMail Dev
   - 主机: 192.168.2.200
   - 端口: 5432
   - 用户: postgres
   - 密码: 8QMZn3yfrbkVG7

3. 右键点击 "Databases" → "Create" → "Database"
4. 输入数据库名: `fusionmail-dev`
5. Owner: postgres
6. 点击 "Save"

7. 打开 Query Tool，执行以下 SQL：
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

### 使用 DBeaver

1. 打开 DBeaver
2. 新建连接 → PostgreSQL
3. 输入连接信息：
   - 主机: 192.168.2.200
   - 端口: 5432
   - 数据库: postgres
   - 用户: postgres
   - 密码: 8QMZn3yfrbkVG7

4. 测试连接
5. 右键点击连接 → "SQL Editor" → "New SQL Script"
6. 执行以下 SQL：

```sql
-- 创建数据库
CREATE DATABASE "fusionmail-dev"
    WITH 
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

-- 连接到新数据库（在 DBeaver 中需要刷新并连接到新数据库）
-- 然后执行以下命令创建扩展

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

### 使用 TablePlus

1. 打开 TablePlus
2. 新建连接 → PostgreSQL
3. 输入连接信息：
   - Host: 192.168.2.200
   - Port: 5432
   - User: postgres
   - Password: 8QMZn3yfrbkVG7
   - Database: postgres

4. 连接后，点击 "SQL" 按钮
5. 执行创建数据库的 SQL（同上）

## 方法 2: 使用在线 SQL 工具

如果你有 Web 访问权限，可以使用 Adminer 或 phpPgAdmin 等在线工具。

## 方法 3: 使用 Docker（无需安装 psql）

```bash
# 创建数据库
docker run --rm \
    -e PGPASSWORD=8QMZn3yfrbkVG7 \
    postgres:15-alpine \
    psql -h 192.168.2.200 -p 5432 -U postgres \
    -c "CREATE DATABASE \"fusionmail-dev\" OWNER postgres;"

# 创建扩展
docker run --rm \
    -e PGPASSWORD=8QMZn3yfrbkVG7 \
    postgres:15-alpine \
    psql -h 192.168.2.200 -p 5432 -U postgres -d fusionmail-dev \
    -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"; CREATE EXTENSION IF NOT EXISTS \"pg_trgm\";"
```

或者使用我们提供的脚本：

```bash
./scripts/create-db-docker.sh
```

## 验证数据库创建

创建完成后，可以通过以下方式验证：

### 使用数据库管理工具
在工具中刷新数据库列表，应该能看到 `fusionmail-dev` 数据库。

### 使用 Docker
```bash
docker run --rm \
    -e PGPASSWORD=8QMZn3yfrbkVG7 \
    postgres:15-alpine \
    psql -h 192.168.2.200 -p 5432 -U postgres \
    -c "\l" | grep fusionmail-dev
```

### 启动项目验证
```bash
./start.sh -b

# 查看日志
tail -f logs/backend.log

# 应该看到：
# [INFO] 数据库连接成功
# [INFO] 数据库迁移完成
```

## 完整的 SQL 脚本

如果你想一次性执行所有命令，可以使用以下完整脚本：

```sql
-- ============================================
-- FusionMail 开发环境数据库创建脚本
-- ============================================

-- 1. 创建数据库
CREATE DATABASE "fusionmail-dev"
    WITH 
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

-- 2. 连接到新创建的数据库
-- 注意：在某些工具中需要手动切换到新数据库

-- 3. 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 4. 授予权限
GRANT ALL PRIVILEGES ON DATABASE "fusionmail-dev" TO postgres;

-- 完成！
```

## 常见问题

### Q: 数据库已存在怎么办？

如果数据库已存在，可以选择：

1. **保留现有数据库**：直接使用，跳过创建步骤
2. **删除并重新创建**：

```sql
-- 断开所有连接
SELECT pg_terminate_backend(pg_stat_activity.pid)
FROM pg_stat_activity
WHERE pg_stat_activity.datname = 'fusionmail-dev'
  AND pid <> pg_backend_pid();

-- 删除数据库
DROP DATABASE IF EXISTS "fusionmail-dev";

-- 重新创建
CREATE DATABASE "fusionmail-dev" OWNER postgres;
```

### Q: 没有权限创建数据库？

确保使用的是 `postgres` 超级用户，或者联系数据库管理员授予创建数据库的权限。

### Q: 扩展创建失败？

某些 PostgreSQL 安装可能没有这些扩展。如果创建失败，可以跳过扩展创建，项目仍然可以运行，只是某些功能可能受限。

---

**提示**: 创建完数据库后，运行 `./start.sh` 启动项目，后端会自动执行数据库迁移创建所需的表结构。

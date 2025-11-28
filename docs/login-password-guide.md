# FusionMail 登录密码指南

## 管理员账户

### 默认凭据

- **用户名**: `admin`
- **密码**: 
  - 开发环境：存储在 `backend/passwd` 文件中
  - Docker 部署：存储在容器内 `/app/passwd` 文件中
  - 或通过环境变量 `ADMIN_PASSWORD` 设置

### 密码管理

#### 1. 查看当前密码

**开发环境**：
```bash
cat backend/passwd
```

**Docker 部署**：
```bash
docker-compose exec app cat /app/passwd
```

#### 2. 设置管理员密码的方式

系统支持两种方式设置管理员密码：

**方式一：环境变量（推荐用于生产环境）**

在 `.env` 文件或 `docker-compose.yml` 中设置：
```bash
ADMIN_PASSWORD=your-strong-password-here
```

优点：
- 密码可预设，便于自动化部署
- 不依赖密码文件
- 更安全（不会在日志中输出）

**方式二：自动生成（默认）**

如果未设置 `ADMIN_PASSWORD`，系统会：
1. 生成一个 16 字符的随机密码
2. 创建管理员用户并保存密码哈希到数据库
3. 将明文密码保存到 `passwd` 文件（开发环境）
4. 在日志中输出密码（开发环境）

#### 3. 密码文件保存策略

系统根据运行模式自动决定是否保存密码文件：

| 环境 | GIN_MODE | 默认行为 | 说明 |
|------|----------|----------|------|
| 开发环境 | debug | 保存到 `passwd` 文件 | 方便开发测试 |
| 生产环境 | release | 不保存 | 安全考虑 |

**强制保存密码文件**（不推荐）：
```bash
SAVE_PASSWORD_FILE=true
```

#### 3. 密码不匹配问题

如果遇到"用户名或密码错误"，可能是因为：
- 数据库中的密码哈希与 `passwd` 文件中的密码不匹配
- 数据库被重置但 `passwd` 文件未更新
- 或相反情况

**解决方案**：使用 Docker 直接更新数据库密码

```bash
# 1. 读取 passwd 文件中的密码
PASSWORD=$(cat backend/passwd)

# 2. 生成新的密码哈希（使用 Go）
cd backend
go run -c "
package main
import (
    \"fmt\"
    \"os\"
    \"golang.org/x/crypto/bcrypt\"
)
func main() {
    password, _ := os.ReadFile(\"passwd\")
    hash, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
    fmt.Print(string(hash))
}
" > /tmp/new_hash.txt

# 3. 更新数据库
NEW_HASH=$(cat /tmp/new_hash.txt)
docker-compose exec postgres psql -U fusionmail -d fusionmail -c \
  "UPDATE users SET password_hash = '$NEW_HASH' WHERE username = 'admin';"
```

或者使用更简单的方法：

```bash
# 直接在 Docker 中执行 SQL 更新
# 注意：需要手动替换 <NEW_HASH> 为实际的哈希值
docker-compose exec postgres psql -U fusionmail -d fusionmail -c \
  "UPDATE users SET password_hash = '<NEW_HASH>' WHERE username = 'admin';"
```

#### 4. 重置管理员密码

如果需要重置为新的随机密码：

```bash
# 1. 停止服务
./stop.sh

# 2. 删除管理员用户
docker-compose exec postgres psql -U fusionmail -d fusionmail -c \
  "DELETE FROM users WHERE username = 'admin';"

# 3. 删除 passwd 文件
rm backend/passwd

# 4. 重新启动（会自动创建新的管理员用户和密码）
./start.sh
```

## 开发环境 vs Docker 部署

### 开发环境（./start.sh）

- 后端直接运行在宿主机上
- 数据库配置使用 `localhost`
- 密码文件：`backend/passwd`
- 端口：3333（后端）、4444（前端）

### Docker 部署（docker-compose up）

- 后端运行在 Docker 容器中
- 数据库配置使用服务名 `postgres`
- 环境变量会覆盖 `.env` 文件中的配置
- 密码文件：容器内 `/app/passwd`
- 端口：3333（应用，包含前后端）

**重要提示**：
- 如果同时运行开发环境和 Docker 部署，会导致端口冲突
- 必须先停止开发环境（`./stop.sh`）再启动 Docker 部署
- 或者先停止 Docker 部署（`docker-compose down`）再启动开发环境

### 配置说明

`backend/.env` 文件中的数据库配置：

```env
# 本地开发使用 localhost
DB_HOST=localhost
DB_PORT=5432

# Docker Compose 部署时会被环境变量覆盖为 postgres
```

`docker-compose.yml` 中的环境变量会覆盖：

```yaml
environment:
  DB_HOST: postgres  # 覆盖 .env 中的 localhost
  DB_PORT: 5432
```

## 安全建议

### 开发环境

1. **不要提交 `backend/passwd` 文件到 Git**
   - 已在 `.gitignore` 中配置忽略
   
2. **定期更换密码**
   - 开发环境建议每月更换一次

### 生产环境

1. **首次部署后立即修改密码**
   ```bash
   # 登录后在设置页面修改密码
   ```

2. **使用强密码**
   - 至少 16 字符
   - 包含大小写字母、数字和特殊字符

3. **启用双因素认证**（未来版本）

4. **定期审计登录日志**
   ```sql
   SELECT username, last_login_at, last_login_ip, failed_login_attempts
   FROM users
   WHERE role = 'admin';
   ```

## 故障排查

### 问题：登录失败，提示"用户名或密码错误"

**检查步骤**：

1. **确认使用正确的密码文件**
   
   开发环境：
   ```bash
   cat backend/passwd
   ```
   
   Docker 部署：
   ```bash
   docker-compose exec app cat /app/passwd
   ```

2. **检查是否有端口冲突**
   
   查看 3333 端口占用情况：
   ```bash
   lsof -i :3333
   ```
   
   如果发现有多个进程占用端口：
   ```bash
   # 停止开发环境
   ./stop.sh
   
   # 或停止 Docker 部署
   docker-compose down
   ```

3. **检查数据库中的用户**
   ```bash
   docker-compose exec postgres psql -U fusionmail -d fusionmail -c \
     "SELECT username, created_at, failed_login_attempts FROM users WHERE username = 'admin';"
   ```

4. **检查后端日志**
   
   开发环境：
   ```bash
   tail -f logs/backend.log
   ```
   
   Docker 部署：
   ```bash
   docker-compose logs -f app
   ```

5. **验证密码哈希**
   ```bash
   # 使用测试脚本验证
   cd backend
   go run test_bcrypt.go
   ```

### 问题：账户被锁定

如果连续登录失败 5 次，账户会被锁定 30 分钟。

**解锁方法**：

```bash
docker-compose exec postgres psql -U fusionmail -d fusionmail -c \
  "UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE username = 'admin';"
```

## 相关文件

- `backend/passwd` - 管理员密码明文（仅开发环境）
- `backend/.env` - 环境变量配置
- `docker-compose.yml` - Docker 部署配置
- `backend/internal/service/init_service.go` - 系统初始化服务
- `backend/pkg/crypto/crypto.go` - 密码加密工具

## 更新日志

- 2025-11-29: 修复了开发环境登录密码不匹配的问题
- 2025-11-29: 添加了密码重置指南

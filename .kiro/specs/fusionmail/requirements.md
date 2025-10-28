# FusionMail 需求文档（精简版）

## 项目简介

FusionMail 是一款轻量级邮件接收聚合系统，专注于从多个邮箱账户收集邮件，并通过自动化机制与其他产品和系统集成。系统主要面向个人用户和小团队，提供高效的邮件聚合、智能分类和灵活的自动化触发能力。

**核心定位**：
- 当前版本优先实现邮件接收功能，邮件发送功能暂缓实现
- 当前版本不包含联系人管理功能
- 聚焦邮件聚合 + 自动化集成
- 支持与第三方系统无缝对接

---

## 术语表

- **FusionMail 系统**：本邮件聚合管理系统
- **邮箱账户**：用户添加到系统中的邮箱（Gmail、Outlook、iCloud 等）
- **同步任务**：后台自动从邮箱服务器拉取邮件的定时任务
- **邮件规则**：用户定义的自动化规则，用于邮件分类、标签和触发动作
- **Webhook**：当特定邮件事件发生时，向外部 URL 推送通知的机制
- **API 接口**：供外部系统调用的 RESTful API
- **代理配置**：为邮箱账户配置的网络代理（HTTP/SOCKS5）
- **增量同步**：只同步新邮件，避免重复拉取已有邮件
- **邮件事件**：邮件到达、标记、归档等操作产生的事件
- **Gmail API**：Google 提供的官方邮件访问接口
- **Microsoft Graph**：Microsoft 提供的统一 API 接口，用于访问 Outlook/Hotmail
- **IMAP**：互联网邮件访问协议（Internet Message Access Protocol），用于从邮件服务器读取邮件，支持双向同步
- **POP3**：邮局协议第 3 版（Post Office Protocol 3），用于从邮件服务器下载邮件到本地，不支持双向同步
- **OAuth2**：开放授权标准，用于安全的第三方应用授权
- **应用专用密码**：为第三方应用生成的独立密码，用于 iCloud 等服务
- **Provider ID**：邮箱服务商提供的邮件原生唯一标识（如 Gmail 的 Message ID、IMAP 的 UID）
- **UID**：邮箱账户的唯一标识符，用于区分不同的邮箱账户
- **Message-ID**：邮件头部的标准标识字段，用于邮件关联和线程追踪
- **只读镜像模式**：邮件状态变更（标记、归档、删除等）仅影响 FusionMail 本地数据，不回写到源邮箱服务器
- **状态回写**：将本地邮件状态变更同步回源邮箱服务器（当前版本不支持）
- **对象存储**：云端文件存储服务，如 AWS S3、阿里云 OSS、腾讯云 COS 等
- **预签名 URL**：带有时效性和访问权限的临时 URL，用于安全地访问私有对象存储中的文件

---

## 邮件状态管理策略

FusionMail 当前版本采用**只读镜像模式（方案 A）**，以降低系统复杂度并确保数据安全。

### 方案 A：只读镜像（当前实现）

**核心原则**：
- 所有邮件状态变更（标记已读/未读、星标、归档、删除、添加标签等）仅影响 FusionMail 系统内部数据
- 不会将任何状态变更回写到源邮箱服务器
- 源邮箱保持原始状态不变

**适用场景**：
- 邮件聚合与查看
- 本地分类与管理
- 自动化规则处理
- 数据分析与统计

**优势**：
1. **降低复杂度**：无需处理不同邮箱协议的状态映射和同步逻辑
2. **提高稳定性**：避免回写失败、冲突处理等问题
3. **保护源数据**：源邮箱数据不受影响，降低数据丢失风险
4. **简化权限**：只需要读取权限，无需写入权限

**用户体验要求**：
- UI 界面需明确标注"仅本地"或"本地视图"
- 在邮件操作按钮旁显示提示信息
- 在帮助文档中说明状态变更不会同步到源邮箱

### 方案 B：双向元数据同步（未来扩展）

**核心原则**：
- 本地邮件状态变更可在可控条件下回写到源邮箱
- 需要定义完整的状态映射表
- 需要处理同步冲突和失败重试

**技术挑战**：
1. **协议差异**：不同邮箱服务商的状态表示方式不同
   - Gmail：标签系统（Labels）
   - IMAP：文件夹 + 标志位（Flags）
   - Outlook：类别（Categories）

2. **状态映射**：
   - 已读/未读：IMAP SEEN 标志 ↔ Gmail 标签
   - 星标：IMAP FLAGGED ↔ Gmail Starred
   - 归档：移动到 Archive 文件夹 ↔ Gmail Archive 标签
   - 删除：移动到 Trash 文件夹 ↔ Gmail Trash 标签

3. **冲突处理**：
   - 源邮箱在外部被修改时如何合并
   - 回写失败时的重试策略
   - 网络异常时的数据一致性

**实现时机**：
- 在用户明确需要双向同步功能时
- 在系统稳定运行并积累足够经验后
- 作为可选功能，由用户选择是否启用

---

## 邮箱支持矩阵

| 邮箱服务商 | 首选方案 | 备选方案 | 认证方式 | 说明 |
|-----------|---------|---------|---------|------|
| **Gmail** | Gmail API | IMAP | OAuth2 | 推荐使用 Gmail API，性能更好且支持推送通知；IMAP 作为备选方案 |
| **Outlook** | Microsoft Graph | IMAP | OAuth2 | 推荐使用 Microsoft Graph API，功能更完整；IMAP 作为备选方案 |
| **Hotmail** | Microsoft Graph | IMAP | OAuth2 | 与 Outlook 使用相同的 Microsoft Graph API |
| **iCloud** | IMAP | POP3 | 应用专用密码 | 必须在 Apple ID 设置中生成应用专用密码；推荐使用 IMAP |
| **QQ 邮箱** | IMAP | POP3 | 授权码 | 需要在 QQ 邮箱设置中开启 IMAP/POP3 并获取授权码；推荐使用 IMAP |
| **163 邮箱** | IMAP | POP3 | 授权码 | 需要在 163 邮箱设置中开启 IMAP/POP3 并获取授权码；推荐使用 IMAP |
| **其他邮箱** | IMAP | POP3 | 用户名密码 | 支持标准 IMAP/POP3 协议的邮箱服务；推荐使用 IMAP |

### 协议优先级说明

**厂商 API（首选）**：
1. **Gmail**：优先使用 Gmail API（更高效、支持推送、实时通知），降级使用 IMAP + OAuth2
2. **Outlook/Hotmail**：优先使用 Microsoft Graph API（功能完整、性能更好），降级使用 IMAP + OAuth2

**通用协议（备选）**：
3. **iCloud**：支持 IMAP（推荐）或 POP3，使用应用专用密码认证
4. **国内邮箱（QQ/163）**：支持 IMAP（推荐）或 POP3，使用授权码认证
5. **其他邮箱**：支持标准 IMAP（推荐）或 POP3 协议

**协议选择建议**：
- **IMAP 优先**：支持双向同步、多设备访问、服务器端邮件管理，更适合邮件聚合场景
- **POP3 备选**：仅支持下载到本地，不推荐用于邮件聚合系统
- **厂商 API 最优**：性能最好、功能最完整，但仅限特定邮箱服务商

---

## 需求列表

### 需求 1：多邮箱账户管理

**用户故事**：作为用户，我希望能够添加和管理多个邮箱账户，以便在一个系统中聚合所有邮件。

#### 验收标准

1. WHEN 用户添加新邮箱账户时，THE FusionMail 系统 SHALL 支持 Gmail、Outlook、Hotmail、iCloud Email、QQ 邮箱、163 邮箱和通用 IMAP/POP3 协议
2. WHEN 用户添加 Gmail 账户时，THE FusionMail 系统 SHALL 优先使用 Gmail API，并支持降级到 IMAP + OAuth2
3. WHEN 用户添加 Outlook 或 Hotmail 账户时，THE FusionMail 系统 SHALL 优先使用 Microsoft Graph API，并支持降级到 IMAP + OAuth2
4. WHEN 用户配置邮箱账户凭证时，THE FusionMail 系统 SHALL 根据邮箱类型自动选择合适的认证方式（OAuth2、应用专用密码、授权码或用户名密码）
5. THE FusionMail 系统 SHALL 使用 AES-256 加密算法存储所有敏感信息（密码、Token、代理密码）
6. THE FusionMail 系统 SHALL 允许用户为每个邮箱账户独立配置 HTTP 代理或 SOCKS5 代理
7. WHEN 用户测试邮箱连接时，THE FusionMail 系统 SHALL 在 10 秒内返回连接成功或失败的结果，并显示使用的协议类型
8. IF Gmail API 或 Microsoft Graph API 不可用，THEN THE FusionMail 系统 SHALL 自动降级到 IMAP 协议并通知用户

---

### 需求 2：后台自动同步

**用户故事**：作为用户，我希望系统能够自动从所有邮箱拉取新邮件，以便实时获取最新邮件内容。

#### 验收标准

1. THE FusionMail 系统 SHALL 支持用户配置同步频率，范围为 1 分钟到 60 分钟
2. WHEN 同步任务启动时，THE FusionMail 系统 SHALL 使用增量同步机制，只拉取自上次同步后的新邮件
3. WHEN 多个邮箱需要同步时，THE FusionMail 系统 SHALL 并发执行同步任务，最大并发数为 5
4. IF 邮箱连接失败，THEN THE FusionMail 系统 SHALL 使用指数退避算法重试，最多重试 3 次
5. WHEN 同步任务执行时，THE FusionMail 系统 SHALL 记录同步开始时间、结束时间、拉取邮件数量和错误信息
6. THE FusionMail 系统 SHALL 允许用户手动触发立即同步操作

---

### 需求 3：邮件存储与索引

**用户故事**：作为用户，我希望系统能够高效存储和索引邮件，以便快速检索和查看邮件内容。

#### 验收标准

1. WHEN 邮件同步完成时，THE FusionMail 系统 SHALL 存储邮件的标题、发件人、收件人、抄送人、时间、正文和附件信息
2. THE FusionMail 系统 SHALL 为邮件的标题、发件人、正文内容创建全文搜索索引
3. THE FusionMail 系统 SHALL 使用邮箱服务商原生 ID（Provider ID）和邮箱账户唯一标识（UID）的组合作为邮件的主键
4. WHEN 邮件已存在时，THE FusionMail 系统 SHALL 通过 Provider ID + UID 组合进行去重判断，避免重复存储
5. THE FusionMail 系统 SHALL 支持存储邮件的原始 HTML 格式和纯文本格式
6. WHEN 用户删除邮箱账户时，THE FusionMail 系统 SHALL 提供选项保留或删除该账户的历史邮件
7. THE FusionMail 系统 SHALL 默认使用本地存储保存邮件附件
8. THE FusionMail 系统 SHALL 支持多种附件存储方式，包括本地存储（默认）、AWS S3、阿里云 OSS 等对象存储
9. WHEN 系统配置为使用对象存储时，THE FusionMail 系统 SHALL 自动将附件上传到配置的对象存储服务
10. THE FusionMail 系统 SHALL 为附件生成访问 URL，支持预签名 URL 用于临时访问控制

---

### 需求 4：邮件查看与基础操作（只读镜像模式）

**用户故事**：作为用户，我希望能够查看邮件列表和详情，并进行本地管理操作，以便在 FusionMail 中组织和查看我的邮件。

#### 验收标准

1. THE FusionMail 系统 SHALL 提供统一收件箱视图，聚合显示所有邮箱的邮件
2. WHEN 用户查看邮件列表时，THE FusionMail 系统 SHALL 显示邮件的标题、发件人、时间、未读状态和附件标识
3. THE FusionMail 系统 SHALL 支持按邮箱账户、时间范围、已读未读状态筛选邮件
4. WHEN 用户点击邮件时，THE FusionMail 系统 SHALL 显示邮件的完整内容，包括正文和附件列表
5. THE FusionMail 系统 SHALL 允许用户在本地标记邮件为已读或未读、星标、归档、删除，该状态仅在 FusionMail 系统内生效
6. THE FusionMail 系统 SHALL 在邮件操作界面显示"仅本地"提示，说明操作不会影响源邮箱
7. THE FusionMail 系统 SHALL 支持批量操作，允许用户同时操作多封邮件
8. WHEN 用户查看邮件附件时，THE FusionMail 系统 SHALL 支持图片和 PDF 文件的在线预览及下载

---

### 需求 5：邮件搜索与过滤

**用户故事**：作为用户，我希望能够快速搜索和过滤邮件，以便找到我需要的邮件内容。

#### 验收标准

1. THE FusionMail 系统 SHALL 支持全文搜索，搜索范围包括邮件标题、发件人、收件人和正文内容
2. WHEN 用户输入搜索关键词时，THE FusionMail 系统 SHALL 在 2 秒内返回搜索结果
3. THE FusionMail 系统 SHALL 支持高级搜索，允许用户按发件人、日期范围、是否有附件、标签进行组合筛选
4. THE FusionMail 系统 SHALL 支持保存搜索条件为智能文件夹，自动应用保存的搜索条件并显示匹配邮件
5. THE FusionMail 系统 SHALL 在搜索结果中高亮显示匹配的关键词

---

### 需求 6：邮件规则引擎

**用户故事**：作为用户，我希望能够创建自动化规则，以便系统自动分类和处理邮件。

#### 验收标准

1. THE FusionMail 系统 SHALL 允许用户创建邮件规则，规则包含触发条件和执行动作
2. THE FusionMail 系统 SHALL 支持以下触发条件：发件人匹配、标题包含关键词、正文包含关键词、有附件、来自特定邮箱账户
3. THE FusionMail 系统 SHALL 支持以下执行动作：添加标签、标记为已读、归档、移入文件夹、触发 Webhook
4. WHEN 新邮件到达时，THE FusionMail 系统 SHALL 按规则优先级顺序执行匹配的规则
5. THE FusionMail 系统 SHALL 允许用户设置规则的优先级，数字越小优先级越高
6. THE FusionMail 系统 SHALL 允许用户启用或禁用规则
7. WHEN 规则执行时，THE FusionMail 系统 SHALL 记录规则执行日志，包括触发时间、匹配邮件和执行结果

---

### 需求 7：Webhook 集成

**用户故事**：作为用户，我希望系统能够在特定邮件事件发生时推送通知到外部系统，以便实现自动化工作流。

#### 验收标准

1. THE FusionMail 系统 SHALL 允许用户配置 Webhook URL，用于接收邮件事件通知
2. THE FusionMail 系统 SHALL 支持以下邮件事件：新邮件到达、邮件标记为已读、邮件归档、邮件删除
3. WHEN 邮件事件触发时，THE FusionMail 系统 SHALL 在 5 秒内向配置的 Webhook URL 发送 POST 请求
4. THE FusionMail 系统 SHALL 在 Webhook 请求中包含邮件的 ID、标题、发件人、时间、正文摘要和事件类型
5. IF Webhook 请求失败，THEN THE FusionMail 系统 SHALL 重试 3 次，重试间隔为 10 秒、30 秒、60 秒
6. THE FusionMail 系统 SHALL 允许用户为 Webhook 配置自定义 HTTP 头部，用于身份验证
7. THE FusionMail 系统 SHALL 记录所有 Webhook 调用日志，包括请求时间、响应状态码和响应时间

---

### 需求 8：RESTful API 接口

**用户故事**：作为开发者，我希望系统提供 API 接口，以便外部系统可以查询和操作邮件数据。

#### 验收标准

1. THE FusionMail 系统 SHALL 提供 RESTful API 接口，支持 JSON 格式的请求和响应
2. THE FusionMail 系统 SHALL 要求 API 调用时提供 API Key 进行身份验证
3. THE FusionMail 系统 SHALL 提供以下 API 端点：获取邮件列表、获取邮件详情、搜索邮件、标记邮件、归档邮件、删除邮件
4. WHEN API 调用获取邮件列表时，THE FusionMail 系统 SHALL 支持分页参数，每页最多返回 100 条记录
5. THE FusionMail 系统 SHALL 提供 API 端点用于管理邮箱账户和触发手动同步操作
6. WHEN API 调用失败时，THE FusionMail 系统 SHALL 返回标准的 HTTP 状态码和错误信息
7. THE FusionMail 系统 SHALL 对 API 调用实施速率限制，每个 API Key 每分钟最多调用 100 次
8. THE FusionMail 系统 SHALL 记录所有 API 调用日志，包括调用时间、端点、参数和响应状态

---

### 需求 9：第三方自动化平台集成

**用户故事**：作为用户，我希望系统能够与主流自动化平台集成，以便快速搭建自动化工作流。

#### 验收标准

1. THE FusionMail 系统 SHALL 提供标准化的 Webhook 格式，兼容 Zapier、Make（Integromat）、n8n 等自动化平台
2. THE FusionMail 系统 SHALL 提供 API 文档，说明所有可用的 API 端点、参数和响应格式
3. WHERE 用户使用自动化平台时，THE FusionMail 系统 SHALL 支持通过 API Key 进行身份验证
4. THE FusionMail 系统 SHALL 提供示例工作流配置，展示如何与常见自动化平台集成

---

### 需求 10：代理管理与健康检查

**用户故事**：作为用户，我希望系统能够管理代理配置并自动检测代理可用性，以便确保邮件同步的稳定性。

#### 验收标准

1. THE FusionMail 系统 SHALL 允许用户为每个邮箱账户配置主代理和备用代理
2. WHEN 主代理连接失败时，THE FusionMail 系统 SHALL 自动切换到备用代理
3. THE FusionMail 系统 SHALL 每 5 分钟检查一次代理的可用性
4. WHEN 代理不可用时，THE FusionMail 系统 SHALL 在系统日志中记录告警信息
5. THE FusionMail 系统 SHALL 显示每个代理的连接状态、响应时间和最后检查时间
6. WHERE 邮箱账户未配置代理时，THE FusionMail 系统 SHALL 使用直连方式访问邮箱服务器

---

### 需求 11：系统监控与日志

**用户故事**：作为管理员，我希望系统提供监控和日志功能，以便及时发现和排查问题。

#### 验收标准

1. THE FusionMail 系统 SHALL 提供健康检查接口，返回系统运行状态
2. THE FusionMail 系统 SHALL 在监控面板中显示以下指标：邮箱账户总数、活跃邮箱数、今日同步次数、今日接收邮件数、API 调用次数
3. THE FusionMail 系统 SHALL 记录结构化日志，包含时间戳、日志级别、模块名称和日志内容
4. THE FusionMail 系统 SHALL 支持按日志级别（DEBUG、INFO、WARN、ERROR）过滤日志
5. WHEN 邮箱连接失败超过 3 次时，THE FusionMail 系统 SHALL 发送告警通知
6. THE FusionMail 系统 SHALL 记录每个邮箱账户的同步历史，包括同步时间、拉取邮件数和错误信息

---

### 需求 12：数据安全与隐私

**用户故事**：作为用户，我希望系统能够保护我的邮箱凭证和邮件数据，以便确保数据安全。

#### 验收标准

1. THE FusionMail 系统 SHALL 使用 AES-256 加密算法存储邮箱账户的密码和 Token
2. THE FusionMail 系统 SHALL 要求用户设置系统登录密码，密码长度至少为 8 个字符
3. THE FusionMail 系统 SHALL 支持会话超时机制，30 分钟无操作后自动登出
4. THE FusionMail 系统 SHALL 记录用户登录日志，包括登录时间、IP 地址和登录结果
5. THE FusionMail 系统 SHALL 支持 HTTPS 加密传输
6. THE FusionMail 系统 SHALL 提供数据导出功能，允许用户导出邮件数据为 JSON 或 CSV 格式
7. THE FusionMail 系统 SHALL 提供数据删除功能，允许用户永久删除所有邮件数据

---

### 需求 13：用户界面与交互

**用户故事**：作为用户，我希望系统提供简洁易用的界面，以便快速完成邮件管理任务。

#### 验收标准

1. THE FusionMail 系统 SHALL 提供响应式 Web 界面，支持桌面和移动设备访问
2. THE FusionMail 系统 SHALL 支持深色模式和浅色模式切换
3. WHEN 用户查看邮件列表时，THE FusionMail 系统 SHALL 支持虚拟滚动，流畅加载大量邮件
4. THE FusionMail 系统 SHALL 在邮件列表顶部显示同步状态指示器
5. THE FusionMail 系统 SHALL 支持快捷键操作，包括标记已读（R）、归档（E）、删除（Delete）、搜索（/）
6. WHEN 新邮件到达时，THE FusionMail 系统 SHALL 显示桌面通知（需用户授权）

---

### 需求 14：部署与配置

**用户故事**：作为管理员，我希望系统易于部署和配置，以便快速启动和运行。

#### 验收标准

1. THE FusionMail 系统 SHALL 支持 Docker 容器化部署
2. THE FusionMail 系统 SHALL 提供 Docker Compose 配置文件，一键启动所有依赖服务
3. THE FusionMail 系统 SHALL 支持通过环境变量配置数据库连接、Redis 连接和系统端口
4. THE FusionMail 系统 SHALL 在首次启动时自动创建数据库表结构
5. THE FusionMail 系统 SHALL 支持数据持久化，数据存储在挂载的 Volume 中
6. THE FusionMail 系统 SHALL 提供安装文档，说明部署步骤和配置方法

---

## 需求优先级

### P0（MVP 核心功能）
- 需求 1：多邮箱账户管理
- 需求 2：后台自动同步
- 需求 3：邮件存储与索引
- 需求 4：邮件查看与基础操作
- 需求 12：数据安全与隐私
- 需求 14：部署与配置

### P1（重要功能）
- 需求 5：邮件搜索与过滤
- 需求 6：邮件规则引擎
- 需求 7：Webhook 集成
- 需求 8：RESTful API 接口
- 需求 11：系统监控与日志

### P2（增强功能）
- 需求 9：第三方自动化平台集成
- 需求 10：代理管理与健康检查
- 需求 13：用户界面与交互

---

## 非功能性需求

### 性能要求
- 邮件列表加载时间不超过 2 秒
- 邮件搜索响应时间不超过 2 秒
- 支持单个邮箱账户存储至少 10 万封邮件
- 支持同时管理至少 20 个邮箱账户

### 可靠性要求
- 系统可用性达到 99.5%
- 邮件同步成功率达到 99%
- 支持自动故障恢复

### 可扩展性要求
- 支持水平扩展，增加服务器节点提升处理能力
- 支持插件机制，允许扩展新的邮箱协议
- API 接口设计遵循 RESTful 规范，便于第三方集成

### 兼容性要求
- 支持主流浏览器（Chrome、Firefox、Safari、Edge）
- 支持移动设备浏览器访问
- 支持 Docker 19.03 及以上版本

---

## 约束条件

1. **当前版本范围**：优先实现邮件接收功能，邮件发送功能暂不在当前版本范围内
2. **功能边界**：当前版本不包含联系人管理功能
3. **状态管理模式**：采用只读镜像模式，邮件状态变更仅在本地生效，不回写到源邮箱
4. **目标用户**：系统主要面向个人用户和小团队（10 人以下）
5. **网络要求**：系统需要支持跨境访问，必须支持代理配置
6. **资源限制**：系统需要轻量化部署，资源占用不超过 2GB 内存

---

## 未来扩展方向

### 短期扩展（下一版本考虑）
1. **邮件发送功能**：
   - 支持通过 SMTP、Gmail API、Microsoft Graph 发送邮件
   - 支持回复和转发邮件
   - 支持草稿保存和定时发送
   - 支持富文本编辑和附件上传
2. **双向状态同步**：支持将本地状态变更回写到源邮箱（可选功能）
3. **联系人管理**：通讯录同步和管理
4. **邮件模板**：常用回复模板和快捷回复
5. **附件存储增强**：
   - 支持更多对象存储服务（腾讯云 COS、华为云 OBS）
   - 支持附件去重和压缩
   - 支持附件病毒扫描

### 中期扩展
5. **邮件加密**：支持 PGP/S/MIME 加密邮件
6. **移动端应用**：iOS 和 Android 原生应用
7. **团队协作**：共享邮箱、邮件分配、协作处理

### 长期扩展
8. **AI 智能功能**：智能分类、自动摘要、智能回复建议
9. **高级协议**：Exchange、IMAP IDLE 实时推送
10. **企业功能**：SSO 单点登录、审计日志、合规管理

---

## 变更记录

| 版本 | 日期 | 变更内容 | 变更人 |
|-----|------|---------|--------|
| v1.7 | 2025-10-28 | 添加附件存储扩展需求（支持对象存储），完善邮件发送扩展方向说明 | - |
| v1.6 | 2025-10-28 | 精简验收标准，将多个需求的详细标准合并为核心指标 | - |
| v1.5 | 2025-10-27 | 明确状态管理策略：采用只读镜像模式（方案 A），邮件状态变更仅在本地生效 | - |
| v1.4 | 2025-10-27 | 澄清版本范围：邮件发送功能暂缓而非永不实现，调整未来扩展方向的优先级 | - |
| v1.3 | 2025-10-27 | 修正协议类型：将 SMTP（发送协议）改为 IMAP/POP3（接收协议），明确协议优先级 | - |
| v1.2 | 2025-10-27 | 更新邮件唯一标识策略，改为 Provider ID + UID 组合，Message-ID 作为辅助 | - |
| v1.1 | 2025-10-27 | 添加邮箱支持矩阵，明确各邮箱服务商的首选和备选方案 | - |
| v1.0 | 2025-10-27 | 初始版本，定义核心需求 | - |

---

**文档版本**：v1.7  
**创建日期**：2025-10-27  
**最后更新**：2025-10-28

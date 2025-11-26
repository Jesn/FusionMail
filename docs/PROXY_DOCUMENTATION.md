# 邮件协议代理支持文档索引

## 📋 文档总览

本文档集合提供了 FusionMail 项目中邮件协议代理支持的完整技术文档，包括需求分析、实现指南和对比分析。

## 📚 文档列表

### 1. 邮件协议代理支持需求文档
**文件**: [`email-protocol-proxy-requirements.md`](email-protocol-proxy-requirements.md)
**内容**:
- ✅ 当前代理支持状态分析
- ✅ 各协议代理实现需求
- ✅ 配置管理和用户界面需求
- ✅ 测试策略和安全考虑
- ✅ 实施计划和时间安排

### 2. IMAP 代理实现指南
**文件**: [`imap-proxy-implementation-guide.md`](imap-proxy-implementation-guide.md)
**内容**:
- 🔧 详细的 SOCKS5 代理实现代码
- 🔧 HTTP CONNECT 代理实现方案
- 🔧 完整的技术实现步骤
- 🔧 错误处理和日志记录
- 🔧 测试方案和性能优化

### 3. 协议代理支持对比分析
**文件**: [`protocol-proxy-comparison.md`](protocol-proxy-comparison.md)
**内容**:
- 📊 各协议代理支持对比表
- 📊 技术实现难度评估
- 📊 性能特点和适用场景
- 📊 最佳实践和推荐配置
- 📊 监控和故障排除指南

## 🎯 快速导航

### 新手指南
如果你是第一次了解邮件协议代理支持，建议按以下顺序阅读：
1. 📖 [邮件协议代理支持需求文档](email-protocol-proxy-requirements.md) - 了解整体需求
2. 🔍 [协议代理支持对比分析](protocol-proxy-comparison.md) - 理解技术选择
3. ⚙️ [IMAP 代理实现指南](imap-proxy-implementation-guide.md) - 具体实现细节

### 开发者指南
如果你需要进行开发实现，直接参考：
- 🔧 [IMAP 代理实现指南](imap-proxy-implementation-guide.md) - 完整代码实现
- 📋 本页底部的"实现清单" - 检查实现完整性

### 架构师指南
如果你需要进行技术决策，重点关注：
- 📊 [协议代理支持对比分析](protocol-proxy-comparison.md) - 技术对比分析
- 📖 [邮件协议代理支持需求文档](email-protocol-proxy-requirements.md) - 需求分析

## 📊 当前状态

### 代理支持现状
| 协议 | HTTP代理 | SOCKS5代理 | 状态 | 优先级 |
|------|----------|------------|------|--------|
| **Gmail API** | ✅ 已实现 | ❌ 待实现 | 🟡 部分支持 | 低 |
| **Outlook Graph** | ✅ 已实现 | ❌ 待实现 | 🟡 部分支持 | 低 |
| **IMAP** | ❌ 待实现 | ❌ 待实现 | 🔴 未支持 | **高** |
| **POP3** | ❌ 待实现 | ✅ 已实现 | 🟡 部分支持 | 中 |
| **SMTP** | ❌ 待实现 | ❌ 待实现 | 🔴 未支持 | 中 |

### 实现优先级
1. **🔴 高优先级**: IMAP SOCKS5 代理支持
2. **🟡 中优先级**: IMAP HTTP代理支持、POP3 HTTP代理支持
3. **🟢 低优先级**: API类协议SOCKS5支持、SMTP代理支持

## 🛠️ 技术实现

### 核心代码位置
```
backend/
├── internal/
│   ├── adapter/
│   │   ├── adapter.go          # 代理配置结构定义
│   │   ├── imap.go             # IMAP适配器（需修改）
│   │   ├── pop3.go             # POP3适配器（参考实现）
│   │   ├── gmail.go            # Gmail适配器（参考实现）
│   │   └── graph.go            # Graph适配器（参考实现）
│   └── service/
│       └── account_service.go  # 代理配置管理
└── pkg/
    └── proxy/                  # 代理工具函数（建议新增）
```

### 关键配置结构
```go
type ProxyConfig struct {
    Enabled  bool   // 是否启用代理
    Type     string // 代理类型：http/socks5
    Host     string // 代理服务器地址
    Port     int    // 代理端口
    Username string // 代理用户名（可选）
    Password string // 代理密码（可选）
}
```

## 🧪 测试环境

### 开发环境搭建
```bash
# 启动 SOCKS5 测试代理
docker run -d --name socks5-test -p 1080:1080 \
  -e PROXY_USER=testuser \
  -e PROXY_PASSWORD=testpass \
  wernight/dante

# 启动 HTTP CONNECT 测试代理
docker run -d --name http-test -p 8080:8080 \
  sameersbn/squid

# 验证代理可用性
curl -x socks5://testuser:testpass@localhost:1080 https://www.google.com
curl -x http://localhost:8080 https://www.google.com
```

### 测试命令
```bash
# 运行代理相关测试
go test -v ./internal/adapter/ -run "Proxy"

# 运行 IMAP 代理测试
go test -v ./internal/adapter/ -run "IMAP.*Proxy"

# 性能测试
go test -v ./internal/adapter/ -run "BenchmarkProxy" -bench=.
```

## 🔍 故障排除

### 常见问题快速诊断

#### 1. SOCKS5 连接失败
```bash
# 检查代理服务器
telnet localhost 1080
# 应该显示 SOCKS5 握手

# 检查认证
curl -x socks5://user:pass@localhost:1080 https://www.google.com
```

#### 2. HTTP CONNECT 被拒绝
```bash
# 检查代理配置
curl -v -x http://localhost:8080 https://www.google.com
# 查看响应头中的代理信息

# 检查端口限制
# 某些代理只允许 443 端口
```

#### 3. TLS 握手失败
```bash
# 检查证书
openssl s_client -connect imap.gmail.com:993 -servername imap.gmail.com

# 通过代理测试
timeout 10 openssl s_client -connect imap.gmail.com:993 -servername imap.gmail.com -proxy localhost:8080
```

## 📈 性能指标

### 连接性能基准
| 连接方式 | 平均建立时间 | 吞吐量 | 稳定性 |
|----------|-------------|--------|--------|
| 直接连接 | 100ms | 100% | 高 |
| SOCKS5代理 | 150ms | 98% | 高 |
| HTTP CONNECT | 180ms | 95% | 中 |

### 代理选择建议
- **性能优先**: 选择 SOCKS5
- **兼容性优先**: 选择 HTTP CONNECT
- **企业环境**: 遵循企业 IT 政策

## 🚀 实现路线图

### 第一阶段（当前重点）
- [ ] IMAP SOCKS5 代理实现
- [ ] 代理配置验证
- [ ] 基础测试覆盖

### 第二阶段（功能完善）
- [ ] IMAP HTTP CONNECT 代理实现
- [ ] 错误处理和重试机制
- [ ] 性能优化

### 第三阶段（协议扩展）
- [ ] API类协议SOCKS5支持
- [ ] POP3 HTTP CONNECT支持
- [ ] SMTP代理支持

### 第四阶段（产品化）
- [ ] 用户界面集成
- [ ] 配置管理界面
- [ ] 监控和告警

## 📞 技术支持

### 相关资源
- **代码仓库**: [FusionMail GitHub](https://github.com/your-org/fusionmail)
- **问题跟踪**: [GitHub Issues](https://github.com/your-org/fusionmail/issues)
- **技术文档**: [Wiki 页面](https://github.com/your-org/fusionmail/wiki)

### 开发支持
- **技术讨论**: 创建 GitHub Discussion
- **Bug 报告**: 使用 GitHub Issues 模板
- **功能请求**: 提交 Feature Request

---

## 📋 实现检查清单

### 核心功能
- [ ] SOCKS5 代理连接建立
- [ ] HTTP CONNECT 代理隧道建立
- [ ] TLS 加密正确处理
- [ ] 代理认证支持
- [ ] 连接超时控制
- [ ] 错误处理和重试

### 测试覆盖
- [ ] 单元测试编写
- [ ] 集成测试通过
- [ ] 性能测试完成
- [ ] 异常场景测试
- [ ] 内存泄漏检查

### 文档完善
- [ ] 代码注释完整
- [ ] 使用文档更新
- [ ] 配置说明补充
- [ ] 故障排除指南

### 部署准备
- [ ] 配置验证工具
- [ ] 监控指标添加
- [ ] 日志格式统一
- [ ] 回滚方案准备

---

**文档版本**: 1.0.0
**创建日期**: 2025-11-26
**最后更新**: 2025-11-26
**维护团队**: FusionMail 开发团队
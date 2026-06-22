# CONTEXT.md — go-ldap-admin 领域文档

## 系统定位

`go-ldap-admin` 是企业内网的**用户目录同步与管理中台**。它以 OpenLDAP 为核心用户目录，将飞书的组织架构和员工数据单向同步进来，同时提供 Web UI 供管理员手动管理用户、分组和权限。

LDAP 目录对外服务两类消费者：
- 内网应用（SSO、VPN 等）通过 LDAP bind 认证
- WiFi RADIUS 服务器通过 PEAP/MSCHAPv2 认证（依赖 `sambaNTPassword`）

## 系统边界

本项目**不是**整个认证体系的唯一服务，须与另一个独立服务协同：

| 服务 | 职责 | 写入 LDAP 的字段 |
|---|---|---|
| **go-ldap-admin**（本项目） | 夜间批量同步飞书组织架构；Web UI 用户管理 | 除 `sambaNTPassword` 以外的所有用户属性 |
| **飞书机器人 Go 后端**（独立项目） | 实时响应员工改密请求 | `sambaNTPassword`（NT Hash） |

两个服务写的 LDAP 属性互不重叠，不存在竞争写冲突。

## 核心领域概念

### 用户（User）

LDAP 中的 `inetOrgPerson + extensibleObject` 条目。本地 MySQL 中有对应记录作为管理态数据。

关键字段：

| 字段 | LDAP 属性 | 含义 |
|---|---|---|
| `username` | `uid` / `cn` | 登录名，由 `custom_uid_short` 算法生成 |
| `nickname` | `sn` / `displayName` | 中文姓名 |
| `source_user_id` | `employeeType` | 飞书 user_id（带前缀，如 `feishu_ou_xxx`）；**去重唯一键** |
| `source_union_id` | — | 飞书 union_id，用于离职判断 |
| `mobile` | `mobile` | 手机号；老数据迁移时作为兜底认领键 |

### custom_uid_short（短 UID）

从飞书中文姓名生成 LDAP `uid` 的算法：
- **规则**：姓氏全拼 + 名字每个字的拼音首字母
- **复姓白名单**：内置 30 个常见复姓（欧阳、司马等），避免错误拆分
- **冲突处理**：若 `uid` 已存在，追加递增数字后缀（`zhangw` → `zhangw2` → `zhangw3`）
- **实现**：`public/tools/type.go` 中的 `ConvertToUIDShort()`

### 分组（Group）

对应飞书部门，映射到 LDAP 的 `cn=xxx,ou=xxx` 条目。层级结构通过 `source_dept_parent_id` 维护。

### 同步流程

飞书同步为**单向**、**定时**批处理：

```
飞书 API → GetAllDepts/GetAllUsers → ConvertDeptData/ConvertUserData
         → AddDepts/AddUsers → CommonAddGroup/CommonAddUser
         → 写 LDAP + 写 MySQL
```

离职处理：调用飞书 EHR API 获取离职用户 ID，逐页翻页（每页 100 条），从 LDAP 删除条目，MySQL 状态置为 2（离职）。

### 字段关系（FieldRelation）

飞书原始字段名到本系统字段名的映射表，存储在 MySQL，可通过 Web UI 配置。默认 `username` 映射为 `custom_uid_short`（虚拟字段，由飞书客户端代码注入）。

## 不在本项目范围内的事

- `sambaNTPassword` 的生成和写入 → 由飞书机器人 Go 后端负责
- WiFi RADIUS 认证逻辑 → 由 FreeRADIUS 或类似服务负责
- 飞书之外的 IM 同步（钉钉、企业微信）→ 上游原始功能，本项目未二开

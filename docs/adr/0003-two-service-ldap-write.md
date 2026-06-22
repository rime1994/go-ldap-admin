# ADR-0003：两个服务共享 LDAP，写入属性互不重叠

**状态**：已采用  
**日期**：2026-06

## 背景

WiFi RADIUS 认证（PEAP/MSCHAPv2）需要 LDAP 条目中有 `sambaNTPassword`（NT Hash）。但 NT Hash 只能从明文密码计算，go-ldap-admin 的同步流程使用配置中的初始密码且不持有用户的真实明文密码。

## 决策

`sambaNTPassword` 的写入由**独立的飞书机器人 Go 后端**负责：员工通过飞书机器人修改密码时，该服务实时计算 NT Hash 并写入 LDAP。

go-ldap-admin 的职责是：
1. 在创建用户时加入 `extensibleObject` objectClass（否则 OpenLDAP schema 不允许写 `sambaNTPassword`）
2. 存储飞书 `user_id` 到 `employeeType` 字段，供飞书机器人后端关联用户

两个服务写入的 LDAP 属性集合不重叠，定时任务（夜间）与实时写入（白天）在时间上也不冲突。

## 后果

- 架构边界清晰，两个服务可独立部署和迭代
- `sambaNTPassword` 依赖员工主动通过飞书机器人改密才会写入；新同步进来但从未改过密码的用户没有 NT Hash，无法 WiFi 认证
- `employeeType` 字段被挪作存储飞书 user_id，偏离该字段的语义原意（LDAP 标准中 employeeType 用于职位类型），需注意与其他 LDAP 消费者的兼容性

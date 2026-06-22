# ADR-0002：以 source_user_id 作为飞书用户去重键

**状态**：已采用  
**日期**：2026-06-08

## 背景

上游原始代码以 `user_dn` 作为去重键。但 `user_dn` 依赖 `uid`，而 `uid` 由 `custom_uid_short` 生成——同名员工在不同时间同步时可能得到不同后缀，导致同一个人被重复创建。

## 决策

改用飞书的 `source_user_id`（格式：`feishu_<open_user_id>`）作为唯一键。这是飞书分配给员工的不变 ID，与姓名无关。

同时增加**老数据迁移兜底**：若 `source_user_id` 未命中，但 `mobile` 已存在于数据库，则认领该旧记录并补写 `source_user_id`，后续走更新分支而非重复创建。

实现位于 `logic/feishu_logic.go` 的 `AddUsers()`。

## 后果

- 同步幂等性大幅提升，重复触发同步不会产生重复用户
- 依赖飞书 `open_user_id` 的稳定性（飞书官方承诺此 ID 不变）
- 老数据若既无 `source_user_id` 又无 `mobile`，无法自动认领，需手动处理

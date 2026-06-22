# ADR-0001：用中文姓名拼音生成 LDAP uid（custom_uid_short）

**状态**：已采用  
**日期**：2026-06-08

## 背景

飞书同步过来的员工数据只有中文姓名，没有预设的英文登录名。LDAP `uid` 必须是 ASCII 字符串，且在目录中唯一。

## 决策

实现 `ConvertToUIDShort()`，规则为：**姓氏全拼 + 名字每个字的拼音首字母**。

示例：
- 张伟 → `zhangw`
- 欧阳娜娜 → `ouyangnn`（复姓整体保留）

内置 30 个复姓白名单，避免将"欧阳"拆成"欧"+"阳"。

冲突时追加递增数字后缀：`zhangw` → `zhangw2` → `zhangw3`，由 `uniqueUsername()` 在 AddUsers 时处理。

飞书 FieldRelation 的默认 `username` 映射指向虚拟字段 `custom_uid_short`，由 `public/client/feishu/feishu.go` 在构建原始数据时注入。

## 后果

- uid 对中文名友好，比随机字符串可读性高
- 同名员工会得到带数字后缀的 uid，轻微影响可读性，但可接受
- 复姓白名单需要人工维护，遇到未收录的复姓会产生错误拆分

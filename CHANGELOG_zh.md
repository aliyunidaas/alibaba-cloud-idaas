# 更新日志

[English](CHANGELOG.md)

已发布版本见 [GitHub Releases](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases)。本文件按版本倒序汇总主要变更，标注“待发布”的条目属于开发构建。

### v0.2.5-beta

2026-11-11 已发布。[发布说明](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases/tag/v0.2.5-beta)。

- 修复隐式默认指向非 cloud profile 时的无歧义角色发现兜底，保留显式选择与歧义保护，并补齐命令参数表。

- 增加可选的无凭证 onboard --result-file JSON、完整只读状态标记及共用 profile 登录参数解析；补齐 Skill 授权阶段口径和旧配置路径优先级。

- 修复 AWS 中国区/GovCloud profile 的地域选择，保留兼容的用户地域，增加显式 --aws-region 覆盖。
- 不再自动建立 AWS default，保留既有默认及 SSO；增加仅修改 Broker 的 configure switch。
- 实例/client-id 推断改为确定性、同实例选择，保护已有身份名称及未知配置字段。
- 配置写入增加校验、原子替换与准确的部分失败退出码，修正各工具示例和外部命令路径。
- 增加只读客户端/默认身份诊断，完善查询命令及中英文帮助、迁移说明。
- 兼容性：未知 target、上下文歧义及写入失败明确报错；部分阿里云消费端要求无空格路径。腾讯云已有静态凭证文件不再静默覆盖；AWS 尊重自定义配置路径并检查共享凭证冲突。保留原配置版本和凭证缓存行为。

### v0.2.4-beta

2026-11-05 已发布。[发布说明](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases/tag/v0.2.4-beta)。

- `onboard` 和 `show roles` 新增权限模板角色发现，展示“模板创建”属性，以 yes/no/unknown 表示是／否／未知，不代表授权来源。
- 修复角色发现仅返回第一页的问题。两个命令均取完全部分页后，再展示完整结果或生成配置。
- 修复错误响应被当作空角色列表的问题。请求失败、响应无效或游标重复时明确报错，不返回部分结果。
- 调整 `show roles --json`，输出完整的 `entities` 数组和实际汇总的 `totalCount`，不输出单页游标或单页 `maxResults`。

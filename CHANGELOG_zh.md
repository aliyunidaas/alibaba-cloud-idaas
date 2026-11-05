# 更新日志

[English](CHANGELOG.md)

已发布版本见 [GitHub Releases](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases)。本文件按版本倒序汇总主要变更，标注“待发布”的条目属于开发构建。

### v0.2.4-beta

待发布。

- `onboard` 和 `show roles` 新增权限模板角色发现，展示“模板创建”属性，以 yes/no/unknown 表示是／否／未知，不代表授权来源。
- 修复角色发现仅返回第一页的问题。两个命令均取完全部分页后，再展示完整结果或生成配置。
- 修复错误响应被当作空角色列表的问题。请求失败、响应无效或游标重复时明确报错，不返回部分结果。
- 调整 `show roles --json`，输出完整的 `entities` 数组和实际汇总的 `totalCount`，不输出单页游标或单页 `maxResults`。

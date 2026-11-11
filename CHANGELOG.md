# CHANGE LOG

[简体中文](CHANGELOG_zh.md)

For published releases, see [GitHub Releases](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases). This file summarizes the major changes in each version, newest first. Entries marked as pending release describe development builds.

### v0.2.5-beta

Pending release.

- Restored unambiguous cloud discovery when the implicit default profile is non-cloud, while preserving explicit selections and rejecting ambiguous contexts; completed the command argument tables.

- Added opt-in credential-free onboard --result-file JSON, complete read-only status markers, and a shared profile-login context resolver; clarified Skill authorization stages and legacy config precedence.

- Fixed AWS China/GovCloud profile regions and preserved compatible user-selected regions. Added an explicit --aws-region override.
- Stopped silently creating AWS default profiles; preserved existing defaults and SSO configuration. Added explicit Broker-only configure switch.
- Made instance/client-id resolution deterministic and scoped to the selected instance. Protected existing identity names and unknown configuration fields.
- Added validated, atomic configuration writes and honest partial-failure exit status. Corrected target-specific examples and external command paths.
- Added read-only client/default diagnostics, implemented safe query behavior, and aligned English/Chinese help and documentation.
- Compatibility: unknown targets/ambiguous context and failed writes now fail explicitly; some Alibaba consumers require paths without spaces. Existing TencentCloud credential files are never silently replaced. AWS custom configuration paths and shared credential conflicts are checked. Existing config version and credential caching are unchanged.

### v0.2.4-beta

Released on 2026-11-05. [Release notes](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases/tag/v0.2.4-beta).

- Added permission-template role discovery to `onboard` and `show roles`, including a yes/no/unknown template-created attribute that does not indicate the authorization source.
- Fixed role discovery returning only the first page. Both commands now fetch all pages before displaying the complete result or generating profiles.
- Fixed error responses being treated as empty role lists. Discovery now fails explicitly on request errors, invalid responses or repeated cursors instead of returning partial results.
- Changed `show roles --json` to return the complete `entities` array and collected `totalCount`, without a page cursor or per-page `maxResults`.

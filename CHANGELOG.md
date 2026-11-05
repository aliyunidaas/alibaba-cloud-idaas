# CHANGE LOG

[简体中文](CHANGELOG_zh.md)

For published releases, see [GitHub Releases](https://github.com/aliyunidaas/alibaba-cloud-idaas/releases). This file summarizes the major changes in each version, newest first. Entries marked as pending release describe development builds.

### v0.2.4-beta

Pending release.

- Added permission-template role discovery to `onboard` and `show roles`, including a yes/no/unknown template-created attribute that does not indicate the authorization source.
- Fixed role discovery returning only the first page. Both commands now fetch all pages before displaying the complete result or generating profiles.
- Fixed error responses being treated as empty role lists. Discovery now fails explicitly on request errors, invalid responses or repeated cursors instead of returning partial results.
- Changed `show roles --json` to return the complete `entities` array and collected `totalCount`, without a page cursor or per-page `maxResults`.

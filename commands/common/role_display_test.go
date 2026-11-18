package common

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

func TestRoleDisplayMetadata(t *testing.T) {
	long := strings.Repeat("完整描述", 32)
	role := AssumableRole{
		CloudAccountName: "生产账号", CloudAccountDescription: long,
		CloudPermissionTemplateName: "只读模板", CloudPermissionTemplateDescription: long,
		CloudAccountRoleName: "real-role", CloudAccountRoleDescription: "角色描述",
	}
	if role.DisplayDescription() != long || RoleDisplayCell(long) != long {
		t.Fatal("full template description must be preserved")
	}
	encoded, err := json.Marshal(role)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AssumableRole
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CloudAccountName != role.CloudAccountName || decoded.CloudAccountDescription != long || decoded.CloudPermissionTemplateName != role.CloudPermissionTemplateName || decoded.CloudAccountRoleName != "real-role" || decoded.DisplayDescription() != long {
		t.Fatal("metadata lost in structured output")
	}
	role.CloudPermissionTemplateDescription = " \t\n"
	if role.DisplayDescription() != "角色描述" {
		t.Fatal("must fall back to role description")
	}
	role.CloudAccountRoleDescription = ""
	if RoleDisplayCell(role.DisplayDescription()) != "-" {
		t.Fatal("empty description must use placeholder")
	}
}

func TestRoleDisplayOldResponseAndControlCharacters(t *testing.T) {
	var role AssumableRole
	if err := json.Unmarshal([]byte(`{"cloudAccountRoleName":"legacy"}`), &role); err != nil {
		t.Fatal(err)
	}
	if role.CloudAccountRoleName != "legacy" || RoleDisplayCell(role.CloudAccountName) != "-" || role.CloudPermissionTemplateName != "" {
		t.Fatal("old response must remain usable without invented metadata")
	}
	if got := RoleDisplayCell("a\nb\rc\td\x1b"); got != `a\nb\rc\td\x1b` {
		t.Fatalf("unsafe terminal controls: %q", got)
	}
}

func TestRoleDisplayNullMetadataPreservesLegacyIdentity(t *testing.T) {
	var role AssumableRole
	input := `{"cloudAccountId":"account-1","cloudAccountRoleId":"role-1","cloudAccountRoleName":"legacy-role","cloudAccountRoleExternalId":"role-arn","cloudAccountVendorType":"aliyun","status":"enabled","createdByPermissionTemplate":null,"cloudAccountName":null,"cloudAccountDescription":null,"cloudPermissionTemplateId":null,"cloudPermissionTemplateName":null,"cloudPermissionTemplateDescription":null,"cloudAccountRoleDescription":null,"futureField":"ignored"}`
	if err := json.Unmarshal([]byte(input), &role); err != nil {
		t.Fatal(err)
	}
	if role.CloudAccountId != "account-1" || role.CloudAccountRoleId != "role-1" || role.CloudAccountRoleExternalId != "role-arn" || role.CloudAccountVendorType != "aliyun" || role.Status != "enabled" || role.CreatedByPermissionTemplate != nil {
		t.Fatalf("legacy identity or unknown template state changed: %#v", role)
	}
	if got := FormatRoleTable([]AssumableRole{role}); !strings.Contains(got, "1  -\t-\t-\tlegacy-role\t-\n") {
		t.Fatalf("null metadata must degrade without losing role: %q", got)
	}
}

func TestFormatRoleTableKeepsIdentityAndFullDescriptions(t *testing.T) {
	long := strings.Repeat("完整描述", 32)
	got := FormatRoleTable([]AssumableRole{
		{CloudAccountName: "生产账号", CloudAccountDescription: long, CloudPermissionTemplateName: "只读模板", CloudAccountRoleName: "real-role", CloudPermissionTemplateDescription: long, CloudAccountRoleDescription: "fallback"},
		{CloudAccountName: "研发账号", CloudAccountRoleName: "ordinary-role", CloudAccountRoleDescription: "角色描述"},
	})
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 3 || strings.Count(lines[0], "\t") != 4 || !strings.HasPrefix(lines[0], "#  CLOUD ACCOUNT NAME\t") {
		t.Fatalf("expected six columns and two roles: %q", got)
	}
	if lines[1] != "1  生产账号\t"+long+"\t只读模板\treal-role\t"+long {
		t.Fatalf("template role metadata truncated or substituted: %q", lines[1])
	}
	if lines[2] != "2  研发账号\t-\t-\tordinary-role\t角色描述" {
		t.Fatalf("ordinary role fallback incorrect: %q", lines[2])
	}
}

func TestRoleDisplayEscapesAllTerminalControls(t *testing.T) {
	input := "角色\x00\a\b\v\f\x7f\u0085\u009b31m\u2028\u2029结束"
	got := RoleDisplayCell(input)
	for _, r := range got {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			t.Fatalf("unescaped control in display: %q", got)
		}
	}
	if !strings.HasPrefix(got, "角色\\x00\\a\\b\\v\\f\\x7f") || !strings.HasSuffix(got, "结束") {
		t.Fatalf("display must retain visible text and escaped controls: %q", got)
	}
	role := AssumableRole{CloudAccountRoleDescription: input}
	RoleDisplayCell(role.DisplayDescription())
	if role.CloudAccountRoleDescription != input {
		t.Fatal("display escaping must not alter raw metadata")
	}
}

func TestFormatRoleTableMultiDigitNumbers(t *testing.T) {
	roles := make([]AssumableRole, 10)
	for i := range roles {
		roles[i].CloudAccountRoleName = "legacy-role"
	}
	lines := strings.Split(strings.TrimSuffix(FormatRoleTable(roles), "\n"), "\n")
	if len(lines) != 11 || !strings.HasPrefix(lines[1], "1   -\t") || !strings.HasPrefix(lines[10], "10  -\t") {
		t.Fatalf("numbers must share minimal width without left padding: %q", lines)
	}
	for _, line := range lines[1:] {
		if !strings.HasSuffix(line, "\tlegacy-role\t-") {
			t.Fatalf("legacy roles must remain identifiable: %q", line)
		}
	}
}

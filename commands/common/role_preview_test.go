package common

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestRolePreviewBoundaryAndFullExport(t *testing.T) {
	for _, count := range []int{0, 49, 50, 51, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			roles := make([]AssumableRole, count)
			for i := range roles {
				roles[i].CloudAccountRoleName = fmt.Sprintf("unique-role-%03d", i+1)
			}
			preview, filename, err := PrepareRolePreview(roles)
			if err != nil {
				t.Fatal(err)
			}
			if count <= 50 {
				if filename != "" || preview != FormatRoleTable(roles) {
					t.Fatal("small lists must remain complete without an export")
				}
				return
			}
			defer os.Remove(filename)
			full, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if string(full) != FormatRoleTable(roles) {
				t.Fatal("export must preserve every role and numbering")
			}
			if preview != FormatRoleTable(roles[:50]) || strings.Contains(preview, "unique-role-051") {
				t.Fatal("preview must contain exactly the first 50 roles")
			}
			info, err := os.Stat(filename)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
				t.Fatal("export must be private")
			}
		})
	}
}

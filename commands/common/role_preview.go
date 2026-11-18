package common

import (
	"fmt"
	"os"
)

// RolePreviewLimit bounds only human-readable output, never discovery or JSON.
const RolePreviewLimit = 50

// Keep JSON output complete; limit only the human-readable preview.
func PrepareRolePreview(roles []AssumableRole) (string, string, error) {
	if len(roles) <= RolePreviewLimit {
		return FormatRoleTable(roles), "", nil
	}
	file, err := os.CreateTemp("", "idaas-roles-*.txt")
	if err != nil {
		return "", "", fmt.Errorf("create full role list: %w", err)
	}
	filename := file.Name()
	_, writeErr := file.WriteString(FormatRoleTable(roles))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(filename)
		if writeErr != nil {
			return "", "", fmt.Errorf("write full role list: %w", writeErr)
		}
		return "", "", fmt.Errorf("close full role list: %w", closeErr)
	}
	return FormatRoleTable(roles[:RolePreviewLimit]), filename, nil
}

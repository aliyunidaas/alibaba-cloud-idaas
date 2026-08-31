package logout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var (
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "Profile to logout (clears all if not specified)",
	}
	boolFlagDryRun = &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Show what would be cleared without deleting",
	}
)

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Clear cached tokens (profile config is preserved)",
		Flags: []cli.Flag{stringFlagProfile, boolFlagDryRun},
		Action: func(c *cli.Context) error {
			return runLogout(c.String("profile"), c.Bool("dry-run"))
		},
	}
}

func runLogout(profile string, dryRun bool) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	if profile == "" {
		// Clear all cache
		utils.Stdout.Printf("Clearing all cache in %s ...\n", cacheDir)
		if dryRun {
			listFiles(cacheDir)
			return nil
		}
		return os.RemoveAll(cacheDir)
	}

	// Load config to do smart profile-specific cache clearing.
	cloudCredentialConfig, err := config.LoadCloudCredentialConfig("")
	if err != nil {
		return errors.Wrap(err, "failed to load config for profile logout")
	}
	if cloudCredentialConfig == nil {
		return errors.New("no config found")
	}
	_, targetProfileConfig := cloudCredentialConfig.FindProfile(profile)
	if targetProfileConfig == nil {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	targetKeys := collectOidcCacheKeys(targetProfileConfig)
	otherKeys := make(map[string]struct{})
	for otherProfile, otherProfileConfig := range cloudCredentialConfig.Profile {
		if otherProfile == profile {
			continue
		}
		for _, key := range collectOidcCacheKeys(otherProfileConfig) {
			otherKeys[key] = struct{}{}
		}
	}

	utils.Stdout.Printf("Clearing cache for profile '%s' in %s ...\n", profile, cacheDir)

	var cleared, skipped []string

	// 1. Delete oidc_token / token_response files used only by this profile.
	for _, category := range []string{constants.CategoryOidcToken, constants.CategoryTokenResponse} {
		categoryDir := filepath.Join(cacheDir, category)
		entries, readErr := os.ReadDir(categoryDir)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return errors.Wrapf(readErr, "failed to read %s cache dir", category)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !isTargetOidcKey(name, targetKeys) {
				continue
			}
			path := filepath.Join(categoryDir, name)
			if _, shared := otherKeys[name]; shared {
				skipped = append(skipped, fmt.Sprintf("  skipped (shared with other profile): %s", path))
				continue
			}
			if dryRun {
				cleared = append(cleared, fmt.Sprintf("  would delete: %s", path))
			} else {
				if err := os.Remove(path); err != nil {
					return errors.Wrapf(err, "failed to delete %s", path)
				}
				cleared = append(cleared, fmt.Sprintf("  deleted: %s", path))
			}
		}
	}

	// 2. Delete cloud_token files owned by this profile.
	cloudTokenDir := filepath.Join(cacheDir, constants.CategoryCloudToken)
	cloudTokenEntries, cloudTokenReadErr := os.ReadDir(cloudTokenDir)
	if cloudTokenReadErr != nil {
		if !os.IsNotExist(cloudTokenReadErr) {
			return errors.Wrapf(cloudTokenReadErr, "failed to read %s cache dir", constants.CategoryCloudToken)
		}
	} else {
		for _, entry := range cloudTokenEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !isCloudTokenFileForProfile(name, profile) {
				continue
			}
			path := filepath.Join(cloudTokenDir, name)
			if dryRun {
				cleared = append(cleared, fmt.Sprintf("  would delete: %s", path))
			} else {
				if err := os.Remove(path); err != nil {
					return errors.Wrapf(err, "failed to delete %s", path)
				}
				cleared = append(cleared, fmt.Sprintf("  deleted: %s", path))
			}
		}
	}

	for _, line := range cleared {
		utils.Stdout.Println(line)
	}
	for _, line := range skipped {
		utils.Stdout.Println(line)
	}

	if len(cleared) == 0 && len(skipped) == 0 {
		utils.Stdout.Println("No cache files found for this profile.")
		return nil
	}
	utils.Stdout.Println("Logout complete. Profile config preserved.")
	return nil
}

func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir")
	}
	return filepath.Join(home, constants.ConfigRootDir, constants.ConfigIdaasDir), nil
}

func collectOidcCacheKeys(profileConfig *config.CloudStsConfig) []string {
	var keys []string
	for _, provider := range profileConfig.OidcTokenProviders() {
		if provider != nil {
			keys = append(keys, provider.GetCacheKey())
		}
	}
	return keys
}

func isTargetOidcKey(name string, targetKeys []string) bool {
	for _, key := range targetKeys {
		if name == key {
			return true
		}
	}
	return false
}

var hex32Regexp = regexp.MustCompile("^[0-9a-f]{32}$")

func isCloudTokenFileForProfile(name, profile string) bool {
	prefix := profile + "_"
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	suffix := name[len(prefix):]
	return hex32Regexp.MatchString(suffix)
}

func listFiles(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		utils.Stdout.Printf("  would delete: %s\n", path)
		return nil
	})
}

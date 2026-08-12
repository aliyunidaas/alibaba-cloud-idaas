package logout

import (
	"os"
	"path/filepath"
	"strings"

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
	// Clear specific profile's cache files
	utils.Stdout.Printf("Clearing cache for profile '%s' in %s ...\n", profile, cacheDir)
	var cleared []string
	err = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, profile+"_") {
			cleared = append(cleared, path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walk cache dir failed")
	}
	if len(cleared) == 0 {
		utils.Stdout.Println("No cache files found for this profile.")
		return nil
	}
	for _, f := range cleared {
		if dryRun {
			utils.Stdout.Printf("  would delete: %s\n", f)
		} else {
			os.Remove(f)
			utils.Stdout.Printf("  deleted: %s\n", f)
		}
	}
	utils.Stdout.Println("Logout complete. Profile config preserved.")
	return nil
}

func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir")
	}
	return filepath.Join(home, ".aliyun", constants.ConfigIdaasDir), nil
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

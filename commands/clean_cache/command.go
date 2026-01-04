package clean_cache

import (
	"os"
	"path/filepath"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func BuildCommand() *cli.Command {
	var flags []cli.Flag
	return &cli.Command{
		Name:  "clean-cache",
		Usage: "Clean cache",
		Flags: flags,
		Action: func(context *cli.Context) error {
			return cleanCache()
		},
	}
}

func cleanCache() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "failed to get user home dir")
	}

	logsCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.LogDir)
	deleteFiles(logsCacheDir, func(filename string) bool {
		return !idaaslog.IsCurrentLog(filename)
	})

	oidcCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.CategoryOidc)
	deleteFiles(oidcCacheDir, func(filename string) bool { return true })

	oidcTokenCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.CategoryOidcToken)
	deleteFiles(oidcTokenCacheDir, func(filename string) bool { return true })

	cloudTokenCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.CategoryCloudToken)
	deleteFiles(cloudTokenCacheDir, func(filename string) bool { return true })

	tokenResponseCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.CategoryTokenResponse)
	deleteFiles(tokenResponseCacheDir, func(filename string) bool { return true })

	return nil
}

func deleteFiles(dir string, acceptFileFilter func(string) bool) {
	utils.Stderr.Fprintf("Clearing dir: %s ...\n", dir)
	logFiles, err := os.ReadDir(dir)
	if err != nil {
		utils.Stderr.Fprintf("Read dir: %s failed: %s\n", dir, err)
	}
	skipFileCount := 0
	deleteFileCount := 0
	for _, file := range logFiles {
		if !file.IsDir() {
			if !acceptFileFilter(file.Name()) {
				continue
			}
			removeErr := os.Remove(filepath.Join(dir, file.Name()))
			if removeErr != nil {
				utils.Stderr.Fprintf("Remove file: %s failed: %s\n", file.Name(), removeErr)
			} else {
				deleteFileCount++
				utils.Stderr.Fprintf("Remove file: %s success\n", file.Name())
			}
		}
	}
	utils.Stderr.Fprintf("Delete: %d file(s), skipped: %d\n", deleteFileCount, skipFileCount)
	utils.Stderr.Println("")
}

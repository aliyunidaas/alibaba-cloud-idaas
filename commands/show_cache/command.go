package show_cache

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
	stringFlagCategory = &cli.StringFlag{
		Name:    "category",
		Aliases: []string{"c"},
		Usage:   "Category",
	}
	stringFlagName = &cli.StringFlag{
		Name:    "name",
		Aliases: []string{"n"},
		Usage:   "Name",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagCategory,
		stringFlagName,
	}
	return &cli.Command{
		Name:  "show-cache",
		Usage: "Show cache",
		Flags: flags,
		Action: func(context *cli.Context) error {
			category := context.String("category")
			name := context.String("name")
			return showCache(category, name)
		},
	}
}

func showCache(category, name string) error {
	if category == "" && name == "" {
		var categories []string
		categories = append(categories, "- "+constants.CategoryOidc)
		categories = append(categories, "- "+constants.CategoryOidcToken)
		categories = append(categories, "- "+constants.CategoryCloudToken)
		if len(categories) == 0 {
			utils.Stdout.Fprintf("No categories found\n")
		} else {
			utils.Stdout.Fprintf("Categories: \n%s\n", strings.Join(categories, "\n"))
		}
		return nil
	}

	if name == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "failed to get user home dir")
		}
		cacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, category)
		logFiles, err := os.ReadDir(cacheDir)
		if err != nil {
			utils.Stderr.Fprintf("Read dir: %s failed: %s\n", cacheDir, err)
			return nil
		}
		var files []string
		for _, file := range logFiles {
			if !file.IsDir() {
				files = append(files, "- "+file.Name())
			}
		}
		if len(files) == 0 {
			utils.Stdout.Fprintf("No files found in category: %s\n", category)
		} else {
			utils.Stdout.Fprintf("Files in category: %s: \n%s\n", category, strings.Join(files, "\n"))
		}
		return nil
	}

	data, err := utils.ReadCacheFileWithEncryption(category, name)
	if err != nil {
		utils.Stderr.Fprintf("Read cache file [%s, %s] with encryption failed: %v\n", category, name, err)
		return nil
	}
	utils.Stdout.Fprintf("%s\n", data)
	return nil
}

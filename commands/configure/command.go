package configure

import (
	"encoding/json"
	"fmt"

	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/common"
	"github.com/aliyunidaas/alibaba-cloud-idaas/config"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/urfave/cli/v2"
)

// This command changes the Broker default only. Cloud CLI defaults are independent.
func BuildCommand() *cli.Command {
	return &cli.Command{Name: "configure", Usage: "Explicitly configure the Broker (cloud CLI defaults are unchanged)", Subcommands: []*cli.Command{{
		Name: "switch", Usage: "Set the Broker current_profile without changing cloud CLI defaults",
		Flags:  []cli.Flag{&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Required: true}, &cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Broker config path"}},
		Action: func(c *cli.Context) error { return switchProfile(c.String("config"), c.String("profile")) },
	}}}
}

func switchProfile(filename, profile string) error {
	if filename == "" {
		var err error
		filename, err = config.GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return err
		}
	}
	err := common.UpdateConfigFile(filename, func(original []byte) ([]byte, error) {
		cfg, err := config.ReadCloudCredentialConfig(filename)
		if err != nil {
			return nil, err
		}
		if cfg == nil || cfg.Profile[profile] == nil {
			return nil, fmt.Errorf("Broker profile %s not found; no default changed", profile)
		}
		root, err := common.DecodeConfigObject(original)
		if err != nil {
			return nil, err
		}
		root["current_profile"] = profile
		return json.MarshalIndent(root, "", "  ")
	})
	if err == nil {
		utils.Stdout.Printf("Broker default is %s; cloud CLI defaults are unchanged.\n", profile)
	}
	return err
}

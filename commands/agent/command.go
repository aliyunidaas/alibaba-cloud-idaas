package agent

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/access_token"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/decrypt_secret"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/get_secret"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/put_secret"
	"github.com/aliyunidaas/alibaba-cloud-idaas/commands/agent/token_exchange"
	"github.com/urfave/cli/v2"
)

func BuildCommand() *cli.Command {
	var flags []cli.Flag
	return &cli.Command{
		Name:  "agent",
		Usage: "Agent subcommands",
		Flags: flags,
		Subcommands: []*cli.Command{
			get_secret.BuildSubcommand(),
			put_secret.BuildSubcommand(),
			decrypt_secret.BuildSubcommand(),
			access_token.BuildSubcommand(),
			token_exchange.BuildSubcommand(),
		},
	}
}

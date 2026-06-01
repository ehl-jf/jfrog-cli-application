package system

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

import (
	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/systems"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

type pingCommand struct {
	systemService systems.SystemService
	serverDetails *coreConfig.ServerDetails
}

func (pc *pingCommand) Run() error {
	ctx, err := service.NewContext(*pc.serverDetails)
	if err != nil {
		return err
	}

	return pc.systemService.Ping(ctx)
}

func (pc *pingCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return pc.serverDetails, nil
}

func (pc *pingCommand) CommandName() string {
	return commands.Ping
}

func (pc *pingCommand) prepareAndRunCommand(ctx *components.Context) error {
	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	pc.serverDetails = serverDetails
	return commonCLiCommands.Exec(pc)
}

func GetPingCommand(appContext app.Context) components.Command {
	cmd := &pingCommand{systemService: appContext.GetSystemService()}
	return components.Command{
		Name:        commands.Ping,
		Description: "Ping AppTrust server.",
		AIDescription: `Send a health-check request to the AppTrust service on the configured JFrog Platform.

When to use:
- Verify that AppTrust is reachable and the configured credentials authenticate before running other apptrust commands.
- Quickly diagnose connectivity or auth issues in CI scripts.

Prerequisites:
- A configured server (jf c add or jf login) or per-command flags --url and either --user/--access-token or --server-id.

Common patterns:
  $ jf apptrust ping
  $ jf at p --server-id=my-server
  $ jf apptrust ping --url=https://my.jfrog.io --access-token=$JFROG_TOKEN

Gotchas:
- Returns a non-zero exit code if the server is unreachable or credentials are invalid; does not verify per-application permissions.

Related: jf c show, jf rt ping`,
		Category:  common.CategorySystem,
		Aliases:   []string{"p"},
		Arguments: []components.Argument{},
		Flags:     commands.GetCommandFlags(commands.Ping),
		Action:    cmd.prepareAndRunCommand,
	}
}

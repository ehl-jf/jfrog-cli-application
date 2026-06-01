package version

import (
	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/versions"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

type deleteAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
}

func (dv *deleteAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*dv.serverDetails)
	if err != nil {
		return err
	}

	return dv.versionService.DeleteAppVersion(ctx, dv.applicationKey, dv.version)
}

func (dv *deleteAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return dv.serverDetails, nil
}

func (dv *deleteAppVersionCommand) CommandName() string {
	return commands.VersionDelete
}

func (dv *deleteAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 2 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	dv.applicationKey = ctx.Arguments[0]
	dv.version = ctx.Arguments[1]

	var err error
	dv.serverDetails, err = utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	return commonCLiCommands.Exec(dv)
}

func GetDeleteAppVersionCommand(appContext app.Context) components.Command {
	cmd := &deleteAppVersionCommand{versionService: appContext.GetVersionService()}
	return components.Command{
		Name:        commands.VersionDelete,
		Description: "Delete application version.",
		AIDescription: `Permanently delete an application version from the AppTrust service.

When to use:
- Clean up obsolete or invalid versions (e.g., failed drafts, test versions).
- Free up names after an incorrect version was created.

Prerequisites:
- Configured server and delete permission on the application's project.
- The version should not be actively promoted to stages you cannot reach with delete permissions.

Common patterns:
  $ jf apptrust version-delete my-app 1.0.0
  $ jf at vd my-app 0.1.0-rc1 --server-id=my-server

Gotchas:
- Deletion is irreversible; consider version-rollback first if you only want to undo a promotion.
- Deleting a released version may be restricted by platform policy.

Related: jf apptrust version-rollback, jf apptrust version-create`,
		Category: common.CategoryVersion,
		Aliases:  []string{"vd"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The application key.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The name of the version to delete.",
				Optional:    false,
			},
		},
		Flags:  commands.GetCommandFlags(commands.VersionDelete),
		Action: cmd.prepareAndRunCommand,
	}
}

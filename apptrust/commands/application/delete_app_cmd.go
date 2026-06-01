package application

import (
	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"

	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/applications"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

type deleteAppCommand struct {
	serverDetails      *coreConfig.ServerDetails
	applicationService applications.ApplicationService
	applicationKey     string
}

func (dac *deleteAppCommand) Run() error {
	ctx, err := service.NewContext(*dac.serverDetails)
	if err != nil {
		return err
	}

	return dac.applicationService.DeleteApplication(ctx, dac.applicationKey)
}

func (dac *deleteAppCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return dac.serverDetails, nil
}

func (dac *deleteAppCommand) CommandName() string {
	return commands.AppDelete
}

func (dac *deleteAppCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 1 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	dac.applicationKey = ctx.Arguments[0]

	var err error
	dac.serverDetails, err = utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	return commonCLiCommands.Exec(dac)
}

func GetDeleteAppCommand(appContext app.Context) components.Command {
	cmd := &deleteAppCommand{
		applicationService: appContext.GetApplicationService(),
	}
	return components.Command{
		Name:        commands.AppDelete,
		Description: "Delete an application.",
		AIDescription: `Permanently delete an application from AppTrust, identified by its application key.

When to use:
- Remove an application that is no longer maintained.
- Clean up after creating an application with the wrong key.

Prerequisites:
- The application must exist.
- Configured server and delete permission on the application's project.
- Existing versions/bindings under the application may need to be removed first depending on platform policy.

Common patterns:
  $ jf apptrust app-delete my-app
  $ jf at ad my-app --server-id=my-server

Gotchas:
- Deletion is irreversible and may fail if the application still has active versions or package bindings.
- This does not delete artifacts in Artifactory; only the AppTrust application record is removed.

Related: jf apptrust app-create, jf apptrust version-delete, jf apptrust package-unbind`,
		Category: common.CategoryApplication,
		Aliases:  []string{"ad"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The key of the application to delete.",
				Optional:    false,
			},
		},
		Flags:  commands.GetCommandFlags(commands.AppDelete),
		Action: cmd.prepareAndRunCommand,
	}
}

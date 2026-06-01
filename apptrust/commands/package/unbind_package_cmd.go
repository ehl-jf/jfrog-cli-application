package packagecmds

import (
	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/packages"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

type unbindPackageCommand struct {
	packageService packages.PackageService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	packageType    string
	packageName    string
	packageVersion string
}

func (up *unbindPackageCommand) Run() error {
	ctx, err := service.NewContext(*up.serverDetails)
	if err != nil {
		return err
	}
	return up.packageService.UnbindPackage(ctx, up.applicationKey, up.packageType, up.packageName, up.packageVersion)
}

func (up *unbindPackageCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return up.serverDetails, nil
}

func (up *unbindPackageCommand) CommandName() string {
	return commands.PackageUnbind
}

func (up *unbindPackageCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 4 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	var err error
	up.serverDetails, err = utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	// Extract from arguments
	up.applicationKey = ctx.Arguments[0]
	up.packageType = ctx.Arguments[1]
	up.packageName = ctx.Arguments[2]
	up.packageVersion = ctx.Arguments[3]

	return commonCLiCommands.Exec(up)
}

func GetUnbindPackageCommand(appContext app.Context) components.Command {
	cmd := &unbindPackageCommand{packageService: appContext.GetPackageService()}
	return components.Command{
		Name:        commands.PackageUnbind,
		Description: "Unbind packages from an application.",
		AIDescription: `Remove a previously created binding between a specific package version and an application.

When to use:
- Revoke an association between a package version and an application (e.g., the package was bound by mistake or moved to a different application).
- Clean up bindings before deleting an application.

Prerequisites:
- The binding must currently exist.
- Configured server and unbind permission on the application's project.

Common patterns:
  $ jf apptrust package-unbind my-app docker my-image 1.0.0
  $ jf apptrust package-unbind my-app npm @scope/my-lib 2.3.1
  $ jf at pu my-app maven com.acme:widget 1.0.0 --server-id=my-server

Gotchas:
- All four positional arguments are required and must exactly match the existing binding.
- Unbinding does not delete the package from Artifactory; only the AppTrust association is removed.

Related: jf apptrust package-bind, jf apptrust app-delete`,
		Category: common.CategoryPackage,
		Aliases:  []string{"pu"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The key of the application to unbind the package from.",
			},
			{
				Name:        "package-type",
				Description: "Package type (e.g., npm, docker, maven, generic).",
			},
			{
				Name:        "package-name",
				Description: "Package name.",
			},
			{
				Name:        "package-version",
				Description: "Package version.",
			},
		},
		Flags:  commands.GetCommandFlags(commands.PackageUnbind),
		Action: cmd.prepareAndRunCommand,
	}
}

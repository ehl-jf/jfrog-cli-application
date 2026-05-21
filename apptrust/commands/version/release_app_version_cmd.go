package version

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

import (
	"os"

	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/versions"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

// orderedReleaseAppVersionKeys defines the display order for version-release table output.
var orderedReleaseAppVersionKeys = []string{
	"application_key",
	"version",
	"status",
	"current_stage",
}

type releaseAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
	requestPayload *model.ReleaseAppVersionRequest
	sync           bool
	responseBody   []byte
}

func (rv *releaseAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*rv.serverDetails)
	if err != nil {
		return err
	}

	rv.responseBody, err = rv.versionService.ReleaseAppVersion(ctx, rv.applicationKey, rv.version, rv.requestPayload, rv.sync)
	return err
}

func (rv *releaseAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return rv.serverDetails, nil
}

func (rv *releaseAppVersionCommand) CommandName() string {
	return commands.VersionRelease
}

func (rv *releaseAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 2 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	// Extract from arguments
	rv.applicationKey = ctx.Arguments[0]
	rv.version = ctx.Arguments[1]

	// Extract sync flag value
	rv.sync = ctx.GetBoolTFlagValue(commands.SyncFlag)

	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	rv.serverDetails = serverDetails
	rv.requestPayload, err = rv.buildRequestPayload(ctx)
	if errorutils.CheckError(err) != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(rv); err != nil {
		return err
	}

	return common.PrintResponse(rv.responseBody, outputFormat, os.Stdout, orderedReleaseAppVersionKeys)
}

func (rv *releaseAppVersionCommand) buildRequestPayload(ctx *components.Context) (*model.ReleaseAppVersionRequest, error) {
	promotionType, includedRepos, excludedRepos, err := BuildPromotionParams(ctx)
	if err != nil {
		return nil, err
	}

	artifactProps, err := ParseArtifactProps(ctx)
	if err != nil {
		return nil, err
	}

	overwriteStrategy, err := ParseOverwriteStrategy(ctx)
	if err != nil {
		return nil, err
	}

	return model.NewReleaseAppVersionRequest(
		promotionType,
		includedRepos,
		excludedRepos,
		artifactProps,
		overwriteStrategy,
	), nil
}

func GetReleaseAppVersionCommand(appContext app.Context) components.Command {
	cmd := &releaseAppVersionCommand{
		versionService: appContext.GetVersionService(),
	}
	return components.Command{
		Name:        commands.VersionRelease,
		Description: "Release application version.",
		AIDescription: `Release an application version, marking it as the final/released artifact set and optionally copying or moving artifacts according to the release configuration.

When to use:
- Finalize an application version after it has been promoted through earlier stages.
- Produce the released set of artifacts for downstream consumers.

Prerequisites:
- The application version must already exist and be eligible for release.
- Configured server and release permission on the application's project.

Common patterns:
  $ jf apptrust version-release my-app 1.0.0
  $ jf apptrust version-release my-app 1.0.0 --promotion-type=move
  $ jf apptrust version-release my-app 1.0.0 --include-repos="prod-local" --props="released=true"
  $ jf apptrust version-release my-app 1.0.0 --overwrite-strategy=fail

Gotchas:
- Release is a one-way transition; use version-rollback to undo promotions but a released version typically cannot be re-released.
- --promotion-type defaults to "copy".
- --sync defaults to true; pass --sync=false for asynchronous behavior.

Related: jf apptrust version-promote, jf apptrust version-rollback`,
		Category:    common.CategoryVersion,
		Aliases:     []string{"vr"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The application key.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version to release.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionRelease),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		DefaultFormat:    coreformat.Json,
		Action:           cmd.prepareAndRunCommand,
	}
}

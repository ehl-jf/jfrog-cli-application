package version

import (
	"os"

	"github.com/jfrog/jfrog-cli-application/apptrust/service/versions"

	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type createAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	requestPayload *model.CreateAppVersionRequest
	sync           bool
	dryRun         bool
	responseBody   []byte
}

func (cv *createAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*cv.serverDetails)
	if err != nil {
		return err
	}

	cv.responseBody, err = cv.versionService.CreateAppVersion(ctx, cv.requestPayload, cv.sync, cv.dryRun)
	return err
}

func (cv *createAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return cv.serverDetails, nil
}

func (cv *createAppVersionCommand) CommandName() string {
	return commands.VersionCreate
}

func (cv *createAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if err := validateCreateAppVersionContext(ctx); err != nil {
		return err
	}
	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	cv.serverDetails = serverDetails
	cv.sync = ctx.GetBoolTFlagValue(commands.SyncFlag)
	cv.requestPayload, err = cv.buildRequestPayload(ctx)
	if errorutils.CheckError(err) != nil {
		return err
	}
	cv.dryRun = ctx.GetBoolFlagValue(commands.DryRunFlag)

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(cv); err != nil {
		return err
	}

	return common.PrintResponse(cv.responseBody, outputFormat, os.Stdout, common.OrderedAppVersionKeys)
}

func (cv *createAppVersionCommand) buildRequestPayload(ctx *components.Context) (*model.CreateAppVersionRequest, error) {
	sources, filters, err := buildSourcesAndFiltersFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &model.CreateAppVersionRequest{
		ApplicationKey: ctx.Arguments[0],
		Version:        ctx.Arguments[1],
		Sources:        sources,
		Tag:            ctx.GetStringFlagValue(commands.TagFlag),
		Draft:          ctx.GetBoolFlagValue(commands.DraftFlag),
		Filters:        filters,
	}, nil
}

func validateCreateAppVersionContext(ctx *components.Context) error {
	if err := validateNoSpecAndFlagsTogether(ctx); err != nil {
		return err
	}
	if len(ctx.Arguments) != 2 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}
	return validateAtLeastOneSourceFlag(ctx)
}

func GetCreateAppVersionCommand(appContext app.Context) components.Command {
	cmd := &createAppVersionCommand{versionService: appContext.GetVersionService()}
	return components.Command{
		Name:        commands.VersionCreate,
		Description: "Create application version.",
		AIDescription: `Create a new application version from one or more sources (builds, release bundles, other application versions, packages, or artifacts).

When to use:
- Assemble a new releasable application version from artifacts produced upstream in CI.
- Create a draft version that can later be finalized via version-update-sources.

Prerequisites:
- The application (--app-key) must already exist (see app-create).
- Configured server with AppTrust enabled and write permission on the application's project.
- At least one source must be provided either via --spec or one of the --source-type-* flags.

Common patterns:
  $ jf apptrust version-create my-app 1.0.0 --source-type-builds="name=my-build, id=42"
  $ jf apptrust version-create my-app 1.0.0 --source-type-packages="type=docker, name=my-image, version=1.0.0, repo-key=docker-local"
  $ jf apptrust version-create my-app 1.0.0 --spec=version-spec.json --spec-vars="BUILD=42"
  $ jf apptrust version-create my-app 1.0.0 --source-type-builds="name=b, id=1" --draft --dry-run

Gotchas:
- The version argument must be a valid SemVer string.
- --spec cannot be combined with --source-type-* flags; choose one approach.
- --sync defaults to true; pass --sync=false to return as soon as the request is accepted.

Related: jf apptrust version-update-sources, jf apptrust version-promote, jf apptrust version-release`,
		Category:    common.CategoryVersion,
		Aliases:     []string{"vc"},
		Arguments: []components.Argument{
			{
				Name:        "app-key",
				Description: "The application key of the application for which the version is being created.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version number (in SemVer format) for the new application version.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionCreate),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		DefaultFormat:    coreformat.Json,
		Action:           cmd.prepareAndRunCommand,
	}
}

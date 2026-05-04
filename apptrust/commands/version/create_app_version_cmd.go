package version

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

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
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
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

	return printCreateAppVersionResponse(cv.responseBody, outputFormat, os.Stdout)
}

// orderedCreateAppVersionKeys defines the display order for version-create table output.
var orderedCreateAppVersionKeys = []string{
	"application_key",
	"version",
	"status",
	"current_stage",
	"tag",
}

// printCreateAppVersionResponse formats and prints the create-app-version response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printCreateAppVersionResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printCreateAppVersionTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printCreateAppVersionTable renders the create-app-version response as a FIELD/VALUE table.
func printCreateAppVersionTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse version response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedCreateAppVersionKeys {
		val, ok := fields[key]
		if !ok || val == nil {
			continue
		}
		strVal := fmt.Sprintf("%v", val)
		if strVal == "" {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\n", key, strVal)
	}
	return tw.Flush()
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
		Action:           cmd.prepareAndRunCommand,
	}
}

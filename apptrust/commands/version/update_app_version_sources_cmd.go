package version

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

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
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type updateAppVersionSourcesCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
	requestPayload *model.UpdateVersionSourcesRequest
	sync           bool
	dryRun         bool
	failFast       bool
	responseBody   []byte
}

func (cmd *updateAppVersionSourcesCommand) Run() error {
	ctx, err := service.NewContext(*cmd.serverDetails)
	if err != nil {
		log.Error("Failed to create service context:", err)
		return err
	}

	cmd.responseBody, err = cmd.versionService.UpdateAppVersionSources(ctx, cmd.applicationKey, cmd.version, cmd.requestPayload, cmd.sync, cmd.dryRun, cmd.failFast)
	if err != nil {
		log.Error("Failed to update application version sources:", err)
		return err
	}

	return nil
}

func (cmd *updateAppVersionSourcesCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return cmd.serverDetails, nil
}

func (cmd *updateAppVersionSourcesCommand) CommandName() string {
	return commands.VersionUpdateSources
}

func (cmd *updateAppVersionSourcesCommand) prepareAndRunCommand(ctx *components.Context) error {
	if err := validateUpdateSourcesContext(ctx); err != nil {
		return err
	}

	if err := cmd.parseFlagsAndSetFields(ctx); err != nil {
		return err
	}

	var err error
	cmd.requestPayload, err = cmd.buildRequestPayload(ctx)
	if errorutils.CheckError(err) != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(cmd); err != nil {
		return err
	}

	return printUpdateAppVersionSourcesResponse(cmd.responseBody, outputFormat, os.Stdout)
}

func validateUpdateSourcesContext(ctx *components.Context) error {
	if len(ctx.Arguments) != 2 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}
	if err := validateNoSpecAndFlagsTogether(ctx); err != nil {
		return err
	}
	return validateAtLeastOneSourceFlag(ctx)
}

// parseFlagsAndSetFields parses CLI flags and sets struct fields accordingly.
func (cmd *updateAppVersionSourcesCommand) parseFlagsAndSetFields(ctx *components.Context) error {
	cmd.applicationKey = ctx.Arguments[0]
	cmd.version = ctx.Arguments[1]

	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	cmd.serverDetails = serverDetails

	cmd.sync = ctx.GetBoolTFlagValue(commands.SyncFlag)
	cmd.dryRun = ctx.GetBoolFlagValue(commands.DryRunFlag)
	cmd.failFast = ctx.GetBoolTFlagValue(commands.FailFastFlag)

	return nil
}

func (cmd *updateAppVersionSourcesCommand) buildRequestPayload(ctx *components.Context) (*model.UpdateVersionSourcesRequest, error) {
	sources, filters, err := buildSourcesAndFiltersFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &model.UpdateVersionSourcesRequest{
		AddSources: sources,
		Filters:    filters,
	}, nil
}

// orderedUpdateAppVersionSourcesKeys defines the display order for version-update-sources table output.
var orderedUpdateAppVersionSourcesKeys = []string{
	"application_key",
	"version",
	"status",
	"current_stage",
	"tag",
}

// printUpdateAppVersionSourcesResponse formats and prints the update-app-version-sources response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printUpdateAppVersionSourcesResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printUpdateAppVersionSourcesTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printUpdateAppVersionSourcesTable renders the update-app-version-sources response as a FIELD/VALUE table.
func printUpdateAppVersionSourcesTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse update sources response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedUpdateAppVersionSourcesKeys {
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

func GetUpdateAppVersionSourcesCommand(appContext app.Context) components.Command {
	cmd := &updateAppVersionSourcesCommand{versionService: appContext.GetVersionService()}
	return components.Command{
		Name:        commands.VersionUpdateSources,
		Description: "Updates the sources for a draft application version.",
		Category:    common.CategoryVersion,
		Aliases:     []string{"vus"},
		Arguments: []components.Argument{
			{
				Name:        "app-key",
				Description: "The application key of the application for which the version sources are being updated.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version number (in SemVer format) for the application version to update sources.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionUpdateSources),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Action:           cmd.prepareAndRunCommand,
	}
}

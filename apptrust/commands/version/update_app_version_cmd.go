package version

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

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

type updateAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
	requestPayload *model.UpdateAppVersionRequest
	responseBody   []byte
}

func (uv *updateAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*uv.serverDetails)
	if err != nil {
		log.Error("Failed to create service context:", err)
		return err
	}

	uv.responseBody, err = uv.versionService.UpdateAppVersion(ctx, uv.applicationKey, uv.version, uv.requestPayload)
	if err != nil {
		log.Error("Failed to update application version:", err)
		return err
	}

	return nil
}

func (uv *updateAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return uv.serverDetails, nil
}

func (uv *updateAppVersionCommand) CommandName() string {
	return commands.VersionUpdate
}

func (uv *updateAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 2 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	if err := uv.parseFlagsAndSetFields(ctx); err != nil {
		return err
	}

	var err error
	uv.requestPayload, err = uv.buildRequestPayload(ctx)
	if errorutils.CheckError(err) != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(uv); err != nil {
		return err
	}

	return printUpdateAppVersionResponse(uv.responseBody, outputFormat, os.Stdout)
}

// parseFlagsAndSetFields parses CLI flags and sets struct fields accordingly.
func (uv *updateAppVersionCommand) parseFlagsAndSetFields(ctx *components.Context) error {
	uv.applicationKey = ctx.Arguments[0]
	uv.version = ctx.Arguments[1]

	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	uv.serverDetails = serverDetails
	return nil
}

func (uv *updateAppVersionCommand) buildRequestPayload(ctx *components.Context) (*model.UpdateAppVersionRequest, error) {
	request := &model.UpdateAppVersionRequest{}

	if ctx.IsFlagSet(commands.TagFlag) {
		request.Tag = ctx.GetStringFlagValue(commands.TagFlag)
	}

	// Handle properties - use spec format: key=value1[,value2,...]
	if ctx.IsFlagSet(commands.PropertiesFlag) {
		properties, err := utils.ParseListPropertiesFlag(ctx.GetStringFlagValue(commands.PropertiesFlag))
		if err != nil {
			return nil, err
		}
		request.Properties = properties
	}

	// Handle delete properties
	if ctx.IsFlagSet(commands.DeletePropertiesFlag) {
		deleteProps := utils.ParseSliceFlag(ctx.GetStringFlagValue(commands.DeletePropertiesFlag))
		request.DeleteProperties = deleteProps
	}

	return request, nil
}

// orderedUpdateAppVersionKeys defines the display order for version-update table output.
var orderedUpdateAppVersionKeys = []string{
	"application_key",
	"version",
	"status",
	"current_stage",
	"tag",
}

// printUpdateAppVersionResponse formats and prints the update-app-version response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printUpdateAppVersionResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printUpdateAppVersionTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printUpdateAppVersionTable renders the update-app-version response as a FIELD/VALUE table.
func printUpdateAppVersionTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse update response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedUpdateAppVersionKeys {
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

func GetUpdateAppVersionCommand(appContext app.Context) components.Command {
	cmd := &updateAppVersionCommand{versionService: appContext.GetVersionService()}
	return components.Command{
		Name:        commands.VersionUpdate,
		Description: "Updates the user-defined annotations (tag and custom key-value properties) for a specified application version.",
		Category:    common.CategoryVersion,
		Aliases:     []string{"vu"},
		Arguments: []components.Argument{
			{
				Name:        "app-key",
				Description: "The application key of the application for which the version is being updated.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version number (in SemVer format) for the application version to update.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionUpdate),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Action:           cmd.prepareAndRunCommand,
	}
}

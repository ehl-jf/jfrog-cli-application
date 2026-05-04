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
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type rollbackAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
	requestPayload *model.RollbackAppVersionRequest
	fromStage      string
	sync           bool
	responseBody   []byte
}

func (rv *rollbackAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*rv.serverDetails)
	if err != nil {
		return err
	}

	rv.responseBody, err = rv.versionService.RollbackAppVersion(ctx, rv.applicationKey, rv.version, rv.requestPayload, rv.sync)
	return err
}

func (rv *rollbackAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return rv.serverDetails, nil
}

func (rv *rollbackAppVersionCommand) CommandName() string {
	return commands.VersionRollback
}

func (rv *rollbackAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 3 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	rv.applicationKey = ctx.Arguments[0]
	rv.version = ctx.Arguments[1]
	rv.fromStage = ctx.Arguments[2]

	rv.sync = ctx.GetBoolTFlagValue(commands.SyncFlag)

	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	rv.serverDetails = serverDetails
	rv.requestPayload = model.NewRollbackAppVersionRequest(rv.fromStage)

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(rv); err != nil {
		return err
	}

	return printRollbackAppVersionResponse(rv.responseBody, outputFormat, os.Stdout)
}

// orderedRollbackAppVersionKeys defines the display order for version-rollback table output.
var orderedRollbackAppVersionKeys = []string{
	"application_key",
	"version",
	"project_key",
	"rollback_from_stage",
	"rollback_to_stage",
}

// printRollbackAppVersionResponse formats and prints the rollback-app-version response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printRollbackAppVersionResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printRollbackAppVersionTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printRollbackAppVersionTable renders the rollback-app-version response as a FIELD/VALUE table.
func printRollbackAppVersionTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse rollback response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedRollbackAppVersionKeys {
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

func GetRollbackAppVersionCommand(appContext app.Context) components.Command {
	cmd := &rollbackAppVersionCommand{
		versionService: appContext.GetVersionService(),
	}
	return components.Command{
		Name:        commands.VersionRollback,
		Description: "Roll back application version promotion.",
		Category:    common.CategoryVersion,
		Aliases:     []string{"vrb"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The application key.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version to roll back.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionRollback),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Action:           cmd.prepareAndRunCommand,
	}
}

package application

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"

	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/applications"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type updateAppCommand struct {
	serverDetails      *coreConfig.ServerDetails
	applicationService applications.ApplicationService
	requestBody        *model.AppDescriptor
	responseBody       []byte
}

func (uac *updateAppCommand) Run() error {
	ctx, err := service.NewContext(*uac.serverDetails)
	if err != nil {
		return err
	}

	uac.responseBody, err = uac.applicationService.UpdateApplication(ctx, uac.requestBody)
	return err
}

func (uac *updateAppCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return uac.serverDetails, nil
}

func (uac *updateAppCommand) CommandName() string {
	return commands.AppUpdate
}

func (uac *updateAppCommand) buildRequestPayload(ctx *components.Context) (*model.AppDescriptor, error) {
	applicationKey := ctx.Arguments[0]

	descriptor := &model.AppDescriptor{
		ApplicationKey: applicationKey,
	}

	err := populateApplicationFromFlags(ctx, descriptor)
	if err != nil {
		return nil, err
	}

	return descriptor, nil
}

func (uac *updateAppCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 1 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	var err error
	uac.requestBody, err = uac.buildRequestPayload(ctx)
	if err != nil {
		return err
	}

	uac.serverDetails, err = utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(uac); err != nil {
		return err
	}

	return printUpdateAppResponse(uac.responseBody, outputFormat, os.Stdout)
}

// orderedUpdateAppKeys defines the display order for app-update table output.
var orderedUpdateAppKeys = []string{
	"application_key",
	"application_name",
	"project_key",
	"description",
	"criticality",
	"maturity_level",
}

// printUpdateAppResponse formats and prints the update-application response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printUpdateAppResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printUpdateAppTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printUpdateAppTable renders the update-application response as a FIELD/VALUE table.
func printUpdateAppTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse application response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedUpdateAppKeys {
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

func GetUpdateAppCommand(appContext app.Context) components.Command {
	cmd := &updateAppCommand{
		applicationService: appContext.GetApplicationService(),
	}
	return components.Command{
		Name:             commands.AppUpdate,
		Description:      "Update an existing application",
		Category:         common.CategoryApplication,
		Aliases:          []string{"au"},
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The key of the application to update",
				Optional:    false,
			},
		},
		Flags:  commands.GetCommandFlags(commands.AppUpdate),
		Action: cmd.prepareAndRunCommand,
	}
}

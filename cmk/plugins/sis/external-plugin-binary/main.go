package main

import (
	"fmt"
	"os"

	"github.com/openkcm/common-sdk/pkg/utils"
	pluginoption "github.com/openkcm/plugin-sdk/api/plugin-option"
	"github.com/openkcm/plugin-sdk/pkg/plugin"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	"github.com/openkcm/sis-plugin/external-plugin-binary/plugin/sis"
)

var BuildInfo = "{}"

func main() {
	value, err := utils.ExtractFromComplexValue(BuildInfo)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to extract BuildInfo")
		os.Exit(1)
	}
	p := sis.NewPlugin(value)

	err = plugin.ServeOptions(
		pluginoption.WithPluginServer(systeminformationv1.SystemInformationServicePluginServer(p)),
		pluginoption.WithServiceServer(configv1.ConfigServiceServer(p)),
		pluginoption.EnableInputValidation(),
		pluginoption.EnableOutputValidation(),
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to set up SIS plugin")
		os.Exit(1)
	}
}

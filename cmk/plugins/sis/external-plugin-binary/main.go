package main

import (
	"log/slog"

	"github.com/openkcm/common-sdk/pkg/utils"
	"github.com/openkcm/plugin-sdk/pkg/plugin"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	"github.com/openkcm/sis-plugin/external-plugin-binary/plugin/sis"
)

var BuildInfo = "{}"

func main() {
	value, err := utils.ExtractFromComplexValue(BuildInfo)
	if err != nil {
		slog.Warn("Failed to extract BuildInfo")
	}
	p := sis.NewPlugin(value)

	plugin.Serve(
		systeminformationv1.SystemInformationServicePluginServer(p),
		configv1.ConfigServiceServer(p),
	)
}

package serviceapi

import (
	"io"

	"github.com/openkcm/sis-builtin-plugin/internal/service/api/systeminformation"
)

// Registry defines the central contract for accessing and managing system services.
// It embeds io.Closer to facilitate the graceful shutdown of all active subsystems.
type Registry interface {
	io.Closer

	// SystemInformation returns the active SystemInformation service.
	// The boolean returns false if the service is not configured or available.
	SystemInformation() (systeminformation.SystemInformation, error)
}

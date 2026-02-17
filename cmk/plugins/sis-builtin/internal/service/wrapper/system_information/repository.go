package system_information

import (
	"errors"

	"github.com/openkcm/sis-builtin-plugin/internal/service/api/systeminformation"
)

var (
	ErrSystemInformationNotDefined = errors.New(`system information not defined`)
)

type Repository struct {
	Instance systeminformation.SystemInformation
}

func (repo *Repository) SystemInformation() (systeminformation.SystemInformation, error) {
	if repo.Instance == nil {
		return nil, ErrSystemInformationNotDefined
	}
	return repo.Instance, nil
}

func (repo *Repository) SetSystemInformation(instance systeminformation.SystemInformation) {
	repo.Instance = instance
}

func (repo *Repository) Clear() {
	repo.Instance = nil
}

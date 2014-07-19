package cloudinit

import (
	"fmt"

	"github.com/juju/juju/version"
)

type Renderer interface {
	// MkDir
	MkDir(path string) []string
	WriteFile(ilename string, contents string, permission int) []string
	Render(conf *Config) ([]byte, error)
	FromSlash(path string) string
	PathJoin(path ...string) string
}

func NewRenderer(series string) (Renderer, error) {
	operatingSystem, err := version.GetOSFromSeries(series)
	if err != nil {
		return nil, err
	}

	switch operatingSystem {
	case version.Windows:
		return &WindowsRenderer{}, nil
	case version.Ubuntu:
		return &UbuntuRenderer{}, nil
	default:
		return nil, fmt.Errorf("No renderer could be found for %s", series)
	}
}

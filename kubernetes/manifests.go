package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type manifest struct {
	source        string
	containerPath string
	bind          bool
}

func (c *Cluster) prepareManifests() error {
	for index := range c.manifests {
		if isURL(c.manifests[index].source) {
			c.manifests[index].containerPath = c.manifests[index].source
			continue
		}

		path, err := filepath.Abs(c.manifests[index].source)
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err != nil {
			return err
		}

		c.manifests[index].source = path
		c.manifests[index].containerPath = fmt.Sprintf("/scaffold-manifest-%d-%s", index+1, filepath.Base(path))
		c.manifests[index].bind = true
	}

	return nil
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

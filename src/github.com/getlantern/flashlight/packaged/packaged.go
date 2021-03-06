// Package packaged provided access to configuration embedded directly in Lantern installation
// packages. On OSX, that means data embedded in the Lantern.app app bundle in
// Lantern.app/Contents/Resources/.lantern.yaml, while on Windows that means data embedded
// in AppData/Roaming/Lantern/.lantern.yaml. This allows customization embedded in the
// installer outside of the auto-updated binary that should only be used under special
// circumstances.
package packaged

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/getlantern/appdir"
	"github.com/getlantern/golog"
	"github.com/getlantern/yaml"
)

var (
	log  = golog.LoggerFor("flashlight.packaged")
	name = ".packaged-lantern.yaml"

	// This is the local copy of our embedded configuration file. This is necessary
	// to ensure we remember the embedded configuration across auto-updated
	// binaries. We write to the local file system instead of to the package
	// itself (app bundle on OSX, install directory on Windows) because
	// we're not always sure we can write to that directory.
	local = appdir.General("Lantern") + "/" + name
)

// PackagedSettings provided access to configuration embedded in the package.
type PackagedSettings struct {
	StartupUrl string
}

// ReadSettings reads packaged settings from pre-determined paths
// on the various OSes.
func ReadSettings() (string, *PackagedSettings, error) {
	yamlPath, err := packagedSettingsPath()
	if err != nil {
		return "", &PackagedSettings{}, err
	}

	path, ps, er := readSettingsFromFile(yamlPath)
	if er != nil {
		return readSettingsFromFile(local)
	}
	return path, ps, nil
}

// ReadSettingsFromFile reads PackagedSettings from the yaml file at the specified
// path.
func readSettingsFromFile(yamlPath string) (string, *PackagedSettings, error) {
	log.Debugf("Opening file at: %v", yamlPath)
	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		// This will happen whenever there's no packaged settings, which is often
		log.Debugf("Error reading file %v", err)
		return "", &PackagedSettings{}, err
	}

	trimmed := strings.TrimSpace(string(data))

	log.Debugf("Read bytes: %v", trimmed)

	if trimmed == "" {
		log.Debugf("Ignoring empty string")
		return "", &PackagedSettings{}, errors.New("Empty string")
	}
	var s PackagedSettings
	err = yaml.Unmarshal([]byte(trimmed), &s)

	if err != nil {
		log.Errorf("Could not read yaml: %v", err)
		return "", &PackagedSettings{}, err
	}
	return yamlPath, &s, nil
}

func packagedSettingsPath() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Errorf("Could not get current directory %v", err)
		return "", err
	}
	log.Debugf("Opening externalUrl from: %v", dir)
	var yamldir string
	if runtime.GOOS == "windows" {
		yamldir = dir
	} else if runtime.GOOS == "darwin" {
		// Code signing doesn't like this file in the current directory
		// for whatever reason, so we grab it from the Resources/en.lproj
		// directory in the app bundle. See:
		// https://developer.apple.com/library/mac/technotes/tn2206/_index.html#//apple_ref/doc/uid/DTS40007919-CH1-TNTAG402
		yamldir = dir + "/../Resources/en.lproj"
		if _, err := ioutil.ReadDir(yamldir); err != nil {
			// This likely means the user originally installed with an older version that didn't include en.lproj
			// in the app bundle, so just look in the old location in Resources.
			yamldir = dir + "/../Resources"
		}
	} else if runtime.GOOS == "linux" {
		yamldir = dir
	}
	yamlPath := yamldir + "/" + name
	return yamlPath, nil
}

func writeToDisk(ps *PackagedSettings) (string, error) {
	data, err := yaml.Marshal(ps)
	if err != nil {
		log.Errorf("Could not write to disk: %v", err)
		return "", err
	}
	err = ioutil.WriteFile(local, data, 0644)
	if err != nil {
		log.Errorf("Could not write to disk: %v", err)
		return "", err
	}
	return local, nil
}

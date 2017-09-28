package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	//	"time"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/config/types"
	// "github.com/coreos/ignition/config/validate/report"
	// "github.com/davecgh/go-spew/spew"
)

var (
	// FLAGS
	ignitionFilePath = flag.String("ignition", "", "Path to the new ignition config")
	silent           = flag.Bool("silent", false, "Don't report changes just output files")
	outPath          = flag.String("out", "./", "Path to output files to")
	helpFlag         = flag.Bool("help", false, "Usage")
)

type UnitStatus struct {
	UnitName string `json:"unitname"`
	Enable   bool   `json:"enable"`
}

func main() {
	flag.Parse()

	if *helpFlag {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *ignitionFilePath == "" {
		fmt.Println("Error: Requires both --old and --new flags to be set.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	normalizePath(outPath)

	if !*silent {
		fmt.Printf("Ignition config: %s\n", *ignitionFilePath)
	}
	newc, err := loadConfig(ignitionFilePath)
	if err != nil {
		fmt.Printf("Unable to load/parse: %s\n", err)
		os.Exit(1)
	}

	err = processFiles(newc, outPath)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	err = processSystemd(newc, outPath)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

}

func normalizePath(path *string) {
	last := (*path)[len(*path)-1:]
	if last == "/" {
		return
	}
	*path += "/"
}

func loadConfig(filepath *string) (types.Config, error) {
	data, err := ioutil.ReadFile(*filepath)
	if err != nil {
		return types.Config{}, err
	}
	c, _, err := config.Parse(data)
	if err != nil {
		return types.Config{}, err
	}

	return c, nil
}

func processFiles(config types.Config, outputBasePath *string) error {
	err := os.MkdirAll(*outputBasePath+"files", 0755)
	if err != nil {
		return err
	}

	for _, v := range config.Storage.Files {
		localDir := *outputBasePath + "files" + filepath.Dir(v.Path)
		localPath := *outputBasePath + "files" + v.Path
		err := os.MkdirAll(localDir, 0755)
		if err != nil {
			return err
		}
		content, err := url.QueryUnescape(v.Contents.Source[6:])
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(localPath, []byte(content), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func processSystemd(config types.Config, outputBasePath *string) error {
	basepath := *outputBasePath + "systemd/system/"
	err := os.MkdirAll(basepath, 0755)
	if err != nil {
		return err
	}

	var units []UnitStatus
	units = make([]UnitStatus, 0)

	for _, v := range config.Systemd.Units {
		name := v.Name
		unit := UnitStatus{name, v.Enable}
		if v.Contents != "" {
			units = append(units, unit)
			err = ioutil.WriteFile(basepath+name, []byte(v.Contents), 0644)
			if err != nil {
				return err
			}
		}
		if len(v.Dropins) > 0 {
			dropinPath := basepath + "/" + name + ".d"
			os.MkdirAll(dropinPath, 0755)

			for _, d := range v.Dropins {
				err = ioutil.WriteFile(dropinPath+"/"+d.Name, []byte(d.Contents), 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	c, err := json.Marshal(units)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(*outputBasePath+"/units.json", c, 0644)
	if err != nil {
		return err
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/src-d/proteus"
	"github.com/src-d/proteus/report"
	"gopkg.in/urfave/cli.v1"
)

var (
	packages cli.StringSlice
	path     string
	verbose  bool
)

func main() {
	app := cli.NewApp()
	app.Description = "Generate .proto files from your Go packages."
	app.Version = "0.9.0"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "pkg, p",
			Usage: "Generate a .proto file for `PACKAGE`. You can use this flag multiple times to specify more than one package.",
			Value: &packages,
		},
		cli.StringFlag{
			Name:        "folder, f",
			Usage:       "All generated .proto files will be written to `FOLDER`.",
			Destination: &path,
		},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Print all warnings and info messages.",
			Destination: &verbose,
		},
	}

	app.Action = action
	app.Run(os.Args)
}

func action(c *cli.Context) error {
	if path == "" {
		return errors.New("destination path cannot be empty")
	}

	if err := checkFolder(path); err != nil {
		return err
	}

	if len(packages) == 0 {
		return errors.New("no package provided, there is nothing to generate")
	}

	if !verbose {
		report.Silent()
	}

	return proteus.GenerateProtos(proteus.Options{
		BasePath: path,
		Packages: packages,
	})
}

func checkFolder(p string) error {
	fi, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		return errors.New("folder does not exist, please create it first")
	case err != nil:
		return err
	case !fi.IsDir():
		return fmt.Errorf("folder is not directory: %s", p)
	}
	return nil
}

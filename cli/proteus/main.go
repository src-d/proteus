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

	baseFlags := []cli.Flag{
		cli.StringSliceFlag{
			Name:  "pkg, p",
			Usage: "Use `PACKAGE` as input for the generation. You can use this flag multiple times to specify more than one package.",
			Value: &packages,
		},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Print all warnings and info messages.",
			Destination: &verbose,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "proto",
			Usage:  "Generates .proto files from Go packages",
			Action: initCmd(genProtos),
			Flags: append(baseFlags, cli.StringFlag{
				Name:        "folder, f",
				Usage:       "All generated .proto files will be written to `FOLDER`.",
				Destination: &path,
			}),
		},
		{
			Name:   "rpc",
			Usage:  "Generates gRPC server implementation",
			Action: initCmd(genRPCServer),
			Flags:  baseFlags,
		},
	}

	app.Run(os.Args)
}

type action func(c *cli.Context) error

func initCmd(next action) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if len(packages) == 0 {
			return errors.New("no package provided, there is nothing to generate")
		}

		if !verbose {
			report.Silent()
		}

		return next(c)
	}
}

func genProtos(c *cli.Context) error {
	if path == "" {
		return errors.New("destination path cannot be empty")
	}

	if err := checkFolder(path); err != nil {
		return err
	}

	return proteus.GenerateProtos(proteus.Options{
		BasePath: path,
		Packages: packages,
	})
}

func genRPCServer(c *cli.Context) error {
	return proteus.GenerateRPCServer(packages)
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

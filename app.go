package main

import "github.com/urfave/cli/v2"

var app = &cli.App{
	Name:                 "metric-collector",
	EnableBashCompletion: true,
	Commands: []*cli.Command{
		CmdRun,
	},
}

package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/DATA-DOG/godog/colors"
	"github.com/alileza/potato/agent"
	"github.com/alileza/potato/server"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// AppHelpTemplate is the text template for the Default help topic.
// cli.go uses text/template to render templates. You can
// render custom help text by setting this variable.
const AppHelpTemplate = `Usage: {{if .UsageText}}{{.UsageText}}{{else}}potato {{if .VisibleFlags}}[options]{{end}}{{if .ArgsUsage}}{{.ArgsUsage}}{{else}} <agent|server>{{end}}{{end}}
Options:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}
`

func main() {
	var config struct {
		ID               string
		LogLevel         string
		ListenAddress    string
		AdvertiseAddress string
	}
	cli.AppHelpTemplate = AppHelpTemplate

	log := log.New(os.Stdout, "", 0)

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "env.file, e",
			Usage: "Environment variable file path",
		},
		cli.StringFlag{
			Name:        "id",
			Usage:       "Node identifier",
			EnvVar:      "ID",
			Destination: &config.ID,
		},
		cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level",
			Value:       "info",
			EnvVar:      "LOG_LEVEL",
			Destination: &config.LogLevel,
		},
		cli.StringFlag{
			Name:        "listen-address",
			Usage:       "Port to wait incoming request from client",
			Value:       "0.0.0.0:9000",
			EnvVar:      "LISTEN_ADDRESS",
			Destination: &config.ListenAddress,
		},
		cli.StringFlag{
			Name:        "advertise-address",
			Usage:       "Port to advertise metrics over http",
			Value:       "0.0.0.0:9000",
			EnvVar:      "ADVERTISE_ADDRESS",
			Destination: &config.AdvertiseAddress,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		if envFile := ctx.String("env.file"); envFile != "" {
			return godotenv.Load(envFile)
		}

		if config.ID == "" {
			config.ID, _ = os.Hostname()
		}

		return nil
	}

	app.Action = func(ctx *cli.Context) (err error) {
		l := logrus.New()
		l.SetLevel(logrus.InfoLevel)

		ctxx := context.Background()
		switch ctx.Args().First() {
		case "server":
			err = server.NewServer(l, config.ListenAddress).Serve(ctxx)
		case "agent":
			err = agent.NewAgent(l, config.ID, config.ListenAddress, config.AdvertiseAddress).Start(ctxx)
		default:
			return errors.New("This command takes one argument: <agent|server>\nFor additional help try 'potato -help'")
		}
		return
	}

	if err := app.Run(os.Args); err != nil {
		log.Printf("%v", colors.Bold(colors.Red)(err))
		os.Exit(1)
	}
}

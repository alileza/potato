package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/DATA-DOG/godog/colors"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/alileza/potato/agent"
	"github.com/alileza/potato/server"
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
		MigrationPath    string
		DatabaseDSN      string
		SkipMigration    bool
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
			Value:       "0.0.0.0:9001",
			EnvVar:      "ADVERTISE_ADDRESS",
			Destination: &config.AdvertiseAddress,
		},
		cli.StringFlag{
			Name:        "migration-path",
			Usage:       "Database migration path for updating servers",
			Value:       "./migrations",
			EnvVar:      "MIGRATION_PATH",
			Destination: &config.MigrationPath,
		},
		cli.StringFlag{
			Name:        "database-dsn",
			Usage:       "Database data source name",
			Value:       "postgres://potato:potato@localhost:5432/potato?sslmode=disable",
			EnvVar:      "DATABASE_DSN",
			Destination: &config.DatabaseDSN,
		},
		cli.BoolFlag{
			Name:        "skip-migration",
			Usage:       "Skip database migration",
			EnvVar:      "SKIP_MIGRATION",
			Destination: &config.SkipMigration,
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

		dockerClient, err := client.NewEnvClient()
		if err != nil {
			return err
		}

		ctxx := context.Background()
		switch ctx.Args().First() {
		case "server":
			if !config.SkipMigration {
				err = migrateUp(config.MigrationPath, config.DatabaseDSN)
			}
			if err == nil {
				err = server.NewServer(l, config.ListenAddress, config.DatabaseDSN).Serve(ctxx)
			}
		case "agent":
			err = agent.NewAgent(l, dockerClient, config.ID, config.ListenAddress, config.AdvertiseAddress).Start(ctxx)
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

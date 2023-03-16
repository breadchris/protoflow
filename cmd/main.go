package main

import (
	"github.com/breadchris/protoflow/pkg"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	log.Logger = log.With().Caller().Logger()
	app := &cli.App{
		Name:        "protoflow",
		Description: "Coding as easy as playing with legos.",
		Flags:       []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name: "run",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "function",
					},
				},
				Action: func(ctx *cli.Context) error {
					file := ctx.Args().First()
					function := ctx.String("function")

					output, err := pkg.CallFunction(file, function, "Run")
					if err != nil {
						log.Error().Err(err).Msg("failed to run function")
						return err
					}
					println(string(output))
					return nil
				},
			},
			{
				Name: "serve",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name: "port",
					},
				},
				Action: func(ctx *cli.Context) error {
					port := ctx.Int("port")
					if port != 0 {
						log.Info().Int("port", port).Msg("running on port")
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("failed to run app")
	}
}

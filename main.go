package main

import (
	"os"
	"time"

	"github.com/le0pard/postal_server/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func main() {
	cmd.Execute()
}

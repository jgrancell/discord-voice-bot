package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger() {

	environment := environmentEnvironmentVariable
	if environment != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	log.Logger = log.With().Caller().Logger()

	zerolog.SetGlobalLevel(logLevel)
	log.Info().Msgf("setting log level to %s", logLevel.String())

}

func BotLog(i *discordgo.InteractionCreate, event *zerolog.Event) *zerolog.Event {
	event = event.
		Str("guild", i.GuildID).
		Str("channel", i.ChannelID).
		Str("user", i.Member.User.Username)
	return event
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	botCommand         string = "voice"
	channelCreatorName string = "+ Create Channel"

	discordTokenEnvironmentVariable      string = "BOT_DISCORD_TOKEN"
	logLevelEnvironmentVariable          string = "BOT_LOG_LEVEL"
	environmentEnvironmentVariable       string = "BOT_ENVIRONMENT"
	guildEnvironmentVariable             string = "BOT_GUILD_ID"
	categoryEnabledEnvironmentVariable   string = "BOT_CATEGORY_ENABLED"
	deletionThresholdEnvironmentVariable string = "BOT_DELETION_THRESHOLD"

	channelSuffix string = ""
)

var (
	// Our Default Configuration
	logLevel          zerolog.Level = 1
	categoryEnabled   []string      = []string{}
	deletionThreshold int           = 3
	enabledGuilds     []string      = []string{}

	admins []string = []string{}
)

func main() {
	// Environment Variable Parsing
	guildIds := os.Getenv(guildEnvironmentVariable)
	if guildIds != "" {
		for _, id := range strings.Split(guildIds, ",") {
			enabledGuilds = append(enabledGuilds, id)
		}
	}

	levelEnv := os.Getenv(logLevelEnvironmentVariable)
	if levelEnv != "" {
		if level, err := zerolog.ParseLevel(levelEnv); err != nil && level != zerolog.NoLevel {
			logLevel = level
		}
	}

	timeout := os.Getenv(deletionThresholdEnvironmentVariable)
	if timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			deletionThreshold = t
		}
	}

	// Logger Setup

	setupLogger()
	log.Info().Msg("starting bot")

	token := os.Getenv(discordTokenEnvironmentVariable)
	if token == "" {
		log.Fatal().Msgf("environment variable %s is not set", discordTokenEnvironmentVariable)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating discord session")
	}

	err = dg.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("error opening connection to discord gateway")
	}
	defer dg.Close()

	// Register message handler
	dg.AddHandler(handleSlashCommand)
	dg.AddHandler(handleVoiceStateUpdate)

	// Register slash commands
	cleanupSlashCommands(dg)
	time.Sleep(5 * time.Second) // Wait for commands to be deleted
	registerSlashCommands(dg)

	startChannelCreator(dg)

	go startPrometheusServer()

	log.Info().Msg("bot is has started")

	// Gracefully handle shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info().Msg("interrupt caught, shutting down bot")
	cleanupSlashCommands(dg)
}

func isEnabledCategory(categoryID string) bool {
	for _, id := range categoryEnabled {
		if id == categoryID {
			return true
		}
	}
	return false
}

func createVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate, name string, userLimit int, categoryID string) {
	// Add suffix to the channel name
	channelName := fmt.Sprintf("%s%s", name, channelSuffix)

	// Create the voice channel
	channel, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildVoice,
		ParentID:             categoryID,
		UserLimit:            userLimit,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			// Permissions inherited from category
		},
	})
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to create voice channel.",
			},
		})
		BotLog(i, log.Error().Err(err)).Msg("error creating channel")
		return
	}

	response := fmt.Sprintf("Voice channel '%s' created.", channelName)

	// Move the user who issued the command into the newly created voice channel
	err = s.GuildMemberMove(i.GuildID, i.Member.User.ID, &channel.ID)
	moved, err := moveUserIfConnected(s, i.GuildID, i.Member.User.ID, channel.ID)
	if err != nil {
		BotLog(i, log.Error().Err(err)).Msg("error moving user to the new channel")
	} else if !moved {
		response = fmt.Sprintf("%s. This channel will be deleted in %d minutes if no one joins.", response, deletionThreshold)
		BotLog(i, log.Debug()).Msg("user not connected to a voice channel")
	}

	// Respond to the interaction indicating success
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Voice channel '%s' created.", channelName),
		},
	})

	// Monitor the channel for activity
	go monitorChannel(s, channel.ID, i.GuildID, categoryID)
}

func moveUserIfConnected(s *discordgo.Session, guildID, userID, channelID string) (bool, error) {
	// Get the user's voice state in the guild
	voiceState, err := getUserVoiceState(s, guildID, userID)
	if err != nil {
		return false, err // Handle other errors like network issues
	}

	// Check if the user is connected to any voice channel
	if voiceState == nil {
		return false, nil
	}

	// Move the user to the new channel if they are connected
	if err := s.GuildMemberMove(guildID, userID, &channelID); err != nil {
		return false, err
	}
	return true, nil
}

// Helper function to retrieve the user's voice state
func getUserVoiceState(s *discordgo.Session, guildID, userID string) (*discordgo.VoiceState, error) {
	// Get all voice states for the guild
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return nil, err
	}

	// Loop through all voice states to find the user
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs, nil // User is connected to a voice channel
		}
	}

	// Return nil if the user is not connected to any voice channel
	return nil, nil
}

func monitorChannel(s *discordgo.Session, channelID, guildID, categoryID string) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	emptyDuration := 0

	for range ticker.C {
		channel, err := s.Channel(channelID)
		if err != nil {
			log.Error().
				Err(err).
				Str("guild", guildID).
				Str("category", categoryID).
				Str("channel", channelID).
				Msg("error retrieving channel")
			return
		}

		if len(channel.Members) == 0 {
			emptyDuration += 1
			if emptyDuration >= deletionThreshold { // 3 minutes of being empty
				s.ChannelDelete(channelID)
				// Track channel deletion
				voiceChannelsDeleted.WithLabelValues(channel.GuildID, channel.ParentID).Inc()
				activeVoiceChannels.WithLabelValues(channel.GuildID, channel.ParentID).Dec()
				log.Debug().Str("channel", channel.Name).Msg("deleted empty channel")
				return
			}
		} else {
			emptyDuration = 0
		}
	}
}

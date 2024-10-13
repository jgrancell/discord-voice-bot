package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func channelCreatorExists(s *discordgo.Session, guildID, categoryID string) (bool, error) {
	// Get all channels for the guild
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return false, err
	}

	// Check if a creator channel already exists in the specified category
	for _, channel := range channels {
		if channel.ParentID == categoryID && channel.Name == channelCreatorName && channel.Type == discordgo.ChannelTypeGuildVoice {
			return true, nil
		}
	}

	return false, nil
}

func startChannelCreator(s *discordgo.Session) {
	ticker := time.NewTicker(10 * time.Second)

	// Goroutine to check for Channel Creator channels without blocking
	go func() {
		for {
			select {
			case <-ticker.C:
				// Call the function that checks and creates Channel Creator channels
				log.Debug().Msg("checking for channel creator channels...")
				createChannelCreatorChannels(s)
			}
		}
	}()
}

func createChannelCreatorChannels(s *discordgo.Session) {
	for _, categoryID := range categoryEnabled {
		// Check if the creator channel already exists in the category
		exists, err := channelCreatorExists(s, s.State.Guilds[0].ID, categoryID)
		if err != nil {
			log.Error().Err(err).Str("category", categoryID).Msgf("failed to check for existing '%s' channel", channelCreatorName)
			continue
		}

		if exists {
			log.Debug().Str("category", categoryID).Msgf("'%s' already exists, reusing it", channelCreatorName)
		} else {
			// Create creator channel for the whitelisted category
			channelName := channelCreatorName
			_, err := s.GuildChannelCreateComplex(s.State.Guilds[0].ID, discordgo.GuildChannelCreateData{
				Name:     channelName,
				Type:     discordgo.ChannelTypeGuildVoice,
				ParentID: categoryID,
			})
			if err != nil {
				log.Error().Err(err).Str("category", categoryID).Msgf("failed to create '%s' channel", channelCreatorName)
			} else {
				log.Info().Str("category", categoryID).Msgf("created '%s' channel", channelCreatorName)
			}
		}
	}
}

func handleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Check if the user has joined a creator channel
	if v.ChannelID == "" {
		return
	}
	joinedChannel, err := s.Channel(v.ChannelID)
	if err != nil {
		log.Error().Err(err).Str("channel", v.ChannelID).Msg("error retrieving channel")
		return
	}

	if joinedChannel.Name == channelCreatorName {
		// Create a personal channel for the user
		userChannelName := fmt.Sprintf("%s's Channel", v.Member.User.Username)

		newChannel, err := s.GuildChannelCreateComplex(v.GuildID, discordgo.GuildChannelCreateData{
			Name:     userChannelName,
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: joinedChannel.ParentID, // Keep it in the same category
		})
		if err != nil {
			log.Error().Err(err).Str("channel", userChannelName).Msg("error creating personal channel for user")
			return
		}

		// Move the user into their new personal channel
		err = s.GuildMemberMove(v.GuildID, v.UserID, &newChannel.ID)
		if err != nil {
			log.Error().Err(err).Str("channel", userChannelName).Msg("error moving user to personal channel")
			return
		}

		log.Info().Str("user", v.Member.User.Username).Str("channel", userChannelName).Msg("created personal channel and moved user")

		// Monitor the personal channel and clean up when empty
		go monitorChannel(s, newChannel.ID, v.GuildID, newChannel.ParentID)
	}
}

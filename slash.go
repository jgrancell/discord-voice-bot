package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func registerSlashCommands(dg *discordgo.Session) {
	command := &discordgo.ApplicationCommand{
		Name:        "voice",
		Description: "Create a temporary voice channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "create",
				Description: "Create a temporary voice channel",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "channel_name",
						Description: "The name of the voice channel",
						Type:        discordgo.ApplicationCommandOptionString,
						Required:    true,
					},
					{
						Name:        "max_users",
						Description: "Maximum number of users allowed in the channel",
						Type:        discordgo.ApplicationCommandOptionInteger,
						Required:    false,
					},
				},
			},
			//			{
			//				Name:        "enable",
			//				Description: "Enables this channel category for temporary voice channels",
			//				Type:        discordgo.ApplicationCommandOptionSubCommand,
			//			},
		},
	}

	//for _, id := range enabledGuilds {
	log.Info().Str("guild", "").Msg("registering slash commands")
	//voiceCommand, err := dg.ApplicationCommandCreate(dg.State.User.ID, id, command)
	voiceCommand, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", command)
	if err != nil {
		log.Error().Err(err).Msg("cannot create command")
	} else {
		log.Info().
			Str("command", botCommand).
			Str("id", voiceCommand.ID).
			Str("name", voiceCommand.Name).
			Str("guild", voiceCommand.GuildID).
			Msg("Successfully created command")
		//}

		commands, _ := dg.ApplicationCommands(dg.State.User.ID, "")
		for _, cmd := range commands {
			log.Info().
				Str("name", cmd.Name).
				Str("id", cmd.ID).
				Str("guild", cmd.GuildID).
				Msg("registered command")
		}
	}
}

func cleanupSlashCommands(dg *discordgo.Session) {
	log.Debug().Msg("cleaning up existing commands")
	// Fetch all commands for the application
	//for _, id := range enabledGuilds {
	commands, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch application commands")
		return
	}

	// Loop through each command and delete it
	for _, cmd := range commands {
		log.Debug().Str("command", cmd.Name).Msg("deleting command")
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
		if err != nil {
			log.Error().Err(err).Str("command", cmd.Name).Msg("cannot delete command")
		} else {
			log.Info().Str("command", cmd.Name).Msg("successfully deleted command")
		}
	}
	//}

}

func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug().Str("command", i.ApplicationCommandData().Options[0].Name).Msg("received command")

	switch i.ApplicationCommandData().Options[0].Name {
	case "create":
		handleVoiceCreate(s, i)
	case "enable":
		handleVoiceEnable(s, i)
	}
}

func handleVoiceCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Extract the options provided by the user
	options := i.ApplicationCommandData().Options[0].Options
	userLimit := 0 // Default to no limit
	var channelName string = ""

	// Get the max_users option (if provided)
	for _, opt := range options {
		switch opt.Name {
		case "max_users":
			userLimit = int(opt.IntValue())
		case "channel_name":
			channelName = opt.StringValue()
		}
	}

	// Get the category ID of the current channel
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		log.Printf("error retrieving channel: %v", err)
		return
	}

	// Ensure it's a enabled category
	if isEnabledCategory(channel.ParentID) {
		createVoiceChannel(s, i, channelName, userLimit, channel.ParentID)
	} else {
		log.Info().Str("category", channel.ParentID).Msg("category not enabled")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This Discord category is not whitelisted. Talk to your server admin.",
			},
		})
	}
}

func handleVoiceEnable(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle the "/voice enable" command
	// Get the channel where the command was invoked
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		log.Error().Err(err).Str("channel", i.ChannelID).Msg("Error retrieving channel")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to retrieve channel information.",
			},
		})
		return
	}

	// Check if the channel has a ParentID (meaning it's in a category)
	if channel.ParentID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This channel is not part of a category.",
			},
		})
		return
	}

	// Add the category to the allowlist if it's not already there
	if !isEnabledCategory(channel.ParentID) {
		categoryEnabled = append(categoryEnabled, channel.ParentID)
		log.Info().Str("category", channel.ParentID).Msg("Category added to allowlist")
	}

	// Respond to the user that the category has been allowed
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Category ID %s has been added to the allowlist.", channel.ParentID),
		},
	})

}

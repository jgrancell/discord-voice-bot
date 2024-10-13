package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func init() {
	prometheus.MustRegister(totalCommands)
	prometheus.MustRegister(voiceChannelsCreated)
	prometheus.MustRegister(voiceChannelsDeleted)
	prometheus.MustRegister(botErrors)
	prometheus.MustRegister(activeVoiceChannels)
}

var totalCommands = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discord_bot_commands_total",
		Help: "Total number of commands received by the bot.",
	},
	[]string{"server", "user"},
)

var voiceChannelsCreated = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discord_bot_voice_channels_created_total",
		Help: "Total number of voice channels created by the bot.",
	},
	[]string{"server", "category", "user"},
)

var voiceChannelsDeleted = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discord_bot_voice_channels_deleted_total",
		Help: "Total number of voice channels deleted by the bot.",
	},
	[]string{"server", "category"},
)

var botErrors = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discord_bot_errors_total",
		Help: "Total number of errors encountered by the bot.",
	},
	[]string{"server", "user"},
)

var activeVoiceChannels = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discord_bot_active_voice_channels",
		Help: "The current number of active voice channels created by the bot.",
	},
	[]string{"server", "category"},
)

func startPrometheusServer() {
	log.Error().Err(http.ListenAndServe(":2112", promhttp.Handler())).Msg("error starting Prometheus server")
}

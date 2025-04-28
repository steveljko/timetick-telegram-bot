package main

import (
	"log"
	"os"
)

type App struct {
	bot *Bot
}

func NewApp(bot *Bot) *App {
	return &App{
		bot: bot,
	}
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	bot, err := NewTelegramBot(botToken)
	if err != nil {
		log.Fatal("Failed to initialize bot: ", err)
	}

	app := NewApp(bot)

	app.bot.Start()
}

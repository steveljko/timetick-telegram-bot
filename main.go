package main

import (
	"log"
	"os"
	"strconv"
	"strings"
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

	authorizedUsersEnv := os.Getenv("AUTHORIZED_USERS")
	authorizedUsers := convertStringToIntArray(authorizedUsersEnv)
	if len(authorizedUsers) == 0 {
		log.Println("No authorized users specified.")
	}

	db, err := NewDatabase("database.db")
	if err != nil {
		log.Fatal(err)
	}

	bot, err := NewTelegramBot(botToken, authorizedUsers, db)
	if err != nil {
		log.Fatal("Failed to initialize bot: ", err)
	}

	app := NewApp(bot)

	app.bot.Start()
}

// Converts comma-separated string into slice of integers.
//
// Example:
//
//	Input: "123,456,789"
//	Output: []int64{123, 456, 789}
func convertStringToIntArray(arrString string) []int64 {
	var intArr []int64

	stringSlice := strings.Split(arrString, ",")

	for _, numString := range stringSlice {
		trimmed := strings.TrimSpace(numString) // remove trailing whitespace

		intValue, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			log.Printf("Skipping invalid integer: %q - %v", trimmed, err)
			continue
		}

		intArr = append(intArr, intValue)
	}

	return intArr
}

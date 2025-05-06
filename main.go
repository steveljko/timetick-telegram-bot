package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type App struct {
	db  *Database
	bot *Bot
}

func NewApp(db *Database, bot *Bot) *App {
	return &App{
		db:  db,
		bot: bot,
	}
}

func main() {
	args := os.Args
	app := createApp()

	if len(args) > 1 {
		command := args[1]
		handleCommand(app, command)
		return
	}

	app.Start()
}

func createApp() *App {
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

	return NewApp(db, bot)
}

func (a *App) Start() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		a.bot.Start()
	}()

	go func() {
		defer wg.Done()
		err := StartAPIServer(a, a.db, 3000)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
}

func (a *App) GenerateAPIToken() {
	token, err := GenerateToken()
	if err != nil {
		log.Fatal("Failed to generate token: ", err)
	}

	err = a.db.CreateApiToken(token)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("API Token generated successfully!")
	fmt.Println(token)
	fmt.Println("IMPORTANT: Save this token now. You won't be able to see it again!")
}

func handleCommand(app *App, command string) {
	switch command {
	case "start":
		app.Start()
	case "gen-api-token":
		app.GenerateAPIToken()
	}
}

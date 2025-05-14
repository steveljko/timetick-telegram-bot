package main

import (
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	authorizedUsers map[int64]bool
	db              *Database
	pendingNotes    map[int64]bool
}

type Sender struct {
	Id       int64
	Username string
}

func NewTelegramBot(token string, users []int64, db *Database) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	authorizedUsers := make(map[int64]bool)
	for _, id := range users {
		authorizedUsers[id] = true
	}

	return &Bot{
		api:             api,
		authorizedUsers: authorizedUsers,
		db:              db,
		pendingNotes:    make(map[int64]bool),
	}, nil
}

func (b *Bot) Start() {
	fmt.Printf("Authorized as %s\n", b.api.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		sender := &Sender{
			Id:       update.Message.From.ID,
			Username: update.Message.From.UserName,
		}

		if !b.isAuthorized(sender.Id) {
			text := fmt.Sprintf("You are not authorized to use this bot. \nYour Telegram ID is: %d", sender.Id)
			b.sendMessage(update.Message.Chat.ID, text, update.Message.MessageID)
			continue
		}

		// check if user has active request for starting a timer without note
		if b.hasPendingNote(sender.Id) {
			b.processPendingNote(update.Message.Chat.ID, update.Message.MessageID, update.Message.Text, sender.Id)
			continue
		}

		if update.Message.IsCommand() {
			log.Printf("Received command from %s (ID: %d)\n", sender.Username, sender.Id)
			b.handleCommand(update.Message)
			continue
		}
	}
}

// Check if user has a pending note request
func (b *Bot) hasPendingNote(userId int64) bool {
	_, exists := b.pendingNotes[userId]
	return exists
}

// Process pending note request
func (b *Bot) processPendingNote(chatId int64, messageId int, note string, userId int64) {
	// convert "x" to empty string
	if note == "x" {
		note = ""
	}

	err := b.db.StartTracking(strconv.FormatInt(userId, 10), note)
	if err != nil {
		b.sendMessage(chatId, fmt.Sprintf("%s", err), messageId)
		delete(b.pendingNotes, userId)
		return
	}

	message := "⏲️ Timer is started."
	if note != "" {
		message += " Note is: " + note
	}
	message += "\nUse /stop for stopping timer."

	b.sendMessage(chatId, message, messageId)
	delete(b.pendingNotes, userId)
}

// Checks if user has permission to interact with the bot.
//
// Returns true if:
// 1. No authorized users are configured (open access mode)
// 2. The user's ID exists in the authorized users map with a value of true
func (b *Bot) isAuthorized(userID int64) bool {
	// If the authorized users list is empty, allow access to everyone
	if len(b.authorizedUsers) == 0 {
		return true
	}

	// Check if the user exists in the map and is authorized
	isAllowed, exists := b.authorizedUsers[userID]
	return exists && isAllowed
}

// Sends message to specific Telegram chat with an option to reply to another message.
//
// Parameters:
//
//	chatID: The unique identifier of the target chat
//	text: The content of the message to be sent
//	replyToID: The ID of the message to reply to (0 for no reply)
func (b *Bot) sendMessage(chatID int64, text string, replyToID int) {
	msg := tgbotapi.NewMessage(chatID, text)
	if replyToID > 0 {
		msg.ReplyToMessageID = replyToID
	}

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// Processes incoming bot commands and routes them to appropriate functionalities.
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	command := message.Command()
	args := message.CommandArguments()
	userAjDi := message.From.ID
	userID := strconv.FormatInt(message.From.ID, 10)

	log.Printf("Handling command: %s, args: %s, userID: %s", command, args, userID)

	switch command {
	case "start":
		if len(args) == 0 {
			b.pendingNotes[userAjDi] = true
			b.sendMessage(message.Chat.ID, "Please enter your note or type 'x' if you do not wish to provide a note.", message.MessageID)
			return
		}
		err := b.db.StartTracking(userID, args)
		if err != nil {
			b.sendMessage(message.Chat.ID, fmt.Sprintf("%s", err), message.MessageID)
			return
		}
		b.sendMessage(message.Chat.ID, "⏲️ Timer is started.\nUse /stop for stopping timer.", message.MessageID)
	case "stop":
		_, err := b.db.StopTracking(userID)
		if err != nil {
			b.sendMessage(message.Chat.ID, fmt.Sprintf("%s", err), message.MessageID)
			return
		}
		b.sendMessage(message.Chat.ID, "❌ Timer is stopped.", message.MessageID)
		delete(b.pendingNotes, userAjDi)
	case "help":
		helpText := "Available commands:\n" +
			"/start - Starts timer with optional note\n" +
			"/stop - Stops timer\n" +
			"/help - Show this help message"
		b.sendMessage(message.Chat.ID, helpText, message.MessageID)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Type /help to see available commands.", message.MessageID)
	}
}

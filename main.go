package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	refusalMessage = "Désolé mais je ne peux répondre qu'aux questions concernant les concerts et les playlists de Groove Station, le meilleur groupe de Funk du monde."
	errorMessage   = "Désolé, je n'arrive pas à récupérer l'information pour le moment. Réessaie un peu plus tard."
)

type Bot struct {
	config      Config
	db          *SQLiteDB
	llm         *OpenRouterClient
	schema      string
	outputFile  string
	outputMutex sync.Mutex
}

type questionDetails struct {
	path         string
	query        string
	rows         int
	participants int
}

func (bot *Bot) answerQuestion(ctx context.Context, question string, now time.Time) (string, questionDetails, error) {
	if date, ok := rehearsalParticipantsDate(question); ok {
		answer, count, err := bot.answerRehearsalParticipantsOnDate(ctx, date)
		return answer, questionDetails{path: "rehearsal participants by date", participants: count}, err
	}
	if isNextRehearsalParticipantsQuestion(question) {
		answer, count, err := bot.answerNextRehearsalParticipantsWithCount(ctx, now)
		return answer, questionDetails{path: "next rehearsal participants", participants: count}, err
	}
	query, err := bot.llm.GenerateSQL(ctx, question, bot.schema)
	if err != nil {
		return "", questionDetails{path: "LLM"}, err
	}
	query, err = ValidateSelectQuery(query)
	if err == nil {
		err = validateGigTypeQuery(query, classifyGigQuestion(question))
	}
	if err != nil {
		return "", questionDetails{path: "LLM", query: query}, err
	}
	rows, err := bot.db.FetchRows(ctx, query)
	if err != nil {
		return "", questionDetails{path: "LLM", query: query}, err
	}
	answer, err := bot.llm.GenerateAnswer(ctx, question, query, rows)
	return answer, questionDetails{path: "LLM", query: query, rows: len(rows)}, err
}

func (bot *Bot) setup(ctx context.Context) error {
	var err error
	bot.schema, err = bot.db.SchemaSummary(ctx)
	return err
}

func (bot *Bot) onReady(session *discordgo.Session, event *discordgo.Ready) {
	guildIDs := make([]string, 0, len(bot.config.DiscordGuildIDs))
	for id := range bot.config.DiscordGuildIDs {
		guildIDs = append(guildIDs, id)
	}
	sort.Strings(guildIDs)
	words := make([]string, 0, len(bot.config.WhitelistWords))
	for word := range bot.config.WhitelistWords {
		words = append(words, word)
	}
	sort.Strings(words)
	fmt.Printf("Logged in as %s\nAllowed Discord guild IDs: %v\nWhitelist words: %v\nMessage log file: %s\n", session.State.User.Username, guildIDs, words, bot.outputFile)
}

func (bot *Bot) onMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if session.State.User == nil {
		return
	}
	if message.Author.ID == session.State.User.ID {
		return
	}
	if !mentionsUser(message.Mentions, session.State.User.ID) {
		return
	}
	fmt.Printf("Received mention message=%s guild_id=%s channel_id=%s author=%s\n", message.ID, message.GuildID, message.ChannelID, message.Author)
	if len(bot.config.DiscordGuildIDs) > 0 && !bot.config.DiscordGuildIDs[message.GuildID] {
		fmt.Printf("Ignoring message=%s: guild not allowed\n", message.ID)
		return
	}
	if !isWhitelistedMessage(message.Content, bot.config.WhitelistWords) {
		fmt.Printf("Rejecting message=%s: whitelist mismatch\n", message.ID)
		if _, err := session.ChannelMessageSendReply(message.ChannelID, refusalMessage, message.Reference()); err != nil {
			fmt.Printf("Failed to send refusal reply for message %s: %v\n", message.ID, err)
		}
		return
	}
	status, err := sendStatusReply(session, message)
	if err != nil {
		fmt.Printf("Failed to send status reply for message %s: %v\n", message.ID, err)
	}
	bot.writePayload(message)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	answer, _, err := bot.answerQuestion(ctx, message.Content, time.Now())
	if err != nil {
		fmt.Printf("Failed to process message %s: %v\n", message.ID, err)
		bot.replyOrEdit(session, message, status, errorMessage)
		return
	}
	if len(answer) > 1900 {
		answer = answer[:1900]
	}
	bot.replyOrEdit(session, message, status, answer)
}

func sendStatusReply(session *discordgo.Session, message *discordgo.MessageCreate) (*discordgo.Message, error) {
	return session.ChannelMessageSendComplex(message.ChannelID, &discordgo.MessageSend{
		Content:   "Je cherche...",
		Reference: message.Reference(),
	})
}

func (bot *Bot) replyOrEdit(session *discordgo.Session, original *discordgo.MessageCreate, status *discordgo.Message, content string) {
	if status == nil {
		_, err := session.ChannelMessageSendReply(original.ChannelID, content, original.Reference())
		if err != nil {
			fmt.Printf("Failed to send reply for message %s: %v\n", original.ID, err)
		}
		return
	}
	if _, err := session.ChannelMessageEdit(status.ChannelID, status.ID, content); err != nil {
		fmt.Printf("Failed to edit status message %s: %v\n", status.ID, err)
		if _, replyErr := session.ChannelMessageSendReply(original.ChannelID, content, original.Reference()); replyErr != nil {
			fmt.Printf("Failed to send fallback reply for message %s: %v\n", original.ID, replyErr)
		}
	}
}

func (bot *Bot) writePayload(message *discordgo.MessageCreate) {
	payload := map[string]any{
		"received_at": time.Now().UTC().Format(time.RFC3339Nano),
		"guild_id":    message.GuildID,
		"channel_id":  message.ChannelID,
		"author_id":   message.Author.ID,
		"author":      message.Author.String(),
		"content":     message.Content,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to encode Discord message log: %v\n", err)
		return
	}
	bot.outputMutex.Lock()
	defer bot.outputMutex.Unlock()
	if err := os.MkdirAll(filepath.Dir(bot.outputFile), 0o755); err != nil {
		fmt.Printf("Failed to create Discord message log directory: %v\n", err)
		return
	}
	file, err := os.OpenFile(bot.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("Failed to write Discord message log: %v\n", err)
		return
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		fmt.Printf("Failed to write Discord message log: %v\n", err)
	}
}

var messageWordPattern = regexp.MustCompile(`[\pL\pN_]+`)

func isWhitelistedMessage(content string, whitelist map[string]bool) bool {
	allowed := make(map[string]bool)
	for word := range whitelist {
		for variant := range wordVariants(word) {
			allowed[variant] = true
		}
	}
	for _, word := range messageWordPattern.FindAllString(strings.ToLower(content), -1) {
		if variants := wordVariants(word); anyVariantInSet(variants, allowed) {
			return true
		}
	}
	return false
}

func wordVariants(word string) map[string]bool {
	word = strings.ToLower(strings.TrimSpace(word))
	variants := map[string]bool{word: true}
	// The message tokenizer splits off apostrophes, so whitelist entries such
	// as "répet'" must also match the token "répet".
	if withoutApostrophe := strings.TrimRight(word, "'’"); withoutApostrophe != word {
		variants[withoutApostrophe] = true
	}
	if len([]rune(word)) > 3 {
		for _, suffix := range []string{"s", "x", "es"} {
			if strings.HasSuffix(word, suffix) {
				variants[strings.TrimSuffix(word, suffix)] = true
			}
		}
	}
	return variants
}

func anyVariantInSet(variants, allowed map[string]bool) bool {
	for variant := range variants {
		if allowed[variant] {
			return true
		}
	}
	return false
}

func mentionsUser(mentions []*discordgo.User, userID string) bool {
	for _, user := range mentions {
		if user.ID == userID {
			return true
		}
	}
	return false
}

func main() {
	question := flag.String("question", "", "answer one question without connecting to Discord")
	dbPath := flag.String("db", "", "SQLite database path for -question (defaults to SQLITE_PATH in .env)")
	debug := flag.Bool("debug", false, "show question path and row counts in CLI mode")
	nowFlag := flag.String("now", "", "override current time in CLI mode (RFC3339)")
	flag.Parse()
	if *question != "" {
		if err := runQuestionCLI(*question, *dbPath, *debug, *nowFlag, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *dbPath != "" || *debug || *nowFlag != "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "-db, -debug and -now require -question; pass the question with -question")
		os.Exit(2)
	}
	configPath := findEnvFile()
	config, err := LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	database, err := OpenSQLite(config.SQLitePath)
	if err != nil {
		panic(err)
	}
	defer database.Close()

	bot := &Bot{
		config:     config,
		db:         database,
		llm:        NewOpenRouterClient(config),
		outputFile: defaultValue(os.Getenv("BOT_OUTPUT_FILE"), filepath.Join(filepath.Dir(configPath), "logs", "discord_messages.jsonl")),
	}
	if err := bot.setup(context.Background()); err != nil {
		panic(fmt.Errorf("load database schema: %w", err))
	}

	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		panic(err)
	}
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onMessage)
	if err := session.Open(); err != nil {
		panic(err)
	}
	defer session.Close()
	fmt.Println("Bot is running")
	select {}
}

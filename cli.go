package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

func runQuestionCLI(question, dbPath string, debug bool, nowFlag string, output, diagnostics io.Writer) error {
	if strings.TrimSpace(question) == "" {
		return fmt.Errorf("-question must not be empty")
	}
	now := time.Now()
	if nowFlag != "" {
		var err error
		now, err = time.Parse(time.RFC3339, nowFlag)
		if err != nil {
			return fmt.Errorf("parse -now (expected RFC3339): %w", err)
		}
	}

	values, envErr := readEnvFile(findEnvFile())
	if dbPath == "" {
		if envErr != nil {
			return envErr
		}
		dbPath = defaultValue(values["SQLITE_PATH"], values["DB_PATH"])
	}
	if dbPath == "" {
		return fmt.Errorf("-db or SQLITE_PATH in .env is required")
	}

	database, err := OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()
	bot := &Bot{db: database}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if !isNextRehearsalParticipantsQuestion(question) {
		if envErr != nil {
			return fmt.Errorf("LLM questions require an .env file: %w", envErr)
		}
		key := defaultValue(values["OPENROUTER_API_KEY"], values["LLM_TOKEN"])
		if key == "" {
			return fmt.Errorf("LLM questions require OPENROUTER_API_KEY or LLM_TOKEN in .env")
		}
		bot.llm = NewOpenRouterClient(Config{OpenRouterAPIKey: key, OpenRouterModel: defaultValue(values["OPENROUTER_MODEL"], "openrouter/auto")})
		if err := bot.setup(ctx); err != nil {
			return fmt.Errorf("load database schema: %w", err)
		}
	}
	answer, details, err := bot.answerQuestion(ctx, question, now)
	if debug {
		fmt.Fprintf(diagnostics, "Path: %s\n", details.path)
		if details.path == "next rehearsal participants" {
			fmt.Fprintf(diagnostics, "Participants returned: %d\n", details.participants)
		} else if details.query != "" {
			fmt.Fprintf(diagnostics, "SQL: %s\nRows returned: %d\n", details.query, details.rows)
		}
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, answer)
	return err
}

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	containerEnvFile = "/home/bot/.env"
	hostEnvFile      = "/home/setpilot/agent/.env"
)

type Config struct {
	DiscordToken     string
	DiscordGuildIDs  map[string]bool
	WhitelistWords   map[string]bool
	OpenRouterAPIKey string
	OpenRouterModel  string
	SQLitePath       string
}

func LoadConfig(path string) (Config, error) {
	values, err := readEnvFile(path)
	if err != nil {
		return Config{}, err
	}

	discordToken, err := require(values, "DISCORD_TOKEN", path)
	if err != nil {
		return Config{}, err
	}
	apiKey, err := requireAny(values, []string{"OPENROUTER_API_KEY", "LLM_TOKEN"}, path)
	if err != nil {
		return Config{}, err
	}
	sqlitePath, err := requireAny(values, []string{"SQLITE_PATH", "DB_PATH"}, path)
	if err != nil {
		return Config{}, err
	}

	guildIDs, err := parseIntSet(values["DISCORD_GUILD_IDS"])
	if err != nil {
		return Config{}, fmt.Errorf("invalid DISCORD_GUILD_IDS: %w", err)
	}

	return Config{
		DiscordToken:     discordToken,
		DiscordGuildIDs:  guildIDs,
		WhitelistWords:   parseStringSet(values["WHITE_LIST_WORDS"]),
		OpenRouterAPIKey: apiKey,
		OpenRouterModel:  defaultValue(values["OPENROUTER_MODEL"], "openrouter/auto"),
		SQLitePath:       sqlitePath,
	}, nil
}

func findEnvFile() string {
	if path := os.Getenv("SET_PILOT_ENV_FILE"); path != "" {
		return path
	}
	for _, path := range []string{containerEnvFile, hostEnvFile, ".env"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return containerEnvFile
}

func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(name)] = strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return values, nil
}

func require(values map[string]string, key, path string) (string, error) {
	if value := values[key]; value != "" {
		return value, nil
	}
	return "", fmt.Errorf("%s is required in %s", key, path)
}

func requireAny(values map[string]string, keys []string, path string) (string, error) {
	for _, key := range keys {
		if value := values[key]; value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("one of %s is required in %s", strings.Join(keys, ", "), path)
}

func parseIntSet(value string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, err := strconv.ParseInt(item, 10, 64); err != nil {
			return nil, err
		}
		result[item] = true
	}
	return result, nil
}

func parseStringSet(value string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result[item] = true
		}
	}
	return result
}

func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Config holds pagination state
type Config struct {
	NextURL string
	PrevURL string
}

type cliCommand struct {
	name        string
	description string
	callback    func(*Config) error
}

var commands = map[string]cliCommand{
	"exit": {
		name:        "exit",
		description: "Exit the Pokedex",
		callback:    commandExit,
	},
	"help": {
		name:        "help",
		description: "Displays a help message",
		callback:    commandHelp,
	},
	"map": {
		name:        "map",
		description: "Displays the next 20 location names",
		callback:    commandMap,
	},
	"mapb": {
		name:        "mapb",
		description: "Displays the previous 20 location names",
		callback:    commandMapBack,
	},
}

func commandExit(cfg *Config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config) error {
	fmt.Println(`Welcome to the Pokedex!
Usage:
  help   - Displays this help message
  exit   - Exit the Pokedex
  map    - Shows the next 20 locations
  mapb   - Shows the previous 20 locations`)
	return nil
}

func commandMap(cfg *Config) error {
	url := cfg.NextURL
	if url == "" {
		url = "https://pokeapi.co/api/v2/location-area/"
	}

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var data struct {
		Count    int    `json:"count"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
		Results  []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	// update config with new pagination links
	cfg.NextURL = data.Next
	cfg.PrevURL = data.Previous

	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}

	return nil
}

func commandMapBack(cfg *Config) error {
	if cfg.PrevURL == "" {
		fmt.Println("you're on the first page")
		return nil
	}

	// set NextURL to PrevURL so commandMap uses it
	cfg.NextURL = cfg.PrevURL
	cfg.PrevURL = "" // will be reset by commandMap
	return commandMap(cfg)
}

func main() {
	cfg := &Config{}
	reader := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")
		if !reader.Scan() {
			break
		}
		words := cleanInput(reader.Text())

		if len(words) == 0 {
			continue
		}

		cmd, exists := commands[words[0]]
		if !exists {
			fmt.Println("Unknown command:", words[0])
			continue
		}

		if err := cmd.callback(cfg); err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func cleanInput(text string) []string {
	cleaned := strings.TrimSpace(strings.ToLower(text))
	return strings.Fields(cleaned)
}


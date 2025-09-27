package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ask-elad/pokedex/internal/utils"
)

// Config holds pagination state + cache
type Config struct {
	NextURL string
	PrevURL string
	Cache   *utils.Cache
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
	"explore": {
		name:        "explore",
		description: "Explore the particular map location",
		callback:    nil, // handled specially in main()
	},
}

func commandExit(cfg *Config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

// explore with area argument
func commandExplore(cfg *Config, locationArea string) error {
	// 🔹 Check cache
	if cached, ok := cfg.Cache.Get(locationArea); ok {
		return printExplore(cached)
	}

	// 🔹 Fetch from API
	url := "https://pokeapi.co/api/v2/location-area/" + locationArea
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	cfg.Cache.Add(locationArea, body)
	return printExplore(body)
}

// helper for explore
func printExplore(body []byte) error {
	var data struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	for _, p := range data.PokemonEncounters {
		fmt.Println(p.Pokemon.Name)
	}
	return nil
}

func commandHelp(cfg *Config) error {
	fmt.Println(`Welcome to the Pokedex!
Usage:
  help             - Displays this help message
  exit             - Exit the Pokedex
  map              - Shows the next 20 locations
  mapb             - Shows the previous 20 locations
  explore <area>   - Explore Pokémon in a location area`)
	return nil
}

func commandMap(cfg *Config) error {
	url := cfg.NextURL
	if url == "" {
		url = "https://pokeapi.co/api/v2/location-area/"
	}

	// 🔹 Check cache first
	if cached, ok := cfg.Cache.Get(url); ok {
		return printLocations(cfg, cached)
	}

	// 🔹 Otherwise make request
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Save to cache
	cfg.Cache.Add(url, body)

	return printLocations(cfg, body)
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

// 🔹 helper to decode and print locations
func printLocations(cfg *Config, body []byte) error {
	var data struct {
		Count    int    `json:"count"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
		Results  []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
	}

	err := json.Unmarshal(body, &data)
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

func main() {
	cfg := &Config{
		Cache: utils.NewCache(5 * time.Minute), // cache entries live for 5m
	}

	reader := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		if !reader.Scan() {
			break
		}
		words := CleanInput(reader.Text())
		if len(words) == 0 {
			continue
		}

		cmd, exists := commands[words[0]]
		if !exists {
			fmt.Println("Unknown command:", words[0])
			continue
		}

		// 🔹 handle explore specially
		if words[0] == "explore" {
			if len(words) < 2 {
				fmt.Println("Usage: explore <location-area>")
				continue
			}
			if err := commandExplore(cfg, words[1]); err != nil {
				fmt.Println("Error:", err)
			}
			continue
		}

		if err := cmd.callback(cfg); err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func CleanInput(text string) []string {
	cleaned := strings.TrimSpace(strings.ToLower(text))
	return strings.Fields(cleaned)
}

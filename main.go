package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ask-elad/pokedex/internal/utils"
)

type Config struct {
	NextURL         string
	PrevURL         string
	Cache           *utils.Cache
	Pokedex         map[string]Pokemon // caught pokemon
	CurrentLocation string             // last location-area we explored
}

type Pokemon struct {
	Name           string `json:"name"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	BaseExperience int    `json:"base_experience"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
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
		description: "Show help",
		callback:    commandHelp,
	},
	"map": {
		name:        "map",
		description: "Show next 20 location areas",
		callback:    commandMap,
	},
	"mapb": {
		name:        "mapb",
		description: "Show previous 20 location areas",
		callback:    commandMapBack,
	},
	"explore": {
		name:        "explore",
		description: "Explore a location-area (special-cased in main)",
		callback:    nil,
	},
	"catch": {
		name:        "catch",
		description: "Try to catch a Pokémon (special-cased in main)",
		callback:    nil,
	},
	"inspect": {
		name:        "inspect",
		description: "Inspect a Pokémon you've caught (special-cased in main)",
		callback:    nil,
	},
	"pokedex": {
		name:        "pokedex",
		description: "List your caught Pokémon (special-cased in main)",
		callback:    nil,
	},
}

func commandCatch(cfg *Config, pokemon string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	pokeName := strings.ToLower(pokemon)

	if cfg.CurrentLocation == "" {
		fmt.Println("You haven't explored any location yet. Use: explore <location-area>")
		return nil
	}

	// make sure we have the location data cached so we can check encounters
	locKey := cfg.CurrentLocation
	locBody, ok := cfg.Cache.Get(locKey)
	if !ok {
		url := "https://pokeapi.co/api/v2/location-area/" + locKey
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed fetching location: %s", res.Status)
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.Cache.Add(locKey, data)
		locBody = data
	}

	// if the requested pokemon isn't in the current area's encounters, bail out
	present, err := locationHasPokemon(locBody, pokeName)
	if err != nil {
		return err
	}
	if !present {
		fmt.Printf("there isn't any %s here explore more\n", pokeName)
		return nil
	}

	// now fetch the pokemon data (or get from cache)
	var body []byte
	if cached, ok := cfg.Cache.Get(pokeName); ok {
		body = cached
	} else {
		url := "https://pokeapi.co/api/v2/pokemon/" + pokeName
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("pokemon not found or API error: %s", res.Status)
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.Cache.Add(pokeName, data)
		body = data
	}

	var poke Pokemon
	if err := json.Unmarshal(body, &poke); err != nil {
		return err
	}

	fmt.Printf("Throwing a Pokeball at %s...\n", poke.Name)

	// catch logic: higher base_experience -> harder to catch
	threshold := 50
	roll := rand.Intn(poke.BaseExperience + threshold)
	if roll < threshold {
		fmt.Printf("%s was caught!\n", poke.Name)
		if cfg.Pokedex == nil {
			cfg.Pokedex = make(map[string]Pokemon)
		}
		// store under lowercase name so lookups are consistent
		cfg.Pokedex[strings.ToLower(poke.Name)] = poke
	} else {
		fmt.Printf("%s escaped!\n", poke.Name)
	}
	return nil
}

func locationHasPokemon(body []byte, pokemonLower string) (bool, error) {
	var data struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return false, err
	}
	for _, e := range data.PokemonEncounters {
		if strings.ToLower(e.Pokemon.Name) == pokemonLower {
			return true, nil
		}
	}
	return false, nil
}

func commandInspect(cfg *Config, pokemon string) error {
	if cfg == nil || cfg.Pokedex == nil {
		fmt.Println("you have not caught that pokemon yet")
		return nil
	}
	name := strings.ToLower(pokemon)
	p, ok := cfg.Pokedex[name]
	if !ok {
		fmt.Println("you have not caught that pokemon")
		return nil
	}

	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Height: %d\n", p.Height)
	fmt.Printf("Weight: %d\n", p.Weight)

	fmt.Println("Stats:")
	for _, s := range p.Stats {
		fmt.Printf("  -%s: %d\n", s.Stat.Name, s.BaseStat)
	}

	fmt.Println("Types:")
	for _, t := range p.Types {
		fmt.Printf("  - %s\n", t.Type.Name)
	}
	return nil
}

func commandPokedex(cfg *Config) error {
	if cfg == nil || cfg.Pokedex == nil || len(cfg.Pokedex) == 0 {
		fmt.Println("you haven't caught any pokemon yet")
		return nil
	}
	names := make([]string, 0, len(cfg.Pokedex))
	for n := range cfg.Pokedex {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("Your Pokedex:")
	for _, n := range names {
		fmt.Printf(" - %s\n", n)
	}
	return nil
}

func commandExit(cfg *Config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandExplore(cfg *Config, locationArea string) error {
	key := strings.ToLower(locationArea)
	if cached, ok := cfg.Cache.Get(key); ok {
		cfg.CurrentLocation = key
		return printExplore(cached)
	}

	url := "https://pokeapi.co/api/v2/location-area/" + key
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("location-area not found or API error: %s", res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	cfg.Cache.Add(key, body)
	cfg.CurrentLocation = key
	return printExplore(body)
}

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
help        - show this message
exit        - quit
map         - show next 20 location areas
mapb        - show previous 20 location areas
explore <area>    - inspect a specific location-area
catch <pokemon>   - try to catch a Pokémon (only if present in your current area)
inspect <pokemon> - view details of a Pokémon you've caught
pokedex           - list the Pokémon you've caught`)
	return nil
}

func commandMap(cfg *Config) error {
	url := cfg.NextURL
	if url == "" {
		url = "https://pokeapi.co/api/v2/location-area/"
	}
	if cached, ok := cfg.Cache.Get(url); ok {
		return printLocations(cfg, cached)
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
	cfg.Cache.Add(url, body)
	return printLocations(cfg, body)
}

func commandMapBack(cfg *Config) error {
	if cfg.PrevURL == "" {
		fmt.Println("you're on the first page")
		return nil
	}
	cfg.NextURL = cfg.PrevURL
	cfg.PrevURL = ""
	return commandMap(cfg)
}

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
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}
	cfg.NextURL = data.Next
	cfg.PrevURL = data.Previous
	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}
	return nil
}

func main() {
	// seed once so catches are random
	rand.Seed(time.Now().UnixNano())

	cfg := &Config{
		Cache:   utils.NewCache(5 * time.Minute),
		Pokedex: make(map[string]Pokemon),
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

		switch words[0] {
		case "explore":
			if len(words) < 2 {
				fmt.Println("Usage: explore <location-area>")
				continue
			}
			if err := commandExplore(cfg, words[1]); err != nil {
				fmt.Println("Error:", err)
			}
			continue
		case "catch":
			if len(words) < 2 {
				fmt.Println("Usage: catch <pokemon>")
				continue
			}
			if err := commandCatch(cfg, words[1]); err != nil {
				fmt.Println("Error:", err)
			}
			continue
		case "inspect":
			if len(words) < 2 {
				fmt.Println("Usage: inspect <pokemon>")
				continue
			}
			if err := commandInspect(cfg, words[1]); err != nil {
				fmt.Println("Error:", err)
			}
			continue
		case "pokedex":
			if err := commandPokedex(cfg); err != nil {
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

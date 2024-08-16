package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Environments []Environment `yaml:"environments"`
	Requests     []Request     `yaml:"requests"`
}

type Environment struct {
	Name      string     `yaml:"name"`
	Variables []KeyValue `yaml:"variables"`
}

type Request struct {
	Name    string     `yaml:"name"`
	Method  string     `yaml:"method"`
	URL     string     `yaml:"url"`
	Headers []KeyValue `yaml:"headers"`
	Body    string     `yaml:"body"`
	Asserts []Assert   `yaml:"asserts"`
}

type Assert struct {
	Description string `yaml:"description"`
	Key         string `yaml:"key"`
	Value       string `yaml:"value"`
}
type KeyValue struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func loadConfig() Config {
	// load mjau.yaml file
	var config Config

	file, err := os.ReadFile("mjau.yaml")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return config
}

func (config Config) replaceVariables(request Request, environment Environment) Request {
	// Replace variables in request with values from environment
	for _, variable := range environment.Variables {
		request.URL = strings.ReplaceAll(request.URL, "{"+variable.Key+"}", variable.Value)
		request.Body = strings.ReplaceAll(request.Body, "{"+variable.Key+"}", variable.Value)
		for _, header := range request.Headers {
			header.Value = strings.ReplaceAll(header.Value, "{"+variable.Key+"}", variable.Value)
		}
	}
	return request
}

func AnsiColor(str string, r, g, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, str)
}

func run(requestName string) {
	config := loadConfig()
	errors := 0
	found := false
	for _, request := range config.Requests {
		if request.Name == requestName {
			found = true
			request = config.replaceVariables(request, config.Environments[0])
			println("😺 Running request - " + request.Name)

			req, err := http.NewRequest(
				request.Method,
				request.URL,
				strings.NewReader(request.Body),
			)
			if err != nil {
				log.Fatal(err)
			}
			for _, header := range request.Headers {
				req.Header.Set(header.Key, header.Value)
			}
			start := time.Now()
			resp, err := http.DefaultClient.Do(req)
			elapsed := time.Since(start)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}
			println("🕒 Request took " + elapsed.String())
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			body = []byte(strings.TrimSuffix(string(body), "\n"))
			println(
				AnsiColor("HTTP/1.1 ", 53, 42, 226)+
					AnsiColor(strconv.Itoa(
						resp.StatusCode), 53, 42, 226),
				http.StatusText(
					resp.StatusCode,
				),
			)
			for key, value := range resp.Header {
				println(AnsiColor(key, 53, 177, 226) + ": " + strings.Join(value, ", "))
			}
			println("")
			println(string(body) + "\n")

			if len(request.Asserts) > 0 {
				println("👀 Asserts:")
			}
			for _, assert := range request.Asserts {
				println("" + AnsiColor(assert.Description, 53, 177, 226))
				if assert.Key == "status_code" {
					if strconv.Itoa(resp.StatusCode) != assert.Value {
						errors++
						fmt.Printf(
							"  ❌ Status code does not match. Expected: %s, got: %d\n",
							assert.Value,
							resp.StatusCode,
						)
					} else {
						println("  ✅ Status code matches")
					}
				}
				if assert.Key == "body" {
					if string(body) != assert.Value {
						errors++
						fmt.Printf(
							"  ❌ Body does not match. Expected: '%s', got: '%s'\n",
							assert.Value,
							string(body),
						)
					} else {
						println("  ✅ Body matches")
					}
				}
			}

		}
	}
	println("")
	if !found {
		println("😿 Request not found")
		os.Exit(1)
	}
	if errors > 0 {
		println("😿 There were errors in the request")
		os.Exit(1)
	} else {
		println("😻 Request ran successfully")
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		println("Please provide a command to run. Example: mjau run <request>")
		os.Exit(1)
	}
	command := args[0]
	if command == "run" {
		if len(args) == 1 {
			println("Please provide a request to run. Example: mjau run healthz")
			os.Exit(1)
		}
		run(args[1])
	}
	if command == "runall" {
		config := loadConfig()
		for i, request := range config.Requests {
			if i > 0 {
				println("----------------------------------------")
			}
			run(request.Name)
		}
	}
	os.Exit(0)
}

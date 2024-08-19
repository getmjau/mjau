package mjau

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Environments    []Environment `yaml:"environments"`
	StoredVariables []KeyValue
	Requests        []Request `yaml:"requests"`
}

type Environment struct {
	Name      string     `yaml:"name"`
	Variables []KeyValue `yaml:"variables"`
}

type Request struct {
	Name               string              `yaml:"name"`
	Method             string              `yaml:"method"`
	URL                string              `yaml:"url"`
	Headers            []KeyValue          `yaml:"headers"`
	Body               string              `yaml:"body"`
	StoreJsonVariables []StoreJsonVariable `yaml:"store_json_variables"`
	Asserts            []Assert            `yaml:"asserts"`
}

type StoreJsonVariable struct {
	Path string `yaml:"path"`
	Key  string `yaml:"key"`
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

func loadConfig(configfile string) Config {
	// load mjau.yaml file
	var config Config

	file, err := os.ReadFile(configfile)
	if err != nil {
		println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		println(err)
		os.Exit(1)
	}

	return config
}

func (config Config) replaceVariables(
	request Request,
	environment Environment,
	savedVariables []KeyValue,
) Request {
	// Replace variables in request with values from environment
	for _, variable := range environment.Variables {
		request.URL = strings.ReplaceAll(request.URL, "{"+variable.Key+"}", variable.Value)
		request.Body = strings.ReplaceAll(request.Body, "{"+variable.Key+"}", variable.Value)
		for i, header := range request.Headers {
			header.Value = strings.ReplaceAll(header.Value, "{"+variable.Key+"}", variable.Value)
			request.Headers[i] = header
		}
	}
	// Replace variables in request with saved replaceVariables
	for _, variable := range savedVariables {
		request.URL = strings.ReplaceAll(request.URL, "{"+variable.Key+"}", variable.Value)
		request.Body = strings.ReplaceAll(request.Body, "{"+variable.Key+"}", variable.Value)
		for i, header := range request.Headers {
			header.Value = strings.ReplaceAll(header.Value, "{"+variable.Key+"}", variable.Value)
			request.Headers[i] = header
		}
	}
	return request
}

func AnsiColor(str string, r, g, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, str)
}

func GetJsonValueFromPath(jsonStr string, path string) string {
	return gjson.Get(jsonStr, path).String()
}

func GetJsonValues(jsonStr string, request Request, config *Config) {
	for _, getJsonVariable := range request.StoreJsonVariables {
		value := GetJsonValueFromPath(jsonStr, getJsonVariable.Path)
		if value != "" {
			config.StoredVariables = append(
				config.StoredVariables,
				KeyValue{Key: getJsonVariable.Key, Value: value},
			)
		}
	}
}

func PrettyPrintJson(body []byte) {
	var objmap interface{}
	json.Unmarshal(body, &objmap)
	f := colorjson.NewFormatter()
	f.Indent = 2
	s, _ := f.Marshal(objmap)
	fmt.Println(string(s))
}

func RunRequest(cmd *cobra.Command, requestName string, config *Config) {
	errors := 0
	found := false
	println("😺 Running request " + requestName)
	for _, request := range config.Requests {
		if request.Name == requestName {
			found = true
			request = config.replaceVariables(
				request,
				config.Environments[0],
				config.StoredVariables,
			)

			println("🚀 " + request.Method + " " + request.URL)

			req, err := http.NewRequest(
				request.Method,
				request.URL,
				strings.NewReader(request.Body),
			)
			if err != nil {
				println(err)
				os.Exit(1)
			}
			for _, header := range request.Headers {
				req.Header.Set(header.Key, header.Value)
				//println(AnsiColor(header.Key, 53, 177, 226) + ": " + header.Value)
			}
			start := time.Now()
			resp, err := http.DefaultClient.Do(req)
			elapsed := time.Since(start)
			if err != nil {
				println(err)
				os.Exit(1)
			}
			println("🕒 Request took " + elapsed.String())
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				println(err)
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
			json := false
			for key, value := range resp.Header {
				println(AnsiColor(key, 53, 177, 226) + ": " + strings.Join(value, ", "))
				if key == "Content-Type" && strings.Contains(value[0], "application/json") {
					json = true
				}
			}
			println("")
			if json {
				PrettyPrintJson(body)
				GetJsonValues(string(body), request, config)
				println("")
			} else {
				println(string(body) + "\n")
			}

			if len(config.StoredVariables) > 0 {
				println("🔑 Stored variables:")
				for _, variable := range config.StoredVariables {
					println("  " + AnsiColor(variable.Key, 53, 177, 226) + ": " + variable.Value)
				}
				println("")
			}

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
	if !found {
		println("😿 Request " + requestName + " not found")
		os.Exit(1)
	}
	if errors > 0 {
		println("😿 There were errors in the request")
		os.Exit(1)
	} else {
		println("😻 Request ran successfully")
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(runAllCmd)
}

var runCmd = &cobra.Command{
	Use:   "run <request(s)>",
	Short: "Run one or more requests",
	Long:  `Run one or more requests separated by comma ,`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfig(cmd.Flag("config").Value.String())
		if strings.Contains(args[0], ",") {
			requests := strings.Split(args[0], ",")
			for i, request := range requests {
				if i > 0 {
					println("----------------------------------------")
				}
				RunRequest(cmd, request, &config)
			}
		} else {
			RunRequest(cmd, args[0], &config)
		}
	},
}

var runAllCmd = &cobra.Command{
	Use:   "runall",
	Short: "Run all requests",
	Long:  `Run all requests`,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfig(cmd.Flag("config").Value.String())
		for i, request := range config.Requests {
			if i > 0 {
				println("----------------------------------------")
			}
			RunRequest(cmd, request.Name, &config)
		}

	},
}

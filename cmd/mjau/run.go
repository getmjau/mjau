package mjau

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
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
	Commands           []Command           `yaml:"commands"`
	Asserts            []Assert            `yaml:"asserts"`
}

type Command struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	Variable    string `yaml:"variable"`
	Value       string `yaml:"value"`
}

type StoreJsonVariable struct {
	Path string `yaml:"path"`
	Key  string `yaml:"key"`
}

type Assert struct {
	Description string `yaml:"description"`
	Key         string `yaml:"key"`
	Variable    string `yaml:"variable"`
	Comparison  string `yaml:"comparison"`
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
		fmt.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return config
}

func (config *Config) ClearRequestResponseVariables() {
	for _, request := range config.StoredVariables {
		if strings.HasPrefix(request.Key, "request.") ||
			strings.HasPrefix(request.Key, "response.") {
			config.RemoveStoredVariable(request.Key)
		}
	}
}

func (config *Config) LoadEnvironment(name string) {
	for _, environment := range config.Environments {
		if environment.Name == name {
			config.StoreVariable("environment.name", environment.Name)
			for _, variable := range environment.Variables {
				config.StoreVariable("environment."+variable.Key, variable.Value)
			}
		}
	}
}

func (config *Config) replaceVariables(
	request Request,
	environment Environment,
) Request {
	// Replace variables in request with values from environment
	config.StoreVariable("env.name", environment.Name)
	for _, variable := range environment.Variables {
		request.URL = strings.ReplaceAll(request.URL, "{{"+variable.Key+"}}", variable.Value)
		request.Body = strings.ReplaceAll(request.Body, "{{"+variable.Key+"}}", variable.Value)
		for i, header := range request.Headers {
			header.Value = strings.ReplaceAll(header.Value, "{{"+variable.Key+"}}", variable.Value)
			request.Headers[i] = header
		}
		config.StoreVariable("env."+variable.Key, variable.Value)
	}
	return request
}

func (config *Config) StoreVariable(key, value string) {
	for i, variable := range config.StoredVariables {
		if variable.Key == key {
			config.StoredVariables[i].Value = value
			return
		}
	}
	config.StoredVariables = append(
		config.StoredVariables,
		KeyValue{Key: key, Value: value},
	)
}

func (config *Config) GetVariable(key string) string {
	for _, variable := range config.StoredVariables {
		if variable.Key == key {
			return variable.Value
		}
	}
	return ""
}

func (config *Config) InsertVariables(str string) string {
	for _, variable := range config.StoredVariables {
		str = strings.ReplaceAll(str, "{{"+variable.Key+"}}", variable.Value)
	}
	return str
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

func Compare(a, b string, comparison string) bool {
	switch comparison {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "contains":
		return strings.Contains(a, b)
	default:
		return false
	}
}

func (config *Config) RemoveStoredVariable(key string) {
	for i, variable := range config.StoredVariables {
		if variable.Key == key {
			config.StoredVariables = append(
				config.StoredVariables[:i],
				config.StoredVariables[i+1:]...)
		}
	}
}

func RunRequest(cmd *cobra.Command, requestName string, config *Config) {
	errors := 0
	found := false
	fmt.Println("😺 Running request " + requestName)
	config.LoadEnvironment(cmd.Flag("env").Value.String())
	config.ClearRequestResponseVariables()
	for _, request := range config.Requests {
		if request.Name == requestName {
			found = true
			request.URL = config.InsertVariables(request.URL)
			request.Body = config.InsertVariables(request.Body)
			for _, header := range request.Headers {
				header.Value = config.InsertVariables(header.Value)
			}

			fmt.Println("🚀 " + request.Method + " " + request.URL)

			config.StoreVariable("request.name", request.Name)
			config.StoreVariable("request.method", request.Method)
			config.StoreVariable("request.url", request.URL)
			config.StoreVariable("request.body", request.Body)

			req, err := http.NewRequest(
				request.Method,
				request.URL,
				strings.NewReader(request.Body),
			)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			userAgentFound := false
			for _, header := range request.Headers {
				req.Header.Set(header.Key, header.Value)
				if cmd.Flag("full-request").Value.String() == "true" {
					fmt.Println(AnsiColor(header.Key, 53, 177, 226) + ": " + header.Value)
				}
				if header.Key == "User-Agent" {
					userAgentFound = true
				}
			}
			if !userAgentFound {
				req.Header.Set(
					"User-Agent",
					"mjau/"+Version+" ("+runtime.GOOS+"; "+runtime.GOARCH+")",
				)
				if cmd.Flag("full-request").Value.String() == "true" {
					fmt.Println(
						AnsiColor(
							"User-Agent",
							53,
							177,
							226,
						) + ": mjau/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")",
					)
				}
			}
			for _, header := range request.Headers {
				config.StoreVariable("request.headers."+header.Key, header.Value)
			}

			if cmd.Flag("full-request").Value.String() == "true" {
				fmt.Println("\n" + request.Body)
			}
			start := time.Now()
			resp, err := http.DefaultClient.Do(req)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("🕒 Request took " + elapsed.String())
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
			}

			config.StoreVariable("response.status_code", strconv.Itoa(resp.StatusCode))
			config.StoreVariable("response.body", string(body))
			config.StoreVariable("response.elapsed", elapsed.String())
			config.StoreVariable("response.status_text", http.StatusText(resp.StatusCode))

			fmt.Println(
				AnsiColor("HTTP/1.1 ", 53, 42, 226)+
					AnsiColor(strconv.Itoa(
						resp.StatusCode), 53, 42, 226),
				http.StatusText(
					resp.StatusCode,
				),
			)
			json := false
			for key, value := range resp.Header {
				fmt.Println(AnsiColor(key, 53, 177, 226) + ": " + strings.Join(value, ", "))
				if key == "Content-Type" && strings.Contains(value[0], "application/json") {
					json = true
				}
				config.StoreVariable("response.headers."+key, strings.Join(value, ", "))
			}
			fmt.Println("")
			if json {
				PrettyPrintJson(body)
				GetJsonValues(string(body), request, config)
				fmt.Println("")
			} else {
				fmt.Println(string(body) + "\n")
			}

			if len(request.Commands) > 0 {
				fmt.Println("🔧 Commands:")
			}
			for _, command := range request.Commands {
				if command.Command == "echo" {
					fmt.Println("  " + config.InsertVariables(command.Value))
				}
				if command.Command == "add_variable" {
					config.StoreVariable(command.Variable, command.Value)
				}
			}

			if len(config.StoredVariables) > 0 {
				fmt.Println("🔑 Stored variables:")
				for _, variable := range config.StoredVariables {
					if strings.HasPrefix(variable.Key, "environment.") {
						fmt.Println(
							"  " + AnsiColor(variable.Key, 53, 177, 226) + ": " + variable.Value,
						)
					}
				}
				for _, variable := range config.StoredVariables {
					if strings.HasPrefix(variable.Key, "request.") {
						fmt.Println(
							"  " + AnsiColor(variable.Key, 53, 177, 226) + ": " + variable.Value,
						)
					}
				}
				for _, variable := range config.StoredVariables {
					if strings.HasPrefix(variable.Key, "response.") {
						fmt.Println(
							"  " + AnsiColor(variable.Key, 53, 177, 226) + ": " + variable.Value,
						)
					}
				}
				for _, variable := range config.StoredVariables {
					if !strings.HasPrefix(variable.Key, "response.") &&
						!strings.HasPrefix(variable.Key, "request.") &&
						!strings.HasPrefix(variable.Key, "environment.") {
						fmt.Println(
							"  " + AnsiColor(variable.Key, 53, 177, 226) + ": " + variable.Value,
						)
					}
				}

				fmt.Println("")
			}

			if len(request.Asserts) > 0 {
				fmt.Println("👀 Asserts:")
			}
			for _, assert := range request.Asserts {
				if !Compare(config.GetVariable(assert.Variable), assert.Value, assert.Comparison) {
					//if assert.Value != config.GetVariable(assert.Variable) {
					errors++
					fmt.Printf(
						"  ❌ %s does not match. Expected: %s %s %s\n",
						assert.Description,
						config.GetVariable(assert.Variable),
						assert.Comparison,
						assert.Value,
					)
				} else {
					fmt.Printf(
						"  ✅ %s matches\n",
						assert.Description,
					)
				}
			}
			// for _, assert := range request.Asserts {
			// 	fmt.Println("" + AnsiColor(assert.Description, 53, 177, 226))
			// 	if assert.Key == "status_code" {
			// 		if strconv.Itoa(resp.StatusCode) != assert.Value {
			// 			errors++
			// 			fmt.Printf(
			// 				"  ❌ Status code does not match. Expected: %s, got: %d\n",
			// 				assert.Value,
			// 				resp.StatusCode,
			// 			)
			// 		} else {
			// 			fmt.Println("  ✅ Status code matches")
			// 		}
			// 	}
			// 	if assert.Key == "body" {
			// 		if string(body) != assert.Value {
			// 			errors++
			// 			fmt.Printf(
			// 				"  ❌ Body does not match. Expected: '%s', got: '%s'\n",
			// 				assert.Value,
			// 				string(body),
			// 			)
			// 		} else {
			// 			fmt.Println("  ✅ Body matches")
			// 		}
			// 	}
			// }

		}
	}
	if !found {
		fmt.Println("😿 Request " + requestName + " not found")
		os.Exit(1)
	}
	if errors > 0 {
		fmt.Println("😿 There were errors in the request")
		os.Exit(1)
	} else {
		fmt.Println("😻 Request ran successfully")
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
					fmt.Println("----------------------------------------")
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
				fmt.Println("----------------------------------------")
			}
			RunRequest(cmd, request.Name, &config)
		}

	},
}

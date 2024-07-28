package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const SYSTEM_PROMPT = `
You are an AI programming assistant.
When asked for your name, you must respond with "GitHub Copilot".
Follow the user's requirements carefully & to the letter.
Follow Microsoft content policies.
Avoid content that violates copyrights.
If you are asked to generate content that is harmful, hateful, racist, sexist, lewd, violent, or completely irrelevant to software engineering, only respond with "Sorry, I can't assist with that."
Keep your answers short and impersonal.
You can answer general programming questions and perform the following tasks:
* Ask a question about the files in your current workspace
* Explain how the code in your active editor works
* Generate unit tests for the selected code
* Propose a fix for the problems in the selected code
* Scaffold code for a new workspace
* Create a new Jupyter Notebook
* Find relevant code to your query
* Propose a fix for the a test failure
* Ask questions about Neovim
* Generate query parameters for workspace search
* Ask how to do something in the terminal
* Explain what just happened in the terminal
You use the GPT-4 version of OpenAI's GPT models.
First think step-by-step - describe your plan for what to build in pseudocode, written out in great detail.
Then output the code in a single code block. This code block should not contain line numbers (line numbers are not necessary for the code to be understood, they are in format number: at beginning of lines).
Minimize any other prose.
Use Markdown formatting in your answers.
Make sure to include the programming language name at the start of the Markdown code blocks.
Avoid wrapping the whole response in triple backticks.
The user works in an IDE called Neovim which has a concept for editors with open files, integrated unit test support, an output pane that shows the output of running the code as well as an integrated terminal.
The user is working on a %s machine. Please respond with system specific commands if applicable.
The active document is the source code the user is looking at right now.
You can only give one reply for each conversation turn.
`

const COPILOT_TOKEN_API = "https://api.github.com/copilot_internal/v2/token"
const COPILOT_COMPLETION_API = "https://api.githubcopilot.com/chat/completions"

type TokenResponse struct {
	Token string `json:"token"`
}

type GithubCopilotConfigFile struct {
	GitHubCom struct {
		User string `json:"user"`

		OAuthToken string `json:"oauth_token"`
	} `json:"github.com"`
}

type Message struct {
	Choices []struct {
		FinishReason         string `json:"finish_reason"`
		Index                int    `json:"index"`
		ContentFilterOffsets struct {
			CheckOffset int `json:"check_offset"`
			StartOffset int `json:"start_offset"`
			EndOffset   int `json:"end_offset"`
		} `json:"content_filter_offsets"`
		ContentFilterResults struct {
			Hate struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"hate"`
			SelfHarm struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"self_harm"`
			Sexual struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"sexual"`
			Violence struct {
				Filtered bool   `json:"filtered"`
				Severity string `json:"severity"`
			} `json:"violence"`
		} `json:"content_filter_results"`
		Delta struct {
			Content interface{} `json:"content"`
			Role    interface{} `json:"role"`
		} `json:"delta"`
	} `json:"choices"`
	Created int64  `json:"created"`
	ID      string `json:"id"`
}

type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Param   string `json:"param"`
	Type    string `json:"type"`
}

type CopilotRequest struct {
	Token     string
	SessionId string
	UUID      string
	MachineID string
}

type Request struct {
	Intent      bool             `json:"intent"`
	Model       string           `json:"model"`
	N           int              `json:"n"`
	Stream      bool             `json:"stream"`
	Temperature float32          `json:"temperature"`
	TopP        int              `json:"top_p"`
	Messages    []HistoryMessage `json:"messages"`
	History     []HistoryMessage `json:"history"`
	Maxtokens   int              `json:"max_tokens"`
}

func generateAskRequest(history []HistoryMessage) (Request, error) {
	req := Request{
		Intent:      true,
		Model:       "gpt-3.5-turbo",
		N:           1,
		Stream:      true,
		Temperature: 0.1,
		TopP:        1,
		Messages:    history,
		Maxtokens:   4096,
	}

	return req, nil
}

func readConfig() GithubCopilotConfigFile {
	filePath := os.Getenv("HOME") + "/.config/github-copilot/hosts.json"

	file, error := os.Open(filePath)

	defer file.Close()

	if error != nil {
		panic(error)
	}

	content, err := io.ReadAll(file)

	var config GithubCopilotConfigFile

	err = json.Unmarshal(content, &config)

	if err != nil {
		panic(err)
	}

	return config
}

func getToken() string {
	req, err := http.NewRequest("GET", COPILOT_TOKEN_API, nil)

	if err != nil {
		panic(err)
	}

	config := readConfig()

	req.Header.Set("authorization", "token "+config.GitHubCom.OAuthToken)
	req.Header.Set("accept", "application/json")
	req.Header.Set("editor-version", "vscode/1.85.1")
	req.Header.Set("editor-plugin-version", "copilot-chat/0.12.2023120701")
	req.Header.Set("user-agent", "GitHubCopilotChat/0.12.2023120701")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	var tokenResponse TokenResponse

	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)

	if err != nil {
		panic(err)
	}

	return tokenResponse.Token
}

func generateCopilotRequest() CopilotRequest {
	return CopilotRequest{
		Token:     getToken(),
		SessionId: sessionID(),
		UUID:      uuid(),
		MachineID: machineID(),
	}
}

func getResponse(m *model, callback func(string, bool, bool)) string {
	request, _ := generateAskRequest(m.history)
	body, err := json.Marshal(request)

	log.Println("History:", m.history[1:])

	if err != nil {
		panic(err)
	}

	renewToken(m)

	bodyBuffer := bytes.NewBuffer(body)

	req, err := http.NewRequest("POST", COPILOT_COMPLETION_API, bodyBuffer)

	if err != nil {
		panic(err)
	}

	req.Header.Set("authorization", "Bearer "+m.copilotRequest.Token)
	req.Header.Set("vscode-sessionid", m.copilotRequest.SessionId)
	req.Header.Set("x-request-id", m.copilotRequest.UUID)
	req.Header.Set("vscode-machineid", m.copilotRequest.MachineID)

	req.Header.Set("content-type", "application/json")
	req.Header.Set("openai-intent", "conversation-panel")
	req.Header.Set("openai-organization", "github-copilot")
	req.Header.Set("user-agent", "GitHubCopilotChat/0.14.2024032901")
	req.Header.Set("editor-version", "vscode/1.88.0")
	req.Header.Set("editor-plugin-version", "copilot-chat/0.14.2024032901")
	req.Header.Set("x-github-api-version", "2023-07-07")
	req.Header.Set("copilot-integration-id", "vscode-chat")

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip,deflate,br")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	return parseResponse(resp.Body, callback)
}

func parseResponse(s io.ReadCloser, callback func(string, bool, bool)) string {
	dec := bufio.NewReader(s)
	isError := false

	reply := make([]byte, 0)

	for {
		content, err := dec.ReadBytes('\n')

		if err != nil {
			log.Println("Response was not nil:", err)

			break
		}

		s := strings.Trim(string(content), " \n\t")

		log.Println("Content:", s)

		if strings.HasPrefix(s, `{"error":`) {
			var error ErrorResponse

			json.Unmarshal([]byte(s), &error)

			reply = []byte(error.Error.Message)
			isError = true

			break
		}

		if strings.HasPrefix(s, "[DONE]") {
			break
		}

		if !strings.HasPrefix(s, "data:") {
			continue
		}

		jsonExtract := removeUntilData(string(s))

		var message Message

		json.Unmarshal([]byte(jsonExtract), &message)

		if len(message.Choices) > 0 {
			if message.Choices[0].Delta.Content != nil {
				txt := message.Choices[0].Delta.Content.(string)
				reply = append(reply, []byte(txt)...)

				callback(string(reply), false, isError)
			}
		}
	}

	callback(string(reply), true, isError)

	log.Println("Reply:", string(reply))

	return string(reply)
}

func removeUntilData(s string) string {
	index := strings.Index(s, "data:")

	if index == -1 {
		return s
	}

	return s[index+len("data: "):]
}

func extractExpiration(s string) int64 {
	exp := "0"

	pairs := strings.Split(s, ";")

	for _, pair := range pairs {
		if strings.HasPrefix(pair, "exp=") {
			exp = strings.Split(pair, "=")[1]

			break
		}
	}

	timestamp, err := strconv.ParseInt(exp, 10, 64)

	if err != nil {
		log.Println("Failed to parse timestamp:", err)

		return 0
	}

	return timestamp
}

func isExpired(t int64) bool {
	return t+60 < time.Now().Unix()
}

func renewToken(m *model) {
	if isExpired(extractExpiration(m.copilotRequest.Token)) {
		log.Println("Renewing expired token")

		m.copilotRequest.Token = getToken()
	}
}

/* NOTE: the following functions have been ported from Lua using Copilot.
 *       original source:
 *       https://github.com/CopilotC-Nvim/CopilotChat.nvim/blob/9898b4cd1b19c6ca639b77b34bb599a119356c1f/lua/CopilotChat/copilot.lua
 */

func uuid() string {
	rand.NewSource(time.Now().UnixNano())
	template := "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
	uuid := ""

	for _, c := range template {
		if c == 'x' {
			uuid += fmt.Sprintf("%x", rand.Intn(16))
		} else if c == 'y' {
			uuid += fmt.Sprintf("%x", rand.Intn(4)+8)
		} else {
			uuid += string(c)
		}
	}

	return uuid
}

func machineID() string {
	length := 65
	hexChars := "0123456789abcdef"
	hex := ""
	rand.NewSource(time.Now().UnixNano())

	for i := 0; i < length; i++ {
		hex += string(hexChars[rand.Intn(len(hexChars))])
	}

	return hex
}

func sessionID() string {
	return uuid() + fmt.Sprint(time.Now().UnixNano()/int64(time.Millisecond))
}

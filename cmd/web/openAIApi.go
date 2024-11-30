package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type WhisperApiResponse struct {
	Text string `json:"text"`
}

type WhisperApiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletion struct {
	Model          string         `json:"model"`
	Messages       []ChatMessage  `json:"messages"`
	ResponseFormat ResponseFormat `json:"response_format"`
}

type PropertyDefinition struct {
	Description string              `json:"description"`
	Type        string              `json:"type"`
	Items       *PropertyDefinition `json:"items,omitempty"`
}

type Properties struct {
	LogicalParagraphs PropertyDefinition `json:"logical_paragraphs"`
	Summary           PropertyDefinition `json:"summary"`
	ActionItems       PropertyDefinition `json:"action_items"`
}

type Schema struct {
	Type                 string     `json:"type"`
	Properties           Properties `json:"properties"`
	AdditionalProperties bool       `json:"additionalProperties"`
}

type JsonSchema struct {
	Name   string `json:"name"`
	Schema Schema `json:"schema"`
}

type ResponseFormat struct {
	Type       string     `json:"type"`
	JsonSchema JsonSchema `json:"json_schema"`
}

type ChatResponse struct {
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func doOpenAIRequest(endpoint string, payload io.Reader, method string, contentType string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, "https://api.openai.com/v1/"+endpoint, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (app *application) sendTranscriptionToWhisper(uploadedFilePath string, filename string) (string, error) {
	if app.config.mockOpenAI {
		b, err := os.ReadFile("./mocks/completed-transcription.txt")
		if err != nil {
			app.logger.Error(err.Error())
			return "", err
		}
		return string(b), nil
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	fileBytes, err := readFromExternalStorage(uploadedFilePath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, &fileBytes)
	if err != nil {
		return "", err
	}

	err = writer.WriteField("model", "whisper-1")
	if err != nil {
		return "", err
	}
	writer.Close() // don't defer this

	resp, err := doOpenAIRequest("audio/transcriptions", body, "POST", writer.FormDataContentType())
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var whisperError WhisperApiError
		err = json.Unmarshal(b, &whisperError)
		if err != nil {
			return "", err
		}
		return "", errors.New(whisperError.Error.Message)
	}

	var whisperResponse WhisperApiResponse
	err = json.Unmarshal(b, &whisperResponse)
	if err != nil {
		return "", err
	}

	err = os.WriteFile("./mocks/completed-transcription.txt", []byte(whisperResponse.Text), 0644)
	if err != nil {
		return "", err
	}

	return whisperResponse.Text, nil
}

func (app *application) formatAndSummarizeTranscription(transcribedText string) (string, error) {
	if app.config.mockOpenAI {
		b, err := os.ReadFile("./mocks/completed-summary.json")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	chatCompletion := &ChatCompletion{
		Model: "gpt-4o-mini",
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are an assistant who's job is to take an audio transcription and first break up the text into logical paragraphs. Each paragraph needs to be under 2000 characters. Finally, you will create a summary of the transcription.",
			},
			{
				Role:    "user",
				Content: transcribedText,
			},
		},
		ResponseFormat: ResponseFormat{
			Type: "json_schema",
			JsonSchema: JsonSchema{
				Name: "response_schema",
				Schema: Schema{
					Type: "object",
					Properties: Properties{
						LogicalParagraphs: PropertyDefinition{
							Description: "The logical paragraphs of the transcribed audio",
							Type:        "string",
						},
						Summary: PropertyDefinition{
							Description: "The summary of the transcribed audio",
							Type:        "string",
						},
					},
					AdditionalProperties: false,
				},
			},
		},
	}

	marshalled, err := json.Marshal(chatCompletion)
	if err != nil {
		return "", err
	}

	resp, err := doOpenAIRequest("chat/completions", bytes.NewReader(marshalled), "POST", "application/json")

	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	err = os.WriteFile("./mocks/completed-summary.txt", b, 0644)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

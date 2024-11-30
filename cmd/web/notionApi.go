package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type Parent struct {
	Type       string `json:"type"`
	DatabaseId string `json:"database_id"`
}

type Text struct {
	Content string `json:"content"`
}

type RichText struct {
	Text Text `json:"text"`
}

type Block struct {
	RichText []RichText `json:"rich_text"`
}

type Title struct {
	Text Text `json:"text"`
}

type Name struct {
	Title []Title `json:"title"`
}

type Property struct {
	Name Name `json:"Name"`
}

type Children struct {
	Object    string `json:"object"`
	Paragraph *Block `json:"paragraph,omitempty"`
	Heading2  *Block `json:"heading_2,omitempty"`
}

type NotionPage struct {
	Parent     Parent     `json:"parent"`
	Properties Property   `json:"properties"`
	Children   []Children `json:"children"`
}

type NotionApiError struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"validation_error"`
	Message string `json:"message"`
}

type TokenRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	RedirectUri string `json:"redirect_uri"`
}

type NotionOwner struct {
	Type string     `json:"type"`
	User NotionUser `json:"user"`
}

type NotionUser struct {
	Object    string `json:"object"`
	Id        string `json:"id"`
	Name      string `json:"name"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Type      string `json:"type"`
	Person    struct {
		Email string `json:"email"`
	} `json:"person"`
}

type TokenResponse struct {
	AccessToken          string      `json:"access_token"`
	TokenType            string      `json:"token_type"`
	BotId                string      `json:"bot_id"`
	WorkspaceName        string      `json:"workspace_name"`
	WorkspaceIcon        string      `json:"workspace_icon,omitempty"`
	WorkspaceId          string      `json:"workspace_id"`
	Owner                NotionOwner `json:"owner"`
	DuplicatedTemplateId string      `json:"duplicated_template_id,omitempty"`
	RequestId            string      `json:"request_id"`
}

type Filter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}

type SearchRequestBody struct {
	Query  string  `json:"query,omitempty"`
	Filter *Filter `json:"filter,omitempty"`
}

type Icon struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

type NotionResult struct {
	Id    string  `json:"id"`
	Title []Title `json:"title"`
	Icon  Icon    `json:"icon"`
}

type SearchResponseBody struct {
	Results []NotionResult `json:"results"`
}

type ResponseSchemaForNotion struct {
	LogicalParagraphs string   `json:"logical_paragraphs"`
	Summary           string   `json:"summary"`
	ActionItems       []string `json:"action_items"`
}

func generateAuthHeader(authType string, credentials string) string {
	switch authType {
	case "basic":
		return "Basic " + credentials
	default:
		return "Bearer " + credentials
	}
}

func doNotionApiRequest(endpoint string, payload []byte, auth string, method string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, "https://api.notion.com/v1/"+endpoint, bytes.NewReader(payload))

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Notion-Version", "2022-06-28")
	req.Header.Add("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func searchSharedDatabases(notionAccessToken string) ([]NotionResult, error) {
	searchRequest := &SearchRequestBody{
		Filter: &Filter{
			Value:    "database",
			Property: "object",
		},
	}

	marshalled, err := json.Marshal(searchRequest)
	if err != nil {
		return []NotionResult{}, err
	}

	resp, err := doNotionApiRequest("search", marshalled, generateAuthHeader("bearer", notionAccessToken), "POST")
	if err != nil {
		return []NotionResult{}, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []NotionResult{}, err
	}
	defer resp.Body.Close()

	var searchResponse = &SearchResponseBody{}

	err = json.Unmarshal(b, searchResponse)
	if err != nil {
		return []NotionResult{}, err
	}

	return searchResponse.Results, nil
}

func (app *application) createNotionPage(fileName string, chatResponseString string, notionPageId string, notionAccessToken string) error {
	paragraphs, err := mapChatResponseToNotionPage(chatResponseString)
	if err != nil {
		return err
	}

	newNotionPage := &NotionPage{
		Parent: Parent{
			Type:       "database_id",
			DatabaseId: notionPageId,
		},
		Properties: Property{
			Name: Name{
				Title: []Title{
					{
						Text: Text{
							Content: fileName + " Transcribed Audio",
						},
					},
				},
			},
		},
		Children: paragraphs,
	}

	marshalled, err := json.Marshal(newNotionPage)
	if err != nil {
		return err
	}

	resp, err := doNotionApiRequest("pages", marshalled, generateAuthHeader("bearer", notionAccessToken), "POST")
	if err != nil {
		return err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var notionError NotionApiError
		err = json.Unmarshal(b, &notionError)
		if err != nil {
			return err
		}
		return errors.New(notionError.Message)
	}

	return nil
}

func createParagraphElement(content string) Children {
	return Children{
		Object: "block",
		Paragraph: &Block{
			RichText: []RichText{
				{
					Text{
						Content: content,
					},
				},
			},
		},
	}
}

func createHeading2Element(content string) Children {
	return Children{
		Object: "block",
		Heading2: &Block{
			RichText: []RichText{
				{
					Text{
						Content: content,
					},
				},
			},
		},
	}
}

func mapChatResponseToNotionPage(chatResponseString string) ([]Children, error) {
	paragraphs := []Children{}
	chatResponse := ChatResponse{}

	err := json.Unmarshal([]byte(chatResponseString), &chatResponse)
	if err != nil {
		return paragraphs, err
	}

	responseSchemaForNotion := ResponseSchemaForNotion{}
	err = json.Unmarshal([]byte(chatResponse.Choices[0].Message.Content), &responseSchemaForNotion)
	if err != nil {
		return paragraphs, err
	}

	paragraphs = append(paragraphs,
		createHeading2Element("Transcription"),
	)

	splitParagraphs := strings.Split(responseSchemaForNotion.LogicalParagraphs, "\n\n")

	for _, splitStr := range splitParagraphs {
		paragraphs = append(paragraphs, createParagraphElement(splitStr))
	}

	paragraphs = append(paragraphs,
		createHeading2Element("Summary"),
		createParagraphElement(responseSchemaForNotion.Summary),
	)

	return paragraphs, nil
}

func getBotDataFromToken(notionAccessToken string) error {
	resp, err := doNotionApiRequest("users/me", []byte{}, generateAuthHeader("bearer", notionAccessToken), "GET")
	if err != nil {
		return err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var notionError NotionApiError
		err = json.Unmarshal(b, &notionError)
		if err != nil {
			return err
		}
		return errors.New(notionError.Message)
	}

	return nil
}

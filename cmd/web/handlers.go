package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"
)

func (app *application) renderHomepage(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/pages/home.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func clearCookie(cookieName string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	notionAccessToken, err := r.Cookie("notion_token")

	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			app.renderHomepage(w, r)
			return
		} else {
			app.serverError(w, r, err)
			return
		}
	}

	err = getBotDataFromToken(notionAccessToken.Value)
	if err != nil {
		http.SetCookie(w, clearCookie("notion_token"))

		app.renderHomepage(w, r)
	} else {
		http.Redirect(w, r, "/upload", http.StatusSeeOther)
	}
}

type TemplateData struct {
	NotionPages []NotionResult
}

func (app *application) uploadForm(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/pages/upload.tmpl",
	}

	cookie, err := r.Cookie("notion_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		app.serverError(w, r, err)
		return
	}

	results, err := searchSharedDatabases(cookie.Value)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", &TemplateData{
		NotionPages: results,
	})
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) uploadSuccessful(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/pages/transcribe-complete.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) transcribeAndPushToNotionPage(savedPath string, filename string, notionPageId string, notionAccessToken string) {
	var transcribedText string

	transcribedText, err := app.sendTranscriptionToWhisper(savedPath, filename)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Debug("Whisper transcription completed")

	chatResponse, err := app.formatAndSummarizeTranscription(transcribedText)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Debug("Summary completed")

	err = app.createNotionPage(filename, chatResponse, notionPageId, notionAccessToken)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Debug("Notion page created")
}

func isValidAudioFile(contentType string) bool {
	validFileTypes := []string{"audio/mpeg", "video/mp4", "video/mpeg"}

	return slices.Contains(validFileTypes, contentType)
}

func (app *application) createTranscription(w http.ResponseWriter, r *http.Request) {
	notionAccessTokenCookie, err := r.Cookie("notion_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			app.clientError(w, http.StatusUnauthorized)
			return
		}
		app.serverError(w, r, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 25*1024*1024)

	err = r.ParseMultipartForm(25 * 1024 * 1024)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	notionPageId := r.FormValue("notion-page-id")
	if notionPageId == "" {
		app.serverError(w, r, errors.New("no Notion page ID supplied"))
		return
	}

	uploadedFile, handler, err := r.FormFile("audio-file")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer uploadedFile.Close()

	uploadedBytes, err := io.ReadAll(uploadedFile)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	contentType := http.DetectContentType(uploadedBytes)
	if !isValidAudioFile(contentType) {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	savedPath, err := writeToExternalStorage(uploadedBytes, handler.Filename, contentType)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go app.transcribeAndPushToNotionPage(savedPath, handler.Filename, notionPageId, notionAccessTokenCookie.Value)

	http.Redirect(w, r, "/upload/success", http.StatusSeeOther)
}

func (app *application) notionAuthCallback(w http.ResponseWriter, r *http.Request) {
	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	errorParam := params.Get("error")
	if errorParam == "access_denied" {
		w.Write([]byte("Client Error"))
		return
	}

	code := params.Get("code")
	if code == "" {
		w.Write([]byte("Client Error: No code"))
		return
	}

	tokenRequest := &TokenRequest{
		GrantType:   "authorization_code",
		Code:        code,
		RedirectUri: app.config.appUri + "/auth/callback",
	}

	marshalled, err := json.Marshal(tokenRequest)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	encodedBasicCredentials := generateAuthHeader("basic", base64.StdEncoding.EncodeToString([]byte(os.Getenv("NOTION_CLIENT_ID")+":"+os.Getenv("NOTION_CLIENT_SECRET"))))
	resp, err := doNotionApiRequest("oauth/token", marshalled, encodedBasicCredentials, "POST")

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer resp.Body.Close()

	var tokenResponse = &TokenResponse{}

	err = json.Unmarshal(b, tokenResponse)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if tokenResponse.AccessToken == "" {
		app.serverError(w, r, errors.New("could not get access token"))
		return
	}

	notionAccessToken := http.Cookie{
		Name:     "notion_token",
		Value:    tokenResponse.AccessToken,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &notionAccessToken)
	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

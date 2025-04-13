package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

type UserData struct {
	Name string
}

func main() {
	godotenv.Load()
	e := echo.New()
	e.Renderer = newTemplate()

	e.GET("/", func(c echo.Context) error {
		cookie, err := c.Cookie("access_token")
		if err != nil {
			return c.Render(http.StatusOK, "index", nil)
		}

		// Call user info API with the access token as Bearer token
		userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		if err != nil {
			return err
		}
		userReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", cookie.Value))

		userResp, err := http.DefaultClient.Do(userReq)
		if err != nil {
			return err
		}
		defer userResp.Body.Close()

		var userInfo map[string]interface{}
		if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
			return err
		}

		userData := UserData{
			Name: userInfo["given_name"].(string),
		}

		return c.Render(http.StatusOK, "index", userData)
	})

	e.GET("/oauth/authorize", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, fmt.Sprintf("https://accounts.google.com/o/oauth2/auth?client_id=%v&redirect_uri=%v&scope=https://www.googleapis.com/auth/userinfo.profile&response_type=code&access_type=offline", os.Getenv("CLIENT_ID"), os.Getenv("REDIRECT_URI")))
	})

	e.GET("/oauth/callback", func(c echo.Context) error {
		query := c.QueryParams()
		code := query["code"]

		// Prepare token request
		requestData := url.Values{
			"code":          {code[0]},
			"client_id":     {os.Getenv("CLIENT_ID")},
			"client_secret": {os.Getenv("CLIENT_SECRET")},
			"redirect_uri":  {os.Getenv("REDIRECT_URI")},
			"grant_type":    {"authorization_code"},
		}

		req, err := http.NewRequest("POST", "https://www.googleapis.com/oauth2/v4/token", strings.NewReader(requestData.Encode()))
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		// Execute token request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		// Call user info API with the access token as Bearer token
		userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		if err != nil {
			return err
		}
		userReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", result["access_token"]))

		userResp, err := client.Do(userReq)
		if err != nil {
			return err
		}
		defer userResp.Body.Close()

		var userInfo map[string]interface{}
		if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
			return err
		}

		// Set cookie with proper Path so it is available across the site
		cookie := &http.Cookie{
			Name:    "access_token",
			Value:   result["access_token"].(string),
			Path:    "/",
			Expires: time.Now().Add(365 * 24 * time.Hour),
			Secure:  true,
		}
		//c.SetCookie(cookie)
		c.Response().Header().Set("Set-Cookie", cookie.String())

		return c.Redirect(http.StatusFound, "/")
	})
	e.POST("/logout", func(c echo.Context) error {
		cookie := &http.Cookie{
			Name:    "access_token",
			Value:   "",
			Path:    "/",
			Expires: time.Now().Add(-1 * time.Hour),
			Secure:  true,
		}
		c.SetCookie(cookie)
		return c.Redirect(http.StatusFound, "/")
	})

	e.Logger.Fatal(e.Start(":42069"))
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	//"github.com/labstack/echo/v4/middleware"
	"html/template"
	"io"

	"github.com/joho/godotenv"
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

func main() {
	godotenv.Load()
	e := echo.New()
	//e.Use(middleware.Logger())
	e.Renderer = newTemplate()
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})
	e.GET("/oauth/callback", func(c echo.Context) error {
		query := c.QueryParams()
		code := query["code"]
		///https://www.googleapis.com/oauth2/v4/token
		requestData := url.Values{
			"code":          {code[0]},
			"client_id":     {os.Getenv("CLIENT_ID")},
			"client_secret": {os.Getenv("CLIENT_SECRET")},
			"redirect_uri":  {os.Getenv("REDIRECT_URI")},
			"grant_type":    {"authorization_code"},
		}
		req, err := http.NewRequest("POST", "https://www.googleapis.com/oauth2/v4/token", strings.NewReader(requestData.Encode()))
		if err != nil {
			panic(err)
		}

		// Set headers
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			panic(err)
		}
		fmt.Println(result["access_token"])
		// call user info api with access token as bearer token
		//https://www.googleapis.com/oauth2/v3/userinfo
		req, err = http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", result["access_token"]))

		resp, err = client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var userInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			panic(err)
		}
		fmt.Println(userInfo)

		return c.Redirect(302, "/")
	})
	e.Logger.Fatal(e.Start(":42069"))
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	loggly "github.com/jamespearly/loggly"
)

type TotalResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After      string      `json:"after"`
		Dist       int         `json:"dist"`
		Modhash    string      `json:"Modhash"`
		Geo_Filter string      `json:"geo_filter"`
		Children   []ChildData `json:"children"`
		Before     string      `json:"before"`
	} `json:"data"`
}

type ChildData struct {
	Kind_1 string           `json:"kind"`
	Data   RequiredResponse `json:"data"`
}
type RequiredResponse struct {
	Id     string `json:"id"`
	Author string `json:"author"`
	Body   string `json:"body"`
	Score  int    `json:"score"`
}

func main() {
	logglyToken := os.Getenv("LOGGLY_TOKEN")
	clientID := os.Getenv("REDDIT_CLIENT-ID")
	clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")
	client := loggly.New(logglyToken)
	accessToken, err := getAccessToken(clientID, clientSecret)
	if err != nil {
		client.EchoSend("Error ", "Access token error")
	}
	ticker := time.NewTicker(5 * time.Minute)
	fetchData(client, accessToken)

	for range ticker.C {
		fetchData(client, accessToken)
	}
}

func getAccessToken(clientID, clientSecret string) (string, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "kritika/1.0 (by Key_Excuse_5158)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, er := client.Do(req)

	if er != nil {
		return "", er
	}

	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken, ok := result["access_token"].(string)

	if !ok {
		return "", fmt.Errorf("failed to retrieve access token")
	}

	return accessToken, nil
}

func fetchData(client *loggly.ClientType, accessToken string) {
	c := http.Client{Timeout: time.Duration(15) * time.Second}
	req, err := http.NewRequest("GET", "http://www.reddit.com/r/all/comments.json?limit=20", nil)
	if err != nil {
		client.EchoSend("error", "error creating request")
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Comments-Display by Key_Excuse_5158")

	resp, err := c.Do(req)
	if err != nil {
		client.EchoSend("error", "error making request")
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		client.EchoSend("error", "error reading body")
	}
	if resp.StatusCode != http.StatusOK {
		client.EchoSend("error", fmt.Sprintf("Unexpected status code: %d", resp.StatusCode))
		return
	}
	client.EchoSend("debug", fmt.Sprintf("Size: %d", len(body)))

	var response TotalResponse

	err = json.Unmarshal(body, &response)

	if err != nil {
		client.EchoSend("error", "error decoding data")
	}
	size := len(response.Data.Children)

	s := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(s)

	tableName := "kparajul_Reddit_Comments"

	client.EchoSend("info", fmt.Sprintf("Number of comments pulled: %d", size))
	for _, child := range response.Data.Children {
		comment := child.Data
		item := RequiredResponse{
			Id:     comment.Id,
			Author: comment.Author,
			Body:   comment.Body,
			Score:  comment.Score,
		}

		av, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
			client.EchoSend("error", fmt.Sprintf("Marshalling error: %s", err))

		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(tableName),
		}

		_, err = svc.PutItem(input)

		if err != nil {
			client.EchoSend("error", fmt.Sprintf("Error during putItem %s", err))
			return
		}

	}

}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	client := loggly.New(logglyToken)
	ticker := time.NewTicker(1 * time.Minute)
	fetchData(client)

	for range ticker.C {
		fetchData(client)
	}
}

func fetchData(client *loggly.ClientType) {
	c := http.Client{Timeout: time.Duration(15) * time.Second}
	req, err := http.NewRequest("GET", "http://www.reddit.com/r/all/comments.json?limit=20", nil)
	if err != nil {
		client.EchoSend("error", "error creating request")
		return
	}
	req.Header.Set("Host", "www.reddit.com")
	req.Header.Set("User-Agent", "kritika/1.0 by Key_Excuse_5158")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accep-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

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
		fmt.Printf("\n Fetched comments: Id: %s, author: %s, body: %s, score: %d \n", comment.Id, comment.Author, comment.Body, comment.Score)

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

		client.EchoSend("info", "Successful adding to dynamoDB")

	}

}

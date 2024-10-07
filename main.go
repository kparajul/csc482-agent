package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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
	client := loggly.New("My-Go-Demo")
	ticker := time.NewTicker(5 * time.Minute)
	fetchData(client)

	for range ticker.C {
		fetchData(client)
	}
}

func fetchData(client *loggly.ClientType) {
	c := http.Client{Timeout: time.Duration(15) * time.Second}
	req, err := http.NewRequest("GET", "http://www.reddit.com/r/all/comments.json?limit=100", nil)
	if err != nil {
		client.EchoSend("error", "error creating request")
		return
	}
	req.Header.Set("Host", "www.reddit.com")
	req.Header.Set("User-Agent", "kritika/1.0 by Key_Excuse_5158")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accep-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	//req.Header.Set("Connection")
	resp, err := c.Do(req)
	if err != nil {
		client.EchoSend("error", "error making request")
		fmt.Printf("Error", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		client.EchoSend("error", "error reading body")
	}
	fmt.Printf(string(body) + "\n")

	var response TotalResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		client.EchoSend("error", "error decoding data")
		fmt.Printf("Error", err)
	}
	for _, child := range response.Data.Children {
		comment := child.Data
		logData := fmt.Sprintf("Fetched comments: Id: %s, author: %s, body: %s, score: %d", comment.Id, comment.Author, comment.Body, comment.Score)
		client.EchoSend("info", logData)
	}

}

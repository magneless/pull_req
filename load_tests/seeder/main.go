package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const baseURL = "http://pull_req:8080"

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PullRequestReq struct {
	PRID     string `json:"pull_request_id"`
	PRName   string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

func main() {
	log.Println("Waiting for service to be ready...")
	waitForService()

	log.Println("Starting database seeding...")
	seedData()
	log.Println("Database seeded successfully!")
}

func waitForService() {
	for range 60 {
		resp, err := http.Get(baseURL + "/team/get?team_name=check")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Fatal("Service unavailable after 60 seconds")
}

func seedData() {
	users := make([]string, 0, 200)

	for i := 1; i <= 20; i++ {
		teamName := fmt.Sprintf("team_%d", i)
		members := make([]TeamMember, 0, 10)

		for j := 1; j <= 10; j++ {
			userID := fmt.Sprintf("u_%d_%d", i, j)
			members = append(members, TeamMember{
				UserID:   userID,
				Username: fmt.Sprintf("User %d-%d", i, j),
				IsActive: true,
			})
			users = append(users, userID)
		}

		team := Team{TeamName: teamName, Members: members}
		sendJSON("/team/add", team)
	}

	for i := 1; i <= 50; i++ {
		if len(users) == 0 {
			break
		}
		author := users[rand.Intn(len(users))]
		pr := PullRequestReq{
			PRID:     fmt.Sprintf("pr_seed_%d", i),
			PRName:   fmt.Sprintf("Feature %d", i),
			AuthorID: author,
		}
		sendJSON("/pullRequest/create", pr)
	}
}

func sendJSON(path string, data interface{}) {
	b, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Printf("Failed to POST %s: %v", path, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		log.Printf("Error POST %s: Status %d", path, resp.StatusCode)
	}
}
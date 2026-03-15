package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"the_governor/server"
	"the_governor/usecase"
)

func readContent(ghUtility usecase.GithubUtility, ctx context.Context, wg *sync.WaitGroup, owner, repoName, path string) {

	if isRepoAvailableForOwner(ghUtility, ctx, owner, repoName) {
		content, err := ghUtility.ReadFileContent(ctx, owner, repoName, path)
		if err != nil {
			fmt.Printf("Error reading file content: %v\n", err)
			return
		}
		fmt.Printf("File content:\n%s\n", content)
	}
	wg.Done()
}

func isRepoAvailableForOwner(ghUtility usecase.GithubUtility, ctx context.Context, owner, repoName string) bool {
	_, err := ghUtility.GetRepository(ctx, owner, repoName)
	if err != nil {
		fmt.Printf("Repository %s/%s not found or error: %v\n", owner, repoName, err)
		return false
	}
	return true
}

func init() {
	// Load configuration from config.json
	configData, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return
	}
	//set env  variable for configs by looping and parsing JSON data
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return
	}
	for key, value := range config {
		switch v := value.(type) {
		case string:
			os.Setenv(key, v)
		case float64:
			os.Setenv(key, strconv.Itoa(int(v)))
		case int:
			os.Setenv(key, strconv.Itoa(v))
		}
	}

}

func main() {
	fmt.Println("=== The Governor - Decentralized Gateway Configuration Manager ===")
	server.StartServer()
}

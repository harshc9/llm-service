package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:8080"

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Client struct {
	ApiToken string `json:"api_token"`
}

func main() {
	fmt.Println("🚀 Starting Gemini Service Integration Test...")

	// 1. Create Project
	fmt.Print("Step 1: Creating Project... ")
	projID := createProject("Test Project")
	fmt.Printf("Done (ID: %s)\n", projID)

	// 2. Add API Key (Using a dummy key for the test, but sync might fail)
	fmt.Print("Step 2: Adding API Key... ")
	addKey(projID, "test-key-1", "DUMMY_KEY_FOR_TESTING")
	fmt.Println("Done")

	// 3. Create Client
	fmt.Print("Step 3: Registering Client... ")
	clientName := fmt.Sprintf("Integration Tester %d", time.Now().Unix())
	token := createClient(clientName)
	fmt.Println("Done")

	// 4. Sync Models (This will fail with a dummy key, so we'll just check for 200/500)
	fmt.Print("Step 4: Triggering Sync... ")
	syncModels()
	fmt.Println("Done")

	// 5. Test Health
	fmt.Print("Step 5: Checking Health... ")
	checkHealth()
	fmt.Println("Done")

	// 6. Test Generation (Will fail if no valid key was added, but tests the auth/routing flow)
	fmt.Print("Step 6: Testing Text Generation... ")
	testGeneration(token)
	fmt.Println("Done")

	fmt.Println("\n✅ Sequence Complete! Visit http://localhost:8080/dashboard to see the results.")
}

func testGeneration(token string) {
	payload := map[string]string{"prompt": "Hello, this is an automated test."}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/generate", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	// We expect a 200 if key is valid, or 500 if key is dummy, but 401/403/404 would be failures
	if err != nil {
		fmt.Printf("\n❌ Generation request failed: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404 {
		fmt.Printf("\n❌ Generation failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
}

func createProject(name string) string {
	payload := map[string]string{"name": name, "provider": "google"}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/v1/admin/projects", "application/json", bytes.NewBuffer(data))
	if err != nil || resp.StatusCode != 201 {
		fmt.Printf("\n❌ Failed to create project: %v (Status: %d)\n", err, resp.StatusCode)
		os.Exit(1)
	}
	var p Project
	json.NewDecoder(resp.Body).Decode(&p)
	return p.ID
}

func addKey(projID, alias, key string) {
	payload := map[string]interface{}{
		"project_id": projID,
		"alias":      alias,
		"api_key":    key,
		"priority":   1,
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/v1/admin/keys", "application/json", bytes.NewBuffer(data))
	if err != nil || resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("\n❌ Failed to add key: %v (Status: %d, Body: %s)\n", err, resp.StatusCode, string(bodyBytes))
		os.Exit(1)
	}
}

func createClient(name string) string {
	payload := map[string]string{"name": name}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/v1/admin/clients", "application/json", bytes.NewBuffer(data))
	if err != nil || resp.StatusCode != 201 {
		fmt.Printf("\n❌ Failed to create client: %v (Status: %d)\n", err, resp.StatusCode)
		os.Exit(1)
	}
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res["api_token"].(string)
}

func syncModels() {
	resp, err := http.Post(baseURL+"/v1/admin/sync-models", "application/json", nil)
	// We expect a 500 if the key is invalid, but the route should exist
	if err != nil {
		fmt.Printf("\n❌ Failed to reach sync endpoint: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode == 404 {
		fmt.Printf("\n❌ Sync endpoint not found (404)\n")
		os.Exit(1)
	}
}

func checkHealth() {
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("\n❌ Health check failed: %v\n", err)
		os.Exit(1)
	}
}

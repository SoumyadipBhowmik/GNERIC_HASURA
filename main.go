package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Can't load env file")
	}
}

func executeGraphQLQuery(query string, variables map[string]interface{}, endpoint string, authToken string) ([]byte, error) {
	requestPayload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func handleGraphQLRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `mutation InsertUser($name: String!, $age: Int!) {
		insert_users(objects: [{ name: $name, age: $age }]) {
			returning {
				id
				name
				age
			}
		}
	}`

	variables := map[string]interface{}{
		"name": input.Name,
		"age":  input.Age,
	}

	endpoint := os.Getenv("HASURA_ENDPOINT")
	authToken := os.Getenv("HASURA_ADMIN_SECRET")

	if endpoint == "" || authToken == "" {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response, err := executeGraphQLQuery(query, variables, endpoint, authToken)
	if err != nil {
		http.Error(w, "GraphQL query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/graphql", handleGraphQLRequest).Methods(http.MethodPost)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5188"
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

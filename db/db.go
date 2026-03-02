package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// SudokuData is the puzzle content stored in the sudoku field
type SudokuData struct {
	Grid      []int   `json:"grid"`
	Solution  []int   `json:"solution"`
	Regions   [][]int `json:"regions"`
	BoxWidth  int     `json:"boxWidth"`
	BoxHeight int     `json:"boxHeight"`
}

// UploadResponse is the JSON body returned by POST /api/sudokus/external
type UploadResponse struct {
	ID      string `json:"id"`
	Skipped bool   `json:"skipped,omitempty"`
}

var (
	apiURL     string
	apiUser    string
	apiPass    string
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: No .env file found")
	}

	apiURL = os.Getenv("SUDOKU_API_URL")
	if apiURL == "" {
		apiURL = "https://sudoku.mljr.eu"
	}
	apiUser = os.Getenv("SUDOKU_API_USER")
	apiPass = os.Getenv("SUDOKU_API_PASSWORD")
}

// Authenticate verifies the API credentials are set and the server is reachable.
func Authenticate() error {
	if apiUser == "" || apiPass == "" {
		return fmt.Errorf("missing SUDOKU_API_USER or SUDOKU_API_PASSWORD")
	}

	// POST with wrong JSON to get a quick 400 (not 401) – proves auth works
	req, err := http.NewRequest("GET", apiURL+"/api/sudokus?perPage=1", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiUser, apiPass)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach sudoku API at %s: %v", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed: invalid credentials")
	}
	return nil
}

// SudokuExists checks if a sudoku with the given ID is already in the database.
// Uses the open HEAD endpoint (no auth required).
func SudokuExists(id string) (bool, error) {
	req, err := http.NewRequest("HEAD", apiURL+"/api/sudokus/"+id, nil)
	if err != nil {
		return false, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check sudoku existence: %v", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// UploadSudoku posts a sudoku to the authenticated external upload endpoint.
func UploadSudoku(sudokuData map[string]interface{}) (*UploadResponse, error) {
	id, ok := sudokuData["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("invalid or missing id in sudoku data")
	}

	body, err := json.Marshal(sudokuData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sudoku data: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL+"/api/sudokus/external", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiUser, apiPass)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 401:
		return nil, fmt.Errorf("authentication failed: check SUDOKU_API_USER and SUDOKU_API_PASSWORD")
	case 200, 201:
		var result UploadResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}
		return &result, nil
	default:
		return nil, fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
}

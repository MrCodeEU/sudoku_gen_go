package db

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/habibrosyad/pocketbase-go-sdk"
	"github.com/joho/godotenv"
)

// SudokuData represents the structure of a sudoku puzzle
type SudokuData struct {
	Grid      [][]int `json:"grid"`
	Solution  [][]int `json:"solution"`
	Regions   [][]int `json:"regions"`
	BoxWidth  int     `json:"boxWidth"`
	BoxHeight int     `json:"boxHeight"`
}

// SudokuRecord represents a record in the PocketBase database
type SudokuRecord struct {
	ID         string     `json:"id"`
	Sudoku     SudokuData `json:"sudoku"`
	Difficulty string     `json:"difficulty"`
	Size       string     `json:"size"`
	Layout     string     `json:"layout"`
	Created    string     `json:"created"`
	Updated    string     `json:"updated"`
}

var client *pocketbase.Client

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Warning: No .env file found")
	}

	email := os.Getenv("POCKETBASE_EMAIL")
	password := os.Getenv("POCKETBASE_PASSWORD")

	// Create client with superuser authentication
	client = pocketbase.NewClient("https://base.mljr.eu",
		pocketbase.WithSuperuserEmailPassword(email, password))

	// Attempt initial authorization
	if err := client.Authorize(); err != nil {
		fmt.Printf("⚠️ Initial authorization failed: %v\n", err)
	}
}

// Authenticate tries to authenticate with PocketBase
func Authenticate() error {
	// Simple authorization check
	err := client.Authorize()
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	// Start the re-authentication timer
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			if err := client.Authorize(); err != nil {
				fmt.Printf("⚠️ Re-authentication failed: %v\n", err)
			} else {
				fmt.Println("🔄 Successfully re-authenticated with PocketBase")
			}
		}
	}()
	return nil
}

func UploadSudoku(sudokuData map[string]interface{}) (*pocketbase.ResponseCreate, error) {
	// Validate ID length
	id, ok := sudokuData["id"].(string)
	if !ok || len(id) > 6 {
		return nil, fmt.Errorf("invalid ID: must be a string of max 6 characters")
	}

	layoutConfig := "jigsaw"
	if sudokuData["layoutType"] != "jigsaw" {
		layoutConfig = fmt.Sprintf("%dx%d",
			sudokuData["boxWidth"].(int),
			sudokuData["boxHeight"].(int))
	}

	sudokuJSON, err := json.Marshal(map[string]interface{}{
		"grid":      sudokuData["grid"],
		"solution":  sudokuData["solution"],
		"regions":   sudokuData["regions"],
		"boxWidth":  sudokuData["boxWidth"],
		"boxHeight": sudokuData["boxHeight"],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sudoku data: %v", err)
	}

	data := map[string]any{
		"id":         id,
		"sudoku":     string(sudokuJSON),
		"difficulty": fmt.Sprintf("%v", sudokuData["difficulty"]),
		"size":       fmt.Sprintf("%v", sudokuData["size"]),
		"layout":     layoutConfig,
	}

	// Check if record with this ID already exists
	exists, err := SudokuExists(id)
	if err != nil {
		return nil, fmt.Errorf("failed to check if sudoku exists: %v", err)
	}
	if exists {
		return nil, fmt.Errorf("sudoku with ID %s already exists", id)
	}

	record, err := client.Create("sudokus", data)
	if err != nil {
		return nil, fmt.Errorf("failed to upload sudoku: %v", err)
	}
	return &record, nil
}

func GetSudoku(id string) (map[string]interface{}, error) {
	record, err := client.One("sudokus", id)
	if err != nil {
		return nil, fmt.Errorf("failed to load sudoku %s: %v", id, err)
	}

	var sudokuData map[string]interface{}
	err = json.Unmarshal([]byte(record["sudoku"].(string)), &sudokuData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sudoku data: %v", err)
	}

	result := map[string]interface{}{
		"id":         record["id"],
		"difficulty": record["difficulty"],
		"size":       record["size"],
		"layoutType": record["layout"],
		"created":    record["created"],
		"updated":    record["updated"],
	}

	for k, v := range sudokuData {
		result[k] = v
	}

	return result, nil
}

func ListSudokus(page int, perPage int, filters map[string]string, sortField string, sortOrder string) (*pocketbase.ResponseList[map[string]any], error) {
	var filterRules []string

	if diff, ok := filters["difficulty"]; ok {
		filterRules = append(filterRules, fmt.Sprintf("difficulty >= %s && difficulty <= %s", diff, diff))
	}
	if size, ok := filters["size"]; ok {
		filterRules = append(filterRules, fmt.Sprintf("size = \"%s\"", size))
	}
	if layout, ok := filters["layout"]; ok {
		switch layout {
		case "jigsaw":
			filterRules = append(filterRules, "layout = \"jigsaw\"")
		case "regular":
			filterRules = append(filterRules, "layout != \"jigsaw\"")
		default:
			if strings.Contains(layout, "x") {
				filterRules = append(filterRules, fmt.Sprintf("layout = \"%s\"", layout))
			}
		}
	}

	sort := sortField
	if sortOrder == "desc" {
		sort = "-" + sortField
	}

	params := pocketbase.ParamsList{
		Page:    page,
		Size:    perPage,
		Sort:    sort,
		Filters: strings.Join(filterRules, " && "),
	}

	result, err := client.List("sudokus", params)
	return &result, err
}

func SudokuExists(id string) (bool, error) {
	_, err := client.One("sudokus", id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

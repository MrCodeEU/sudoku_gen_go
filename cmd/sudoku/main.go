package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sudoku_gen_go/db"
	"sudoku_gen_go/internal/generator"
	"sudoku_gen_go/internal/types"
	"sudoku_gen_go/internal/visualizer"
	"time"
)

func generateID() string {
	bytes := make([]byte, 15)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func main() {
	// Check environment variables first
	if os.Getenv("POCKETBASE_EMAIL") == "" || os.Getenv("POCKETBASE_PASSWORD") == "" {
		fmt.Println("❌ Error: Missing environment variables")
		fmt.Println("Please set POCKETBASE_EMAIL and POCKETBASE_PASSWORD in .env file")
		os.Exit(1)
	}

	// First, try to authenticate with PocketBase
	fmt.Println("\nAuthenticating with PocketBase...")
	if err := db.Authenticate(); err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Successfully authenticated with PocketBase")

	reader := bufio.NewReader(os.Stdin)

	// Get user preferences
	size := getUserInput(reader, "Enter grid size (9, 12, or 16): ", validateSize)
	layout := getUserInput(reader, "Enter layout type (normal/jigsaw): ", validateLayout)
	difficulty := getUserInput(reader, "Enter difficulty (1-5): ", validateDifficulty)
	count := getUserInput(reader, "How many puzzles to generate: ", validateCount)
	threads := getUserInput(reader, "Enter number of threads (1-32): ", validateThreads)

	numPuzzles, _ := strconv.Atoi(count)
	sizeNum, _ := strconv.Atoi(size)
	diffNum, _ := strconv.Atoi(difficulty)
	threadNum, _ := strconv.Atoi(threads)
	sudokuType := types.Normal
	if layout == "jigsaw" {
		sudokuType = types.Jigsaw
	}

	for i := 0; i < numPuzzles; i++ {
		fmt.Printf("\nGenerating puzzle %d/%d\n", i+1, numPuzzles)
		maxMainRetries := 10
		mainRetries := 0

		for mainRetries < maxMainRetries {
			fmt.Printf("\nGenerating %v Sudoku %dx%d (Difficulty: %d)\n",
				sudokuType, sizeNum, sizeNum, diffNum)

			start := time.Now()
			generator := generator.NewClassicGenerator(sizeNum, sudokuType)
			generator.SetDifficulty(diffNum)
			generator.SetThreads(threadNum)
			generator.SetMaxRetries(maxMainRetries) // Add this line

			grid, err := generator.Generate()
			elapsed := time.Since(start)
			fmt.Printf("Generation time: %v\n", elapsed)

			if err != nil {
				fmt.Printf("Error generating puzzle: %v\n", err)
				if mainRetries < maxMainRetries-1 {
					fmt.Print("Would you like to retry? (y/n): ")
					response, _ := reader.ReadString('\n')
					response = strings.TrimSpace(strings.ToLower(response))
					if response == "y" {
						mainRetries++
						continue
					}
				}
				break
			}

			// Visualize the grid
			viz := visualizer.NewVisualizer(grid)
			if sudokuType == types.Jigsaw {
				viz.PrintJigsaw()
			} else {
				viz.Print()
			}

			// Replace the JSON file saving code with this:
			layoutConfig := "normal"
			if sudokuType == types.Jigsaw {
				layoutConfig = "jigsaw"
			}

			// Convert difficulty from 1-5 scale to 0-1 scale
			normalizedDifficulty := float64(diffNum) / 5.0

			// Prepare the data for upload
			sudokuData := map[string]interface{}{
				"id":         generateID(),
				"grid":       grid.Puzzle,
				"solution":   grid.Solution,
				"regions":    grid.SubGrids,
				"boxWidth":   grid.BoxWidth,
				"boxHeight":  grid.BoxHeight,
				"size":       grid.Size,
				"difficulty": normalizedDifficulty,
				"layoutType": layoutConfig,
			}

			// Upload to PocketBase
			fmt.Printf("\nUploading puzzle to PocketBase...\n")
			record, err := db.UploadSudoku(sudokuData)
			if err != nil {
				fmt.Printf("❌ Error uploading to PocketBase: %v\n", err)
				continue
			}
			fmt.Printf("✅ Successfully uploaded sudoku with ID: %s\n", record.ID)
			break
		}
	}
}

func getUserInput(reader *bufio.Reader, prompt string, validator func(string) bool) string {
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if validator(input) {
			return input
		}
		fmt.Println("Invalid input, please try again.")
	}
}

func validateSize(input string) bool {
	validSizes := map[string]bool{"9": true, "12": true, "16": true}
	return validSizes[input]
}

func validateLayout(input string) bool {
	return input == "normal" || input == "jigsaw"
}

func validateDifficulty(input string) bool {
	diff, err := strconv.Atoi(input)
	return err == nil && diff >= 1 && diff <= 5
}

func validateCount(input string) bool {
	count, err := strconv.Atoi(input)
	return err == nil && count > 0 && count <= 100
}

// Add new validator
func validateThreads(input string) bool {
	threads, err := strconv.Atoi(input)
	return err == nil && threads >= 1 && threads <= 32
}

func verifySolution(grid *types.Grid) bool {
	// Verify rows
	for i := 0; i < grid.Size; i++ {
		if !isValidSet(grid.Solution[i]) {
			return false
		}
	}

	// Verify columns
	for i := 0; i < grid.Size; i++ {
		col := make([]int, grid.Size)
		for j := 0; j < grid.Size; j++ {
			col[j] = grid.Solution[j][i]
		}
		if !isValidSet(col) {
			return false
		}
	}

	// Verify subgrids
	for _, region := range grid.SubGrids {
		values := make([]int, len(region))
		for i, idx := range region {
			row, col := idx/grid.Size, idx%grid.Size
			values[i] = grid.Solution[row][col]
		}
		if !isValidSet(values) {
			return false
		}
	}

	return true
}

func isValidSet(nums []int) bool {
	seen := make(map[int]bool)
	for _, num := range nums {
		if num == 0 {
			continue
		}
		if seen[num] {
			return false
		}
		seen[num] = true
	}
	return true
}

package main

import (
	"bufio"
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

	successfulPuzzles := 0
	for successfulPuzzles < numPuzzles {
		fmt.Printf("\nGenerating puzzle %d/%d\n", successfulPuzzles+1, numPuzzles)

		fmt.Printf("\nGenerating %v Sudoku %dx%d (Difficulty: %d)\n",
			sudokuType, sizeNum, sizeNum, diffNum)

		start := time.Now()
		generator := generator.NewClassicGenerator(sizeNum, sudokuType)
		generator.SetDifficulty(diffNum)
		generator.SetThreads(threadNum)

		grid, err := generator.Generate()
		elapsed := time.Since(start)
		fmt.Printf("Generation time: %v\n", elapsed)

		if err != nil {
			fmt.Printf("Error generating puzzle: %v\n", err)
			continue
		}

		// Visualize the grid
		viz := visualizer.NewVisualizer(grid)
		if sudokuType == types.Jigsaw {
			viz.PrintJigsaw()
		} else {
			viz.Print()
		}

		// Flatten 2D arrays into 1D arrays
		flatPuzzle := make([]int, grid.Size*grid.Size)
		flatSolution := make([]int, grid.Size*grid.Size)
		for i := 0; i < grid.Size; i++ {
			for j := 0; j < grid.Size; j++ {
				index := i*grid.Size + j
				flatPuzzle[index] = grid.Puzzle[i][j]
				flatSolution[index] = grid.Solution[i][j]
			}
		}

		layoutConfig := "regular"
		if sudokuType == types.Jigsaw {
			layoutConfig = "jigsaw"
		}

		normalizedDifficulty := float64(diffNum) / 5.0

		sudokuData := map[string]interface{}{
			"id":         generateSudokuID(grid.Puzzle, grid.Size, grid.BoxWidth, grid.BoxHeight, normalizedDifficulty),
			"grid":       flatPuzzle,
			"solution":   flatSolution,
			"regions":    grid.SubGrids,
			"boxWidth":   grid.BoxWidth,
			"boxHeight":  grid.BoxHeight,
			"size":       grid.Size,
			"difficulty": normalizedDifficulty,
			"layoutType": layoutConfig,
			"timestamp":  time.Now().UnixMilli(),
		}

		fmt.Printf("\nUploading puzzle to PocketBase...\n")
		record, err := db.UploadSudoku(sudokuData)
		if err != nil {
			fmt.Printf("❌ Error uploading to PocketBase: %v\n", err)
			continue
		}
		fmt.Printf("✅ Successfully uploaded sudoku with ID: %s\n", record.ID)
		successfulPuzzles++
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

// generateSudokuID creates a deterministic hash ID based on sudoku properties
func generateSudokuID(grid [][]int, size int, boxWidth int, boxHeight int, difficulty float64) string {
	// Create a flattened string of all grid numbers
	var gridStr strings.Builder
	for _, row := range grid {
		for _, num := range row {
			gridStr.WriteString(strconv.Itoa(num))
		}
	}

	// Combine all properties into a single string
	str := fmt.Sprintf("%s%d%d%d%f", gridStr.String(), size, boxWidth, boxHeight, difficulty)

	// Calculate hash similar to the JS implementation
	var hash int
	for _, char := range str {
		hash = ((hash << 5) - hash) + int(char)
		hash = hash & hash // Keep within 32-bit range
	}

	// Convert to base36 and take first 6 characters
	id := strconv.FormatInt(int64(hash&0x7fffffff), 36)
	if len(id) > 6 {
		id = id[:6]
	}
	return id
}

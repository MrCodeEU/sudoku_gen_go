package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sudoku_gen_go/internal/generator"
	"sudoku_gen_go/internal/types"
	"sudoku_gen_go/internal/visualizer"
	"time"
)

func main() {
	// Test different sudoku types and sizes
	testCases := []struct {
		size int
		typ  types.SudokuType
		diff int
	}{
		{9, types.Normal, 2},
		{9, types.Jigsaw, 3},
		{16, types.Normal, 1},
		{12, types.Normal, 4},
		{12, types.Jigsaw, 4},
	}

	reader := bufio.NewReader(os.Stdin)

	for _, tc := range testCases {
		maxMainRetries := 3
		mainRetries := 0

		for mainRetries < maxMainRetries {
			fmt.Printf("\nGenerating %v Sudoku %dx%d (Difficulty: %d)\n",
				tc.typ, tc.size, tc.size, tc.diff)

			start := time.Now()
			generator := generator.NewClassicGenerator(tc.size, tc.typ)
			generator.SetDifficulty(tc.diff)

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
			if tc.typ == types.Jigsaw {
				viz.PrintJigsaw()
			} else {
				viz.Print()
			}

			// Verify solution
			if !verifySolution(grid) {
				fmt.Println("Warning: Invalid solution generated!")
				continue
			}

			// Save to JSON
			jsonData, err := grid.ToJSON()
			if err != nil {
				fmt.Printf("Error serializing to JSON: %v\n", err)
				continue
			}

			// Write to file for testing
			filename := fmt.Sprintf("sudoku_%v_%dx%d_diff%d.json",
				tc.typ, tc.size, tc.size, tc.diff)
			if err := os.WriteFile(filename, jsonData, 0644); err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
			}
			break
		}
	}
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

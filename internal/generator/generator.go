package generator

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sudoku_gen_go/internal/types"
	"time"
)

// SudokuGenerator interface defines methods for generating Sudoku puzzles
type SudokuGenerator interface {
	Generate() (*types.Grid, error)
	SetDifficulty(level int) error
}

// ClassicGenerator implements SudokuGenerator
type ClassicGenerator struct {
	difficulty int
	size       int
	sudokuType types.SudokuType
	threads    int
	maxRetries int // Add this field
}

func NewClassicGenerator(size int, typ types.SudokuType) *ClassicGenerator {
	return &ClassicGenerator{
		difficulty: 1,
		size:       size,
		sudokuType: typ,
		threads:    4,  // Default threads
		maxRetries: 10, // Default max retries
	}
}

func (g *ClassicGenerator) SetThreads(threads int) {
	g.threads = threads
}

// Add this method
func (g *ClassicGenerator) SetMaxRetries(retries int) {
	g.maxRetries = retries
}

// Generate implements the backtracking algorithm with MRV
func (g *ClassicGenerator) Generate() (*types.Grid, error) {
	startTime := time.Now()
	maxTime := time.Duration(g.getMaxGenerationTime()) * time.Millisecond
	retries := 0

	for retries < g.maxRetries { // Use g.maxRetries instead of hardcoded value
		fmt.Printf("Attempt %d/%d...\n", retries+1, g.maxRetries)
		grid := types.NewGrid(g.size, g.sudokuType)
		if g.sudokuType == types.Jigsaw {
			regions, err := g.generateJigsawRegions()
			if err != nil {
				fmt.Printf("Failed to generate jigsaw regions: %v\n", err)
				retries++
				continue
			}
			grid.SubGrids = regions
		} else {
			grid.SubGrids = g.generateNormalSubgrids()
		}

		if solved := g.solve(grid, startTime, maxTime); solved {
			fmt.Printf("Successfully generated puzzle on attempt %d\n", retries+1)
			// Copy solution
			grid.Solution = make([][]int, g.size)
			for i := range grid.Puzzle {
				grid.Solution[i] = make([]int, g.size)
				copy(grid.Solution[i], grid.Puzzle[i])
			}

			// Remove numbers based on difficulty
			g.removeNumbers(grid)
			return grid, nil
		}

		fmt.Printf("Failed to solve grid on attempt %d\n", retries+1)
		retries++
	}

	return nil, fmt.Errorf("failed to generate valid puzzle after %d attempts", g.maxRetries)
}

func (g *ClassicGenerator) solve(grid *types.Grid, startTime time.Time, maxTime time.Duration) bool {
	if time.Since(startTime) > maxTime {
		return false
	}

	pos := g.findEmptyPositionWithMRV(grid)
	if pos == nil {
		return true
	}

	row, col := pos[0], pos[1]
	nums := g.getShuffledNumbers()

	for _, num := range nums {
		if g.isValid(grid, num, row, col) {
			grid.Puzzle[row][col] = num
			if g.solve(grid, startTime, maxTime) {
				return true
			}
			grid.Puzzle[row][col] = 0
		}
	}

	return false
}

func (g *ClassicGenerator) findEmptyPositionWithMRV(grid *types.Grid) []int {
	minPossibilities := g.size + 1
	var bestPos []int

	for i := 0; i < g.size; i++ {
		for j := 0; j < g.size; j++ {
			if grid.Puzzle[i][j] == 0 {
				count := 0
				for num := 1; num <= g.size; num++ {
					if g.isValid(grid, num, i, j) {
						count++
					}
				}
				if count < minPossibilities {
					minPossibilities = count
					bestPos = []int{i, j}
				}
			}
		}
	}

	return bestPos
}

func (g *ClassicGenerator) isValid(grid *types.Grid, num, row, col int) bool {
	// Check row
	for x := 0; x < g.size; x++ {
		if grid.Puzzle[row][x] == num {
			return false
		}
	}

	// Check column
	for x := 0; x < g.size; x++ {
		if grid.Puzzle[x][col] == num {
			return false
		}
	}

	// Check subgrid
	cellIndex := row*g.size + col
	regionIndex := g.findRegionIndex(grid, cellIndex)

	for _, idx := range grid.SubGrids[regionIndex] {
		r, c := idx/g.size, idx%g.size
		if grid.Puzzle[r][c] == num {
			return false
		}
	}

	return true
}

func (g *ClassicGenerator) getShuffledNumbers() []int {
	nums := make([]int, g.size)
	for i := range nums {
		nums[i] = i + 1
	}
	rand.Shuffle(len(nums), func(i, j int) {
		nums[i], nums[j] = nums[j], nums[i]
	})
	return nums
}

func (g *ClassicGenerator) getMaxGenerationTime() int {
	if g.sudokuType == types.Jigsaw {
		return max(15000, g.size*g.size*50)
	}
	return 5000
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *ClassicGenerator) SetDifficulty(level int) error {
	if level < 1 || level > 5 {
		return errors.New("difficulty must be between 1 and 5")
	}
	g.difficulty = level
	return nil
}

func (g *ClassicGenerator) generateNormalSubgrids() [][]int {
	size := g.size
	boxSize := int(math.Sqrt(float64(size)))
	// Handle non-square sizes (like 12)
	if boxSize*boxSize != size {
		boxWidth := 3  // Default for size 12
		boxHeight := 4 // Default for size 12
		return g.generateRectangularSubgrids(boxWidth, boxHeight)
	}

	regions := make([][]int, size)

	for boxRow := 0; boxRow < boxSize; boxRow++ {
		for boxCol := 0; boxCol < boxSize; boxCol++ {
			region := make([]int, 0, g.size)
			for i := 0; i < boxSize; i++ {
				for j := 0; j < boxSize; j++ {
					row := boxRow*boxSize + i
					col := boxCol*boxSize + j
					region = append(region, row*g.size+col)
				}
			}
			regions[boxRow*boxSize+boxCol] = region
		}
	}

	return regions
}

func (g *ClassicGenerator) generateRectangularSubgrids(boxWidth, boxHeight int) [][]int {
	size := g.size
	regions := make([][]int, size)

	for boxRow := 0; boxRow < size/boxHeight; boxRow++ {
		for boxCol := 0; boxCol < size/boxWidth; boxCol++ {
			region := make([]int, 0, size)
			for i := 0; i < boxHeight; i++ {
				for j := 0; j < boxWidth; j++ {
					row := boxRow*boxHeight + i
					col := boxCol*boxWidth + j
					region = append(region, row*size+col)
				}
			}
			regions[boxRow*(size/boxWidth)+boxCol] = region
		}
	}

	return regions
}

func (g *ClassicGenerator) generateJigsawRegions() ([][]int, error) {
	if g.threads <= 1 {
		return g.generateJigsawRegionsSerial()
	}
	return g.generateJigsawRegionsParallel()
}

func (g *ClassicGenerator) generateJigsawRegionsParallel() ([][]int, error) {
	maxAttempts := 1000000
	attemptsPerThread := maxAttempts / g.threads
	resultChan := make(chan [][]int)
	errorChan := make(chan error)
	doneChan := make(chan struct{})
	defer close(doneChan)

	// Launch worker goroutines
	for i := 0; i < g.threads; i++ {
		go func(threadID int) {
			startProgress := attemptsPerThread * threadID
			endProgress := startProgress + attemptsPerThread
			lastProgress := 0

			for attempts := startProgress; attempts < endProgress; attempts++ {
				// Check if we should terminate
				select {
				case <-doneChan:
					return
				default:
				}

				// Show progress every 10%
				progress := (attempts - startProgress) * 100 / attemptsPerThread
				if progress/10 > lastProgress/10 {
					fmt.Printf("Thread %d: Generating jigsaw regions... %d%%\n", threadID, progress)
					lastProgress = progress
				}

				regions, valid := g.tryGenerateJigsawRegions()
				if valid {
					resultChan <- regions
					return
				}
			}
			if threadID == 0 {
				errorChan <- errors.New("failed to generate valid jigsaw regions")
			}
		}(i)
	}

	// Wait for result or error
	select {
	case regions := <-resultChan:
		close(resultChan)
		return regions, nil
	case err := <-errorChan:
		close(errorChan)
		return nil, err
	}
}

func (g *ClassicGenerator) tryGenerateJigsawRegions() ([][]int, bool) {
	size := g.size
	regions := make([][]int, size)
	used := make(map[int]bool)
	adjacency := g.buildAdjacencyList()

	valid := true
	for regionIdx := 0; regionIdx < size; regionIdx++ {
		available := make([]int, 0)
		for i := 0; i < size*size; i++ {
			if !used[i] {
				available = append(available, i)
			}
		}

		if len(available) == 0 {
			valid = false
			break
		}

		start := available[rand.Intn(len(available))]
		region := []int{start}
		used[start] = true

		for len(region) < size {
			candidates := make([]int, 0)
			for _, cell := range region {
				for _, neighbor := range adjacency[cell] {
					if !used[neighbor] {
						candidates = append(candidates, neighbor)
					}
				}
			}

			if len(candidates) == 0 {
				valid = false
				break
			}

			next := candidates[rand.Intn(len(candidates))]
			region = append(region, next)
			used[next] = true
		}

		if !valid {
			break
		}

		regions[regionIdx] = region
	}

	if valid && len(used) == size*size {
		return regions, true
	}
	return nil, false
}

// Rename existing generateJigsawRegions to generateJigsawRegionsSerial
func (g *ClassicGenerator) generateJigsawRegionsSerial() ([][]int, error) {
	maxAttempts := 1000000
	size := g.size
	lastProgress := 0

	for attempts := 0; attempts < maxAttempts; attempts++ {
		// Show progress every 10%
		progress := attempts * 100 / maxAttempts
		if progress/10 > lastProgress/10 {
			fmt.Printf("Generating jigsaw regions... %d%%\n", progress)
			lastProgress = progress
		}

		regions := make([][]int, size)
		used := make(map[int]bool)
		adjacency := g.buildAdjacencyList()

		valid := true
		for regionIdx := 0; regionIdx < size; regionIdx++ {
			available := make([]int, 0)
			for i := 0; i < size*size; i++ {
				if !used[i] {
					available = append(available, i)
				}
			}

			if len(available) == 0 {
				valid = false
				break
			}

			// Start with a random available cell
			start := available[rand.Intn(len(available))]
			region := []int{start}
			used[start] = true

			// Grow the region
			for len(region) < size {
				candidates := make([]int, 0)
				for _, cell := range region {
					for _, neighbor := range adjacency[cell] {
						if !used[neighbor] {
							candidates = append(candidates, neighbor)
						}
					}
				}

				if len(candidates) == 0 {
					valid = false
					break
				}

				next := candidates[rand.Intn(len(candidates))]
				region = append(region, next)
				used[next] = true
			}

			if !valid {
				break
			}

			regions[regionIdx] = region
		}

		if valid && len(used) == size*size {
			return regions, nil
		}
	}

	return nil, errors.New("failed to generate valid jigsaw regions")
}

func (g *ClassicGenerator) buildAdjacencyList() [][]int {
	size := g.size
	adjacency := make([][]int, size*size)

	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			idx := r*size + c
			neighbors := make([]int, 0, 4)

			// Check all four directions
			dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
			for _, dir := range dirs {
				newR, newC := r+dir[0], c+dir[1]
				if newR >= 0 && newR < size && newC >= 0 && newC < size {
					neighbors = append(neighbors, newR*size+newC)
				}
			}

			adjacency[idx] = neighbors
		}
	}

	return adjacency
}

func (g *ClassicGenerator) findRegionIndex(grid *types.Grid, cellIndex int) int {
	for i, region := range grid.SubGrids {
		for _, cell := range region {
			if cell == cellIndex {
				return i
			}
		}
	}
	return -1
}

func (g *ClassicGenerator) removeNumbers(grid *types.Grid) {
	cells := make([]int, g.size*g.size)
	for i := range cells {
		cells[i] = i
	}

	rand.Shuffle(len(cells), func(i, j int) {
		cells[i], cells[j] = cells[j], cells[i]
	})

	// Calculate cells to remove based on difficulty (1-5)
	// Difficulty 1: 30%, 2: 40%, 3: 50%, 4: 60%, 5: 70%
	cellsToRemove := (g.difficulty*10 + 20) * g.size * g.size / 100

	for i := 0; i < cellsToRemove; i++ {
		cellIdx := cells[i]
		row, col := cellIdx/g.size, cellIdx%g.size
		grid.Puzzle[row][col] = 0
	}
}

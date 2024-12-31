package generator

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sudoku_gen_go/internal/types"
	"sync"
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
		threads:    4,    // Default threads
		maxRetries: 1000, // Default max retries
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
	resultChan := make(chan *types.Grid)
	errorChan := make(chan error, 1)
	stopChan := make(chan struct{})
	var wg sync.WaitGroup

	// Launch worker goroutines
	attemptsPerThread := g.maxRetries / g.threads
	if attemptsPerThread < 1 {
		attemptsPerThread = 1
	}

	for i := 0; i < g.threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			for attempt := 0; attempt < attemptsPerThread; attempt++ {
				select {
				case <-stopChan:
					return
				default:
				}

				fmt.Printf("Thread %d: Attempt %d/%d\n", threadID, attempt+1, attemptsPerThread)

				grid := types.NewGrid(g.size, g.sudokuType)
				var err error

				if g.sudokuType == types.Jigsaw {
					grid.SubGrids, err = g.generateJigsawRegionsSerial()
					if err != nil {
						continue
					}
				} else {
					grid.SubGrids = g.generateNormalSubgrids()
				}

				if solved := g.solve(grid, startTime, maxTime); solved {
					// Copy solution
					grid.Solution = make([][]int, g.size)
					for i := range grid.Puzzle {
						grid.Solution[i] = make([]int, g.size)
						copy(grid.Solution[i], grid.Puzzle[i])
					}

					// Remove numbers based on difficulty
					g.removeNumbers(grid)

					select {
					case resultChan <- grid:
						close(stopChan) // Signal other goroutines to stop
						return
					case <-stopChan:
						return
					}
				}
			}
		}(i)
	}

	// Start a goroutine to close channels after all workers are done
	go func() {
		wg.Wait()
		select {
		case <-stopChan:
			// Already closed due to success
		default:
			errorChan <- fmt.Errorf("failed to generate valid puzzle after %d attempts", g.maxRetries)
		}
		close(resultChan)
	}()

	// Wait for result or timeout
	select {
	case grid, ok := <-resultChan:
		if !ok {
			// Channel closed without result
			err := <-errorChan
			return nil, err
		}
		fmt.Println("Successfully generated puzzle")
		return grid, nil
	case <-time.After(maxTime):
		close(stopChan) // Signal all goroutines to stop
		wg.Wait()       // Wait for all goroutines to finish
		return nil, fmt.Errorf("generation timed out after %v", maxTime)
	}
}

// Remove the parallel region generation since we're now parallelizing the whole generation
func (g *ClassicGenerator) generateJigsawRegions() ([][]int, error) {
	return g.generateJigsawRegionsSerial()
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

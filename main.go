package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ProgressReport struct {
	Phase     string
	Progress  float64
	Message   string
	Completed bool
}

type ProgressCallback func(ProgressReport)

type SudokuPuzzle struct {
	Grid     []int
	Solution []int
	Size     int
	Regions  [][]int
}

type SudokuGenerator struct {
	size              int
	boxWidth          int
	boxHeight         int
	layoutType        string
	regions           [][]int
	grid              []int
	filledCells       int
	maxGenerationTime time.Duration
	maxRetries        int
	progressChan      chan<- ProgressReport
}

func NewSudokuGenerator(size, boxWidth, boxHeight int, layoutType string) *SudokuGenerator {
	sg := &SudokuGenerator{
		size:              size,
		boxWidth:          boxWidth,
		boxHeight:         boxHeight,
		layoutType:        layoutType,
		grid:              make([]int, size*size),
		maxGenerationTime: time.Second * 15,
		maxRetries:        7,
	}

	var err error
	if layoutType == "jigsaw" {
		sg.regions, err = generateJigsawRegions(size)
		if err != nil {
			panic(err) // In production, handle this error more gracefully
		}
	} else {
		sg.regions = sg.generateRegions()
	}
	return sg
}

func (sg *SudokuGenerator) reset() {
	sg.grid = make([]int, sg.size*sg.size)
	sg.filledCells = 0
}

func (sg *SudokuGenerator) isValid(num int, pos int) bool {
	row := pos / sg.size
	col := pos % sg.size

	// Check row
	rowStart := row * sg.size
	for i := 0; i < sg.size; i++ {
		if sg.grid[rowStart+i] == num && i != col {
			return false
		}
	}

	// Check column
	for i := 0; i < sg.size; i++ {
		if sg.grid[i*sg.size+col] == num && i != row {
			return false
		}
	}

	// Check region
	regionIdx := -1
	for i, region := range sg.regions {
		for _, cell := range region {
			if cell == pos {
				regionIdx = i
				break
			}
		}
		if regionIdx != -1 {
			break
		}
	}

	for _, cell := range sg.regions[regionIdx] {
		if sg.grid[cell] == num && cell != pos {
			return false
		}
	}

	return true
}

func (sg *SudokuGenerator) generateRegions() [][]int {
	regions := make([][]int, 0)
	boxesPerRow := (sg.size + sg.boxWidth - 1) / sg.boxWidth
	boxesPerCol := (sg.size + sg.boxHeight - 1) / sg.boxHeight

	for boxY := 0; boxY < boxesPerCol; boxY++ {
		for boxX := 0; boxX < boxesPerRow; boxX++ {
			region := make([]int, 0)
			for i := 0; i < sg.boxHeight; i++ {
				for j := 0; j < sg.boxWidth; j++ {
					row := boxY*sg.boxHeight + i
					col := boxX*sg.boxWidth + j
					if row < sg.size && col < sg.size {
						region = append(region, row*sg.size+col)
					}
				}
			}
			if len(region) > 0 {
				regions = append(regions, region)
			}
		}
	}
	return regions
}

func (sg *SudokuGenerator) findEmptyPosition() int {
	for pos := 0; pos < sg.size*sg.size; pos++ {
		if sg.grid[pos] == 0 {
			return pos
		}
	}
	return -1
}

func (sg *SudokuGenerator) generate() bool {
	pos := sg.findEmptyPosition()
	if pos == -1 {
		return true
	}

	numbers := rand.Perm(sg.size)
	for _, num := range numbers {
		num++ // Convert to 1-based numbers
		if sg.isValid(num, pos) {
			sg.grid[pos] = num
			sg.filledCells++

			if sg.generate() {
				return true
			}

			sg.grid[pos] = 0
			sg.filledCells--
		}
	}

	return false
}

func (sg *SudokuGenerator) createPuzzle(difficulty float64) (*SudokuPuzzle, error) {
	sg.reset()
	if !sg.generate() {
		return nil, fmt.Errorf("failed to generate puzzle")
	}

	solution := make([]int, len(sg.grid))
	copy(solution, sg.grid)

	// Remove cells based on difficulty
	cellsToRemove := int(float64(sg.size*sg.size) * difficulty)
	positions := rand.Perm(sg.size * sg.size)
	for i := 0; i < cellsToRemove; i++ {
		sg.grid[positions[i]] = 0
	}

	return &SudokuPuzzle{
		Grid:     sg.grid,
		Solution: solution,
		Size:     sg.size,
		Regions:  sg.regions,
	}, nil
}

// ConcurrentPuzzleGenerator generates multiple puzzles concurrently
func ConcurrentPuzzleGenerator(size, boxWidth, boxHeight int, layoutType string, count int, difficulty float64, progressChan chan<- ProgressReport) []*SudokuPuzzle {
	puzzles := make([]*SudokuPuzzle, count)
	var wg sync.WaitGroup
	puzzleChan := make(chan *SudokuPuzzle, count)
	workerCount := int(math.Min(float64(count), float64(runtime.NumCPU())))

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			generator := NewSudokuGenerator(size, boxWidth, boxHeight, layoutType)
			generator.progressChan = progressChan
			for {
				puzzle, err := generator.createPuzzle(difficulty)
				if err == nil {
					puzzleChan <- puzzle
					if progressChan != nil {
						progressChan <- ProgressReport{
							Phase:    "generation",
							Progress: float64(len(puzzles)) / float64(count),
							Message:  fmt.Sprintf("Generated puzzle %d/%d", len(puzzles), count),
						}
					}
					return
				}
			}
		}(i)
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(puzzleChan)
	}()

	// Collect results
	for i := 0; i < count; i++ {
		puzzles[i] = <-puzzleChan
	}

	return puzzles
}

func getFactors(n int) []int {
	factors := make([]int, 0)
	for i := 1; i <= int(math.Sqrt(float64(n))); i++ {
		if n%i == 0 {
			factors = append(factors, i)
			if i != n/i {
				factors = append(factors, n/i)
			}
		}
	}
	sort.Ints(factors)
	return factors
}

func getValidBoxCombinations(size int) []struct {
	width, height int
	layoutType    string
} {
	factors := getFactors(size)
	combinations := make([]struct {
		width, height int
		layoutType    string
	}, 0)

	// Support jigsaw for common sizes
	if size == 9 || size == 12 || size == 16 {
		combinations = append(combinations, struct {
			width, height int
			layoutType    string
		}{size, 1, "jigsaw"})
	}

	for _, width := range factors {
		for _, height := range factors {
			if width*height == size && width > 1 && height > 1 {
				combinations = append(combinations, struct {
					width, height int
					layoutType    string
				}{width, height, "regular"})
			}
		}
	}
	return combinations
}

func generateJigsawRegions(size int) ([][]int, error) {
	return generateJigsawRegionsConcurrent(size, 1000000, nil)
}

func generateJigsawRegionsConcurrent(size int, maxAttempts int, progressChan chan<- ProgressReport) ([][]int, error) {
	attemptChan := make(chan [][]int, runtime.NumCPU())
	errorChan := make(chan error, runtime.NumCPU())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	workerCount := runtime.NumCPU()
	attemptsPerWorker := maxAttempts / workerCount
	var wg sync.WaitGroup

	var totalAttempts int64
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			seed := time.Now().UnixNano() + int64(workerID)
			rng := rand.New(rand.NewSource(seed))

			for attempt := 0; attempt < attemptsPerWorker; attempt++ {
				select {
				case <-ctx.Done():
					return
				default:
					atomic.AddInt64(&totalAttempts, 1)
					if regions, err := generateJigsawRegionsAttempt(size, rng); err == nil {
						attemptChan <- regions
						return
					}
					if attempt%100 == 0 && progressChan != nil {
						current := atomic.LoadInt64(&totalAttempts)
						progressChan <- ProgressReport{
							Phase:    "jigsaw",
							Progress: float64(current) / float64(maxAttempts),
							Message:  fmt.Sprintf("Workers: %d, Attempts: %d", workerCount, current),
						}
					}
				}
			}
			errorChan <- fmt.Errorf("worker %d failed after %d attempts", workerID, attemptsPerWorker)
		}(i)
	}

	// Wait for success or all failures
	go func() {
		wg.Wait()
		close(attemptChan)
		close(errorChan)
	}()

	select {
	case regions := <-attemptChan:
		if progressChan != nil {
			progressChan <- ProgressReport{
				Phase:     "jigsaw",
				Progress:  1.0,
				Message:   "Jigsaw generation successful",
				Completed: true,
			}
		}
		return regions, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout generating jigsaw regions")
	case err := <-errorChan:
		return nil, err
	}
}

func generateJigsawRegionsAttempt(size int, rng *rand.Rand) ([][]int, error) {
	maxAttempts := 1000000
	attempts := 0

	for attempts < maxAttempts {
		attempts++
		regions := make([][]int, size)
		cells := make(map[int]bool)
		adjacencies := make([][]int, size*size)

		// Build adjacency list
		for r := 0; r < size; r++ {
			for c := 0; c < size; c++ {
				index := r*size + c
				neighbors := make([]int, 0)
				if r > 0 {
					neighbors = append(neighbors, (r-1)*size+c)
				}
				if r < size-1 {
					neighbors = append(neighbors, (r+1)*size+c)
				}
				if c > 0 {
					neighbors = append(neighbors, r*size+(c-1))
				}
				if c < size-1 {
					neighbors = append(neighbors, r*size+(c+1))
				}
				adjacencies[index] = neighbors
			}
		}

		isValid := true
		// Generate each region
		for regionIndex := 0; regionIndex < size; regionIndex++ {
			var potentialStarts []int
			for i := 0; i < size*size; i++ {
				if !cells[i] {
					potentialStarts = append(potentialStarts, i)
				}
			}

			if len(potentialStarts) == 0 {
				isValid = false
				break
			}

			startCell := potentialStarts[rng.Intn(len(potentialStarts))]
			region := []int{startCell}
			cells[startCell] = true

			// Grow the region
			for len(region) < size {
				var candidates []int
				for _, cell := range region {
					for _, neighbor := range adjacencies[cell] {
						if !cells[neighbor] {
							candidates = append(candidates, neighbor)
						}
					}
				}

				if len(candidates) == 0 {
					isValid = false
					break
				}

				nextCell := candidates[rng.Intn(len(candidates))]
				region = append(region, nextCell)
				cells[nextCell] = true
			}

			if !isValid {
				break
			}
			regions[regionIndex] = region
		}

		if isValid && len(cells) == size*size {
			return regions, nil
		}
	}

	return nil, fmt.Errorf("failed to generate valid jigsaw regions after %d attempts", maxAttempts)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func printProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %.1f%%", bar, progress*100)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	progressChan := make(chan ProgressReport)

	// Track statistics for each generation
	stats := &GenerationStats{
		StartTime: time.Now(),
	}

	// Start progress monitoring goroutine with improved formatting
	go func() {
		lastLine := ""
		clearLine := func() {
			if len(lastLine) > 0 {
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(lastLine)))
			}
		}

		for progress := range progressChan {
			clearLine()
			var line string
			switch progress.Phase {
			case "jigsaw":
				elapsed := formatDuration(time.Since(stats.StartTime))
				line = fmt.Sprintf("\rGenerating jigsaw regions... %s - %s [%s elapsed]",
					printProgressBar(progress.Progress, 20),
					progress.Message,
					elapsed)
			case "generation":
				if progress.Message == "starting" {
					stats.StartTime = time.Now()
					line = "\nStarting puzzle generation..."
				} else {
					elapsed := formatDuration(time.Since(stats.StartTime))
					line = fmt.Sprintf("\rGenerating puzzles... %s - %s [%s elapsed]",
						printProgressBar(progress.Progress, 20),
						progress.Message,
						elapsed)
				}
			}
			fmt.Print(line)
			lastLine = line

			if progress.Completed {
				fmt.Println()
				if progress.Phase == "jigsaw" {
					fmt.Printf("✓ Jigsaw regions generated successfully in %s\n",
						formatDuration(time.Since(stats.StartTime)))
				}
			}
		}
	}()

	// Test different puzzle types
	testCases := []struct {
		size       int
		boxWidth   int
		boxHeight  int
		layoutType string
		count      int
	}{
		{9, 3, 3, "regular", 2},
		{9, 9, 1, "jigsaw", 2},
		{16, 4, 4, "regular", 1},
	}

	// Modify the test cases loop
	for _, tc := range testCases {
		fmt.Printf("\n━━━ Generating %d %dx%d %s Sudoku puzzles ━━━\n",
			tc.count, tc.size, tc.size, tc.layoutType)

		start := time.Now()
		puzzles := ConcurrentPuzzleGenerator(tc.size, tc.boxWidth, tc.boxHeight,
			tc.layoutType, tc.count, 0.5, progressChan)
		elapsed := time.Since(start)

		fmt.Printf("\n✓ Generated %d puzzles in %s\n", len(puzzles), formatDuration(elapsed))
		fmt.Printf("\nExample puzzle (%s layout):\n", tc.layoutType)
		printPuzzle(puzzles[0])

		// Print region information with improved formatting
		fmt.Printf("\nRegion information:\n")
		fmt.Printf("• Number of regions: %d\n", len(puzzles[0].Regions))
		fmt.Printf("• First region cells: %v\n", puzzles[0].Regions[0])
		fmt.Printf("• Average cells per region: %.1f\n",
			float64(tc.size*tc.size)/float64(len(puzzles[0].Regions)))

		fmt.Println("\n" + strings.Repeat("─", 50))
	}

	close(progressChan)
}

func printPuzzle(puzzle *SudokuPuzzle) {
	borderLine := strings.Repeat("─", puzzle.Size*2+4)
	fmt.Printf("┌%s┐\n", borderLine)

	for i := 0; i < puzzle.Size; i++ {
		fmt.Print("│ ")
		for j := 0; j < puzzle.Size; j++ {
			if puzzle.Grid[i*puzzle.Size+j] == 0 {
				fmt.Print("· ")
			} else {
				fmt.Printf("%d ", puzzle.Grid[i*puzzle.Size+j])
			}
		}
		fmt.Println("│")

		if i < puzzle.Size-1 && i%3 == 2 {
			fmt.Printf("├%s┤\n", borderLine)
		}
	}
	fmt.Printf("└%s┘\n", borderLine)
}

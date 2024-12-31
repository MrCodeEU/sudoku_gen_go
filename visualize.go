package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/MrCodeEU/sudoku_gen_go/types"
)

// Visualizer handles grid visualization
type Visualizer struct {
	grid *types.Grid
}

func NewVisualizer(grid *types.Grid) *Visualizer {
	return &Visualizer{grid: grid}
}

func (v *Visualizer) Print() {
	size := v.grid.Size

	// Print top border
	v.printHorizontalBorder(size)

	// Print rows
	for i := 0; i < size; i++ {
		fmt.Print("│ ")
		for j := 0; j < size; j++ {
			if v.grid.Cells[i][j] == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", v.grid.Cells[i][j])
			}

			// Print vertical borders
			if (j+1)%v.getBoxSize() == 0 && j < size-1 {
				fmt.Print("│ ")
			}
		}
		fmt.Println("│")

		// Print horizontal borders
		if (i+1)%v.getBoxSize() == 0 && i < size-1 {
			v.printHorizontalBorder(size)
		}
	}

	// Print bottom border
	v.printHorizontalBorder(size)
}

func (v *Visualizer) getBoxSize() int {
	return int(math.Sqrt(float64(v.grid.Size)))
}

func (v *Visualizer) printHorizontalBorder(size int) {
	boxSize := v.getBoxSize()
	fmt.Print("├")
	for i := 0; i < size; i++ {
		fmt.Print("──")
		if (i+1)%boxSize == 0 && i < size-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")
}

func (v *Visualizer) PrintJigsaw() {
	size := v.grid.Size

	// ANSI color codes for different regions
	colors := []string{
		"\033[41m",  // Red background
		"\033[42m",  // Green background
		"\033[43m",  // Yellow background
		"\033[44m",  // Blue background
		"\033[45m",  // Magenta background
		"\033[46m",  // Cyan background
		"\033[47m",  // White background
		"\033[100m", // Bright Black background
		"\033[101m", // Bright Red background
		"\033[102m", // Bright Green background
		"\033[103m", // Bright Yellow background
		"\033[104m", // Bright Blue background
		"\033[105m", // Bright Magenta background
		"\033[106m", // Bright Cyan background
		"\033[107m", // Bright White background
	}
	reset := "\033[0m"

	// Print top border
	fmt.Println("┌" + strings.Repeat("─", size*2+1) + "┐")

	// Print rows
	for i := 0; i < size; i++ {
		fmt.Print("│ ")
		for j := 0; j < size; j++ {
			cellIndex := i*size + j
			regionIndex := v.findRegionIndex(cellIndex)
			colorCode := colors[regionIndex%len(colors)]

			if v.grid.Cells[i][j] == 0 {
				fmt.Printf("%s %s%s", colorCode, ".", reset)
			} else {
				fmt.Printf("%s%d%s ", colorCode, v.grid.Cells[i][j], reset)
			}
		}
		fmt.Println("│")
	}

	// Print bottom border
	fmt.Println("└" + strings.Repeat("─", size*2+1) + "┘")
}

func (v *Visualizer) findRegionIndex(cellIndex int) int {
	for i, region := range v.grid.SubGrids {
		for _, cell := range region {
			if cell == cellIndex {
				return i
			}
		}
	}
	return -1
}
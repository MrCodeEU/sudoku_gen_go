package types

import "encoding/json"

type SudokuType string

const (
	Normal SudokuType = "normal"
	Jigsaw SudokuType = "jigsaw"
)

// Grid represents a flexible Sudoku grid
type Grid struct {
	Size      int        `json:"size"`
	BoxWidth  int        `json:"boxWidth"`
	BoxHeight int        `json:"boxHeight"`
	Puzzle    [][]int    `json:"grid"` // Renamed from Cells to match JS
	Solution  [][]int    `json:"solution"`
	SubGrids  [][]int    `json:"regions"`    // Renamed from SubGrids to match JS
	Type      SudokuType `json:"layoutType"` // Renamed from Type to match JS
}

// NewGrid creates a new Grid instance
func NewGrid(size int, typ SudokuType) *Grid {
	puzzle := make([][]int, size)
	solution := make([][]int, size)
	for i := range puzzle {
		puzzle[i] = make([]int, size)
		solution[i] = make([]int, size)
	}

	boxWidth, boxHeight := getBoxDimensions(size)

	return &Grid{
		Size:      size,
		BoxWidth:  boxWidth,
		BoxHeight: boxHeight,
		Puzzle:    puzzle,
		Solution:  solution,
		Type:      typ,
	}
}

func getBoxDimensions(size int) (width, height int) {
	switch size {
	case 9:
		return 3, 3
	case 12:
		return 3, 4
	case 16:
		return 4, 4
	default:
		return 3, 3
	}
}

// ToJSON converts the grid to JSON bytes
func (g *Grid) ToJSON() ([]byte, error) {
	return json.Marshal(g)
}

// FromJSON creates a Grid from JSON bytes
func FromJSON(data []byte) (*Grid, error) {
	var grid Grid
	err := json.Unmarshal(data, &grid)
	return &grid, err
}

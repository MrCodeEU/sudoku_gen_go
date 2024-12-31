package types

import "encoding/json"

type SudokuType string

const (
	Normal SudokuType = "normal"
	Jigsaw SudokuType = "jigsaw"
)

// Grid represents a flexible Sudoku grid
type Grid struct {
	Size       int        `json:"size"`
	Cells      [][]int    `json:"cells"`
	Type       SudokuType `json:"type"`
	SubGrids   [][]int    `json:"subgrids"`
	Solution   [][]int    `json:"solution"`
	Difficulty int        `json:"difficulty"`
}

// NewGrid creates a new Grid instance
func NewGrid(size int, typ SudokuType) *Grid {
	cells := make([][]int, size)
	for i := range cells {
		cells[i] = make([]int, size)
	}
	return &Grid{
		Size:  size,
		Type:  typ,
		Cells: cells,
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

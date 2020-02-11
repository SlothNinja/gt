package got

import "strconv"

type areas []*Area
type grid []areas

const (
	rowA int = iota
	rowB
	rowC
	rowD
	rowE
	rowF
	rowG
	noRow int = -1
)

var rowIDStrings = map[int]string{noRow: "None", rowA: "A", rowB: "B", rowC: "C",
	rowD: "D", rowE: "E", rowF: "F", rowG: "G"}

// RowString outputs a row label.
func (a *Area) RowString() string {
	return rowIDStrings[a.Row]
}

// RowIDString outputs an row id.
func (a *Area) RowIDString() string {
	return strconv.Itoa(a.Row)
}

const (
	col1 int = iota
	col2
	col3
	col4
	col5
	col6
	col7
	col8
	noCol int = -1
)

var columnIDStrings = map[int]string{noCol: "None", col1: "1", col2: "2", col3: "3", col4: "4",
	col5: "5", col6: "6", col7: "7", col8: "8"}

// ColString outputs a column label.
func (a *Area) ColString() string {
	return columnIDStrings[a.Column]
}

// ColIDString outputs an column id.
func (a *Area) ColIDString() string {
	return strconv.Itoa(a.Column)
}

// Area of the grid.
type Area struct {
	Row    int
	Column int
	Thief  int
	Card   *Card
}

// SelectedArea returns a previously selected area.
func (g *Game) SelectedArea() (a *Area) {
	if g.SelectedAreaF != nil {
		a = g.Grid[g.SelectedAreaF.Row][g.SelectedAreaF.Column]
	}
	return
}

// SelectedThiefArea returns the area corresponding to a previously selected thief.
func (g *Game) SelectedThiefArea() (a *Area) {
	if g.SelectedThiefAreaF != nil {
		a = g.Grid[g.SelectedThiefAreaF.Row][g.SelectedThiefAreaF.Column]
	}
	return
}

func newArea(row, col int, card *Card) *Area {
	return &Area{
		Row:    row,
		Column: col,
		Thief:  noPID,
		Card:   card,
	}
}

func (g *Game) lastRow() int {
	row := rowG
	if g.NumPlayers == 2 {
		row = rowF
	}
	return row
}

func (g *Game) createGrid() {
	deck := newDeck()
	g.Grid = make(grid, g.lastRow()+1)
	for row := 0; row < g.lastRow()+1; row++ {
		g.Grid[row] = make(areas, 8)
		for col := 0; col < 8; col++ {
			g.Grid[row][col] = newArea(row, col, deck.draw())
		}
	}
}

func (a *Area) hasThief() bool {
	return a.Thief != noPID
}

func (a *Area) hasCard() bool {
	return a.Card != nil
}

func (as areas) include(a2 *Area) bool {
	for _, a1 := range as {
		if a1.Row == a2.Row && a1.Column == a2.Column {
			return true
		}
	}
	return false
}

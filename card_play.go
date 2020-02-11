package got

import (
	"encoding/gob"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(playCardEntry))
}

func (g *Game) startCardPlay(ctx context.Context) (tmpl string, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = playCard
	g.Turn = 1
	return
}

func (g *Game) playCard(ctx context.Context) (tmpl string, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validatePlayCard(ctx); err != nil {
		tmpl = "got/flash_notice"
		return
	}

	cp := g.CurrentPlayer()
	card := cp.Hand.playCardAt(g.SelectedCardIndex)
	cp.DiscardPile = append(Cards{card}, cp.DiscardPile...)
	if card.Type == jewels {
		pc := g.Jewels
		g.PlayedCard = &pc
		g.JewelsPlayed = true
	} else {
		g.PlayedCard = card
	}

	// Log placement
	e := g.newPlayCardEntryFor(cp, card)
	restful.AddNoticef(ctx, string(e.HTML(g)))

	return g.startSelectThief(ctx)
}

func (g *Game) validatePlayCard(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err := g.validatePlayerAction(ctx); err != nil {
		return err
	}

	switch card := g.SelectedCard(); {
	case card == nil:
		return sn.NewVError("You must select a card.")
	default:
		return nil
	}
}

type playCardEntry struct {
	*Entry
	Type cType
}

func (g *Game) newPlayCardEntryFor(p *Player, c *Card) *playCardEntry {
	e := &playCardEntry{
		Entry: g.newEntryFor(p),
		Type:  c.Type,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *playCardEntry) HTML(g *Game) template.HTML {
	return restful.HTML("%s played %s card.", g.NameByPID(e.PlayerID), e.Type)
}

func (g *Game) isLampArea(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.lampAreas().include(a)
	}
	return
}

func (g *Game) lampAreas() areas {
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a1 := g.SelectedThiefArea()
	// Move Left
	var a2 *Area
	for col := a1.Column - 1; col >= col1; col-- {
		if temp := g.Grid[a1.Row][col]; !canMoveTo(temp) {
			break
		} else {
			a2 = temp
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move right
	a2 = nil
	for col := a1.Column + 1; col <= col8; col++ {
		if temp := g.Grid[a1.Row][col]; !canMoveTo(temp) {
			break
		} else {
			a2 = temp
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move Up
	a2 = nil
	for row := a1.Row - 1; row >= rowA; row-- {
		if temp := g.Grid[row][a1.Column]; !canMoveTo(temp) {
			break
		} else {
			a2 = temp
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move Down
	a2 = nil
	for row := a1.Row + 1; row <= g.lastRow(); row++ {
		if temp := g.Grid[row][a1.Column]; !canMoveTo(temp) {
			break
		} else {
			a2 = temp
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}
	g.ClickAreas = as
	return as
}

func (g *Game) isCamelArea(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.camelAreas().include(a)
	}
	return
}

func (g *Game) camelAreas() areas {
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a := g.SelectedThiefArea()

	// Move Three Left?
	if a.Column-3 >= col1 {
		area1 := g.Grid[a.Row][a.Column-1]
		area2 := g.Grid[a.Row][a.Column-2]
		area3 := g.Grid[a.Row][a.Column-3]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area3)
		}
	}

	// Move Three Right?
	if a.Column+3 <= col8 {
		area1 := g.Grid[a.Row][a.Column+1]
		area2 := g.Grid[a.Row][a.Column+2]
		area3 := g.Grid[a.Row][a.Column+3]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area3)
		}
	}

	// Move Three Up?
	if a.Row-3 >= rowA {
		area1 := g.Grid[a.Row-1][a.Column]
		area2 := g.Grid[a.Row-2][a.Column]
		area3 := g.Grid[a.Row-3][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area3)
		}
	}

	// Move Three Down?
	if a.Row+3 <= g.lastRow() {
		area1 := g.Grid[a.Row+1][a.Column]
		area2 := g.Grid[a.Row+2][a.Column]
		area3 := g.Grid[a.Row+3][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area3)
		}
	}

	// Move Two Left One Up or One Up Two Left or One Left One Up One Left?
	if a.Column-2 >= col1 && a.Row-1 >= rowA {
		area1 := g.Grid[a.Row][a.Column-1]
		area2 := g.Grid[a.Row][a.Column-2]
		area3 := g.Grid[a.Row-1][a.Column-2]
		area4 := g.Grid[a.Row-1][a.Column]
		area5 := g.Grid[a.Row-1][a.Column-1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move Two Left One Down or One Down Two Left or One Left One Down One Left?
	if a.Column-2 >= col1 && a.Row+1 <= g.lastRow() {
		area1 := g.Grid[a.Row][a.Column-1]
		area2 := g.Grid[a.Row][a.Column-2]
		area3 := g.Grid[a.Row+1][a.Column-2]
		area4 := g.Grid[a.Row+1][a.Column]
		area5 := g.Grid[a.Row+1][a.Column-1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move Two Right One Up or One Up Two Right or One Right One Up One Right?
	if a.Column+2 <= col8 && a.Row-1 >= rowA {
		area1 := g.Grid[a.Row][a.Column+1]
		area2 := g.Grid[a.Row][a.Column+2]
		area3 := g.Grid[a.Row-1][a.Column+2]
		area4 := g.Grid[a.Row-1][a.Column]
		area5 := g.Grid[a.Row-1][a.Column+1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move Two Right One Down or One Down Two Right or One Right One Down One Right?
	if a.Column+2 <= col8 && a.Row+1 <= g.lastRow() {
		area1 := g.Grid[a.Row][a.Column+1]
		area2 := g.Grid[a.Row][a.Column+2]
		area3 := g.Grid[a.Row+1][a.Column+2]
		area4 := g.Grid[a.Row+1][a.Column]
		area5 := g.Grid[a.Row+1][a.Column+1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move One Right Two Down or Two Down One Right or One Down One Right One Down?
	if a.Column+1 <= col8 && a.Row+2 <= g.lastRow() {
		area1 := g.Grid[a.Row+1][a.Column]
		area2 := g.Grid[a.Row+2][a.Column]
		area3 := g.Grid[a.Row+2][a.Column+1]
		area4 := g.Grid[a.Row][a.Column+1]
		area5 := g.Grid[a.Row+1][a.Column+1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move One Right Two Up or Two Up One Right or One Up One Right One Up?
	if a.Column+1 <= col8 && a.Row-2 >= rowA {
		area1 := g.Grid[a.Row-1][a.Column]
		area2 := g.Grid[a.Row-2][a.Column]
		area3 := g.Grid[a.Row-2][a.Column+1]
		area4 := g.Grid[a.Row][a.Column+1]
		area5 := g.Grid[a.Row-1][a.Column+1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move One Left Two Down or Two Down One Left or One Down One Left One Down?
	if a.Column-1 >= col1 && a.Row+2 <= g.lastRow() {
		area1 := g.Grid[a.Row+1][a.Column]
		area2 := g.Grid[a.Row+2][a.Column]
		area3 := g.Grid[a.Row+2][a.Column-1]
		area4 := g.Grid[a.Row][a.Column-1]
		area5 := g.Grid[a.Row+1][a.Column-1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move One Left Two Up or Two Up One Left or One Up One Left One Up?
	if a.Column-1 >= col1 && a.Row-2 >= rowA {
		area1 := g.Grid[a.Row-1][a.Column]
		area2 := g.Grid[a.Row-2][a.Column]
		area3 := g.Grid[a.Row-2][a.Column-1]
		area4 := g.Grid[a.Row][a.Column-1]
		area5 := g.Grid[a.Row-1][a.Column-1]
		if canMoveTo(area1, area2, area3) || canMoveTo(area3, area4, area5) || canMoveTo(area1, area5, area3) {
			as = append(as, area3)
		}
	}

	// Move One Left One Up One Right or One Up One Left One Down?
	if a.Column-1 >= col1 && a.Row-1 >= rowA {
		area1 := g.Grid[a.Row][a.Column-1]
		area2 := g.Grid[a.Row-1][a.Column-1]
		area3 := g.Grid[a.Row-1][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area1)
			as = append(as, area3)
		}
	}

	// Move One Up One Right One Down or One Right One Up One Left?
	if a.Column+1 <= col8 && a.Row-1 >= rowA {
		area1 := g.Grid[a.Row][a.Column+1]
		area2 := g.Grid[a.Row-1][a.Column+1]
		area3 := g.Grid[a.Row-1][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area1)
			as = append(as, area3)
		}
	}

	// Move One Left One Down One Right or One Down One Left One Up?
	if a.Column-1 >= col1 && a.Row+1 <= g.lastRow() {
		area1 := g.Grid[a.Row][a.Column-1]
		area2 := g.Grid[a.Row+1][a.Column-1]
		area3 := g.Grid[a.Row+1][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area1)
			as = append(as, area3)
		}
	}

	// Move One Down One Right One Up or One Right One Down One Left?
	if a.Column+1 <= col8 && a.Row+1 <= g.lastRow() {
		area1 := g.Grid[a.Row][a.Column+1]
		area2 := g.Grid[a.Row+1][a.Column+1]
		area3 := g.Grid[a.Row+1][a.Column]
		if canMoveTo(area1, area2, area3) {
			as = append(as, area1)
			as = append(as, area3)
		}
	}

	g.ClickAreas = as
	return as
}

func canMoveTo(as ...*Area) bool {
	for _, a := range as {
		if a.hasThief() || !a.hasCard() {
			return false
		}
	}
	return true
}

func (g *Game) isSwordArea(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.swordAreas().include(a)
	}
	return
}

func (g *Game) swordAreas() areas {
	cp := g.CurrentPlayer()
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a := g.SelectedThiefArea()

	// Move Left
	if area, row := a, a.Row; a.Column >= col3 {
		// Left as far as permitted
		for col := a.Column - 1; col >= col3; col-- {
			if temp := g.Grid[row][col]; !canMoveTo(temp) {
				break
			} else {
				area = temp
			}
		}

		// Check for Thief and Place to Bump
		moveTo, bumpTo := g.Grid[row][area.Column-1], g.Grid[row][area.Column-2]
		if cp.anotherThiefIn(moveTo) && canMoveTo(bumpTo) {
			as = append(as, moveTo)
		}
	}

	// Move Right
	if area, row := a, a.Row; a.Column <= col6 {
		// Right as far as permitted
		for col := a.Column + 1; col <= col6; col++ {
			if temp := g.Grid[row][col]; !canMoveTo(temp) {
				break
			} else {
				area = temp
			}
		}

		// Check for Thief and Place to Bump
		moveTo, bumpTo := g.Grid[row][area.Column+1], g.Grid[row][area.Column+2]
		if cp.anotherThiefIn(moveTo) && canMoveTo(bumpTo) {
			as = append(as, moveTo)
		}
	}

	// Move Up
	if area, col := a, a.Column; a.Row >= rowC {
		// Up as far as permitted
		for row := a.Row - 1; row >= rowC; row-- {
			if temp := g.Grid[row][col]; !canMoveTo(temp) {
				break
			} else {
				area = temp
			}
		}

		// Check for Thief and Place to Bump
		moveTo, bumpTo := g.Grid[area.Row-1][col], g.Grid[area.Row-2][col]
		if cp.anotherThiefIn(moveTo) && canMoveTo(bumpTo) {
			as = append(as, moveTo)
		}
	}

	// Move Down
	if area, col := a, a.Column; a.Row <= g.lastRow()-2 {
		// Down as far as permitted
		for row := a.Row + 1; row <= g.lastRow()-2; row++ {
			//g.debugf("Row: %v Col: %v", row, col)
			if temp := g.Grid[row][col]; !canMoveTo(temp) {
				break
			} else {
				area = temp
			}
		}

		// Check for Thief and Place to Bump
		moveTo, bumpTo := g.Grid[area.Row+1][col], g.Grid[area.Row+2][col]
		if cp.anotherThiefIn(moveTo) && canMoveTo(bumpTo) {
			as = append(as, moveTo)
		}
	}

	g.ClickAreas = as
	return as
}

func (p *Player) anotherThiefIn(a *Area) bool {
	return a.hasThief() && a.Thief != p.ID()
}

func (g *Game) isCarpetArea(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.carpetAreas().include(a)
	}
	return
}

func (g *Game) carpetAreas() areas {
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a1 := g.SelectedThiefArea()

	// Move Left
	var a2, empty *Area
MoveLeft:
	for col := a1.Column - 1; col >= col1; col-- {
		switch temp := g.Grid[a1.Row][col]; {
		case temp.Card == nil:
			empty = temp
		case empty != nil && canMoveTo(temp):
			a2 = temp
			break MoveLeft
		default:
			break MoveLeft
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move Right
	a2, empty = nil, nil
MoveRight:
	for col := a1.Column + 1; col <= col8; col++ {
		switch temp := g.Grid[a1.Row][col]; {
		case temp.Card == nil:
			empty = temp
		case empty != nil && canMoveTo(temp):
			a2 = temp
			break MoveRight
		default:
			break MoveRight
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move Up
	a2, empty = nil, nil
MoveUp:
	for row := a1.Row - 1; row >= rowA; row-- {
		switch temp := g.Grid[row][a1.Column]; {
		case temp.Card == nil:
			empty = temp
		case empty != nil && canMoveTo(temp):
			a2 = temp
			break MoveUp
		default:
			break MoveUp
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	// Move Down
	a2, empty = nil, nil
MoveDown:
	for row := a1.Row + 1; row <= g.lastRow(); row++ {
		switch temp := g.Grid[row][a1.Column]; {
		case temp.Card == nil:
			empty = temp
		case empty != nil && canMoveTo(temp):
			a2 = temp
			break MoveDown
		default:
			break MoveDown
		}
	}
	if a2 != nil {
		as = append(as, a2)
	}

	g.ClickAreas = as
	return as
}

func (g *Game) isTurban0Area(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.turban0Areas().include(a)
	}
	return
}

func (g *Game) turban0Areas() areas {
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a := g.SelectedThiefArea()

	// Move Left
	if col := a.Column - 1; col >= col1 {
		if area := g.Grid[a.Row][col]; canMoveTo(area) {
			// Left
			if col := col - 1; col >= col1 && canMoveTo(g.Grid[area.Row][col]) {
				as = append(as, area)
			}
			// Up
			if row := area.Row - 1; row >= rowA && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
			// Down
			if row := area.Row + 1; row <= g.lastRow() && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
		}
	}

	// Move Right
	if col := a.Column + 1; col <= col8 {
		if area := g.Grid[a.Row][col]; canMoveTo(area) {
			// Right
			if col := col + 1; col <= col8 && canMoveTo(g.Grid[area.Row][col]) {
				as = append(as, area)
			}
			// Up
			if row := area.Row - 1; row >= rowA && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
			// Down
			if row := area.Row + 1; row <= g.lastRow() && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
		}
	}

	// Move Up
	if row := a.Row - 1; row >= rowA {
		if area := g.Grid[row][a.Column]; canMoveTo(area) {
			// Left
			if col := area.Column - 1; col >= col1 && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
			// Right
			if col := area.Column + 1; col <= col8 && canMoveTo(g.Grid[area.Row][col]) {
				as = append(as, area)
			}
			// Up
			if row := row - 1; row >= rowA && canMoveTo(g.Grid[row][a.Column]) {
				as = append(as, area)
			}
		}
	}

	// Move Down
	if row := a.Row + 1; row <= g.lastRow() {
		if area := g.Grid[row][a.Column]; canMoveTo(area) {
			// Left
			if col := area.Column - 1; col >= col1 && canMoveTo(g.Grid[row][col]) {
				as = append(as, area)
			}
			// Right
			if col := area.Column + 1; col <= col8 && canMoveTo(g.Grid[area.Row][col]) {
				as = append(as, area)
			}
			// Down
			if row := row + 1; row <= g.lastRow() && canMoveTo(g.Grid[row][a.Column]) {
				as = append(as, area)
			}
		}
	}

	g.ClickAreas = as
	return as
}

func (g *Game) isTurban1Area(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.turban1Areas().include(a)
	}
	return
}

func (g *Game) turban1Areas() areas {
	if g.ClickAreas != nil {
		return g.ClickAreas
	}
	as := make(areas, 0)
	a := g.SelectedThiefArea()

	// Move Left
	if a.Column-1 >= col1 && canMoveTo(g.Grid[a.Row][a.Column-1]) {
		as = append(as, g.Grid[a.Row][a.Column-1])
	}

	// Move Right
	if a.Column+1 <= col8 && canMoveTo(g.Grid[a.Row][a.Column+1]) {
		as = append(as, g.Grid[a.Row][a.Column+1])
	}

	// Move Up
	if a.Row-1 >= rowA && canMoveTo(g.Grid[a.Row-1][a.Column]) {
		as = append(as, g.Grid[a.Row-1][a.Column])
	}

	// Move Down
	if a.Row+1 <= g.lastRow() && canMoveTo(g.Grid[a.Row+1][a.Column]) {
		as = append(as, g.Grid[a.Row+1][a.Column])
	}

	g.ClickAreas = as
	return as
}

func (g *Game) isCoinsArea(a *Area) (b bool) {
	if g.SelectedThiefArea() != nil {
		b = g.coinsAreas().include(a)
	}
	return
}

func (g *Game) coinsAreas() areas {
	return g.turban1Areas()
}

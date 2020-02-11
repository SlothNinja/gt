package got

import (
	"strings"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
)

type cType int

const (
	noType cType = iota
	lamp
	camel
	sword
	carpet
	coins
	turban
	jewels
	guard
	sCamel
	sLamp
)

func (g *Game) cardTypes() []cType {
	return []cType{
		lamp,
		camel,
		sword,
		carpet,
		coins,
		turban,
		jewels,
		guard,
		sCamel,
		sLamp,
	}
}

var ctypeStrings = map[cType]string{
	noType: "None",
	lamp:   "Lamp",
	camel:  "Camel",
	sword:  "Sword",
	carpet: "Carpet",
	coins:  "Coins",
	turban: "Turban",
	jewels: "Jewels",
	guard:  "Guard",
	sCamel: "Camel",
	sLamp:  "Lamp",
}

var stringsCType = map[string]cType{
	"none":        noType,
	"lamp":        lamp,
	"camel":       camel,
	"sword":       sword,
	"carpet":      carpet,
	"coins":       coins,
	"turban":      turban,
	"jewels":      jewels,
	"guard":       guard,
	"start-camel": sCamel,
	"start-lamp":  sLamp,
}

func toCType(s string) (t cType) {
	s = strings.ToLower(s)

	var ok bool
	if t, ok = stringsCType[s]; !ok {
		t = noType
	}
	return
}

var ctypeValues = map[cType]int{
	noType: 0,
	lamp:   1,
	camel:  4,
	sword:  5,
	carpet: 3,
	coins:  3,
	turban: 2,
	jewels: 2,
	guard:  -1,
	sCamel: 0,
	sLamp:  0,
}

func (t cType) String() string {
	return ctypeStrings[t]
}

func (t cType) LString() string {
	return strings.ToLower(t.String())
}

func (t cType) IDString() string {
	switch t {
	case sCamel:
		return "start-camel"
	case sLamp:
		return "start-lamp"
	default:
		return t.LString()
	}
}

// Card is a playing card used to form grid, player's hand, and player's deck.
type Card struct {
	Type   cType
	FaceUp bool
}

func newCard(t cType, f bool) *Card {
	return &Card{
		Type:   t,
		FaceUp: f,
	}
}

// Cards is a slice of cards used to form player's hand or deck.
type Cards []*Card

func newDeck() Cards {
	deck := make(Cards, 64)
	for j := 0; j < 8; j++ {
		for i, typ := range []cType{lamp, camel, sword, carpet, coins, turban, jewels, guard} {
			deck[i+j*8] = &Card{Type: typ}
		}
	}
	return deck
}

func (cs Cards) removeAt(i int) Cards {
	return append(cs[:i], cs[i+1:]...)
}

func (cs *Cards) playCardAt(i int) *Card {
	card := (*cs)[i]
	*cs = cs.removeAt(i)
	return card
}

func (cs *Cards) draw() *Card {
	var card *Card
	*cs, card = cs.drawS()
	return card
}

func (cs Cards) drawS() (Cards, *Card) {
	i := sn.MyRand.Intn(len(cs))
	card := cs[i]
	cards := cs.removeAt(i)
	return cards, card
}

func (cs *Cards) append(cards ...*Card) {
	*cs = cs.appendS(cards...)
}

func (cs Cards) appendS(cards ...*Card) Cards {
	if len(cards) == 0 {
		return cs
	}
	return append(cs, cards...)
}

// IDString outputs a card id.
func (c Card) IDString() string {
	return c.Type.IDString()
}

func newStartHand() Cards {
	return Cards{newCard(sLamp, true), newCard(sLamp, true), newCard(sCamel, true)}
}

var toolTipStrings = map[cType]string{
	noType: "None",
	lamp:   "Move in a straight line until coming to the edge of the grid, an empty space, or another Thief.",
	camel:  "Move exactly 3 spaces in any direction. The spaces do not have to be in a straight line, but you cannot move over the same space twice.",
	sword:  "Move in a straight line until you come to another player's thief. Bump that thief to the next card and place your thief on the vacated card.",
	carpet: "Move in a straight line over at least one empty space.  Stop moving your thief on the first card after the empty space(s).",
	coins:  "Move one space and then draw an additional card during the draw step. Your hand size is permanently increased by 1.",
	turban: "Move two spaces. Claim the first Magic Item you pass over in addition to the card you claim in the Claim Magic Item step.",
	jewels: "Move as if you played the card that was last played by an opponent.",
	guard:  "This card cannot be played and does nothing for you in your hand.",
	sCamel: "Move exactly 3 spaces in any direction. The spaces do not have to be in a straight line, but you cannot move over the same space twice.",
	sLamp:  "Move in a straight line until coming to the edge of the grid, an empty space, or another Thief.",
}

// ToolTip outputs a description of the cards ability.
func (c Card) ToolTip() string {
	return c.Type.toolTip()
}

func (t cType) toolTip() string {
	return toolTipStrings[t]
}

// SelectedCard provides a previously selected card.
func (g *Game) SelectedCard() (c *Card) {
	if cp := g.CurrentPlayer(); g.SelectedCardIndex >= 0 && g.SelectedCardIndex < len(cp.Hand) {
		c = cp.Hand[g.SelectedCardIndex]
	}
	return
}

// Value provides the point value of a card.
func (c *Card) Value() int {
	return ctypeValues[c.Type]
}

package got

import (
	"encoding/gob"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(drawCardEntry))
}

func (g *Game) drawCard(ctx context.Context) (tmp string, err error) {
	g.Phase = drawCard
	cp := g.CurrentPlayer()

	if g.Turn != 1 {
		card, shuffle := cp.draw()
		e := g.newDrawCardEntryFor(cp, card, shuffle)
		restful.AddNoticef(ctx, string(e.HTML(g)))
		if g.PlayedCard.Type == coins {
			card, shuffle := cp.draw()
			e := g.newDrawCardEntryFor(cp, card, shuffle)
			restful.AddNoticef(ctx, string(e.HTML(g)))
		}
	}
	cp.PerformedAction = true
	return "got/move_thief_update", nil
}

type drawCardEntry struct {
	*Entry
	Card    Card
	Shuffle bool
}

func (g *Game) newDrawCardEntryFor(p *Player, c *Card, shuffle bool) *drawCardEntry {
	e := &drawCardEntry{
		Entry:   g.newEntryFor(p),
		Card:    *c,
		Shuffle: shuffle,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *drawCardEntry) HTML(g *Game) (t template.HTML) {
	n := g.NameByPID(e.PlayerID)
	if e.Shuffle {
		t = restful.HTML("%s shuffled discard pile and drew card from newly formed draw pile.", n)
	} else {
		t = restful.HTML("%s drew card from draw pile.", n)
	}
	return
}

func (p *Player) draw() (*Card, bool) {
	shuffle := false
	if len(p.DrawPile) == 0 {
		shuffle = true
		p.DrawPile = p.DiscardPile
		for _, card := range p.DrawPile {
			card.FaceUp = false
		}
		p.DiscardPile = make(Cards, 0)
	}
	card := p.DrawPile.draw()
	p.Hand.append(card)
	return card, shuffle
}

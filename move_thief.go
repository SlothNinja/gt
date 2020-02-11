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
	gob.Register(new(moveThiefEntry))
}

func (g *Game) startMoveThief(ctx context.Context) (tmpl string, err error) {
	g.Phase = moveThief
	return "got/select_thief_update", nil
}

func (g *Game) moveThief(ctx context.Context) (tmpl string, err error) {
	if err = g.validateMoveThief(ctx); err != nil {
		tmpl = "got/flash_notice"
		return
	}

	cp := g.CurrentPlayer()
	e := g.newMoveThiefEntryFor(cp)
	restful.AddNoticef(ctx, string(e.HTML(g)))

	switch {
	case g.PlayedCard.Type == sword:
		g.BumpedPlayerID = g.SelectedArea().Thief
		bumpedTo := g.bumpedTo(g.SelectedThiefArea(), g.SelectedArea())
		bumpedTo.Thief = g.BumpedPlayerID
		g.BumpedPlayer().Score += bumpedTo.Card.Value() - g.SelectedArea().Card.Value()
	case g.PlayedCard.Type == turban && g.Stepped == 0:
		g.Stepped = 1
	case g.PlayedCard.Type == turban && g.Stepped == 1:
		g.Stepped = 2
	}
	g.SelectedArea().Thief = cp.ID()
	cp.Score += g.SelectedArea().Card.Value()
	return g.claimItem(ctx)
}

func (g *Game) validateMoveThief(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	a := g.SelectedArea()
	switch err = g.validatePlayerAction(ctx); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("You must select a space which to move your thief.")
	case g.SelectedThiefArea() != nil && g.SelectedThiefArea().Thief != g.CurrentPlayer().ID():
		err = sn.NewVError("You must first select one of your thieves.")
	case (g.PlayedCard.Type == lamp || g.PlayedCard.Type == sLamp) && !g.isLampArea(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case (g.PlayedCard.Type == camel || g.PlayedCard.Type == sCamel) && !g.isCamelArea(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == coins && !g.isCoinsArea(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == sword && !g.isSwordArea(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == carpet && !g.isCarpetArea(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == turban && g.Stepped == 0 && !g.isTurban0Area(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == turban && g.Stepped == 1 && !g.isTurban1Area(a):
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	case g.PlayedCard.Type == guard:
		err = sn.NewVError("You can't move the selected thief to area %s%s", a.Row, a.Column)
	}
	return
}

type moveThiefEntry struct {
	*Entry
	Card Card
	From Area
	To   Area
}

func (g *Game) newMoveThiefEntryFor(p *Player) (e *moveThiefEntry) {
	e = &moveThiefEntry{
		Entry: g.newEntryFor(p),
		Card:  *(g.PlayedCard),
		From:  *(g.SelectedThiefArea()),
		To:    *(g.SelectedArea()),
	}
	if g.JewelsPlayed {
		e.Card = *(newCard(jewels, true))
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *moveThiefEntry) HTML(g *Game) (t template.HTML) {
	from := e.From
	to := e.To
	n := g.NameByPID(e.PlayerID)
	if e.Card.Type == sword {
		bumped := g.bumpedTo(&from, &to)
		t = restful.HTML("%s moved thief from %s card at %s%s to %s card at %s%s and bumped thief to card at %s%s.",
			n, from.Card.Type, from.RowString(), from.ColString(), to.Card.Type,
			to.RowString(), to.ColString(), bumped.RowString(), bumped.ColString())
	} else {
		t = restful.HTML("%s moved thief from %s card at %s%s to %s card at %s%s.", n,
			from.Card.Type, from.RowString(), from.ColString(), to.Card.Type, to.RowString(),
			to.ColString())
	}
	return
}

func (g *Game) bumpedTo(from, to *Area) *Area {
	switch {
	case from.Row > to.Row:
		return g.Grid[to.Row-1][from.Column]
	case from.Row < to.Row:
		return g.Grid[to.Row+1][from.Column]
	case from.Column > to.Column:
		return g.Grid[from.Row][to.Column-1]
	case from.Column < to.Column:
		return g.Grid[from.Row][to.Column+1]
	default:
		return nil
	}
}

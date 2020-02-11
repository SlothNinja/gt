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
	gob.Register(new(placeThiefEntry))
}

func (g *Game) placeThieves(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = placeThieves
	return nil
}

func (g *Game) placeThief(ctx context.Context) (tmpl string, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validatePlaceThief(ctx); err != nil {
		tmpl = "got/flash_notice"
		return
	}
	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	cp.Score += g.SelectedArea().Card.Value()
	g.SelectedArea().Thief = cp.ID()

	// Log placement
	e := g.newPlaceThiefEntryFor(cp)
	restful.AddNoticef(ctx, string(e.HTML(g)))
	return "got/place_thief_update", nil
}

func (g *Game) validatePlaceThief(ctx context.Context) error {
	if err := g.validatePlayerAction(ctx); err != nil {
		return err
	}

	//g.debugf("Place Thief Area: %#v", g.SelectedArea)

	switch area := g.SelectedArea(); {
	case area == nil:
		return sn.NewVError("You must select an area.")
	case area.Card == nil:
		return sn.NewVError("You must select an area with a card.")
	case area.Thief != noPID:
		return sn.NewVError("You must select an area without a thief.")
	default:
		return nil
	}
}

type placeThiefEntry struct {
	*Entry
	Area Area
}

func (g *Game) newPlaceThiefEntryFor(p *Player) (e *placeThiefEntry) {
	area := g.SelectedArea()
	e = &placeThiefEntry{
		Entry: g.newEntryFor(p),
		Area:  *area,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placeThiefEntry) HTML(g *Game) template.HTML {
	return restful.HTML("%s placed thief on %s at %s%s.",
		g.NameByPID(e.PlayerID), e.Area.Card.Type, e.Area.RowString(), e.Area.ColString())
}

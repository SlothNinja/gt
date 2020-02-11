package got

import (
	"encoding/gob"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(passEntry))
}

func (g *Game) pass(ctx context.Context) (string, game.ActionType, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err := g.validatePass(ctx); err != nil {
		return "got/flash_notice", game.None, err
	}

	cp := g.CurrentPlayer()
	cp.Passed = true
	cp.PerformedAction = true
	g.Phase = drawCard

	// Log Pass
	e := g.newPassEntryFor(cp)
	restful.AddNoticef(ctx, string(e.HTML(g)))

	return "got/pass_update", game.Cache, nil
}

func (g *Game) validatePass(ctx context.Context) error {
	if err := g.validatePlayerAction(ctx); err != nil {
		return err
	}
	return nil
}

type passEntry struct {
	*Entry
}

func (g *Game) newPassEntryFor(p *Player) (e *passEntry) {
	e = &passEntry{
		Entry: g.newEntryFor(p),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *passEntry) HTML(g *Game) template.HTML {
	return restful.HTML("%s passed.", g.NameByPID(e.PlayerID))
}

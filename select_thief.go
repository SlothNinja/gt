package got

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"golang.org/x/net/context"
)

func (g *Game) startSelectThief(ctx context.Context) (tmpl string, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = selectThief
	return "got/played_card_update", nil
}

func (g *Game) selectThief(ctx context.Context) (tmpl string, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validateSelectThief(ctx); err != nil {
		tmpl = "got/flash_notice"
	} else {
		g.SelectedThiefAreaF = g.SelectedArea()
		tmpl, err = g.startMoveThief(ctx)
	}
	return
}

func (g *Game) validateSelectThief(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch area, err := g.SelectedArea(), g.validatePlayerAction(ctx); {
	case err != nil:
		return err
	case area == nil || area.Thief != g.CurrentPlayer().ID():
		return sn.NewVError("You must select one of your thieves.")
	default:
		return nil
	}
}

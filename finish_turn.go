package got

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

func finish(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))

		g := gameFrom(ctx)
		switch g.Phase {
		case placeThieves:
			if err := g.placeThievesFinishTurn(ctx); err != nil {
				log.Errorf(ctx, "g.placeThievesFinishTurn error: %v", err)
				return
			}

		case drawCard:
			if err := g.moveThiefFinishTurn(ctx); err != nil {
				log.Errorf(ctx, "g.moveThiefFinishTurn error: %v", err)
				return
			}

		}
	}
}

func showPath(prefix string, sid string) string {
	return fmt.Sprintf("/%s/game/show/%s", prefix, sid)
}

func (g *Game) validateFinishTurn(ctx context.Context) (*stats.Stats, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	switch cp, s := g.CurrentPlayer(), stats.Fetched(ctx); {
	case s == nil:
		return nil, sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(ctx):
		return nil, sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		return nil, sn.NewVError("%s has yet to perform an action.", g.NameFor(cp))
	default:
		return s, nil
	}
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) previousPlayer(ps ...game.Playerer) *Player {
	if nper := g.PreviousPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (g *Game) placeThievesNextPlayer(pers ...game.Playerer) (p *Player) {
	numThieves := 3
	if g.TwoThiefVariant {
		numThieves = 2
	}

	p = g.previousPlayer(pers...)

	if g.Round >= numThieves {
		p = nil
	} else if p.Equal(g.Players()[0]) {
		g.Round++
		p.beginningOfTurnReset()
	}
	return
}

func (g *Game) placeThievesFinishTurn(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	s, err := g.validatePlaceThievesFinishTurn(ctx)
	if err != nil {
		return err
	}

	oldCP := g.CurrentPlayer()
	np := g.placeThievesNextPlayer()
	if np == nil {
		g.SetCurrentPlayerers(g.Players()[0])
		g.CurrentPlayer().beginningOfTurnReset()
		g.startCardPlay(ctx)
	} else {
		g.SetCurrentPlayerers(np)
		np.beginningOfTurnReset()
	}

	newCP := g.CurrentPlayer()
	if newCP != nil && oldCP.ID() != newCP.ID() {
		g.SendTurnNotificationsTo(ctx, newCP)
	}

	return g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt)))
}

func (g *Game) validatePlaceThievesFinishTurn(ctx context.Context) (*stats.Stats, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	switch s, err := g.validateFinishTurn(ctx); {
	case err != nil:
		return nil, err
	case g.Phase != placeThieves:
		return nil, sn.NewVError("Expected %q phase but have %q phase.", placeThieves, g.Phase)
	default:
		return s, nil
	}
}

func (g *Game) moveThiefNextPlayer(pers ...game.Playerer) (np *Player) {
	cp := g.CurrentPlayer()
	g.endOfTurnUpdateFor(cp)
	ps := g.Players()
	np = g.nextPlayer(pers...)
	for !ps.allPassed() {
		if np.Passed {
			np = g.nextPlayer(np)
		} else {
			np.beginningOfTurnReset()
			return
		}
	}
	np = nil
	return
}

func (g *Game) moveThiefFinishTurn(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	var s *stats.Stats
	if s, err = g.validateMoveThiefFinishTurn(ctx); err != nil {
		return
	}

	oldCP := g.CurrentPlayer()
	np := g.moveThiefNextPlayer()

	// If no next player, end game
	if np == nil {
		g.finalClaim(ctx)
		ps := g.endGame(ctx)
		cs := contest.GenContests(ctx, ps)
		g.Status = game.Completed
		g.Phase = gameOver

		// Need to call SendTurnNotificationsTo before saving the new contests
		// SendEndGameNotifications relies on pulling the old contests from the db.
		// Saving the contests resulting in double counting.
		if err = g.sendEndGameNotifications(ctx, ps, cs); err != nil {
			log.Warningf(ctx, err.Error())
			err = nil
		}

		es := make([]interface{}, len(cs)+1)
		es[0] = s.GetUpdate(ctx, time.Time(g.UpdatedAt))
		for i, c := range cs {
			es[i+1] = c
		}

		if err = g.save(ctx, es...); err != nil {
			return
		}

		return
	}

	// Otherwise, select next player and continue moving theives.
	g.SetCurrentPlayerers(np)
	if np.Equal(g.Players()[0]) {
		g.Turn++
	}
	g.Phase = playCard

	if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
		if err = g.SendTurnNotificationsTo(ctx, newCP); err != nil {
			log.Warningf(ctx, err.Error())
		}
	}
	return g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt)))
}

func (g *Game) validateMoveThiefFinishTurn(ctx context.Context) (*stats.Stats, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	switch s, err := g.validateFinishTurn(ctx); {
	case err != nil:
		return nil, err
	case g.Phase != drawCard:
		return nil, sn.NewVError(`Expected "Draw Card" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

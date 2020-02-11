package got

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"github.com/gin-gonic/gin/binding"
	"go.chromium.org/gae/service/datastore"
	"golang.org/x/net/context"
)

const kind = "Game"

// New creates a new Guild of Thieves game.
func New(ctx context.Context) *Game {
	g := new(Game)
	g.Header = game.NewHeader(ctx, g)
	g.State = newState()
	g.Parent = pk(ctx)
	g.Type = gType.GOT
	return g
}

func newState() *State {
	return &State{TempData: new(TempData)}
}

func pk(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, gType.GOT.SString(), "root", 0, game.GamesRoot(ctx))
}

func (g *Game) init(ctx context.Context) (err error) {
	if err = g.Header.AfterLoad(g); err == nil {
		for _, player := range g.Players() {
			player.init(g)
		}
	}

	//	for _, entry := range g.Log {
	//		entry.Init(g)
	//	}
	return
}

func (g *Game) afterCache() error {
	return g.init(g.CTX())
}

func (g *Game) options() (s string) {
	if g.TwoThiefVariant {
		s = "Two Thief Variant"
	}
	return
}

func (g *Game) fromForm(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	s := new(State)
	if err = restful.BindWith(ctx, s, binding.FormPost); err == nil {
		g.TwoThiefVariant = s.TwoThiefVariant
	}
	log.Debugf(ctx, "err: %v s:%#v", err, s)
	return
}

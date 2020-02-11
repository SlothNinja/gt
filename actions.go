package got

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"golang.org/x/net/context"
)

func (g *Game) validatePlayerAction(ctx context.Context) (err error) {
	if !g.CUserIsCPlayerOrAdmin(ctx) {
		err = sn.NewVError("Only the current player can perform an action.")
	}
	return
}

func (g *Game) validateAdminAction(ctx context.Context) (err error) {
	if !user.IsAdmin(ctx) {
		err = sn.NewVError("Only an admin can perform the selected action.")
	}
	return
}

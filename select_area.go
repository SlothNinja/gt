package got

import (
	"strconv"
	"strings"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"golang.org/x/net/context"
)

func (g *Game) selectArea(ctx context.Context) (string, game.ActionType, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err := g.validateSelectArea(ctx); err != nil {
		return "got/flash_notice", game.None, err
	}

	switch cp := g.CurrentPlayer(); {
	case g.Admin == "admin-header":
		return "got/admin/header_dialog", game.Cache, nil
	case g.Admin == "admin-player-row-0":
		g.SelectedPlayerID = 0
		return "got/admin/player_dialog", game.Cache, nil
	case g.Admin == "admin-player-row-1":
		g.SelectedPlayerID = 1
		return "got/admin/player_dialog", game.Cache, nil
	case g.Admin == "admin-player-row-2":
		g.SelectedPlayerID = 2
		return "got/admin/player_dialog", game.Cache, nil
	case g.Admin == "admin-player-row-3":
		g.SelectedPlayerID = 3
		return "got/admin/player_dialog", game.Cache, nil
	case g.CanPlaceThief(ctx, cp):
		template, err := g.placeThief(ctx)
		return template, game.Cache, err
	case g.CanSelectCard(ctx, cp):
		template, err := g.playCard(ctx)
		return template, game.Cache, err
	case g.CanSelectThief(ctx, cp):
		template, err := g.selectThief(ctx)
		return template, game.Cache, err
	case g.CanMoveThief(ctx, cp):
		template, err := g.moveThief(ctx)
		return template, game.Cache, err
	default:
		return "got/flash_notice", game.None, sn.NewVError("Can't find action for selection.")
	}
}

func (g *Game) validateSelectArea(ctx context.Context) (err error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(ctx) {
		err = sn.NewVError("Only the current player can perform an action.")
	} else if !user.IsAdmin(ctx) && cp != nil && !g.CanPlaceThief(ctx, cp) && !g.CanSelectCard(ctx, cp) && !g.CanSelectThief(ctx, cp) && !g.CanMoveThief(ctx, cp) {
		err = sn.NewVError("You can't select an area right now.")
	}

	if err != nil {
		return
	}

	areaID := restful.GinFrom(ctx).PostForm("area")
	switch splits := strings.Split(areaID, "-"); splits[0] {
	case "admin":
		g.Admin = areaID
	case "area":
		var row, col int
		if row, err = strconv.Atoi(splits[1]); err == nil {
			col, err = strconv.Atoi(splits[2])
		}

		switch {
		case err != nil:
		case row < rowA:
			err = sn.NewVError("Row too small")
		case row > rowG:
			err = sn.NewVError("Row too large")
		case g.NumPlayers == 2 && row > rowF:
			err = sn.NewVError("Row too large")
		case col < col1:
			err = sn.NewVError("Column too small")
		case col > col8:
			err = sn.NewVError("Column too large")
		default:
			g.SelectedAreaF = g.Grid[row][col]
		}
	case "card":
		if cardType := toCType(strings.TrimPrefix(areaID, "card-")); cardType == noType {
			err = sn.NewVError("Received invalid card type.")
		} else {
			for i, card := range cp.Hand {
				if card.Type == cardType {
					g.SelectedCardIndex = i
					return
				}
			}
			err = sn.NewVError("You don't have a %q card to play.", cardType)
		}
	default:
		err = sn.NewVError("Unable to determine selection.")
	}
	return
}

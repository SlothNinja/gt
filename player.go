package got

import (
	"encoding/gob"
	"html/template"
	"sort"

	"bitbucket.org/SlothNinja/slothninja-games/sn/color"
	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/schema"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"go.chromium.org/gae/service/datastore"
	"golang.org/x/net/context"
)

func init() {
	gob.RegisterName("GOTPlayer", newPlayer())
}

// Player represents one of the players of the game.
type Player struct {
	*game.Player
	Log         GameLog
	Hand        Cards
	DrawPile    Cards
	DiscardPile Cards
}

// Players is a slice of players of the game.
type Players []*Player

func (ps Players) allPassed() bool {
	for _, p := range ps {
		if !p.Passed {
			return false
		}
	}
	return true
}

// Len is part of the sort.Interface interface
func (ps Players) Len() int { return len(ps) }

// Swap is part of the sort.Interface interface
func (ps Players) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }

// ByScore enables sorting players by their score.
type ByScore struct{ Players }

// Less defines when a player has a lower score than another player.
func (bs ByScore) Less(i, j int) bool {
	return bs.Players[i].compareByScore(bs.Players[j]) == game.LessThan
}

func (p *Player) compareByScore(p2 *Player) (result game.Comparison) {
	if byScore := p.CompareByScore(p2.Player); byScore != game.EqualTo {
		result = byScore
	} else if byLamps := p.compareByLamps(p2); byLamps != game.EqualTo {
		result = byLamps
	} else if byCamels := p.compareByCamels(p2); byCamels != game.EqualTo {
		result = byCamels
	} else {
		result = p.compareByCards(p2)
	}
	return
}

func (p *Player) compareByLamps(p2 *Player) game.Comparison {
	switch c0, c1 := lampCount(p.Hand...), lampCount(p2.Hand...); {
	case c0 < c1:
		return game.LessThan
	case c0 > c1:
		return game.GreaterThan
	}
	return game.EqualTo
}

// CountFor provides the number of faceUp and faceDown cards a player has.
func (cs Cards) CountFor(t cType) (faceUp, faceDown int) {
	for _, c := range cs {
		switch {
		case c.Type == t && c.FaceUp:
			faceUp++
		case c.Type == t && !c.FaceUp:
			faceDown++
		}
	}
	return
}

func lampCount(cs ...*Card) (count int) {
	for _, c := range cs {
		if c.Type == lamp || c.Type == sLamp {
			count++
		}
	}
	return count
}

func (p *Player) compareByCamels(p2 *Player) game.Comparison {
	switch c0, c1 := camelCount(p.Hand...), camelCount(p2.Hand...); {
	case c0 < c1:
		return game.LessThan
	case c0 > c1:
		return game.GreaterThan
	}
	return game.EqualTo
}

func camelCount(cs ...*Card) (count int) {
	for _, c := range cs {
		if c.Type == camel || c.Type == sCamel {
			count++
		}
	}
	return count
}

func (p *Player) compareByCards(p2 *Player) game.Comparison {
	switch c0, c1 := len(p.Hand), len(p2.Hand); {
	case c0 < c1:
		return game.LessThan
	case c0 > c1:
		return game.GreaterThan
	}
	return game.EqualTo
}

//func (g *Game) determinePlaces() []Players {
//	// sort players by score
//	players := g.Players()
//	sort.Sort(Reverse{ByScore{players}})
//	g.setPlayers(players)
//
//	places := make([]Players, 0)
//	for _, p := range g.Players() {
//		places = append(places, Players{p})
//	}
//	return places
//}

func (g *Game) determinePlaces(ctx context.Context) contest.Places {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	// sort players by score
	players := g.Players()
	sort.Sort(Reverse{ByScore{players}})
	g.setPlayers(players)

	places := make(contest.Places, 0)
	rmap := make(contest.ResultsMap, 0)
	for i, p1 := range g.Players() {
		results := make(contest.Results, 0)
		tie := false
		for j, p2 := range g.Players() {
			result := &contest.Result{
				GameID: g.ID,
				Type:   g.Type,
				R:      p2.Rating().R,
				RD:     p2.Rating().RD,
			}
			switch c := p1.compareByScore(p2); {
			case i == j:
			case c == game.GreaterThan:
				result.Outcome = 1
				results = append(results, result)
			case c == game.LessThan:
				result.Outcome = 0
				results = append(results, result)
			case c == game.EqualTo:
				result.Outcome = 0.5
				results = append(results, result)
				tie = true
			}
		}
		rmap[datastore.KeyForObj(ctx, p1.User())] = results
		if !tie {
			places = append(places, rmap)
			rmap = make(contest.ResultsMap, 0)
		} else if i == len(g.Players())-1 {
			places = append(places, rmap)
		}
	}
	return places
}

// Reverse is a wrapper for sorting in reverse order.
type Reverse struct{ sort.Interface }

// Less indicates if one element should preceed another.
func (r Reverse) Less(i, j int) bool { return r.Interface.Less(j, i) }

//var NotFound = errors.New("Not Found")

func (p *Player) init(gr game.Gamer) {
	p.SetGame(gr)

	//	g, ok := gr.(*Game)
	//	if !ok {
	//		return
	//	}

	//	for _, entry := range p.Log {
	//		entry.Init(g)
	//	}
}

func newPlayer() *Player {
	p := &Player{
		Hand:        newStartHand(),
		DrawPile:    make(Cards, 0),
		DiscardPile: make(Cards, 0),
	}
	p.Player = game.NewPlayer()
	return p
}

func (g *Game) addNewPlayer(u *user.User) {
	p := createPlayer(g, u)
	g.Playerers = append(g.Playerers, p)
}

func createPlayer(g *Game, u *user.User) *Player {
	p := newPlayer()
	p.SetID(int(len(g.Players())))
	p.SetGame(g)

	colorMap := g.DefaultColorMap()
	p.SetColorMap(make(color.Colors, g.NumPlayers))

	for i := 0; i < g.NumPlayers; i++ {
		index := (i - p.ID()) % g.NumPlayers
		if index < 0 {
			index += g.NumPlayers
		}
		color := colorMap[index]
		p.ColorMap()[i] = color
	}

	return p
}

func (p *Player) beginningOfTurnReset() {
	p.clearActions()
}

func (g *Game) beginningOfPhaseReset() {
	for _, p := range g.Players() {
		p.clearActions()
		p.Passed = false
	}
}

func (p *Player) clearActions() {
	p.PerformedAction = false
	p.Log = make(GameLog, 0)
}

// CanClick indicates whether a particular player can select an area.
func (g *Game) CanClick(ctx context.Context, p *Player, a *Area) (b bool) {
	cp := g.CurrentPlayer()
	switch g.Phase {
	case placeThieves:
		b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && a.Thief == noPID
	case selectThief:
		b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && a.Thief == cp.ID()
	case moveThief:
		switch {
		case g.PlayedCard.Type == lamp || g.PlayedCard.Type == sLamp:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isLampArea(a)
		case g.PlayedCard.Type == camel || g.PlayedCard.Type == sCamel:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isCamelArea(a)
		case g.PlayedCard.Type == sword:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isSwordArea(a)
		case g.PlayedCard.Type == carpet:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isCarpetArea(a)
		case g.PlayedCard.Type == turban && g.Stepped == 0:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isTurban0Area(a)
		case g.PlayedCard.Type == turban && g.Stepped == 1:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isTurban1Area(a)
		case g.PlayedCard.Type == coins:
			b = g.CUserIsCPlayerOrAdmin(ctx) && !cp.PerformedAction && g.isCoinsArea(a)
		}
	}
	return
}

// CanPlaceThief indicates whether a particular player can place a thief.
func (g *Game) CanPlaceThief(ctx context.Context, p *Player) bool {
	return g.Phase == placeThieves &&
		g.CUserIsCPlayerOrAdmin(ctx) &&
		!p.PerformedAction
}

// CanSelectCard indicates whether a particular player can select a card to play.
func (g *Game) CanSelectCard(ctx context.Context, p *Player) bool {
	return g.Phase == playCard &&
		g.CUserIsCPlayerOrAdmin(ctx) &&
		!p.PerformedAction
}

// CanSelectThief indicates whether a particular player can select a thief.
func (g *Game) CanSelectThief(ctx context.Context, p *Player) bool {
	return g.Phase == selectThief &&
		g.CUserIsCPlayerOrAdmin(ctx) &&
		!p.PerformedAction
}

// CanMoveThief indicates whether a particular player can move a thief.
func (g *Game) CanMoveThief(ctx context.Context, p *Player) bool {
	return g.Phase == moveThief &&
		g.CUserIsCPlayerOrAdmin(ctx) &&
		!p.PerformedAction &&
		g.SelectedThiefArea() != nil
}

func (g *Game) endOfTurnUpdateFor(p *Player) {
	if g.PlayedCard != nil {
		g.Jewels = *(g.PlayedCard)
	}

	for _, card := range p.Hand {
		card.FaceUp = true
	}
}

var playerValues = sslice{"Player.Passed", "Player.PerformedAction", "Player.Score"}

func (g *Game) adminPlayer(c context.Context) (string, game.ActionType, error) {
	log.Debugf(c, "Entering")
	defer log.Debugf(c, "Exiting")

	if err := g.adminUpdatePlayer(c, playerValues); err != nil {
		return "got/flash_notice", game.None, err
	}

	return "", game.Save, nil
}

func (g *Game) adminUpdatePlayer(ctx context.Context, ss sslice) error {
	if err := g.validateAdminAction(ctx); err != nil {
		return err
	}

	p := g.selectedPlayer()
	values := make(map[string][]string)
	for _, key := range ss {
		c := restful.GinFrom(ctx)
		if v := c.PostForm(key); v != "" {
			values[key] = []string{v}
		}
	}

	return schema.Decode(p, values)
}

func (g *Game) handMapFor(p *Player) (hm map[cType]int, count int) {
	hm = make(map[cType]int)
	for _, t := range g.cardTypes() {
		faceUp, faceDown := p.Hand.CountFor(t)
		if faceUp > 0 {
			hm[t] = faceUp
		}
		count += faceDown
	}
	return
}

// PlayCardDisplayFor outputs html for displaying a player's cards.
func (g *Game) PlayCardDisplayFor(p *Player) (s template.HTML) {
	cardTypes := 0
	hm, _ := g.handMapFor(p)
	for t, count := range hm {
		if count > 0 {
			cardTypes++
			pos := "push-right"
			if cardTypes%2 != 0 {
				s += restful.HTML("<div class='row' style='height:160px'>")
				pos = "pull-left"
			}

			name := t.IDString()
			s += restful.HTML("<div class=%q>", pos)
			s += restful.HTML("<div id='card-%s' data-tip=%q class='clickable card %s'></div>",
				name, t.toolTip(), name)
			s += restful.HTML("<div class='center'>%d</div></div>", count)

			if cardTypes%2 == 0 {
				s += restful.HTML("</div>")
			}
		}
	}
	if len(hm)%2 != 0 {
		s += restful.HTML("</div>")
	}
	return
}

// DisplayHandFor outputs html for displaying a player's hand.
func (g *Game) DisplayHandFor(ctx context.Context, p *Player) (s template.HTML) {
	s = restful.HTML("<div id='player-hand-%d'>", p.ID())
	hm, faceDown := g.handMapFor(p)
	if user.IsAdmin(ctx) || p.IsCurrentUser(ctx) || g.Phase == gameOver {
		for t, count := range hm {
			if count > 0 {
				name := t.IDString()
				s += restful.HTML("<div class='pull-left'>")
				s += restful.HTML("<div data-tip=%q class='card %s'></div>", t.toolTip(), name)
				s += restful.HTML("<div class='center'>%d</div></div>", count)
			}
		}
		if faceDown > 0 {
			s += restful.HTML("<div class='pull-left'>")
			s += restful.HTML("<div class='card card-back'></div>")
			s += restful.HTML("<div class='center'>%d</div></div>", faceDown)
		}
	} else {
		s += restful.HTML("<div class='pull-left'>")
		s += restful.HTML("<div class='card card-back'></div>")
		s += restful.HTML("<div class='center'>%d</div></div>", len(p.Hand))
	}
	s += restful.HTML("</div>")
	return
}

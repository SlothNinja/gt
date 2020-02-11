// Package got implements the card game, Guild of Thieves.
package got

import (
	"encoding/gob"
	"html/template"
	"reflect"
	"strconv"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/schema"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(setupEntry))
	gob.Register(new(startEntry))
}

// Register assigns a game type and routes.
func Register(t gType.Type, r *gin.Engine) {
	gob.Register(new(Game))
	game.Register(t, newGamer, phaseNames, nil)
	addRoutes(t.Prefix(), r)
}

//var ErrMustBeGame = errors.New("Resource must have type *Game.")

const noPID = game.NoPlayerID

// Game stores game state and header information.
type Game struct {
	*game.Header
	*State
}

// State stores the game state.
type State struct {
	Playerers       game.Playerers
	Log             GameLog
	Grid            grid
	Jewels          Card
	TwoThiefVariant bool `form:"two-thief-variant"`
	*TempData
}

// TempData provides storage for non-persistent values.
// They are memcached but ignored by datastore
type TempData struct {
	SelectedPlayerID   int
	BumpedPlayerID     int
	SelectedAreaF      *Area
	SelectedCardIndex  int
	Stepped            int
	PlayedCard         *Card
	JewelsPlayed       bool
	SelectedThiefAreaF *Area
	ClickAreas         areas
	Admin              string
}

// GetPlayerers implements the GetPlayerers interfaces of the sn/games package.
// Generally used to support common player manipulation functions of sn/games package.
func (g *Game) GetPlayerers() game.Playerers {
	return g.Playerers
}

// Players returns a slice of player structs that store various information about each player.
func (g *Game) Players() (ps Players) {
	pers := g.GetPlayerers()
	length := len(pers)
	if length > 0 {
		ps = make(Players, length)
		for i, p := range pers {
			ps[i] = p.(*Player)
		}
	}
	return
}

func (g *Game) setPlayers(ps Players) {
	length := len(ps)
	if length > 0 {
		pers := make(game.Playerers, length)
		for i, p := range ps {
			pers[i] = p
		}
		g.Playerers = pers
	}
}

// Games is a slice of Guild of Thieves games.
type Games []*Game

// Start begins a Guild of Thieves game.
func (g *Game) Start(ctx context.Context) error {
	g.Status = game.Running
	return g.setupPhase(ctx)
}

func (g *Game) addNewPlayers() {
	for _, u := range g.Users {
		g.addNewPlayer(u)
	}
}

func (g *Game) setupPhase(ctx context.Context) error {
	g.Turn = 0
	g.Phase = setup
	g.addNewPlayers()
	g.RandomTurnOrder()
	g.createGrid()
	for _, p := range g.Players() {
		g.newSetupEntryFor(p)
	}
	cp := g.previousPlayer(g.Players()[0])
	g.setCurrentPlayers(cp)
	g.beginningOfPhaseReset()
	return g.start(ctx)
}

type setupEntry struct {
	*Entry
}

func (g *Game) newSetupEntryFor(p *Player) (e *setupEntry) {
	e = new(setupEntry)
	e.Entry = g.newEntryFor(p)
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *setupEntry) HTML(g *Game) template.HTML {
	return restful.HTML("%s received 2 lamps and 1 camel.", g.NameByPID(e.PlayerID))
}

func (g *Game) start(ctx context.Context) error {
	g.Phase = startGame
	g.newStartEntry()
	return g.placeThieves(ctx)
}

type startEntry struct {
	*Entry
}

func (g *Game) newStartEntry() *startEntry {
	e := new(startEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *startEntry) HTML(g *Game) template.HTML {
	names := make([]string, g.NumPlayers)
	for i, p := range g.Players() {
		names[i] = g.NameFor(p)
	}
	return restful.HTML("Good luck %s.  Have fun.", restful.ToSentence(names))
}

func (g *Game) setCurrentPlayers(ps ...*Player) {
	var pers game.Playerers

	switch length := len(ps); {
	case length == 0:
		pers = nil
	case length == 1:
		pers = game.Playerers{ps[0]}
	default:
		pers = make(game.Playerers, length)
		for i, player := range ps {
			pers[i] = player
		}
	}
	g.SetCurrentPlayerers(pers...)
}

// PlayerByID returns the player having the provided player id.
func (g *Game) PlayerByID(id int) (p *Player) {
	if per := g.PlayererByID(id); per != nil {
		p = per.(*Player)
	}
	return
}

//func (g *Game) PlayerBySID(sid string) (p *Player) {
//	if per := g.Header.PlayerBySID(sid); per != nil {
//		p = per.(*Player)
//	}
//	return
//}

// PlayerByUserID returns the player having the user id.
func (g *Game) PlayerByUserID(id int64) (player *Player) {
	if p := g.PlayererByUserID(id); p != nil {
		player = p.(*Player)
	}
	return
}

//func (g *Game) PlayerByIndex(index int) (player *Player) {
//	if p := g.PlayererByIndex(index); p != nil {
//		player = p.(*Player)
//	}
//	return
//}

func (g *Game) undoTurn(ctx context.Context) (string, game.ActionType, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if !g.CUserIsCPlayerOrAdmin(ctx) {
		return "", game.None, sn.NewVError("Only the current player may perform this action.")
	}

	if cp := g.CurrentPlayer(); cp != nil {
		restful.AddNoticef(ctx, "%s undid turn.", g.NameFor(cp))
	}
	return "", game.Undo, nil
}

// CurrentPlayer returns the player whose turn it is.
func (g *Game) CurrentPlayer() (player *Player) {
	if p := g.CurrentPlayerer(); p != nil {
		player = p.(*Player)
	}
	return
}

// Convenience method for conditionally logging Debug information
// based on package global const debug
//const debug = true
//
//func (g *Game) debugf(format string, args ...interface{}) {
//	if debug {
//		g.Debugf(format, args...)
//	}
//}

type sslice []string

func (ss sslice) include(s string) bool {
	for _, str := range ss {
		if str == s {
			return true
		}
	}
	return false
}

var headerValues = sslice{
	"Header.Title",
	"Header.Turn",
	"Header.Phase",
	"Header.Round",
	"Header.Password",
	"Header.CPUserIndices",
	"Header.WinnerIDS",
	"Header.Status",
}

func (g *Game) adminHeader(ctx context.Context) (string, game.ActionType, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err := g.adminUpdateHeader(ctx, headerValues); err != nil {
		return "got/flash_notice", game.None, err
	}

	return "", game.Save, nil
}

func (g *Game) adminUpdateHeader(ctx context.Context, ss sslice) error {
	if err := g.validateAdminAction(ctx); err != nil {
		return err
	}

	values := make(map[string][]string)
	for _, key := range ss {
		if v := restful.GinFrom(ctx).PostForm(key); v != "" {
			values[key] = []string{v}
		}
	}

	schema.RegisterConverter(game.Phase(0), convertPhase)
	schema.RegisterConverter(game.Status(0), convertStatus)
	return schema.Decode(g, values)
}

func convertPhase(value string) reflect.Value {
	if v, err := strconv.ParseInt(value, 10, 0); err == nil {
		return reflect.ValueOf(game.Phase(v))
	}
	return reflect.Value{}
}

func convertStatus(value string) reflect.Value {
	if v, err := strconv.ParseInt(value, 10, 0); err == nil {
		return reflect.ValueOf(game.Status(v))
	}
	return reflect.Value{}
}

func (g *Game) selectedPlayer() *Player {
	return g.PlayerByID(g.SelectedPlayerID)
}

// BumpedPlayer identifies the player whose theif was bumped to another card due to a played sword.
func (g *Game) BumpedPlayer() *Player {
	return g.PlayerByID(g.BumpedPlayerID)
}

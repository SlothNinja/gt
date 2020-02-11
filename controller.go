package got

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/codec"
	"bitbucket.org/SlothNinja/slothninja-games/sn/color"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/mlog"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/gae/service/memcache"
	"golang.org/x/net/context"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
)

func gameFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(gameKey).(*Game)
	return
}

func withGame(c *gin.Context, g *Game) {
	c.Set(gameKey, g)
}

func jsonFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(jsonKey).(*Game)
	return
}

func withJSON(c *gin.Context, g *Game) {
	c.Set(jsonKey, g)
}

func show(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := gameFrom(ctx)
		cu := user.CurrentFrom(ctx)
		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    ctx,
			"VersionID":  info.VersionID(ctx),
			"CUser":      cu,
			"Game":       g,
			"IsAdmin":    user.IsAdmin(ctx),
			"Admin":      game.AdminFrom(ctx),
			"MessageLog": mlog.From(ctx),
			"ColorMap":   color.MapFrom(ctx),
		})
	}
}

func (g *Game) update(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	c := restful.GinFrom(ctx)
	a := c.PostForm("action")
	log.Debugf(ctx, "action: %#v", a)
	switch a {
	case "select-area":
		tmpl, act, err = g.selectArea(ctx)
	case "admin-header":
		tmpl, act, err = g.adminHeader(ctx)
	case "admin-player":
		tmpl, act, err = g.adminPlayer(ctx)
	case "pass":
		tmpl, act, err = g.pass(ctx)
	case "undo":
		tmpl, act, err = g.undoTurn(ctx)
	default:
		act, err = game.None, fmt.Errorf("%v is not a valid action", a)
	}
	return
}

func undo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "Controller#Update Game Not Found")
			return
		}
		mkey := g.UndoKey(ctx)
		if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
			log.Errorf(ctx, "Controller#Undo Error: %s", err)
		}
	}
}

func update(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
		template, actionType, err := g.update(ctx)
		switch {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(ctx, "%v", err)
			withJSON(c, g)
		case err != nil:
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			mkey := g.UndoKey(ctx)
			item := memcache.NewItem(ctx, mkey).SetExpiration(time.Minute * 30)
			v, err := codec.Encode(g)
			if err != nil {
				log.Errorf(ctx, err.Error())
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
				return
			}
			item.SetValue(v)
			if err := memcache.Set(ctx, item); err != nil {
				log.Errorf(ctx, err.Error())
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
				return
			}
		case actionType == game.SaveAndStatUpdate:
			cu := user.CurrentFrom(ctx)
			s, err := stats.ByUser(ctx, cu)
			if err != nil {
				log.Errorf(ctx, "stat.ByUser error: %v", err)
				restful.AddErrorf(ctx, "stats.ByUser error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
				return
			}

			if err := g.save(ctx, s); err != nil {
				log.Errorf(ctx, "g.save error: %s", err)
				restful.AddErrorf(ctx, "g.save error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
				return
			}
		case actionType == game.Save:
			if err := g.save(ctx); err != nil {
				log.Errorf(ctx, "%s", err)
				restful.AddErrorf(ctx, "Controller#Update Save Error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(ctx)
			if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
				log.Errorf(ctx, "memcache.Delete error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
			}
		}

		switch jData := jsonFrom(ctx); {
		case jData != nil && template == "json":
			c.JSON(http.StatusOK, jData)
		case template == "":
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param("hid")))
		default:
			cu := user.CurrentFrom(ctx)
			d := gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     cu,
				"Game":      g,
				"IsAdmin":   user.IsAdmin(ctx),
				"Notices":   restful.NoticesFrom(ctx),
				"Errors":    restful.ErrorsFrom(ctx),
			}
			log.Debugf(ctx, "d: %#v", d)
			c.HTML(http.StatusOK, template, d)
		}
	}
}

func (g *Game) save(ctx context.Context, es ...interface{}) (err error) {
	err = datastore.RunInTransaction(ctx, func(tc context.Context) (terr error) {
		oldG := New(tc)
		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
			terr = fmt.Errorf("unable to populate game with key")
			return
		}

		if terr = datastore.Get(tc, oldG.Header); terr != nil {
			return
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			terr = fmt.Errorf("game state changed unexpectantly -- try again")
			return
		}

		if terr = g.encode(ctx); terr != nil {
			return
		}

		if terr = datastore.Put(tc, append(es, g.Header)); terr != nil {
			return
		}

		if terr = memcache.Delete(tc, g.UndoKey(tc)); terr == memcache.ErrCacheMiss {
			terr = nil
		}
		return
	}, &datastore.TransactionOptions{XG: true})
	return
}

func (g *Game) encode(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.TempData = nil

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

//func (g *Game) saveAndUpdateStats(c *gin.Context) error {
//	ctx := restful.ContextFrom(c)
//	cu := user.CurrentFrom(c)
//	s, err := stats.ByUser(c, cu)
//	if err != nil {
//		return err
//	}
//
//	return datastore.RunInTransaction(ctx, func(tc context.Context) error {
//		c = restful.WithContext(c, tc)
//		oldG := New(c)
//		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
//			return fmt.Errorf("Unable to populate game with key.")
//		}
//		if err := datastore.Get(tc, oldG.Header); err != nil {
//			return err
//		}
//
//		if oldG.UpdatedAt != g.UpdatedAt {
//			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
//		}
//
//		g.TempData = nil
//		if encoded, err := codec.Encode(g.State); err != nil {
//			return err
//		} else {
//			g.SavedState = encoded
//		}
//
//		es := []interface{}{s, g.Header}
//		if err := datastore.Put(tc, es); err != nil {
//			return err
//		}
//		if err := memcache.Delete(tc, g.UndoKey(c)); err != nil && err != memcache.ErrCacheMiss {
//			return err
//		}
//		return nil
//	}, &datastore.TransactionOptions{XG: true})
//}

func newGamer(ctx context.Context) game.Gamer {
	return New(ctx)
}

func index(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		gs := game.GamersFrom(ctx)
		switch status := game.StatusFrom(ctx); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.GOT.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.GOT.String(),
				"Status":    status,
			})
		}
	}
}

//func Index(prefix string) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		ctx := restful.ContextFrom(c)
//		log.Debugf(ctx, "Entering")
//		defer log.Debugf(ctx, "Exiting")
//
//		gs := game.GamersFrom(ctx)
//		switch {
//		case game.StatusFrom(ctx) == game.Recruiting:
//			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
//				"Context":   ctx,
//				"VersionID": info.VersionID(ctx),
//				"CUser":     user.CurrentFrom(ctx),
//				"Games":     gs,
//			})
//		case gType.TypeFrom(ctx) == gType.All:
//			c.HTML(http.StatusOK, "shared/multi_games_index", gin.H{
//				"Context":   ctx,
//				"VersionID": info.VersionID(ctx),
//				"CUser":     user.CurrentFrom(ctx),
//				"Games":     gs,
//			})
//		default:
//			c.HTML(http.StatusOK, "shared/games_index", gin.H{
//				"Context":   ctx,
//				"VersionID": info.VersionID(ctx),
//				"CUser":     user.CurrentFrom(ctx),
//				"Games":     gs,
//			})
//		}
//	}
//}

func recruitingPath(prefix string) string {
	return fmt.Sprintf("/%s/games/recruiting", prefix)
}

func newPath(prefix string) string {
	return fmt.Sprintf("/%s/game/new", prefix)
}

func newAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := New(ctx)
		withGame(c, g)
		if err := g.FromParams(ctx, gType.GOT); err != nil {
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   ctx,
			"VersionID": info.VersionID(ctx),
			"CUser":     user.CurrentFrom(ctx),
			"Game":      g,
		})
	}
}

func create(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)

		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := New(ctx)
		withGame(c, g)

		var err error
		if err = g.FromParams(ctx, g.Type); err == nil {
			err = g.fromForm(ctx)
		}

		if err == nil {
			err = g.encode(ctx)
		}

		if err == nil {
			err = datastore.RunInTransaction(ctx, func(tc context.Context) (err error) {
				if err = datastore.Put(tc, g.Header); err != nil {
					return
				}

				m := mlog.New()
				m.ID = g.ID
				return datastore.Put(tc, m)

			}, &datastore.TransactionOptions{XG: true})
		}

		if err == nil {
			restful.AddNoticef(ctx, "<div>%s created.</div>", g.Title)
		} else {
			log.Errorf(ctx, err.Error())
		}
	}
}

func accept(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var (
			start bool
			err   error
		)

		u := user.CurrentFrom(ctx)
		if start, err = g.Accept(ctx, u); err == nil && start {
			err = g.Start(ctx)
		}

		if err == nil {
			err = g.save(ctx)
		}

		if err == nil && start {
			g.SendTurnNotificationsTo(ctx, g.CurrentPlayer())
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
		}

	}
}

func drop(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var err error

		u := user.CurrentFrom(ctx)
		if err = g.Drop(u); err == nil {
			err = g.save(ctx)
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
			restful.AddErrorf(ctx, err.Error())
		}

	}
}

func fetch(c *gin.Context) {
	ctx := restful.ContextFrom(c)
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")
	// create Gamer
	log.Debugf(ctx, "hid: %v", c.Param("hid"))
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	log.Debugf(ctx, "id: %v", id)
	g := New(ctx)
	g.ID = id

	switch action := c.PostForm("action"); {
	case action == "reset":
		// pull from memcache/datastore
		// same as undo
		fallthrough
	case action == "undo":
		// pull from memcache/datastore
		if err := dsGet(ctx, g); err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	default:
		if user.CurrentFrom(ctx) != nil {
			// pull from memcache and return if successful; otherwise pull from datastore
			if err := mcGet(ctx, g); err == nil {
				return
			}
		}

		log.Debugf(ctx, "g: %#v", g)
		log.Debugf(ctx, "k: %v", datastore.KeyForObj(ctx, g.Header))
		if err := dsGet(ctx, g); err != nil {
			log.Debugf(ctx, "dsGet error: %v", err)
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	}
}

// pull temporary game state from memcache.  Note may be different from value stored in datastore.
func mcGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var (
		mkey string
		item memcache.Item
	)

	mkey = g.GetHeader().UndoKey(ctx)
	if item, err = memcache.GetKey(ctx, mkey); err != nil {
		return
	}

	if err = codec.Decode(g, item.Value()); err != nil {
		return
	}

	if err = g.afterCache(); err != nil {
		return
	}

	c := restful.GinFrom(ctx)
	withGame(c, g)
	color.WithMap(c, g.ColorMapFor(user.CurrentFrom(ctx)))
	return
}

// pull game state from memcache/datastore.  returned memcache should be same as datastore.
func dsGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch err = datastore.Get(ctx, g.Header); {
	case err != nil:
		restful.AddErrorf(ctx, err.Error())
		return
	case g == nil:
		err = fmt.Errorf("Unable to get game for id: %v", g.ID)
		restful.AddErrorf(ctx, err.Error())
		return
	}

	s := newState()
	if err = codec.Decode(&s, g.SavedState); err != nil {
		restful.AddErrorf(ctx, err.Error())
		return
	}

	g.State = s

	if err = g.init(ctx); err != nil {
		restful.AddErrorf(ctx, err.Error())
		return
	}

	c := restful.GinFrom(ctx)
	withGame(c, g)
	cm := g.ColorMapFor(user.CurrentFrom(ctx))
	color.WithMap(c, cm)
	return nil
}

func jsonIndexAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		game.JSONIndexAction(c)
	}
}

func (g *Game) updateHeader() {
	g.OptString = g.options()
	switch g.Phase {
	case gameOver:
		g.Progress = g.PhaseName()
	default:
		g.Progress = fmt.Sprintf("<div>Turn: %d</div><div>Phase: %s</div>", g.Turn, g.PhaseName())
	}
	if u := g.Creator; u != nil {
		g.CreatorSID = user.GenID(u.GoogleID)
		g.CreatorName = u.Name
	}

	if l := len(g.Users); l > 0 {
		g.UserSIDS = make([]string, l)
		g.UserNames = make([]string, l)
		g.UserEmails = make([]string, l)
		for i, u := range g.Users {
			g.UserSIDS[i] = user.GenID(u.GoogleID)
			g.UserNames[i] = u.Name
			g.UserEmails[i] = u.Email
		}
	}
}

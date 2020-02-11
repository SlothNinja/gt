package got

import "bitbucket.org/SlothNinja/slothninja-games/sn/game"

const (
	noPhase game.Phase = iota
	setup
	startGame
	placeThieves
	playCard
	selectThief
	moveThief
	claimItem
	drawCard
	finalClaim
	announceWinners
	gameOver
	endGame
	awaitPlayerInput
)

var phaseNames = game.PhaseNameMap{
	noPhase:          "None",
	setup:            "Setup",
	startGame:        "Start Game",
	placeThieves:     "Place Thieves",
	playCard:         "Play Card",
	selectThief:      "Select Thief",
	moveThief:        "Move Thief",
	claimItem:        "Claim Magical Item",
	drawCard:         "Draw Card",
	finalClaim:       "Final Claim",
	announceWinners:  "Announce Winners",
	gameOver:         "Game Over",
	endGame:          "End Of Game",
	awaitPlayerInput: "Await Player Input",
}

// PhaseNames returns all phase names used by the game.
func (g *Game) PhaseNames() game.PhaseNameMap {
	return phaseNames
}

// PhaseName returns the name of the current phase of the game.
func (g *Game) PhaseName() string {
	return phaseNames[g.Phase]
}

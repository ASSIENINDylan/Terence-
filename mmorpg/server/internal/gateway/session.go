package gateway

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/zone"
)

// session porte l'état d'un joueur connecté et sérialise toutes les mutations de
// son personnage (changements de zone : transitions client-initiées et
// réapparitions initiées par le serveur) dans une unique goroutine de contrôle.
// Elle satisfait zone.Client : la zone peut lui envoyer des messages et lui
// demander une relocalisation.
type session struct {
	gw        *Gateway
	conn      *Conn
	accountID int64

	mu   sync.Mutex
	char domain.Character

	control chan func()
	done    chan struct{}
}

func newSession(gw *Gateway, conn *Conn, accountID int64, char domain.Character) *session {
	return &session{
		gw: gw, conn: conn, accountID: accountID, char: char,
		control: make(chan func(), 8),
		done:    make(chan struct{}),
	}
}

// SendEnvelope satisfait zone.Sender.
func (s *session) SendEnvelope(msgType string, seq uint64, data any) {
	s.conn.SendEnvelope(msgType, seq, data)
}

// Relocate satisfait zone.Client : demande non bloquante de changement de zone
// (réapparition après la mort). Le travail réel est fait par la goroutine de
// contrôle pour ne pas bloquer la zone appelante ni entrer en course avec elle.
func (s *session) Relocate(char domain.Character) {
	select {
	case s.control <- func() { s.doRelocate(char) }:
	case <-s.done:
	}
}

func (s *session) getChar() domain.Character {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.char
}

func (s *session) setChar(c domain.Character) {
	s.mu.Lock()
	s.char = c
	s.mu.Unlock()
}

// controlLoop exécute en série les mutations de zone du personnage.
func (s *session) controlLoop() {
	for {
		select {
		case fn := <-s.control:
			fn()
		case <-s.done:
			return
		}
	}
}

// doRelocate place le personnage dans sa nouvelle zone (le perdant a déjà été
// retiré de la zone du combat par l'acteur de zone) et lui envoie un snapshot.
func (s *session) doRelocate(char domain.Character) {
	s.setChar(char)
	snap := s.gw.world.Enter(char, s)
	s.conn.SendEnvelope(protocol.TypeZoneSnapshot, 0, snap)
	s.persist(char)
	s.gw.log.Info("réapparition", "character_id", char.ID, "zone_id", char.ZoneID, "xp", char.XP)
}

// doTransition emprunte un portail/barrière et bascule le joueur de zone.
func (s *session) doTransition(linkID int) {
	char := s.getChar()
	newChar, snap, err := s.gw.world.Transition(char, linkID, s)
	if err != nil {
		code, msg := "transition_refused", "transition impossible"
		var te *zone.TransitionError
		if errors.As(err, &te) {
			code, msg = te.Code, te.Message
		}
		s.conn.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{Code: code, Message: msg})
		return
	}
	s.setChar(newChar)
	s.conn.SendEnvelope(protocol.TypeZoneSnapshot, 0, snap)
	s.persist(newChar)
	s.gw.log.Info("transition de zone", "character_id", newChar.ID, "zone_id", newChar.ZoneID)
}

// persist sauvegarde l'état du personnage (position, zone, PV, XP) dans un
// contexte détaché de la requête.
func (s *session) persist(char domain.Character) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.gw.characters.SaveState(ctx, char); err != nil {
		s.gw.log.Warn("sauvegarde de l'état échouée", "character_id", char.ID, "err", err)
	}
}

// handle route un message applicatif entrant. Les mutations de zone
// (transition) passent par la goroutine de contrôle ; le reste est traité en
// ligne à partir de l'état courant.
func (s *session) handle(env protocol.Envelope) {
	switch env.Type {
	case protocol.TypeMoveIntent:
		var mi moveIntentPayload
		if err := env.DecodeData(&mi); err != nil {
			s.conn.SendEnvelope(protocol.TypeError, env.Seq, protocol.ErrorData{
				Code: "invalid_move_intent", Message: "intention de déplacement mal formée",
			})
			return
		}
		s.gw.world.Move(s.getChar(), zone.Intent{DX: mi.DX, DY: mi.DY})

	case protocol.TypeCombatAction:
		var ca combatActionPayload
		if err := env.DecodeData(&ca); err != nil {
			s.conn.SendEnvelope(protocol.TypeError, env.Seq, protocol.ErrorData{
				Code: "invalid_combat_action", Message: "action de combat mal formée",
			})
			return
		}
		s.gw.world.CombatAction(s.getChar(), ca.Action, ca.TechID)

	case protocol.TypeSpendAttr:
		var p spendAttrPayload
		if err := env.DecodeData(&p); err != nil {
			return
		}
		s.gw.world.SpendAttr(s.getChar(), p.Attr, p.Perfect)

	case protocol.TypeLearnTech:
		var p learnTechPayload
		if err := env.DecodeData(&p); err != nil {
			return
		}
		s.gw.world.LearnTech(s.getChar(), p.TechID)

	case protocol.TypeBuyPotion:
		s.gw.world.BuyPotion(s.getChar())

	case protocol.TypeUsePotion:
		s.gw.world.UsePotion(s.getChar())

	case protocol.TypeZoneTransition:
		var t transitionPayload
		if err := env.DecodeData(&t); err != nil {
			s.conn.SendEnvelope(protocol.TypeError, env.Seq, protocol.ErrorData{
				Code: "invalid_transition", Message: "transition mal formée",
			})
			return
		}
		linkID := t.LinkID
		select {
		case s.control <- func() { s.doTransition(linkID) }:
		case <-s.done:
		}

	default:
		// Types non gérés : écho (hérité de T2).
		s.conn.SendEnvelope(protocol.TypeEcho, env.Seq, env.Data)
	}
}

// run démarre la goroutine de contrôle, traite les messages jusqu'à la fermeture
// de la connexion, puis retire le personnage du monde et sauvegarde son état.
func (s *session) run() {
	go s.controlLoop()
	s.conn.run(s.handle)
	close(s.done)

	// Sauvegarde l'état AUTORITATIF (position/PV/XP tenus par l'acteur de zone),
	// pas la copie figée de la session.
	final := s.getChar()
	if cur, ok := s.gw.world.CurrentChar(final); ok {
		final = cur
	}
	s.gw.world.Leave(final)
	s.persist(final)
}

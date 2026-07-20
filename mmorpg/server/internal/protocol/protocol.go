// Package protocol définit l'enveloppe de message échangée entre le client et le
// serveur (§5 du plan technique).
//
// Toute communication temps réel passe par une enveloppe uniforme {type, seq,
// data}. La (dé)sérialisation est isolée ici : en v0.1 c'est du JSON, mais la
// logique applicative ne manipule que des Envelope — on pourra basculer vers un
// format binaire plus tard sans toucher au reste du serveur.
package protocol

import (
	"encoding/json"
	"fmt"
)

// Types de messages de la v0.1. La convention « domaine.action » laisse la place
// aux messages de jeu à venir (zone.snapshot, move.intent, zone.delta… T3+).
const (
	// TypeAuthOK confirme un handshake authentifié réussi (T2).
	TypeAuthOK = "auth.ok"
	// TypeZoneSnapshot porte l'état initial d'une zone à l'entrée du joueur (T3).
	TypeZoneSnapshot = "zone.snapshot"
	// TypeMoveIntent est l'intention de déplacement envoyée par le client (T4).
	// Le serveur ne reçoit qu'une direction ; il reste seul maître de la position.
	TypeMoveIntent = "move.intent"
	// TypeZoneTransition : le client emprunte un portail/barrière (link_id) pour
	// changer de zone. Le serveur valide et déplace le joueur.
	TypeZoneTransition = "zone.transition"
	// TypeZoneDelta diffuse les changements d'une zone à chaque tick (T4) :
	// arrivées, déplacements, départs.
	TypeZoneDelta = "zone.delta"
	// Combat au tour par tour (T5), déclenché par une rencontre.
	// TypeCombatStart : un combat s'engage (participants, à qui de jouer).
	TypeCombatStart = "combat.start"
	// TypeCombatAction : action choisie par le joueur actif (client→serveur).
	TypeCombatAction = "combat.action"
	// TypeCombatEvent : résultat d'une action (dégâts, PV, tour suivant).
	TypeCombatEvent = "combat.event"
	// TypeCombatEnd : fin du combat (mort et réapparition, ou fuite).
	TypeCombatEnd = "combat.end"
	// TypeCharUpdate : état complet du personnage (fiche) après tout changement
	// (niveau, stats, or, énergie, points, techniques).
	TypeCharUpdate = "char.update"
	// Services du village (client→serveur) : académie, temple, boutique.
	TypeSpendAttr = "char.spend_attr" // {attr:"str|def|agi", perfect:bool}
	TypeLearnTech = "char.learn_tech" // {tech_id}
	TypeBuyPotion = "char.buy_potion"
	TypeUsePotion = "char.use_potion"
	// TypeEcho est renvoyé par le serveur en écho d'un message reçu (T2).
	TypeEcho = "echo"
	// TypeError signale une erreur applicative sur la connexion.
	TypeError = "error"
	// TypePing / TypePong : keepalive applicatif optionnel (au-dessus du
	// ping/pong natif WebSocket).
	TypePing = "ping"
	TypePong = "pong"
)

// Envelope est le conteneur commun à tous les messages. data reste brut
// (json.RawMessage) pour n'être décodé qu'au moment où le destinataire connaît
// le type concret attendu.
type Envelope struct {
	// Type identifie la nature du message (voir constantes ci-dessus).
	Type string `json:"type"`
	// Seq est un numéro de séquence optionnel, utile pour corréler une réponse
	// à une requête et détecter les pertes côté client.
	Seq uint64 `json:"seq,omitempty"`
	// Data porte la charge utile spécifique au type, décodée à la demande.
	Data json.RawMessage `json:"data,omitempty"`
}

// Decode désérialise une enveloppe depuis des octets JSON.
func Decode(raw []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return Envelope{}, fmt.Errorf("protocol: enveloppe invalide: %w", err)
	}
	if e.Type == "" {
		return Envelope{}, fmt.Errorf("protocol: champ 'type' manquant")
	}
	return e, nil
}

// Encode sérialise une enveloppe en JSON. La charge utile data est d'abord
// marshalée puis intégrée telle quelle.
func Encode(msgType string, seq uint64, data any) ([]byte, error) {
	e := Envelope{Type: msgType, Seq: seq}
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("protocol: sérialisation de data (%s): %w", msgType, err)
		}
		e.Data = payload
	}
	return json.Marshal(e)
}

// DecodeData désérialise la charge utile d'une enveloppe dans dst.
func (e Envelope) DecodeData(dst any) error {
	if len(e.Data) == 0 {
		return fmt.Errorf("protocol: data vide pour le type %q", e.Type)
	}
	if err := json.Unmarshal(e.Data, dst); err != nil {
		return fmt.Errorf("protocol: décodage de data (%s): %w", e.Type, err)
	}
	return nil
}

// ErrorData est la charge utile standard d'un message TypeError.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

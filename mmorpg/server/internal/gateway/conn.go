package gateway

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

// Paramètres du transport WebSocket (v0.1). Le ping period doit rester
// nettement inférieur au pong wait pour détecter une connexion morte.
const (
	writeWait      = 10 * time.Second // délai max d'écriture d'un frame
	pongWait       = 60 * time.Second // silence max toléré avant de couper
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8 << 10 // 8 KiB : suffisant pour les intents client
	sendBuffer     = 64      // messages en attente d'écriture avant saturation
)

// Conn enveloppe une connexion WebSocket authentifiée. Un unique writePump
// possède l'écriture (les writes gorilla ne sont pas concurrents) ; les autres
// goroutines poussent des messages via Send. C'est la brique réutilisée dès T3
// pour envoyer snapshots et deltas.
type Conn struct {
	accountID int64
	ws        *websocket.Conn
	log       *slog.Logger

	send      chan []byte
	closeOnce sync.Once
}

func newConn(accountID int64, ws *websocket.Conn, log *slog.Logger) *Conn {
	return &Conn{
		accountID: accountID,
		ws:        ws,
		log:       log.With("account_id", accountID),
		send:      make(chan []byte, sendBuffer),
	}
}

// Send met un message en file d'écriture. Non bloquant : si le client ne draine
// pas assez vite (buffer plein), la connexion est fermée plutôt que de bloquer
// le serveur (protection contre les consommateurs lents).
func (c *Conn) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		c.log.Warn("file d'envoi saturée, fermeture de la connexion")
		c.close()
	}
}

// SendEnvelope encode puis met en file une enveloppe de protocole.
func (c *Conn) SendEnvelope(msgType string, seq uint64, data any) {
	raw, err := protocol.Encode(msgType, seq, data)
	if err != nil {
		c.log.Error("échec d'encodage d'enveloppe", "type", msgType, "err", err)
		return
	}
	c.Send(raw)
}

// close ferme le canal d'envoi une seule fois ; le writePump fermera la socket.
func (c *Conn) close() {
	c.closeOnce.Do(func() { close(c.send) })
}

// run lance les deux pumps et bloque jusqu'à la fermeture de la connexion.
func (c *Conn) run() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); c.writePump() }()
	go func() { defer wg.Done(); c.readPump() }()
	wg.Wait()
}

// readPump lit les messages entrants et les traite. Il possède exclusivement la
// lecture de la socket. Toute erreur de lecture termine la connexion.
func (c *Conn) readPump() {
	defer c.close()

	c.ws.SetReadLimit(maxMessageSize)
	_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.log.Warn("fermeture inattendue de la connexion", "err", err)
			}
			return
		}
		c.handleMessage(raw)
	}
}

// handleMessage traite un message entrant. En T2, tout message valide est
// renvoyé en écho (en préservant son numéro de séquence) ; un message invalide
// reçoit une enveloppe d'erreur. C'est le point d'extension du routage de jeu
// (move.intent, chat.send… T3+).
func (c *Conn) handleMessage(raw []byte) {
	env, err := protocol.Decode(raw)
	if err != nil {
		c.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code:    "invalid_message",
			Message: "message mal formé",
		})
		return
	}

	switch env.Type {
	case protocol.TypePing:
		c.SendEnvelope(protocol.TypePong, env.Seq, nil)
	default:
		// Écho : on renvoie la charge utile reçue telle quelle (T2).
		c.SendEnvelope(protocol.TypeEcho, env.Seq, env.Data)
	}
}

// writePump possède l'écriture de la socket : il draine le canal send et émet
// les pings de keepalive. Sa fin ferme la connexion réseau.
func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send fermé : on notifie proprement le pair puis on sort.
				_ = c.ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.log.Warn("échec d'écriture, fermeture", "err", err)
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

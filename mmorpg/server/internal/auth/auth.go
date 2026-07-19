// Package auth porte l'identité des joueurs (§6, T1 du plan de build) :
// création de compte, connexion HTTP, et gestion des tokens de session.
//
// Frontières :
//   - Les mots de passe sont hachés (bcrypt) et ne transitent jamais en clair
//     au-delà de ce package ; seul le hash atteint la base.
//   - Un token de session est une valeur opaque aléatoire, stockée côté serveur
//     dans Redis avec une durée de vie. Le client ne fait que le présenter.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/assienindylan/terence-/mmorpg/server/internal/store"
)

// Erreurs métier de l'authentification, mappées vers des codes HTTP par la
// couche httpapi. On ne distingue jamais « email inconnu » de « mauvais mot de
// passe » côté client : les deux donnent ErrInvalidCredentials (anti-énumération).
var (
	ErrEmailTaken         = errors.New("auth: email déjà utilisé")
	ErrInvalidCredentials = errors.New("auth: identifiants invalides")
	ErrInvalidEmail       = errors.New("auth: email invalide")
	ErrWeakPassword       = errors.New("auth: mot de passe trop court")
	ErrAccountNotActive   = errors.New("auth: compte non actif")
	ErrInvalidToken       = errors.New("auth: token de session invalide ou expiré")
)

// minPasswordLen est la longueur minimale acceptée pour un mot de passe (v0.1).
const minPasswordLen = 8

// AccountRepo est l'accès durable aux comptes dont le service a besoin.
type AccountRepo interface {
	CreateAccount(ctx context.Context, email, passwordHash string) (store.Account, error)
	GetAccountByEmail(ctx context.Context, email string) (store.Account, error)
	TouchLastLogin(ctx context.Context, id int64) error
}

// SessionRepo est le stockage volatile des tokens de session (Redis).
type SessionRepo interface {
	CreateSession(ctx context.Context, token string, accountID int64, ttl time.Duration) error
	GetSession(ctx context.Context, token string) (int64, error)
	DeleteSession(ctx context.Context, token string) error
}

// Service orchestre inscription, connexion et validation de session.
type Service struct {
	accounts   AccountRepo
	sessions   SessionRepo
	sessionTTL time.Duration
}

// NewService construit le service avec ses dépendances et la durée de vie des
// sessions.
func NewService(accounts AccountRepo, sessions SessionRepo, sessionTTL time.Duration) *Service {
	return &Service{accounts: accounts, sessions: sessions, sessionTTL: sessionTTL}
}

// SessionTTL expose la durée de vie configurée des sessions (utile pour
// renseigner expires_in dans la réponse de login).
func (s *Service) SessionTTL() time.Duration { return s.sessionTTL }

// Register crée un compte à partir d'un email et d'un mot de passe en clair.
// Le mot de passe est haché avant tout stockage. Renvoie ErrEmailTaken si
// l'email est déjà pris.
func (s *Service) Register(ctx context.Context, email, password string) (store.Account, error) {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return store.Account{}, ErrInvalidEmail
	}
	if len(password) < minPasswordLen {
		return store.Account{}, ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return store.Account{}, fmt.Errorf("auth: hachage du mot de passe: %w", err)
	}

	acc, err := s.accounts.CreateAccount(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			return store.Account{}, ErrEmailTaken
		}
		return store.Account{}, err
	}
	return acc, nil
}

// Login vérifie les identifiants et, en cas de succès, crée un token de session
// dans Redis. Retourne le token opaque à présenter aux requêtes suivantes.
func (s *Service) Login(ctx context.Context, email, password string) (token string, acc store.Account, err error) {
	email = normalizeEmail(email)

	acc, err = s.accounts.GetAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrAccountNotFound) {
			// On hache tout de même une valeur bidon pour ne pas révéler par le
			// temps de réponse qu'un email est inconnu (timing anti-énumération).
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return "", store.Account{}, ErrInvalidCredentials
		}
		return "", store.Account{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)); err != nil {
		return "", store.Account{}, ErrInvalidCredentials
	}
	if acc.Status != "active" {
		return "", store.Account{}, ErrAccountNotActive
	}

	token, err = newToken()
	if err != nil {
		return "", store.Account{}, err
	}
	if err := s.sessions.CreateSession(ctx, token, acc.ID, s.sessionTTL); err != nil {
		return "", store.Account{}, err
	}

	// Non bloquant pour la connexion : un échec de traçage ne doit pas empêcher
	// le joueur d'entrer. On ignore volontairement l'erreur ici.
	_ = s.accounts.TouchLastLogin(ctx, acc.ID)

	return token, acc, nil
}

// Authenticate résout un token de session vers l'identifiant de compte.
// C'est le point d'entrée de vérification réutilisé par le handshake WebSocket
// (T2). Renvoie ErrInvalidToken si le token est absent ou expiré.
func (s *Service) Authenticate(ctx context.Context, token string) (accountID int64, err error) {
	if token == "" {
		return 0, ErrInvalidToken
	}
	accountID, err = s.sessions.GetSession(ctx, token)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return accountID, nil
}

// Logout révoque un token de session (idempotent).
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.DeleteSession(ctx, token)
}

// dummyHash est un hash bcrypt valide d'une valeur arbitraire, comparé lorsque
// l'email est inconnu pour uniformiser le temps de réponse.
var dummyHash = mustHash("timing-uniformization-placeholder")

func mustHash(s string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return h
}

// newToken génère un token de session opaque de 256 bits, encodé en base64url
// sans padding (sûr en URL et en en-tête HTTP).
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: génération du token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(email)
}

// validEmail applique une validation volontairement minimale (v0.1) : présence
// d'un « @ » entouré de caractères et d'un « . » dans le domaine.
func validEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".")
}

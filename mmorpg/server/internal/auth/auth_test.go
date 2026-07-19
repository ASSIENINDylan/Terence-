package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/store"
)

// fakeAccounts est un AccountRepo en mémoire pour tester le service sans base.
type fakeAccounts struct {
	byEmail map[string]store.Account
	nextID  int64
	touched map[int64]int
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{byEmail: map[string]store.Account{}, nextID: 1, touched: map[int64]int{}}
}

func (f *fakeAccounts) CreateAccount(_ context.Context, email, passwordHash string) (store.Account, error) {
	if _, ok := f.byEmail[email]; ok {
		return store.Account{}, store.ErrEmailTaken
	}
	a := store.Account{ID: f.nextID, Email: email, PasswordHash: passwordHash, Status: "active"}
	f.nextID++
	f.byEmail[email] = a
	return a, nil
}

func (f *fakeAccounts) GetAccountByEmail(_ context.Context, email string) (store.Account, error) {
	a, ok := f.byEmail[email]
	if !ok {
		return store.Account{}, store.ErrAccountNotFound
	}
	return a, nil
}

func (f *fakeAccounts) TouchLastLogin(_ context.Context, id int64) error {
	f.touched[id]++
	return nil
}

// fakeSessions est un SessionRepo en mémoire (le TTL n'est pas simulé).
type fakeSessions struct {
	tokens map[string]int64
}

func newFakeSessions() *fakeSessions { return &fakeSessions{tokens: map[string]int64{}} }

func (f *fakeSessions) CreateSession(_ context.Context, token string, accountID int64, _ time.Duration) error {
	f.tokens[token] = accountID
	return nil
}

func (f *fakeSessions) GetSession(_ context.Context, token string) (int64, error) {
	id, ok := f.tokens[token]
	if !ok {
		return 0, errors.New("not found")
	}
	return id, nil
}

func (f *fakeSessions) DeleteSession(_ context.Context, token string) error {
	delete(f.tokens, token)
	return nil
}

func newTestService() (*Service, *fakeAccounts, *fakeSessions) {
	fa, fs := newFakeAccounts(), newFakeSessions()
	return NewService(fa, fs, time.Hour), fa, fs
}

func TestRegisterValidation(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "no-at-sign", "validpass"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("email invalide: attendu ErrInvalidEmail, obtenu %v", err)
	}
	if _, err := svc.Register(ctx, "a@b.com", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("mot de passe court: attendu ErrWeakPassword, obtenu %v", err)
	}
}

func TestRegisterHashesAndRejectsDuplicate(t *testing.T) {
	svc, fa, _ := newTestService()
	ctx := context.Background()

	acc, err := svc.Register(ctx, "player@example.com", "s3cr3t-pass")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if acc.PasswordHash == "s3cr3t-pass" || acc.PasswordHash == "" {
		t.Fatalf("le mot de passe doit être haché, obtenu %q", acc.PasswordHash)
	}
	if _, ok := fa.byEmail["player@example.com"]; !ok {
		t.Fatal("compte non persisté")
	}

	if _, err := svc.Register(ctx, "player@example.com", "another-pass"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("doublon: attendu ErrEmailTaken, obtenu %v", err)
	}
}

func TestLoginFlow(t *testing.T) {
	svc, fa, _ := newTestService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, "player@example.com", "s3cr3t-pass"); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Mauvais mot de passe et email inconnu ⇒ même erreur (anti-énumération).
	if _, _, err := svc.Login(ctx, "player@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("mauvais mot de passe: attendu ErrInvalidCredentials, obtenu %v", err)
	}
	if _, _, err := svc.Login(ctx, "ghost@example.com", "whatever"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("email inconnu: attendu ErrInvalidCredentials, obtenu %v", err)
	}

	// Connexion réussie ⇒ token exploitable, last_login_at touché.
	token, acc, err := svc.Login(ctx, "player@example.com", "s3cr3t-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("token vide")
	}
	if fa.touched[acc.ID] != 1 {
		t.Fatalf("last_login_at devait être touché une fois, obtenu %d", fa.touched[acc.ID])
	}

	id, err := svc.Authenticate(ctx, token)
	if err != nil || id != acc.ID {
		t.Fatalf("authenticate: id=%d err=%v", id, err)
	}

	// Après logout, le token n'est plus valable.
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("après logout: attendu ErrInvalidToken, obtenu %v", err)
	}
}

func TestTokensAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		tok, err := newToken()
		if err != nil {
			t.Fatalf("newToken: %v", err)
		}
		if seen[tok] {
			t.Fatalf("collision de token à l'itération %d", i)
		}
		seen[tok] = true
	}
}

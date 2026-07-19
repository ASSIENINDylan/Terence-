package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Erreurs métier renvoyées par la couche comptes, pour que l'appelant décide de
// la réponse HTTP sans inspecter les codes SQL bruts.
var (
	// ErrEmailTaken : un compte existe déjà pour cet email (violation d'unicité).
	ErrEmailTaken = errors.New("store: email déjà utilisé")
	// ErrAccountNotFound : aucun compte ne correspond au critère demandé.
	ErrAccountNotFound = errors.New("store: compte introuvable")
)

// Account est la projection Go d'une ligne de la table accounts (§6).
// Le mot de passe n'existe jamais en clair : seul son hash est stocké.
type Account struct {
	ID           int64
	Email        string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// CreateAccount insère un nouveau compte et retourne la ligne créée.
// Renvoie ErrEmailTaken si l'email est déjà pris (contrainte UNIQUE).
func (s *Store) CreateAccount(ctx context.Context, email, passwordHash string) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, status, created_at, last_login_at`,
		email, passwordHash,
	).Scan(&a.ID, &a.Email, &a.PasswordHash, &a.Status, &a.CreatedAt, &a.LastLoginAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return Account{}, ErrEmailTaken
		}
		return Account{}, fmt.Errorf("store: création du compte: %w", err)
	}
	return a, nil
}

// GetAccountByEmail lit un compte par son email (insensible à la casse grâce à
// citext). Renvoie ErrAccountNotFound si aucun compte ne correspond.
func (s *Store) GetAccountByEmail(ctx context.Context, email string) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, status, created_at, last_login_at
		FROM accounts
		WHERE email = $1`,
		email,
	).Scan(&a.ID, &a.Email, &a.PasswordHash, &a.Status, &a.CreatedAt, &a.LastLoginAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, fmt.Errorf("store: lecture du compte: %w", err)
	}
	return a, nil
}

// TouchLastLogin met à jour l'horodatage de dernière connexion à maintenant.
func (s *Store) TouchLastLogin(ctx context.Context, id int64) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE accounts SET last_login_at = now() WHERE id = $1`, id,
	); err != nil {
		return fmt.Errorf("store: mise à jour last_login_at: %w", err)
	}
	return nil
}

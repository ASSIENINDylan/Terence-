// Package migrations embarque les fichiers SQL versionnés du schéma.
//
// Convention (§12) : un fichier NNNN_nom.sql par changement de schéma, joué une
// seule fois et dans l'ordre lexicographique. Jamais d'ALTER manuel en base.
package migrations

import "embed"

// FS contient tous les fichiers .sql de ce dossier, embarqués dans le binaire.
//
//go:embed *.sql
var FS embed.FS

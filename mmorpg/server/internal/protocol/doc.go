// Package protocol portera l'enveloppe de message commune (type, seq, data) et
// la (dé)sérialisation (JSON en v0.1), conçue pour basculer vers un format
// binaire plus tard sans changer la logique applicative (§5). Dès T2.
package protocol

// Package gateway portera la terminaison WebSocket (wss://), l'authentification
// par token de session, le rate-limiting par connexion et le routage des
// messages entrants vers la bonne zone (§3, §5). Implémenté à partir de T2.
package gateway

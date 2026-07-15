package main

import "log"

func main() {
	// Este alias historico no puede arrancar un servidor con una composicion
	// distinta o ignorar TLS. Solo cmd/vec-server es una entrada soportada.
	log.Fatal("bolsa-server esta retirado: use el arranque canonico vec-server")
}

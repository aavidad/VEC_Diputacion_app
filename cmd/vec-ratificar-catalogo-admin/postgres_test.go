package main

import (
	"crypto/tls"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestCanalExigeTLSVerificadoOSocketExplicito(t *testing.T) {
	tcp := &pgconn.Config{Host: "db.ejemplo", User: "login_prueba", Database: "vec_prueba"}
	if canalValido(tcp, false) {
		t.Fatal("aceptó TCP sin TLS")
	}
	tcp.TLSConfig = &tls.Config{ServerName: "db.ejemplo", MinVersion: tls.VersionTLS13}
	tcp.TLSConfig.InsecureSkipVerify = true // #nosec G402 -- caso negativo
	if canalValido(tcp, false) {
		t.Fatal("aceptó TLS sin verificar")
	}
	tcp.TLSConfig = &tls.Config{ServerName: "db.ejemplo", MinVersion: tls.VersionTLS13}
	if !canalValido(tcp, false) {
		t.Fatal("rechazó TLS verificado")
	}
	socket := &pgconn.Config{Host: "/run/postgresql", User: "login_prueba", Database: "vec_prueba"}
	if canalValido(socket, false) || !canalValido(socket, true) {
		t.Fatal("el socket no exige habilitación explícita")
	}
	for _, dsn := range []string{"host=db.ejemplo dbname=vec_prueba", "postgres://login_prueba@db.ejemplo/",
		"password='secreto host=db.ejemplo' user=login_prueba dbname=vec_prueba"} {
		if dsnDeclaraIdentidad(dsn) == nil {
			t.Fatal("aceptó DSN con identidad o base implícita")
		}
	}
}

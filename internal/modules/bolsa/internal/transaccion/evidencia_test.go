package transaccion

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestHuellaCanonicaDistingueParticionesAmbiguas(t *testing.T) {
	izquierda := HuellaCanonica("ab", "c")
	derecha := HuellaCanonica("a", "bc")
	if izquierda == derecha || len(izquierda) != 64 || len(derecha) != 64 {
		t.Fatal("la codificacion canonica no separa las partes")
	}
	if izquierda != HuellaCanonica("ab", "c") {
		t.Fatal("la huella canonica no es determinista")
	}
}

func TestReferenciasYTokensTienen256Bits(t *testing.T) {
	referencia, err := NuevaReferenciaOpaca()
	if err != nil {
		t.Fatalf("generar referencia: %v", err)
	}
	contenido, err := base64.RawURLEncoding.DecodeString(referencia)
	if err != nil || len(contenido) != 32 || strings.ContainsAny(referencia, "=+/\r\n") {
		t.Fatal("referencia opaca fuera del contrato")
	}
	token, err := GenerarTokenReserva()
	if err != nil || token.Validar() != nil {
		t.Fatal("token de reserva fuera del contrato")
	}
	huella := HuellaTokenReserva(token)
	if len(huella) != 64 || huella != HuellaTokenReserva(token) {
		t.Fatal("huella de token fuera del contrato")
	}
	otroToken, err := GenerarTokenReserva()
	if err != nil || HuellaTokenReserva(otroToken) == huella {
		t.Fatal("el generador reutilizo una capacidad de reserva")
	}
}

func TestMismoUsoExigeDecisionYHuellasExactas(t *testing.T) {
	base := UsoAutorizacion{
		DecisionRef: "decision:01", HuellaDecisionSHA256: strings.Repeat("a", 64),
		HuellaEfectoSHA256: strings.Repeat("b", 64),
	}
	if !MismoUso(base, base) {
		t.Fatal("el mismo uso valido no coincide")
	}
	mutado := base
	mutado.HuellaEfectoSHA256 = strings.Repeat("c", 64)
	if MismoUso(base, mutado) {
		t.Fatal("se acepto reutilizar la decision para otro efecto")
	}
}

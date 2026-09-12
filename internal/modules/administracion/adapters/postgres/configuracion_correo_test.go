package postgres

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAADConfiguracionCorreoLigaReferenciaYVersion(t *testing.T) {
	primera, err := aadConfiguracionCorreo(1)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := aadConfiguracionCorreo(2)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(primera, segunda) || !strings.Contains(string(primera), referenciaConfiguracionCorreo) {
		t.Fatalf("AAD no liga referencia y versión: %q / %q", primera, segunda)
	}
}

func TestSobreJSONNoContieneTextoClaro(t *testing.T) {
	sobre := SobreSecretoCorreo{Version: 3, ClaveRef: "kms:admin:v1", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: []byte("texto-cifrado")}
	contenido := sobreJSON(sobre, true)
	if len(contenido) == 0 || bytes.Contains(contenido, []byte("secreto-supersecreto")) || !bytes.Contains(contenido, []byte("huella_aad_sha256")) {
		t.Fatalf("serialización de sobre inválida: %s", contenido)
	}
}

func TestConstructorFallaCerradoSinDependencias(t *testing.T) {
	if registro, err := NuevoRegistroConfiguracionCorreoPostgreSQL(nil, nil); registro != nil || !errors.Is(err, ErrConfiguracionCorreoNoDisponible) {
		t.Fatalf("constructor permisivo: registro=%v err=%v", registro, err)
	}
}

func TestUsoSecretoSinRegistroFallaCerrado(t *testing.T) {
	var registro *RegistroConfiguracionCorreoPostgreSQL
	if err := registro.UsarSecretoCorreo(context.Background(), nil, func([]byte) error { return nil }); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) {
		t.Fatalf("uso sin protector no denegado: %v", err)
	}
}

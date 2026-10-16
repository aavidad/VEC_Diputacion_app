package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreflightFirmaV2AcotaPDFDeEntradaAntesDePublicarDescriptor(t *testing.T) {
	s, _, _, q := prepararPreflightEntradaV2(t, 0)
	r, err := s.consultarNominalConEntrada(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64), true)
	if err != nil {
		t.Fatal(err)
	}
	r.EntradaContenido = append([]byte("%PDF-"), bytes.Repeat([]byte("x"), ports.MaximoDocumentoFirmaBytes-5)...)
	suma := sha256.Sum256(r.EntradaContenido)
	r.EntradaDocumentoHuella = hex.EncodeToString(suma[:])
	if err := ValidarResultadoPreflightFirmaR5V2(r, q); err != nil {
		t.Fatalf("entrada de 2 MiB y huella exacta rechazada: %v", err)
	}
	r.EntradaContenido = append(r.EntradaContenido, 'x')
	if err := ValidarResultadoPreflightFirmaR5V2(r, q); err == nil {
		t.Fatal("entrada mayor de 2 MiB publicada")
	}
}

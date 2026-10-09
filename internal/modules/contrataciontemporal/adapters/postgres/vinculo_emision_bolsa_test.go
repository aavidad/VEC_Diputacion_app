package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCodecVinculoEmisionBolsaConservaCanonYHuella(t *testing.T) {
	llamamiento := "llamamiento:" + strings.Repeat("a", 64)
	s := ports.SolicitudVinculoEmisionBolsa{OrganizacionRef: "organizacion:prueba",
		ExpedienteRef: "expediente:prueba", VersionEsperada: 4,
		BolsaRef: "bolsa:prueba", LlamamientoRef: llamamiento,
		ReciboEmisionRef:  "recibo:" + llamamiento,
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
	r := &RepositorioVinculoEmisionBolsaPostgreSQL{pool: nil}
	if _, _, err := r.CodificarMaterialVinculoEmisionBolsa(s); !errors.Is(err, ports.ErrVinculoEmisionBolsaInvalido) {
		t.Fatal("codec sin repositorio disponible")
	}
	r.pool = &pgxpool.Pool{} // El codec puro no abre conexiones.
	contenido, huella, err := r.CodificarMaterialVinculoEmisionBolsa(s)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.ct.vinculo-emision-bolsa.v1","organizacion_ref":"organizacion:prueba",` +
		`"expediente_ref":"expediente:prueba","version_esperada":4,"bolsa_ref":"bolsa:prueba",` +
		`"llamamiento_ref":"` + llamamiento + `","recibo_emision_ref":"recibo:` + llamamiento + `",` +
		`"clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`
	if !bytes.Equal(contenido, []byte(esperado)) {
		t.Fatal("cambió el canon material CT201")
	}
	suma := sha256.Sum256(contenido)
	if huella != hex.EncodeToString(suma[:]) {
		t.Fatal("cambió la huella del canon")
	}
}

func TestVinculoEmisionBolsaSQL22023EsEntradaInvalida(t *testing.T) {
	err := normalizarErrorVinculoBolsa(t.Context(), &pgconn.PgError{Code: "22023"})
	if !errors.Is(err, ports.ErrVinculoEmisionBolsaInvalido) {
		t.Fatal("22023 no se tradujo a 422 nominal")
	}
}

package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// La lectura real de PostgreSQL ya acreditó este caso. Esta regresión aislada
// usa el mismo codec binario por defecto, sin conexión ni efectos de negocio.
func TestReciboIntegracionDesarrolloTimestamptzLocalNormalizaUTC(t *testing.T) {
	esperado := time.Date(2026, time.September, 5, 0, 7, 16, 220132000, time.UTC)
	tipos := pgtype.NewMap()
	binario, err := tipos.Encode(pgtype.TimestamptzOID, pgtype.BinaryFormatCode,
		pgtype.Timestamptz{Time: esperado, Valid: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var recibo ports.ReciboLlamamientoDesarrollo
	if err := tipos.Scan(pgtype.TimestamptzOID, pgtype.BinaryFormatCode,
		binario, &recibo.ConfirmadaEn); err != nil {
		t.Fatal(err)
	}
	if recibo.ConfirmadaEn.Location() != time.Local ||
		recibo.ConfirmadaEn.Location() == time.UTC {
		t.Fatalf("el codec por defecto no reprodujo Local: %v", recibo.ConfirmadaEn.Location())
	}
	if !instantePostgreSQLLlamamientoValido(recibo.ConfirmadaEn) {
		t.Fatal("el valor leído debe superar la guarda de precisión PostgreSQL")
	}
	leido := recibo.ConfirmadaEn
	recibo.ConfirmadaEn = recibo.ConfirmadaEn.UTC()
	if recibo.ConfirmadaEn.Location() != time.UTC ||
		!instantePostgreSQLLlamamientoValido(recibo.ConfirmadaEn) ||
		recibo.ConfirmadaEn != esperado || !recibo.ConfirmadaEn.Equal(leido) ||
		recibo.ConfirmadaEn.UnixMicro() != leido.UnixMicro() {
		t.Fatal("normalizar debe entregar UTC sin alterar el instante ni la precisión")
	}
}

package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func TestPrelecturaCircuitoNoAbreSQLSinIdentidadCompetente(t *testing.T) {
	tx := &txBorradorPrueba{}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	s := dietasports.SolicitudPrelecturaCircuito{Referencia: "dco_" + strings.Repeat("b", 22), Etapa: domain.EtapaRevision, UnidadRef: "unidad:prueba"}
	_, err := repo.Preleer(context.Background(), dietasports.IdentidadEfectivaPrelecturaCircuito{}, s)
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || pool.inicios != 0 || len(tx.consultas) != 0 {
		t.Fatalf("prelectura sin competencia abrió SQL: %v %d", err, pool.inicios)
	}
}

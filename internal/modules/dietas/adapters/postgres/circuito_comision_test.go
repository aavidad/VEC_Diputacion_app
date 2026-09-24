package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func TestCircuitoDeniegaIdentidadNoAcreditadaAntesDeSQL(t *testing.T) {
	tx := &txBorradorPrueba{}
	pool := &poolBorradorPrueba{tx: tx}
	repo, _ := nuevoRepositorioBorradorComisionPostgreSQL(pool)
	ref := "dco_" + strings.Repeat("b", 22)
	orden := dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:prueba", Etapa: domain.EtapaRevision, Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_revision_0001", VersionEsperada: 2}
	if _, err := repo.Decidir(context.Background(), dietasports.IdentidadEfectivaCircuito{}, orden); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("decisión sin identidad: %v", err)
	}
	lista := dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:prueba", Limite: 20}
	if _, err := repo.ListarPendientes(context.Background(), dietasports.IdentidadEfectivaCircuito{}, lista); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("bandeja sin identidad: %v", err)
	}
	if pool.inicios != 0 || len(tx.consultas) != 0 {
		t.Fatal("se abrió transacción sin identidad")
	}
}

func TestCircuitoTraduceConflictoYNoFiltraMensajeSQL(t *testing.T) {
	err := normalizarErrorCircuito(context.Background(), &pgconn.PgError{Code: "PD005", Message: "detalle privado"}, true)
	if !errors.Is(err, dietasports.ErrEstadoCircuitoConflicto) || strings.Contains(err.Error(), "privado") {
		t.Fatalf("error SQL expuesto: %v", err)
	}
}

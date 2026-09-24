package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type repoPrelecturaPrueba struct{ llamadas int }

func (r *repoPrelecturaPrueba) Preleer(context.Context, dietasports.IdentidadEfectivaPrelecturaCircuito, dietasports.SolicitudPrelecturaCircuito) (dietasports.ContextoComisionCircuito, error) {
	r.llamadas++
	return dietasports.ContextoComisionCircuito{}, nil
}

func TestPrelecturaExigeUnidadCompetenteYV3AntesDeRepositorio(t *testing.T) {
	r := &repoPrelecturaPrueba{}
	s, err := NuevoServicioPrelecturaCircuito(r)
	if err != nil {
		t.Fatal(err)
	}
	q := dietasports.SolicitudPrelecturaCircuito{Referencia: "dco_" + strings.Repeat("a", 22), Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno"}
	if err := ValidarSolicitudPrelecturaCircuito(q); err != nil {
		t.Fatal(err)
	}
	_, err = s.Preleer(context.Background(), dietasports.IdentidadEfectivaPrelecturaCircuito{}, q)
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || r.llamadas != 0 {
		t.Fatalf("sin V3: %v %d", err, r.llamadas)
	}
	q.UnidadRef = ""
	if !errors.Is(ValidarSolicitudPrelecturaCircuito(q), domain.ErrDecisionCircuitoInvalida) {
		t.Fatal("unidad vacía aceptada")
	}
}

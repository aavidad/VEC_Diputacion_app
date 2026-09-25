package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestDecodificarCompetenciasRechazaCardinalidadYCamposExtra(t *testing.T) {
	fecha, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	s := personaldomain.SolicitudCompetenciasAsignacionDietas{FechaReferencia: fecha}
	e := personalports.EvidenciaConsultaCompetenciasAsignacionDietas{
		ReciboRef: "rca_" + strings.Repeat("a", 32), DecisionRef: "decision:uno", EfectoRef: "efecto:uno",
		ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:uno", ConsultadaEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
	}
	objeto := map[string]any{"asignacion_ref": "ads_" + strings.Repeat("a", 24), "relacion_ref": "rel_" + strings.Repeat("a", 24), "unidad_ref": "unidad_sintetica", "rol": "responsable", "vigente_desde": "2026-09-01", "version": 1}
	b, _ := json.Marshal([]any{objeto})
	emitida, expira := e.ConsultadaEn.Add(-time.Second), e.ConsultadaEn.Add(time.Second)
	if r, err := decodificarCompetenciasAsignacion(b, 1, s, "decision:uno", "efecto:uno", emitida, expira, e); err != nil || len(r.Competencias) != 1 {
		t.Fatalf("contrato: %v", err)
	}
	if _, err := decodificarCompetenciasAsignacion(b, 0, s, "decision:uno", "efecto:uno", emitida, expira, e); err == nil {
		t.Fatal("aceptó cardinalidad distinta")
	}
	objeto["persona_ref"] = "per_" + strings.Repeat("b", 24)
	b, _ = json.Marshal([]any{objeto})
	if _, err := decodificarCompetenciasAsignacion(b, 1, s, "decision:uno", "efecto:uno", emitida, expira, e); err == nil {
		t.Fatal("filtró campo personal extra")
	}
}

func TestErroresCompetenciasDistinguenDenegacionDeDependencia(t *testing.T) {
	for _, tc := range []struct {
		codigo   string
		esperado error
	}{
		{"42501", personalports.ErrCompetenciasAsignacionDietasDenegadas},
		{"P7201", personalports.ErrCompetenciasAsignacionDietasDenegadas},
		{"P7202", personalports.ErrCompetenciasAsignacionDietasNoDisponibles},
	} {
		if got := normalizarErrorCompetenciasAsignacion(context.Background(), &pgconn.PgError{Code: tc.codigo}); !errors.Is(got, tc.esperado) {
			t.Fatalf("SQLSTATE %s: %v", tc.codigo, got)
		}
	}
}

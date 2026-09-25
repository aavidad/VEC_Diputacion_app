package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestListaCompetenteRevierteRespuestaAjenaAntesDeConfirmar(t *testing.T) {
	base := ordenP(t)
	actor := base.Material.Solicitud().Actor
	fecha, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	material, err := personaldomain.NuevoMaterialRectificacionesCompetentesDietas(personaldomain.SolicitudRectificacionesCompetentesDietas{Actor: actor, FechaReferencia: fecha})
	if err != nil {
		t.Fatal(err)
	}
	objeto := `[{"solicitud_ref":"srd_` + strings.Repeat("b", 32) + `","estado":"pendiente","persona_ref":"per_` + strings.Repeat("b", 24) + `","empleado_ref":"emp_` + strings.Repeat("b", 24) + `","relacion_ref":"rel_` + strings.Repeat("b", 24) + `","unidad_ref":"unidad:x","asignacion_ref":"ads_` + strings.Repeat("b", 24) + `","version_origen":1,"fecha_referencia":"2026-09-20","campos_a_revisar":["centro_ref"],"motivo_revision":"centro incorrecto","detalle_solicitado":"","registrada_en":"2026-09-20T10:00:00Z","asignacion_actual":{"asignacion_ref":"ads_` + strings.Repeat("b", 24) + `","centro_ref":"centro:x","administrativo_persona_ref":"per_` + strings.Repeat("c", 24) + `","responsable_persona_ref":"per_` + strings.Repeat("d", 24) + `","grupo_dieta":2,"vigente_desde":"2026-09-01","version":1}}]`
	tx := &txP{fila: filaP{vals: []any{
		"rrd_" + strings.Repeat("a", 32), "dec_prueba", "per_" + strings.Repeat("a", 24),
		strings.Repeat("a", 64), "auditoria:uno", time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), 1, []byte(objeto),
	}}}
	repo := &RepositorioRectificacionesCompetentesDietasPostgreSQL{pool: &poolP{tx: tx}}
	_, err = repo.ConsultarRectificacionesCompetentesDietas(context.Background(), personalports.OrdenConsultaRectificacionesCompetentesDietas{Material: material, Autorizacion: base.Autorizacion})
	if !errors.Is(err, personalports.ErrRectificacionDietasNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("respuesta ajena: err=%v commits=%d rollbacks=%d", err, tx.commits, tx.rollbacks)
	}
}

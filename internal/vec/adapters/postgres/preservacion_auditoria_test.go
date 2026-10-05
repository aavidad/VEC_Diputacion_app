package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestPreservacionNoEntregaDatosConCommitIncierto(t *testing.T) {
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	for _, publicar := range []bool{true, false} {
		tx := &txPeriodicoPrueba{raw: []byte(`{"estado":"publicada","configuracion_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`), commitErr: errors.New("respuesta_perdida")}
		p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{tx}}
		f := &FuentePreservacionAuditoriaPostgreSQL{pool: p, rol: "vec_auditoria_preservacion_configurador"}
		var r domain.ResultadoPreservacionAuditoria
		var err error
		if publicar {
			r, err = f.PublicarPreservacionAuditoria(ctx, domain.SolicitudPreservacionAuditoria{PublicacionRef: "preservacion_" + strings.Repeat("1", 32), Version: 1, PreimagenSHA256: strings.Repeat("0", 64), DecisionTecnicaRef: "decision_tecnica_" + strings.Repeat("2", 32), Estado: "provisional", Medida: "conservar_todo_sin_expurgo"})
		} else {
			r, err = f.ConsultarPreservacionAuditoria(ctx, 0)
		}
		if !errors.Is(err, ErrPreservacionCommitIndeterminado) || r != (domain.ResultadoPreservacionAuditoria{}) || p.usados != 1 {
			t.Fatal("datos o intento falso tras COMMIT incierto")
		}
	}
}

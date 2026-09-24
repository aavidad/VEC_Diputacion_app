package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestConsultaRectificacionSinSolicitudConservaAuditoria(t *testing.T) {
	base := ordenP(t)
	fecha, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	m, err := personaldomain.NuevoMaterialRectificacionDietas(personaldomain.SolicitudRectificacionDietas{
		Actor: base.Material.Solicitud().Actor, Operacion: personaldomain.ConsultarRectificacionDietas,
		RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad:x", FechaReferencia: fecha,
	})
	if err != nil {
		t.Fatal(err)
	}
	tx := &txP{fila: filaP{err: pgx.ErrNoRows}}
	repo, err := nuevoRepositorioRectificacionDietasPostgreSQL(&poolP{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.EjecutarRectificacionDietas(context.Background(), personalports.OrdenRectificacionDietas{Material: m, Autorizacion: base.Autorizacion})
	if !errors.Is(err, personalports.ErrRectificacionDietasNoEncontrada) || tx.commits != 1 || tx.rollbacks != 0 || len(tx.q) != 2 || tx.q[1] != consultarRectificacionDietasSQL {
		t.Fatalf("lectura sin solicitud: err=%v commits=%d rollbacks=%d consultas=%v", err, tx.commits, tx.rollbacks, tx.q)
	}
}

func TestConfirmacionRectificacionUsaUnaTransaccionDosMateriales(t *testing.T) {
	base := ordenP(t)
	actor := base.Material.Solicitud().Actor
	fecha, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	clave := "12345678-1234-4123-8123-123456789abc"
	ref := "srd_" + strings.Repeat("b", 32)
	persona, empleado := "per_"+strings.Repeat("b", 24), "emp_"+strings.Repeat("b", 24)
	correccion := personaldomain.SolicitudAsignacionDietas{
		Actor: actor, Operacion: personaldomain.CorregirAsignacionDietas,
		PersonaRef: persona, EmpleadoRef: empleado, RelacionRef: "rel_" + strings.Repeat("b", 24),
		UnidadRef: "unidad:x", FechaReferencia: fecha, VersionEsperada: 1,
		ClaveIdempotencia: clave, CentroRef: "centro:nuevo",
		AdministrativoPersonaRef: "per_" + strings.Repeat("a", 24),
		ResponsablePersonaRef:    "per_" + strings.Repeat("c", 24), GrupoDieta: 2,
		VigenteDesde: fecha, MotivoRevision: "correccion autorizada", ProcedenciaActoRef: ref,
	}
	materialCorreccion, err := personaldomain.NuevoMaterialAsignacionDietas(correccion)
	if err != nil {
		t.Fatal(err)
	}
	s := personaldomain.SolicitudRectificacionDietas{
		Actor: actor, Operacion: personaldomain.ConfirmarRectificacionDietas,
		PersonaRef: persona, EmpleadoRef: empleado, RelacionRef: correccion.RelacionRef,
		UnidadRef: correccion.UnidadRef, AsignacionRef: "ads_" + strings.Repeat("b", 24),
		VersionEsperada: 1, FechaReferencia: fecha, ClaveIdempotencia: clave,
		SolicitudRef: ref, MotivoRevision: correccion.MotivoRevision, Correccion: &correccion,
	}
	material, err := personaldomain.NuevoMaterialRectificacionDietas(s)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	tx := &txP{fila: filaAsignacionP{valores: []any{
		ref, "rrd_" + strings.Repeat("a", 32), "confirmada", ahora,
		s.AsignacionRef, int64(1), "dec_prueba", "rel_" + strings.Repeat("b", 24), strings.Repeat("a", 64), "auditoria:uno",
	}}}
	repo, err := nuevoRepositorioRectificacionDietasPostgreSQL(&poolP{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.EjecutarRectificacionDietas(context.Background(), personalports.OrdenRectificacionDietas{
		Material: material, Autorizacion: base.Autorizacion,
		Correccion: materialCorreccion, AutorizacionCorreccion: base.Autorizacion,
	})
	if err != nil || tx.commits != 1 || tx.rollbacks != 0 || len(tx.a) != 1 || len(tx.a[0]) != 22 || tx.q[1] != confirmarRectificacionDietasSQL {
		t.Fatalf("confirmación no atómica: err=%v commits=%d rollbacks=%d argumentos=%d query=%v", err, tx.commits, tx.rollbacks, len(tx.a[0]), tx.q)
	}
}

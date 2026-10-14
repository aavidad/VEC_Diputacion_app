package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaAsignacionP struct {
	valores []any
	err     error
}

func (f filaAsignacionP) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	for i, destino := range destinos {
		switch p := destino.(type) {
		case *string:
			*p = f.valores[i].(string)
		case *time.Time:
			*p = f.valores[i].(time.Time)
		case *int16:
			*p = f.valores[i].(int16)
		case *int64:
			*p = f.valores[i].(int64)
		default:
			return errors.New("destino no previsto")
		}
	}
	return nil
}

func ordenAsignacionP(t *testing.T) personalports.OrdenAsignacionDietas {
	t.Helper()
	base := ordenP(t)
	fecha, err := personaldomain.NuevaFechaCivil("2026-09-20")
	if err != nil {
		t.Fatal(err)
	}
	s := personaldomain.SolicitudAsignacionDietas{Actor: base.Material.Solicitud().Actor, Operacion: personaldomain.ConsultarAsignacionDietas, RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad:x", FechaReferencia: fecha}
	m, err := personaldomain.NuevoMaterialAsignacionDietas(s)
	if err != nil {
		t.Fatal(err)
	}
	a := atestacionAsignacionP(t, m, base.Autorizacion,
		personalports.AccionConsultarAsignacionDietas, personalports.AudienciaConsultarAsignacionDietas)
	return personalports.OrdenAsignacionDietas{Material: m, Autorizacion: a}
}

func ordenConsultaPropiaAsignacionP(t *testing.T) personalports.OrdenAsignacionDietas {
	t.Helper()
	o := ordenAsignacionP(t)
	s := o.Material.Solicitud()
	s.Operacion = personaldomain.ConsultarPropiaAsignacionDietas
	m, err := personaldomain.NuevoMaterialAsignacionDietasPropia(s)
	if err != nil {
		t.Fatal(err)
	}
	a := atestacionAsignacionP(t, m, o.Autorizacion,
		personalports.AccionConsultarPropiaAsignacionDietas, personalports.AudienciaConsultarPropiaAsignacionDietas)
	return personalports.OrdenAsignacionDietas{Material: m, Autorizacion: a}
}

func atestacionAsignacionP(t *testing.T, m personaldomain.MaterialAsignacionDietas,
	base vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, audiencia string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	x := base.ResumenCapacidad()
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(),
		x.DecisionHuellaSHA256(), x.MotivoHuellaSHA256(), x.ContextoRef(), x.ContextoHuellaSHA256(),
		accion, m.Recurso().Referencia, h, audiencia, x.EmitidaEn(), x.ExpiraEn())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(base.CapacidadCanonica(),
		resumen, base.DecisionCanonica(), base.MotivoCanonico(),
		base.ContextoActorCanonico(), base.PersonaVersion(), base.PerfilVersion(),
		base.PayloadVECAD3(), base.SobreCOSESign1(),
		base.EvidenciaVerificacion(), base.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func filaAsignacionCorrecta(o personalports.OrdenAsignacionDietas) filaAsignacionP {
	s := o.Material.Solicitud()
	a := o.Autorizacion.ResumenCapacidad()
	return filaAsignacionP{valores: []any{
		"rad_" + strings.Repeat("a", 32), a.DecisionRef(), a.EfectoRef(), strings.Repeat("a", 64), "auditoria:uno", a.EmitidaEn().Add(time.Microsecond), "consultada",
		"ads_" + strings.Repeat("a", 24), s.RelacionRef, s.PersonaRef, s.UnidadRef, "centro:uno", "per_" + strings.Repeat("b", 24), "per_" + strings.Repeat("c", 24), int16(2), "2026-09-01", int64(1),
	}}
}

func TestRepositorioAsignacionUsaFachadaYTransaccion(t *testing.T) {
	o := ordenAsignacionP(t)
	tx := &txP{fila: filaAsignacionCorrecta(o)}
	pool := &poolP{tx: tx}
	repo, err := nuevoRepositorioAsignacionDietasPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.EjecutarAsignacionDietas(context.Background(), o)
	if err != nil || r.EstadoLocal != "consultada" || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.q[1] != consultaAsignacionDietasSQL {
		t.Fatalf("fachada: err=%v estado=%s commits=%d rollback=%d query=%v", err, r.EstadoLocal, tx.commits, tx.rollbacks, tx.q)
	}
}

func TestRepositorioConsultaPropiaSeleccionaFachadaYRechazaAudienciaAntigua(t *testing.T) {
	o := ordenConsultaPropiaAsignacionP(t)
	tx := &txP{fila: filaAsignacionCorrecta(o)}
	pool := &poolP{tx: tx}
	repo, err := nuevoRepositorioAsignacionDietasPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.EjecutarAsignacionDietas(context.Background(), o)
	if err != nil || r.EstadoLocal != "consultada" || tx.commits != 1 || tx.rollbacks != 0 ||
		pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite ||
		len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.q[1] != consultaPropiaAsignacionDietasSQL ||
		tx.a[0][0] != string(o.Material.Canonico()) {
		t.Fatalf("consulta propia: err=%v estado=%s query=%v", err, r.EstadoLocal, tx.q)
	}
	antigua := ordenAsignacionP(t)
	o.Autorizacion = antigua.Autorizacion
	tx2 := &txP{fila: filaAsignacionCorrecta(o)}
	repo, _ = nuevoRepositorioAsignacionDietasPostgreSQL(&poolP{tx: tx2})
	if _, err := repo.EjecutarAsignacionDietas(context.Background(), o); !errors.Is(err, personalports.ErrAsignacionDietasDenegada) || len(tx2.q) != 0 {
		t.Fatalf("consulta propia aceptó audiencia antigua: %v, sentencias=%v", err, tx2.q)
	}
	materialAntiguo := antigua.Material
	aNueva := atestacionAsignacionP(t, materialAntiguo, antigua.Autorizacion,
		personalports.AccionConsultarPropiaAsignacionDietas, personalports.AudienciaConsultarPropiaAsignacionDietas)
	antigua.Autorizacion = aNueva
	tx3 := &txP{fila: filaAsignacionCorrecta(antigua)}
	repo, _ = nuevoRepositorioAsignacionDietasPostgreSQL(&poolP{tx: tx3})
	if _, err := repo.EjecutarAsignacionDietas(context.Background(), antigua); !errors.Is(err, personalports.ErrAsignacionDietasDenegada) || len(tx3.q) != 0 {
		t.Fatalf("consulta antigua aceptó audiencia propia: %v, sentencias=%v", err, tx3.q)
	}
}

func TestRepositorioAsignacionRevierteRespuestaNoConfiable(t *testing.T) {
	o := ordenAsignacionP(t)
	f := filaAsignacionCorrecta(o)
	f.valores[9] = "per_" + strings.Repeat("f", 24)
	tx := &txP{fila: f}
	repo, err := nuevoRepositorioAsignacionDietasPostgreSQL(&poolP{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.EjecutarAsignacionDietas(context.Background(), o)
	if !errors.Is(err, personalports.ErrAsignacionDietasNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("resultado ajeno: err=%v commits=%d rollback=%d", err, tx.commits, tx.rollbacks)
	}
}

func TestErroresAsignacionNoFiltranPostgres(t *testing.T) {
	for _, tc := range []struct {
		codigo string
		want   error
	}{
		{"P7201", personalports.ErrAsignacionDietasDenegada},
		{"P7203", personalports.ErrVersionAsignacionDietas},
		{"P7204", personalports.ErrClaveAsignacionDietas},
		{"42501", personalports.ErrAsignacionDietasDenegada},
		{"22023", personalports.ErrAsignacionDietasNoDisponible},
	} {
		got := normalizarErrorAsignacionDietas(context.Background(), &pgconn.PgError{Code: tc.codigo})
		if !errors.Is(got, tc.want) {
			t.Errorf("SQLSTATE %s: %v", tc.codigo, got)
		}
	}
}

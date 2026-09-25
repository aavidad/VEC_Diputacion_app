package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestErrorResolucionTraduceSoloRechazosNominales(t *testing.T) {
	ctx := context.Background()
	for codigo, esperado := range map[string]error{
		"PC001": ports.ErrSolicitudCronosInvalida,
		"PC002": ports.ErrClaveOperacionEnConflicto,
		"PC003": ports.ErrDependenciaNoDisponible,
		"PC011": ports.ErrResolucionEstadoCambiado,
		"PC012": ports.ErrResolucionNoCompetente,
		"PC013": ports.ErrDependenciaNoDisponible,
		"PC014": ports.ErrResolucionPendienteAsignacion,
		"42501": ports.ErrDependenciaNoDisponible,
		"P0002": ports.ErrDependenciaNoDisponible,
	} {
		if err := errorResolucion(ctx, &pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := errorResolucion(cancelado, &pgconn.PgError{Code: "PC012"}); !errors.Is(err, context.Canceled) {
		t.Fatal("la cancelación no se conserva", err)
	}
}

func TestRepositoriosResolucionFallanCerradosSinOrden(t *testing.T) {
	ctx := context.Background()
	if _, err := NuevoRepositorioResolucionPermisos(nil); err == nil {
		t.Fatal("repositorio sin pool")
	}
	if _, err := NuevoRepositorioAvisosPropios(nil); err == nil {
		t.Fatal("repositorio sin pool")
	}
	res := &RepositorioResolucionPermisos{db: iniciadorNulo{}}
	if _, err := res.ConsultarBandeja(ctx, ports.OrdenResolucionPermisos{}, domain.MaterialBandejaPermisos{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := res.ResolverPermiso(ctx, ports.OrdenResolucionPermisos{}, domain.MaterialResolucionPermiso{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	av := &RepositorioAvisosPropios{db: iniciadorNulo{}}
	if _, err := av.ConsultarAvisos(ctx, ports.OrdenAvisosPropios{}, domain.MaterialConsultaAvisosPropios{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := av.ArchivarAviso(ctx, ports.OrdenAvisosPropios{}, domain.MaterialArchivoAvisoPropio{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}

type proveedorResolucionPrueba struct {
	t         *testing.T
	audiencia string
	llamadas  int
}

func (p *proveedorResolucionPrueba) ProveerMaterialBandejaPermisos(_ context.Context, m domain.MaterialBandejaPermisos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoBandejaPermisos(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultarBandeja, p.audiencia, r), nil
}

func (p *proveedorResolucionPrueba) ProveerMaterialResolucionPermiso(_ context.Context, m domain.MaterialResolucionPermiso) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoResolucionPermiso(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionResolverPermiso, p.audiencia, r), nil
}

const bandejaSQLPrueba = `{"paso":"responsable","pendientes":[{"solicitud_ref":"permiso:cronos:solicitud:perm-va-00001","empleado_ref":"emp_AAAAAAAAAAAAAAAAAAAAAA","empleado_etiqueta":null,"permiso_ref":"permiso:cronos:vacaciones","nombre":"Vacaciones","circuito":"J-A","pendiente_asignacion":false,"justificante_exigido":false,"desde":"2026-10-05","hasta":"2026-10-07","hora_inicio":null,"hora_fin":null,"cantidad":3,"unidad":"dia","estado":"solicitado","version":1,"solicitada_en":"2026-09-24T08:00:00.123456+00:00"}]}`

const reciboResolucionSQLPrueba = `{"resolucion_ref":"permiso:cronos:resolucion:res-va-00001","solicitud_ref":"permiso:cronos:solicitud:perm-va-00001","recibo_ref":"recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f","estado":"pendiente_administracion","version":2,"instante_utc":"2026-09-25T08:00:00.123456+00:00","replay":false}`

func TestResolucionConsumeV3YEnviaElMaterialExacto(t *testing.T) {
	actor := actorPrueba(t, empleadoPrueba)
	m := domain.MaterialResolucionPermiso{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, ClaveOperacion: "res-va-00001",
		SolicitudRef: "permiso:cronos:solicitud:perm-va-00001", Paso: domain.PasoResponsable, Decision: domain.DecisionAprobar, VersionEsperada: 1, ZonaHoraria: domain.ZonaSaldoPeninsula}
	ajena := &proveedorResolucionPrueba{t: t, audiencia: application.AudienciaSolicitudPermisoPropio}
	orden, _ := ports.NuevaOrdenResolucionPermisos(actor, ajena)
	r := &RepositorioResolucionPermisos{db: dbLecturaPrueba{t: t}}
	if _, err := r.ResolverPermiso(context.Background(), orden, m); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta V3 de otra audiencia", err)
	}
	otro := m
	otro.EmpleadoRef = "emp_otroempleado0123456789"
	if _, err := r.ResolverPermiso(context.Background(), orden, otro); !errors.Is(err, ports.ErrDependenciaNoDisponible) || ajena.llamadas != 1 {
		t.Fatal("resuelve con un empleado propio ajeno a la sesión", err)
	}
	tx := &txLecturaPrueba{respuestas: [][]byte{[]byte(reciboResolucionSQLPrueba)}}
	r = &RepositorioResolucionPermisos{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenResolucionPermisos(actor, &proveedorResolucionPrueba{t: t, audiencia: application.AudienciaResolucionPermiso})
	recibo, err := r.ResolverPermiso(context.Background(), orden, m)
	esperado, _ := m.Canonico()
	if err != nil || tx.commits != 1 || tx.consultas[0] != consultaResolverPermiso || tx.materiales[0] != string(esperado) ||
		recibo.Estado != domain.EstadoPermisoPendienteAdministracion || recibo.Version != 2 || recibo.InstanteUTC.Location() != time.UTC {
		t.Fatal(err, tx.commits, recibo)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(bandejaSQLPrueba)}}
	r = &RepositorioResolucionPermisos{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenResolucionPermisos(actor, &proveedorResolucionPrueba{t: t, audiencia: application.AudienciaBandejaPermisos})
	mb := domain.MaterialBandejaPermisos{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, Paso: domain.PasoResponsable, ZonaHoraria: domain.ZonaSaldoPeninsula}
	b, err := r.ConsultarBandeja(context.Background(), orden, mb)
	if err != nil || len(b.Pendientes) != 1 || b.Pendientes[0].EmpleadoEtiqueta != "" || b.Pendientes[0].Circuito != domain.CircuitoResponsableAdministracion || tx.consultas[0] != consultaBandejaPermisos {
		t.Fatal(err, b)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(`{"paso":"responsable","pendientes":[],"extra":1}`)}}
	r = &RepositorioResolucionPermisos{db: dbLecturaPrueba{t: t, tx: tx}}
	if _, err := r.ConsultarBandeja(context.Background(), orden, mb); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta campos desconocidos de la fuente", err)
	}
}

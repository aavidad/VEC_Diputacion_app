package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorNotificacionesPrueba struct {
	t         *testing.T
	audiencia string
	llamadas  int
}

func (p *proveedorNotificacionesPrueba) ProveerMaterialRegistroNotificacion(_ context.Context, m domain.MaterialRegistroNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoRegistroNotificacion(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionRegistrarNotificacion, p.audiencia, r), nil
}
func (p *proveedorNotificacionesPrueba) ProveerMaterialConsultaNotificacionesPropias(_ context.Context, m domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoConsultaNotificacionesPropias(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultarNotificacionesPropias, p.audiencia, r), nil
}
func (p *proveedorNotificacionesPrueba) ProveerMaterialBandejaNotificaciones(_ context.Context, m domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoBandejaNotificaciones(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultarBandejaNotif, p.audiencia, r), nil
}
func (p *proveedorNotificacionesPrueba) ProveerMaterialAtencionNotificacion(_ context.Context, m domain.MaterialAtencionNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	r, err := application.RecursoAtencionNotificacion(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionAtenderNotificacion, p.audiencia, r), nil
}

const (
	notificacionesSQLPrueba        = `{"empleado_ref":"emp_0123456789abcdefghijkl","tipos":[{"tipo_version_ref":"notificacion:cronos:tipo:otra-comunicacion:sintetico-1","tipo_ref":"notificacion:cronos:tipo:otra-comunicacion","nombre":"Otra comunicación a RRHH"}],"notificaciones":[{"notificacion_ref":"notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001","tipo_ref":"notificacion:cronos:tipo:otra-comunicacion","tipo_nombre":"Otra comunicación a RRHH","fecha_referida":"2026-09-24","texto":"Texto","adjunto_ref":null,"adjunto_sha256":null,"registrada_en":"2026-09-24T08:00:00.123456+00:00","estado":"atendida","atendida_en":"2026-09-25T08:00:00+02:00"}]}`
	reciboNotificacionSQLPrueba    = `{"notificacion_ref":"notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001","recibo_ref":"recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f","instante_utc":"2026-09-25T08:00:00.123456+00:00","replay":true}`
	bandejaNotificacionesSQLPrueba = `{"notificaciones":[{"notificacion_ref":"notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001","empleado_ref":"emp_AAAAAAAAAAAAAAAAAAAAAA","empleado_etiqueta":null,"tipo_ref":"notificacion:cronos:tipo:otra-comunicacion","tipo_nombre":"Otra","fecha_referida":"2026-09-24","texto":"Texto","adjunto_ref":"registro:sintetico:0001","adjunto_sha256":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `","registrada_en":"2026-09-24T08:00:00+00:00","atendida":false,"atendida_en":null}]}`
	reciboAtencionSQLPrueba        = `{"atencion_ref":"notificacion:cronos:atencion:0f0e0d0c-0b0a-4000-8000-000000000002","notificacion_ref":"notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001","recibo_ref":"recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f","instante_utc":"2026-09-25T08:00:00.123456+00:00","replay":false}`
)

func TestErrorRegistroNotificacionTraduceTipoNoVigente(t *testing.T) {
	ctx := context.Background()
	for codigo, esperado := range map[string]error{
		"PC001": ports.ErrSolicitudCronosInvalida,
		"PC002": ports.ErrClaveOperacionEnConflicto,
		"PC011": ports.ErrTipoNotificacionNoVigente,
		"PC003": ports.ErrDependenciaNoDisponible,
	} {
		if err := errorRegistroNotificacion(ctx, &pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
}

func TestNotificacionesPropiasConsumenV3YEnvianElMaterialExacto(t *testing.T) {
	actor := actorPrueba(t, empleadoPrueba)
	m := domain.MaterialRegistroNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, ClaveOperacion: "not-a-00001",
		TipoVersionRef: "notificacion:cronos:tipo:otra-comunicacion:sintetico-1", FechaReferida: "2026-09-24", Texto: "Texto <con> símbolos", ZonaHoraria: domain.ZonaSaldoPeninsula}
	ajena := &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaArchivoAvisoPropio}
	orden, _ := ports.NuevaOrdenNotificacionesPropias(actor, ajena)
	r := &RepositorioNotificacionesPropias{db: dbLecturaPrueba{t: t}}
	if _, err := r.RegistrarNotificacion(context.Background(), orden, m); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta V3 de otra audiencia", err)
	}
	otro := m
	otro.EmpleadoRef = "emp_otroempleado0123456789"
	if _, err := r.RegistrarNotificacion(context.Background(), orden, otro); !errors.Is(err, ports.ErrDependenciaNoDisponible) || ajena.llamadas != 1 {
		t.Fatal("registra con un empleado ajeno a la sesión", err)
	}
	tx := &txLecturaPrueba{respuestas: [][]byte{[]byte(reciboNotificacionSQLPrueba)}}
	r = &RepositorioNotificacionesPropias{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenNotificacionesPropias(actor, &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaRegistroNotificacion})
	recibo, err := r.RegistrarNotificacion(context.Background(), orden, m)
	esperado, _ := m.Canonico()
	if err != nil || tx.commits != 1 || tx.consultas[0] != consultaRegistrarNotificacion || tx.materiales[0] != string(esperado) || !recibo.Replay || recibo.InstanteUTC.Location() != time.UTC {
		t.Fatal(err, tx.commits, recibo)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(notificacionesSQLPrueba)}}
	r = &RepositorioNotificacionesPropias{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenNotificacionesPropias(actor, &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaConsultaNotificacionesPropias})
	mc := domain.MaterialConsultaNotificaciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, ZonaHoraria: domain.ZonaSaldoPeninsula}
	c, err := r.ConsultarPropias(context.Background(), orden, mc)
	if err != nil || len(c.Tipos) != 1 || len(c.Notificaciones) != 1 || c.Notificaciones[0].AtendidaEnUTC == nil || c.Notificaciones[0].AtendidaEnUTC.Location() != time.UTC ||
		c.Notificaciones[0].AdjuntoRef != "" || tx.consultas[0] != consultaNotificacionesPropias {
		t.Fatal(err, c)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(strings.Replace(notificacionesSQLPrueba, `"empleado_ref":"emp_0123456789abcdefghijkl"`, `"empleado_ref":"emp_AAAAAAAAAAAAAAAAAAAAAA"`, 1))}}
	r = &RepositorioNotificacionesPropias{db: dbLecturaPrueba{t: t, tx: tx}}
	if _, err := r.ConsultarPropias(context.Background(), orden, mc); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta las notificaciones de otra persona", err)
	}
}

func TestBandejaDeNotificacionesConsumeV3DeRRHH(t *testing.T) {
	actor := actorPrueba(t, empleadoPrueba)
	mb := domain.MaterialConsultaNotificaciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, ZonaHoraria: domain.ZonaSaldoPeninsula}
	tx := &txLecturaPrueba{respuestas: [][]byte{[]byte(bandejaNotificacionesSQLPrueba)}}
	r := &RepositorioBandejaNotificaciones{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ := ports.NuevaOrdenBandejaNotificaciones(actor, &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaBandejaNotificaciones})
	b, err := r.ConsultarBandeja(context.Background(), orden, mb)
	if err != nil || len(b.Notificaciones) != 1 || b.Notificaciones[0].AdjuntoRef != "registro:sintetico:0001" || b.Notificaciones[0].Atendida || tx.consultas[0] != consultaBandejaNotificaciones {
		t.Fatal(err, b)
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(reciboAtencionSQLPrueba)}}
	r = &RepositorioBandejaNotificaciones{db: dbLecturaPrueba{t: t, tx: tx}}
	orden, _ = ports.NuevaOrdenBandejaNotificaciones(actor, &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaAtencionNotificacion})
	ma := domain.MaterialAtencionNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleadoPrueba, ClaveOperacion: "ate-r-00001",
		NotificacionRef: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001", ZonaHoraria: domain.ZonaSaldoPeninsula}
	recibo, err := r.AtenderNotificacion(context.Background(), orden, ma)
	esperado, _ := ma.Canonico()
	if err != nil || recibo.Replay || tx.materiales[0] != string(esperado) || tx.consultas[0] != consultaAtenderNotificacion {
		t.Fatal(err, recibo)
	}
	ajena, _ := ports.NuevaOrdenBandejaNotificaciones(actor, &proveedorNotificacionesPrueba{t: t, audiencia: application.AudienciaBandejaPermisos})
	r = &RepositorioBandejaNotificaciones{db: dbLecturaPrueba{t: t}}
	if _, err := r.ConsultarBandeja(context.Background(), ajena, mb); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta la audiencia de la bandeja de permisos", err)
	}
	if _, err := NuevoRepositorioBandejaNotificaciones(nil); err == nil {
		t.Fatal("repositorio sin pool")
	}
	if _, err := NuevoRepositorioNotificacionesPropias(nil); err == nil {
		t.Fatal("repositorio sin pool")
	}
	vacio := &RepositorioBandejaNotificaciones{db: iniciadorNulo{}}
	if _, err := vacio.AtenderNotificacion(context.Background(), ports.OrdenBandejaNotificaciones{}, ma); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}

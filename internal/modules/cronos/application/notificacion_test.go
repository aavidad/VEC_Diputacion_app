package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorNotificacionesPrueba struct{}

func (proveedorNotificacionesPrueba) ProveerMaterialRegistroNotificacion(context.Context, domain.MaterialRegistroNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorNotificacionesPrueba) ProveerMaterialConsultaNotificacionesPropias(context.Context, domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorNotificacionesPrueba) ProveerMaterialBandejaNotificaciones(context.Context, domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorNotificacionesPrueba) ProveerMaterialAtencionNotificacion(context.Context, domain.MaterialAtencionNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}

type repoNotificacionesPrueba struct {
	consulta  ports.ConsultaNotificacionesPropias
	recibo    ports.ReciboNotificacion
	bandeja   ports.BandejaNotificaciones
	atencion  ports.ReciboAtencionNotificacion
	registro  domain.MaterialRegistroNotificacion
	atendida  domain.MaterialAtencionNotificacion
	consultas []domain.MaterialConsultaNotificaciones
	llamadas  int
}

func (r *repoNotificacionesPrueba) ConsultarPropias(_ context.Context, _ ports.OrdenNotificacionesPropias, m domain.MaterialConsultaNotificaciones) (ports.ConsultaNotificacionesPropias, error) {
	r.consultas = append(r.consultas, m)
	r.llamadas++
	return r.consulta, nil
}
func (r *repoNotificacionesPrueba) RegistrarNotificacion(_ context.Context, _ ports.OrdenNotificacionesPropias, m domain.MaterialRegistroNotificacion) (ports.ReciboNotificacion, error) {
	r.registro = m
	r.llamadas++
	return r.recibo, nil
}
func (r *repoNotificacionesPrueba) ConsultarBandeja(_ context.Context, _ ports.OrdenBandejaNotificaciones, m domain.MaterialConsultaNotificaciones) (ports.BandejaNotificaciones, error) {
	r.consultas = append(r.consultas, m)
	r.llamadas++
	return r.bandeja, nil
}
func (r *repoNotificacionesPrueba) AtenderNotificacion(_ context.Context, _ ports.OrdenBandejaNotificaciones, m domain.MaterialAtencionNotificacion) (ports.ReciboAtencionNotificacion, error) {
	r.atendida = m
	r.llamadas++
	return r.atencion, nil
}

const refNotificacionPrueba = "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001"

func notificacionPropiaPrueba() ports.NotificacionPropia {
	return ports.NotificacionPropia{NotificacionRef: refNotificacionPrueba, TipoRef: "notificacion:cronos:tipo:incidencia-marcaje", TipoNombre: "Incidencia en el marcaje",
		FechaReferida: "2026-09-24", Texto: "No pude fichar", RegistradaEnUTC: time.Now().UTC(), Estado: ports.EstadoNotificacionRegistrada}
}

func TestNotificacionesPropiasDerivanEmpleadoYRechazanFuenteIncoherente(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, err := ports.NuevaOrdenNotificacionesPropias(actor, proveedorNotificacionesPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	tipo := ports.TipoNotificacion{TipoVersionRef: "notificacion:cronos:tipo:incidencia-marcaje:sintetico-1", TipoRef: "notificacion:cronos:tipo:incidencia-marcaje", Nombre: "Incidencia"}
	repo := &repoNotificacionesPrueba{consulta: ports.ConsultaNotificacionesPropias{Tipos: []ports.TipoNotificacion{tipo}, Notificaciones: []ports.NotificacionPropia{notificacionPropiaPrueba()}}}
	s, _ := NuevoServicioNotificacionesPropias(repo, relojMarcajePrueba{time.Now().UTC()}, zona)
	c, err := s.ConsultarPropias(context.Background(), orden)
	if err != nil || len(c.Notificaciones) != 1 || repo.consultas[0].EmpleadoRef != "emp_0123456789abcdefghijkl" {
		t.Fatalf("consulta distinta: %+v %v", c, err)
	}
	atendidaEn := time.Now().UTC()
	for nombre, alterar := range map[string]func(*ports.NotificacionPropia){
		"estado inventado":         func(n *ports.NotificacionPropia) { n.Estado = "resuelta" },
		"atendida sin instante":    func(n *ports.NotificacionPropia) { n.Estado = ports.EstadoNotificacionAtendida },
		"registrada con instante":  func(n *ports.NotificacionPropia) { n.AtendidaEnUTC = &atendidaEn },
		"texto largo":              func(n *ports.NotificacionPropia) { n.Texto = strings.Repeat("x", 513) },
		"adjunto a medias":         func(n *ports.NotificacionPropia) { n.AdjuntoRef = "registro:sintetico:0001" },
		"referencia de otra cosa":  func(n *ports.NotificacionPropia) { n.NotificacionRef = "notificacion:cronos:x" },
		"tipo con versión":         func(n *ports.NotificacionPropia) { n.TipoRef = tipo.TipoVersionRef },
		"sin instante de registro": func(n *ports.NotificacionPropia) { n.RegistradaEnUTC = time.Time{} },
	} {
		n := notificacionPropiaPrueba()
		alterar(&n)
		repo.consulta.Notificaciones = []ports.NotificacionPropia{n}
		if _, err := s.ConsultarPropias(context.Background(), orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
	repo.consulta = ports.ConsultaNotificacionesPropias{Tipos: []ports.TipoNotificacion{tipo}}
	if _, err := s.ConsultarPropias(context.Background(), orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta una lista ausente", err)
	}
	if _, err := s.ConsultarPropias(context.Background(), ports.OrdenNotificacionesPropias{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("consulta sin orden", err)
	}
}

func TestRegistroDeNotificacionValidaAntesDelRepositorioYCompruebaRecibo(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, _ := ports.NuevaOrdenNotificacionesPropias(actor, proveedorNotificacionesPrueba{})
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repoNotificacionesPrueba{recibo: ports.ReciboNotificacion{NotificacionRef: refNotificacionPrueba, ReciboRef: "recibo:cronos:x", InstanteUTC: ahora}}
	s, _ := NuevoServicioNotificacionesPropias(repo, relojMarcajePrueba{ahora}, zona)
	h := sha256.Sum256([]byte("documento sintético"))
	p := ports.PeticionRegistroNotificacion{ClaveOperacion: "not-a-00001", TipoVersionRef: "notificacion:cronos:tipo:incidencia-marcaje:sintetico-1",
		FechaReferida: "2026-09-24", Texto: "No pude fichar", AdjuntoRef: "registro:sintetico:0001", AdjuntoSHA256: hex.EncodeToString(h[:])}
	r, err := s.RegistrarNotificacion(context.Background(), orden, p)
	if err != nil || r.NotificacionRef != refNotificacionPrueba || repo.registro.EmpleadoRef != "emp_0123456789abcdefghijkl" || repo.registro.ZonaHoraria != "Europe/Madrid" {
		t.Fatalf("registro distinto: %+v %+v %v", r, repo.registro, err)
	}
	for nombre, alterar := range map[string]func(*ports.PeticionRegistroNotificacion){
		"sin huella":   func(p *ports.PeticionRegistroNotificacion) { p.AdjuntoSHA256 = "" },
		"texto vacío":  func(p *ports.PeticionRegistroNotificacion) { p.Texto = "" },
		"fecha mala":   func(p *ports.PeticionRegistroNotificacion) { p.FechaReferida = "24/09/2026" },
		"tipo sin ver": func(p *ports.PeticionRegistroNotificacion) { p.TipoVersionRef = "notificacion:cronos:tipo:x" },
	} {
		q := p
		alterar(&q)
		llamadas := repo.llamadas
		if _, err := s.RegistrarNotificacion(context.Background(), orden, q); !errors.Is(err, ports.ErrSolicitudCronosInvalida) || repo.llamadas != llamadas {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
	repo.recibo.InstanteUTC = ahora.In(zona)
	if _, err := s.RegistrarNotificacion(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un recibo fuera de UTC", err)
	}
}

func TestBandejaDeNotificacionesExcluyeLoPropioYAtencionCompruebaRecibo(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, err := ports.NuevaOrdenBandejaNotificaciones(actor, proveedorNotificacionesPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	fila := ports.NotificacionRecibida{NotificacionRef: refNotificacionPrueba, EmpleadoRef: "emp_AAAAAAAAAAAAAAAAAAAAAA", EmpleadoEtiqueta: "Persona sintética A",
		TipoRef: "notificacion:cronos:tipo:incidencia-marcaje", TipoNombre: "Incidencia", FechaReferida: "2026-09-24", Texto: "No pude fichar", RegistradaEnUTC: ahora}
	repo := &repoNotificacionesPrueba{bandeja: ports.BandejaNotificaciones{Notificaciones: []ports.NotificacionRecibida{fila}},
		atencion: ports.ReciboAtencionNotificacion{AtencionRef: "notificacion:cronos:atencion:0f0e0d0c-0b0a-4000-8000-000000000002", NotificacionRef: refNotificacionPrueba,
			ReciboRef: "recibo:cronos:y", InstanteUTC: ahora}}
	s, _ := NuevoServicioBandejaNotificaciones(repo, relojMarcajePrueba{ahora}, zona)
	if b, err := s.ConsultarBandeja(context.Background(), orden); err != nil || len(b.Notificaciones) != 1 {
		t.Fatal(b, err)
	}
	propia := fila
	propia.EmpleadoRef = "emp_0123456789abcdefghijkl"
	atendida := fila
	atendida.Atendida = true
	for nombre, f := range map[string]ports.NotificacionRecibida{"lo propio": propia, "atendida sin instante": atendida} {
		repo.bandeja.Notificaciones = []ports.NotificacionRecibida{f}
		if _, err := s.ConsultarBandeja(context.Background(), orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
	r, err := s.AtenderNotificacion(context.Background(), orden, ports.PeticionAtencionNotificacion{ClaveOperacion: "ate-r-00001", NotificacionRef: refNotificacionPrueba})
	if err != nil || r.ReciboRef != "recibo:cronos:y" || repo.atendida.EmpleadoRef != "emp_0123456789abcdefghijkl" {
		t.Fatalf("atención distinta: %+v %+v %v", r, repo.atendida, err)
	}
	repo.atencion.NotificacionRef = "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000009"
	if _, err := s.AtenderNotificacion(context.Background(), orden, ports.PeticionAtencionNotificacion{ClaveOperacion: "ate-r-00001", NotificacionRef: refNotificacionPrueba}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un recibo de otra notificación", err)
	}
	llamadas := repo.llamadas
	if _, err := s.AtenderNotificacion(context.Background(), orden, ports.PeticionAtencionNotificacion{ClaveOperacion: "x", NotificacionRef: refNotificacionPrueba}); !errors.Is(err, ports.ErrSolicitudCronosInvalida) || repo.llamadas != llamadas {
		t.Fatal("clave corta llega al repositorio", err)
	}
}

func TestRecursosDeNotificacionesLiganPersonaOPasoYHuella(t *testing.T) {
	m := domain.MaterialConsultaNotificaciones{ActorRef: "per_RRRRRRRRRRRRRRRRRRRRRR", PerfilRef: "prf_QQQQQQQQQQQQQQQQQQQQQQ", EmpleadoRef: "emp_RRRRRRRRRRRRRRRRRRRRRR", ZonaHoraria: "Europe/Madrid"}
	b, err := RecursoBandejaNotificaciones(m)
	if err != nil || b.Referencia != domain.BandejaNotificacionesRef || b.Ambitos["paso_resolucion"] != "administracion" || b.Ambitos["persona_ref"] != m.ActorRef {
		t.Fatal(b, err)
	}
	p, err := RecursoConsultaNotificacionesPropias(m)
	if err != nil || p.Ambitos["empleado_ref"] != m.EmpleadoRef || p.Atributos["material_sha256"] == b.Atributos["material_sha256"] && p.Tipo == b.Tipo {
		t.Fatal(p, err)
	}
}

// El circuito aplicado (cronos_v1 000010): lo pendiente de asignación sólo
// llega a RRHH, solicitado y en J-A; A sólo a RRHH y sin pendiente.
func TestBandejaDePermisosAceptaSoloElCircuitoAplicado(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	orden := ordenResolucionPrueba(t)
	repo := &repoResolucionPrueba{}
	s, _ := NuevoServicioResolucionPermisos(repo, relojMarcajePrueba{time.Now().UTC()}, zona)
	pendiente := pendientePrueba()
	pendiente.PendienteAsignacion = true
	directo := pendientePrueba()
	directo.Circuito = domain.CircuitoAdministracion
	for _, caso := range []struct {
		paso domain.PasoPermiso
		fila ports.SolicitudPendiente
		ok   bool
	}{
		{domain.PasoAdministracion, pendiente, true},
		{domain.PasoAdministracion, directo, true},
		{domain.PasoResponsable, pendiente, false},
		{domain.PasoResponsable, directo, false},
		{domain.PasoAdministracion, func() ports.SolicitudPendiente { d := directo; d.PendienteAsignacion = true; return d }(), false},
	} {
		repo.bandeja = ports.BandejaPermisos{Paso: caso.paso, Pendientes: []ports.SolicitudPendiente{caso.fila}}
		_, err := s.ConsultarBandeja(context.Background(), orden, caso.paso)
		if (err == nil) != caso.ok {
			t.Fatalf("%s %+v: %v", caso.paso, caso.fila, err)
		}
	}
}

package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorResolucionPrueba struct{}

func (proveedorResolucionPrueba) ProveerMaterialBandejaPermisos(context.Context, domain.MaterialBandejaPermisos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorResolucionPrueba) ProveerMaterialResolucionPermiso(context.Context, domain.MaterialResolucionPermiso) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorResolucionPrueba) ProveerMaterialConsultaAvisosPropios(context.Context, domain.MaterialConsultaAvisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorResolucionPrueba) ProveerMaterialArchivoAvisoPropio(context.Context, domain.MaterialArchivoAvisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}

type repoResolucionPrueba struct {
	bandeja   ports.BandejaPermisos
	recibo    ports.ReciboResolucionPermiso
	materialB domain.MaterialBandejaPermisos
	materialR domain.MaterialResolucionPermiso
	llamadas  int
}

func (r *repoResolucionPrueba) ConsultarBandeja(_ context.Context, _ ports.OrdenResolucionPermisos, m domain.MaterialBandejaPermisos) (ports.BandejaPermisos, error) {
	r.materialB = m
	r.llamadas++
	return r.bandeja, nil
}

func (r *repoResolucionPrueba) ResolverPermiso(_ context.Context, _ ports.OrdenResolucionPermisos, m domain.MaterialResolucionPermiso) (ports.ReciboResolucionPermiso, error) {
	r.materialR = m
	r.llamadas++
	return r.recibo, nil
}

func ordenResolucionPrueba(t *testing.T) ports.OrdenResolucionPermisos {
	t.Helper()
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	o, err := ports.NuevaOrdenResolucionPermisos(actor, proveedorResolucionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func pendientePrueba() ports.SolicitudPendiente {
	return ports.SolicitudPendiente{SolicitudRef: "permiso:cronos:solicitud:perm-va-00001", EmpleadoRef: "emp_AAAAAAAAAAAAAAAAAAAAAA",
		PermisoRef: "permiso:cronos:vacaciones", Nombre: "Vacaciones", Circuito: domain.CircuitoResponsableAdministracion,
		Desde: "2026-10-05", Hasta: "2026-10-07", Cantidad: 3, Unidad: domain.LeaveUnitDay, Estado: domain.EstadoPermisoSolicitado,
		Version: 1, SolicitadaEnUTC: time.Now().UTC()}
}

func TestBandejaDerivaEmpleadoPropioYRechazaFuenteIncoherente(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	orden := ordenResolucionPrueba(t)
	repo := &repoResolucionPrueba{bandeja: ports.BandejaPermisos{Paso: domain.PasoResponsable, Pendientes: []ports.SolicitudPendiente{pendientePrueba()}}}
	s, _ := NuevoServicioResolucionPermisos(repo, relojMarcajePrueba{time.Now().UTC()}, zona)
	b, err := s.ConsultarBandeja(context.Background(), orden, domain.PasoResponsable)
	if err != nil || len(b.Pendientes) != 1 || repo.materialB.EmpleadoRef != "emp_0123456789abcdefghijkl" || repo.materialB.Paso != domain.PasoResponsable {
		t.Fatalf("bandeja o material distintos: %+v %+v %v", b, repo.materialB, err)
	}
	if _, err := s.ConsultarBandeja(context.Background(), orden, "jefatura"); !errors.Is(err, ports.ErrSolicitudCronosInvalida) || repo.llamadas != 1 {
		t.Fatal("paso desconocido llega al repositorio", err)
	}
	for nombre, alterar := range map[string]func(*ports.SolicitudPendiente){
		"lo propio":              func(p *ports.SolicitudPendiente) { p.EmpleadoRef = "emp_0123456789abcdefghijkl" },
		"circuito A en jefatura": func(p *ports.SolicitudPendiente) { p.Circuito = domain.CircuitoAdministracion },
		"ya concedida":           func(p *ports.SolicitudPendiente) { p.Estado = domain.EstadoPermisoConcedido },
		"sin versión":            func(p *ports.SolicitudPendiente) { p.Version = 0 },
	} {
		p := pendientePrueba()
		alterar(&p)
		repo.bandeja.Pendientes = []ports.SolicitudPendiente{p}
		if _, err := s.ConsultarBandeja(context.Background(), orden, domain.PasoResponsable); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
	repo.bandeja = ports.BandejaPermisos{Paso: domain.PasoAdministracion, Pendientes: []ports.SolicitudPendiente{pendientePrueba()}}
	if _, err := s.ConsultarBandeja(context.Background(), orden, domain.PasoResponsable); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta la bandeja de otro paso", err)
	}
}

func TestResolverNormalizaMotivoYCompruebaRecibo(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	orden := ordenResolucionPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repoResolucionPrueba{recibo: ports.ReciboResolucionPermiso{ResolucionRef: "permiso:cronos:resolucion:res-va-00001", SolicitudRef: "permiso:cronos:solicitud:perm-va-00001",
		ReciboRef: "recibo:cronos:x", Estado: domain.EstadoPermisoDenegado, Version: 2, InstanteUTC: ahora}}
	s, _ := NuevoServicioResolucionPermisos(repo, relojMarcajePrueba{ahora}, zona)
	p := ports.PeticionResolucionPermiso{ClaveOperacion: "res-va-00001", SolicitudRef: "permiso:cronos:solicitud:perm-va-00001",
		Paso: domain.PasoResponsable, Decision: domain.DecisionDenegar, Motivo: "  Coincide con el cierre  ", VersionEsperada: 1}
	r, err := s.ResolverPermiso(context.Background(), orden, p)
	if err != nil || r.Estado != domain.EstadoPermisoDenegado || repo.materialR.Motivo != "Coincide con el cierre" || repo.materialR.EmpleadoRef != "emp_0123456789abcdefghijkl" {
		t.Fatalf("resolución distinta: %+v %+v %v", r, repo.materialR, err)
	}
	for nombre, alterar := range map[string]func(*ports.PeticionResolucionPermiso){
		"denegar sin motivo":  func(p *ports.PeticionResolucionPermiso) { p.Motivo = "   " },
		"motivo con control":  func(p *ports.PeticionResolucionPermiso) { p.Motivo = "a\nb" },
		"decisión inventada":  func(p *ports.PeticionResolucionPermiso) { p.Decision = "conceder" },
		"versión cero":        func(p *ports.PeticionResolucionPermiso) { p.VersionEsperada = 0 },
		"solicitud mal hecha": func(p *ports.PeticionResolucionPermiso) { p.SolicitudRef = "permiso:cronos:solicitud:x" },
	} {
		q := p
		alterar(&q)
		antes := repo.llamadas
		if _, err := s.ResolverPermiso(context.Background(), orden, q); !errors.Is(err, ports.ErrSolicitudCronosInvalida) || repo.llamadas != antes {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
	// Un recibo con otro estado, versión o referencia no se da por bueno.
	repo.recibo.Estado = domain.EstadoPermisoPendienteAdministracion
	if _, err := s.ResolverPermiso(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un estado que no corresponde a la decisión", err)
	}
	repo.recibo.Estado, repo.recibo.Version = domain.EstadoPermisoDenegado, 3
	if _, err := s.ResolverPermiso(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta una versión que no es la siguiente", err)
	}
}

type repoAvisosPrueba struct {
	consulta ports.ConsultaAvisosPropios
	recibo   ports.ReciboArchivoAviso
}

func (r *repoAvisosPrueba) ConsultarAvisos(context.Context, ports.OrdenAvisosPropios, domain.MaterialConsultaAvisosPropios) (ports.ConsultaAvisosPropios, error) {
	return r.consulta, nil
}
func (r *repoAvisosPrueba) ArchivarAviso(context.Context, ports.OrdenAvisosPropios, domain.MaterialArchivoAvisoPropio) (ports.ReciboArchivoAviso, error) {
	return r.recibo, nil
}

func TestAvisosRechazanFuenteIncoherenteYReciboAjeno(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, _ := ports.NuevaOrdenAvisosPropios(actor, proveedorResolucionPrueba{})
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	aviso := ports.AvisoPropio{AvisoRef: "aviso:cronos:00000000-0000-4000-8000-000000000001", SolicitudRef: "permiso:cronos:solicitud:perm-va-00001",
		Estado: domain.EstadoPermisoDenegado, Motivo: "Cierre", ResueltoEnUTC: ahora, PermisoRef: "permiso:cronos:vacaciones", Nombre: "Vacaciones",
		Desde: "2026-10-05", Hasta: "2026-10-07", Cantidad: 3, Unidad: domain.LeaveUnitDay}
	repo := &repoAvisosPrueba{consulta: ports.ConsultaAvisosPropios{Avisos: []ports.AvisoPropio{aviso}}}
	s, _ := NuevoServicioAvisosPropios(repo, relojMarcajePrueba{ahora}, zona)
	if c, err := s.ConsultarAvisos(context.Background(), orden); err != nil || len(c.Avisos) != 1 {
		t.Fatal(c, err)
	}
	sinMotivo := aviso
	sinMotivo.Motivo = ""
	repo.consulta.Avisos = []ports.AvisoPropio{sinMotivo}
	if _, err := s.ConsultarAvisos(context.Background(), orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta una denegación sin motivo", err)
	}
	pendiente := aviso
	pendiente.Estado = domain.EstadoPermisoPendienteAdministracion
	repo.consulta.Avisos = []ports.AvisoPropio{pendiente}
	if _, err := s.ConsultarAvisos(context.Background(), orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un aviso de un paso intermedio", err)
	}
	repo.recibo = ports.ReciboArchivoAviso{ArchivoRef: "aviso:cronos:archivo:arch-00000001", AvisoRef: aviso.AvisoRef, ReciboRef: "recibo:cronos:y", InstanteUTC: ahora}
	if _, err := s.ArchivarAviso(context.Background(), orden, ports.PeticionArchivoAviso{ClaveOperacion: "arch-00000001", AvisoRef: aviso.AvisoRef}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ArchivarAviso(context.Background(), orden, ports.PeticionArchivoAviso{ClaveOperacion: "arch-00000002", AvisoRef: aviso.AvisoRef}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta el recibo de otra clave", err)
	}
	if _, err := s.ArchivarAviso(context.Background(), orden, ports.PeticionArchivoAviso{ClaveOperacion: "arch-00000001", AvisoRef: "aviso:x"}); !errors.Is(err, ports.ErrSolicitudCronosInvalida) {
		t.Fatal("acepta un aviso mal formado", err)
	}
}

// La huella del recurso de quien resuelve debe coincidir con la que calcula
// huella_contexto_resolutor_v1 de cronos_v1 000009 (ámbitos en orden).
func TestRecursoResolutorCoincideConLaHuellaSQL(t *testing.T) {
	m := domain.MaterialResolucionPermiso{ActorRef: "per_JJJJJJJJJJJJJJJJJJJJJJ", PerfilRef: "prf_QQQQQQQQQQQQQQQQQQQQQQ", EmpleadoRef: "emp_JJJJJJJJJJJJJJJJJJJJJJ",
		ClaveOperacion: "res-va-00001", SolicitudRef: "permiso:cronos:solicitud:perm-va-00001", Paso: domain.PasoAdministracion,
		Decision: domain.DecisionAprobar, VersionEsperada: 2, ZonaHoraria: domain.ZonaSaldoPeninsula}
	canonico, _ := m.Canonico()
	r, err := RecursoResolucionPermiso(m)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(canonico)
	sql := `{"ambitos":{"paso_resolucion":"administracion","persona_ref":"per_JJJJJJJJJJJJJJJJJJJJJJ"},"atributos":{"material_sha256":"` + hex.EncodeToString(h[:]) + `"}}`
	esperada := sha256.Sum256([]byte(sql))
	obtenida, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || obtenida != hex.EncodeToString(esperada[:]) || r.Referencia != "permiso:cronos:resolucion:res-va-00001" {
		t.Fatalf("huella distinta de la SQL: %s %v", obtenida, err)
	}
}

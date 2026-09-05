package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func solicitudComunicacionLlamamientoDesarrolloPrueba() ports.SolicitudRegistrarComunicacionLlamamiento {
	return ports.SolicitudRegistrarComunicacionLlamamiento{
		ClaveIdempotencia: "018f47a6-5d2b-4c10-8a11-1234567890ab",
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:sintetico:001", LlamamientoRef: "llamamiento:sintetico:001",
		VersionEsperada: 1, PruebaEntregaRef: "recibo:seleccion:sintetica:001",
	}
}

type autorizacionComunicacionDesarrolloPrueba struct {
	llamadas int
	accion   string
	recurso  dominiovec.RecursoAutorizable
}

func (a *autorizacionComunicacionDesarrolloPrueba) AutorizarOperacion(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.accion, a.recurso = accion, recurso
	return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func escenarioComunicacionLlamamientoDesarrolloPrueba(t *testing.T) (context.Context, *proveedorComunicacionLlamamientoDesarrollo, *autorizacionComunicacionDesarrolloPrueba) {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	solicitud := solicitudComunicacionLlamamientoDesarrolloPrueba()
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaRegistroComunicacionLlamamiento)
	// Doble reducido para comprobar la ligadura de composición. No se usa como
	// agregado válido ni como evidencia PG; el ejecutor real exige Validar().
	expediente := ports.ExpedienteParaSeleccion{
		Fiscalizado: domain.Expediente{
			Referencia: solicitud.ExpedienteRef, OrganizacionRef: solicitud.OrganizacionRef,
			Version: 6, FaseActual: domain.FaseFiscalizacion, EstadoActual: domain.EstadoEnCurso,
			Fiscalizacion: &domain.FiscalizacionRegistrada{Resultado: domain.FiscalizacionFavorable},
		}, VersionActual: 6,
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveSolicitudComunicacionLlamamientoDesarrollo{}, solicitud)
	a := &autorizacionComunicacionDesarrolloPrueba{}
	p := &proveedorComunicacionLlamamientoDesarrollo{
		soporte: soporte, autorizador: a, reloj: relojFijoAltaContratacionTemporalDesarrollo{
			ahora: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
		},
	}
	return ctx, p, a
}

func TestComunicacionLlamamientoDesarrolloCatalogosNoInventanEntrega(t *testing.T) {
	ctx, p, _ := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	material, err := p.PrepararRegistroComunicacion(ctx, solicitudComunicacionLlamamientoDesarrolloPrueba())
	if err != nil || material.Validar() != nil {
		t.Fatal(err)
	}
	canal := sha256.Sum256([]byte(canalComunicacionLlamamientoDesarrollo))
	politica := sha256.Sum256([]byte(politicaComunicacionLlamamientoDesarrollo))
	if material.Canal.HuellaSHA256 != hex.EncodeToString(canal[:]) ||
		material.Politica.HuellaSHA256 != hex.EncodeToString(politica[:]) ||
		material.Canal.Version != 1 || material.Politica.Version != 1 ||
		!strings.Contains(canalComunicacionLlamamientoDesarrollo, `"envio_externo":false`) ||
		!strings.Contains(politicaComunicacionLlamamientoDesarrollo, `"abre_plazo":false`) {
		t.Fatal("catálogo divergente o entrega inventada")
	}
}

func TestComunicacionLlamamientoDesarrolloIdentidadNoSustituyeV3NiReplay(t *testing.T) {
	ctx, p, a := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	m, err := p.PrepararRegistroComunicacion(ctx, solicitudComunicacionLlamamientoDesarrolloPrueba())
	if err != nil {
		t.Fatal(err)
	}
	for intento := 1; intento <= 2; intento++ {
		material, err := p.AutorizarRegistroComunicacion(ctx, m)
		if !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) ||
			material.ValidarEstructura() == nil || a.llamadas != intento ||
			a.accion != postgresct.AccionRegistroComunicacionLlamamiento {
			t.Fatal("identidad o replay sustituyeron autorización V3")
		}
	}
	esperado, _ := postgresct.RecursoRegistroComunicacionLlamamiento(m)
	h1, _ := esperado.HuellaContextoAutorizacionSHA256()
	h2, _ := a.recurso.HuellaContextoAutorizacionSHA256()
	if h1 != h2 || a.recurso.Referencia != m.Solicitud.ExpedienteRef {
		t.Fatal("recurso enviado al autorizador divergente")
	}
}

func TestComunicacionLlamamientoDesarrolloRechazaMaterialAjeno(t *testing.T) {
	ctx, p, a := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	s := solicitudComunicacionLlamamientoDesarrolloPrueba()
	m, err := p.PrepararRegistroComunicacion(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	m.Politica.Version++
	if _, err := p.AutorizarRegistroComunicacion(ctx, m); err == nil || a.llamadas != 0 {
		t.Fatal("política declarada por cliente alcanzó autorización")
	}
	for nombre, mutar := range map[string]func(*ports.SolicitudRegistrarComunicacionLlamamiento){
		"organizacion":       func(s *ports.SolicitudRegistrarComunicacionLlamamiento) { s.OrganizacionRef = "org:ajena" },
		"expediente":         func(s *ports.SolicitudRegistrarComunicacionLlamamiento) { s.ExpedienteRef += "b" },
		"llamamiento":        func(s *ports.SolicitudRegistrarComunicacionLlamamiento) { s.LlamamientoRef += "b" },
		"recibo":             func(s *ports.SolicitudRegistrarComunicacionLlamamiento) { s.PruebaEntregaRef += "b" },
		"version_expediente": func(s *ports.SolicitudRegistrarComunicacionLlamamiento) { s.VersionEsperada = 6 },
		"clave": func(s *ports.SolicitudRegistrarComunicacionLlamamiento) {
			s.ClaveIdempotencia = "118f47a6-5d2b-4c10-8a11-1234567890ab"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := s
			mutar(&alterada)
			if _, err := p.PrepararRegistroComunicacion(ctx, alterada); err == nil {
				t.Fatal("petición fuera de la ligadura aceptada")
			}
		})
	}
}

type lectorComunicacionLlamamientoDesarrolloPrueba struct{ llamadas int }

func (l *lectorComunicacionLlamamientoDesarrolloPrueba) LeerExpedienteParaSeleccion(context.Context, string, string, uint64) (ports.ExpedienteParaSeleccion, error) {
	l.llamadas++
	return ports.ExpedienteParaSeleccion{}, errors.New("fallo interno sintético")
}

type ejecutorComunicacionDesarrolloPrueba struct {
	llamadas int
	recibo   *ports.ComunicacionProbatoria
}

func (e *ejecutorComunicacionDesarrolloPrueba) Registrar(context.Context, ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	e.llamadas++
	if e.recibo != nil {
		return *e.recibo, nil
	}
	return ports.ComunicacionProbatoria{}, errors.New("no debe ejecutarse")
}
func (e *ejecutorComunicacionDesarrolloPrueba) Resolver(context.Context, ports.SolicitudResolverLlamamiento) (ports.ResultadoResolucionLlamamiento, error) {
	e.llamadas++
	return ports.ResultadoResolucionLlamamiento{}, errors.New("no debe ejecutarse")
}

func TestComunicacionLlamamientoDesarrolloEjecutorFallaCerrado(t *testing.T) {
	ctx, p, _ := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	lector := &lectorComunicacionLlamamientoDesarrolloPrueba{}
	servicio := &ejecutorComunicacionDesarrolloPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, lector: lector, servicio: servicio}
	s := solicitudComunicacionLlamamientoDesarrolloPrueba()
	if _, err := e.Registrar(context.Background(), s); !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) || lector.llamadas != 0 {
		t.Fatal("lectura sin capacidad confiable")
	}
	r, err := e.Registrar(ctx, s)
	if !errors.Is(err, application.ErrComunicacionLlamamientoNoDisponible) ||
		r != (ports.ComunicacionProbatoria{}) || servicio.llamadas != 0 || lector.llamadas != 1 ||
		strings.Contains(err.Error(), "interno") {
		t.Fatal("fallo de lectura convertido en efecto o expuesto")
	}
	ctxCancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := e.Registrar(ctxCancelado, s); !errors.Is(err, context.Canceled) || lector.llamadas != 1 {
		t.Fatal("lectura tras cancelación")
	}
	if ejecutor, err := nuevoEjecutorComunicacionLlamamientoDesarrollo(nil, nil, nil, nil, nil, ""); err == nil || ejecutor != nil {
		t.Fatal("constructor incompleto admitido")
	}
}

func TestComunicacionLlamamientoDesarrolloPreparacionNoCruzaRutaOVigencia(t *testing.T) {
	ctx, p, _ := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	s := solicitudComunicacionLlamamientoDesarrolloPrueba()
	if _, err := p.PrepararRegistroComunicacion(context.Background(), s); err == nil {
		t.Fatal("preparación sin autoridad")
	}
	capacidad, _ := p.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaSeleccionLlamamiento
	ctxAjeno := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if _, err := p.PrepararRegistroComunicacion(ctxAjeno, s); err == nil {
		t.Fatal("ruta de selección usada para comunicación")
	}
	p.reloj = relojFijoAltaContratacionTemporalDesarrollo{ahora: time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := p.PrepararRegistroComunicacion(ctx, s); err == nil {
		t.Fatal("fuente sintética vencida admitida")
	}
}

func TestComunicacionLlamamientoDesarrolloResolverNoUsaPermisoDeRegistro(t *testing.T) {
	ctx, p, a := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	lector := &lectorComunicacionLlamamientoDesarrolloPrueba{}
	servicio := &ejecutorComunicacionDesarrolloPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, lector: lector, servicio: servicio}
	s := ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia: "018f47a6-5d2b-4c10-8a11-1234567890ab",
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:sintetico:001",
		LlamamientoRef: "llamamiento:sintetico:001", ComunicacionRef: "comunicacion:sintetica:001",
		VersionEsperada: 2, Respuesta: ports.RespuestaLlamamientoExpirada,
	}
	r, err := e.Resolver(ctx, s)
	if !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) ||
		r != (ports.ResultadoResolucionLlamamiento{}) || servicio.llamadas != 0 ||
		lector.llamadas != 0 || a.llamadas != 0 {
		t.Fatal("registro local convertido en resolución")
	}
}

func TestResolucionAceptacionDesarrolloPendienteSinEfectos(t *testing.T) {
	ctx, p, autorizador := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	capacidad, _ := p.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaResolucionComunicacionLlamamiento
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	lector := &lectorComunicacionLlamamientoDesarrolloPrueba{}
	servicio := &ejecutorComunicacionDesarrolloPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, lector: lector, servicio: servicio}
	s := ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia: "018f47a6-5d2b-4c10-8a11-1234567890ab",
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:sintetico:001", LlamamientoRef: "llamamiento:sintetico:001",
		ComunicacionRef: "comunicacion:sintetica:001", VersionEsperada: 2,
		Respuesta: ports.RespuestaLlamamientoAceptada, PruebaRespuestaRef: "justificante:sintetico:001",
	}
	r, err := e.Resolver(ctx, s)
	if !errors.Is(err, application.ErrValidacionRespuestaLlamamientoPendiente) || r != (ports.ResultadoResolucionLlamamiento{}) {
		t.Fatal("no se conservó la validación pendiente sin recibo")
	}
	if _, err = e.Resolver(context.Background(), s); !errors.Is(err, application.ErrComunicacionLlamamientoDenegada) {
		t.Fatal("referencias públicas sustituyeron identidad")
	}
	s.VersionEsperada = 3
	if _, err = e.Resolver(ctx, s); !errors.Is(err, application.ErrVersionComunicacionLlamamientoEnConflicto) {
		t.Fatal("versión de resolución no comprobada")
	}
	if lector.llamadas != 0 || servicio.llamadas != 0 || autorizador.llamadas != 0 {
		t.Fatal("validación pendiente abrió lectura, permiso o efecto")
	}
}

func reciboAvisoComunicacionDesarrolloPrueba(t *testing.T, ctx context.Context, p *proveedorComunicacionLlamamientoDesarrollo) ports.ComunicacionProbatoria {
	t.Helper()
	s := solicitudComunicacionLlamamientoDesarrolloPrueba()
	m, err := p.PrepararRegistroComunicacion(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ComunicacionProbatoria{
		Solicitud: s, ComunicacionRef: "comunicacion:sintetica:001", Canal: m.Canal, Politica: m.Politica,
		ReciboRef: "recibo:comunicacion:sintetica:001", AuditoriaRef: "auditoria:sintetica:001",
		VersionResultante: 2, Estado: ports.ResultadoComunicacionLlamamientoLocal,
		RegistradaEn:      time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
		IntencionEnvioRef: "outbox:comunicacion:sintetica:001",
	}
}

func TestComunicacionLlamamientoDesarrolloAvisoFicheroYReplay(t *testing.T) {
	ctx, p, _ := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	r := reciboAvisoComunicacionDesarrolloPrueba(t, ctx, p)
	s := r.Solicitud
	base := t.TempDir()
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	directorio := filepath.Join(base, "comunicaciones")
	servicio := &ejecutorComunicacionDesarrolloPrueba{recibo: &r}
	e := &ejecutorComunicacionLlamamientoDesarrollo{servicio: servicio, directorioComunicaciones: directorio}
	confirmado, err := e.registrarConAviso(ctx, s)
	if err != nil || confirmado != r || servicio.llamadas != 1 {
		t.Fatal("registro con aviso falló", err)
	}
	dirInfo, err := os.Stat(directorio)
	if err != nil || dirInfo.Mode().Perm() != 0700 {
		t.Fatal("directorio no privado", err)
	}
	entradas, err := os.ReadDir(directorio)
	if err != nil || len(entradas) != 1 {
		t.Fatal("cantidad de ficheros inesperada", err)
	}
	ruta := filepath.Join(directorio, entradas[0].Name())
	inicial, err := os.Lstat(ruta)
	if err != nil || inicial.Mode().Perm() != 0600 || !inicial.Mode().IsRegular() {
		t.Fatal("fichero no privado", err)
	}
	contenido, err := os.ReadFile(ruta)
	var aviso avisoComunicacionDesarrollo
	if err != nil || json.Unmarshal(contenido, &aviso) != nil ||
		aviso.Texto != "Aviso de desarrollo, no enviado, no abre plazo" ||
		aviso.IntencionEnvioRef != r.IntencionEnvioRef || aviso.ReciboRef != r.ReciboRef ||
		aviso.RegistradaEn != r.RegistradaEn || aviso.ReciboSeleccionRef != s.PruebaEntregaRef ||
		aviso.OrganizacionRef != s.OrganizacionRef || aviso.ExpedienteRef != s.ExpedienteRef ||
		aviso.LlamamientoRef != s.LlamamientoRef || aviso.ClaveIdempotencia != s.ClaveIdempotencia ||
		bytes.Contains(contenido, []byte("respuesta_hasta")) || bytes.Contains(contenido, []byte("destinatario")) {
		t.Fatal("aviso inventado o desligado del recibo", err)
	}
	r.Estado = ports.ResultadoComunicacionLlamamientoReplayLocal
	repetido, err := e.registrarConAviso(ctx, s)
	if err != nil || repetido != r || servicio.llamadas != 2 {
		t.Fatal("replay no consultó de nuevo al servicio", err)
	}
	actual, _ := os.Lstat(ruta)
	releido, _ := os.ReadFile(ruta)
	entradas, _ = os.ReadDir(directorio)
	if !os.SameFile(inicial, actual) || !bytes.Equal(contenido, releido) || len(entradas) != 1 {
		t.Fatal("replay creó o sobrescribió un fichero")
	}
}

func TestComunicacionLlamamientoDesarrolloAvisoFalloYReintentoExplicito(t *testing.T) {
	ctx, p, _ := escenarioComunicacionLlamamientoDesarrolloPrueba(t)
	r := reciboAvisoComunicacionDesarrolloPrueba(t, ctx, p)
	s := r.Solicitud
	base := t.TempDir()
	padre := filepath.Join(base, "material")
	directorio := filepath.Join(padre, "comunicaciones")
	servicio := &ejecutorComunicacionDesarrolloPrueba{recibo: &r}
	e := &ejecutorComunicacionLlamamientoDesarrollo{servicio: servicio, directorioComunicaciones: directorio}
	// Simula commit PG seguido de fallo de disco/configuración. No es una
	// prueba PostgreSQL: comprueba que composición no publica éxito ni reintenta.
	resultado, err := e.registrarConAviso(ctx, s)
	if !errors.Is(err, application.ErrComunicacionLlamamientoNoDisponible) ||
		resultado != (ports.ComunicacionProbatoria{}) || servicio.llamadas != 1 {
		t.Fatal("fallo de fichero convertido en éxito o reintentado", err)
	}
	if err := os.Mkdir(padre, 0700); err != nil {
		t.Fatal(err)
	}
	r.Estado = ports.ResultadoComunicacionLlamamientoReplayLocal
	if resultado, err = e.registrarConAviso(ctx, s); err != nil || resultado != r || servicio.llamadas != 2 {
		t.Fatal("reintento explícito no recuperó recibo y aviso", err)
	}
	entradas, _ := os.ReadDir(directorio)
	if len(entradas) != 1 {
		t.Fatal("salida ausente o duplicada")
	}
	ruta := filepath.Join(directorio, entradas[0].Name())
	// Fichero de prueba ajeno al contenido esperado: nunca debe sobrescribirse.
	ajeno := []byte("contenido ajeno conservado")
	if err := os.WriteFile(ruta, ajeno, 0600); err != nil {
		t.Fatal(err)
	}
	resultado, err = e.registrarConAviso(ctx, s)
	restante, _ := os.ReadFile(ruta)
	entradas, _ = os.ReadDir(directorio)
	if !errors.Is(err, application.ErrComunicacionLlamamientoNoDisponible) ||
		resultado != (ports.ComunicacionProbatoria{}) || servicio.llamadas != 3 ||
		!bytes.Equal(restante, ajeno) || len(entradas) != 1 {
		t.Fatal("fichero divergente sobrescrito o falso éxito", err)
	}
}

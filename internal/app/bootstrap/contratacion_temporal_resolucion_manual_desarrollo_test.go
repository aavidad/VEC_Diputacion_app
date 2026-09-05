package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func escenarioRevisionManualPrueba(t *testing.T) (context.Context, *proveedorComunicacionLlamamientoDesarrollo, aceptacionRevisadaDesarrollo) {
	t.Helper()
	p, ctx, s, seleccion, _, _, _ := escenarioAceptacionPuentePrueba(t)
	preparacion, _ := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	_, doc, err := p.fuente(preparacion)
	if err != nil {
		t.Fatal(err)
	}
	ahora := seleccion.ConfirmadaEn
	seleccion.CorrelacionRef = "correlacion:revision-prueba"
	seleccion.AccionEvento = referenciaCatalogoPuenteLlamamientoDesarrollo("bolsa.llamamiento.evento.registrar")
	seleccion.RetencionSeleccion = referenciaCatalogoPuenteLlamamientoDesarrollo("retencion:seleccion:sintetica:desarrollo")
	seleccion.Procedencia = p.procedencia(ports.DatosContextoPeticionIntegracionBolsa{OperacionRef: seleccion.OperacionRef, ValidaHasta: ahora.Add(time.Minute)}, doc, "llamamiento", ahora)
	// Fixture nominal del antecedente ya consultado, no emisor ni verificador
	// criptográfico: los recorridos PostgreSQL usan el recibo firmado original.
	seleccion.Procedencia.Evidencia.ClaveVerificacionRef = "vec.contratacion-temporal.integracion-bolsa-respuesta/v1"
	seleccion.Procedencia.Evidencia.SelloHMAC = "hmac-sha256:" + seleccion.Procedencia.Evidencia.ClaveVerificacionRef + ":" + strings.Repeat("a", 64)
	s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH, s.CriterioValidacionRef = true, true, criterioRevisionManualDesarrollo
	j := ports.JustificanteRespuestaRecibida{Seleccion: seleccion, Respuesta: ports.RespuestaRecibidaRegistrada{
		Solicitud: ports.SolicitudRegistrarRespuestaRecibida{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111",
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef,
			ComunicacionRef: s.ComunicacionRef, VersionComunicacionEsperada: 2, Respuesta: s.Respuesta,
			CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64), RecibidaEn: ahora},
		JustificanteRef: s.PruebaRespuestaRef, ReciboRef: "recibo:respuesta-prueba", AuditoriaRef: "auditoria:respuesta-prueba",
		RegistradaEn: ahora, Estado: ports.EstadoRespuestaRecibidaRegistrada,
	}}
	if err := j.ValidarPara(s); err != nil {
		t.Fatal("fixture justificante", err)
	}
	l := aceptacionRevisadaDesarrollo{solicitud: s, justificante: j}
	l.local = ports.ResultadoResolucionLlamamiento{Solicitud: s, Politica: politicaManualDesarrollo(),
		EvaluacionPlazoRef: "evaluacion:manual-prueba", EstadoPlazo: ports.PlazoLlamamientoVigente,
		ResolucionRef: "resolucion:manual-prueba", ReciboLocalRef: "recibo:manual-prueba", AuditoriaRef: "auditoria:manual-prueba",
		VersionResultante: 3, ResueltaEn: ahora, Estado: ports.ResultadoComunicacionLlamamientoConfirmado}
	ctx = context.WithValue(ctx, claveResolucionManualDesarrollo{}, l)
	ctx = context.WithValue(ctx, claveConsultaJustificanteRespuestaDesarrollo{}, s)
	proveedor := &proveedorComunicacionLlamamientoDesarrollo{soporte: p.alta.soporte, reloj: p.reloj,
		autorizadorResolucion: &autorizacionComunicacionDesarrolloPrueba{}}
	return ctx, proveedor, l
}

func TestResolucionManualDesarrolloExigeRevisionExplicitaYPemisoPropio(t *testing.T) {
	ctx, p, l := escenarioRevisionManualPrueba(t)
	m, err := p.PrepararResolucionManual(ctx, l.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	r, err := postgresct.RecursoResolucionManualLlamamiento(m)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal",
		Accion: postgresct.AccionResolucionManualLlamamiento, ReferenciaMotivo: motivoResolucionManualDesarrollo(false), Recurso: r}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, d) {
		t.Fatal("permiso ligado rechazado")
	}
	d.ReferenciaMotivo = motivoConsultaJustificanteRespuestaDesarrollo()
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, d) {
		t.Fatal("lectura concedió escritura")
	}
	for _, alterar := range []func(*ports.SolicitudResolverLlamamiento){
		func(s *ports.SolicitudResolverLlamamiento) { s.RevisionPlazoRRHH = false },
		func(s *ports.SolicitudResolverLlamamiento) { s.CriterioValidacionRef = "politica:ajena" },
		func(s *ports.SolicitudResolverLlamamiento) { s.PruebaRespuestaRef = "justificante:ajeno" },
	} {
		s := l.solicitud
		alterar(&s)
		if _, err := p.PrepararResolucionManual(ctx, s); !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) {
			t.Fatal("revisión ajena admitida", err)
		}
	}
	if _, err := p.PrepararResolucionManual(context.Background(), l.solicitud); !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) {
		t.Fatal("identidad omitida")
	}
	if _, err := p.AutorizarResolucionManual(ctx, m); !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) {
		t.Fatal("material V3 vacío aceptado")
	}
}

type servicioRevisionManualPrueba struct {
	local    ports.ResultadoResolucionLlamamiento
	llamadas int
}

func (s *servicioRevisionManualPrueba) Registrar(context.Context, ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	return ports.ComunicacionProbatoria{}, errors.New("no usado")
}
func (s *servicioRevisionManualPrueba) Resolver(context.Context, ports.SolicitudResolverLlamamiento) (ports.ResultadoResolucionLlamamiento, error) {
	s.llamadas++
	r := s.local
	if s.llamadas > 1 {
		r.Estado = ports.ResultadoComunicacionLlamamientoReplay
	}
	return r, nil
}

type aceptadorRevisionManualPrueba struct {
	falla    bool
	llamadas int
}

func (a *aceptadorRevisionManualPrueba) AceptarRespuestaRRHH(ctx context.Context, s ports.SolicitudResolverLlamamiento, seleccion ports.ReciboSolicitudLlamamientoBolsa, r puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error) {
	if s.Respuesta != ports.RespuestaLlamamientoAceptada {
		return puertosbolsa.ReciboLlamamientoDesarrollo{}, errors.New("no es aceptación")
	}
	return a.resolver(ctx, s, r)
}
func (a *aceptadorRevisionManualPrueba) RenunciarRespuestaRRHH(ctx context.Context, s ports.SolicitudResolverLlamamiento, seleccion ports.ReciboSolicitudLlamamientoBolsa, r puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error) {
	if s.Respuesta != ports.RespuestaLlamamientoRenunciada {
		return puertosbolsa.ReciboLlamamientoDesarrollo{}, errors.New("no es renuncia")
	}
	return a.resolver(ctx, s, r)
}
func (a *aceptadorRevisionManualPrueba) resolver(ctx context.Context, s ports.SolicitudResolverLlamamiento, r puertosbolsa.ResolucionLlamamientoDesarrollo) (puertosbolsa.ReciboLlamamientoDesarrollo, error) {
	a.llamadas++
	l, ok := ctx.Value(claveAceptacionRevisadaDesarrollo{}).(aceptacionRevisadaDesarrollo)
	if !ok || l.local.ValidarPara(s) != nil || a.falla {
		return puertosbolsa.ReciboLlamamientoDesarrollo{}, errors.New("bolsa no confirmada")
	}
	r.ResueltaEn = l.local.ResueltaEn
	tipo := "aceptacion_rrhh"
	if s.Respuesta == ports.RespuestaLlamamientoRenunciada {
		tipo = "renuncia_rrhh"
	}
	return puertosbolsa.ReciboLlamamientoDesarrollo{Registro: puertosbolsa.RegistroLlamamientoDesarrollo{
		Tipo: tipo, OperacionRef: operacionAceptacionManualDesarrollo(l), Resolucion: &r},
		ReciboRef: "recibo:bolsa-prueba", ConfirmadaEn: r.ResueltaEn}, nil
}

func TestResolucionManualDesarrolloSoloEntregaReciboTrasAmbosCommits(t *testing.T) {
	ctx, p, l := escenarioRevisionManualPrueba(t)
	servicio := &servicioRevisionManualPrueba{local: l.local}
	aceptador := &aceptadorRevisionManualPrueba{falla: true}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, servicio: servicio, aceptador: aceptador}
	r, err := e.resolverConRevisionManual(ctx, l.solicitud, l.justificante)
	if !errors.Is(err, application.ErrComunicacionLlamamientoNoDisponible) || r != (ports.ResultadoResolucionLlamamiento{}) || servicio.llamadas != 1 || aceptador.llamadas != 1 {
		t.Fatal("éxito parcial o se omitió Bolsa", err)
	}
	aceptador.falla = false
	r, err = e.resolverConRevisionManual(ctx, l.solicitud, l.justificante)
	if err != nil || !r.EsReplayConfirmado() || r.ReciboLocalRef != l.local.ReciboLocalRef || r.ResueltaEn != l.local.ResueltaEn || servicio.llamadas != 2 {
		t.Fatal("reintento perdió recibo original", err)
	}
}

func TestResolucionManualDesarrolloRenunciaConservaIntencionSinLlamarSiguiente(t *testing.T) {
	ctx, p, l := escenarioRevisionManualPrueba(t)
	l.solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
	l.justificante.Respuesta.Solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
	l.local.Solicitud = l.solicitud
	l.local.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
		Solicitud: l.solicitud, ResolucionRef: l.local.ResolucionRef, LlamamientoRef: l.solicitud.LlamamientoRef,
		ClaveIdempotencia: l.solicitud.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
		IntencionRef: "outbox:renuncia-prueba", ComandoOpacoRef: "comando:renuncia-prueba",
		Estado: ports.OutboxSiguienteCandidatoPendiente, ActualizadaEn: l.local.ResueltaEn,
	}
	ctx = context.WithValue(ctx, claveResolucionManualDesarrollo{}, l)
	ctx = context.WithValue(ctx, claveConsultaJustificanteRespuestaDesarrollo{}, l.solicitud)
	m, err := p.PrepararResolucionManual(ctx, l.solicitud)
	if err != nil || m.Solicitud != l.solicitud {
		t.Fatal("revisión de renuncia rechazada", err)
	}
	servicio := &servicioRevisionManualPrueba{local: l.local}
	aceptador := &aceptadorRevisionManualPrueba{}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.soporte, servicio: servicio, aceptador: aceptador}
	for i := 0; i < 2; i++ {
		r, err := e.resolverConRevisionManual(ctx, l.solicitud, l.justificante)
		if err != nil || r.IntencionSiguiente != l.local.IntencionSiguiente || r.ReciboLocalRef != l.local.ReciboLocalRef || r.EsReplayConfirmado() != (i > 0) {
			t.Fatal("renuncia perdió intención o recibo", err)
		}
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal",
		Accion: puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, ReferenciaMotivo: motivoRenunciaBolsaDesarrollo(),
		Recurso: dominiovec.RecursoAutorizable{Referencia: operacionAceptacionManualDesarrollo(l),
			ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa", Ambitos: map[string]string{
				"categoria_ref": "categoria:desarrollo:c2", "unidad_ref": unidadCoberturaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"necesidad_ref": l.justificante.Seleccion.Necesidad.Referencia, "contenido_sha256": strings.Repeat("a", 64)}}}
	ctx = context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, l)
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, d) {
		t.Fatal("permiso propio de renuncia rechazado")
	}
	d.Accion, d.ReferenciaMotivo = puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, motivoResolucionManualDesarrollo(true)
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionComunicacionLlamamiento, d) {
		t.Fatal("aceptación autorizó renuncia")
	}
}

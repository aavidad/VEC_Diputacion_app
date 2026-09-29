package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadAsignacionesContratacionTemporalDesarrolloPrueba struct {
	preparadas int
	publicadas int
	// asignaciones: la asignación publicada de cada perfil fijo, que la
	// prueba fija como si ya estuviera en PostgreSQL (solo lectura).
	asignaciones map[string]instantaneaPublicadaDesarrollo
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) leerAsignacionPublicada(
	_ context.Context, perfilRef string,
) (instantaneaPublicadaDesarrollo, bool, error) {
	publicada, existe := a.asignaciones[perfilRef]
	return publicada, existe, nil
}

type registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba struct {
	concesiones   int
	denegaciones  int
	huella        string
	errConcesion  error
	errDenegacion error
}

// El doble se instala solo en fixtures: la ruta productiva exige el proveedor
// registrado, mientras estas pruebas focales ejercitan el PDP con su semilla.
type proveedorSesionOperativaCTPrueba struct {
	contexto ports.ContextoAutorizacionAltaV3
}

func (p proveedorSesionOperativaCTPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	resultado, err := p.contexto.Resultado.Clonar()
	return contextoSeguridadComunDesarrollo{Vinculo: p.contexto.Vinculo, Resultado: resultado}, err
}

func (r *registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if r.errConcesion != nil {
		return time.Time{}, r.errConcesion
	}
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	huella, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.Decision)
	if err != nil {
		return time.Time{}, err
	}
	emitidaEn, _, err := datos.Decision.VentanaValidez()
	if err != nil {
		return time.Time{}, err
	}
	r.concesiones++
	r.huella = huella
	return emitidaEn, nil
}

func (r *registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba) RegistrarDenegacionAutorizacionLigadaV3(
	_ context.Context,
	orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if r.errDenegacion != nil {
		return r.errDenegacion
	}
	if _, err := orden.Datos(); err != nil {
		return err
	}
	r.denegaciones++
	return nil
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PrepararInstantanea(
	_ context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	a.preparadas++
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(instantanea), nil
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PublicarInstantanea(
	context.Context,
	dominiovec.InstantaneaAutorizacion,
) error {
	a.publicadas++
	return nil
}

func TestAutorizacionCoberturaDesarrolloSeparaRutasYAmbitos(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alta, err := soporte.ObtenerInstantaneaAutorizacion(
		contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaAltaSolicitudes),
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	propuestaCtx := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaPropuestaCobertura,
	)
	coberturaVEC, err := soporte.ObtenerInstantaneaAutorizacion(
		propuestaCtx,
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(alta.AsignacionPerfil.Ambitos) != 3 ||
		len(coberturaVEC.AsignacionPerfil.Ambitos) != 2 ||
		len(coberturaVEC.VersionRol.Concesiones) != 4 {
		t.Fatalf(
			"instantaneas mezcladas: alta=%+v cobertura=%+v",
			alta.AsignacionPerfil.Ambitos,
			coberturaVEC.AsignacionPerfil.Ambitos,
		)
	}
	tiposPorAccion := make(map[string]string, len(coberturaVEC.VersionRol.Concesiones))
	for _, concesion := range coberturaVEC.VersionRol.Concesiones {
		tiposPorAccion[concesion.Accion] = concesion.TipoRecurso
	}
	for accion, tipoEsperado := range map[string]string{
		accionPropuestaCoberturaDesarrollo:                ports.TipoRecursoExpediente,
		string(domain.AccionDecidirCoberturaGobernada):    "decision_cobertura_gobernada",
		string(domain.AccionRectificarCoberturaGobernada): "decision_cobertura_gobernada",
		string(ports.AccionConsultarResultadoCobertura):   ports.TipoRecursoExpediente,
	} {
		if tiposPorAccion[accion] != tipoEsperado {
			t.Fatalf("tipo de recurso para %s=%q, esperado %q", accion, tiposPorAccion[accion], tipoEsperado)
		}
	}
	claves := map[string]string{}
	for _, ambito := range coberturaVEC.AsignacionPerfil.Ambitos {
		claves[ambito.Clave] = ambito.Valores[0]
	}
	if len(claves) != 2 ||
		claves["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		claves["unidad_ejecutora_ref"] != unidadCoberturaContratacionTemporalDesarrollo {
		t.Fatalf("ambitos de cobertura no exactos: %+v", claves)
	}
	if _, err := soporte.ResolverContextoCanalCobertura(propuestaCtx); err != nil {
		t.Fatalf("contexto de propuesta denegado: %v", err)
	}
	if _, err := soporte.ResolverContextoCanalCobertura(
		contextoRutaCoberturaDesarrolloPrueba(
			soporte,
			principal,
			httpinterno.RutaResultadoCobertura,
		),
	); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("resultado obtuvo autoridad de efecto: %v", err)
	}
}

func TestPreparacionDecisionCoberturaPublicaLaAsignacionQueReferencia(t *testing.T) {
	soporte, consultas, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	delegado, ok := consultas.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	if !ok {
		t.Fatal("el autorizador de cobertura no expone la preparacion compuesta")
	}
	autorizador := &autorizadorAnalisisContratacionTemporalDesarrollo{
		delegado: delegado,
		soporte:  soporte,
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaDecisionCobertura,
	)
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: soporte.contexto.Vinculo,
			ReferenciaMotivo:          soporte.motivoDecisionCobertura,
			Accion:                    string(domain.AccionDecidirCoberturaGobernada),
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: "reserva:ct:desarrollo:cobertura:0001",
				ModuloID:   ports.ModuloContratacion,
				Tipo:       tipoRecursoDecisionCoberturaDesarrollo,
				Ambitos: map[string]string{
					"organizacion_ref":     organizacionAltaContratacionTemporalDesarrollo,
					"unidad_ejecutora_ref": unidadCoberturaContratacionTemporalDesarrollo,
				},
				Atributos: map[string]string{},
			},
			Finalidad:   finalidadDecisionCoberturaDesarrollo,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, candidata, err := autorizador.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx,
		solicitud,
		soporte.contexto.Resultado,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if _, errResumen := candidata.Resumen(); err != nil || errResumen != nil {
		t.Fatalf("candidata de cobertura no preparada: %v", err)
	}
	autoridad, ok := soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if !ok {
		t.Fatal("autoridad de asignaciones de prueba inesperada")
	}
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 {
		t.Fatalf("asignacion de cobertura no publicada antes de la candidata: %+v", autoridad)
	}
}

func TestAutorizadorConsultasCoberturaUsaServicioV3Real(t *testing.T) {
	soporte, autorizador, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctxPropuesta := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaPropuestaCobertura,
	)
	canal, err := soporte.ResolverContextoCanalCobertura(ctxPropuesta)
	if err != nil {
		t.Fatal(err)
	}
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: canal.AutenticacionRef,
		SesionRef:        canal.SesionRef,
		PerfilRef:        canal.PerfilRef,
	}
	contexto, err := soporte.ResolverContextoAutorizacionAltaV3(
		ctxPropuesta,
		solicitudContexto,
	)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		canal.OrganizacionRef,
		"expediente_temporal_desarrollo_0001",
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxPropuesta,
		solicitudContexto,
		contexto,
		analisis,
		soporte.reloj.Ahora(),
	); err != nil {
		t.Fatalf("propuesta no autorizada por V3: %v, autoridad=%+v, registro=%+v", err, soporte.autoridadAsignaciones, soporte.registroDecisionesAnalisis)
	}

	ctxResultado := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaResultadoCobertura,
	)
	contextoRecuperacion, err :=
		soporte.ResolverContextoRecuperacionResultadoCobertura(ctxResultado)
	if err != nil {
		t.Fatal(err)
	}
	solicitudLectura, err := ports.NuevaSolicitudLecturaResultadoCobertura(
		contextoRecuperacion,
		"expediente_temporal_desarrollo_0001",
		soporte.reloj.Ahora(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := autorizador.AutorizarLecturaResultadoCobertura(
		ctxResultado,
		solicitudLectura,
	)
	if err != nil || resultado != ports.AutorizacionLecturaResultadoCoberturaConcedida {
		t.Fatalf("lectura no autorizada por V3: %v %v", resultado, err)
	}
	soporte.mu.Lock()
	totalConcesiones := len(soporte.concesiones)
	soporte.mu.Unlock()
	registro := soporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	if totalConcesiones != 1 || registro.concesiones != 1 || registro.huella == "" {
		t.Fatalf("registro de propuesta no durable: memoria=%d, registro=%+v", totalConcesiones, registro)
	}
}

func TestAutorizacionPropuestaCoberturaExigeCamposYRegistroDurable(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		preparar func(*soporteAltaContratacionTemporalDesarrollo)
		esperado error
	}{
		{"concesion completa", nil, nil},
		{"campo ausente", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.instantaneaCobertura.VersionRol.Concesiones[0].CamposPermitidos =
				append([]string(nil), camposPreparacionPropuestaCoberturaDesarrollo[:len(camposPreparacionPropuestaCoberturaDesarrollo)-1]...)
		}, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
		{"campo vacio", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.instantaneaCobertura.VersionRol.Concesiones[0].CamposPermitidos = nil
		}, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
		{"obligacion desconocida", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.instantaneaCobertura.VersionRol.Concesiones[0].Obligaciones = []string{"obligacion_futura"}
		}, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
		{"concesion revocada", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.instantaneaCobertura.ControlVigenciaVersionRol.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
			s.instantaneaCobertura.ControlVigenciaVersionRol.Revision++
			s.instantaneaCobertura.ControlVigenciaVersionRol.ActoRef = "acto:revocacion:desarrollo"
			s.instantaneaCobertura.ControlVigenciaVersionRol.MotivoCodigo = "revocada"
		}, application.ErrPresentacionPropuestaCoberturaDenegada},
		{"denegacion sin registro", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.instantaneaCobertura.VersionRol.Concesiones = s.instantaneaCobertura.VersionRol.Concesiones[1:]
			s.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba).
				errDenegacion = puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
		}, application.ErrPresentacionPropuestaCoberturaNoDisponible},
		{"registrador caido", func(s *soporteAltaContratacionTemporalDesarrollo) {
			s.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba).
				errConcesion = puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
		}, application.ErrPresentacionPropuestaCoberturaNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, autorizador, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			var bitacora bytes.Buffer
			autorizador.incidencias = slog.New(slog.NewJSONHandler(&bitacora, nil))
			if caso.preparar != nil {
				caso.preparar(soporte)
			}
			ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaPropuestaCobertura)
			vinculo, err := soporte.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
				AutenticacionRef: vinculo.AutenticacionRef,
				SesionRef:        vinculo.SesionRef,
				PerfilRef:        vinculo.PerfilActivoRef,
			}
			analisis, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
				organizacionAltaContratacionTemporalDesarrollo,
				"expediente_temporal_desarrollo_0001", 2,
			)
			if err != nil {
				t.Fatal(err)
			}
			err = autorizador.AutorizarPresentacionPropuestaCobertura(
				ctx, solicitudContexto, soporte.contexto, analisis, soporte.reloj.Ahora(),
			)
			if caso.esperado == nil && err != nil ||
				caso.esperado != nil && !errors.Is(err, caso.esperado) {
				t.Fatalf("autorizacion=%v, esperado=%v", err, caso.esperado)
			}
			registro := soporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
			soporte.mu.Lock()
			memoria := len(soporte.concesiones)
			soporte.mu.Unlock()
			if memoria != 0 || caso.esperado == nil && (registro.concesiones != 1 || registro.huella == "") ||
				caso.nombre == "concesion revocada" && registro.denegaciones != 1 {
				t.Fatalf("registro de propuesta: memoria=%d, durable=%+v", memoria, registro)
			}
			if errors.Is(caso.esperado, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil) {
				registroTexto := bitacora.String()
				if registro.concesiones != 1 || registro.denegaciones != 0 ||
					!strings.Contains(registroTexto, `"evento":"proyeccion_cobertura_restringida"`) ||
					!strings.Contains(registroTexto, `"resultado":"sin_datos"`) ||
					strings.Contains(registroTexto, principal.ID) ||
					strings.Contains(registroTexto, principal.Attributes["certificate_sha256"]) ||
					strings.Contains(registroTexto, "expediente_temporal_desarrollo_0001") ||
					strings.Contains(registroTexto, organizacionAltaContratacionTemporalDesarrollo) {
					t.Fatalf("restriccion sin cierre o con datos privados: concesiones=%d denegaciones=%d registro=%q", registro.concesiones, registro.denegaciones, registroTexto)
				}
			}
		})
	}
}

func TestInstantaneaCoberturaCopiaCamposYObligaciones(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	copia := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantaneaCobertura)
	copia.VersionRol.Concesiones[0].CamposPermitidos[0] = "alterado"
	copia.VersionRol.Concesiones[0].Obligaciones[0] = "alterado"
	original := soporte.instantaneaCobertura.VersionRol.Concesiones[0]
	if original.CamposPermitidos[0] != camposPreparacionPropuestaCoberturaDesarrollo[0] ||
		original.Obligaciones[0] != "registrar_acceso" {
		t.Fatal("la copia altero la concesion original")
	}
}

func TestAutorizacionCoberturaDesarrolloFallaCerrado(t *testing.T) {
	soporte, autorizador, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if _, err := nuevoAutorizadorConsultasCoberturaDesarrollo(
		soporte,
		nil,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	); !errors.Is(err, errAutorizacionCoberturaDesarrolloNoDisponible) {
		t.Fatalf("autorizador nulo aceptado: %v", err)
	}
	ctxAlta := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaAltaSolicitudes,
	)
	vinculo, _ := soporte.contexto.Vinculo.Datos()
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
	}
	analisis, _ := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		organizacionAltaContratacionTemporalDesarrollo,
		"expediente_temporal_desarrollo_0001",
		2,
	)
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxAlta,
		solicitudContexto,
		soporte.contexto,
		analisis,
		soporte.reloj.Ahora(),
	); !errors.Is(err, application.ErrPresentacionPropuestaCoberturaDenegada) {
		t.Fatalf("propuesta autorizada bajo ruta de alta: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(ctxAlta)
	cancelar()
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxCancelado,
		solicitudContexto,
		soporte.contexto,
		analisis,
		soporte.reloj.Ahora(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
}

func escenarioAutorizacionCoberturaDesarrolloPrueba(
	t *testing.T,
) (
	*soporteAltaContratacionTemporalDesarrollo,
	*autorizadorConsultasCoberturaDesarrollo,
	dominiovec.Principal,
) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	principal := dominiovec.Principal{
		ID:            "certificado_rrhh_autorizacion_cobertura",
		Roles:         []string{"tecnico_rrhh"},
		AuthMethod:    dominiovec.AuthMethodCertificate,
		AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad":          AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("d", 64),
		},
	}
	contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alta, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	coberturaVEC, err :=
		nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
			vinculo.PrincipalID,
			vinculo.PerfilActivoRef,
			ahora,
		)
	if err != nil {
		t.Fatal(err)
	}
	analisisVEC, err :=
		nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(
			vinculo.PrincipalID,
			vinculo.PerfilActivoRef,
			ahora,
			fasesOperacionPredeterminadasCT()[operacionFaseAnalisisCT],
		)
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello:             &selloConsultasContratacionTemporalDesarrollo{},
		principalID:       principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto,
		motivo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_autorizacion",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos"),
			EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
				"motivo_",
				"crear-solicitud",
			),
		},
		instantanea:                  alta,
		instantaneaAnalisis:          analisisVEC,
		motivoRegistroAnalisis:       referenciaMotivoAutorizacionAnalisisDesarrollo("registro"),
		motivoRectificacionAnalisis:  referenciaMotivoAutorizacionAnalisisDesarrollo("rectificacion"),
		instantaneaCobertura:         coberturaVEC,
		motivoPropuestaCobertura:     referenciaMotivoAutorizacionCoberturaDesarrollo("propuesta"),
		motivoDecisionCobertura:      referenciaMotivoAutorizacionCoberturaDesarrollo("decision"),
		motivoRectificacionCobertura: referenciaMotivoAutorizacionCoberturaDesarrollo("rectificacion"),
		motivoResultadoCobertura:     referenciaMotivoAutorizacionCoberturaDesarrollo("resultado"),
		reloj:                        relojContratacionTemporalDesarrollo{},
		concesiones:                  make(map[string]struct{}),
		autoridadAsignaciones:        &autoridadAsignacionesContratacionTemporalDesarrolloPrueba{},
		registroDecisionesAnalisis:   &registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba{},
		instantaneasPorSolicitud:     make(map[string]dominiovec.InstantaneaAutorizacion),
	}
	soporte.contextoEsperadoRegistrado = contexto.Resultado
	soporte.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: contexto}
	generador := seguridadvec.GeneradorReferenciasCriptograficas{}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		soporte,
		soporte,
		soporte,
		soporte,
		soporte.reloj,
		generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{
			VigenciaDecision: 90 * time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	autorizador, err := nuevoAutorizadorConsultasCoberturaDesarrollo(
		soporte,
		servicio,
		generador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return soporte, autorizador, principal
}

func contextoRutaCoberturaDesarrolloPrueba(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	principal dominiovec.Principal,
	ruta string,
) context.Context {
	ahora := soporte.reloj.Ahora()
	return context.WithValue(
		context.Background(),
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{
			sello: soporte.sello, ruta: ruta, principal: principal,
			certificadoVerificadoEn: ahora.Add(-time.Second), certificadoValidoHasta: ahora.Add(time.Hour),
			contextoOperacion: &contextoOperacionCTDesarrollo{},
		},
	)
}

func TestRolAnalisisRectificacionNoReutilizaVersionHistorica(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, soporte.reloj.Ahora(), fasesOperacionPredeterminadasCT()[operacionFaseAnalisisCT])
	if err != nil {
		t.Fatal(err)
	}
	if actual.VersionRol.Version != 2 || actual.Validar() != nil {
		t.Fatal("rol nuevo inválido")
	}
	if actual.AsignacionPerfil.VersionRolRef != actual.VersionRol.Referencia() || actual.ControlVigenciaVersionRol.VersionRolRef != actual.VersionRol.Referencia() {
		t.Fatal("referencias de autoridad incoherentes")
	}
	historico := actual.VersionRol
	historico.Version = 1
	historico.Concesiones = append([]dominiovec.ConcesionRol(nil), actual.VersionRol.Concesiones[:1]...)
	if historico.Validar() != nil || historico.Referencia() == actual.VersionRol.Referencia() {
		t.Fatal("se reutiliza la identidad del rol anterior")
	}
	if len(actual.VersionRol.Concesiones) != 2 || actual.VersionRol.Concesiones[0].Accion != ports.AccionRegistrarAnalisis || actual.VersionRol.Concesiones[1].Accion != ports.AccionRectificarAnalisis {
		t.Fatal("concesiones inesperadas")
	}
}

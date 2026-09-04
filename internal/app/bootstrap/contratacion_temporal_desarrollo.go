package bootstrap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	contratacioncomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const expedienteContratacionTemporalDesarrolloRef = "expediente:ct:demo:0001"

type selloConsultasContratacionTemporalDesarrollo struct{}

type claveCapacidadConsultasContratacionTemporalDesarrollo struct{}

type capacidadConsultaContratacionTemporalDesarrollo struct {
	sello     *selloConsultasContratacionTemporalDesarrollo
	ruta      string
	principal vecdomain.Principal
}

// autoridadConsultasContratacionTemporalDesarrollo solo reconoce capacidades
// efimeras emitidas tras revalidar el certificado mTLS local. No representa
// autoridad corporativa ni se construye fuera del perfil de desarrollo.
type autoridadConsultasContratacionTemporalDesarrollo struct {
	sello       *selloConsultasContratacionTemporalDesarrollo
	resolvedor  *resolvedorIdentidadDesarrollo
	noCompuesta *capacidadNoCompuestaContratacionTemporalDesarrollo
}

type autorizadorLigadoContratacionTemporalDesarrollo interface {
	puertosvec.AutorizadorSolicitudLigadaV3
	puertosvec.PreparadorRegistroCompuestoSolicitudLigadaV3
}

type claveSolicitudAutorizacionContratacionTemporalDesarrollo struct{}

// autorizadorAnalisisContratacionTemporalDesarrollo liga a este contexto
// interno la solicitud ya construida por el caso de uso tras preparar el
// expediente y resolver su politica. soporteAlta no lee cuerpo ni cabeceras
// HTTP para ampliar los ambitos de la instantanea.
type autorizadorAnalisisContratacionTemporalDesarrollo struct {
	delegado autorizadorLigadoContratacionTemporalDesarrollo
	soporte  *soporteAltaContratacionTemporalDesarrollo
}

func (a *autorizadorAnalisisContratacionTemporalDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
) (
	vecdomain.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	if datos.Accion == ports.AccionRegistrarAnalisis ||
		datos.Accion == ports.AccionCrearSolicitud ||
		datos.Accion == ports.AccionRegistrarAsignacion ||
		datos.Accion == ports.AccionEmitirInformeJuridico {
		if ctx == nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
				errAltaContratacionTemporalDesarrolloNoDisponible
		}
		ctx = context.WithValue(
			ctx,
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
			datos,
		)
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (a *autorizadorAnalisisContratacionTemporalDesarrollo) PrepararRegistroCompuestoSolicitudLigadaV3(
	ctx context.Context,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
) (
	vecdomain.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	ruta := ""
	switch datos.Accion {
	case string(domain.AccionDecidirCoberturaGobernada):
		ruta = httpinterno.RutaDecisionCobertura
	case string(domain.AccionRectificarCoberturaGobernada):
		ruta = httpinterno.RutaRectificacionCobertura
	}
	if ruta != "" {
		if a.soporte == nil || ctx == nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
				errAltaContratacionTemporalDesarrolloNoDisponible
		}
		ctx = context.WithValue(
			ctx,
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
			datos,
		)
		if err := a.soporte.publicarInstantaneaDecisionCobertura(
			ctx,
			ruta,
		); err != nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
				err
		}
	}
	return a.delegado.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx, solicitud, resultado, generador,
	)
}

func nuevasRutasContratacionTemporalDesarrollo(
	cfg config.Config,
	resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
	registro io.Writer,
) (
	[]vechttp.RutaExacta,
	*autoridadConsultasContratacionTemporalDesarrollo,
	func(),
	error,
) {
	cfg = cfg.Normalize()
	resolvedorDesarrollo, esDesarrollo := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !cfg.DevelopmentEnabledByDoubleKey() || validarRedLocalDesarrollo(cfg) != nil ||
		!esDesarrollo || resolvedorDesarrollo == nil || derivador == nil ||
		!derivador.valido() {
		return nil, nil, nil, ErrActivacionDesarrolloInvalida
	}
	noCompuesta, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(registro)
	if err != nil {
		return nil, nil, nil, err
	}
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo()
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	reloj := relojContratacionTemporalDesarrollo{}
	alta, err := nuevasDependenciasAltaContratacionTemporalDesarrollo(
		cfg, resolvedorDesarrollo, derivador, sello, reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	cerrarAlta := true
	defer func() {
		if cerrarAlta {
			alta.cerrar()
		}
	}()
	servicioAnalisis, err := nuevasDependenciasAnalisisContratacionTemporalDesarrollo(
		&alta,
		derivador,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	coberturaReal, err := nuevasDependenciasCoberturaContratacionTemporalDesarrollo(
		derivador,
		&alta,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	asignacionReal, err := nuevasDependenciasAsignacionContratacionTemporalDesarrollo(
		derivador,
		&alta,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	informeJuridicoReal, err := nuevasDependenciasInformeJuridicoContratacionTemporalDesarrollo(
		derivador,
		&alta,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	cerrarCobertura := true
	defer func() {
		if cerrarCobertura {
			coberturaReal.cerrar()
		}
	}()
	rutaCatalogosAlta, err := nuevaRutaCatalogosAltaContratacionTemporalDesarrollo(origen)
	if err != nil {
		return nil, nil, nil, err
	}
	rutaConfiguracionAnalisis, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo()
	if err != nil {
		return nil, nil, nil, err
	}
	rutas, err := contratacioncomposicion.NuevasRutas(
		contratacioncomposicion.DependenciasRutas{
			AutoridadAlta:      alta.soporte,
			EjecutorAlta:       alta.servicio,
			Reloj:              reloj,
			AutoridadCobertura: alta.soporte,
			Presentador:        coberturaReal.presentador,
			Decisor:            coberturaReal.decisor,
			ConsultorResultado: coberturaReal.consultor,
			AutoridadAnalisis:  alta.soporte,
			EjecutorAnalisis:   servicioAnalisis,
			ConsultorCuadroRRHH: &consultorCuadroNoCompuestoContratacionTemporalDesarrollo{
				capacidadNoCompuestaContratacionTemporalDesarrollo: noCompuesta,
			},
			ConsultorDetalleRRHH: &consultorDetalleNoCompuestoContratacionTemporalDesarrollo{
				capacidadNoCompuestaContratacionTemporalDesarrollo: noCompuesta,
			},
			EjecutorSeleccion:               noCompuesta,
			AutoridadPropuestaFormalizacion: noCompuesta,
			EjecutorPropuestaFormalizacion:  noCompuesta,
			AutoridadCierreAdministrativo:   noCompuesta,
			EjecutorCierreAdministrativo:    noCompuesta,
			AutoridadAsignacion:             alta.soporte,
			EjecutorAsignacion:              asignacionReal,
			AutoridadInformeJuridico:        alta.soporte,
			EjecutorInformeJuridico:         informeJuridicoReal,
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	rutas = append(rutas, rutaCatalogosAlta, rutaConfiguracionAnalisis)
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sello:       sello,
		resolvedor:  resolvedorDesarrollo,
		noCompuesta: noCompuesta,
	}
	var cierre sync.Once
	cerrar := func() {
		cierre.Do(func() {
			coberturaReal.cerrar()
			alta.cerrar()
		})
	}
	cerrarCobertura = false
	cerrarAlta = false
	return rutas, autoridad, cerrar, nil
}

func (a *autoridadConsultasContratacionTemporalDesarrollo) proteger(
	siguiente http.Handler,
) http.Handler {
	return &revalidadorConsultasContratacionTemporalDesarrollo{
		siguiente: siguiente,
		autoridad: a,
	}
}

func (a *autoridadConsultasContratacionTemporalDesarrollo) AutorizarRutaExacta(
	ctx context.Context,
	ruta string,
) error {
	if a == nil || a.sello == nil || a.resolvedor == nil || ctx == nil ||
		ctx.Err() != nil {
		return vechttp.ErrAutoridadRutaExactaNoDisponible
	}
	capacidad, existe := ctx.Value(
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
	).(capacidadConsultaContratacionTemporalDesarrollo)
	if !existe {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if capacidad.sello != a.sello || capacidad.ruta != ruta {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	if a.noCompuesta != nil && a.noCompuesta.esRuta(ruta) {
		return a.noCompuesta.denegarRuta(ctx, ruta)
	}
	return nil
}

type revalidadorConsultasContratacionTemporalDesarrollo struct {
	siguiente http.Handler
	autoridad *autoridadConsultasContratacionTemporalDesarrollo
}

func (m *revalidadorConsultasContratacionTemporalDesarrollo) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if m == nil || m.siguiente == nil || m.autoridad == nil ||
		m.autoridad.sello == nil || m.autoridad.resolvedor == nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	r = peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
	if !esRutaContratacionTemporalDesarrollo(r) {
		m.siguiente.ServeHTTP(w, r)
		return
	}
	principal, err := m.autoridad.resolvedor.ResolveDemoIdentity(
		r.Context(), r,
	)
	if err == nil && principalContratacionTemporalDesarrolloValido(principal) {
		capacidad := capacidadConsultaContratacionTemporalDesarrollo{
			sello:     m.autoridad.sello,
			ruta:      r.URL.Path,
			principal: clonarPrincipalDesarrollo(principal),
		}
		r = r.WithContext(context.WithValue(
			r.Context(),
			claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidad,
		))
	}
	m.siguiente.ServeHTTP(w, r)
}

// peticionIdentidadConsultasContratacionTemporalDesarrollo conserva la hoja
// que identifica al cliente cuando la libreria TLS entrega tambien su cadena.
// Solo normaliza certificados que ya pertenecen, en el mismo orden, a la unica
// cadena verificada por mTLS; una cadena ambigua o con extras sigue llegando al
// resolvedor sin cambios y este la rechaza.
func peticionIdentidadConsultasContratacionTemporalDesarrollo(
	peticion *http.Request,
) *http.Request {
	if peticion == nil || peticion.TLS == nil || len(peticion.TLS.PeerCertificates) <= 1 ||
		len(peticion.TLS.VerifiedChains) != 1 ||
		len(peticion.TLS.PeerCertificates) > len(peticion.TLS.VerifiedChains[0]) {
		return peticion
	}
	cadenaVerificada := peticion.TLS.VerifiedChains[0]
	for indice, certificado := range peticion.TLS.PeerCertificates {
		if certificado == nil || cadenaVerificada[indice] == nil ||
			!bytes.Equal(certificado.Raw, cadenaVerificada[indice].Raw) {
			return peticion
		}
	}
	copia := new(http.Request)
	*copia = *peticion
	estadoTLS := *peticion.TLS
	estadoTLS.PeerCertificates = estadoTLS.PeerCertificates[:1:1]
	copia.TLS = &estadoTLS
	return copia
}

func esRutaContratacionTemporalDesarrollo(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if _, noCompuesta := rutasCapacidadNoCompuestaContratacionTemporal[r.URL.Path]; noCompuesta {
		return true
	}
	return r.URL.Path == httpinterno.RutaRegistroAnalisisRRHH ||
		r.URL.Path == httpinterno.RutaAltaSolicitudes ||
		r.URL.Path == httpinterno.RutaPropuestaCobertura ||
		r.URL.Path == httpinterno.RutaDecisionCobertura ||
		r.URL.Path == httpinterno.RutaRectificacionCobertura ||
		r.URL.Path == httpinterno.RutaResultadoCobertura ||
		r.URL.Path == httpinterno.RutaAsignaciones ||
		r.URL.Path == httpinterno.RutaReasignaciones ||
		r.URL.Path == httpinterno.RutaPreparacionesInformeJuridico ||
		r.URL.Path == rutaCatalogosAltaContratacionTemporalDesarrollo ||
		r.URL.Path == rutaConfiguracionAnalisisContratacionTemporalDesarrollo
}

func principalContratacionTemporalDesarrolloValido(
	principal vecdomain.Principal,
) bool {
	return principal.Validate() == nil &&
		principal.AuthMethod == vecdomain.AuthMethodCertificate &&
		principal.AuthAssurance == vecdomain.AuthAssuranceHigh &&
		len(principal.Roles) == 1 && principal.Roles[0] == "tecnico_rrhh" &&
		len(principal.Permissions) == 0 &&
		principal.Attributes["autoridad"] == AutoridadNoAutoritativa &&
		principal.Attributes["perfil_ejecucion"] == config.ExecutionProfileDevelopment
}

// origenConsultasContratacionTemporalDesarrollo es una fuente efimera,
// sintetica y no autoritativa. Solo satisface los puertos de lectura existentes
// para que la interfaz pueda demostrar el cuadro y su detalle.
type origenConsultasContratacionTemporalDesarrollo struct {
	mu        sync.RWMutex
	autoridad string
	pagina    ports.PaginaCuadroRRHH
	detalles  map[string]ports.DetalleExpedienteRRHH
}

func nuevoOrigenConsultasContratacionTemporalDesarrollo() *origenConsultasContratacionTemporalDesarrollo {
	creadoEn := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	actualizadoEn := time.Date(2026, 9, 2, 9, 30, 0, 0, time.UTC)
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef:   expedienteContratacionTemporalDesarrolloRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		NumeroVisible:   "2026/CT-0001",
		Version:         3,
		FlujoRef:        "flujo:ct:desarrollo",
		FlujoVersion:    1,
		FlujoHuella:     strings.Repeat("d", 64),
		FaseClave:       domain.ClaveFase("analisis"),
		EstadoClave:     domain.EstadoEnCurso,
		CentroRef:       centroAltaContratacionTemporalDesarrollo,
		CategoriaRef:    categoriaAltaContratacionTemporalDesarrollo,
		ModalidadClave:  domain.ClaveCatalogo("interinidad"),
		CreadoEn:        creadoEn,
		ActualizadoEn:   actualizadoEn,
	}
	resumenDetalle := resumen
	resumenDetalle.ModalidadClave = ""
	detalle := ports.DetalleExpedienteRRHH{
		Resumen: resumenDetalle,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: grupoSubgrupoAltaContratacionTemporalDesarrollo,
			MotivoClave:   motivoAltaContratacionTemporalDesarrollo,
			PeriodoInicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			PeriodoFin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		Hitos: []ports.HitoExpedienteRRHH{
			{
				Secuencia: 1, VersionExpediente: 1,
				AccionClave:   domain.ClaveCatalogo("alta"),
				RealizadaEn:   creadoEn,
				FaseDestino:   domain.ClaveFase("solicitud"),
				EstadoOrigen:  domain.EstadoPendiente,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 2, VersionExpediente: 2,
				AccionClave:   domain.ClaveCatalogo("analizar"),
				RealizadaEn:   creadoEn.Add(time.Hour),
				FaseOrigen:    domain.ClaveFase("solicitud"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 3, VersionExpediente: 3,
				AccionClave:   domain.ClaveCatalogo("actualizar"),
				RealizadaEn:   actualizadoEn,
				FaseOrigen:    domain.ClaveFase("analisis"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
		},
	}
	return &origenConsultasContratacionTemporalDesarrollo{
		autoridad: AutoridadNoAutoritativa,
		pagina: ports.PaginaCuadroRRHH{
			GeneradaEn:  actualizadoEn.Add(time.Minute),
			Expedientes: []ports.ResumenExpedienteRRHH{resumen},
		},
		detalles: map[string]ports.DetalleExpedienteRRHH{
			expedienteContratacionTemporalDesarrolloRef: detalle,
		},
	}
}

func (o *origenConsultasContratacionTemporalDesarrollo) registrarExpediente(
	expediente domain.Expediente,
) error {
	if o == nil || o.autoridad != AutoridadNoAutoritativa || expediente.Validar() != nil {
		return application.ErrConsultaRRHHNoDisponible
	}
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef: expediente.Referencia, OrganizacionRef: expediente.OrganizacionRef,
		NumeroVisible: expediente.NumeroVisible, Version: expediente.Version,
		FlujoRef: expediente.Flujo.DefinicionRef, FlujoVersion: expediente.Flujo.Version,
		FlujoHuella: expediente.Flujo.HuellaSHA256, FaseClave: expediente.FaseActual,
		EstadoClave: expediente.EstadoActual, CentroRef: expediente.Solicitud.CentroRef,
		CategoriaRef: expediente.Solicitud.CategoriaRef,
		CreadoEn:     expediente.CreadoEn, ActualizadoEn: expediente.ActualizadoEn,
	}
	hitos := make([]ports.HitoExpedienteRRHH, len(expediente.Actuaciones))
	for indice, actuacion := range expediente.Actuaciones {
		hitos[indice] = ports.HitoExpedienteRRHH{
			Secuencia: actuacion.Secuencia, VersionExpediente: actuacion.VersionExpediente,
			AccionClave: actuacion.AccionClave, RealizadaEn: actuacion.RealizadaEn,
			FaseOrigen: actuacion.FaseOrigen, FaseDestino: actuacion.FaseDestino,
			EstadoOrigen: actuacion.EstadoOrigen, EstadoDestino: actuacion.EstadoDestino,
		}
	}
	detalle := ports.DetalleExpedienteRRHH{
		Resumen: resumen,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: expediente.Solicitud.GrupoSubgrupo,
			MotivoClave:   expediente.Solicitud.MotivoClave,
			PeriodoInicio: expediente.Solicitud.Periodo.Inicio,
			PeriodoFin:    expediente.Solicitud.Periodo.Fin,
		},
		Hitos: hitos,
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, existe := o.detalles[expediente.Referencia]; existe {
		return application.ErrConsultaRRHHNoDisponible
	}
	o.pagina.Expedientes = append(o.pagina.Expedientes, resumen)
	sort.Slice(o.pagina.Expedientes, func(i, j int) bool {
		primero, segundo := o.pagina.Expedientes[i], o.pagina.Expedientes[j]
		if !primero.ActualizadoEn.Equal(segundo.ActualizadoEn) {
			return primero.ActualizadoEn.After(segundo.ActualizadoEn)
		}
		return primero.ExpedienteRef > segundo.ExpedienteRef
	})
	if o.pagina.GeneradaEn.Before(expediente.ActualizadoEn) {
		o.pagina.GeneradaEn = expediente.ActualizadoEn
	}
	o.detalles[expediente.Referencia] = detalle
	return nil
}

type consultorCuadroContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorCuadroContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if ctx == nil {
		return ports.PaginaCuadroRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	if c == nil || c.origen == nil {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	c.origen.mu.RLock()
	defer c.origen.mu.RUnlock()
	if c.origen.autoridad != AutoridadNoAutoritativa {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	expedientes := make([]ports.ResumenExpedienteRRHH, 0, len(c.origen.pagina.Expedientes))
	if solicitud.Cursor() == "" {
		for _, resumen := range c.origen.pagina.Expedientes {
			coincide := (solicitud.Texto() == "" || strings.HasPrefix(resumen.NumeroVisible, solicitud.Texto())) &&
				(solicitud.EstadoClave() == "" || solicitud.EstadoClave() == resumen.EstadoClave) &&
				(solicitud.FaseClave() == "" || solicitud.FaseClave() == resumen.FaseClave)
			if coincide {
				expedientes = append(expedientes, resumen)
			}
		}
	}
	if len(expedientes) > int(solicitud.Limite()) {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  c.origen.pagina.GeneradaEn,
		Expedientes: expedientes,
	}, nil
}

type consultorDetalleContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorDetalleContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	if ctx == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if c == nil || c.origen == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	c.origen.mu.RLock()
	defer c.origen.mu.RUnlock()
	detalle, existe := c.origen.detalles[solicitud.ExpedienteRef()]
	if c.origen.autoridad != AutoridadNoAutoritativa || !existe ||
		solicitud.VersionObservada() != 0 &&
			solicitud.VersionObservada() != detalle.Resumen.Version {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoObservable
	}
	return detalle.Clonar(), nil
}

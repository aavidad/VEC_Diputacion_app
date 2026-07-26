package cobertura

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitudGobiernoOperacionCoberturaInvalida = errors.New(
		"contratacion temporal: solicitud de gobierno de cobertura invalida",
	)
	ErrGobiernoOperacionCoberturaNoDisponible = errors.New(
		"contratacion temporal: gobierno de operacion de cobertura no disponible",
	)
	ErrGobiernoOperacionCoberturaNoConfiable = errors.New(
		"contratacion temporal: gobierno de operacion de cobertura no confiable",
	)
)

// RelojGobiernoOperacionCobertura es una autoridad servidor. HTTP, CLI, MCP y
// escritorio no suministran instantes; la composición inyecta esta capacidad.
type RelojGobiernoOperacionCobertura interface {
	AhoraGobiernoOperacionCobertura(context.Context) (time.Time, error)
}

// SolicitudGobiernoOperacionCobertura solo contiene la intención y las
// coordenadas mínimas. No acepta acción ni instante libres.
type SolicitudGobiernoOperacionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	organizacionRef   string
	expedienteRef     string
	versionExpediente uint64
	tipo              domain.TipoDecisionCoberturaGobernada
	accion            domain.ClaveCatalogo
}

func NuevaSolicitudGobiernoDecisionCobertura(
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
) (SolicitudGobiernoOperacionCobertura, error) {
	return nuevaSolicitudGobiernoOperacionCobertura(
		organizacionRef,
		expedienteRef,
		versionExpediente,
		domain.DecisionCoberturaInicial,
	)
}

func NuevaSolicitudGobiernoRectificacionCobertura(
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
) (SolicitudGobiernoOperacionCobertura, error) {
	return nuevaSolicitudGobiernoOperacionCobertura(
		organizacionRef,
		expedienteRef,
		versionExpediente,
		domain.DecisionCoberturaRectificacion,
	)
}

func nuevaSolicitudGobiernoOperacionCobertura(
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	tipo domain.TipoDecisionCoberturaGobernada,
) (SolicitudGobiernoOperacionCobertura, error) {
	solicitud := SolicitudGobiernoOperacionCobertura{
		organizacionRef:   organizacionRef,
		expedienteRef:     expedienteRef,
		versionExpediente: versionExpediente,
		tipo:              tipo,
		accion:            accionOperacionDecisionCobertura(tipo),
	}
	if solicitud.validar() != nil {
		return SolicitudGobiernoOperacionCobertura{},
			ErrSolicitudGobiernoOperacionCoberturaInvalida
	}
	return solicitud, nil
}

func (s SolicitudGobiernoOperacionCobertura) validar() error {
	if !domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!domain.ReferenciaOpacaValida(s.expedienteRef) ||
		s.versionExpediente == 0 ||
		s.versionExpediente >= MaximoEnteroSeguroOperacionDecisionCobertura ||
		!tipoDecisionCoberturaValido(s.tipo) ||
		s.accion != accionOperacionDecisionCobertura(s.tipo) {
		return ErrSolicitudGobiernoOperacionCoberturaInvalida
	}
	return nil
}

// SolicitudResolucionGobiernoOperacionCobertura nace únicamente dentro de
// ObtenerGobierno... después de consultar el reloj servidor.
type SolicitudResolucionGobiernoOperacionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	solicitud SolicitudGobiernoOperacionCobertura
	instante  time.Time
}

func (s SolicitudResolucionGobiernoOperacionCobertura) Coordenadas() (
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	accion domain.ClaveCatalogo,
	instante time.Time,
	err error,
) {
	if s.solicitud.validar() != nil ||
		!instanteOperacionDecisionCoberturaValido(s.instante) {
		return "", "", 0, "", time.Time{},
			ErrSolicitudGobiernoOperacionCoberturaInvalida
	}
	return s.solicitud.organizacionRef,
		s.solicitud.expedienteRef,
		s.solicitud.versionExpediente,
		s.solicitud.accion,
		s.instante,
		nil
}

// PublicacionGobiernoOperacionCobertura es una respuesta nominal no
// autoritativa. Solo ObtenerGobierno... puede convertirla en capacidad.
type PublicacionGobiernoOperacionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Catalogo          domain.CatalogoViasCobertura
	Politica          domain.PoliticaDecisionCobertura
	PoliticaActuacion PublicacionPoliticaActuacionCobertura
}

// ResolutorGobiernoOperacionCobertura debe estar implementado e inyectado por
// la composición servidor; forma parte del TCB junto con su almacén durable.
type ResolutorGobiernoOperacionCobertura interface {
	ResolverGobiernoOperacionCobertura(
		context.Context,
		SolicitudResolucionGobiernoOperacionCobertura,
	) (PublicacionGobiernoOperacionCobertura, error)
}

// GobiernoOperacionCobertura es una capacidad efímera opaca. O4-04 debe
// revalidar la publicación durable dentro de la transacción del efecto.
type GobiernoOperacionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	solicitud         SolicitudGobiernoOperacionCobertura
	catalogoPublicado domain.PublicacionCatalogoViasCobertura
	politicaPublicada domain.PublicacionPoliticaDecisionCobertura
	actuacion         PublicacionPoliticaActuacionCobertura
	evaluadaEn        time.Time
	validaHasta       time.Time
}

// DatosGobiernoOperacionCobertura es una vista defensiva, no una autoridad.
type DatosGobiernoOperacionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Catalogo           domain.CatalogoViasCobertura
	Politica           domain.PoliticaDecisionCobertura
	PoliticaActuacion  PublicacionPoliticaActuacionCobertura
	Accion             domain.ClaveCatalogo
	FinalidadCTClave   domain.ClaveCatalogo
	FinalidadCTRef     string
	FinalidadVEC       domain.ClaveCatalogo
	UnidadEjecutoraRef string
	FaseDestino        domain.ClaveFase
	EstadoDestino      domain.EstadoOperativo
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	EvaluadaEn         time.Time
	ValidaHasta        time.Time
}

// ObtenerGobiernoOperacionCobertura crea el instante y la solicitud interna
// desde el reloj servidor. Errores privados del reloj o resolutor no salen.
func ObtenerGobiernoOperacionCobertura(
	ctx context.Context,
	reloj RelojGobiernoOperacionCobertura,
	resolutor ResolutorGobiernoOperacionCobertura,
	solicitud SolicitudGobiernoOperacionCobertura,
) (GobiernoOperacionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(reloj) ||
		dependenciaGobiernoOperacionCoberturaNula(resolutor) ||
		solicitud.validar() != nil {
		return GobiernoOperacionCobertura{},
			ErrSolicitudGobiernoOperacionCoberturaInvalida
	}
	instante, err := ahoraGobiernoOperacionCobertura(ctx, reloj)
	if err != nil {
		return GobiernoOperacionCobertura{}, err
	}
	interna := SolicitudResolucionGobiernoOperacionCobertura{
		solicitud: solicitud,
		instante:  instante,
	}
	publicacion, err := resolutor.ResolverGobiernoOperacionCobertura(
		ctx,
		interna,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return GobiernoOperacionCobertura{},
			errors.Join(
				ErrGobiernoOperacionCoberturaNoDisponible,
				errContexto,
			)
	}
	if err != nil {
		return GobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoDisponible
	}
	gobierno, err := sellarGobiernoOperacionCobertura(
		interna,
		publicacion,
	)
	if err != nil {
		return GobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	return gobierno, nil
}

func ahoraGobiernoOperacionCobertura(
	ctx context.Context,
	reloj RelojGobiernoOperacionCobertura,
) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{},
			errors.Join(ErrGobiernoOperacionCoberturaNoDisponible, err)
	}
	instante, err := reloj.AhoraGobiernoOperacionCobertura(ctx)
	if errContexto := ctx.Err(); errContexto != nil {
		return time.Time{},
			errors.Join(
				ErrGobiernoOperacionCoberturaNoDisponible,
				errContexto,
			)
	}
	if err != nil {
		return time.Time{}, ErrGobiernoOperacionCoberturaNoDisponible
	}
	if !instanteOperacionDecisionCoberturaValido(instante) {
		return time.Time{}, ErrGobiernoOperacionCoberturaNoConfiable
	}
	return instante, nil
}

func sellarGobiernoOperacionCobertura(
	solicitud SolicitudResolucionGobiernoOperacionCobertura,
	publicacion PublicacionGobiernoOperacionCobertura,
) (GobiernoOperacionCobertura, error) {
	catalogo, politica, actuacion, instante, err :=
		validarPublicacionGobiernoOperacionCobertura(
			solicitud,
			publicacion,
		)
	if err != nil {
		return GobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	validaHasta, valida := limiteVigenciaGobiernoOperacionCobertura(
		instante,
		catalogo.Vigencia(),
		politica.Vigencia(),
		actuacion.Vigencia,
	)
	if !valida {
		return GobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	return GobiernoOperacionCobertura{
		solicitud:         solicitud.solicitud,
		catalogoPublicado: catalogo.Publicacion(),
		politicaPublicada: politica.Publicacion(),
		actuacion:         actuacion,
		evaluadaEn:        instante,
		validaHasta:       validaHasta,
	}, nil
}

func validarPublicacionGobiernoOperacionCobertura(
	solicitud SolicitudResolucionGobiernoOperacionCobertura,
	publicacion PublicacionGobiernoOperacionCobertura,
) (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
	PublicacionPoliticaActuacionCobertura,
	time.Time,
	error,
) {
	organizacion, expediente, version, accion, instante, err :=
		solicitud.Coordenadas()
	if err != nil ||
		publicacion.OrganizacionRef != organizacion ||
		publicacion.ExpedienteRef != expediente ||
		publicacion.VersionExpediente != version ||
		publicacion.PoliticaActuacion.Validar() != nil {
		return gobiernoOperacionCoberturaInvalido()
	}
	catalogo, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		publicacion.Catalogo.Publicacion(),
	)
	politica, errPolitica := domain.RestaurarPoliticaDecisionCobertura(
		publicacion.Politica.Publicacion(),
		catalogo,
	)
	finalidadClave, finalidadRef := politica.Finalidad()
	actuacion := publicacion.PoliticaActuacion
	if errCatalogo != nil || errPolitica != nil ||
		!catalogo.VigenteEn(instante) ||
		politica.ValidarPara(
			catalogo,
			organizacion,
			finalidadClave,
			finalidadRef,
			instante,
		) != nil ||
		actuacion.OrganizacionRef != organizacion ||
		actuacion.Accion != accion ||
		actuacion.Catalogo != catalogo.Identidad() ||
		actuacion.Politica != politica.Identidad() ||
		actuacion.FinalidadContratacionClave != finalidadClave ||
		actuacion.FinalidadContratacionRef != finalidadRef ||
		actuacion.PublicadaEn.After(instante) ||
		!vigenciaContieneInstante(actuacion.Vigencia, instante) {
		return gobiernoOperacionCoberturaInvalido()
	}
	return catalogo, politica, actuacion, instante, nil
}

func gobiernoOperacionCoberturaInvalido() (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
	PublicacionPoliticaActuacionCobertura,
	time.Time,
	error,
) {
	return domain.CatalogoViasCobertura{},
		domain.PoliticaDecisionCobertura{},
		PublicacionPoliticaActuacionCobertura{},
		time.Time{},
		ErrGobiernoOperacionCoberturaNoConfiable
}

func limiteVigenciaGobiernoOperacionCobertura(
	desde time.Time,
	vigencias ...domain.VigenciaCatalogoCobertura,
) (time.Time, bool) {
	hasta := desde.Add(MaximoLeaseOperacionDecisionCobertura)
	for _, vigencia := range vigencias {
		if vigencia.Validar() != nil ||
			!vigenciaContieneInstante(vigencia, desde) {
			return time.Time{}, false
		}
		if !vigencia.Hasta.IsZero() && vigencia.Hasta.Before(hasta) {
			hasta = vigencia.Hasta
		}
	}
	return hasta, instanteOperacionDecisionCoberturaValido(hasta) &&
		hasta.After(desde)
}

func vigenciaContieneInstante(
	vigencia domain.VigenciaCatalogoCobertura,
	instante time.Time,
) bool {
	return vigencia.Validar() == nil &&
		!instante.Before(vigencia.Desde) &&
		(vigencia.Hasta.IsZero() || instante.Before(vigencia.Hasta))
}

// DesplegarPara consulta otra vez el reloj servidor. Rechaza cancelación,
// retroceso temporal, caducidad y reutilización con otras coordenadas.
func (g GobiernoOperacionCobertura) DesplegarPara(
	ctx context.Context,
	reloj RelojGobiernoOperacionCobertura,
	solicitud SolicitudGobiernoOperacionCobertura,
) (DatosGobiernoOperacionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(reloj) ||
		!solicitudesGobiernoOperacionCoberturaIguales(
			g.solicitud,
			solicitud,
		) {
		return DatosGobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	instanteUso, err := ahoraGobiernoOperacionCobertura(ctx, reloj)
	if err != nil {
		return DatosGobiernoOperacionCobertura{}, err
	}
	if instanteUso.Before(g.evaluadaEn) ||
		!instanteUso.Before(g.validaHasta) {
		return DatosGobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	catalogo, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		g.catalogoPublicado,
	)
	politica, errPolitica := domain.RestaurarPoliticaDecisionCobertura(
		g.politicaPublicada,
		catalogo,
	)
	interna := SolicitudResolucionGobiernoOperacionCobertura{
		solicitud: solicitud,
		instante:  g.evaluadaEn,
	}
	publicacion := PublicacionGobiernoOperacionCobertura{
		OrganizacionRef:   solicitud.organizacionRef,
		ExpedienteRef:     solicitud.expedienteRef,
		VersionExpediente: solicitud.versionExpediente,
		Catalogo:          catalogo,
		Politica:          politica,
		PoliticaActuacion: g.actuacion,
	}
	catalogo, politica, actuacion, _, err :=
		validarPublicacionGobiernoOperacionCobertura(
			interna,
			publicacion,
		)
	if errCatalogo != nil || errPolitica != nil || err != nil {
		return DatosGobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	motivo := motivoAutorizacionGobiernoParaAccion(
		actuacion,
		solicitud.accion,
	)
	if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return DatosGobiernoOperacionCobertura{},
			ErrGobiernoOperacionCoberturaNoConfiable
	}
	return DatosGobiernoOperacionCobertura{
		Catalogo:           catalogo,
		Politica:           politica,
		PoliticaActuacion:  actuacion,
		Accion:             solicitud.accion,
		FinalidadCTClave:   actuacion.FinalidadContratacionClave,
		FinalidadCTRef:     actuacion.FinalidadContratacionRef,
		FinalidadVEC:       actuacion.FinalidadAutorizacionVEC,
		UnidadEjecutoraRef: actuacion.UnidadEjecutoraRef,
		FaseDestino:        actuacion.FaseDestino,
		EstadoDestino:      actuacion.EstadoDestino,
		MotivoAutorizacion: motivo,
		EvaluadaEn:         g.evaluadaEn,
		ValidaHasta:        g.validaHasta,
	}, nil
}

func motivoAutorizacionGobiernoParaAccion(
	politica PublicacionPoliticaActuacionCobertura,
	accion domain.ClaveCatalogo,
) dominiovec.ReferenciaEntradaCatalogo {
	if accion == domain.AccionDecidirCoberturaGobernada {
		return politica.MotivoAutorizacionDecidir
	}
	if accion == domain.AccionRectificarCoberturaGobernada {
		return politica.MotivoAutorizacionRectificar
	}
	return dominiovec.ReferenciaEntradaCatalogo{}
}

func solicitudesGobiernoOperacionCoberturaIguales(
	primera SolicitudGobiernoOperacionCobertura,
	segunda SolicitudGobiernoOperacionCobertura,
) bool {
	return primera.validar() == nil && segunda.validar() == nil &&
		primera.versionExpediente == segunda.versionExpediente &&
		primera.tipo == segunda.tipo &&
		primera.accion == segunda.accion &&
		referenciasOperacionDecisionCoberturaIguales(
			primera.organizacionRef,
			segunda.organizacionRef,
		) &&
		referenciasOperacionDecisionCoberturaIguales(
			primera.expedienteRef,
			segunda.expedienteRef,
		)
}

func dependenciaGobiernoOperacionCoberturaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

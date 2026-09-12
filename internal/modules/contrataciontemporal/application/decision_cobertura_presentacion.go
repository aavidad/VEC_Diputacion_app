package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	TiempoMaximoPresentacionPropuestaCobertura = 5 * time.Second
	redaccionSolicitudProponerCobertura        = "[SOLICITUD-PROPONER-COBERTURA-REDACTADA]"
)

var (
	ErrServicioPresentacionPropuestaCoberturaInvalido = errors.New(
		"contratacion temporal: servicio de presentacion de propuesta de cobertura invalido",
	)
	ErrSolicitudProponerCoberturaInvalida = errors.New(
		"contratacion temporal: solicitud de propuesta de cobertura invalida",
	)
	ErrPresentacionPropuestaCoberturaDenegada = errors.New(
		"contratacion temporal: presentacion de propuesta de cobertura denegada",
	)
	ErrPresentacionPropuestaCoberturaNoDisponible = errors.New(
		"contratacion temporal: presentacion de propuesta de cobertura no disponible",
	)
	ErrPresentacionPropuestaCoberturaEnConflicto = errors.New(
		"contratacion temporal: presentacion de propuesta de cobertura en conflicto",
	)
	ErrPresentacionPropuestaCoberturaNoConfiable = errors.New(
		"contratacion temporal: presentacion de propuesta de cobertura no confiable",
	)
)

// SolicitudProponerCobertura contiene solo coordenadas opacas del canal. El
// perfil es una petición de activación que las autoridades VEC deben resolver;
// nunca se interpreta como actor, rol ni permiso efectivo.
type SolicitudProponerCobertura struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
	ExpedienteRef    string
	VersionEsperada  uint64
}

func (SolicitudProponerCobertura) String() string {
	return redaccionSolicitudProponerCobertura
}

func (s SolicitudProponerCobertura) GoString() string { return s.String() }

func (s SolicitudProponerCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (s SolicitudProponerCobertura) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// PresentacionPropuestaCobertura omite la propuesta exacta y todo su material
// transitorio. Las claves se traducen en el adaptador de entrada mediante i18n
// y las vías proceden del catálogo publicado, nunca de una lista compilada.
type PresentacionPropuestaCobertura struct {
	Estado             domain.EstadoPropuestaDecisionCobertura
	ViaRecomendada     domain.ClaveCatalogo
	Evaluaciones       []domain.EvaluacionViaPropuestaCobertura
	MotivosAlternativa []MotivoAlternativaPropuestaCobertura
	IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
}

// MotivoAlternativaPropuestaCobertura es la única vista transportable del
// motivo vigente que habilita escoger una vía distinta de la recomendada.
// La clave procede de la publicación que el resolutor acaba de verificar; la
// etiqueta es una clave i18n nominal de la composición. No transporta una
// autorización ni una referencia de catálogo.
type MotivoAlternativaPropuestaCobertura struct {
	Clave        domain.ClaveCatalogo
	ViaClave     domain.ClaveCatalogo
	EtiquetaI18n domain.ClaveCatalogo
}

// MotivoAlternativaCobertura identifica una alternativa configurada por la
// composición. Su mera presencia no la autoriza: cada propuesta la resuelve
// contra la publicación vigente antes de exponerla.
type MotivoAlternativaCobertura struct {
	ViaClave     domain.ClaveCatalogo
	Clave        domain.ClaveCatalogo
	EtiquetaI18n domain.ClaveCatalogo
}

type resolutorClaveMotivoPresentacionCobertura interface {
	ResolverClave(
		context.Context,
		domain.ClaveCatalogo,
		time.Time,
	) (cobertura.ResolucionMotivoDecisionCobertura, error)
}

// AutorizadorPresentacionPropuestaCobertura es la puerta de lectura dinámica.
// Recibe capacidades ya resueltas en servidor y no admite actor, rol, acción,
// reloj, catálogo o política declarados como texto por un canal. Solo devuelve
// nil después de aplicar finalidad y alcance exactos y registrar la evidencia
// de acceso en el subsistema separado de auditoría; nunca produce efectos de
// negocio sobre cobertura.
type AutorizadorPresentacionPropuestaCobertura interface {
	AutorizarPresentacionPropuestaCobertura(
		context.Context,
		ports.SolicitudResolverContextoAutorizacionAltaV3,
		ports.ContextoAutorizacionAltaV3,
		cobertura.SolicitudInstantaneaAnalisisDurableO3,
		time.Time,
	) error
}

type ServicioPresentacionPropuestaCobertura struct {
	contextos    ports.ResolutorContextoAutorizacionAltaV3
	accesos      AutorizadorPresentacionPropuestaCobertura
	analisis     cobertura.LectorExpedienteAnalisisDurableO3
	reloj        cobertura.RelojGobiernoOperacionCobertura
	gobierno     cobertura.ResolutorGobiernoOperacionCobertura
	motivos      resolutorClaveMotivoPresentacionCobertura
	alternativas []MotivoAlternativaCobertura
	coberturas   *PreparadorGlobalCobertura
}

func NuevoServicioPresentacionPropuestaCobertura(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	accesos AutorizadorPresentacionPropuestaCobertura,
	analisis cobertura.LectorExpedienteAnalisisDurableO3,
	reloj cobertura.RelojGobiernoOperacionCobertura,
	gobierno cobertura.ResolutorGobiernoOperacionCobertura,
	motivos resolutorClaveMotivoPresentacionCobertura,
	alternativas []MotivoAlternativaCobertura,
	coberturas *PreparadorGlobalCobertura,
) (*ServicioPresentacionPropuestaCobertura, error) {
	if dependenciaNula(contextos) || dependenciaNula(accesos) ||
		dependenciaNula(analisis) || dependenciaNula(reloj) ||
		dependenciaNula(gobierno) || dependenciaNula(motivos) ||
		!alternativasCoberturaValidas(alternativas) || dependenciaNula(coberturas) {
		return nil, ErrServicioPresentacionPropuestaCoberturaInvalido
	}
	return &ServicioPresentacionPropuestaCobertura{
		contextos: contextos, accesos: accesos, analisis: analisis,
		reloj: reloj, gobierno: gobierno, motivos: motivos,
		alternativas: append([]MotivoAlternativaCobertura(nil), alternativas...),
		coberturas:   coberturas,
	}, nil
}

func (s *ServicioPresentacionPropuestaCobertura) Proponer(
	ctx context.Context,
	solicitud SolicitudProponerCobertura,
) (PresentacionPropuestaCobertura, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return PresentacionPropuestaCobertura{},
			ErrServicioPresentacionPropuestaCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return PresentacionPropuestaCobertura{}, err
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoPresentacionPropuestaCobertura,
	)
	defer cancelar()

	solicitudContexto, solicitudAnalisis, err :=
		solicitudesPresentacionPropuestaCobertura(solicitud)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrSolicitudProponerCoberturaInvalida
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(
		operacion,
		solicitudContexto,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloContexto(operacion, err)
	}
	instanteAcceso, err := s.ahora(operacion)
	if err != nil ||
		contexto.ValidarPara(solicitudContexto, instanteAcceso) != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloContexto(operacion, err)
	}
	if err := s.autorizar(
		operacion, solicitudContexto, contexto, solicitudAnalisis,
		instanteAcceso,
	); err != nil {
		return PresentacionPropuestaCobertura{}, err
	}

	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		operacion,
		s.analisis,
		solicitudAnalisis,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloDependencia(operacion, err)
	}
	expediente, analisisRef, analisisHuella, err :=
		instantanea.DesplegarPara(solicitudAnalisis)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	solicitudGobierno, err := solicitudGobiernoParaPresentacion(expediente)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaEnConflicto
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(
		operacion,
		s.reloj,
		s.gobierno,
		solicitudGobierno,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloDependencia(operacion, err)
	}
	datosGobierno, err := gobierno.DesplegarPara(
		operacion,
		s.reloj,
		solicitudGobierno,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloDependencia(operacion, err)
	}
	if expediente.Analisis == nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	datosGlobales, err := nuevosDatosPreparacionGlobalCobertura(
		analisisRef,
		analisisHuella,
		datosGobierno.Catalogo,
		datosGobierno.Politica,
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
		expediente.Analisis.CategoriaRef,
		expediente.Analisis.Periodo,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	preparacion, err := s.coberturas.Preparar(operacion, datosGlobales)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloDependencia(operacion, err)
	}
	// Resolver los motivos puede consultar la publicación durable. Se hace
	// antes de la revalidación final, para que ésta cubra también esa E/S.
	instanteMotivos, err := s.ahora(operacion)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloContexto(operacion, err)
	}
	datosMotivos, err := preparacion.DatosCrearPropuestaEn(instanteMotivos)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	propuestaMotivos, err := domain.CrearPropuestaDecisionCobertura(datosMotivos)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	motivosAlternativa, err := s.proyectarMotivosAlternativa(
		operacion,
		instanteMotivos,
		propuestaMotivos.Evaluaciones(),
	)
	if err != nil {
		return PresentacionPropuestaCobertura{}, err
	}

	instantePresentacion, err := s.ahora(operacion)
	if err != nil ||
		contexto.ValidarPara(
			solicitudContexto,
			instantePresentacion,
		) != nil {
		return PresentacionPropuestaCobertura{},
			s.clasificarFalloContexto(operacion, err)
	}
	if err := s.autorizar(
		operacion, solicitudContexto, contexto, solicitudAnalisis,
		instantePresentacion,
	); err != nil {
		return PresentacionPropuestaCobertura{}, err
	}
	datosPropuesta, err := preparacion.DatosCrearPropuestaEn(
		instantePresentacion,
	)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	propuesta, err := domain.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	identidad, err := propuesta.IdentidadSemantica()
	if err != nil {
		return PresentacionPropuestaCobertura{},
			ErrPresentacionPropuestaCoberturaNoConfiable
	}
	return PresentacionPropuestaCobertura{
		Estado:             propuesta.Estado(),
		ViaRecomendada:     propuesta.ViaPropuesta(),
		Evaluaciones:       propuesta.Evaluaciones(),
		MotivosAlternativa: motivosAlternativa,
		IdentidadSemantica: identidad,
	}, nil
}

// ProponerParaAdaptador toma una instantánea privada antes de que la salida
// pueda cruzar a una frontera. La presentación pública continúa disponible
// para consumidores de aplicación, pero no es una fuente confiable para HTTP.
func (s *ServicioPresentacionPropuestaCobertura) ProponerParaAdaptador(
	ctx context.Context,
	solicitud SolicitudProponerCobertura,
) (ResultadoPropuestaCoberturaParaAdaptador, error) {
	presentacion, err := s.Proponer(ctx, solicitud)
	if err != nil {
		return ResultadoPropuestaCoberturaParaAdaptador{}, err
	}
	return nuevaResultadoPropuestaCoberturaParaAdaptador(presentacion)
}

func (s *ServicioPresentacionPropuestaCobertura) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.contextos) &&
		!dependenciaNula(s.accesos) && !dependenciaNula(s.analisis) &&
		!dependenciaNula(s.reloj) && !dependenciaNula(s.gobierno) &&
		!dependenciaNula(s.motivos) && alternativasCoberturaValidas(s.alternativas) &&
		!dependenciaNula(s.coberturas)
}

func alternativasCoberturaValidas(
	alternativas []MotivoAlternativaCobertura,
) bool {
	if len(alternativas) == 0 || len(alternativas) > 64 {
		return false
	}
	vias := make(map[domain.ClaveCatalogo]struct{}, len(alternativas))
	for _, alternativa := range alternativas {
		if !alternativa.ViaClave.Valida() || !alternativa.Clave.Valida() ||
			!alternativa.EtiquetaI18n.Valida() {
			return false
		}
		if _, repetida := vias[alternativa.ViaClave]; repetida {
			return false
		}
		vias[alternativa.ViaClave] = struct{}{}
	}
	return true
}

func (s *ServicioPresentacionPropuestaCobertura) proyectarMotivosAlternativa(
	ctx context.Context,
	instante time.Time,
	evaluaciones []domain.EvaluacionViaPropuestaCobertura,
) ([]MotivoAlternativaPropuestaCobertura, error) {
	viables := make(map[domain.ClaveCatalogo]struct{}, len(evaluaciones))
	for _, evaluacion := range evaluaciones {
		if evaluacion.Estado == domain.EvaluacionViaCoberturaViable {
			viables[evaluacion.ViaClave] = struct{}{}
		}
	}
	salida := make([]MotivoAlternativaPropuestaCobertura, 0, len(s.alternativas))
	for _, alternativa := range s.alternativas {
		if _, viable := viables[alternativa.ViaClave]; !viable {
			continue
		}
		resolucion, err := s.motivos.ResolverClave(ctx, alternativa.Clave, instante)
		if err != nil {
			if errContexto := ctx.Err(); errContexto != nil {
				return nil, errContexto
			}
			return nil, ErrPresentacionPropuestaCoberturaNoConfiable
		}
		motivo, err := resolucion.Motivo()
		if err != nil || motivo.ReferenciaCatalogo.EntradaClave != string(alternativa.Clave) {
			return nil, ErrPresentacionPropuestaCoberturaNoConfiable
		}
		salida = append(salida, MotivoAlternativaPropuestaCobertura{
			Clave: alternativa.Clave, ViaClave: alternativa.ViaClave,
			EtiquetaI18n: alternativa.EtiquetaI18n,
		})
	}
	return salida, nil
}

func solicitudesPresentacionPropuestaCobertura(
	solicitud SolicitudProponerCobertura,
) (
	ports.SolicitudResolverContextoAutorizacionAltaV3,
	cobertura.SolicitudInstantaneaAnalisisDurableO3,
	error,
) {
	contexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.AutenticacionRef,
		SesionRef:        solicitud.SesionRef,
		PerfilRef:        solicitud.PerfilRef,
	}
	analisis, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		solicitud.OrganizacionRef,
		solicitud.ExpedienteRef,
		solicitud.VersionEsperada,
	)
	if contexto.Validar() != nil || err != nil {
		return ports.SolicitudResolverContextoAutorizacionAltaV3{},
			cobertura.SolicitudInstantaneaAnalisisDurableO3{},
			ErrSolicitudProponerCoberturaInvalida
	}
	return contexto, analisis, nil
}

func solicitudGobiernoParaPresentacion(
	expediente domain.Expediente,
) (cobertura.SolicitudGobiernoOperacionCobertura, error) {
	if expediente.Validar() != nil || expediente.Analisis == nil ||
		!expediente.Analisis.HabilitaAvance() ||
		expediente.Asignacion != nil {
		return cobertura.SolicitudGobiernoOperacionCobertura{},
			ErrPresentacionPropuestaCoberturaEnConflicto
	}
	if expediente.ViaCobertura == nil &&
		len(expediente.DecisionesCobertura) == 0 {
		return cobertura.NuevaSolicitudGobiernoDecisionCobertura(
			expediente.OrganizacionRef,
			expediente.Referencia,
			expediente.Version,
		)
	}
	if expediente.ViaCobertura == nil ||
		expediente.ViaCobertura.DecisionGobernada == nil ||
		len(expediente.DecisionesCobertura) == 0 {
		return cobertura.SolicitudGobiernoOperacionCobertura{},
			ErrPresentacionPropuestaCoberturaEnConflicto
	}
	return cobertura.NuevaSolicitudGobiernoRectificacionCobertura(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
}

func (s *ServicioPresentacionPropuestaCobertura) ahora(
	ctx context.Context,
) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	instante, err := s.reloj.AhoraGobiernoOperacionCobertura(ctx)
	if errContexto := ctx.Err(); errContexto != nil {
		return time.Time{}, errContexto
	}
	if err != nil || !domain.InstanteUTCCanonico(instante) {
		return time.Time{}, ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return instante, nil
}

func (s *ServicioPresentacionPropuestaCobertura) autorizar(
	ctx context.Context,
	solicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	analisis cobertura.SolicitudInstantaneaAnalisisDurableO3,
	instante time.Time,
) error {
	err := s.accesos.AutorizarPresentacionPropuestaCobertura(
		ctx,
		solicitud,
		contexto,
		analisis,
		instante,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if errors.Is(err, ErrPresentacionPropuestaCoberturaDenegada) {
		return ErrPresentacionPropuestaCoberturaDenegada
	}
	if err != nil {
		return ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return nil
}

func (s *ServicioPresentacionPropuestaCobertura) clasificarFalloContexto(
	ctx context.Context,
	causa error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(causa, ErrPresentacionPropuestaCoberturaDenegada) {
		return ErrPresentacionPropuestaCoberturaDenegada
	}
	return ErrPresentacionPropuestaCoberturaNoDisponible
}

func (s *ServicioPresentacionPropuestaCobertura) clasificarFalloDependencia(
	ctx context.Context,
	_ error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrPresentacionPropuestaCoberturaNoDisponible
}

package application

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const FinalidadConsultaInternaFuenteAutoridad = ports.FinalidadConsultaInternaFuenteAutoridad

var (
	ErrDependenciaConsultaInternaFuenteAutoridadRequerida = errors.New("vec: dependencia de consulta interna de fuente de autoridad requerida")
	ErrOrdenConsultaInternaFuenteAutoridadInvalida        = errors.New("vec: orden de consulta interna de fuente de autoridad invalida")
	ErrResultadoConsultaInternaFuenteAutoridadInvalido    = errors.New("vec: resultado de consulta interna de fuente de autoridad invalido")
	ErrSerializacionOrdenConsultaInternaAutoridad         = errors.New("vec: serializacion de orden interna de consulta de autoridad prohibida")
	ErrSerializacionResultadoConsultaInternaAutoridad     = errors.New("vec: serializacion de resultado interno de consulta de autoridad prohibida")
)

// ServicioConsultaInternaFuentesAutoridad gobierna la revelacion de una
// version exacta. La decision se obtiene antes de invocar el repositorio y se
// entrega a este para que sea consumida junto con el recibo de lectura.
type ServicioConsultaInternaFuentesAutoridad struct {
	consulta ports.ConsultaInternaGobernadaFuentesAutoridad
	exigidor ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	reloj    ports.Reloj
	politica PoliticaUsoDecisionAutorizacion
}

func NuevoServicioConsultaInternaFuentesAutoridad(
	consulta ports.ConsultaInternaGobernadaFuentesAutoridad,
	exigidor ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	reloj ports.Reloj,
) (*ServicioConsultaInternaFuentesAutoridad, error) {
	if dependenciaAutorizacionNula(consulta) || dependenciaAutorizacionNula(exigidor) ||
		dependenciaAutorizacionNula(reloj) {
		return nil, ErrDependenciaConsultaInternaFuenteAutoridadRequerida
	}
	politica, err := NuevaPoliticaUsoDecisionAutorizacion(
		ports.AccionConsultarFuenteAutoridadInterna,
		ports.ModuloFuentesAutoridad,
		ports.TipoRecursoFuenteAutoridad,
		FinalidadConsultaInternaFuenteAutoridad,
		[]string{ports.CampoConsultaInternaFuenteAutoridad},
		PerfilProteccionUsoAutorizacionInternoAlto,
	)
	if err != nil {
		return nil, errors.Join(ErrDependenciaConsultaInternaFuenteAutoridadRequerida, err)
	}
	return &ServicioConsultaInternaFuentesAutoridad{
		consulta: consulta, exigidor: exigidor, reloj: reloj, politica: politica,
	}, nil
}

// OrdenConsultaInternaExactaFuenteAutoridad contiene capacidades resueltas
// por la frontera interna. No admite roles, permisos, finalidad ni atributos
// de recurso declarados por el cliente.
type OrdenConsultaInternaExactaFuenteAutoridad struct {
	bloqueoSerializacionOrdenConsultaInternaAutoridad
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
	Selector                  ports.SelectorVersionFuenteAutoridad
	MotivoCatalogo            domain.ReferenciaEntradaCatalogo
	Correlacion               domain.ReferenciaCorrelacionAutorizacionV2
}

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (OrdenConsultaInternaExactaFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*OrdenConsultaInternaExactaFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (OrdenConsultaInternaExactaFuenteAutoridad) String() string {
	return "[ORDEN-CONSULTA-INTERNA-FUENTE-AUTORIDAD]"
}

func (o OrdenConsultaInternaExactaFuenteAutoridad) GoString() string { return o.String() }

func (o OrdenConsultaInternaExactaFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, o.String())
}

func (o OrdenConsultaInternaExactaFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(o.String())
}

type ResultadoConsultaInternaExactaFuenteAutoridad struct {
	bloqueoSerializacionResultadoConsultaInternaAutoridad
	Encontrada   bool
	Fuente       domain.FuenteAutoridadVersionada
	EstadoExacto ports.ReferenciaEstadoFuenteAutoridad
	Recibo       ports.ReciboConsultaInternaFuenteAutoridad
}

func (ResultadoConsultaInternaExactaFuenteAutoridad) String() string {
	return "[RESULTADO-CONSULTA-INTERNA-FUENTE-AUTORIDAD]"
}

func (r ResultadoConsultaInternaExactaFuenteAutoridad) GoString() string { return r.String() }

func (r ResultadoConsultaInternaExactaFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ResultadoConsultaInternaExactaFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r ResultadoConsultaInternaExactaFuenteAutoridad) Clonar() (
	ResultadoConsultaInternaExactaFuenteAutoridad,
	error,
) {
	datosRecibo, errRecibo := r.Recibo.Datos()
	if errRecibo != nil || r.Encontrada !=
		(datosRecibo.Resultado == ports.ResultadoConsultaFuenteEncontrada) ||
		r.EstadoExacto != datosRecibo.Estado {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
	}
	if !r.Encontrada {
		if !reflect.ValueOf(r.Fuente).IsZero() ||
			r.EstadoExacto != (ports.ReferenciaEstadoFuenteAutoridad{}) ||
			datosRecibo.Resultado != ports.ResultadoConsultaFuenteNoEncontrada {
			return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
		}
		return r, nil
	}
	fuente, err := r.Fuente.ClonarCanonica()
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
	}
	estado, err := ports.EstadoExactoFuenteAutoridad(fuente)
	if err != nil || estado != r.EstadoExacto ||
		datosRecibo.Selector.FuenteID != fuente.ID || datosRecibo.Selector.Version != fuente.Version {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
	}
	r.Fuente = fuente
	return r, nil
}

func (s *ServicioConsultaInternaFuentesAutoridad) ConsultarExacta(
	ctx context.Context,
	orden OrdenConsultaInternaExactaFuenteAutoridad,
) (ResultadoConsultaInternaExactaFuenteAutoridad, error) {
	if ctx == nil || s == nil || dependenciaAutorizacionNula(s.consulta) ||
		dependenciaAutorizacionNula(s.exigidor) || dependenciaAutorizacionNula(s.reloj) ||
		s.politica.validar() != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrDependenciaConsultaInternaFuenteAutoridadRequerida
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}
	actor, err := validarOrdenConsultaInternaFuenteAutoridad(orden)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}
	recurso, err := ports.RecursoAutorizableConsultaInternaFuenteAutoridad(
		orden.Selector, orden.MotivoCatalogo,
	)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrOrdenConsultaInternaFuenteAutoridadInvalida
	}

	evidencia, err := s.exigidor.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx,
		actor,
		orden.VinculoAutenticacionActor,
		recurso,
		orden.Correlacion,
		orden.MotivoCatalogo,
		s.politica,
	)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}
	solicitadaEn := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if solicitadaEn.IsZero() || evidencia.ValidarEn(solicitadaEn) != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			ports.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		)
	}
	datosAutorizacion, err := evidencia.Datos()
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	correlacionRef, err := orden.Correlacion.ValorCanonico()
	if err != nil || datosAutorizacion.Decision.CorrelacionRef != correlacionRef {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			ErrOrdenConsultaInternaFuenteAutoridadInvalida,
			err,
		)
	}
	auditoria := auditoriaSolicitudConsultaInternaFuenteAutoridad(
		actor,
		datosAutorizacion.Decision,
		recurso,
		orden,
		correlacionRef,
		solicitadaEn,
	)
	solicitud, err := ports.NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
		orden.Selector,
		evidencia,
		auditoria,
		orden.MotivoCatalogo,
		orden.Correlacion,
		solicitadaEn,
	)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, errors.Join(
			ErrOrdenConsultaInternaFuenteAutoridadInvalida,
			err,
		)
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}

	resultadoPuerto, err := s.consulta.ConsultarVersionExacta(ctx, solicitud)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, err
	}
	resultadoPuerto, err = resultadoPuerto.ClonarPara(solicitud)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, errors.Join(
			ErrResultadoConsultaInternaFuenteAutoridadInvalido,
			err,
		)
	}
	resultado := ResultadoConsultaInternaExactaFuenteAutoridad{
		Encontrada: resultadoPuerto.Encontrada,
		Recibo:     resultadoPuerto.Recibo,
	}
	if !resultadoPuerto.Encontrada {
		return resultado.Clonar()
	}
	fuente, err := resultadoPuerto.Fuente.ClonarCanonica()
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
	}
	estado, err := ports.EstadoExactoFuenteAutoridad(fuente)
	if err != nil {
		return ResultadoConsultaInternaExactaFuenteAutoridad{}, ErrResultadoConsultaInternaFuenteAutoridadInvalido
	}
	resultado.Fuente, resultado.EstadoExacto = fuente, estado
	return resultado.Clonar()
}

func validarOrdenConsultaInternaFuenteAutoridad(
	orden OrdenConsultaInternaExactaFuenteAutoridad,
) (domain.ContextoActor, error) {
	actor, err := orden.ContextoActor.Clonar()
	errCorrelacion := orden.Correlacion.Validar()
	if err != nil || orden.VinculoAutenticacionActor.ValidarPara(actor) != nil ||
		orden.Selector.Validar() != nil ||
		!ports.ReferenciaMotivoConsultaFuenteAutoridadValida(orden.MotivoCatalogo) ||
		errCorrelacion != nil {
		return domain.ContextoActor{}, errors.Join(
			ErrOrdenConsultaInternaFuenteAutoridadInvalida,
			err,
			errCorrelacion,
		)
	}
	return actor, nil
}

func auditoriaSolicitudConsultaInternaFuenteAutoridad(
	actor domain.ContextoActor,
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	orden OrdenConsultaInternaExactaFuenteAutoridad,
	correlacionRef string,
	instante time.Time,
) domain.AuditEntry {
	return domain.AuditEntry{
		ActorID:          actor.PersonaRef,
		ActorProfile:     actor.PerfilActivoRef,
		AuthMethod:       actor.Principal.AuthMethod,
		AuthAssurance:    actor.Principal.AuthAssurance,
		AuthorizationRef: decision.DecisionRef,
		Purpose:          decision.Finalidad,
		Action:           ports.AccionConsultarFuenteAutoridadInterna,
		ModuleID:         ports.ModuloFuentesAutoridad,
		SubjectRef:       recurso.Referencia,
		Reason:           orden.MotivoCatalogo.EntradaClave,
		CorrelationRef:   correlacionRef,
		OccurredAt:       instante,
		Metadata: map[string]string{
			"fuente_id": orden.Selector.FuenteID, "fuente_version": strconv.FormatUint(orden.Selector.Version, 10),
			ports.AtributoMotivoCatalogoIDConsultaAutoridad:      orden.MotivoCatalogo.CatalogoID,
			ports.AtributoMotivoCatalogoVersionConsultaAutoridad: strconv.Itoa(orden.MotivoCatalogo.CatalogoVersion),
			ports.AtributoMotivoCatalogoHuellaConsultaAutoridad:  orden.MotivoCatalogo.CatalogoHuellaSHA256,
			ports.AtributoMotivoEntradaClaveConsultaAutoridad:    orden.MotivoCatalogo.EntradaClave,
		},
	}
}

type bloqueoSerializacionOrdenConsultaInternaAutoridad struct{}

func (bloqueoSerializacionOrdenConsultaInternaAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*bloqueoSerializacionOrdenConsultaInternaAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (bloqueoSerializacionOrdenConsultaInternaAutoridad) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*bloqueoSerializacionOrdenConsultaInternaAutoridad) UnmarshalCBOR([]byte) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (bloqueoSerializacionOrdenConsultaInternaAutoridad) MarshalYAML() (any, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*bloqueoSerializacionOrdenConsultaInternaAutoridad) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (bloqueoSerializacionOrdenConsultaInternaAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*bloqueoSerializacionOrdenConsultaInternaAutoridad) GobDecode([]byte) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (bloqueoSerializacionOrdenConsultaInternaAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

func (*bloqueoSerializacionOrdenConsultaInternaAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionOrdenConsultaInternaAutoridad
}

type bloqueoSerializacionResultadoConsultaInternaAutoridad struct{}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalCBOR([]byte) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalYAML() (any, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) GobDecode([]byte) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (bloqueoSerializacionResultadoConsultaInternaAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

func (*bloqueoSerializacionResultadoConsultaInternaAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionResultadoConsultaInternaAutoridad
}

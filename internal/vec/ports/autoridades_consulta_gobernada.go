package ports

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
)

const (
	ModuloFuentesAutoridad                                                                = "vec"
	TipoRecursoFuenteAutoridad                                                            = "fuente_autoridad_versionada"
	AccionConsultarFuenteAutoridadInterna                                                 = "vec.fuentes_autoridad.consultar_interna"
	FinalidadConsultaInternaFuenteAutoridad                                               = "gobierno_fuentes_autoridad"
	CampoConsultaInternaFuenteAutoridad                                                   = "fuente_autoridad"
	AtributoMotivoCatalogoIDConsultaAutoridad                                             = "motivo_catalogo_id"
	AtributoMotivoCatalogoVersionConsultaAutoridad                                        = "motivo_catalogo_version"
	AtributoMotivoCatalogoHuellaConsultaAutoridad                                         = "motivo_catalogo_huella_sha256"
	AtributoMotivoEntradaClaveConsultaAutoridad                                           = "motivo_entrada_clave"
	ResultadoConsultaFuenteEncontrada                    ResultadoConsultaFuenteAutoridad = "encontrada"
	ResultadoConsultaFuenteNoEncontrada                  ResultadoConsultaFuenteAutoridad = "no_encontrada"
	maximoMetadatosAuditoriaConsultaAutoridad                                             = 6
	maximoClaveMetadatoAuditoriaConsultaAutoridad                                         = 128
	maximoValorMetadatoAuditoriaConsultaAutoridad                                         = 512
	maximoPresupuestoMetadatosAuditoriaConsultaAutoridad                                  = 4 * 1024
)

var (
	ErrConsultaInternaFuenteAutoridadInvalida = errors.New("vec: consulta interna gobernada de fuente de autoridad invalida")
	ErrReciboConsultaFuenteAutoridadInvalido  = errors.New("vec: recibo de consulta de fuente de autoridad invalido")
	ErrSerializacionGobiernoFuenteAutoridad   = errors.New("vec: serializacion de gobierno de fuente de autoridad prohibida")
)

// ReferenciaMotivoConsultaFuenteAutoridadValida restringe el identificador de
// la entrada a un valor opaco. La semantica y cualquier etiqueta legible viven
// exclusivamente en el catalogo gobernado, no en ordenes, decisiones ni logs.
func ReferenciaMotivoConsultaFuenteAutoridadValida(
	referencia domain.ReferenciaEntradaCatalogo,
) bool {
	return domain.ReferenciaMotivoAutorizacionV2Valida(referencia)
}

func RecursoAutorizableConsultaInternaFuenteAutoridad(
	selector SelectorVersionFuenteAutoridad,
	motivo domain.ReferenciaEntradaCatalogo,
) (domain.RecursoAutorizable, error) {
	if selector.Validar() != nil || !ReferenciaMotivoConsultaFuenteAutoridadValida(motivo) {
		return domain.RecursoAutorizable{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	referencia := "fuente:" + selector.FuenteID + ":v" + strconv.FormatUint(selector.Version, 10)
	recurso := domain.RecursoAutorizable{
		Referencia: referencia, ModuloID: ModuloFuentesAutoridad, Tipo: TipoRecursoFuenteAutoridad,
		Atributos: map[string]string{
			"fuente_id": selector.FuenteID, "fuente_version": strconv.FormatUint(selector.Version, 10),
			AtributoMotivoCatalogoIDConsultaAutoridad:      motivo.CatalogoID,
			AtributoMotivoCatalogoVersionConsultaAutoridad: strconv.Itoa(motivo.CatalogoVersion),
			AtributoMotivoCatalogoHuellaConsultaAutoridad:  motivo.CatalogoHuellaSHA256,
			AtributoMotivoEntradaClaveConsultaAutoridad:    motivo.EntradaClave,
		},
	}
	if recurso.Validar() != nil {
		return domain.RecursoAutorizable{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return recurso, nil
}

type datosSolicitudConsultaInternaFuenteAutoridad struct {
	selector       SelectorVersionFuenteAutoridad
	autorizacion   EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	auditoria      domain.AuditEntry
	motivoCatalogo domain.ReferenciaEntradaCatalogo
	correlacion    domain.ReferenciaCorrelacionAutorizacionV2
	solicitadaEn   time.Time
}

type SolicitudConsultaInternaGobernadaFuenteAutoridad struct {
	bloqueoSerializacionGobiernoFuenteAutoridad
	datos *datosSolicitudConsultaInternaFuenteAutoridad
}

func NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
	selector SelectorVersionFuenteAutoridad,
	autorizacion EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	auditoria domain.AuditEntry,
	motivoCatalogo domain.ReferenciaEntradaCatalogo,
	correlacion domain.ReferenciaCorrelacionAutorizacionV2,
	solicitadaEn time.Time,
) (SolicitudConsultaInternaGobernadaFuenteAutoridad, error) {
	if !prevalidarAuditoriaConsultaFuenteAutoridad(auditoria) {
		return SolicitudConsultaInternaGobernadaFuenteAutoridad{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	datos := datosSolicitudConsultaInternaFuenteAutoridad{
		selector: selector, autorizacion: autorizacion,
		auditoria: clonarAuditoriaFuenteAutoridad(auditoria), motivoCatalogo: motivoCatalogo,
		correlacion: correlacion, solicitadaEn: solicitadaEn,
	}
	if validarSolicitudConsultaInternaFuenteAutoridad(datos) != nil {
		return SolicitudConsultaInternaGobernadaFuenteAutoridad{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return SolicitudConsultaInternaGobernadaFuenteAutoridad{datos: &datos}, nil
}

// prevalidarAuditoriaConsultaFuenteAutoridad impone conteos y presupuesto
// antes de reservar la copia defensiva. La validacion exacta se repite sobre
// el clon dentro de validarSolicitudConsultaInternaFuenteAutoridad.
func prevalidarAuditoriaConsultaFuenteAutoridad(auditoria domain.AuditEntry) bool {
	if len(auditoria.ActorRoles) != 0 ||
		len(auditoria.Metadata) != maximoMetadatosAuditoriaConsultaAutoridad {
		return false
	}
	presupuesto := 0
	for clave, valor := range auditoria.Metadata {
		if len(clave) == 0 || len(clave) > maximoClaveMetadatoAuditoriaConsultaAutoridad ||
			len(valor) == 0 || len(valor) > maximoValorMetadatoAuditoriaConsultaAutoridad {
			return false
		}
		presupuesto += len(clave) + len(valor)
		if presupuesto > maximoPresupuestoMetadatosAuditoriaConsultaAutoridad {
			return false
		}
	}
	return true
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) MotivoCatalogo() (
	domain.ReferenciaEntradaCatalogo,
	error,
) {
	if s.validar() != nil {
		return domain.ReferenciaEntradaCatalogo{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.motivoCatalogo, nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) CorrelacionRef() (string, error) {
	if s.validar() != nil {
		return "", ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.correlacion.ValorCanonico()
}

// Correlacion conserva la capacidad nominal mientras la operacion permanece
// dentro del nucleo. CorrelacionRef revela el valor solo al adaptador durable.
func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Correlacion() (
	domain.ReferenciaCorrelacionAutorizacionV2,
	error,
) {
	if s.validar() != nil {
		return domain.ReferenciaCorrelacionAutorizacionV2{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.correlacion, nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Selector() (
	SelectorVersionFuenteAutoridad,
	error,
) {
	if s.validar() != nil {
		return SelectorVersionFuenteAutoridad{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.selector, nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Autorizacion() (
	EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
) {
	if s.validar() != nil {
		return EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.autorizacion, nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Auditoria() (domain.AuditEntry, error) {
	if s.validar() != nil {
		return domain.AuditEntry{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return clonarAuditoriaFuenteAutoridad(s.datos.auditoria), nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) SolicitadaEn() (time.Time, error) {
	if s.validar() != nil {
		return time.Time{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	return s.datos.solicitadaEn, nil
}

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) validar() error {
	if s.datos == nil {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	return validarSolicitudConsultaInternaFuenteAutoridad(*s.datos)
}

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) String() string {
	return "[SOLICITUD-CONSULTA-INTERNA-FUENTE-AUTORIDAD-OPACA]"
}
func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) GoString() string { return s.String() }
func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type ResultadoConsultaFuenteAutoridad string

func (r ResultadoConsultaFuenteAutoridad) Valido() bool {
	return r == ResultadoConsultaFuenteEncontrada || r == ResultadoConsultaFuenteNoEncontrada
}

type ResultadoConsultaInternaFuenteAutoridad struct {
	bloqueoSerializacionGobiernoFuenteAutoridad
	Encontrada bool
	Fuente     domain.FuenteAutoridadVersionada
	Estado     ReferenciaEstadoFuenteAutoridad
	Recibo     ReciboConsultaInternaFuenteAutoridad
}

func (r ResultadoConsultaInternaFuenteAutoridad) ValidarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) error {
	selector, err := solicitud.Selector()
	datosRecibo, errRecibo := r.Recibo.Datos()
	if err != nil || errRecibo != nil || r.Recibo.ValidarPara(solicitud) != nil ||
		r.Encontrada != (datosRecibo.Resultado == ResultadoConsultaFuenteEncontrada) ||
		r.Estado != datosRecibo.Estado {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	if !r.Encontrada {
		if !reflect.ValueOf(r.Fuente).IsZero() || r.Estado != (ReferenciaEstadoFuenteAutoridad{}) {
			return ErrConsultaInternaFuenteAutoridadInvalida
		}
		return nil
	}
	estado, errEstado := EstadoExactoFuenteAutoridad(r.Fuente)
	if errEstado != nil || r.Fuente.ID != selector.FuenteID || r.Fuente.Version != selector.Version ||
		r.Estado != estado {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	return nil
}

func (r ResultadoConsultaInternaFuenteAutoridad) ClonarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) (ResultadoConsultaInternaFuenteAutoridad, error) {
	if r.ValidarPara(solicitud) != nil {
		return ResultadoConsultaInternaFuenteAutoridad{}, ErrConsultaInternaFuenteAutoridadInvalida
	}
	if r.Encontrada {
		canonica, err := r.Fuente.ClonarCanonica()
		if err != nil {
			return ResultadoConsultaInternaFuenteAutoridad{}, ErrConsultaInternaFuenteAutoridadInvalida
		}
		r.Fuente = canonica
	}
	return r, nil
}

// ConsultaInternaGobernadaFuentesAutoridad es una barrera transaccional, no
// una lectura DAO. En una unica transaccion debe releer y validar la decision
// durable, consumirla exactamente una vez, ejecutar la lectura exacta, fijar
// el resultado de auditoria a encontrada o no_encontrada, encadenar y firmar
// esa auditoria y emitir el recibo. Resultado y Recibo solo se construyen y
// devuelven despues del COMMIT. La ausencia nunca se devuelve como
// ErrFuenteAutoridadNoEncontrada ni sigue un camino sin consumo y auditoria.
type ConsultaInternaGobernadaFuentesAutoridad interface {
	ConsultarVersionExacta(
		context.Context,
		SolicitudConsultaInternaGobernadaFuenteAutoridad,
	) (ResultadoConsultaInternaFuenteAutoridad, error)
}

func validarSolicitudConsultaInternaFuenteAutoridad(
	datos datosSolicitudConsultaInternaFuenteAutoridad,
) error {
	recurso, errRecurso := RecursoAutorizableConsultaInternaFuenteAutoridad(
		datos.selector, datos.motivoCatalogo,
	)
	datosAutorizacion, errAutorizacion := datos.autorizacion.Datos()
	correlacionRef, errCorrelacion := datos.correlacion.ValorCanonico()
	if errRecurso != nil || errAutorizacion != nil ||
		!ReferenciaMotivoConsultaFuenteAutoridadValida(datos.motivoCatalogo) ||
		errCorrelacion != nil ||
		!instantePuertoAutoridadCanonico(datos.solicitadaEn) ||
		datos.autorizacion.ValidarMotivo(datos.motivoCatalogo) != nil ||
		validarUsoAutorizacionFuenteAutoridad(
			datos.autorizacion, AccionConsultarFuenteAutoridadInterna,
			recurso, []string{CampoConsultaInternaFuenteAutoridad}, correlacionRef, datos.solicitadaEn,
		) != nil || validarAuditoriaConsultaFuenteAutoridad(
		datos.auditoria, datosAutorizacion.Decision, recurso, datos.motivoCatalogo,
		correlacionRef, datos.solicitadaEn,
	) != nil {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	return nil
}

func validarUsoAutorizacionFuenteAutoridad(
	evidencia EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	accion string,
	recurso domain.RecursoAutorizable,
	campos []string,
	correlacionRef string,
	instante time.Time,
) error {
	datos, err := evidencia.Datos()
	huellaContexto, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	decision := datos.Decision
	datosVinculo, errVinculo := decision.VinculoAutenticacionActor.Datos()
	superficieInterna := errVinculo == nil && (datosVinculo.Superficie ==
		domain.SuperficieAutenticacionInternaCorporativaV1 || datosVinculo.Superficie ==
		domain.SuperficieAutenticacionAdministracionPrivilegiadaV1)
	if err != nil || errHuella != nil || errVinculo != nil || evidencia.ValidarEn(instante) != nil ||
		decision.Accion != accion || decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != recurso.ModuloID || decision.TipoRecurso != recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != FinalidadConsultaInternaFuenteAutoridad ||
		decision.CorrelacionRef != correlacionRef ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(correlacionRef) ||
		!superficieInterna || datosVinculo.GarantiaObservada != domain.AuthAssuranceHigh ||
		decision.GarantiaMinima != domain.AuthAssuranceHigh || len(decision.Obligaciones) != 0 ||
		!cadenasAutoridadIguales(decision.CamposPermitidos, campos) {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	return nil
}

func validarAuditoriaConsultaFuenteAutoridad(
	auditoria domain.AuditEntry,
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	motivoCatalogo domain.ReferenciaEntradaCatalogo,
	correlacionRef string,
	instante time.Time,
) error {
	datosVinculo, err := decision.VinculoAutenticacionActor.Datos()
	metadatosEsperados := map[string]string{
		"fuente_id": recurso.Atributos["fuente_id"], "fuente_version": recurso.Atributos["fuente_version"],
		AtributoMotivoCatalogoIDConsultaAutoridad:      recurso.Atributos[AtributoMotivoCatalogoIDConsultaAutoridad],
		AtributoMotivoCatalogoVersionConsultaAutoridad: recurso.Atributos[AtributoMotivoCatalogoVersionConsultaAutoridad],
		AtributoMotivoCatalogoHuellaConsultaAutoridad:  recurso.Atributos[AtributoMotivoCatalogoHuellaConsultaAutoridad],
		AtributoMotivoEntradaClaveConsultaAutoridad:    recurso.Atributos[AtributoMotivoEntradaClaveConsultaAutoridad],
	}
	if err != nil || !ReferenciaMotivoConsultaFuenteAutoridadValida(motivoCatalogo) ||
		auditoria.ID != "" || auditoria.Seq != 0 || auditoria.Signature != "" ||
		auditoria.PrevSignature != "" || auditoria.IntegrityAlgorithm != "" ||
		len(auditoria.ActorRoles) != 0 ||
		auditoria.RepresentedSubjectID != "" || auditoria.ExpedienteRef != "" ||
		auditoria.DocumentRef != "" || auditoria.RuleRef != "" || auditoria.Reason != motivoCatalogo.EntradaClave ||
		auditoria.ActorID != decision.PrincipalID || auditoria.ActorProfile != decision.PerfilActivoRef ||
		auditoria.AuthMethod != datosVinculo.MetodoObservado ||
		auditoria.AuthAssurance != datosVinculo.GarantiaObservada ||
		auditoria.AuthorizationRef != decision.DecisionRef ||
		auditoria.Purpose != FinalidadConsultaInternaFuenteAutoridad ||
		auditoria.Action != decision.Accion || auditoria.ModuleID != recurso.ModuloID ||
		auditoria.SubjectRef != recurso.Referencia || auditoria.ObjectVersion != 0 ||
		auditoria.CorrelationRef != correlacionRef || decision.CorrelacionRef != correlacionRef ||
		auditoria.Result != "" ||
		auditoria.BeforeHash != "" || auditoria.AfterHash != "" || !auditoria.OccurredAt.Equal(instante) ||
		!mapaCadenasAutoridadIgual(auditoria.Metadata, metadatosEsperados) {
		return ErrConsultaInternaFuenteAutoridadInvalida
	}
	return nil
}

func clonarAuditoriaFuenteAutoridad(auditoria domain.AuditEntry) domain.AuditEntry {
	clon := auditoria
	clon.ActorRoles = append([]string(nil), auditoria.ActorRoles...)
	if auditoria.Metadata != nil {
		clon.Metadata = make(map[string]string, len(auditoria.Metadata))
		for clave, valor := range auditoria.Metadata {
			clon.Metadata[clave] = valor
		}
	}
	return clon
}

func cadenasAutoridadIguales(primera, segunda []string) bool {
	if len(primera) != len(segunda) {
		return false
	}
	for indice := range primera {
		if primera[indice] != segunda[indice] {
			return false
		}
	}
	return true
}

func mapaCadenasAutoridadIgual(primero, segundo map[string]string) bool {
	if len(primero) != len(segundo) {
		return false
	}
	for clave, valor := range primero {
		if segundo[clave] != valor {
			return false
		}
	}
	return true
}

type bloqueoSerializacionGobiernoFuenteAutoridad struct{}

func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalCBOR([]byte) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalYAML() (any, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) GobDecode([]byte) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (*bloqueoSerializacionGobiernoFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionGobiernoFuenteAutoridad
}
func (bloqueoSerializacionGobiernoFuenteAutoridad) String() string {
	return "[VALOR-GOBIERNO-FUENTE-AUTORIDAD-INTERNO]"
}
func (b bloqueoSerializacionGobiernoFuenteAutoridad) GoString() string { return b.String() }
func (b bloqueoSerializacionGobiernoFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionGobiernoFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

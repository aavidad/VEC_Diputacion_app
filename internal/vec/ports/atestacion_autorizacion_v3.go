package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrSerializacionAtestacionAutorizacionV3Prohibida = errors.New(
	"vec: serializacion generica de atestacion de autorizacion V3 prohibida",
)

// SolicitudFirmaAtestacionAutorizacionV3 conserva la preimagen VEC-AD-3 y solo
// expone compromisos minimizados. No constituye una capacidad de efecto.
type SolicitudFirmaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionAtestacionAutorizacionV3
	cabecera             domain.CabeceraAtestacionAutorizacionV3
	mensaje              []byte
	huellaMensaje        string
	referenciaDecision   string
	huellaDecision       string
	huellaMotivoCatalogo string
	referenciaContexto   string
	huellaContexto       string
	huellaCompromisos    string
}

type resumenDecisionAtestacionAutorizacionV3 struct {
	DecisionRef string `json:"decision_ref"`
}

type compromisosSolicitudAtestacionAutorizacionV3 struct {
	FormatoVersion       uint16 `json:"formato_version"`
	Suite                string `json:"suite"`
	ClaveID              string `json:"clave_id"`
	Audiencia            string `json:"audiencia"`
	HuellaMensaje        string `json:"huella_mensaje_sha256"`
	ReferenciaDecision   string `json:"decision_ref"`
	HuellaDecision       string `json:"decision_huella_sha256"`
	HuellaMotivoCatalogo string `json:"motivo_huella_sha256"`
	ReferenciaContexto   string `json:"contexto_ref"`
	HuellaContexto       string `json:"contexto_huella_sha256"`
}

func NuevaSolicitudFirmaAtestacionAutorizacionV3(
	cabecera domain.CabeceraAtestacionAutorizacionV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (SolicitudFirmaAtestacionAutorizacionV3, error) {
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		referenciaMotivo,
		resultadoContexto,
	)
	if err != nil {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	decisionCanonica, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	var resumen resumenDecisionAtestacionAutorizacionV3
	if err := json.Unmarshal(decisionCanonica, &resumen); err != nil ||
		!referenciaAtestacionValida(resumen.DecisionRef) {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	huellaDecision, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaMotivo, errMotivo := domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || errMotivo != nil || resultadoContexto.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(mensaje)
	solicitud := SolicitudFirmaAtestacionAutorizacionV3{
		cabecera: cabecera, mensaje: append([]byte(nil), mensaje...),
		huellaMensaje:      hex.EncodeToString(suma[:]),
		referenciaDecision: resumen.DecisionRef, huellaDecision: huellaDecision,
		huellaMotivoCatalogo: huellaMotivo,
		referenciaContexto:   resultadoContexto.RegistroContextoRef,
		huellaContexto:       resultadoContexto.HuellaSHA256,
	}
	solicitud.huellaCompromisos = solicitud.calcularHuellaCompromisos()
	if solicitud.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	return solicitud, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) Validar() error {
	if s.cabecera.Validar() != nil ||
		len(s.mensaje) == 0 ||
		len(s.mensaje) > domain.TamanoMaximoMensajeAtestacionAutorizacionV3 ||
		!huellaAtestacionValida(s.huellaMensaje) ||
		!referenciaAtestacionValida(s.referenciaDecision) ||
		!huellaAtestacionValida(s.huellaDecision) ||
		!huellaAtestacionValida(s.huellaMotivoCatalogo) ||
		!referenciaAtestacionValida(s.referenciaContexto) ||
		!huellaAtestacionValida(s.huellaContexto) ||
		!huellaAtestacionValida(s.huellaCompromisos) {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(s.mensaje)
	esperada, err := hex.DecodeString(s.huellaMensaje)
	if err != nil || subtle.ConstantTimeCompare(suma[:], esperada) != 1 {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	compromisos, err := hex.DecodeString(s.calcularHuellaCompromisos())
	esperados, errEsperados := hex.DecodeString(s.huellaCompromisos)
	if err != nil || errEsperados != nil ||
		subtle.ConstantTimeCompare(compromisos, esperados) != 1 {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	return nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) calcularHuellaCompromisos() string {
	contenido, err := json.Marshal(compromisosSolicitudAtestacionAutorizacionV3{
		FormatoVersion: s.cabecera.FormatoVersion,
		Suite:          s.cabecera.Suite, ClaveID: s.cabecera.ClaveID,
		Audiencia: s.cabecera.Audiencia, HuellaMensaje: s.huellaMensaje,
		ReferenciaDecision: s.referenciaDecision, HuellaDecision: s.huellaDecision,
		HuellaMotivoCatalogo: s.huellaMotivoCatalogo,
		ReferenciaContexto:   s.referenciaContexto,
		HuellaContexto:       s.huellaContexto,
	})
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func (s SolicitudFirmaAtestacionAutorizacionV3) Cabecera() (
	domain.CabeceraAtestacionAutorizacionV3,
	error,
) {
	if s.Validar() != nil {
		return domain.CabeceraAtestacionAutorizacionV3{},
			ErrSolicitudFirmaAtestacionInvalida
	}
	return s.cabecera, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudFirmaAtestacionInvalida
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) HuellaMensajeSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaMensaje, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) ReferenciaDecision() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.referenciaDecision, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) HuellaDecisionSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaDecision, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) HuellaMotivoCatalogoSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaMotivoCatalogo, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) ReferenciaContextoActor() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.referenciaContexto, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV3) HuellaContextoActorSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaContexto, nil
}

// ResultadoFirmaAtestacionAutorizacionV3 liga una firma opaca a una única
// solicitud. El adaptador privado de confianza debe verificar el perfil
// criptográfico antes de que la firma alcance una transacción de consumo.
type ResultadoFirmaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionAtestacionAutorizacionV3
	firma                 []byte
	huellaMensaje         string
	evidenciaOperacionRef string
	firmadaEn             time.Time
}

func NuevoResultadoFirmaAtestacionAutorizacionV3(
	solicitud SolicitudFirmaAtestacionAutorizacionV3,
	firma []byte,
	evidenciaOperacionRef string,
	firmadaEn time.Time,
) (ResultadoFirmaAtestacionAutorizacionV3, error) {
	if solicitud.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV3{},
			ErrResultadoFirmaAtestacionInvalido
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	resultado := ResultadoFirmaAtestacionAutorizacionV3{
		firma: append([]byte(nil), firma...), huellaMensaje: huella,
		evidenciaOperacionRef: evidenciaOperacionRef, firmadaEn: firmadaEn,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ResultadoFirmaAtestacionAutorizacionV3{},
			ErrResultadoFirmaAtestacionInvalido
	}
	return resultado, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) Validar() error {
	if len(r.firma) == 0 || len(r.firma) > tamanoMaximoFirmaAtestacion ||
		!huellaAtestacionValida(r.huellaMensaje) ||
		!referenciaAtestacionValida(r.evidenciaOperacionRef) ||
		!instanteAtestacionCanonico(r.firmadaEn) {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV3,
) error {
	if r.Validar() != nil || solicitud.Validar() != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	if !huellasAtestacionIguales(huella, r.huellaMensaje) {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) Firma() ([]byte, error) {
	if r.Validar() != nil {
		return nil, ErrResultadoFirmaAtestacionInvalido
	}
	return append([]byte(nil), r.firma...), nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) HuellaMensajeSHA256() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.huellaMensaje, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) EvidenciaOperacionRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.evidenciaOperacionRef, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV3) FirmadaEn() (time.Time, error) {
	if r.Validar() != nil {
		return time.Time{}, ErrResultadoFirmaAtestacionInvalido
	}
	return r.firmadaEn, nil
}

type AtestacionAutorizacionV3 struct {
	bloqueoSerializacionAtestacionAutorizacionV3
	solicitud SolicitudFirmaAtestacionAutorizacionV3
	resultado ResultadoFirmaAtestacionAutorizacionV3
}

func NuevaAtestacionAutorizacionV3(
	solicitud SolicitudFirmaAtestacionAutorizacionV3,
	resultado ResultadoFirmaAtestacionAutorizacionV3,
) (AtestacionAutorizacionV3, error) {
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil {
		return AtestacionAutorizacionV3{}, ErrResultadoFirmaAtestacionInvalido
	}
	return AtestacionAutorizacionV3{solicitud: solicitud, resultado: resultado}, nil
}

func (a AtestacionAutorizacionV3) Validar() error {
	if a.solicitud.Validar() != nil || a.resultado.ValidarPara(a.solicitud) != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (a AtestacionAutorizacionV3) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV3,
) error {
	if a.Validar() != nil || solicitud.Validar() != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	propia, _ := a.solicitud.HuellaMensajeSHA256()
	esperada, _ := solicitud.HuellaMensajeSHA256()
	if !huellasAtestacionIguales(propia, esperada) ||
		a.resultado.ValidarPara(solicitud) != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (a AtestacionAutorizacionV3) Solicitud() (
	SolicitudFirmaAtestacionAutorizacionV3,
	error,
) {
	if a.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV3{},
			ErrResultadoFirmaAtestacionInvalido
	}
	solicitud := a.solicitud
	solicitud.mensaje = append([]byte(nil), a.solicitud.mensaje...)
	return solicitud, nil
}

func (a AtestacionAutorizacionV3) Resultado() (
	ResultadoFirmaAtestacionAutorizacionV3,
	error,
) {
	if a.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV3{},
			ErrResultadoFirmaAtestacionInvalido
	}
	resultado := a.resultado
	resultado.firma = append([]byte(nil), a.resultado.firma...)
	return resultado, nil
}

type FirmanteAtestacionesAutorizacionV3 interface {
	FirmarAtestacionAutorizacionV3(
		context.Context,
		SolicitudFirmaAtestacionAutorizacionV3,
	) (ResultadoFirmaAtestacionAutorizacionV3, error)
}

type bloqueoSerializacionAtestacionAutorizacionV3 struct{}

func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) GobDecode([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionAtestacionAutorizacionV3) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionAtestacionAutorizacionV3) String() string {
	return "[ATESTACION-AUTORIZACION-V3-REDACTADA-NO-AUTORITATIVA]"
}
func (b bloqueoSerializacionAtestacionAutorizacionV3) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionAtestacionAutorizacionV3) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionAtestacionAutorizacionV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

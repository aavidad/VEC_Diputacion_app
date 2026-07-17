package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrSerializacionAtestacionAutorizacionV2Prohibida = errors.New(
	"vec: serializacion generica de atestacion de autorizacion V2 prohibida",
)

// SolicitudFirmaAtestacionAutorizacionV2 conserva exactamente el mensaje
// VEC-AD-2 que debe recibir el firmante. Es un contrato nominal: ni construir
// la solicitud ni obtener sus datos concede autoridad para ejecutar un efecto.
type SolicitudFirmaAtestacionAutorizacionV2 struct {
	bloqueoSerializacionAtestacionAutorizacionV2
	cabecera              domain.CabeceraAtestacionAutorizacionV2
	mensaje               []byte
	huellaMensaje         string
	referenciaDecision    string
	huellaSolicitudLigada string
	huellaMotivoCatalogo  string
}

func NuevaSolicitudFirmaAtestacionAutorizacionV2(
	cabecera domain.CabeceraAtestacionAutorizacionV2,
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) (SolicitudFirmaAtestacionAutorizacionV2, error) {
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil {
		return SolicitudFirmaAtestacionAutorizacionV2{}, ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(mensaje)
	solicitud := SolicitudFirmaAtestacionAutorizacionV2{
		cabecera:              cabecera,
		mensaje:               append([]byte(nil), mensaje...),
		huellaMensaje:         hex.EncodeToString(suma[:]),
		referenciaDecision:    decision.DecisionRef,
		huellaSolicitudLigada: decision.SolicitudHuellaSHA256,
		huellaMotivoCatalogo:  decision.MotivoHuellaSHA256,
	}
	if solicitud.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV2{}, ErrSolicitudFirmaAtestacionInvalida
	}
	return solicitud, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) Validar() error {
	if s.cabecera.Validar() != nil ||
		!referenciaAtestacionValida(s.referenciaDecision) ||
		!huellaAtestacionValida(s.huellaMensaje) ||
		!huellaAtestacionValida(s.huellaSolicitudLigada) ||
		!huellaAtestacionValida(s.huellaMotivoCatalogo) ||
		len(s.mensaje) == 0 ||
		len(s.mensaje) > domain.TamanoMaximoMensajeAtestacionAutorizacionV2 {
		return ErrSolicitudFirmaAtestacionInvalida
	}

	proyeccion, err := domain.ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(s.mensaje)
	if err != nil {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	cabecera, errCabecera := proyeccion.Cabecera()
	referencia, errReferencia := proyeccion.DecisionRef()
	huellaSolicitud, errSolicitud := proyeccion.SolicitudHuellaSHA256()
	huellaMotivo, errMotivo := proyeccion.MotivoHuellaSHA256()
	if errCabecera != nil || errReferencia != nil || errSolicitud != nil || errMotivo != nil ||
		cabecera != s.cabecera || referencia != s.referenciaDecision ||
		!huellasAtestacionIguales(huellaSolicitud, s.huellaSolicitudLigada) ||
		!huellasAtestacionIguales(huellaMotivo, s.huellaMotivoCatalogo) {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(s.mensaje)
	esperada, err := hex.DecodeString(s.huellaMensaje)
	if err != nil || subtle.ConstantTimeCompare(suma[:], esperada) != 1 {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	return nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) Cabecera() (
	domain.CabeceraAtestacionAutorizacionV2,
	error,
) {
	if s.Validar() != nil {
		return domain.CabeceraAtestacionAutorizacionV2{}, ErrSolicitudFirmaAtestacionInvalida
	}
	return s.cabecera, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudFirmaAtestacionInvalida
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaMensajeSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaMensaje, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) ReferenciaDecision() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.referenciaDecision, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaSolicitudLigadaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaSolicitudLigada, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaMotivoCatalogoSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaMotivoCatalogo, nil
}

// ResultadoFirmaAtestacionAutorizacionV2 conserva la salida opaca del
// proveedor y la liga a una unica solicitud VEC-AD-2 mediante su huella.
// Verificar el perfil criptografico sigue siendo responsabilidad del adaptador
// privado de confianza que consuma esta salida.
type ResultadoFirmaAtestacionAutorizacionV2 struct {
	bloqueoSerializacionAtestacionAutorizacionV2
	firma                 []byte
	huellaMensaje         string
	evidenciaOperacionRef string
	firmadaEn             time.Time
}

func NuevoResultadoFirmaAtestacionAutorizacionV2(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
	firma []byte,
	evidenciaOperacionRef string,
	firmadaEn time.Time,
) (ResultadoFirmaAtestacionAutorizacionV2, error) {
	if solicitud.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	resultado := ResultadoFirmaAtestacionAutorizacionV2{
		firma:                 append([]byte(nil), firma...),
		huellaMensaje:         huella,
		evidenciaOperacionRef: evidenciaOperacionRef,
		firmadaEn:             firmadaEn,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ResultadoFirmaAtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	return resultado, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV2) Validar() error {
	if len(r.firma) == 0 || len(r.firma) > tamanoMaximoFirmaAtestacion ||
		!huellaAtestacionValida(r.huellaMensaje) ||
		!referenciaAtestacionValida(r.evidenciaOperacionRef) ||
		!instanteAtestacionCanonico(r.firmadaEn) {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (r ResultadoFirmaAtestacionAutorizacionV2) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
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

func (r ResultadoFirmaAtestacionAutorizacionV2) Firma() ([]byte, error) {
	if r.Validar() != nil {
		return nil, ErrResultadoFirmaAtestacionInvalido
	}
	return append([]byte(nil), r.firma...), nil
}

func (r ResultadoFirmaAtestacionAutorizacionV2) HuellaMensajeSHA256() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.huellaMensaje, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV2) EvidenciaOperacionRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.evidenciaOperacionRef, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV2) FirmadaEn() (time.Time, error) {
	if r.Validar() != nil {
		return time.Time{}, ErrResultadoFirmaAtestacionInvalido
	}
	return r.firmadaEn, nil
}

// AtestacionAutorizacionV2 conserva juntos el mensaje exacto y la salida del
// firmante. Sigue siendo evidencia nominal hasta superar perfil de confianza,
// vigencia, revocacion, revalidacion y consumo unico en la transaccion final.
type AtestacionAutorizacionV2 struct {
	bloqueoSerializacionAtestacionAutorizacionV2
	solicitud SolicitudFirmaAtestacionAutorizacionV2
	resultado ResultadoFirmaAtestacionAutorizacionV2
}

func NuevaAtestacionAutorizacionV2(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
	resultado ResultadoFirmaAtestacionAutorizacionV2,
) (AtestacionAutorizacionV2, error) {
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil {
		return AtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	atestacion := AtestacionAutorizacionV2{solicitud: solicitud, resultado: resultado}
	if atestacion.Validar() != nil {
		return AtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	return atestacion, nil
}

func (a AtestacionAutorizacionV2) Validar() error {
	if a.solicitud.Validar() != nil || a.resultado.ValidarPara(a.solicitud) != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (a AtestacionAutorizacionV2) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
) error {
	if a.Validar() != nil || solicitud.Validar() != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	huellaPropia, _ := a.solicitud.HuellaMensajeSHA256()
	huellaEsperada, _ := solicitud.HuellaMensajeSHA256()
	if !huellasAtestacionIguales(huellaPropia, huellaEsperada) ||
		a.resultado.ValidarPara(solicitud) != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (a AtestacionAutorizacionV2) Solicitud() (SolicitudFirmaAtestacionAutorizacionV2, error) {
	if a.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	solicitud := a.solicitud
	solicitud.mensaje = append([]byte(nil), a.solicitud.mensaje...)
	return solicitud, nil
}

func (a AtestacionAutorizacionV2) Resultado() (ResultadoFirmaAtestacionAutorizacionV2, error) {
	if a.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV2{}, ErrResultadoFirmaAtestacionInvalido
	}
	resultado := a.resultado
	resultado.firma = append([]byte(nil), a.resultado.firma...)
	return resultado, nil
}

// FirmanteAtestacionesAutorizacionV2 es un puerto deliberadamente distinto de
// V1. La implementacion productiva debe usar la identidad exclusiva del PDP y
// una clave no exportable aprobada para el perfil VEC-AD-2.
type FirmanteAtestacionesAutorizacionV2 interface {
	FirmarAtestacionAutorizacionV2(
		context.Context,
		SolicitudFirmaAtestacionAutorizacionV2,
	) (ResultadoFirmaAtestacionAutorizacionV2, error)
}

func huellasAtestacionIguales(primera, segunda string) bool {
	if !huellaAtestacionValida(primera) || !huellaAtestacionValida(segunda) {
		return false
	}
	primeraBinaria, errPrimera := hex.DecodeString(primera)
	segundaBinaria, errSegunda := hex.DecodeString(segunda)
	return errPrimera == nil && errSegunda == nil &&
		subtle.ConstantTimeCompare(primeraBinaria, segundaBinaria) == 1
}

// bloqueoSerializacionAtestacionAutorizacionV2 evita que mensaje, firma o
// referencias internas terminen accidentalmente en un codec o un log. La
// persistencia productiva debe extraer cada dato por los metodos validados y
// aplicar el esquema explicito del adaptador correspondiente.
type bloqueoSerializacionAtestacionAutorizacionV2 struct{}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalText([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) GobDecode([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (*bloqueoSerializacionAtestacionAutorizacionV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionAtestacionAutorizacionV2Prohibida
}

func (bloqueoSerializacionAtestacionAutorizacionV2) String() string {
	return "[ATESTACION-AUTORIZACION-V2-REDACTADA-NO-AUTORITATIVA]"
}

func (b bloqueoSerializacionAtestacionAutorizacionV2) GoString() string { return b.String() }

func (b bloqueoSerializacionAtestacionAutorizacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacionAtestacionAutorizacionV2) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

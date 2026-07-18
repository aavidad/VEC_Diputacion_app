package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

const DominioCriptograficoMotivoGobiernoConvocatoriaV1 = "bolsa.convocatoria.motivo.v1"

const VigenciaMaximaAtestacionMotivoGobiernoConvocatoria = 5 * time.Minute

var (
	ErrSelladoMotivoGobiernoConvocatoriaInvalido = errors.New("bolsa: sellado HMAC de motivo de convocatoria invalido")
	ErrSerializacionMotivoGobiernoConvocatoria   = errors.New("bolsa: serializacion de motivo de convocatoria prohibida")
)

// SolicitudSellarMotivoGobiernoConvocatoria es una orden interna. El motivo
// en claro solo cruza este puerto hacia un servicio de claves; nunca se guarda
// en idempotencia ni se registra en trazas.
type SolicitudSellarMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico string
	Accion               string
	ConvocatoriaRef      string
	PrincipalRef         string
	CorrelacionRef       string
	Motivo               string
	SolicitadaEn         time.Time
}

func (s SolicitudSellarMotivoGobiernoConvocatoria) Validar() error {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[s.Accion]
	if s.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		!conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(s.ConvocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(s.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(s.CorrelacionRef) ||
		!textoMotivoGobiernoConvocatoriaValido(s.Motivo) ||
		!instanteGobiernoConvocatoriaCanonico(s.SolicitadaEn) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

// HuellaSHA256 fija la preimagen que el sellador debe autenticar. El HSM/KMS
// calcula su HMAC sobre dominio || 0x00 || esta huella, nunca solo sobre el
// motivo. Asi quedan ligados accion, version, principal y correlacion.
func (s SolicitudSellarMotivoGobiernoConvocatoria) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return huellaSemanticaMotivoGobiernoConvocatoria(
		s.Accion, s.ConvocatoriaRef, s.PrincipalRef, s.CorrelacionRef, s.Motivo,
	)
}

// huellaSemanticaMotivoGobiernoConvocatoria es la unica representacion de la
// solicitud que liga el motivo en claro con la transicion administrativa. No
// incluye el instante: un reintento de la misma intencion conserva su huella.
func huellaSemanticaMotivoGobiernoConvocatoria(
	accion, convocatoriaRef, principalRef, correlacionRef, motivo string,
) (string, error) {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[accion]
	if !conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(convocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(principalRef) ||
		!referenciaGobiernoConvocatoriaValida(correlacionRef) ||
		!textoMotivoGobiernoConvocatoriaValido(motivo) {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	preimagen := struct {
		Esquema         string `json:"esquema"`
		Dominio         string `json:"dominio"`
		Accion          string `json:"accion"`
		ConvocatoriaRef string `json:"convocatoria_ref"`
		PrincipalRef    string `json:"principal_ref"`
		CorrelacionRef  string `json:"correlacion_ref"`
		Motivo          string `json:"motivo"`
	}{
		"bolsa.convocatoria.solicitud-sellado-motivo.v1",
		DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		accion, convocatoriaRef, principalRef, correlacionRef, motivo,
	}
	contenido, err := json.Marshal(preimagen)
	if err != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSellarMotivoGobiernoConvocatoria) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSellarMotivoGobiernoConvocatoria) String() string {
	return "[SOLICITUD-SELLADO-MOTIVO-CONVOCATORIA-INTERNA]"
}
func (s SolicitudSellarMotivoGobiernoConvocatoria) GoString() string { return s.String() }
func (s SolicitudSellarMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudSellarMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// HMACMotivoGobiernoConvocatoria identifica dominio y generacion de clave. No
// contiene la clave ni permite calcular otro HMAC.
type HMACMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico string
	GeneracionClave      uint32
	ClaveHMACRef         string
	ValorHMACSHA256      string
}

func (h HMACMotivoGobiernoConvocatoria) Validar() error {
	if h.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		h.GeneracionClave < 1 || !claveValida(h.ClaveHMACRef) ||
		!huellaGobiernoConvocatoriaValida(h.ValorHMACSHA256) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (h HMACMotivoGobiernoConvocatoria) representacionMaterial() (string, error) {
	if h.Validar() != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return "hmac-sha256:" + h.ClaveHMACRef + ":" + h.ValorHMACSHA256, nil
}

func (HMACMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (HMACMotivoGobiernoConvocatoria) String() string     { return "[HMAC-MOTIVO-CONVOCATORIA-OCULTO]" }
func (h HMACMotivoGobiernoConvocatoria) GoString() string { return h.String() }
func (h HMACMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}
func (h HMACMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(h.String())
}

type DatosAtestacionSelladoMotivoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	HMAC                   HMACMotivoGobiernoConvocatoria
	Accion                 string
	ConvocatoriaRef        string
	PrincipalRef           string
	CorrelacionRef         string
	HuellaSolicitudSHA256  string
	SelladorRef            string
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	TokenConsumoRef        string
	AtestacionEmitidaEn    time.Time
	AtestacionValidaHasta  time.Time
}

// AtestacionSelladoMotivoConvocatoria es reconstruible y no concede autoridad.
// La barrera durable debe releerla desde el registro del HSM/KMS, comprobar su
// huella y consumir TokenConsumoRef en la misma transaccion que la mutacion.
type AtestacionSelladoMotivoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosAtestacionSelladoMotivoConvocatoria
}

func NuevaAtestacionSelladoMotivoConvocatoria(
	solicitud SolicitudSellarMotivoGobiernoConvocatoria,
	datos DatosAtestacionSelladoMotivoConvocatoria,
) (AtestacionSelladoMotivoConvocatoria, error) {
	huella, err := solicitud.HuellaSHA256()
	if err != nil || validarDatosAtestacionSelladoMotivo(datos) != nil ||
		datos.Accion != solicitud.Accion || datos.ConvocatoriaRef != solicitud.ConvocatoriaRef ||
		datos.PrincipalRef != solicitud.PrincipalRef || datos.CorrelacionRef != solicitud.CorrelacionRef ||
		datos.HuellaSolicitudSHA256 != huella || datos.HMAC.DominioCriptografico != solicitud.DominioCriptografico ||
		datos.AtestacionEmitidaEn.Before(solicitud.SolicitadaEn) {
		return AtestacionSelladoMotivoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	copia := datos
	return AtestacionSelladoMotivoConvocatoria{datos: &copia}, nil
}

func (a AtestacionSelladoMotivoConvocatoria) DatosParaConsumo() (
	DatosAtestacionSelladoMotivoConvocatoria,
	error,
) {
	if a.datos == nil || validarDatosAtestacionSelladoMotivo(*a.datos) != nil {
		return DatosAtestacionSelladoMotivoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return *a.datos, nil
}

func (a AtestacionSelladoMotivoConvocatoria) material() (
	HMACMotivoGobiernoConvocatoria,
	string,
	error,
) {
	datos, err := a.DatosParaConsumo()
	if err != nil {
		return HMACMotivoGobiernoConvocatoria{}, "", err
	}
	return datos.HMAC, datos.HuellaSolicitudSHA256, nil
}

func (a AtestacionSelladoMotivoConvocatoria) validarParaMaterial(
	material MaterialIntencionGobiernoConvocatoria,
	instante time.Time,
) error {
	datos, err := a.DatosParaConsumo()
	if err != nil || material.Validar() != nil || !a.coincideMaterial(material) ||
		!instanteGobiernoConvocatoriaCanonico(instante) || instante.Before(datos.AtestacionEmitidaEn) ||
		!instante.Before(datos.AtestacionValidaHasta) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (a AtestacionSelladoMotivoConvocatoria) coincideMaterial(
	material MaterialIntencionGobiernoConvocatoria,
) bool {
	datos, err := a.DatosParaConsumo()
	representacion, errHMAC := datos.HMAC.representacionMaterial()
	return err == nil && errHMAC == nil && datos.Accion == material.Accion &&
		datos.ConvocatoriaRef == material.EstadoPrincipalNuevo.Referencia &&
		datos.HuellaSolicitudSHA256 == material.HuellaSolicitudMotivoSHA256 &&
		datos.HMAC.DominioCriptografico == material.DominioCriptograficoMotivo &&
		datos.HMAC.GeneracionClave == material.GeneracionClaveMotivo &&
		representacion == material.HuellaMotivoHMACSHA256
}

func (AtestacionSelladoMotivoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}

func (AtestacionSelladoMotivoConvocatoria) String() string {
	return "[ATESTACION-SELLADO-MOTIVO-CONVOCATORIA-INTERNA]"
}

func (a AtestacionSelladoMotivoConvocatoria) GoString() string { return a.String() }
func (a AtestacionSelladoMotivoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a AtestacionSelladoMotivoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func validarDatosAtestacionSelladoMotivo(datos DatosAtestacionSelladoMotivoConvocatoria) error {
	if datos.HMAC.Validar() != nil || !especificacionesAutorizacionConvocatoria[datos.Accion].mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(datos.ConvocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.CorrelacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaSolicitudSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.SelladorRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.AtestacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaAtestacionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.TokenConsumoRef) ||
		!referenciasGobiernoConvocatoriaDistintas(
			datos.SelladorRef, datos.AtestacionRef, datos.TokenConsumoRef,
		) || !instanteGobiernoConvocatoriaCanonico(datos.AtestacionEmitidaEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.AtestacionValidaHasta) ||
		!datos.AtestacionValidaHasta.After(datos.AtestacionEmitidaEn) ||
		datos.AtestacionValidaHasta.Sub(datos.AtestacionEmitidaEn) > VigenciaMaximaAtestacionMotivoGobiernoConvocatoria {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

// SelladorMotivoGobiernoConvocatoria debe usar un HSM/KMS o servicio de
// claves versionadas. El contrato no admite una clave recibida por parametro.
type SelladorMotivoGobiernoConvocatoria interface {
	SellarMotivo(
		context.Context,
		SolicitudSellarMotivoGobiernoConvocatoria,
	) (AtestacionSelladoMotivoConvocatoria, error)
}

func textoMotivoGobiernoConvocatoriaValido(valor string) bool {
	if valor == "" || len(valor) > 8000 || valor != strings.TrimSpace(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter < 32 && caracter != '\n' && caracter != '\r' && caracter != '\t' {
			return false
		}
	}
	return true
}

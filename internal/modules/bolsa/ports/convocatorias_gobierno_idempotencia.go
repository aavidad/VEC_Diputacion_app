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
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	VersionTestimonioIdempotenciaConvocatoriaV1 = 1
	VigenciaMaximaTestimonioConvocatoria        = 10 * time.Minute
)

var (
	ErrMaterialIntencionConvocatoriaInvalido = errors.New("bolsa: material de intencion de convocatoria invalido")
	ErrIdempotenciaConvocatoriaInvalida      = errors.New("bolsa: idempotencia semantica de convocatoria invalida")
	ErrClaveIdempotenciaConvocatoriaReusada  = errors.New("bolsa: clave de idempotencia reutilizada con otra intencion")
	ErrSerializacionIdempotenciaConvocatoria = errors.New("bolsa: serializacion de idempotencia de convocatoria prohibida")
)

// ReferenciaEstadoVersionConvocatoria fija referencia, revision y huella del
// agregado completo. No es una referencia a contenido ni a «la ultima» fila.
type ReferenciaEstadoVersionConvocatoria struct {
	Referencia         string `json:"referencia"`
	Revision           int    `json:"revision"`
	HuellaEstadoSHA256 string `json:"huella_estado_sha256"`
}

func (r ReferenciaEstadoVersionConvocatoria) Validar() error {
	if !referenciaVersionGobernadaConvocatoriaValida(r.Referencia) || r.Revision < 1 ||
		!huellaGobiernoConvocatoriaValida(r.HuellaEstadoSHA256) {
		return ErrMaterialIntencionConvocatoriaInvalido
	}
	return nil
}

func EstadoVersionConvocatoria(
	version dominiobolsa.VersionConvocatoriaGobernada,
) (ReferenciaEstadoVersionConvocatoria, error) {
	huella, err := version.HuellaSHA256()
	estado := ReferenciaEstadoVersionConvocatoria{
		Referencia: version.Referencia(), Revision: version.Revision, HuellaEstadoSHA256: huella,
	}
	if err != nil || estado.Validar() != nil {
		return ReferenciaEstadoVersionConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
	}
	return estado, nil
}

// MaterialIntencionGobiernoConvocatoria es la preimagen semantica estable de
// una mutacion. Solo contiene referencias y huellas; nunca motivos en claro.
type MaterialIntencionGobiernoConvocatoria struct {
	Esquema                     string                               `json:"esquema"`
	Accion                      string                               `json:"accion"`
	EstadoPrincipalEsperado     *ReferenciaEstadoVersionConvocatoria `json:"estado_principal_esperado,omitempty"`
	EstadoPrincipalNuevo        ReferenciaEstadoVersionConvocatoria  `json:"estado_principal_nuevo"`
	EstadoRelacionadoEsperado   *ReferenciaEstadoVersionConvocatoria `json:"estado_relacionado_esperado,omitempty"`
	EstadoRelacionadoNuevo      *ReferenciaEstadoVersionConvocatoria `json:"estado_relacionado_nuevo,omitempty"`
	DominioCriptograficoMotivo  string                               `json:"dominio_criptografico_motivo"`
	GeneracionClaveMotivo       uint32                               `json:"generacion_clave_motivo"`
	HuellaSolicitudMotivoSHA256 string                               `json:"huella_solicitud_motivo_sha256"`
	HuellaMotivoHMACSHA256      string                               `json:"huella_motivo_hmac_sha256"`
}

func (m MaterialIntencionGobiernoConvocatoria) Validar() error {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[m.Accion]
	if m.Esquema != "bolsa.convocatoria.intencion.v1" || !conocida || !especificacion.mutacion ||
		m.EstadoPrincipalNuevo.Validar() != nil ||
		m.DominioCriptograficoMotivo != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		m.GeneracionClaveMotivo < 1 ||
		!huellaGobiernoConvocatoriaValida(m.HuellaSolicitudMotivoSHA256) ||
		!huellaHMACGobiernoConvocatoriaValida(m.HuellaMotivoHMACSHA256) {
		return ErrMaterialIntencionConvocatoriaInvalido
	}
	if (m.EstadoPrincipalEsperado != nil && m.EstadoPrincipalEsperado.Validar() != nil) ||
		(m.EstadoRelacionadoEsperado != nil && m.EstadoRelacionadoEsperado.Validar() != nil) ||
		(m.EstadoRelacionadoNuevo != nil && m.EstadoRelacionadoNuevo.Validar() != nil) {
		return ErrMaterialIntencionConvocatoriaInvalido
	}
	switch m.Accion {
	case AccionCrearBorradorConvocatoria:
		if m.EstadoPrincipalEsperado != nil || m.EstadoRelacionadoNuevo != nil {
			return ErrMaterialIntencionConvocatoriaInvalido
		}
	case AccionActualizarBorradorConvocatoria:
		if !cambioPrincipalConvocatoriaValido(m, 1) ||
			m.EstadoRelacionadoEsperado != nil || m.EstadoRelacionadoNuevo != nil {
			return ErrMaterialIntencionConvocatoriaInvalido
		}
	case AccionPublicarVersionConvocatoria, AccionRetirarVersionConvocatoria:
		if !cambioPrincipalConvocatoriaValido(m, 0) ||
			m.EstadoRelacionadoEsperado != nil || m.EstadoRelacionadoNuevo != nil {
			return ErrMaterialIntencionConvocatoriaInvalido
		}
	case AccionPublicarYSustituirConvocatoria:
		if !cambioPrincipalConvocatoriaValido(m, 0) ||
			m.EstadoRelacionadoEsperado == nil || m.EstadoRelacionadoNuevo == nil ||
			m.EstadoRelacionadoEsperado.Referencia != m.EstadoRelacionadoNuevo.Referencia ||
			m.EstadoRelacionadoEsperado.Revision != m.EstadoRelacionadoNuevo.Revision ||
			m.EstadoRelacionadoEsperado.HuellaEstadoSHA256 == m.EstadoRelacionadoNuevo.HuellaEstadoSHA256 ||
			m.EstadoRelacionadoNuevo.Referencia == m.EstadoPrincipalNuevo.Referencia {
			return ErrMaterialIntencionConvocatoriaInvalido
		}
	case AccionPublicarTrasRetiradaConvocatoria:
		if !cambioPrincipalConvocatoriaValido(m, 0) ||
			m.EstadoRelacionadoEsperado == nil || m.EstadoRelacionadoNuevo == nil ||
			*m.EstadoRelacionadoEsperado != *m.EstadoRelacionadoNuevo ||
			m.EstadoRelacionadoNuevo.Referencia == m.EstadoPrincipalNuevo.Referencia {
			return ErrMaterialIntencionConvocatoriaInvalido
		}
	}
	return nil
}

func cambioPrincipalConvocatoriaValido(m MaterialIntencionGobiernoConvocatoria, aumento int) bool {
	return m.EstadoPrincipalEsperado != nil &&
		m.EstadoPrincipalEsperado.Referencia == m.EstadoPrincipalNuevo.Referencia &&
		m.EstadoPrincipalNuevo.Revision == m.EstadoPrincipalEsperado.Revision+aumento &&
		m.EstadoPrincipalEsperado.HuellaEstadoSHA256 != m.EstadoPrincipalNuevo.HuellaEstadoSHA256
}

func (m MaterialIntencionGobiernoConvocatoria) HuellaSHA256() (string, error) {
	if m.Validar() != nil {
		return "", ErrMaterialIntencionConvocatoriaInvalido
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", ErrMaterialIntencionConvocatoriaInvalido
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:]), nil
}

// SolicitudProtegerIdempotenciaConvocatoria no es un DTO HTTP. La clave se
// recibe de una frontera limitada y se convierte en indice HMAC antes de
// llegar al repositorio de gobierno.
type SolicitudProtegerIdempotenciaConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	ClaveIdempotencia string
	PrincipalRef      string
	Material          MaterialIntencionGobiernoConvocatoria
	SolicitadaEn      time.Time
}

func (s SolicitudProtegerIdempotenciaConvocatoria) Validar() error {
	if !claveIdempotenciaConvocatoriaValida(s.ClaveIdempotencia) ||
		!referenciaGobiernoConvocatoriaValida(s.PrincipalRef) || s.Material.Validar() != nil ||
		!instanteGobiernoConvocatoriaCanonico(s.SolicitadaEn) {
		return ErrIdempotenciaConvocatoriaInvalida
	}
	return nil
}

func (SolicitudProtegerIdempotenciaConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaConvocatoria
}

func (SolicitudProtegerIdempotenciaConvocatoria) String() string {
	return "[SOLICITUD-IDEMPOTENCIA-CONVOCATORIA-INTERNA]"
}

func (s SolicitudProtegerIdempotenciaConvocatoria) GoString() string { return s.String() }
func (s SolicitudProtegerIdempotenciaConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudProtegerIdempotenciaConvocatoria) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type DatosTestimonioIdempotenciaConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version                   uint16
	GeneracionClave           uint32
	ClaveHMACRef              string
	ProtectorRef              string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	IndiceOperacionHMACSHA256 string
	PrincipalRef              string
	HuellaIntencionSHA256     string
	EmitidoEn                 time.Time
	ValidoHasta               time.Time
}

// TestimonioIdempotenciaConvocatoria es reconstruible y no acredita por si
// solo la procedencia del protector. La barrera durable relee AtestacionRef y
// su huella antes de consultar o crear el indice semantico.
type TestimonioIdempotenciaConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosTestimonioIdempotenciaConvocatoria
}

func NuevoTestimonioIdempotenciaConvocatoria(
	datos DatosTestimonioIdempotenciaConvocatoria,
) (TestimonioIdempotenciaConvocatoria, error) {
	if validarDatosTestimonioConvocatoria(datos) != nil {
		return TestimonioIdempotenciaConvocatoria{}, ErrIdempotenciaConvocatoriaInvalida
	}
	copia := datos
	return TestimonioIdempotenciaConvocatoria{datos: &copia}, nil
}

func (t TestimonioIdempotenciaConvocatoria) Datos() (DatosTestimonioIdempotenciaConvocatoria, error) {
	if t.datos == nil || validarDatosTestimonioConvocatoria(*t.datos) != nil {
		return DatosTestimonioIdempotenciaConvocatoria{}, ErrIdempotenciaConvocatoriaInvalida
	}
	return *t.datos, nil
}

func (t TestimonioIdempotenciaConvocatoria) ValidarPara(
	material MaterialIntencionGobiernoConvocatoria,
	principalRef string,
) error {
	datos, err := t.Datos()
	huella, errHuella := material.HuellaSHA256()
	if err != nil || errHuella != nil || datos.PrincipalRef != principalRef ||
		datos.HuellaIntencionSHA256 != huella {
		return ErrIdempotenciaConvocatoriaInvalida
	}
	return nil
}

func (TestimonioIdempotenciaConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaConvocatoria
}

func (*TestimonioIdempotenciaConvocatoria) UnmarshalJSON([]byte) error {
	return ErrSerializacionIdempotenciaConvocatoria
}

func (TestimonioIdempotenciaConvocatoria) String() string {
	return "[TESTIMONIO-IDEMPOTENCIA-CONVOCATORIA-OPACO]"
}

func (t TestimonioIdempotenciaConvocatoria) GoString() string { return t.String() }
func (t TestimonioIdempotenciaConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TestimonioIdempotenciaConvocatoria) LogValue() slog.Value {
	return slog.StringValue(t.String())
}

// ProtectorIdempotenciaConvocatorias registra antes de devolver el testimonio
// la atestacion exacta, su generacion de clave y el indice HMAC. El repositorio
// de gobierno no confia en la copia recibida: relee ese registro durable.
type ProtectorIdempotenciaConvocatorias interface {
	Proteger(
		context.Context,
		SolicitudProtegerIdempotenciaConvocatoria,
	) (TestimonioIdempotenciaConvocatoria, error)
}

func validarDatosTestimonioConvocatoria(datos DatosTestimonioIdempotenciaConvocatoria) error {
	if datos.Version != VersionTestimonioIdempotenciaConvocatoriaV1 || datos.GeneracionClave < 1 ||
		!referenciaGobiernoConvocatoriaValida(datos.ClaveHMACRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.ProtectorRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.AtestacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaAtestacionSHA256) ||
		!referenciasGobiernoConvocatoriaDistintas(
			datos.ClaveHMACRef, datos.ProtectorRef, datos.AtestacionRef,
		) ||
		!huellaHMACGobiernoConvocatoriaValida(datos.IndiceOperacionHMACSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.PrincipalRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaIntencionSHA256) ||
		!instanteGobiernoConvocatoriaCanonico(datos.EmitidoEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.ValidoHasta) ||
		!datos.ValidoHasta.After(datos.EmitidoEn) ||
		datos.ValidoHasta.Sub(datos.EmitidoEn) > VigenciaMaximaTestimonioConvocatoria {
		return ErrIdempotenciaConvocatoriaInvalida
	}
	return nil
}

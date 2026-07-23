package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	VersionContratoIntegracionBolsa        uint64 = 1
	MaximoEnteroSeguroIntegracionBolsa            = uint64(9_007_199_254_740_991)
	MaximoElementosIntegracionBolsa        uint32 = 250_000
	VigenciaMaximaPeticionIntegracionBolsa        = 15 * time.Minute

	dominioSelloPeticionBolsa  = "vec.contratacion-temporal.integracion-bolsa-peticion"
	dominioSelloRespuestaBolsa = "vec.contratacion-temporal.integracion-bolsa-respuesta"
)

var (
	ErrPeticionIntegracionBolsaInvalida = errors.New("contratacion temporal: peticion de integracion con bolsa invalida")
	ErrRespuestaBolsaNoConfiable        = errors.New("contratacion temporal: respuesta de bolsa no confiable")
	ErrIntegracionBolsaNoDisponible     = errors.New("contratacion temporal: integracion con bolsa no disponible")
	ErrLimiteIntegracionBolsaExcedido   = errors.New("contratacion temporal: limite de integracion con bolsa excedido")
	ErrEvidenciaBolsaNoAutenticada      = errors.New("contratacion temporal: evidencia de bolsa no autenticada")
	ErrSerializacionCapacidadBolsa      = errors.New("contratacion temporal: serializacion de capacidad de bolsa prohibida")
	ErrEventoBolsaInvalido              = errors.New("contratacion temporal: evento de bolsa invalido")
	ErrColisionEventoBolsa              = errors.New("contratacion temporal: identidad de evento de bolsa reutilizada con otra carga")
	ErrSecuenciaEventoBolsaConflicto    = errors.New("contratacion temporal: secuencia de evento de bolsa en conflicto")
	ErrAcuseEventoBolsaNoConfiable      = errors.New("contratacion temporal: acuse de evento de bolsa no confiable")
)

// ReferenciaVersionadaIntegracionBolsa enlaza un recurso opaco con su versión
// y huella exactas. Las versiones se limitan al entero seguro de JSON para que
// todos los conectores interpreten el mismo valor.
type ReferenciaVersionadaIntegracionBolsa struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaVersionadaIntegracionBolsa) Validar() error {
	if !domain.ReferenciaOpacaValida(r.Referencia) ||
		!enteroSeguroBolsa(r.Version) || !huellaSHA256Valida(r.HuellaSHA256) {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// ContextoPeticionIntegracionBolsa es agnóstico de web, escritorio y demás
// transportes. SelloPeticionHMAC tiene forma nominal: su sintaxis no demuestra
// autenticidad y la aplicación solo lo acepta desde su sellador confiable.
type ContextoPeticionIntegracionBolsa struct {
	OperacionRef      string                               `json:"operacion_ref"`
	OrganizacionRef   string                               `json:"organizacion_ref"`
	ExpedienteRef     string                               `json:"expediente_ref"`
	VersionExpediente uint64                               `json:"version_expediente"`
	CorrelacionRef    string                               `json:"correlacion_ref"`
	ContratoVersion   uint64                               `json:"contrato_version"`
	Finalidad         ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	SelloPeticionHMAC string                               `json:"sello_peticion_hmac"`
	SolicitadaEn      time.Time                            `json:"solicitada_en"`
	ValidaHasta       time.Time                            `json:"valida_hasta"`
}

func (c ContextoPeticionIntegracionBolsa) ValidarEn(instante time.Time) error {
	if !domain.ReferenciaOpacaValida(c.OperacionRef) ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ExpedienteRef) ||
		!enteroSeguroBolsa(c.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(c.CorrelacionRef) ||
		c.ContratoVersion != VersionContratoIntegracionBolsa ||
		c.Finalidad.Validar() != nil ||
		!selloHMACBolsaValido(c.SelloPeticionHMAC, dominioSelloPeticionBolsa) ||
		!instanteBolsaCanonico(c.SolicitadaEn) || !instanteBolsaCanonico(c.ValidaHasta) ||
		!c.ValidaHasta.After(c.SolicitadaEn) ||
		c.ValidaHasta.Sub(c.SolicitadaEn) > VigenciaMaximaPeticionIntegracionBolsa ||
		!instanteBolsaCanonico(instante) ||
		instante.Before(c.SolicitadaEn) || !instante.Before(c.ValidaHasta) {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// EvidenciaNominalIntegracionBolsa es la declaración no confiable recibida del
// conector. Solo VerificadorEvidenciaIntegracionBolsa puede promoverla a un
// comprobante opaco tras recomputar el HMAC con la dependencia TCB.
type EvidenciaNominalIntegracionBolsa struct {
	EvidenciaRef string    `json:"evidencia_ref"`
	SelloHMAC    string    `json:"sello_hmac"`
	EmitidaEn    time.Time `json:"emitida_en"`
	ValidaHasta  time.Time `json:"valida_hasta"`
}

func (e EvidenciaNominalIntegracionBolsa) validarSintaxis() bool {
	return domain.ReferenciaOpacaValida(e.EvidenciaRef) &&
		selloHMACBolsaValido(e.SelloHMAC, dominioSelloRespuestaBolsa) &&
		instanteBolsaCanonico(e.EmitidaEn) && instanteBolsaCanonico(e.ValidaHasta) &&
		e.ValidaHasta.After(e.EmitidaEn) &&
		e.ValidaHasta.Sub(e.EmitidaEn) <= VigenciaMaximaPeticionIntegracionBolsa
}

// ProcedenciaIntegracionBolsa identifica autoridad, fuente y evidencia. No se
// confía en ella hasta autenticar el material canónico completo.
type ProcedenciaIntegracionBolsa struct {
	AutoridadRef    string                               `json:"autoridad_ref"`
	RespuestaRef    string                               `json:"respuesta_ref"`
	ContratoVersion uint64                               `json:"contrato_version"`
	Fuente          ReferenciaVersionadaIntegracionBolsa `json:"fuente"`
	Evidencia       EvidenciaNominalIntegracionBolsa     `json:"evidencia"`
}

func (p ProcedenciaIntegracionBolsa) validarNominalEn(instante time.Time) bool {
	return p.validarNominal() &&
		!instante.Before(p.Evidencia.EmitidaEn) && instante.Before(p.Evidencia.ValidaHasta)
}

func (p ProcedenciaIntegracionBolsa) validarNominal() bool {
	return domain.ReferenciaOpacaValida(p.AutoridadRef) &&
		domain.ReferenciaOpacaValida(p.RespuestaRef) &&
		p.ContratoVersion == VersionContratoIntegracionBolsa &&
		p.Fuente.Validar() == nil && p.Evidencia.validarSintaxis()
}

// SelladorHMACVerificacionBolsa debe ser una dependencia criptográfica de la
// composición TCB (HSM/KMS o equivalente), nunca un dato del transporte.
type SelladorHMACVerificacionBolsa interface {
	SellarDatos(context.Context, []byte) (string, error)
}

// solicitudVerificacionEvidenciaBolsa es opaca para impedir que un adaptador
// de entrada sustituya el material ya canonizado y validado.
type solicitudVerificacionEvidenciaBolsa struct {
	material        []byte
	evidencia       EvidenciaNominalIntegracionBolsa
	autoridadRef    string
	organizacionRef string
	expedienteRef   string
	correlacionRef  string
	respuestaRef    string
	huellaMaterial  string
}

// ComprobanteEvidenciaIntegracionBolsa es una capacidad opaca, efímera y no
// serializable. Su valor cero o una copia ligada a otro material son inválidos.
type ComprobanteEvidenciaIntegracionBolsa struct {
	datos *datosComprobanteEvidenciaBolsa
}

func (ComprobanteEvidenciaIntegracionBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ComprobanteEvidenciaIntegracionBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

type datosComprobanteEvidenciaBolsa struct {
	autoridadRef, organizacionRef, expedienteRef string
	correlacionRef, respuestaRef, evidenciaRef   string
	huellaMaterial, selloHMAC                    string
	verificadaEn                                 time.Time
}

// VerificadorEvidenciaIntegracionBolsa es la única fábrica pública del
// comprobante. La composición debe custodiarlo como TCB y suministrarle el
// sellador confiable; una coincidencia sintáctica nunca basta.
type VerificadorEvidenciaIntegracionBolsa struct {
	sellador SelladorHMACVerificacionBolsa
}

func NuevoVerificadorEvidenciaIntegracionBolsa(
	sellador SelladorHMACVerificacionBolsa,
) (*VerificadorEvidenciaIntegracionBolsa, error) {
	if dependenciaIntegracionBolsaNula(sellador) {
		return nil, ErrEvidenciaBolsaNoAutenticada
	}
	return &VerificadorEvidenciaIntegracionBolsa{sellador: sellador}, nil
}

func (v *VerificadorEvidenciaIntegracionBolsa) verificar(
	ctx context.Context,
	solicitud solicitudVerificacionEvidenciaBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if ctx == nil || v == nil || dependenciaIntegracionBolsaNula(v.sellador) ||
		!solicitud.validaEn(instante) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	if err := ctx.Err(); err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	calculado, err := v.sellador.SellarDatos(ctx, append([]byte(nil), solicitud.material...))
	if err != nil {
		if ctx.Err() != nil {
			return ComprobanteEvidenciaIntegracionBolsa{}, ctx.Err()
		}
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	if !selloHMACBolsaValido(calculado, dominioSelloRespuestaBolsa) ||
		!hmac.Equal([]byte(calculado), []byte(solicitud.evidencia.SelloHMAC)) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return ComprobanteEvidenciaIntegracionBolsa{datos: &datosComprobanteEvidenciaBolsa{
		autoridadRef: solicitud.autoridadRef, organizacionRef: solicitud.organizacionRef,
		expedienteRef: solicitud.expedienteRef, correlacionRef: solicitud.correlacionRef,
		respuestaRef: solicitud.respuestaRef, evidenciaRef: solicitud.evidencia.EvidenciaRef,
		huellaMaterial: solicitud.huellaMaterial, selloHMAC: calculado, verificadaEn: instante,
	}}, nil
}

func (s solicitudVerificacionEvidenciaBolsa) validaEn(instante time.Time) bool {
	if len(s.material) == 0 || !s.evidencia.validarSintaxis() ||
		!instanteBolsaCanonico(instante) || instante.Before(s.evidencia.EmitidaEn) ||
		!instante.Before(s.evidencia.ValidaHasta) ||
		!domain.ReferenciaOpacaValida(s.autoridadRef) ||
		!domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!domain.ReferenciaOpacaValida(s.expedienteRef) ||
		!domain.ReferenciaOpacaValida(s.correlacionRef) ||
		!domain.ReferenciaOpacaValida(s.respuestaRef) ||
		!huellaSHA256Valida(s.huellaMaterial) {
		return false
	}
	return huellaBytesBolsa(s.material) == s.huellaMaterial
}

func (c ComprobanteEvidenciaIntegracionBolsa) coincide(
	solicitud solicitudVerificacionEvidenciaBolsa,
) bool {
	if c.datos == nil || !solicitud.validaEn(c.datos.verificadaEn) {
		return false
	}
	return c.datos.autoridadRef == solicitud.autoridadRef &&
		c.datos.organizacionRef == solicitud.organizacionRef &&
		c.datos.expedienteRef == solicitud.expedienteRef &&
		c.datos.correlacionRef == solicitud.correlacionRef &&
		c.datos.respuestaRef == solicitud.respuestaRef &&
		c.datos.evidenciaRef == solicitud.evidencia.EvidenciaRef &&
		c.datos.huellaMaterial == solicitud.huellaMaterial &&
		hmac.Equal([]byte(c.datos.selloHMAC), []byte(solicitud.evidencia.SelloHMAC))
}

func enteroSeguroBolsa(valor uint64) bool {
	return valor > 0 && valor <= MaximoEnteroSeguroIntegracionBolsa
}

func instanteBolsaCanonico(instante time.Time) bool {
	return domain.InstanteUTCCanonico(instante)
}

func selloHMACBolsaValido(valor, dominio string) bool {
	if !SelloHMACSHA256Valido(valor) {
		return false
	}
	partes := strings.Split(valor, ":")
	prefijo := dominio + "/v"
	if len(partes) != 3 || !strings.HasPrefix(partes[1], prefijo) {
		return false
	}
	version := strings.TrimPrefix(partes[1], prefijo)
	numero, err := strconv.ParseUint(version, 10, 32)
	return err == nil && numero > 0 && version[0] != '0'
}

func huellaBytesBolsa(material []byte) string {
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:])
}

func dependenciaIntegracionBolsaNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

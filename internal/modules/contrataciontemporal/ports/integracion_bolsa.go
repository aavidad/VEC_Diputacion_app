package ports

import (
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
	MaximoClavesRetenidasIntegracionBolsa         = 3

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
// y huella exactas. El límite conserva el mismo valor en JSON, CLI, MCP y
// clientes de escritorio.
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

// EvidenciaNominalIntegracionBolsa es transporte no confiable hasta que el
// verificador TCB comprueba autoridad, generación y MAC.
type EvidenciaNominalIntegracionBolsa struct {
	EvidenciaRef         string    `json:"evidencia_ref"`
	ClaveVerificacionRef string    `json:"clave_verificacion_ref"`
	SelloHMAC            string    `json:"sello_hmac"`
	EmitidaEn            time.Time `json:"emitida_en"`
	ValidaHasta          time.Time `json:"valida_hasta"`
}

func (e EvidenciaNominalIntegracionBolsa) validarSintaxis(dominio string) bool {
	referencia, _, valida := descomponerSelloHMACBolsa(e.SelloHMAC, dominio)
	return domain.ReferenciaOpacaValida(e.EvidenciaRef) &&
		referencia == e.ClaveVerificacionRef && valida &&
		instanteBolsaCanonico(e.EmitidaEn) && instanteBolsaCanonico(e.ValidaHasta) &&
		e.ValidaHasta.After(e.EmitidaEn) &&
		e.ValidaHasta.Sub(e.EmitidaEn) <= VigenciaMaximaPeticionIntegracionBolsa
}

// ProcedenciaIntegracionBolsa no acredita por sí sola a Bolsa. La autoridad
// esperada pertenece a la configuración local del verificador.
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
		p.Fuente.Validar() == nil &&
		p.Evidencia.validarSintaxis(dominioSelloRespuestaBolsa)
}

func enteroSeguroBolsa(valor uint64) bool {
	return valor > 0 && valor <= MaximoEnteroSeguroIntegracionBolsa
}

func instanteBolsaCanonico(instante time.Time) bool {
	return domain.InstanteUTCCanonico(instante)
}

func selloHMACBolsaValido(valor, dominio string) bool {
	_, _, valida := descomponerSelloHMACBolsa(valor, dominio)
	return valida
}

func descomponerSelloHMACBolsa(valor, dominio string) (string, uint32, bool) {
	if !SelloHMACSHA256Valido(valor) {
		return "", 0, false
	}
	partes := strings.Split(valor, ":")
	prefijo := dominio + "/v"
	if len(partes) != 3 || !strings.HasPrefix(partes[1], prefijo) {
		return "", 0, false
	}
	version := strings.TrimPrefix(partes[1], prefijo)
	numero, err := strconv.ParseUint(version, 10, 32)
	if err != nil || numero == 0 || version[0] == '0' {
		return "", 0, false
	}
	return partes[1], uint32(numero), true
}

func huellaBytesBolsa(material []byte) string {
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:])
}

func huellasBolsaIguales(primera, segunda string) bool {
	return huellaSHA256Valida(primera) && huellaSHA256Valida(segunda) &&
		hmac.Equal([]byte(primera), []byte(segunda))
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

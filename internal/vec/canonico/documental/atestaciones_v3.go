package documental

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

const (
	AlgoritmoHMACSHA256V3                = "hmac-sha256"
	AudienciaTokenCercadoV3              = "vec.documentos.token-cercado.v3"
	AudienciaInicioEfectoV3              = "vec.documentos.inicio-efecto.v3"
	AudienciaReclamacionDespachoV3       = "vec.documentos.reclamacion-despacho.v3"
	AudienciaComprobacionOrdenDespachoV3 = "vec.documentos.comprobacion-orden-despacho.v3"
	ContextoTokenCercadoV3               = "cercado"
	ContextoInicioEfectoV3               = "inicio"
	ContextoReclamacionDespachoV3        = "reclamacion"
	ContextoComprobacionOrdenDespachoV3  = "inicio-reclamacion-cercada"
	TamanoFirmaHMACSHA256V3              = 32
)

const (
	esquemaVinculoActivacionV3          = "vec.documentos.vinculo-estable-activacion.v3"
	esquemaCercadoEjecucionV3           = "vec.documentos.cercado-ejecucion.v3"
	esquemaAtestacionTokenCercadoV3     = "vec.documentos.atestacion-token-cercado.v3"
	esquemaSolicitudVerificacionTokenV3 = "vec.documentos.solicitud-verificacion-token-cercado.v3"
	esquemaPruebaAtestacionDespachoV3   = "vec.documentos.prueba-cruda-atestacion-despacho.v3"
)

// DatosVinculoActivacionV3 es la proyeccion pura del vinculo nominal. Duplica
// los valores comprometidos por el manifiesto para poder cotejarlos sin
// importar tipos del puerto.
type DatosVinculoActivacionV3 struct {
	ReservaRef                    string
	IndiceIdempotenciaHMAC        string
	HuellaSolicitudHMAC           string
	HuellaEntradaHMAC             string
	HuellaManifiestoSHA256        string
	EfectoManifiestoRef           string
	HuellaPlanManifiestoSHA256    string
	OrdenConsumoDurableV4Ref      string
	DecisionRef                   string
	EfectoDecisionRef             string
	EsquemaHuellaDecision         string
	EsquemaHuellaDecisionEsperado string
	HuellaDecisionSHA256          string
	HuellaPlanDecisionSHA256      string
}

func (d DatosVinculoActivacionV3) Validar() bool {
	return ReferenciaEjecucionV3Valida(d.ReservaRef) &&
		HMACSHA256V3Valido(d.IndiceIdempotenciaHMAC) &&
		HMACSHA256V3Valido(d.HuellaSolicitudHMAC) &&
		HMACSHA256V3Valido(d.HuellaEntradaHMAC) &&
		ClavesHMACSHA256V3Distintas(
			d.IndiceIdempotenciaHMAC, d.HuellaSolicitudHMAC, d.HuellaEntradaHMAC,
		) && SHA256HexadecimalValido(d.HuellaManifiestoSHA256) &&
		ReferenciaEjecucionV3Valida(d.EfectoManifiestoRef) &&
		SHA256HexadecimalValido(d.HuellaPlanManifiestoSHA256) &&
		ReferenciaEjecucionV3Valida(d.OrdenConsumoDurableV4Ref) &&
		ReferenciaEjecucionV3Valida(d.DecisionRef) &&
		ReferenciaEjecucionV3Valida(d.EfectoDecisionRef) &&
		d.OrdenConsumoDurableV4Ref == d.EfectoDecisionRef &&
		d.OrdenConsumoDurableV4Ref == d.EfectoManifiestoRef &&
		d.EsquemaHuellaDecision != "" &&
		d.EsquemaHuellaDecision == d.EsquemaHuellaDecisionEsperado &&
		SHA256HexadecimalValido(d.HuellaDecisionSHA256) &&
		d.HuellaPlanDecisionSHA256 == d.HuellaPlanManifiestoSHA256
}

func (d DatosVinculoActivacionV3) HuellaSHA256() string {
	if !d.Validar() {
		return ""
	}
	return HuellaCamposSHA256V3([]string{
		esquemaVinculoActivacionV3, d.ReservaRef, d.IndiceIdempotenciaHMAC,
		d.HuellaSolicitudHMAC, d.HuellaManifiestoSHA256, d.OrdenConsumoDurableV4Ref,
		d.DecisionRef, d.EfectoDecisionRef, d.EsquemaHuellaDecision,
		d.HuellaDecisionSHA256, d.HuellaPlanDecisionSHA256,
	})
}

// DatosTokenCercadoV3 contiene solo primitivas y cotejos ya calculados por el
// puerto. La MAC se valida estructuralmente; su autenticidad corresponde al KMS.
type DatosTokenCercadoV3 struct {
	Valor                       string
	Secuencia                   uint64
	HuellaVinculoEstableSHA256  string
	HuellaVinculoEsperadoSHA256 string
	HuellaVinculoInternoSHA256  string
	HuellaVinculoCercadoSHA256  string
	ClaveAtestacionRef          string
	RevisionClave               uint64
	MACAtestacion               []byte
	EvidenciaOperacionRef       string
	ClaveHuellaEntradaHMAC      string
}

func HuellaVinculoCercadoV3(secuencia uint64, huellaVinculoEstable string) string {
	if secuencia == 0 || !SHA256HexadecimalValido(huellaVinculoEstable) {
		return ""
	}
	return HuellaCamposSHA256V3([]string{
		esquemaCercadoEjecucionV3, Uint64Decimal(secuencia), huellaVinculoEstable,
	})
}

func (d DatosTokenCercadoV3) Validar() bool {
	return ReferenciaEjecucionV3Valida(d.Valor) && d.Secuencia > 0 &&
		SHA256HexadecimalValido(d.HuellaVinculoEstableSHA256) &&
		d.HuellaVinculoEstableSHA256 == d.HuellaVinculoEsperadoSHA256 &&
		d.HuellaVinculoEstableSHA256 == d.HuellaVinculoInternoSHA256 &&
		SHA256HexadecimalValido(d.HuellaVinculoCercadoSHA256) &&
		d.HuellaVinculoCercadoSHA256 == HuellaVinculoCercadoV3(
			d.Secuencia, d.HuellaVinculoEstableSHA256,
		) && ReferenciaEjecucionV3Valida(d.ClaveAtestacionRef) && d.RevisionClave > 0 &&
		d.ClaveAtestacionRef != d.ClaveHuellaEntradaHMAC &&
		len(d.MACAtestacion) == TamanoFirmaHMACSHA256V3 && BytesNoNulos(d.MACAtestacion) &&
		ReferenciaEjecucionV3Valida(d.EvidenciaOperacionRef)
}

func (d DatosTokenCercadoV3) MensajeAtestacion() []byte {
	if !d.Validar() {
		return nil
	}
	return SerializarCamposV3([]string{
		esquemaAtestacionTokenCercadoV3, d.Valor, Uint64Decimal(d.Secuencia),
		d.HuellaVinculoEstableSHA256, d.HuellaVinculoCercadoSHA256,
		AlgoritmoHMACSHA256V3, AudienciaTokenCercadoV3, ContextoTokenCercadoV3,
		d.ClaveAtestacionRef, Uint64Decimal(d.RevisionClave), d.EvidenciaOperacionRef,
	})
}

func HuellaSolicitudVerificacionTokenV3(mensaje, mac []byte) string {
	return HuellaCamposSHA256V3([]string{
		esquemaSolicitudVerificacionTokenV3, HuellaBytesSHA256(mensaje), HuellaBytesSHA256(mac),
	})
}

// DatosPruebaAtestacionDespachoV3 representa material nominal restaurable.
type DatosPruebaAtestacionDespachoV3 struct {
	Algoritmo               string
	Audiencia               string
	Contexto                string
	ClaveGestionadaRef      string
	RevisionClaveGestionada uint64
	EvidenciaOperacionRef   string
	MensajeCanonico         []byte
	SobreCriptografico      []byte
	HuellaMensajeSHA256     string
	HuellaSobreSHA256       string
}

func PerfilAtestacionDespachoV3Valido(algoritmo, audiencia, contexto string) bool {
	return algoritmo == AlgoritmoHMACSHA256V3 &&
		((audiencia == AudienciaTokenCercadoV3 && contexto == ContextoTokenCercadoV3) ||
			(audiencia == AudienciaInicioEfectoV3 && contexto == ContextoInicioEfectoV3) ||
			(audiencia == AudienciaReclamacionDespachoV3 && contexto == ContextoReclamacionDespachoV3))
}

func (d DatosPruebaAtestacionDespachoV3) Validar() bool {
	return PerfilAtestacionDespachoV3Valido(d.Algoritmo, d.Audiencia, d.Contexto) &&
		ReferenciaEjecucionV3Valida(d.ClaveGestionadaRef) && d.RevisionClaveGestionada > 0 &&
		ReferenciaEjecucionV3Valida(d.EvidenciaOperacionRef) &&
		len(d.MensajeCanonico) > 0 && len(d.MensajeCanonico) <= 64*1024 &&
		len(d.SobreCriptografico) == TamanoFirmaHMACSHA256V3 && BytesNoNulos(d.SobreCriptografico) &&
		SHA256HexadecimalValido(d.HuellaMensajeSHA256) &&
		d.HuellaMensajeSHA256 == HuellaBytesSHA256(d.MensajeCanonico) &&
		SHA256HexadecimalValido(d.HuellaSobreSHA256) &&
		d.HuellaSobreSHA256 == HuellaBytesSHA256(d.SobreCriptografico)
}

func (d DatosPruebaAtestacionDespachoV3) HuellaSHA256() string {
	return HuellaCamposSHA256V3([]string{
		esquemaPruebaAtestacionDespachoV3, d.Algoritmo, d.Audiencia, d.Contexto,
		d.ClaveGestionadaRef, Uint64Decimal(d.RevisionClaveGestionada),
		d.EvidenciaOperacionRef, d.HuellaMensajeSHA256, d.HuellaSobreSHA256,
	})
}

func (DatosPruebaAtestacionDespachoV3) String() string {
	return "[DATOS-PRUEBA-ATESTACION-DESPACHO-V3-CONFIDENCIALES-REDACTADOS]"
}
func (d DatosPruebaAtestacionDespachoV3) GoString() string { return d.String() }
func (d DatosPruebaAtestacionDespachoV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosPruebaAtestacionDespachoV3) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosPruebaAtestacionDespachoV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosPruebaAtestacionDespachoV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosPruebaAtestacionDespachoV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosPruebaAtestacionDespachoV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosPruebaAtestacionDespachoV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosPruebaAtestacionDespachoV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// DatosMetadatosComprobacionV3 concentra el cotejo nominal de una comprobacion.
type DatosMetadatosComprobacionV3 struct {
	HuellaSolicitud         string
	HuellaSolicitudEsperada string
	VerificacionRef         string
	VerificadaEn            time.Time
}

func (d DatosMetadatosComprobacionV3) Validar() bool {
	return SHA256HexadecimalValido(d.HuellaSolicitud) &&
		d.HuellaSolicitud == d.HuellaSolicitudEsperada &&
		ReferenciaEjecucionV3Valida(d.VerificacionRef) && InstanteV3Valido(d.VerificadaEn)
}

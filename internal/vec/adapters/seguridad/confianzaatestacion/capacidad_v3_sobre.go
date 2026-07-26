package confianzaatestacion

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

const (
	esquemaCapacidadAtestacionAutorizacionV3        = "vec.autorizacion.capacidad-registro-consumo-atestado.v3"
	versionCapacidadAtestacionAutorizacionV3 uint16 = 3
	maximoBytesExportacionCapacidadV3               = 32 * 1024
	maximoEnteroExactoJSONCapacidadV3        uint64 = 1<<53 - 1
)

type capacidadAtestacionAutorizacionV3JSON struct {
	Esquema                   string `json:"esquema"`
	Version                   uint16 `json:"version"`
	ClaveID                   string `json:"clave_id"`
	ClaveVersion              uint64 `json:"clave_version"`
	RevisionGobierno          uint64 `json:"revision_gobierno"`
	HuellaGobiernoSHA256      string `json:"huella_gobierno_sha256"`
	EmisorID                  string `json:"emisor_id"`
	AudienciaConsumo          string `json:"audiencia_consumo"`
	Nonce                     string `json:"nonce"`
	EmitidaEn                 string `json:"emitida_en"`
	ExpiraEn                  string `json:"expira_en"`
	DecisionRef               string `json:"decision_ref"`
	HuellaDecisionSHA256      string `json:"huella_decision_sha256"`
	HuellaMotivoSHA256        string `json:"huella_motivo_sha256"`
	HuellaPayloadVECAD3SHA256 string `json:"huella_payload_vec_ad_3_sha256"`
	HuellaSobreCOSESHA256     string `json:"huella_sobre_cose_sign1_sha256"`
	HuellaPruebaSHA256        string `json:"huella_prueba_confianza_sha256"`
	ContextoRef               string `json:"contexto_ref"`
	HuellaContextoSHA256      string `json:"huella_contexto_sha256"`
	AudienciaDespliegue       string `json:"audiencia_despliegue"`
	Operacion                 string `json:"operacion"`
	EfectoRef                 string `json:"efecto_ref"`
	HuellaEfectoSHA256        string `json:"huella_efecto_sha256"`
	DecisionValidaHasta       string `json:"decision_valida_hasta"`
	VerificadaEn              string `json:"verificada_en"`
	RevisionConfianza         string `json:"revision_confianza"`
	ConfiguracionSecuencia    uint64 `json:"configuracion_secuencia"`
	HuellaConfiguracionSHA256 string `json:"huella_configuracion_sha256"`
	ConfiguracionPublicadaEn  string `json:"configuracion_publicada_en"`
	ConfiguracionExpiraEn     string `json:"configuracion_expira_en"`
	RaizClaveID               string `json:"raiz_clave_id"`
	RaizVersion               uint64 `json:"raiz_version"`
	HuellaRaizSPKISHA256      string `json:"huella_raiz_spki_sha256"`
	RaizValidaDesde           string `json:"raiz_valida_desde"`
	RaizValidaHasta           string `json:"raiz_valida_hasta"`
	Suite                     string `json:"suite"`
	MACSHA256                 string `json:"mac_sha256"`
}

func (c capacidadAtestacionAutorizacionV3JSON) valoresAutenticados() []string {
	return []string{
		c.Esquema,
		strconv.FormatUint(uint64(c.Version), 10),
		c.ClaveID,
		strconv.FormatUint(c.ClaveVersion, 10),
		strconv.FormatUint(c.RevisionGobierno, 10),
		c.HuellaGobiernoSHA256,
		c.EmisorID,
		c.AudienciaConsumo,
		c.Nonce,
		c.EmitidaEn,
		c.ExpiraEn,
		c.DecisionRef,
		c.HuellaDecisionSHA256,
		c.HuellaMotivoSHA256,
		c.HuellaPayloadVECAD3SHA256,
		c.HuellaSobreCOSESHA256,
		c.HuellaPruebaSHA256,
		c.ContextoRef,
		c.HuellaContextoSHA256,
		c.AudienciaDespliegue,
		c.Operacion,
		c.EfectoRef,
		c.HuellaEfectoSHA256,
		c.DecisionValidaHasta,
		c.VerificadaEn,
		c.RevisionConfianza,
		strconv.FormatUint(c.ConfiguracionSecuencia, 10),
		c.HuellaConfiguracionSHA256,
		c.ConfiguracionPublicadaEn,
		c.ConfiguracionExpiraEn,
		c.RaizClaveID,
		strconv.FormatUint(c.RaizVersion, 10),
		c.HuellaRaizSPKISHA256,
		c.RaizValidaDesde,
		c.RaizValidaHasta,
		c.Suite,
	}
}

func (c capacidadAtestacionAutorizacionV3JSON) validarEstructura() error {
	emitidaEn, errEmitida := parsearInstanteCapacidadV3(c.EmitidaEn)
	expiraEn, errExpira := parsearInstanteCapacidadV3(c.ExpiraEn)
	decisionHasta, errDecision := parsearInstanteCapacidadV3(c.DecisionValidaHasta)
	verificadaEn, errVerificada := parsearInstanteCapacidadV3(c.VerificadaEn)
	publicadaEn, errPublicada := parsearInstanteCapacidadV3(c.ConfiguracionPublicadaEn)
	configuracionExpira, errConfiguracion := parsearInstanteCapacidadV3(c.ConfiguracionExpiraEn)
	raizDesde, errRaizDesde := parsearInstanteCapacidadV3(c.RaizValidaDesde)
	raizHasta, errRaizHasta := parsearInstanteCapacidadV3(c.RaizValidaHasta)
	if errEmitida != nil || errExpira != nil || errDecision != nil ||
		errVerificada != nil || errPublicada != nil ||
		errConfiguracion != nil || errRaizDesde != nil || errRaizHasta != nil ||
		c.Esquema != esquemaCapacidadAtestacionAutorizacionV3 ||
		c.Version != versionCapacidadAtestacionAutorizacionV3 ||
		!referenciaPruebaConfianzaValida(c.ClaveID) ||
		c.ClaveVersion == 0 || c.RevisionGobierno == 0 ||
		c.ClaveVersion > maximoEnteroExactoJSONCapacidadV3 ||
		c.RevisionGobierno > maximoEnteroExactoJSONCapacidadV3 ||
		!huellaSHA256ConfianzaValida(c.HuellaGobiernoSHA256) ||
		!referenciaPruebaConfianzaValida(c.EmisorID) ||
		!referenciaPruebaConfianzaValida(c.AudienciaConsumo) ||
		!huellaSHA256ConfianzaValida(c.Nonce) ||
		c.Nonce == strings.Repeat("0", 64) ||
		!expiraEn.After(emitidaEn) ||
		expiraEn.Sub(emitidaEn) > VigenciaMaximaCapacidadAtestacionAutorizacionV3 ||
		expiraEn.After(decisionHasta) ||
		expiraEn.After(configuracionExpira) ||
		expiraEn.After(raizHasta) ||
		!referenciaPruebaConfianzaValida(c.DecisionRef) ||
		!huellaSHA256ConfianzaValida(c.HuellaDecisionSHA256) ||
		!huellaSHA256ConfianzaValida(c.HuellaMotivoSHA256) ||
		!huellaSHA256ConfianzaValida(c.HuellaPayloadVECAD3SHA256) ||
		!huellaSHA256ConfianzaValida(c.HuellaSobreCOSESHA256) ||
		!huellaSHA256ConfianzaValida(c.HuellaPruebaSHA256) ||
		!referenciaPruebaConfianzaValida(c.ContextoRef) ||
		!huellaSHA256ConfianzaValida(c.HuellaContextoSHA256) ||
		!referenciaPruebaConfianzaValida(c.AudienciaDespliegue) ||
		!referenciaPruebaConfianzaValida(c.Operacion) ||
		!referenciaPruebaConfianzaValida(c.EfectoRef) ||
		!huellaSHA256ConfianzaValida(c.HuellaEfectoSHA256) ||
		verificadaEn.After(emitidaEn) ||
		!referenciaConfiguracionConfianzaValida(c.RevisionConfianza) ||
		c.ConfiguracionSecuencia == 0 ||
		c.ConfiguracionSecuencia > maximoEnteroExactoJSONCapacidadV3 ||
		!huellaSHA256ConfianzaValida(c.HuellaConfiguracionSHA256) ||
		verificadaEn.Before(publicadaEn) ||
		!verificadaEn.Before(configuracionExpira) ||
		!referenciaPruebaConfianzaValida(c.RaizClaveID) ||
		c.RaizVersion == 0 ||
		c.RaizVersion > maximoEnteroExactoJSONCapacidadV3 ||
		!huellaSHA256ConfianzaValida(c.HuellaRaizSPKISHA256) ||
		verificadaEn.Before(raizDesde) ||
		!verificadaEn.Before(raizHasta) ||
		c.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA ||
		!huellaSHA256ConfianzaValida(c.MACSHA256) {
		return ErrCapacidadAtestacionV3Invalida
	}
	return nil
}

// CapacidadBreveAtestacionAutorizacionV3 es opaca y bloquea todos los codecs
// genericos. La unica salida permitida es ExportacionCanonicaParaConsumidor,
// destinada al puerto SQL cerrado que vuelve a verificar MAC, clave, gobierno,
// revocacion, tiempo y todos los artefactos.
type CapacidadBreveAtestacionAutorizacionV3 struct {
	bloqueoSerializacionCapacidadV3
	contenido []byte
}

func nuevaCapacidadBreveAtestacionAutorizacionV3(
	documento capacidadAtestacionAutorizacionV3JSON,
) (CapacidadBreveAtestacionAutorizacionV3, error) {
	if documento.validarEstructura() != nil {
		return CapacidadBreveAtestacionAutorizacionV3{},
			ErrCapacidadAtestacionV3Invalida
	}
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoBytesExportacionCapacidadV3 {
		return CapacidadBreveAtestacionAutorizacionV3{},
			ErrCapacidadAtestacionV3Invalida
	}
	return CapacidadBreveAtestacionAutorizacionV3{
		contenido: append([]byte(nil), contenido...),
	}, nil
}

func (c CapacidadBreveAtestacionAutorizacionV3) validar() error {
	documento, err := interpretarExportacionCapacidadV3(c.contenido)
	if err != nil {
		return ErrCapacidadAtestacionV3Invalida
	}
	canonica, err := json.Marshal(documento)
	if err != nil || !bytes.Equal(canonica, c.contenido) {
		return ErrCapacidadAtestacionV3Invalida
	}
	return nil
}

func (c CapacidadBreveAtestacionAutorizacionV3) ExportacionCanonicaParaConsumidor() (
	[]byte,
	error,
) {
	if c.validar() != nil {
		return nil, ErrCapacidadAtestacionV3Invalida
	}
	return append([]byte(nil), c.contenido...), nil
}

func (c CapacidadBreveAtestacionAutorizacionV3) ResumenParaConsumidor() (
	ports.ResumenCapacidadAtestacionAutorizacionV3,
	error,
) {
	documento, err := interpretarExportacionCapacidadV3(c.contenido)
	if err != nil {
		return ports.ResumenCapacidadAtestacionAutorizacionV3{},
			ErrCapacidadAtestacionV3Invalida
	}
	emitidaEn, errEmitida := parsearInstanteCapacidadV3(documento.EmitidaEn)
	expiraEn, errExpira := parsearInstanteCapacidadV3(documento.ExpiraEn)
	if errEmitida != nil || errExpira != nil {
		return ports.ResumenCapacidadAtestacionAutorizacionV3{},
			ErrCapacidadAtestacionV3Invalida
	}
	return ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		documento.DecisionRef,
		documento.HuellaDecisionSHA256,
		documento.HuellaMotivoSHA256,
		documento.ContextoRef,
		documento.HuellaContextoSHA256,
		documento.Operacion,
		documento.EfectoRef,
		documento.HuellaEfectoSHA256,
		documento.AudienciaConsumo,
		emitidaEn,
		expiraEn,
	)
}

func interpretarExportacionCapacidadV3(
	contenido []byte,
) (capacidadAtestacionAutorizacionV3JSON, error) {
	var documento capacidadAtestacionAutorizacionV3JSON
	if len(contenido) == 0 || len(contenido) > maximoBytesExportacionCapacidadV3 ||
		validarObjetoPlanoJSONCapacidadV3(contenido) != nil {
		return documento, ErrCapacidadAtestacionV3Invalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&documento); err != nil ||
		decodificador.Decode(&struct{}{}) == nil ||
		documento.validarEstructura() != nil {
		return capacidadAtestacionAutorizacionV3JSON{},
			ErrCapacidadAtestacionV3Invalida
	}
	canonica, err := json.Marshal(documento)
	if err != nil || !bytes.Equal(canonica, contenido) {
		return capacidadAtestacionAutorizacionV3JSON{},
			ErrCapacidadAtestacionV3Invalida
	}
	return documento, nil
}

func validarObjetoPlanoJSONCapacidadV3(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return ErrCapacidadAtestacionV3Invalida
	}
	claves := make(map[string]struct{}, 37)
	for decodificador.More() {
		token, err := decodificador.Token()
		clave, correcta := token.(string)
		if err != nil || !correcta {
			return ErrCapacidadAtestacionV3Invalida
		}
		if _, existe := claves[clave]; existe {
			return ErrCapacidadAtestacionV3Invalida
		}
		claves[clave] = struct{}{}
		var valor any
		if err := decodificador.Decode(&valor); err != nil {
			return ErrCapacidadAtestacionV3Invalida
		}
		switch valor.(type) {
		case string, float64:
		default:
			return ErrCapacidadAtestacionV3Invalida
		}
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim('}') || len(claves) != 37 {
		return ErrCapacidadAtestacionV3Invalida
	}
	return nil
}

func parsearInstanteCapacidadV3(valor string) (time.Time, error) {
	instante, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil || instante.Format(time.RFC3339Nano) != valor ||
		!instanteCanonicoConfianza(instante) {
		return time.Time{}, ErrCapacidadAtestacionV3Invalida
	}
	return instante, nil
}

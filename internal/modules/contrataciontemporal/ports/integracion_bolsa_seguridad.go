package ports

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaEvidenciaDurableIntegracionBolsa = "vec.contratacion-temporal.evidencia-bolsa.v1"

// VerificadorHMACIntegracionBolsa expone solo verificación. A diferencia de
// un sellador, el consumidor no puede emplearlo para fabricar respuestas de
// Bolsa.
type VerificadorHMACIntegracionBolsa interface {
	VerificarDatos(
		context.Context,
		string,
		[]byte,
		string,
	) error
}

type anilloVerificacionHMACBolsa struct {
	activa     string
	retenidas  []string
	permitidas map[string]struct{}
}

func nuevoAnilloVerificacionHMACBolsa(
	dominio string,
	activa string,
	retenidas []string,
) (anilloVerificacionHMACBolsa, error) {
	if len(retenidas) > MaximoClavesRetenidasIntegracionBolsa {
		return anilloVerificacionHMACBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	_, generacionActiva, valida := descomponerSelloHMACBolsa(
		"hmac-sha256:"+activa+":"+digestSintacticoNoNuloBolsa(),
		dominio,
	)
	if !valida {
		return anilloVerificacionHMACBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	anillo := anilloVerificacionHMACBolsa{
		activa:     activa,
		retenidas:  append([]string(nil), retenidas...),
		permitidas: make(map[string]struct{}, len(retenidas)+1),
	}
	anillo.permitidas[activa] = struct{}{}
	anterior := generacionActiva
	for _, referencia := range anillo.retenidas {
		_, generacion, valida := descomponerSelloHMACBolsa(
			"hmac-sha256:"+referencia+":"+digestSintacticoNoNuloBolsa(),
			dominio,
		)
		if !valida || generacion >= anterior {
			return anilloVerificacionHMACBolsa{}, ErrEvidenciaBolsaNoAutenticada
		}
		if _, repetida := anillo.permitidas[referencia]; repetida {
			return anilloVerificacionHMACBolsa{}, ErrEvidenciaBolsaNoAutenticada
		}
		anillo.permitidas[referencia] = struct{}{}
		anterior = generacion
	}
	return anillo, nil
}

func (a anilloVerificacionHMACBolsa) contiene(referencia string) bool {
	_, existe := a.permitidas[referencia]
	return existe
}

// EvidenciaDurableIntegracionBolsa es serializable y minimizada. No concede
// acceso por sí sola: tras reinicio se reautentica contra los bytes canónicos,
// la autoridad esperada y el anillo local.
type EvidenciaDurableIntegracionBolsa struct {
	Esquema               string    `json:"esquema"`
	TipoMaterial          string    `json:"tipo_material"`
	AutoridadRef          string    `json:"autoridad_ref"`
	ClaveVerificacionRef  string    `json:"clave_verificacion_ref"`
	EvidenciaRef          string    `json:"evidencia_ref"`
	PeticionRef           string    `json:"peticion_ref"`
	HuellaPeticionSHA256  string    `json:"huella_peticion_sha256"`
	RespuestaRef          string    `json:"respuesta_ref"`
	HuellaRespuestaSHA256 string    `json:"huella_respuesta_sha256"`
	SelloHMAC             string    `json:"sello_hmac"`
	EmitidaEn             time.Time `json:"emitida_en"`
	ValidaHasta           time.Time `json:"valida_hasta"`
	RetenerHasta          time.Time `json:"retener_hasta"`
}

func (e EvidenciaDurableIntegracionBolsa) Validar() error {
	referencia, _, valida := descomponerSelloHMACBolsa(
		e.SelloHMAC,
		dominioSelloRespuestaBolsa,
	)
	if e.Esquema != esquemaEvidenciaDurableIntegracionBolsa ||
		!claveCanonicaBolsa(e.TipoMaterial) ||
		!domain.ReferenciaOpacaValida(e.AutoridadRef) ||
		!valida || referencia != e.ClaveVerificacionRef ||
		!domain.ReferenciaOpacaValida(e.EvidenciaRef) ||
		!domain.ReferenciaOpacaValida(e.PeticionRef) ||
		!huellaSHA256Valida(e.HuellaPeticionSHA256) ||
		!domain.ReferenciaOpacaValida(e.RespuestaRef) ||
		!huellaSHA256Valida(e.HuellaRespuestaSHA256) ||
		!instanteBolsaCanonico(e.EmitidaEn) ||
		!instanteBolsaCanonico(e.ValidaHasta) ||
		!instanteBolsaCanonico(e.RetenerHasta) ||
		!e.ValidaHasta.After(e.EmitidaEn) ||
		e.ValidaHasta.Sub(e.EmitidaEn) > VigenciaMaximaPeticionIntegracionBolsa ||
		!e.RetenerHasta.After(e.ValidaHasta) {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

type datosComprobanteEvidenciaBolsa struct {
	evidencia    EvidenciaDurableIntegracionBolsa
	verificadaEn time.Time
}

type ComprobanteEvidenciaIntegracionBolsa struct {
	datos *datosComprobanteEvidenciaBolsa
}

func (ComprobanteEvidenciaIntegracionBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ComprobanteEvidenciaIntegracionBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func (c ComprobanteEvidenciaIntegracionBolsa) coincide(
	evidencia EvidenciaDurableIntegracionBolsa,
) bool {
	return c.datos != nil && c.datos.evidencia.Validar() == nil &&
		evidenciasDurablesBolsaIguales(c.datos.evidencia, evidencia)
}

func evidenciasDurablesBolsaIguales(
	primera EvidenciaDurableIntegracionBolsa,
	segunda EvidenciaDurableIntegracionBolsa,
) bool {
	return primera.Validar() == nil && segunda.Validar() == nil &&
		primera.Esquema == segunda.Esquema &&
		primera.TipoMaterial == segunda.TipoMaterial &&
		primera.AutoridadRef == segunda.AutoridadRef &&
		primera.ClaveVerificacionRef == segunda.ClaveVerificacionRef &&
		primera.EvidenciaRef == segunda.EvidenciaRef &&
		primera.PeticionRef == segunda.PeticionRef &&
		huellasBolsaIguales(
			primera.HuellaPeticionSHA256,
			segunda.HuellaPeticionSHA256,
		) &&
		primera.RespuestaRef == segunda.RespuestaRef &&
		huellasBolsaIguales(
			primera.HuellaRespuestaSHA256,
			segunda.HuellaRespuestaSHA256,
		) &&
		hmac.Equal(
			[]byte(primera.SelloHMAC),
			[]byte(segunda.SelloHMAC),
		) &&
		primera.EmitidaEn.Equal(segunda.EmitidaEn) &&
		primera.ValidaHasta.Equal(segunda.ValidaHasta) &&
		primera.RetenerHasta.Equal(segunda.RetenerHasta)
}

func (c ComprobanteEvidenciaIntegracionBolsa) instanteVerificacion() time.Time {
	if c.datos == nil {
		return time.Time{}
	}
	return c.datos.verificadaEn
}

// VerificadorEvidenciaIntegracionBolsa fija la autoridad y generaciones
// admitidas en composición. Un valor AutoridadRef recibido nunca amplía esa
// lista.
type VerificadorEvidenciaIntegracionBolsa struct {
	autoridadEsperada string
	anillo            anilloVerificacionHMACBolsa
	verificador       VerificadorHMACIntegracionBolsa
}

func NuevoVerificadorEvidenciaIntegracionBolsa(
	autoridadEsperada string,
	claveActivaRef string,
	clavesRetenidas []string,
	verificador VerificadorHMACIntegracionBolsa,
) (*VerificadorEvidenciaIntegracionBolsa, error) {
	anillo, err := nuevoAnilloVerificacionHMACBolsa(
		dominioSelloRespuestaBolsa,
		claveActivaRef,
		clavesRetenidas,
	)
	if err != nil || !domain.ReferenciaOpacaValida(autoridadEsperada) ||
		dependenciaIntegracionBolsaNula(verificador) {
		return nil, ErrEvidenciaBolsaNoAutenticada
	}
	return &VerificadorEvidenciaIntegracionBolsa{
		autoridadEsperada: autoridadEsperada,
		anillo:            anillo,
		verificador:       verificador,
	}, nil
}

func (v *VerificadorEvidenciaIntegracionBolsa) verificarFresco(
	ctx context.Context,
	tipoMaterial string,
	peticionRef string,
	materialPeticion []byte,
	materialRespuesta []byte,
	procedencia ProcedenciaIntegracionBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, EvidenciaDurableIntegracionBolsa, error) {
	if !instanteBolsaCanonico(instante) ||
		!procedencia.validarNominalEn(instante) {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrRespuestaBolsaNoConfiable
	}
	evidencia := nuevaEvidenciaDurableBolsa(
		tipoMaterial,
		peticionRef,
		materialPeticion,
		materialRespuesta,
		procedencia,
	)
	comprobante, err := v.reautenticar(
		ctx,
		evidencia,
		materialPeticion,
		materialRespuesta,
		instante,
	)
	return comprobante, evidencia, err
}

func (v *VerificadorEvidenciaIntegracionBolsa) reautenticar(
	ctx context.Context,
	evidencia EvidenciaDurableIntegracionBolsa,
	materialPeticion []byte,
	materialRespuesta []byte,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if ctx == nil || v == nil || dependenciaIntegracionBolsaNula(v.verificador) ||
		evidencia.Validar() != nil || !instanteBolsaCanonico(instante) ||
		instante.Before(evidencia.EmitidaEn) ||
		!instante.Before(evidencia.RetenerHasta) ||
		evidencia.AutoridadRef != v.autoridadEsperada ||
		!v.anillo.contiene(evidencia.ClaveVerificacionRef) ||
		!huellasBolsaIguales(
			evidencia.HuellaPeticionSHA256,
			huellaBytesBolsa(materialPeticion),
		) ||
		!huellasBolsaIguales(
			evidencia.HuellaRespuestaSHA256,
			huellaBytesBolsa(materialRespuesta),
		) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	if err := ctx.Err(); err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	materialFirmado := materialAutenticacionRespuestaBolsa(evidencia, materialRespuesta)
	defer borrarBytesIntegracionBolsa(materialFirmado)
	err := v.verificador.VerificarDatos(
		ctx,
		evidencia.ClaveVerificacionRef,
		materialFirmado,
		evidencia.SelloHMAC,
	)
	if err != nil {
		if ctx.Err() != nil {
			return ComprobanteEvidenciaIntegracionBolsa{}, ctx.Err()
		}
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return ComprobanteEvidenciaIntegracionBolsa{
		datos: &datosComprobanteEvidenciaBolsa{
			evidencia: evidencia, verificadaEn: instante,
		},
	}, nil
}

func nuevaEvidenciaDurableBolsa(
	tipoMaterial string,
	peticionRef string,
	materialPeticion []byte,
	materialRespuesta []byte,
	procedencia ProcedenciaIntegracionBolsa,
) EvidenciaDurableIntegracionBolsa {
	return EvidenciaDurableIntegracionBolsa{
		Esquema:               esquemaEvidenciaDurableIntegracionBolsa,
		TipoMaterial:          tipoMaterial,
		AutoridadRef:          procedencia.AutoridadRef,
		ClaveVerificacionRef:  procedencia.Evidencia.ClaveVerificacionRef,
		EvidenciaRef:          procedencia.Evidencia.EvidenciaRef,
		PeticionRef:           peticionRef,
		HuellaPeticionSHA256:  huellaBytesBolsa(materialPeticion),
		RespuestaRef:          procedencia.RespuestaRef,
		HuellaRespuestaSHA256: huellaBytesBolsa(materialRespuesta),
		SelloHMAC:             procedencia.Evidencia.SelloHMAC,
		EmitidaEn:             procedencia.Evidencia.EmitidaEn,
		ValidaHasta:           procedencia.Evidencia.ValidaHasta,
		RetenerHasta:          procedencia.Evidencia.RetenerHasta,
	}
}

func materialAutenticacionRespuestaBolsa(
	evidencia EvidenciaDurableIntegracionBolsa,
	materialRespuesta []byte,
) []byte {
	c := nuevoCanonicoBolsa("autenticacion-respuesta-bolsa")
	c.campo("tipo_material", evidencia.TipoMaterial)
	c.campo("autoridad_ref", evidencia.AutoridadRef)
	c.campo("clave_verificacion_ref", evidencia.ClaveVerificacionRef)
	c.campo("evidencia_ref", evidencia.EvidenciaRef)
	c.campo("peticion_ref", evidencia.PeticionRef)
	c.campo("huella_peticion_sha256", evidencia.HuellaPeticionSHA256)
	c.campo("respuesta_ref", evidencia.RespuestaRef)
	c.campo("huella_respuesta_sha256", huellaBytesBolsa(materialRespuesta))
	c.instante("emitida_en", evidencia.EmitidaEn)
	c.instante("valida_hasta", evidencia.ValidaHasta)
	c.instante("retener_hasta", evidencia.RetenerHasta)
	return c.bytes()
}

func claveCanonicaBolsa(valor string) bool {
	if valor == "" || len(valor) > 80 {
		return false
	}
	for indice, caracter := range valor {
		if (caracter >= 'a' && caracter <= 'z') ||
			(indice > 0 && caracter >= '0' && caracter <= '9') ||
			(indice > 0 && (caracter == '_' || caracter == '-')) {
			continue
		}
		return false
	}
	return true
}

// HuellaEvidenciaDurableIntegracionBolsa permite indexar la evidencia sin
// registrar el sello ni referencias personales en logs.
func HuellaEvidenciaDurableIntegracionBolsa(
	evidencia EvidenciaDurableIntegracionBolsa,
) (string, error) {
	if evidencia.Validar() != nil {
		return "", ErrEvidenciaBolsaNoAutenticada
	}
	contenido, err := json.Marshal(evidencia)
	if err != nil {
		return "", ErrEvidenciaBolsaNoAutenticada
	}
	defer borrarBytesIntegracionBolsa(contenido)
	return huellaBytesBolsa(contenido), nil
}

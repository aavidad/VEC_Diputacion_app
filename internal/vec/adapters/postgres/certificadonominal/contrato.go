// Package certificadonominal publica o retira, desde vec-admin, el vínculo de
// un certificado de firmante con su persona (CA25), consumiendo una decisión V3
// de la sesión ADMIN con la fachada v4 de AD205.
package certificadonominal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Coordenadas que AD165 y AD205 exigen a la decisión V3; no son configurables.
const (
	AccionPublicar    = "administracion.certificados.nominal.publicar"
	AccionRetirar     = "administracion.certificados.nominal.retirar"
	Audiencia         = "vec_contexto_actor.certificado_nominal.publicar.v1"
	modulo            = "administracion"
	tipo              = "vinculo_certificado_nominal"
	finalidad         = "gestionar_certificados_firmantes"
	esquemaDescriptor = "vec.contexto-actor.certificado-firmante.publicacion.v2"
	// MaximoDescriptor es el límite de AD205 (2..16384 octetos).
	MaximoDescriptor = 16384

	operarSQL = `SELECT vec_autorizacion.operar_certificado_nominal_v4($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)::text`
)

var (
	errDescriptor = errors.New("vec: descriptor de certificado nominal no válido")

	clavesDescriptor = []string{"esquema", "clave", "vinculo_ref", "version", "certificado_der_sha256", "cuenta_ref",
		"persona_ref", "vinculo_cuenta_persona_ref", "organizacion_ref", "estado", "vigente_desde", "vigente_hasta",
		"evidencia_ref", "evidencia_sha256", "preimagen_ref", "preimagen_version", "preimagen_sha256"}
	hex64        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	hex32        = regexp.MustCompile(`^[0-9a-f]{32}$`)
	organizacion = regexp.MustCompile(`^org_[a-z0-9]{16,80}$`)
	unidad       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$`)
	auditoria    = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
	ceros        = strings.Repeat("0", 64)
)

// Contrato es el efecto nominal de vec-admin para certificados de firmante.
func Contrato() efecto.Contrato {
	return efecto.Contrato{
		Audiencia: Audiencia, Modulo: modulo, Tipo: tipo, Finalidad: finalidad, Campos: nil,
		Acciones:      []string{AccionPublicar, AccionRetirar},
		AccionIntento: AccionPublicar, PrefijoIntento: "solicitud_certificado_nominal",
		Recurso: Recurso, Sentencia: operarSQL, Recibo: recibo,
	}
}

// Recurso deriva del descriptor exacto la acción (por su estado) y el recurso
// que AD205 exige: «certificado-nominal:<sha256 del DER>», con la organización
// y la unidad únicas de la asignación del administrador y el SHA-256 del
// descriptor. La organización del destino debe ser la de la asignación. No
// autoriza nada: el PDP decide y AD205 recalcula lo mismo en la transacción.
func Recurso(material []byte, asignacion vd.AsignacionPerfil) (string, vd.RecursoAutorizable, error) {
	var cero vd.RecursoAutorizable
	org, uni, ok := ambitosUnicos(asignacion)
	if !ok || len(material) < 2 || len(material) > MaximoDescriptor || !utf8.Valid(material) ||
		bytes.Contains(material, []byte(`\u0000`)) || !json.Valid(material) {
		return "", cero, errDescriptor
	}
	var d map[string]json.RawMessage
	if json.Unmarshal(material, &d) != nil || len(d) != len(clavesDescriptor) {
		return "", cero, errDescriptor
	}
	for _, clave := range clavesDescriptor {
		if _, ok := d[clave]; !ok {
			return "", cero, errDescriptor
		}
	}
	var esquema, clave, der, estado, destino string
	if cadena(d["esquema"], &esquema) != nil || esquema != esquemaDescriptor ||
		cadena(d["clave"], &clave) != nil || !hex32.MatchString(clave) ||
		cadena(d["certificado_der_sha256"], &der) != nil || !hex64.MatchString(der) || der == ceros ||
		cadena(d["organizacion_ref"], &destino) != nil || !organizacion.MatchString(destino) ||
		cadena(d["estado"], &estado) != nil {
		return "", cero, errDescriptor
	}
	if destino != org {
		return "", cero, efecto.ErrAmbitoNoCubierto
	}
	accion := map[string]string{"vigente": AccionPublicar, "retirado": AccionRetirar}[estado]
	if accion == "" {
		return "", cero, errDescriptor
	}
	suma := sha256.Sum256(material)
	return accion, vd.RecursoAutorizable{
		Referencia: "certificado-nominal:" + der, ModuloID: modulo, Tipo: tipo,
		Ambitos:   map[string]string{"organizacion_ref": org, "unidad_ref": uni},
		Atributos: map[string]string{"descriptor_sha256": hex.EncodeToString(suma[:])},
	}, nil
}

// ambitosUnicos exige exactamente organización y unidad, con un valor cada una.
func ambitosUnicos(a vd.AsignacionPerfil) (string, string, bool) {
	if len(a.Ambitos) != 2 {
		return "", "", false
	}
	valores := map[string]string{}
	for _, x := range a.Ambitos {
		if len(x.Valores) != 1 {
			return "", "", false
		}
		valores[x.Clave] = x.Valores[0]
	}
	org, uni := valores["organizacion_ref"], valores["unidad_ref"]
	return org, uni, len(valores) == 2 && organizacion.MatchString(org) && unidad.MatchString(uni)
}

func cadena(b json.RawMessage, destino *string) error {
	if len(b) == 0 || b[0] != '"' {
		return errDescriptor
	}
	return json.Unmarshal(b, destino)
}

// Recibo es lo que ve el administrador: el recibo de CA25 (en un reintento,
// el original) y si es nuevo o recuperado.
type Recibo struct {
	Esquema              string              `json:"esquema"`
	Clave                string              `json:"clave"`
	VinculoRef           string              `json:"vinculo_ref"`
	Version              json.Number         `json:"version"`
	CertificadoDERSHA256 string              `json:"certificado_der_sha256"`
	Estado               string              `json:"estado"`
	DecisionRef          string              `json:"decision_ref"`
	AuditoriaRef         string              `json:"auditoria_ref"`
	ReciboRef            string              `json:"recibo_ref"`
	DescriptorSHA256     string              `json:"descriptor_sha256"`
	OrganizacionDestino  OrganizacionDestino `json:"organizacion_destino"`
	RegistradaEn         string              `json:"registrada_en"`
	EstadoReplay         string              `json:"estado_replay,omitempty"`
}

// OrganizacionDestino es lo que el administrador ve de la acreditación de la
// organización del destino: la organización y su versión. Las referencias de
// vínculo corporativo y procedencia que devuelve CA25 se leen y se descartan.
type OrganizacionDestino struct {
	OrganizacionRef     string      `json:"organizacion_ref"`
	OrganizacionVersion json.Number `json:"organizacion_version"`
}

type organizacionDestinoCA25 struct {
	OrganizacionRef                string      `json:"organizacion_ref"`
	OrganizacionVersion            json.Number `json:"organizacion_version"`
	OrganizacionHuellaSHA256       string      `json:"organizacion_huella_sha256"`
	OrganizacionProcedenciaRef     string      `json:"organizacion_procedencia_ref"`
	OrganizacionProcedenciaVersion json.Number `json:"organizacion_procedencia_version"`
	OrganizacionProcedenciaSHA256  string      `json:"organizacion_procedencia_sha256"`
	VinculoCorporativoRef          string      `json:"vinculo_corporativo_ref"`
	VinculoCorporativoVersion      json.Number `json:"vinculo_corporativo_version"`
	VinculoCorporativoHuellaSHA256 string      `json:"vinculo_corporativo_huella_sha256"`
}

// recibo lee la respuesta exacta de AD205: el recibo de CA25 y el consumo de
// este acceso. Es «recuperada» si el recibo es de otra decisión.
func recibo(bruto []byte, accion string, recurso vd.RecursoAutorizable) (efecto.Recibo, error) {
	var r struct {
		Recibo struct {
			Recibo
			OrganizacionDestino organizacionDestinoCA25 `json:"organizacion_destino"`
		} `json:"recibo"`
		Consumo struct {
			DecisionRef  string `json:"decision_ref"`
			AuditoriaRef string `json:"auditoria_ref"`
		} `json:"consumo"`
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF {
		return efecto.Recibo{}, errDescriptor
	}
	p, c, o := r.Recibo.Recibo, r.Consumo, r.Recibo.OrganizacionDestino
	p.OrganizacionDestino = OrganizacionDestino{OrganizacionRef: o.OrganizacionRef, OrganizacionVersion: o.OrganizacionVersion}
	estado := map[string]string{AccionPublicar: "vigente", AccionRetirar: "retirado"}[accion]
	if p.Esquema != "vec.contexto-actor.certificado-firmante.recibo.v2" || !hex32.MatchString(p.Clave) ||
		"certificado-nominal:"+p.CertificadoDERSHA256 != recurso.Referencia || p.Estado != estado ||
		p.DescriptorSHA256 != recurso.Atributos["descriptor_sha256"] || p.ReciboRef != "recibo_certificado_nominal:"+p.Clave ||
		!auditoria.MatchString(p.AuditoriaRef) || p.DecisionRef == "" || p.RegistradaEn == "" || p.EstadoReplay != "" ||
		!auditoria.MatchString(c.AuditoriaRef) || c.DecisionRef == "" ||
		o.OrganizacionRef != recurso.Ambitos["organizacion_ref"] || o.OrganizacionVersion == "" {
		return efecto.Recibo{}, errDescriptor
	}
	switch {
	case c.DecisionRef == p.DecisionRef && c.AuditoriaRef == p.AuditoriaRef:
		p.EstadoReplay = "nueva"
	case c.DecisionRef != p.DecisionRef && c.AuditoriaRef != p.AuditoriaRef:
		p.EstadoReplay = "recuperada"
	default:
		return efecto.Recibo{}, errDescriptor
	}
	cuerpo, err := json.Marshal(p)
	if err != nil {
		return efecto.Recibo{}, errDescriptor
	}
	return efecto.Recibo{Cuerpo: cuerpo, ConsumoAuditoriaRef: c.AuditoriaRef}, nil
}

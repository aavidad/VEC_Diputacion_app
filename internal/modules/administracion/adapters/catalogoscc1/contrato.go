// Package catalogoscc1 liga el gobierno de catálogos CC1 al efecto nominal de vec-admin.
package catalogoscc1

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const (
	Audiencia      = "vec.catalogos.cc1.gobierno.v1"
	AccionCrear    = "vec.catalogos.crear"
	AccionPublicar = "vec.catalogos.publicar"
	MaximoMaterial = 1 << 20
	esquema        = "vec.catalogos.cc1.gobierno.v1"
	sentencia      = `SELECT vec_catalogos_configurables.gobernar_publicacion_admin_v1($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)::text`
)

var (
	errMaterial   = errors.New("catalogos cc1: material o recibo inválido")
	identificador = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)
	referencia    = regexp.MustCompile(`^cc1_[0-9a-f]{32}$`)
	huella        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	auditoria     = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
)

// Contrato fija las coordenadas que el PDP y la fachada SQL comprueban.
func Contrato() efecto.Contrato {
	return efecto.Contrato{
		Audiencia: Audiencia, Modulo: "administracion", Tipo: "catalogo_configurable",
		Finalidad: "administrar_aplicacion", Campos: []string{},
		Acciones: []string{AccionCrear, AccionPublicar}, AccionIntento: AccionCrear,
		PrefijoIntento: "solicitud_catalogo_cc1", Recurso: Recurso,
		Sentencia: sentencia, Recibo: recibo,
	}
}

type materialGobierno struct {
	Esquema               string                                             `json:"esquema"`
	Operacion             string                                             `json:"operacion"`
	PropuestaRef          string                                             `json:"propuesta_ref"`
	OrganizacionRef       string                                             `json:"organizacion_ref"`
	UnidadRef             string                                             `json:"unidad_ref"`
	CatalogoID            string                                             `json:"catalogo_id"`
	Version               int                                                `json:"version"`
	DocumentoCanonico     string                                             `json:"documento_canonico"`
	PreimagenesControl    map[string]vd.PreimagenControlGobiernoCategoriaRPT `json:"preimagenes_control"`
	PropuestaHuellaSHA256 *string                                            `json:"propuesta_huella_sha256"`
}

type documento struct {
	ID       string            `json:"id"`
	Version  int               `json:"version"`
	Estado   string            `json:"estado"`
	Entradas []json.RawMessage `json:"entradas"`
}

// Recurso deriva la acción y el contexto del material exacto. La asignación
// nunca se toma del cliente: el emisor y PostgreSQL cotejan sus ámbitos.
func Recurso(bruto []byte, _ vd.AsignacionPerfil) (string, vd.RecursoAutorizable, error) {
	var vacio vd.RecursoAutorizable
	if len(bruto) < 2 || len(bruto) > MaximoMaterial || !utf8.Valid(bruto) || bytes.IndexByte(bruto, 0) >= 0 {
		return "", vacio, errMaterial
	}
	var m materialGobierno
	if decodificar(bruto, &m) != nil || m.Esquema != esquema || !referencia.MatchString(m.PropuestaRef) ||
		!identificador.MatchString(m.OrganizacionRef) || !identificador.MatchString(m.UnidadRef) ||
		!identificador.MatchString(m.CatalogoID) || m.Version < 1 || m.Version > 1<<31-1 ||
		m.PreimagenesControl == nil || len(m.PreimagenesControl) > 10000 ||
		len(m.DocumentoCanonico) < 2 || len(m.DocumentoCanonico) > MaximoMaterial ||
		!utf8.ValidString(m.DocumentoCanonico) || strings.ContainsRune(m.DocumentoCanonico, 0) {
		return "", vacio, errMaterial
	}
	var accion string
	switch m.Operacion {
	case "proponer":
		accion = AccionCrear
		if m.PropuestaHuellaSHA256 != nil {
			return "", vacio, errMaterial
		}
	case "publicar":
		accion = AccionPublicar
		if m.PropuestaHuellaSHA256 == nil || !huella.MatchString(*m.PropuestaHuellaSHA256) {
			return "", vacio, errMaterial
		}
	default:
		return "", vacio, errMaterial
	}
	for clave, p := range m.PreimagenesControl {
		if !identificador.MatchString(clave) || p.Version < 1 || p.Version >= m.Version ||
			p.Revision < 1 || !huella.MatchString(p.HuellaSHA256) ||
			(p.Estado != "habilitada" && p.Estado != "deshabilitada") {
			return "", vacio, errMaterial
		}
	}
	var d documento
	if decodificar([]byte(m.DocumentoCanonico), &d) != nil || d.ID != m.CatalogoID ||
		d.Version != m.Version || d.Estado != "publicado" || len(d.Entradas) < 1 || len(d.Entradas) > 10000 {
		return "", vacio, errMaterial
	}
	vistas := make(map[string]bool, len(d.Entradas))
	for _, brutoEntrada := range d.Entradas {
		var e map[string]json.RawMessage
		if json.Unmarshal(brutoEntrada, &e) != nil || e == nil {
			return "", vacio, errMaterial
		}
		var clave, etiqueta string
		if len(e["clave"]) < 2 || e["clave"][0] != '"' || json.Unmarshal(e["clave"], &clave) != nil ||
			len(e["etiqueta"]) < 2 || e["etiqueta"][0] != '"' || json.Unmarshal(e["etiqueta"], &etiqueta) != nil ||
			!identificador.MatchString(clave) || vistas[clave] || len(etiqueta) < 1 ||
			len(etiqueta) > 2048 || strings.TrimSpace(etiqueta) != etiqueta || strings.ContainsRune(etiqueta, 0) {
			return "", vacio, errMaterial
		}
		vistas[clave] = true
	}
	suma := sha256.Sum256(bruto)
	return accion, vd.RecursoAutorizable{
		Referencia: m.PropuestaRef, ModuloID: "administracion", Tipo: "catalogo_configurable",
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef, "unidad_ref": m.UnidadRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(suma[:]), "operacion": m.Operacion, "catalogo_id": m.CatalogoID},
	}, nil
}

// decodificar rechaza campos desconocidos, claves repetidas a cualquier nivel y cola.
func decodificar(bruto []byte, destino any) error {
	if !json.Valid(bruto) || clavesUnicas(json.NewDecoder(bytes.NewReader(bruto))) != nil {
		return errMaterial
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return errMaterial
	}
	return nil
}

func clavesUnicas(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	x, ok := t.(json.Delim)
	if !ok {
		if s, ok := t.(string); ok && strings.ContainsRune(s, 0) {
			return errMaterial
		}
		return nil
	}
	switch x {
	case '{':
		vistas := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := k.(string)
			if !ok || vistas[s] || strings.ContainsRune(s, 0) {
				return errMaterial
			}
			vistas[s] = true
			if clavesUnicas(d) != nil {
				return errMaterial
			}
		}
	case '[':
		for d.More() {
			if clavesUnicas(d) != nil {
				return errMaterial
			}
		}
	default:
		return errMaterial
	}
	_, err = d.Token()
	return err
}

type reciboPublicacion struct {
	PropuestaRef          string    `json:"propuesta_ref"`
	CatalogoID            string    `json:"catalogo_id"`
	Version               int       `json:"version"`
	Estado                string    `json:"estado"`
	HuellaPropuestaSHA256 string    `json:"huella_propuesta_sha256"`
	PublicacionSHA256     *string   `json:"publicacion_sha256"`
	ReciboRef             string    `json:"recibo_ref"`
	RegistradaEn          time.Time `json:"registrada_en"`
	AuditoriaRef          string    `json:"auditoria_ref"`
}

func recibo(bruto []byte, accion string, recurso vd.RecursoAutorizable) (efecto.Recibo, error) {
	var r struct {
		Recibo              reciboPublicacion `json:"recibo"`
		ConsumoAuditoriaRef string            `json:"consumo_auditoria_ref"`
	}
	if decodificar(bruto, &r) != nil {
		return efecto.Recibo{}, errMaterial
	}
	p := r.Recibo
	estado := map[string]string{AccionCrear: "propuesta", AccionPublicar: "publicado"}[accion]
	if estado == "" || p.Estado != estado || p.PropuestaRef != recurso.Referencia ||
		p.CatalogoID != recurso.Atributos["catalogo_id"] || p.Version < 1 ||
		!huella.MatchString(p.HuellaPropuestaSHA256) || !referencia.MatchString(p.PropuestaRef) ||
		len(p.ReciboRef) < 3 || len(p.ReciboRef) > 160 || strings.ContainsRune(p.ReciboRef, 0) ||
		p.RegistradaEn.IsZero() || !auditoria.MatchString(p.AuditoriaRef) || !auditoria.MatchString(r.ConsumoAuditoriaRef) ||
		(estado == "propuesta" && p.PublicacionSHA256 != nil) ||
		(estado == "publicado" && (p.PublicacionSHA256 == nil || !huella.MatchString(*p.PublicacionSHA256))) {
		return efecto.Recibo{}, errMaterial
	}
	cuerpo, err := json.Marshal(p)
	if err != nil {
		return efecto.Recibo{}, errMaterial
	}
	return efecto.Recibo{Cuerpo: cuerpo, ConsumoAuditoriaRef: r.ConsumoAuditoriaRef}, nil
}

package plannominal

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"unicode/utf8"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Coordenadas fijas del gobierno del plan nominal de firma. Deben coincidir con
// la rama del núcleo de AD178 y con la fachada AD177; no son configurables.
const (
	EsquemaMaterialGobiernoPlanFirma = "vec.catalogos.plan-firma.gobierno.v1"
	ModuloGobiernoPlanFirma          = "contratacion_temporal"
	TipoRecursoGobiernoPlanFirma     = "catalogo_configurable"
	FinalidadGobiernoPlanFirma       = "gestionar_contratacion_temporal"
	AudienciaGobiernoPlanFirma       = "vec_catalogos_configurables.plan_nominal_firma.gobierno.v1"
	maximoMaterialGobiernoPlanFirma  = 4 << 20
)

var (
	clavesMaterialGobiernoPlanFirma = []string{"esquema", "operacion", "catalogo_id", "version",
		"revision_esperada", "huella_esperada", "clave_operacion", "catalogo_canonico_base64",
		"catalogo_sha256", "traza_canonica_base64", "traza_sha256", "evento_canonico_base64", "evento_sha256"}
	idCatalogoGobiernoPlanFirma = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)
	enteroGobiernoPlanFirma     = regexp.MustCompile(`^[1-9][0-9]{0,9}$`)
	sha256GobiernoPlanFirma     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// Mismos formatos que exige AD201 para los ámbitos del recurso.
	organizacionGobiernoPlanFirma = regexp.MustCompile(`^org_[a-z0-9]{16,80}$`)
	unidadGobiernoPlanFirma       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$`)
	estadoPorOperacionGobierno    = map[string]string{
		"crear": "borrador", "actualizar": "borrador", "publicar": "publicado", "retirar": "retirado",
	}
)

// AmbitoGobiernoPlanFirma son los ámbitos de la asignación del administrador
// que gobierna el plan. El PDP común exige que el recurso tenga exactamente
// las dimensiones de esa asignación; AD201 los recibe aparte del material.
type AmbitoGobiernoPlanFirma struct {
	OrganizacionRef string
	UnidadRef       string
}

// RecursoGobiernoPlanFirma deriva, de los bytes exactos que conserva el kit
// (vec-plan-firma-validar preparar), la acción y el recurso que la fachada
// AD201 exige a la decisión V3: recurso «catalogo_id:version», tipo
// catalogo_configurable, con la organización y la unidad del administrador y
// la huella de estado, revisión y SHA-256 de esos mismos bytes. No autoriza
// nada: el PDP decide después, y AD201/CC7 vuelven a calcular lo mismo en la
// transacción del efecto.
func RecursoGobiernoPlanFirma(material []byte, ambito AmbitoGobiernoPlanFirma) (string, vd.RecursoAutorizable, error) {
	var cero vd.RecursoAutorizable
	if !organizacionGobiernoPlanFirma.MatchString(ambito.OrganizacionRef) || !unidadGobiernoPlanFirma.MatchString(ambito.UnidadRef) {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	if len(material) < 2 || len(material) > maximoMaterialGobiernoPlanFirma || !utf8.Valid(material) ||
		bytes.Contains(material, []byte(`\u0000`)) || !json.Valid(material) {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(material, &m); err != nil || len(m) != len(clavesMaterialGobiernoPlanFirma) ||
		claveRepetidaEnObjeto(material) != nil {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	for _, clave := range clavesMaterialGobiernoPlanFirma {
		if _, ok := m[clave]; !ok {
			return "", cero, ct.ErrPlanCompetenciaFirmaV2
		}
	}
	var esquema, operacion, catalogoID, catalogoB64, catalogoSHA string
	var version json.Number
	if cadenaJSON(m["esquema"], &esquema) != nil || esquema != EsquemaMaterialGobiernoPlanFirma ||
		cadenaJSON(m["operacion"], &operacion) != nil || estadoPorOperacionGobierno[operacion] == "" ||
		cadenaJSON(m["catalogo_id"], &catalogoID) != nil || !idCatalogoGobiernoPlanFirma.MatchString(catalogoID) ||
		numeroJSON(m["version"], &version) != nil || !enteroGobiernoPlanFirma.MatchString(version.String()) ||
		cadenaJSON(m["catalogo_canonico_base64"], &catalogoB64) != nil ||
		cadenaJSON(m["catalogo_sha256"], &catalogoSHA) != nil || !sha256GobiernoPlanFirma.MatchString(catalogoSHA) {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	canon, err := base64.StdEncoding.DecodeString(catalogoB64)
	if err != nil || hexSHA256(canon) != catalogoSHA || !utf8.Valid(canon) ||
		bytes.Contains(canon, []byte(`\u0000`)) || claveRepetidaEnObjeto(canon) != nil {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	// Claves exactas, como jsonb (encoding/json emparejaría sin mayúsculas), y
	// json.Unmarshal rechaza texto de sobra tras el objeto.
	var c map[string]json.RawMessage
	var id, modulo, estado string
	var versionCatalogo, revision json.Number
	if json.Unmarshal(canon, &c) != nil ||
		cadenaJSON(c["id"], &id) != nil || id != catalogoID ||
		numeroJSON(c["version"], &versionCatalogo) != nil || versionCatalogo.String() != version.String() ||
		cadenaJSON(c["modulo_id"], &modulo) != nil || modulo != ModuloGobiernoPlanFirma ||
		numeroJSON(c["revision"], &revision) != nil || !enteroGobiernoPlanFirma.MatchString(revision.String()) ||
		cadenaJSON(c["estado"], &estado) != nil || estado != estadoPorOperacionGobierno[operacion] {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	r := vd.RecursoAutorizable{
		Referencia: catalogoID + ":" + version.String(),
		ModuloID:   ModuloGobiernoPlanFirma,
		Tipo:       TipoRecursoGobiernoPlanFirma,
		Ambitos:    map[string]string{"organizacion_ref": ambito.OrganizacionRef, "unidad_ref": ambito.UnidadRef},
		Atributos: map[string]string{
			"estado":          estado,
			"material_sha256": hexSHA256(material),
			"revision":        revision.String(),
		},
	}
	if r.Validar() != nil {
		return "", cero, ct.ErrPlanCompetenciaFirmaV2
	}
	return "vec.catalogos." + operacion, r, nil
}

func cadenaJSON(b json.RawMessage, destino *string) error {
	if len(b) == 0 || b[0] != '"' {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	return json.Unmarshal(b, destino)
}

// numeroJSON sólo admite un número JSON literal (no una cadena numérica).
func numeroJSON(b json.RawMessage, destino *json.Number) error {
	if len(b) == 0 || b[0] < '0' || b[0] > '9' {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	return d.Decode(destino)
}

// claveRepetidaEnObjeto recorre el primer nivel: AD177 compara el recuento
// json con el jsonb, así que un material con claves repetidas se rechaza ya aquí.
// Devuelve el error de lectura o ErrPlanCompetenciaFirmaV2 si hay repetidas.
func claveRepetidaEnObjeto(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	vistas := map[string]bool{}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return err
		}
		clave, ok := t.(string)
		if !ok || vistas[clave] {
			return ct.ErrPlanCompetenciaFirmaV2
		}
		vistas[clave] = true
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return err
		}
	}
	return nil
}

func hexSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// AmbitoGobiernoPlanFirmaDeAsignacion toma organización y unidad de la
// asignación del administrador que gobierna el plan. Exige exactamente esas dos
// dimensiones con un único valor cada una: con varias unidades no se elige
// ninguna por la petición.
func AmbitoGobiernoPlanFirmaDeAsignacion(a vd.AsignacionPerfil) (AmbitoGobiernoPlanFirma, error) {
	var ambito AmbitoGobiernoPlanFirma
	if len(a.Ambitos) != 2 {
		return ambito, ct.ErrPlanCompetenciaFirmaV2
	}
	for _, x := range a.Ambitos {
		if len(x.Valores) != 1 {
			return AmbitoGobiernoPlanFirma{}, ct.ErrPlanCompetenciaFirmaV2
		}
		switch x.Clave {
		case "organizacion_ref":
			ambito.OrganizacionRef = x.Valores[0]
		case "unidad_ref":
			ambito.UnidadRef = x.Valores[0]
		}
	}
	if !organizacionGobiernoPlanFirma.MatchString(ambito.OrganizacionRef) || !unidadGobiernoPlanFirma.MatchString(ambito.UnidadRef) {
		return AmbitoGobiernoPlanFirma{}, ct.ErrPlanCompetenciaFirmaV2
	}
	return ambito, nil
}

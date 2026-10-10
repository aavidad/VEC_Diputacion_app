package inscripcion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
)

const EsquemaMaterialPresentacion = "vec.bolsa.inscripcion.presentar.v1"
const EsquemaMaterialDecision = "vec.bolsa.inscripcion.decidir.v1"
const EsquemaMaterialIncorporacion = "vec.bolsa.inscripcion.incorporar.v1"

// MaterialPresentacion construye una sola serialización canónica para la
// decisión V3 y el efecto PostgreSQL. Su huella se liga al recurso y a la
// capacidad; ningún adaptador vuelve a serializar este comando.
func MaterialPresentacion(p Presentacion) ([]byte, string, error) {
	if err := p.Validar(); err != nil {
		return nil, "", err
	}
	declaraciones := append([]Declaracion{}, p.Declaraciones...)
	sort.Slice(declaraciones, func(i, j int) bool { return declaraciones[i].RequisitoCodigo < declaraciones[j].RequisitoCodigo })
	material := struct {
		Esquema           string        `json:"esquema"`
		ConvocatoriaRef   string        `json:"convocatoria_ref"`
		CategoriaRef      string        `json:"categoria_ref"`
		CatalogoVersion   uint64        `json:"catalogo_version"`
		ClaveIdempotencia string        `json:"clave_idempotencia"`
		Declaraciones     []Declaracion `json:"declaraciones"`
	}{EsquemaMaterialPresentacion, p.ConvocatoriaRef, p.CategoriaRef, p.CatalogoVersion, p.ClaveIdempotencia, declaraciones}
	contenido, err := json.Marshal(material)
	if err != nil {
		return nil, "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(huella[:]), nil
}

func ReferenciaSolicitud(personaRef string, p Presentacion) (string, error) {
	if !referenciaOpaca.MatchString(personaRef) || p.Validar() != nil {
		return "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256([]byte(personaRef + "\x1f" + p.ConvocatoriaRef + "\x1f" + p.CategoriaRef))
	return "solicitud_inscripcion_" + hex.EncodeToString(huella[:]), nil
}

// Los ámbitos llegan de ContextoActor o de la fuente histórica del recurso
// después de resolver la sesión. Los selectores del formulario son atributos.
func recursoEscrituraInscripcion(ambitos, atributos map[string]string) ([]byte, error) {
	if len(ambitos) == 0 || len(atributos) == 0 {
		return nil, ErrAccesoDenegado
	}
	for clave, valor := range ambitos {
		if !referenciaOpaca.MatchString(clave) || !referenciaOpaca.MatchString(valor) {
			return nil, ErrAccesoDenegado
		}
	}
	for clave, valor := range atributos {
		if !referenciaOpaca.MatchString(clave) || valor == "" {
			return nil, ErrSolicitudInvalida
		}
	}
	return json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{ambitos, atributos})
}

// RecursoPresentacion liga los selectores exactos al único candidato
// acreditado de la persona aspirante.
func RecursoPresentacion(p Presentacion, materialSHA256 string, ambitos map[string]string) ([]byte, error) {
	if p.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	if len(ambitos) != 1 || ambitos["candidato_ref"] == "" {
		return nil, ErrAccesoDenegado
	}
	return recursoEscrituraInscripcion(ambitos, map[string]string{
		"convocatoria_ref": p.ConvocatoriaRef, "categoria_ref": p.CategoriaRef,
		"catalogo_version": strconv.FormatUint(p.CatalogoVersion, 10),
		"material_sha256":  materialSHA256,
	})
}

func MaterialDecision(d Decision) ([]byte, string, error) {
	if d.Validar() != nil {
		return nil, "", ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		SolicitudRef      string `json:"solicitud_ref"`
		Decision          string `json:"decision"`
		MotivoCodigo      string `json:"motivo_codigo"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{EsquemaMaterialDecision, d.SolicitudRef, d.Tipo, d.MotivoCodigo, d.VersionEsperada, d.ClaveIdempotencia})
	if err != nil {
		return nil, "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(huella[:]), nil
}

func RecursoDecision(d Decision, materialSHA256 string, ambitos map[string]string,
	conjunto, solicitud AmbitoGestionInscripcion) ([]byte, error) {
	if d.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	if len(ambitos) != 2 || !conjunto.Valido(true) || !solicitud.Valido(false) ||
		ambitos["unidad_ref"] != conjunto.UnidadRef || ambitos["ambito_ref"] != conjunto.AmbitoRef ||
		solicitud.UnidadRef != conjunto.UnidadRef || solicitud.AmbitoRef != conjunto.AmbitoRef {
		return nil, ErrAccesoDenegado
	}
	atributos := map[string]string{
		"solicitud_ref": d.SolicitudRef, "decision": d.Tipo,
		"version_esperada":         strconv.FormatUint(d.VersionEsperada, 10),
		"material_sha256":          materialSHA256,
		"conjunto_ref":             conjunto.ConjuntoRef,
		"conjunto_fuente_ref":      conjunto.FuenteRef,
		"conjunto_fuente_version":  strconv.FormatUint(conjunto.FuenteVersion, 10),
		"conjunto_fuente_sha256":   conjunto.FuenteSHA256,
		"solicitud_fuente_ref":     solicitud.FuenteRef,
		"solicitud_fuente_version": strconv.FormatUint(solicitud.FuenteVersion, 10),
		"solicitud_fuente_sha256":  solicitud.FuenteSHA256,
	}
	if d.MotivoCodigo != "" {
		atributos["motivo_codigo"] = d.MotivoCodigo
	}
	return recursoEscrituraInscripcion(ambitos, atributos)
}

func MaterialIncorporacion(i Incorporacion) ([]byte, string, error) {
	if i.Validar() != nil {
		return nil, "", ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		SolicitudRef      string `json:"solicitud_ref"`
		EvidenciaRef      string `json:"evidencia_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{EsquemaMaterialIncorporacion, i.SolicitudRef, i.EvidenciaRef, i.VersionEsperada, i.ClaveIdempotencia})
	if err != nil {
		return nil, "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(huella[:]), nil
}

func RecursoIncorporacion(i Incorporacion, materialSHA256 string, ambitos map[string]string,
	conjunto, solicitud AmbitoGestionInscripcion) ([]byte, error) {
	if i.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	if len(ambitos) != 2 || !conjunto.Valido(true) || !solicitud.Valido(false) ||
		ambitos["unidad_ref"] != conjunto.UnidadRef || ambitos["ambito_ref"] != conjunto.AmbitoRef ||
		solicitud.UnidadRef != conjunto.UnidadRef || solicitud.AmbitoRef != conjunto.AmbitoRef {
		return nil, ErrAccesoDenegado
	}
	return recursoEscrituraInscripcion(ambitos, map[string]string{
		"solicitud_ref": i.SolicitudRef, "evidencia_ref": i.EvidenciaRef,
		"version_esperada":         strconv.FormatUint(i.VersionEsperada, 10),
		"material_sha256":          materialSHA256,
		"conjunto_ref":             conjunto.ConjuntoRef,
		"conjunto_fuente_ref":      conjunto.FuenteRef,
		"conjunto_fuente_version":  strconv.FormatUint(conjunto.FuenteVersion, 10),
		"conjunto_fuente_sha256":   conjunto.FuenteSHA256,
		"solicitud_fuente_ref":     solicitud.FuenteRef,
		"solicitud_fuente_version": strconv.FormatUint(solicitud.FuenteVersion, 10),
		"solicitud_fuente_sha256":  solicitud.FuenteSHA256,
	})
}

package inscripcion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
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

// RecursoPresentacion entrega los bytes exactos que deben comprometer la
// decisión y la capacidad V3. SQL reconstruye este objeto y coteja su SHA.
func RecursoPresentacion(p Presentacion, materialSHA256 string) ([]byte, error) {
	if p.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Ambitos struct {
			CategoriaRef    string `json:"categoria_ref"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
		} `json:"ambitos"`
		Atributos struct {
			MaterialSHA256 string `json:"material_sha256"`
		} `json:"atributos"`
	}{
		Ambitos: struct {
			CategoriaRef    string `json:"categoria_ref"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
		}{p.CategoriaRef, p.ConvocatoriaRef},
		Atributos: struct {
			MaterialSHA256 string `json:"material_sha256"`
		}{materialSHA256},
	})
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return contenido, nil
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

func RecursoDecision(d Decision, materialSHA256 string) ([]byte, error) {
	if d.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Ambitos struct {
			SolicitudRef string `json:"solicitud_ref"`
		} `json:"ambitos"`
		Atributos struct {
			MaterialSHA256 string `json:"material_sha256"`
		} `json:"atributos"`
	}{
		Ambitos: struct {
			SolicitudRef string `json:"solicitud_ref"`
		}{d.SolicitudRef},
		Atributos: struct {
			MaterialSHA256 string `json:"material_sha256"`
		}{materialSHA256},
	})
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return contenido, nil
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

func RecursoIncorporacion(i Incorporacion, materialSHA256 string) ([]byte, error) {
	if i.Validar() != nil || len(materialSHA256) != 64 {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Ambitos struct {
			SolicitudRef string `json:"solicitud_ref"`
		} `json:"ambitos"`
		Atributos struct {
			MaterialSHA256 string `json:"material_sha256"`
		} `json:"atributos"`
	}{
		Ambitos: struct {
			SolicitudRef string `json:"solicitud_ref"`
		}{i.SolicitudRef},
		Atributos: struct {
			MaterialSHA256 string `json:"material_sha256"`
		}{materialSHA256},
	})
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return contenido, nil
}

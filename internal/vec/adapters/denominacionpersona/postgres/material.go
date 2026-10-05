package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	modulo          = "vec"
	tipo            = "persona_denominacion"
	finalidad       = "presentacion_persona"
	esquemaPublicar = "vec.persona.denominacion.publicar.v1"
	esquemaLeer     = "vec.persona.denominacion.leer.v1"
)

type ambitos struct {
	OrganizacionRef string `json:"organizacion_ref"`
	UnidadRef       string `json:"unidad_ref"`
}
type Procedencia struct {
	Ref                     string
	Version                 uint64
	HuellaSHA256, Autoridad string
}

type publicacion struct {
	Esquema              string                         `json:"esquema"`
	PersonaRef           string                         `json:"persona_ref"`
	VersionEsperada      uint64                         `json:"version_esperada"`
	ProcedenciaRef       string                         `json:"procedencia_ref"`
	ProcedenciaVersion   uint64                         `json:"procedencia_version"`
	ProcedenciaSHA256    string                         `json:"procedencia_sha256"`
	ProcedenciaAutoridad string                         `json:"procedencia_autoridad"`
	SobreSHA256          string                         `json:"sobre_sha256"`
	Sobre                ports.SobreDenominacionPersona `json:"sobre"`
	Ambitos              ambitos                        `json:"ambitos"`
}
type lectura struct {
	Esquema    string  `json:"esquema"`
	PersonaRef string  `json:"persona_ref"`
	Version    uint64  `json:"version"`
	Ambitos    ambitos `json:"ambitos"`
}

func referencia(s string) bool {
	if len(s) < 3 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func sobreCanonico(s ports.SobreDenominacionPersona) ([]byte, error) {
	if s.Esquema != "vec.persona.denominacion.aead.v1" || !domain.ReferenciaPersonaDenominacionValida(s.PersonaRef) || len(s.PersonaRef) > 128 || s.Version == 0 || s.Version > 1<<53-1 || !referencia(s.ClaveRef) || len(s.Nonce) != 12 || len(s.Cifrado) < 17 || len(s.Cifrado) > 4112 || !referencia(s.Indice.AmbitoRef) || !referencia(s.Indice.NormaRef) || !referencia(s.Indice.ClaveRef) || len(s.Indice.NormaSHA256) != 64 || !hashValido(s.Indice.NormaSHA256) || len(s.Indice.Tokens) < 1 || len(s.Indice.Tokens) > 32 {
		return nil, ErrNoDisponible
	}
	for i, t := range s.Indice.Tokens {
		if len(t) != 32 || i > 0 && bytes.Compare(s.Indice.Tokens[i-1], t) >= 0 {
			return nil, ErrNoDisponible
		}
	}
	b, e := json.Marshal(s)
	if e != nil {
		return nil, ErrNoDisponible
	}
	return b, nil
}
func hashValido(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}
func recurso(persona string, a ambitos, b []byte) (domain.RecursoAutorizable, error) {
	if !domain.ReferenciaPersonaDenominacionValida(persona) || len(persona) > 128 || !referencia(a.OrganizacionRef) || !referencia(a.UnidadRef) {
		return domain.RecursoAutorizable{}, ErrNoDisponible
	}
	return domain.RecursoAutorizable{Referencia: persona, ModuloID: modulo, Tipo: tipo, Ambitos: map[string]string{"organizacion_ref": a.OrganizacionRef, "unidad_ref": a.UnidadRef}, Atributos: map[string]string{"material_sha256": sha(b)}}, nil
}

// Estos helpers fijan el material ANTES de pedir la capacidad al emisor V3.
// Sólo contienen ciphertext y referencias opacas; no conceden autorización.
func MaterialPublicacion(p ports.PreparacionDenominacionPersona, procedencia Procedencia, organizacion, unidad string) ([]byte, domain.RecursoAutorizable, error) {
	raw, e := sobreCanonico(p.Sobre)
	if e != nil || p.PersonaRef != p.Sobre.PersonaRef || p.VersionEsperada >= 1<<53-1 || p.Sobre.Version != p.VersionEsperada+1 || !referencia(p.ProcedenciaRef) || !strings.HasPrefix(p.ProcedenciaRef, "prc_") || len(p.ProcedenciaRef) < 26 || p.SobreSHA256 != sha(raw) || procedencia.Ref != p.ProcedenciaRef || procedencia.Version == 0 || procedencia.Version > 1<<53-1 || !hashValido(procedencia.HuellaSHA256) || (procedencia.Autoridad != "no_autoritativa" && procedencia.Autoridad != "autoridad_maestra_acreditada") {
		return nil, domain.RecursoAutorizable{}, ErrNoDisponible
	}
	a := ambitos{organizacion, unidad}
	b, e := json.Marshal(publicacion{esquemaPublicar, p.PersonaRef, p.VersionEsperada, p.ProcedenciaRef, procedencia.Version, procedencia.HuellaSHA256, procedencia.Autoridad, p.SobreSHA256, p.Sobre, a})
	if e != nil {
		return nil, domain.RecursoAutorizable{}, ErrNoDisponible
	}
	r, e := recurso(p.PersonaRef, a, b)
	if e != nil {
		return nil, domain.RecursoAutorizable{}, e
	}
	r.Atributos["procedencia_version"] = strconv.FormatUint(procedencia.Version, 10)
	r.Atributos["procedencia_sha256"] = procedencia.HuellaSHA256
	r.Atributos["procedencia_autoridad"] = procedencia.Autoridad
	if e != nil {
		return nil, domain.RecursoAutorizable{}, e
	}
	return b, r, nil
}
func MaterialLectura(persona string, version uint64, organizacion, unidad string) ([]byte, domain.RecursoAutorizable, error) {
	if version == 0 || version > 1<<53-1 {
		return nil, domain.RecursoAutorizable{}, ErrNoDisponible
	}
	a := ambitos{organizacion, unidad}
	b, e := json.Marshal(lectura{esquemaLeer, persona, version, a})
	if e != nil {
		return nil, domain.RecursoAutorizable{}, ErrNoDisponible
	}
	r, e := recurso(persona, a, b)
	if e != nil {
		return nil, domain.RecursoAutorizable{}, e
	}
	return b, r, nil
}

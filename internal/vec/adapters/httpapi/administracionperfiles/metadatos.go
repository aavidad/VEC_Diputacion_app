package administracionperfiles

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

// Esta variante contiene exclusivamente la proyección nominal de AUT43. Los
// nombres consultados se incorporan sólo después de su lectura CA32 autorizada.
type PerfilUsuarioMetadatos struct {
	PerfilRef     string    `json:"perfil_ref"`
	RolVersionRef string    `json:"rol_version_ref"`
	Version       uint64    `json:"version"`
	Estado        string    `json:"estado"`
	VigenteDesde  time.Time `json:"vigente_desde"`
	VigenteHasta  time.Time `json:"vigente_hasta"`
}

type PersonaMetadatos struct {
	PersonaRef          string                   `json:"persona_ref"`
	UnidadRef           string                   `json:"unidad_ref"`
	DenominacionVersion *uint64                  `json:"denominacion_version"`
	NombreEstado        string                   `json:"nombre_estado"`
	Nombre              string                   `json:"nombre,omitempty"`
	Perfiles            []PerfilUsuarioMetadatos `json:"perfiles"`
}

type PaginaPersonasMetadatos struct {
	Proyeccion      string             `json:"proyeccion"`
	Personas        []PersonaMetadatos `json:"personas"`
	SiguienteCursor string             `json:"siguiente_cursor,omitempty"`
}

type FichaPersonaMetadatos struct {
	Proyeccion string `json:"proyeccion"`
	PersonaMetadatos
	HistoriaEstado string `json:"historia_estado"`
	ActosEstado    string `json:"actos_estado"`
}

func (p PaginaPersonas) MarshalJSON() ([]byte, error) {
	if p.Metadatos != nil {
		return json.Marshal(p.Metadatos)
	}
	type completa PaginaPersonas
	return json.Marshal(completa(p))
}

func (f FichaPersona) MarshalJSON() ([]byte, error) {
	if f.Metadatos != nil {
		return json.Marshal(f.Metadatos)
	}
	type completa FichaPersona
	return json.Marshal(completa(f))
}

func (f FichaPersona) referenciaEmitida() string {
	if f.Metadatos != nil {
		return f.Metadatos.PersonaRef
	}
	return f.PersonaRef
}

func (p PersonaMetadatos) valida() bool {
	if !domain.ReferenciaPersonaDenominacionValida(p.PersonaRef) || len(p.PersonaRef) > 128 || p.UnidadRef == "" || p.Perfiles == nil {
		return false
	}
	switch p.NombreEstado {
	case "no_registrado":
		if p.DenominacionVersion != nil || p.Nombre != "" {
			return false
		}
	case "no_consultado":
		if p.DenominacionVersion == nil || p.Nombre != "" {
			return false
		}
	case "consultado":
		if p.DenominacionVersion == nil || !utf8.ValidString(p.Nombre) || strings.TrimSpace(p.Nombre) == "" || len(p.Nombre) > 4096 {
			return false
		}
	default:
		return false
	}
	if p.DenominacionVersion != nil && (*p.DenominacionVersion == 0 || *p.DenominacionVersion > 1<<53-1) {
		return false
	}
	return true
}

func (p PaginaPersonas) metadatosValidos() bool {
	if p.Metadatos == nil {
		return true
	}
	m := p.Metadatos
	if m.Proyeccion != "metadatos_v1" || m.Personas == nil || len(m.Personas) > 50 {
		return false
	}
	for _, persona := range m.Personas {
		if !persona.valida() {
			return false
		}
	}
	return true
}

func (f FichaPersona) metadatosValidos() bool {
	return f.Metadatos == nil || f.Metadatos.Proyeccion == "metadatos_v1" && f.Metadatos.HistoriaEstado == "no_consultada" && f.Metadatos.ActosEstado == "no_consultados" && f.Metadatos.PersonaMetadatos.valida()
}

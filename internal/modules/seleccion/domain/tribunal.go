package domain

import (
	"errors"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	basess2 "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
)

var ErrMaterialTribunalInvalido = errors.New("seleccion.tribunal.material_invalido")

// MaterialTribunalPropuesto enlaza una revisión de S2 aportada por el operador.
// Las referencias no acreditan existencia, vigencia ni lectura autorizada.
type MaterialTribunalPropuesto struct {
	Alcance           string                                    `json:"alcance"`
	IdentidadMaterial string                                    `json:"identidad_material"`
	VersionMaterial   int                                       `json:"version_material"`
	BasesS2           basess2.Esperada                          `json:"bases_s2"`
	FlujoProceso      bolsa.ReferenciaConfiguracionConvocatoria `json:"flujo_proceso"`
	ReglasBaremacion  bolsa.ReferenciaConfiguracionConvocatoria `json:"reglas_baremacion"`
	FasesPropuestas   []string                                  `json:"fases_propuestas"`
	Miembros          []MiembroTribunalPropuesto                `json:"miembros"`
	Incidencias       []IncidenciaTribunalPropuesta             `json:"incidencias"`
}

// RolRef es una propuesta opaca; no asigna cargo, permiso o habilitación.
type MiembroTribunalPropuesto struct {
	MiembroRef string   `json:"miembro_ref"`
	PersonaRef string   `json:"persona_ref"`
	RolRef     string   `json:"rol_ref"`
	Fases      []string `json:"fases"`
}

// Una incidencia conserva la propuesta y su evidencia por referencia.
// No retira ni sustituye miembros, ni resuelve la abstención o recusación.
type IncidenciaTribunalPropuesta struct {
	PropuestaRef    string                                    `json:"propuesta_ref"`
	Tipo            string                                    `json:"tipo"`
	MiembroRef      string                                    `json:"miembro_ref"`
	Fases           []string                                  `json:"fases"`
	EvidenciaRef    string                                    `json:"evidencia_ref"`
	CausaCatalogada bolsa.ReferenciaConfiguracionConvocatoria `json:"causa_catalogada"`
	SustitutoRef    string                                    `json:"sustituto_ref,omitempty"`
}

type PendienteTribunal struct {
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type PreparacionTribunal struct {
	Estado            string                    `json:"estado"`
	MaterialPropuesto MaterialTribunalPropuesto `json:"material_propuesto"`
	Pendientes        []PendienteTribunal       `json:"pendientes"`
}

// PrepararTribunal comprueba exclusivamente estructura y vínculos locales.
// Nunca infiere composición legal, elegibilidad, quórum o designación.
func PrepararTribunal(m MaterialTribunalPropuesto) (PreparacionTribunal, error) {
	if m.Alcance != "preparacion_sintetica" || !basess2.IdentificadorValido(m.IdentidadMaterial) ||
		m.VersionMaterial < 1 || m.VersionMaterial > 1_000_000 || m.BasesS2.Validar(false) != nil ||
		len(m.FasesPropuestas) > 100 || len(m.Miembros) > 100 || len(m.Incidencias) > 200 ||
		!referenciaTribunalOpcional(m.FlujoProceso) || !referenciaTribunalOpcional(m.ReglasBaremacion) {
		return PreparacionTribunal{}, ErrMaterialTribunalInvalido
	}
	fases, ok := conjuntoTribunal(m.FasesPropuestas, nil)
	if !ok {
		return PreparacionTribunal{}, ErrMaterialTribunalInvalido
	}
	miembros := make(map[string]map[string]bool, len(m.Miembros))
	personas := make(map[string]bool, len(m.Miembros))
	for _, miembro := range m.Miembros {
		asignadas, validas := conjuntoTribunal(miembro.Fases, fases)
		if !basess2.IdentificadorValido(miembro.MiembroRef) || !basess2.IdentificadorValido(miembro.PersonaRef) ||
			!basess2.IdentificadorValido(miembro.RolRef) || !validas || miembros[miembro.MiembroRef] != nil || personas[miembro.PersonaRef] {
			return PreparacionTribunal{}, ErrMaterialTribunalInvalido
		}
		miembros[miembro.MiembroRef] = asignadas
		personas[miembro.PersonaRef] = true
	}
	propuestas := make(map[string]bool, len(m.Incidencias))
	vinculos := make(map[string]map[string]string)
	destinos := make(map[string]map[string]bool)
	incidenciasVistas := make(map[string]bool)
	for _, incidencia := range m.Incidencias {
		asignadas, existe := miembros[incidencia.MiembroRef]
		_, validas := conjuntoTribunal(incidencia.Fases, asignadas)
		if !basess2.IdentificadorValido(incidencia.PropuestaRef) || propuestas[incidencia.PropuestaRef] ||
			!existe || !validas || len(incidencia.Fases) == 0 ||
			incidencia.CausaCatalogada.Validar() != nil || incidencia.CausaCatalogada.Version > 1_000_000 ||
			(incidencia.EvidenciaRef != "" && !basess2.IdentificadorValido(incidencia.EvidenciaRef)) {
			return PreparacionTribunal{}, ErrMaterialTribunalInvalido
		}
		for _, fase := range incidencia.Fases {
			clave := incidencia.Tipo + "\x00" + incidencia.MiembroRef + "\x00" + fase
			if incidenciasVistas[clave] {
				return PreparacionTribunal{}, ErrMaterialTribunalInvalido
			}
			incidenciasVistas[clave] = true
		}
		switch incidencia.Tipo {
		case "abstencion", "recusacion":
			if incidencia.SustitutoRef != "" {
				return PreparacionTribunal{}, ErrMaterialTribunalInvalido
			}
		case "sustitucion":
			sustituto, existe := miembros[incidencia.SustitutoRef]
			if !existe || incidencia.SustitutoRef == incidencia.MiembroRef {
				return PreparacionTribunal{}, ErrMaterialTribunalInvalido
			}
			if _, validas := conjuntoTribunal(incidencia.Fases, sustituto); !validas {
				return PreparacionTribunal{}, ErrMaterialTribunalInvalido
			}
			for _, fase := range incidencia.Fases {
				if vinculos[fase] == nil {
					vinculos[fase] = map[string]string{}
					destinos[fase] = map[string]bool{}
				}
				if destinos[fase][incidencia.SustitutoRef] {
					return PreparacionTribunal{}, ErrMaterialTribunalInvalido
				}
				destinos[fase][incidencia.SustitutoRef] = true
				vinculos[fase][incidencia.MiembroRef] = incidencia.SustitutoRef
			}
		default:
			return PreparacionTribunal{}, ErrMaterialTribunalInvalido
		}
		propuestas[incidencia.PropuestaRef] = true
	}
	for _, fase := range vinculos {
		for origen := range fase {
			vistos := map[string]bool{}
			for actual := origen; actual != ""; actual = fase[actual] {
				if vistos[actual] {
					return PreparacionTribunal{}, ErrMaterialTribunalInvalido
				}
				vistos[actual] = true
			}
		}
	}

	m.FasesPropuestas = append([]string{}, m.FasesPropuestas...)
	m.Miembros = append([]MiembroTribunalPropuesto{}, m.Miembros...)
	for i := range m.Miembros {
		m.Miembros[i].Fases = append([]string{}, m.Miembros[i].Fases...)
	}
	m.Incidencias = append([]IncidenciaTribunalPropuesta{}, m.Incidencias...)
	for i := range m.Incidencias {
		m.Incidencias[i].Fases = append([]string{}, m.Incidencias[i].Fases...)
	}
	pendientes := []PendienteTribunal{{"bases_s2", "referencia_no_verificada"}}
	for _, ref := range []struct {
		campo string
		valor bolsa.ReferenciaConfiguracionConvocatoria
	}{{"flujo_proceso", m.FlujoProceso}} {
		codigo := "referencia_no_verificada"
		if ref.valor == (bolsa.ReferenciaConfiguracionConvocatoria{}) {
			codigo = "referencia_ausente"
		}
		pendientes = append(pendientes, PendienteTribunal{ref.campo, codigo})
	}
	if m.ReglasBaremacion != (bolsa.ReferenciaConfiguracionConvocatoria{}) {
		pendientes = append(pendientes, PendienteTribunal{"reglas_baremacion", "referencia_no_verificada"})
	}
	for _, dato := range []struct {
		campo string
		vacio bool
	}{{"fases_propuestas", len(m.FasesPropuestas) == 0}, {"miembros", len(m.Miembros) == 0}} {
		codigo := "propuesta_no_verificada"
		if dato.vacio {
			codigo = "material_ausente"
		}
		pendientes = append(pendientes, PendienteTribunal{dato.campo, codigo})
	}
	for _, campo := range []string{"acto_designacion", "habilitacion_por_fase", "actas_y_firma"} {
		pendientes = append(pendientes, PendienteTribunal{campo, "circuito_pendiente"})
	}
	if len(m.Incidencias) > 0 {
		pendientes = append(pendientes, PendienteTribunal{"incidencias", "resolucion_pendiente"})
		pendientes = append(pendientes, PendienteTribunal{"causas_catalogadas", "referencia_no_verificada"})
		for _, incidencia := range m.Incidencias {
			if incidencia.EvidenciaRef == "" {
				pendientes = append(pendientes, PendienteTribunal{"evidencia_incidencias", "material_ausente"})
				break
			}
		}
	}
	return PreparacionTribunal{Estado: "pendiente", MaterialPropuesto: m, Pendientes: pendientes}, nil
}

func referenciaTribunalOpcional(ref bolsa.ReferenciaConfiguracionConvocatoria) bool {
	return ref == (bolsa.ReferenciaConfiguracionConvocatoria{}) ||
		(ref.Validar() == nil && ref.Version <= 1_000_000)
}

func conjuntoTribunal(valores []string, permitidos map[string]bool) (map[string]bool, bool) {
	if len(valores) > 100 {
		return nil, false
	}
	conjunto := make(map[string]bool, len(valores))
	for _, valor := range valores {
		if !basess2.IdentificadorValido(valor) || conjunto[valor] || (permitidos != nil && !permitidos[valor]) {
			return nil, false
		}
		conjunto[valor] = true
	}
	return conjunto, true
}

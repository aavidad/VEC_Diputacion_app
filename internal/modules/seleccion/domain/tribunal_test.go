package domain

import (
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	basess2 "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
)

func materialTribunalPrueba() MaterialTribunalPropuesto {
	return MaterialTribunalPropuesto{
		Alcance: "preparacion_sintetica", IdentidadMaterial: "tribunal:ejemplo", VersionMaterial: 1,
		BasesS2:         basess2.Esperada{PreparacionRef: "preparacion:ejemplo", Revision: 2, HuellaMaterialSHA256: strings.Repeat("a", 64)},
		FasesPropuestas: []string{"fase:ejemplo"},
		Miembros: []MiembroTribunalPropuesto{
			{MiembroRef: "miembro:a", PersonaRef: "persona:a", RolRef: "rol:propuesto", Fases: []string{"fase:ejemplo"}},
			{MiembroRef: "miembro:b", PersonaRef: "persona:b", RolRef: "rol:propuesto", Fases: []string{"fase:ejemplo"}},
		},
		Incidencias: []IncidenciaTribunalPropuesta{{PropuestaRef: "propuesta:a", Tipo: "sustitucion",
			MiembroRef: "miembro:a", SustitutoRef: "miembro:b", Fases: []string{"fase:ejemplo"},
			CausaCatalogada: bolsa.ReferenciaConfiguracionConvocatoria{ID: "causa:ejemplo", Version: 1, HuellaContenidoSHA256: strings.Repeat("b", 64)}}},
	}
}

func TestTribunalConservaPropuestaSinDesignar(t *testing.T) {
	m := materialTribunalPrueba()
	r, err := PrepararTribunal(m)
	if err != nil || r.Estado != "pendiente" || r.MaterialPropuesto.BasesS2 != m.BasesS2 || len(r.MaterialPropuesto.Miembros) != 2 {
		t.Fatalf("preparacion: %#v, %v", r, err)
	}
	pendientes := map[string]string{}
	for _, p := range r.Pendientes {
		pendientes[p.Campo] = p.Codigo
	}
	for _, campo := range []string{"bases_s2", "acto_designacion", "habilitacion_por_fase", "actas_y_firma", "incidencias", "causas_catalogadas", "evidencia_incidencias"} {
		if pendientes[campo] == "" {
			t.Errorf("pendiente omitido: %s", campo)
		}
	}
	if pendientes["reglas_baremacion"] != "" {
		t.Fatal("baremo opcional ausente tratado como dependencia")
	}
	m.Miembros[0].Fases[0] = "fase:alterada"
	m.Incidencias[0].Fases[0] = "fase:alterada"
	m.FasesPropuestas[0] = "fase:alterada"
	if r.MaterialPropuesto.Miembros[0].Fases[0] != "fase:ejemplo" || r.MaterialPropuesto.Incidencias[0].Fases[0] != "fase:ejemplo" || r.MaterialPropuesto.FasesPropuestas[0] != "fase:ejemplo" {
		t.Fatal("resultado comparte memoria mutable de entrada")
	}
}

func TestTribunalRechazaEnlacesAmbiguos(t *testing.T) {
	casos := map[string]func(*MaterialTribunalPropuesto){
		"alta s2":        func(m *MaterialTribunalPropuesto) { m.BasesS2.Revision = 0; m.BasesS2.HuellaMaterialSHA256 = "" },
		"huella omitida": func(m *MaterialTribunalPropuesto) { m.BasesS2.HuellaMaterialSHA256 = "" },
		"alcance real":   func(m *MaterialTribunalPropuesto) { m.Alcance = "real" },
		"fase ajena":     func(m *MaterialTribunalPropuesto) { m.Miembros[0].Fases[0] = "fase:ajena" },
		"fase duplicada": func(m *MaterialTribunalPropuesto) {
			m.Miembros[0].Fases = append(m.Miembros[0].Fases, m.Miembros[0].Fases[0])
		},
		"miembro duplicado":    func(m *MaterialTribunalPropuesto) { m.Miembros[1].MiembroRef = m.Miembros[0].MiembroRef },
		"persona duplicada":    func(m *MaterialTribunalPropuesto) { m.Miembros[1].PersonaRef = m.Miembros[0].PersonaRef },
		"sustitucion propia":   func(m *MaterialTribunalPropuesto) { m.Incidencias[0].SustitutoRef = m.Incidencias[0].MiembroRef },
		"sustituto ajeno":      func(m *MaterialTribunalPropuesto) { m.Incidencias[0].SustitutoRef = "miembro:ajeno" },
		"sustituto otra fase":  func(m *MaterialTribunalPropuesto) { m.Miembros[1].Fases = nil },
		"incidencia otra fase": func(m *MaterialTribunalPropuesto) { m.Incidencias[0].Fases[0] = "fase:ajena" },
		"causa ausente": func(m *MaterialTribunalPropuesto) {
			m.Incidencias[0].CausaCatalogada = bolsa.ReferenciaConfiguracionConvocatoria{}
		},
		"tipo inventado": func(m *MaterialTribunalPropuesto) { m.Incidencias[0].Tipo = "designacion" },
		"doble sustitucion": func(m *MaterialTribunalPropuesto) {
			p := m.Incidencias[0]
			p.PropuestaRef = "propuesta:b"
			m.Incidencias = append(m.Incidencias, p)
		},
		"ciclo": func(m *MaterialTribunalPropuesto) {
			p := m.Incidencias[0]
			p.PropuestaRef = "propuesta:b"
			p.MiembroRef, p.SustitutoRef = p.SustitutoRef, p.MiembroRef
			m.Incidencias = append(m.Incidencias, p)
		},
		"mismo sustituto fase": func(m *MaterialTribunalPropuesto) {
			m.Miembros = append(m.Miembros, MiembroTribunalPropuesto{MiembroRef: "miembro:c", PersonaRef: "persona:c", RolRef: "rol:propuesto", Fases: []string{"fase:ejemplo"}})
			p := m.Incidencias[0]
			p.PropuestaRef = "propuesta:c"
			p.MiembroRef = "miembro:c"
			m.Incidencias = append(m.Incidencias, p)
		},
		"cardinalidad": func(m *MaterialTribunalPropuesto) { m.Miembros = make([]MiembroTribunalPropuesto, 101) },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			m := materialTribunalPrueba()
			alterar(&m)
			if _, err := PrepararTribunal(m); err != ErrMaterialTribunalInvalido {
				t.Fatalf("entrada aceptada: %v", err)
			}
		})
	}
}

func TestTribunalPermiteCompletarPreparacionIncompleta(t *testing.T) {
	m := materialTribunalPrueba()
	m.FasesPropuestas, m.Miembros, m.Incidencias = nil, nil, nil
	r, err := PrepararTribunal(m)
	if err != nil || r.Estado != "pendiente" || r.MaterialPropuesto.Miembros == nil {
		t.Fatalf("preparacion incompleta: %#v %v", r, err)
	}
}

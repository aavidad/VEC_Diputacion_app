package domain

import (
	"strings"
	"testing"
)

func materialActaPrueba() MaterialActaPropuesto {
	return MaterialActaPropuesto{
		Alcance: "preparacion_sintetica", IdentidadMaterial: "acta:ejemplo", VersionMaterial: 1,
		AntecedenteTribunal: AntecedenteTribunalPropuesto{IdentidadMaterial: "tribunal:ejemplo", VersionMaterial: 2, HuellaAportadaSHA256: strings.Repeat("a", 64)},
		FasePropuesta:       "fase:ejemplo", SesionRef: "sesion:ejemplo",
		OrdenDiaPropuesto:  []PuntoSesionPropuesto{{PuntoRef: "punto:a", TextoPropuesto: "Texto sintético de agenda."}},
		AcuerdosPropuestos: []AcuerdoSesionPropuesto{{PropuestaRef: "propuesta:a", PuntoRef: "punto:a", TextoPropuesto: "Texto sintético de propuesta."}},
	}
}

func TestActaConservaTextosSinAcreditarSesion(t *testing.T) {
	m := materialActaPrueba()
	r, err := PrepararActa(m)
	if err != nil || r.Estado != "borrador_propuesto" || r.MaterialPropuesto.AntecedenteTribunal != m.AntecedenteTribunal ||
		r.MaterialPropuesto.FechaPropuesta != "" || r.MaterialPropuesto.AcuerdosPropuestos[0] != m.AcuerdosPropuestos[0] {
		t.Fatalf("borrador: %#v %v", r, err)
	}
	pendientes := map[string]string{}
	for _, p := range r.Pendientes {
		pendientes[p.Campo] = p.Codigo
	}
	for _, campo := range []string{"antecedente_tribunal", "fase_propuesta", "designacion", "habilitacion", "sesion_celebrada", "asistencia", "deliberaciones", "acuerdos_adoptados", "aprobacion", "firma", "fecha_propuesta"} {
		if pendientes[campo] == "" {
			t.Errorf("pendiente omitido: %s", campo)
		}
	}
	m.OrdenDiaPropuesto[0].TextoPropuesto = "Alterado"
	m.AcuerdosPropuestos[0].TextoPropuesto = "Alterado"
	if r.MaterialPropuesto.OrdenDiaPropuesto[0].TextoPropuesto == "Alterado" || r.MaterialPropuesto.AcuerdosPropuestos[0].TextoPropuesto == "Alterado" {
		t.Fatal("resultado comparte memoria de entrada")
	}
}

func TestActaRechazaAmbiguedadYEntradasInvalidas(t *testing.T) {
	casos := map[string]func(*MaterialActaPropuesto){
		"alcance real":            func(m *MaterialActaPropuesto) { m.Alcance = "real" },
		"antecedente sin version": func(m *MaterialActaPropuesto) { m.AntecedenteTribunal.VersionMaterial = 0 },
		"antecedente sin huella":  func(m *MaterialActaPropuesto) { m.AntecedenteTribunal.HuellaAportadaSHA256 = "" },
		"fase sin referencia":     func(m *MaterialActaPropuesto) { m.FasePropuesta = "" },
		"fecha invalida":          func(m *MaterialActaPropuesto) { m.FechaPropuesta = "2026-02-30T12:00:00Z" },
		"fecha sin zona":          func(m *MaterialActaPropuesto) { m.FechaPropuesta = "2026-10-03T12:00:00" },
		"punto duplicado": func(m *MaterialActaPropuesto) {
			m.OrdenDiaPropuesto = append(m.OrdenDiaPropuesto, m.OrdenDiaPropuesto[0])
		},
		"acuerdo duplicado": func(m *MaterialActaPropuesto) {
			m.AcuerdosPropuestos = append(m.AcuerdosPropuestos, m.AcuerdosPropuestos[0])
		},
		"acuerdo ajeno":    func(m *MaterialActaPropuesto) { m.AcuerdosPropuestos[0].PuntoRef = "punto:ajeno" },
		"control":          func(m *MaterialActaPropuesto) { m.OrdenDiaPropuesto[0].TextoPropuesto = "texto\x00" },
		"salto de linea":   func(m *MaterialActaPropuesto) { m.OrdenDiaPropuesto[0].TextoPropuesto = "texto\n" },
		"unicode invalido": func(m *MaterialActaPropuesto) { m.AcuerdosPropuestos[0].TextoPropuesto = string([]byte{0xff}) },
		"texto desmedido":  func(m *MaterialActaPropuesto) { m.AcuerdosPropuestos[0].TextoPropuesto = strings.Repeat("a", 4097) },
		"cardinalidad":     func(m *MaterialActaPropuesto) { m.OrdenDiaPropuesto = make([]PuntoSesionPropuesto, 101) },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			m := materialActaPrueba()
			alterar(&m)
			if _, err := PrepararActa(m); err != ErrMaterialActaInvalido {
				t.Fatalf("entrada aceptada: %v", err)
			}
		})
	}
}

func TestActaIncompletaYFechaSoloAportada(t *testing.T) {
	m := materialActaPrueba()
	m.OrdenDiaPropuesto, m.AcuerdosPropuestos, m.SesionRef = nil, nil, ""
	r, err := PrepararActa(m)
	if err != nil || r.MaterialPropuesto.OrdenDiaPropuesto == nil {
		t.Fatalf("incompleta: %#v %v", r, err)
	}
	pendientes := map[string]string{}
	for _, p := range r.Pendientes {
		pendientes[p.Campo] = p.Codigo
	}
	for _, campo := range []string{"sesion_ref", "fecha_propuesta", "orden_dia_propuesto", "acuerdos_propuestos"} {
		if pendientes[campo] != "material_ausente" {
			t.Errorf("dato ausente sin pendiente: %s", campo)
		}
	}
	m.FechaPropuesta = "2026-10-03T12:00:00+02:00"
	r, err = PrepararActa(m)
	if err != nil || r.MaterialPropuesto.FechaPropuesta != m.FechaPropuesta {
		t.Fatalf("fecha propuesta: %#v %v", r, err)
	}
}

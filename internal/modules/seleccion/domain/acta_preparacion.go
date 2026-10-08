package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	preparacion "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
)

var ErrMaterialActaInvalido = errors.New("seleccion.acta_preparacion.material_invalido")

// La huella procede de la entrada, no de un recibo emitido por el CLI de S5.
// Su formato válido no acredita cotejo del antecedente ni composición vigente.
type AntecedenteTribunalPropuesto struct {
	IdentidadMaterial    string `json:"identidad_material"`
	VersionMaterial      int    `json:"version_material"`
	HuellaAportadaSHA256 string `json:"huella_aportada_sha256"`
}

// MaterialActaPropuesto prepara textos para una sesión todavía propuesta.
// S5 conserva las referencias a S2 y al baremo; aquí no se copian ni resuelven.
type MaterialActaPropuesto struct {
	Alcance             string                       `json:"alcance"`
	IdentidadMaterial   string                       `json:"identidad_material"`
	VersionMaterial     int                          `json:"version_material"`
	AntecedenteTribunal AntecedenteTribunalPropuesto `json:"antecedente_tribunal"`
	FasePropuesta       string                       `json:"fase_propuesta"`
	SesionRef           string                       `json:"sesion_ref"`
	FechaPropuesta      string                       `json:"fecha_propuesta,omitempty"`
	OrdenDiaPropuesto   []PuntoSesionPropuesto       `json:"orden_dia_propuesto"`
	AcuerdosPropuestos  []AcuerdoSesionPropuesto     `json:"acuerdos_propuestos"`
}

type PuntoSesionPropuesto struct {
	PuntoRef       string `json:"punto_ref"`
	TextoPropuesto string `json:"texto_propuesto"`
}

type AcuerdoSesionPropuesto struct {
	PropuestaRef   string `json:"propuesta_ref"`
	PuntoRef       string `json:"punto_ref"`
	TextoPropuesto string `json:"texto_propuesto"`
}

type PendienteActa struct {
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type PreparacionActa struct {
	Estado            string                `json:"estado"`
	MaterialPropuesto MaterialActaPropuesto `json:"material_propuesto"`
	Pendientes        []PendienteActa       `json:"pendientes"`
}

// PrepararActa comprueba estructura y enlaces entre propuestas locales.
// No afirma reunión, asistencia, deliberación, votación o acuerdo adoptado.
func PrepararActa(m MaterialActaPropuesto) (PreparacionActa, error) {
	a := m.AntecedenteTribunal
	if m.Alcance != "preparacion_sintetica" || !preparacion.IdentificadorValido(m.IdentidadMaterial) ||
		!revisionActaValida(m.VersionMaterial) || !preparacion.IdentificadorValido(a.IdentidadMaterial) ||
		!revisionActaValida(a.VersionMaterial) || !preparacion.HuellaValida(a.HuellaAportadaSHA256) ||
		!preparacion.IdentificadorValido(m.FasePropuesta) ||
		(m.SesionRef != "" && !preparacion.IdentificadorValido(m.SesionRef)) ||
		len(m.OrdenDiaPropuesto) > 100 || len(m.AcuerdosPropuestos) > 100 {
		return PreparacionActa{}, ErrMaterialActaInvalido
	}
	if m.FechaPropuesta != "" {
		if len(m.FechaPropuesta) > 40 {
			return PreparacionActa{}, ErrMaterialActaInvalido
		}
		if _, err := time.Parse(time.RFC3339Nano, m.FechaPropuesta); err != nil {
			return PreparacionActa{}, ErrMaterialActaInvalido
		}
	}
	puntos := make(map[string]bool, len(m.OrdenDiaPropuesto))
	for _, punto := range m.OrdenDiaPropuesto {
		if !preparacion.IdentificadorValido(punto.PuntoRef) || puntos[punto.PuntoRef] || !textoActaValido(punto.TextoPropuesto) {
			return PreparacionActa{}, ErrMaterialActaInvalido
		}
		puntos[punto.PuntoRef] = true
	}
	acuerdos := make(map[string]bool, len(m.AcuerdosPropuestos))
	for _, acuerdo := range m.AcuerdosPropuestos {
		if !preparacion.IdentificadorValido(acuerdo.PropuestaRef) || acuerdos[acuerdo.PropuestaRef] ||
			!puntos[acuerdo.PuntoRef] || !textoActaValido(acuerdo.TextoPropuesto) {
			return PreparacionActa{}, ErrMaterialActaInvalido
		}
		acuerdos[acuerdo.PropuestaRef] = true
	}
	m.OrdenDiaPropuesto = append([]PuntoSesionPropuesto{}, m.OrdenDiaPropuesto...)
	m.AcuerdosPropuestos = append([]AcuerdoSesionPropuesto{}, m.AcuerdosPropuestos...)
	pendientes := []PendienteActa{{"antecedente_tribunal", "antecedente_no_cotejado"}, {"fase_propuesta", "pertenencia_no_verificada"}}
	for _, campo := range []string{"designacion", "habilitacion", "sesion_celebrada", "asistencia", "deliberaciones",
		"acuerdos_adoptados", "aprobacion", "firma"} {
		pendientes = append(pendientes, PendienteActa{campo, "circuito_pendiente"})
	}
	for _, dato := range []struct {
		campo string
		vacio bool
	}{{"sesion_ref", m.SesionRef == ""}, {"fecha_propuesta", m.FechaPropuesta == ""},
		{"orden_dia_propuesto", len(m.OrdenDiaPropuesto) == 0}, {"acuerdos_propuestos", len(m.AcuerdosPropuestos) == 0}} {
		if dato.vacio {
			pendientes = append(pendientes, PendienteActa{dato.campo, "material_ausente"})
		}
	}
	for _, punto := range m.OrdenDiaPropuesto {
		if strings.TrimSpace(punto.TextoPropuesto) == "" {
			pendientes = append(pendientes, PendienteActa{"textos_orden_dia", "material_ausente"})
			break
		}
	}
	for _, acuerdo := range m.AcuerdosPropuestos {
		if strings.TrimSpace(acuerdo.TextoPropuesto) == "" {
			pendientes = append(pendientes, PendienteActa{"textos_acuerdos", "material_ausente"})
			break
		}
	}
	return PreparacionActa{Estado: "borrador_propuesto", MaterialPropuesto: m, Pendientes: pendientes}, nil
}

func revisionActaValida(revision int) bool { return revision >= 1 && revision <= 1_000_000 }

func textoActaValido(texto string) bool {
	if len(texto) > 4096 || !utf8.ValidString(texto) {
		return false
	}
	for _, caracter := range texto {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

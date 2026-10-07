package domain

import "regexp"

// TipoActividad recoge lo declarado, sin conceder compatibilidad o excepción.
type TipoActividad string

const (
	SegundaActividadPublica TipoActividad = "segunda_publica"
	ActividadPrivada        TipoActividad = "privada"
	ActividadExceptuada     TipoActividad = "exceptuada_declarada"
)

// RelacionAsuntos es un hecho declarado; no constituye valoración jurídica.
type RelacionAsuntos string

const (
	RelacionSi          RelacionAsuntos = "si"
	RelacionNo          RelacionAsuntos = "no"
	RelacionDesconocida RelacionAsuntos = "desconocida"
)

// DeclaracionActividad contiene referencias opacas a hechos de ensayo. Validar
// su forma no acredita la existencia, procedencia ni titularidad de esos hechos.
type DeclaracionActividad struct {
	Tipo              TipoActividad   `json:"tipo"`
	ActividadRef      string          `json:"actividad_ref"`
	FuncionesRef      string          `json:"funciones_ref"`
	TitularRef        string          `json:"titular_ref"`
	JornadaRef        string          `json:"jornada_ref"`
	HorarioRef        string          `json:"horario_ref"`
	RelacionConPuesto RelacionAsuntos `json:"relacion_con_puesto"`
}

var referenciaOpaca = regexp.MustCompile(`^[a-z][a-z0-9_]{1,23}:[0-9a-f]{32}$`)

// CamposInvalidos comprueba únicamente la estructura; entrega nombres de campo,
// nunca valores. Ninguna referencia da acceso a datos ni resuelve una autoridad.
func (d DeclaracionActividad) CamposInvalidos() []string {
	var campos []string
	if d.Tipo != SegundaActividadPublica && d.Tipo != ActividadPrivada && d.Tipo != ActividadExceptuada {
		campos = append(campos, "tipo")
	}
	for _, campo := range []struct{ nombre, valor string }{
		{"actividad_ref", d.ActividadRef},
		{"funciones_ref", d.FuncionesRef},
		{"titular_ref", d.TitularRef},
		{"jornada_ref", d.JornadaRef},
		{"horario_ref", d.HorarioRef},
	} {
		if !referenciaOpaca.MatchString(campo.valor) {
			campos = append(campos, campo.nombre)
		}
	}
	if d.RelacionConPuesto != RelacionSi && d.RelacionConPuesto != RelacionNo && d.RelacionConPuesto != RelacionDesconocida {
		campos = append(campos, "relacion_con_puesto")
	}
	return campos
}

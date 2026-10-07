package domain

import "errors"

var ErrConfiguracion = errors.New("configuracion_seleccion_invalida")

type Configuracion struct {
	Version     int      `json:"version"`
	Modalidad   string   `json:"modalidad"`
	TurnoAcceso string   `json:"turno_acceso"`
	Destino     string   `json:"destino"`
	Plazas      int      `json:"plazas"`
	Fases       []Fase   `json:"fases"`
	Desempates  []string `json:"desempates"`
}
type Fase struct {
	Referencia        string `json:"referencia"`
	Tipo              string `json:"tipo"`
	MinimoMicropuntos *int64 `json:"minimo_micropuntos"`
	MaximoMicropuntos int64  `json:"maximo_micropuntos"`
	Peso              int64  `json:"peso"`
}
type Requisito struct {
	Referencia string `json:"referencia"`
	Estado     string `json:"estado"`
	Causa      string `json:"causa"`
	FuenteRef  string `json:"fuente_ref"`
}
type Solicitud struct {
	Referencia string            `json:"referencia"`
	Nombre     string            `json:"nombre"`
	Requisitos []Requisito       `json:"requisitos"`
	Notas      map[string]*int64 `json:"notas"`
}
type Entrada struct {
	ConvocatoriaRef string      `json:"convocatoria_ref"`
	BasesVersion    int         `json:"bases_version"`
	Solicitudes     []Solicitud `json:"solicitudes"`
}
type ReglaValorada struct {
	Referencia        string `json:"referencia"`
	PuntosMicropuntos int64  `json:"puntos_micropuntos"`
}
type MeritosValorados struct {
	PuntosMicropuntos *int64
	Reglas            []ReglaValorada
	HuellaSHA256      string
}
type FaseResultado struct {
	Referencia          string          `json:"referencia"`
	Tipo                string          `json:"tipo"`
	MinimoMicropuntos   *int64          `json:"minimo_micropuntos"`
	MaximoMicropuntos   int64           `json:"maximo_micropuntos"`
	Peso                int64           `json:"peso"`
	PuntosMicropuntos   *int64          `json:"puntos_micropuntos"`
	Estado              string          `json:"estado"`
	Origen              string          `json:"origen"`
	Reglas              []ReglaValorada `json:"reglas"`
	HuellaMeritosSHA256 string          `json:"huella_meritos_sha256,omitempty"`
}
type SolicitudResultado struct {
	Referencia       string          `json:"referencia"`
	Nombre           string          `json:"nombre"`
	Acceso           string          `json:"acceso"`
	AccesoDetalle    []Requisito     `json:"acceso_detalle"`
	Estado           string          `json:"estado"`
	Causas           []string        `json:"causas"`
	Fases            []FaseResultado `json:"fases"`
	TotalMicropuntos *int64          `json:"total_micropuntos"`
	Orden            *int            `json:"orden"`
	Propuesta        string          `json:"propuesta"`
}
type Resultado struct {
	Version         int                  `json:"version"`
	ConvocatoriaRef string               `json:"convocatoria_ref"`
	BasesVersion    int                  `json:"bases_version"`
	Modalidad       string               `json:"modalidad"`
	TurnoAcceso     string               `json:"turno_acceso"`
	Destino         string               `json:"destino"`
	Plazas          int                  `json:"plazas"`
	Alcance         string               `json:"alcance"`
	Estado          string               `json:"estado"`
	Causas          []string             `json:"causas"`
	Solicitudes     []SolicitudResultado `json:"solicitudes"`
}

// Validar limita el ensayo a reglas declarativas y aritmética exacta acotada.
func (c Configuracion) Validar() error {
	if c.Version < 1 || c.Version > 1_000_000 || c.Plazas < 1 || c.Plazas > 1000 || len(c.Fases) == 0 || len(c.Fases) > 16 || len(c.Desempates) > len(c.Fases) {
		return ErrConfiguracion
	}
	if c.TurnoAcceso != "libre" && c.TurnoAcceso != "promocion_interna" && c.TurnoAcceso != "discapacidad" {
		return ErrConfiguracion
	}
	if c.Destino != "bolsa" && c.Destino != "plaza" {
		return ErrConfiguracion
	}
	refs := map[string]bool{}
	pruebas, meritos, peso := 0, 0, int64(0)
	for _, f := range c.Fases {
		if !referenciaValida(f.Referencia) || refs[f.Referencia] || f.MaximoMicropuntos <= 0 || f.MaximoMicropuntos > 1_000_000_000 || f.Peso < 1 || f.Peso > 100 {
			return ErrConfiguracion
		}
		if f.MinimoMicropuntos != nil && (*f.MinimoMicropuntos < 0 || *f.MinimoMicropuntos > f.MaximoMicropuntos) {
			return ErrConfiguracion
		}
		refs[f.Referencia] = true
		peso += f.Peso
		switch f.Tipo {
		case "prueba":
			pruebas++
		case "meritos":
			meritos++
		default:
			return ErrConfiguracion
		}
	}
	if peso != 100 || meritos > 1 {
		return ErrConfiguracion
	}
	switch c.Modalidad {
	case "oposicion":
		if pruebas == 0 || meritos != 0 {
			return ErrConfiguracion
		}
	case "concurso":
		if pruebas != 0 || meritos != 1 {
			return ErrConfiguracion
		}
	case "concurso_oposicion":
		if pruebas == 0 || meritos != 1 {
			return ErrConfiguracion
		}
	default:
		return ErrConfiguracion
	}
	vistos := map[string]bool{}
	for _, ref := range c.Desempates {
		if !refs[ref] || vistos[ref] {
			return ErrConfiguracion
		}
		vistos[ref] = true
	}
	return nil
}
func referenciaValida(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if r != '_' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// Validar se ejecuta antes de llamar al baremador y también en la evaluación pura.
func (e Entrada) Validar() error {
	if e.ConvocatoriaRef == "" || e.BasesVersion < 1 || len(e.Solicitudes) == 0 || len(e.Solicitudes) > 128 {
		return ErrConfiguracion
	}
	refs := map[string]bool{}
	for _, s := range e.Solicitudes {
		if s.Referencia == "" || refs[s.Referencia] {
			return ErrConfiguracion
		}
		refs[s.Referencia] = true
	}
	return nil
}

func copiarPuntos(p *int64) *int64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func Acceso(requisitos []Requisito) string {
	estado := "cumple"
	if len(requisitos) == 0 {
		return "pendiente"
	}
	for _, r := range requisitos {
		if r.Referencia == "" || r.FuenteRef == "" {
			estado = "pendiente"
			continue
		}
		if r.Estado == "no_cumple" {
			return "no_cumple"
		}
		if r.Estado != "cumple" || r.FuenteRef == "" {
			estado = "pendiente"
		}
	}
	return estado
}

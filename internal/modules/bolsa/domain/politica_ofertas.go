package domain

import (
	"errors"
	"regexp"
)

// PoliticaOfertas conserva el inicio, plazo y calendario configurables de
// cada bolsa. La edición crea una versión nueva; las ofertas anteriores
// conservan la política que las gobernó.
type PoliticaOfertas struct {
	Plazo        PlazoPoliticaOfertas        `json:"plazo"`
	Adjudicacion AdjudicacionPoliticaOfertas `json:"adjudicacion"`
	NoCubierta   NoCubiertaPoliticaOfertas   `json:"no_cubierta"`
	// Plazas es opcional (duda 75). Sin él, cada oferta tiene una sola plaza y
	// la adjudicación confirmada la cubre, como en las versiones anteriores.
	Plazas *PlazasPoliticaOfertas `json:"plazas,omitempty"`
	// Su ausencia conserva las publicaciones anteriores: solo disponibles y
	// ninguna modalidad excluida por una preferencia no declarada.
	CriteriosOfrecimiento *CriteriosOfrecimiento `json:"criterios_ofrecimiento,omitempty"`
}

// PlazasPoliticaOfertas gobierna las ofertas con varias plazas: si se llama a
// la vez a tantas personas como plazas libres o de una en una, cuántas horas
// tiene cada persona para responder y qué se hace con la plaza tras una
// renuncia o una falta de respuesta. Bolsa ejecuta exactamente estos valores.
type PlazasPoliticaOfertas struct {
	Llamada        string `json:"llamada"`
	RespuestaHoras int    `json:"respuesta_horas"`
	TrasRenuncia   string `json:"tras_renuncia"`
}

// Valores admitidos del apartado de plazas.
const (
	LlamadaPlazasSimultanea         = "simultanea"
	LlamadaPlazasSucesiva           = "sucesiva"
	TrasRenunciaSiguienteEnOrden    = "siguiente_en_orden"
	TrasRenunciaLlamamientoDirecto  = "llamamiento_directo"
	MaximoHorasRespuestaPlazaOferta = 720
)

// Validar admite solo valores que ejecuta la proyección de plazas de Bolsa.
func (p PlazasPoliticaOfertas) Validar() error {
	if (p.Llamada != LlamadaPlazasSimultanea && p.Llamada != LlamadaPlazasSucesiva) ||
		p.RespuestaHoras < 1 || p.RespuestaHoras > MaximoHorasRespuestaPlazaOferta ||
		(p.TrasRenuncia != TrasRenunciaSiguienteEnOrden && p.TrasRenuncia != TrasRenunciaLlamamientoDirecto) {
		return ErrPoliticaOfertasInvalida
	}
	return nil
}

type PlazoPoliticaOfertas struct {
	// Vacío conserva la forma histórica; las nuevas ofertas exigen un inicio
	// explícito. Publicación usa el instante durable de la oferta.
	Inicio        string `json:"inicio,omitempty"`
	Unidad        string `json:"unidad"`
	Cantidad      int    `json:"cantidad"`
	Computo       string `json:"computo"`
	MunicipioSede string `json:"municipio_sede"`
}

const (
	InicioPlazoOfertaPublicacion  = "publicacion"
	InicioPlazoOfertaNotificacion = "notificacion"
)

type AdjudicacionPoliticaOfertas struct {
	Criterio     string `json:"criterio"`
	Elegibilidad string `json:"elegibilidad"`
	Confirmacion string `json:"confirmacion,omitempty"`
}

type NoCubiertaPoliticaOfertas struct {
	Accion    string `json:"accion"`
	Condicion string `json:"condicion"`
}

var (
	ErrPoliticaOfertasInvalida = errors.New("bolsa: politica de ofertas invalida")
	municipioINE               = regexp.MustCompile(`^[0-9]{5}$`)
)

// Validar limita el catálogo a reglas que ejecuta el evaluador de Bolsa.
// No acepta una etiqueta libre que pudiera prometer un efecto no implementado.
func (p PoliticaOfertas) Validar() error {
	unidad, cantidad, computo := p.Plazo.Unidad, p.Plazo.Cantidad, p.Plazo.Computo
	plazoValido := ((unidad == "dias_habiles" || unidad == "dias_naturales") &&
		cantidad >= 1 && cantidad <= 30 && computo == "administrativo") ||
		(unidad == "horas_naturales" && cantidad >= 1 && cantidad <= 720 && computo == "continuo_utc")
	if !plazoValido || (p.Plazo.Inicio != "" && p.Plazo.Inicio != InicioPlazoOfertaNotificacion && p.Plazo.Inicio != InicioPlazoOfertaPublicacion) ||
		!municipioINE.MatchString(p.Plazo.MunicipioSede) ||
		p.Adjudicacion.Criterio != "orden_vigente" || p.Adjudicacion.Elegibilidad != "disposicion_en_plazo" ||
		!ConfirmacionAdjudicacionValida(p.Adjudicacion.Confirmacion) ||
		p.NoCubierta.Accion != "llamamiento_directo" || p.NoCubierta.Condicion != "sin_disposiciones_elegibles" {
		return ErrPoliticaOfertasInvalida
	}
	if p.CriteriosOfrecimiento != nil {
		if err := p.CriteriosOfrecimiento.Validar(); err != nil {
			return ErrPoliticaOfertasInvalida
		}
	}
	if p.Plazas != nil {
		return p.Plazas.Validar()
	}
	return nil
}

// ValidarParaOfertasNuevas impide reinterpretar una versión histórica sin
// inicio explícito. La versión vigente decide entre publicación y notificación.
func (p PoliticaOfertas) ValidarParaOfertasNuevas() error {
	if err := p.Validar(); err != nil {
		return err
	}
	if p.Plazo.Inicio != InicioPlazoOfertaNotificacion && p.Plazo.Inicio != InicioPlazoOfertaPublicacion {
		return ErrPoliticaOfertasInvalida
	}
	return nil
}

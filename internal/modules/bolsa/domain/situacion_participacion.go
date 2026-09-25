package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrCambioSituacionParticipacionInvalido = errors.New("bolsa: cambio de situacion de participacion invalido")

const (
	SituacionDisponible             = "disponible"
	SituacionNoDisponible           = "no_disponible"
	SituacionTrabajando             = "trabajando"
	SituacionPendienteIncorporacion = "pendiente_incorporacion"
	SituacionRenuncia               = "renuncia"
	SituacionExcluido               = "excluido"
	SituacionDisponibleDesde        = "disponible_desde"
)

var catalogoSituacionesParticipacion = map[string]struct{}{
	SituacionDisponible: {}, SituacionNoDisponible: {}, SituacionTrabajando: {},
	SituacionPendienteIncorporacion: {}, SituacionRenuncia: {}, SituacionExcluido: {},
	SituacionDisponibleDesde: {},
}

// transicionesSituacionParticipacion es el literal de
// registrar_situacion_participacion_v1 en la migración 000012 de
// bolsa_llamamientos y la versión 1 de su política de transiciones (000032).
// Rige mientras la base no publique otra: desde 000032 la base de datos guarda
// la política vigente, publicada al arrancar desde el catálogo de reglas
// (b28.transiciones.<origen>), y esa política sustituye a esta tabla. Ver
// PoliticaTransicionesSituacion.
var transicionesSituacionParticipacion = map[string]map[string]struct{}{
	SituacionDisponible:             {SituacionNoDisponible: {}, SituacionPendienteIncorporacion: {}, SituacionRenuncia: {}, SituacionExcluido: {}},
	SituacionNoDisponible:           {SituacionDisponible: {}, SituacionExcluido: {}},
	SituacionPendienteIncorporacion: {SituacionTrabajando: {}, SituacionDisponible: {}, SituacionRenuncia: {}, SituacionExcluido: {}},
	SituacionTrabajando:             {SituacionDisponible: {}, SituacionDisponibleDesde: {}, SituacionExcluido: {}},
	SituacionDisponibleDesde:        {SituacionDisponible: {}, SituacionExcluido: {}},
	SituacionRenuncia:               {SituacionDisponible: {}, SituacionExcluido: {}},
	SituacionExcluido:               {},
}

func SituacionesParticipacion() []string {
	return []string{SituacionDisponible, SituacionNoDisponible, SituacionTrabajando, SituacionPendienteIncorporacion, SituacionRenuncia, SituacionExcluido, SituacionDisponibleDesde}
}

func DestinosSituacionParticipacion(origen string) []string {
	destinos := transicionesSituacionParticipacion[origen]
	resultado := make([]string, 0, len(destinos))
	for _, situacion := range SituacionesParticipacion() {
		if _, ok := destinos[situacion]; ok {
			resultado = append(resultado, situacion)
		}
	}
	return resultado
}

type CambioSituacionParticipacion struct {
	ParticipacionRef string
	Origen           string
	Destino          string
	Desde            time.Time
	Motivo           string
	FechaDisponible  *time.Time
	RegistradaEn     time.Time
}

// Validar comprueba el cambio con la tabla compilada.
func (c CambioSituacionParticipacion) Validar() error {
	return c.ValidarCon(PoliticaTransicionesSituacionCompilada())
}

// ValidarCon comprueba el cambio con la política de transiciones vigente.
func (c CambioSituacionParticipacion) ValidarCon(politica PoliticaTransicionesSituacion) error {
	if !referenciaLlamamientoOpacaValida(c.ParticipacionRef) || !situacionParticipacionValida(c.Origen) || !situacionParticipacionValida(c.Destino) ||
		strings.TrimSpace(c.Motivo) != c.Motivo || len(c.Motivo) == 0 || len(c.Motivo) > 1000 ||
		!instanteLlamamientoCanonico(c.Desde) || !instanteLlamamientoCanonico(c.RegistradaEn) || c.Desde.Before(c.RegistradaEn) {
		return ErrCambioSituacionParticipacionInvalido
	}
	if !politica.Admite(c.Origen, c.Destino) {
		return ErrCambioSituacionParticipacionInvalido
	}
	if c.Destino == SituacionDisponibleDesde {
		if c.FechaDisponible == nil || !c.FechaDisponible.After(c.RegistradaEn) {
			return ErrCambioSituacionParticipacionInvalido
		}
	} else if c.FechaDisponible != nil {
		return ErrCambioSituacionParticipacionInvalido
	}
	return nil
}

func situacionParticipacionValida(s string) bool {
	_, ok := catalogoSituacionesParticipacion[s]
	return ok
}

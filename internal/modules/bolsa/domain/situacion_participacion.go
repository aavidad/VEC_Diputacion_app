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

// transicionesSituacionParticipacion es provisional por decisión de Dirección
// hasta la respuesta de RRHH a las dudas 13 y 18. B2 no incorpora readmisión.
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

func (c CambioSituacionParticipacion) Validar() error {
	if !referenciaLlamamientoOpacaValida(c.ParticipacionRef) || !situacionParticipacionValida(c.Origen) || !situacionParticipacionValida(c.Destino) ||
		strings.TrimSpace(c.Motivo) != c.Motivo || len(c.Motivo) == 0 || len(c.Motivo) > 1000 ||
		!instanteLlamamientoCanonico(c.Desde) || !instanteLlamamientoCanonico(c.RegistradaEn) || c.Desde.Before(c.RegistradaEn) {
		return ErrCambioSituacionParticipacionInvalido
	}
	if _, ok := transicionesSituacionParticipacion[c.Origen][c.Destino]; !ok {
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

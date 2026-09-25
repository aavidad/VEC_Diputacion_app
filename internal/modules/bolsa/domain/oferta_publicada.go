package domain

import (
	"errors"
	"strings"
	"time"
)

// Ofertas publicadas (Petición RRHH p. 3; Reglamento de bolsas, art. 8.1):
// RRHH publica una oferta de una bolsa; quienes figuran en ella manifiestan
// disposición en el plazo de la regla b10; al vencer se adjudica a la mejor
// posición del orden vigente entre ellas o se pasa a llamamiento directo.

// Estados deducidos de una oferta. No se almacenan: salen del instante de la
// consulta, del vencimiento y de la resolución confirmada.
const (
	EstadoOfertaAbierta               = "abierta"
	EstadoOfertaPendienteResolucion   = "pendiente_resolucion"
	EstadoOfertaAdjudicada            = "adjudicada"
	EstadoOfertaLlamamientoDirecto    = "llamamiento_directo"
	PropuestaOfertaAdjudicar          = "adjudicar"
	PropuestaOfertaLlamamientoDirecto = "llamamiento_directo"
)

const (
	minimoTextoOferta = 2
	maximoTextoOferta = 2000
	formatoFechaDia   = "2006-01-02"
)

var ErrDatosOfertaInvalidos = errors.New("bolsa: datos de oferta invalidos")

// DatosOferta es lo que RRHH publica y ven las personas de la bolsa. Las
// fechas son días civiles (AAAA-MM-DD); la de fin es opcional.
type DatosOferta struct {
	Categoria   string `json:"categoria"`
	Centro      string `json:"centro"`
	FechaInicio string `json:"fecha_inicio"`
	FechaFin    string `json:"fecha_fin,omitempty"`
	Descripcion string `json:"descripcion"`
}

// Validar exige textos recortados y acotados y fechas coherentes.
func (d DatosOferta) Validar() error {
	for _, texto := range []string{d.Categoria, d.Centro, d.FechaInicio, d.Descripcion} {
		if !textoOfertaValido(texto) {
			return ErrDatosOfertaInvalidos
		}
	}
	inicio, err := time.Parse(formatoFechaDia, d.FechaInicio)
	if err != nil {
		return ErrDatosOfertaInvalidos
	}
	if d.FechaFin == "" {
		return nil
	}
	fin, err := time.Parse(formatoFechaDia, d.FechaFin)
	if err != nil || fin.Before(inicio) {
		return ErrDatosOfertaInvalidos
	}
	return nil
}

func textoOfertaValido(texto string) bool {
	return strings.TrimSpace(texto) == texto && len(texto) >= minimoTextoOferta && len(texto) <= maximoTextoOferta
}

package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Carga de bolsas desde el Excel de CONVOCA por RRHH (B1): RRHH sube el
// fichero, ve la vista previa fila a fila y confirma. La confirmación importa
// el acta y constituye la bolsa consumiendo la decisión V3 propia de la carga
// (AD203/B79) en la misma transacción que la constitución.
const (
	AccionConfirmarCargaConvoca    = "bolsa.carga_convoca.confirmar"
	FinalidadConfirmarCargaConvoca = "carga_bolsa_convoca"
	AudienciaConfirmarCargaConvoca = "vec_bolsa_llamamientos.carga_convoca.confirmar.v1"
	TipoRecursoCargaConvoca        = "carga_convoca"
	ModuloCargaConvoca             = "bolsa"
)

var (
	ErrCargaConvocaNoDisponible = errors.New("bolsa: carga desde CONVOCA no disponible")
	ErrCargaConvocaInvalida     = errors.New("bolsa: carga desde CONVOCA invalida")
)

// SolicitudConfirmarCargaConvoca solo se construye detrás de la frontera de
// RRHH: el vínculo, el contexto, la correlación y el motivo los pone el
// servidor; del navegador solo llegan el fichero, su nombre y la categoría.
type SolicitudConfirmarCargaConvoca struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	// CategoriaRef es la referencia RPT ya validada contra el catálogo
	// (categoria:rpt:<clave>). BolsaRef vacío hace que la constitución derive
	// una referencia estable del acta, incluso si se cargan dos bolsas el mismo día.
	CategoriaRef  string
	BolsaRef      string
	NombreFichero string
	Contenido     []byte
}

// Validar exige lo que necesita la decisión V3 de la carga.
func (s SolicitudConfirmarCargaConvoca) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.ResultadoContexto.Contexto.PersonaRef == "" || s.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) ||
		s.CategoriaRef == "" || s.NombreFichero == "" || len(s.Contenido) == 0 {
		return ErrCargaConvocaInvalida
	}
	return nil
}

// ReciboCargaConvoca es la constitución registrada junto con el acuse del
// consumo de la decisión en la auditoría común.
type ReciboCargaConvoca struct {
	ReciboConstitucion
	DecisionRef  string
	AuditoriaRef string
	ConsumidaEn  time.Time
}

// RepositorioConstitucionCargaConvoca constituye la bolsa consumiendo la
// decisión de la carga en la misma transacción. Si la decisión no vale, no es
// del actor o no es de esa acta, se revierte y no queda bolsa ni consumo.
type RepositorioConstitucionCargaConvoca interface {
	ConstituirCargaConvocaAutorizada(context.Context, Constitucion, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboCargaConvoca, error)
}

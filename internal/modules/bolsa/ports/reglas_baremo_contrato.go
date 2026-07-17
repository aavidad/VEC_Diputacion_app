// Package ports declara las fronteras hexagonales del modulo de Bolsa.
package ports

import "errors"

var (
	// ErrReglasBaremoNoEncontradas indica que no existe el estado exacto
	// solicitado. Ningun adaptador puede sustituirlo por «el vigente».
	ErrReglasBaremoNoEncontradas = errors.New("bolsa: estado exacto de reglas de baremo no encontrado")

	// ErrConflictoOCCReglasBaremo indica que revision o huella esperadas ya no
	// coinciden con el estado durable.
	ErrConflictoOCCReglasBaremo = errors.New("bolsa: conflicto OCC en reglas de baremo")

	// ErrClaveIdempotenciaReglasReutilizada corresponde al indice exacto de
	// intencion ya usado para otro material semantico. La derivacion de ese
	// indice pertenece a application/internal, nunca a este paquete.
	ErrClaveIdempotenciaReglasReutilizada = errors.New("bolsa: indice idempotente reutilizado con otra intencion")

	ErrConfirmacionReglasBaremoInvalida  = errors.New("bolsa: confirmacion transaccional de reglas de baremo invalida")
	ErrFuenteCalculoReglasBaremoInvalida = errors.New("bolsa: fuente exacta de calculo de reglas de baremo invalida")
)

// OperacionGobiernoReglasBaremo identifica el efecto durable ya preparado por
// application. No ejecuta ni valida transiciones de dominio.
type OperacionGobiernoReglasBaremo string

const (
	OperacionAltaBorradorReglasBaremo OperacionGobiernoReglasBaremo = "alta_borrador"
	OperacionPublicarReglasBaremo     OperacionGobiernoReglasBaremo = "publicar"
	OperacionActivarReglasBaremo      OperacionGobiernoReglasBaremo = "activar"
	OperacionSustituirReglasBaremo    OperacionGobiernoReglasBaremo = "sustituir"
	OperacionRetirarReglasBaremo      OperacionGobiernoReglasBaremo = "retirar"
	OperacionDescartarReglasBaremo    OperacionGobiernoReglasBaremo = "descartar"
)

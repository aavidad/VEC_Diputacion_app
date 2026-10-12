package ports

import (
	"context"
	"errors"

	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrComprobanteMeritoNoDisponible = errors.New("meritos.error.comprobante_no_disponible")

// ConsultaComprobanteMeritoConfirmado es una lectura inmutable de Documentos,
// no un DTO de transporte ni un consumo de un solo uso. Su implementacion debe
// autorizar a la persona antes de leer cualquier dato del comprobante, cotejar
// propiedad y version exacta, carga admitida con antivirus limpio, registro
// documental confirmado y lectura verificable del original custodiado. Un
// resultado pendiente, sustituido, ajeno o incierto falla cerrado. La
// referencia devuelta procede del registro documental, nunca del navegador.
// RUM conserva su propia idempotencia y CAS; un fallo posterior permite repetir
// esta lectura autorizada sin repetir la ingesta del archivo.
type ConsultaComprobanteMeritoConfirmado interface {
	ConsultarVersionConfirmadaPropia(context.Context, string, string) (vec.ReferenciaDocumento, error)
}

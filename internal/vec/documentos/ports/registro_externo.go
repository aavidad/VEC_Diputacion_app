package ports

import "context"

// AutorizadorRegistroExterno obtiene la concesion V3 de un registro externo
// ligada a la preimagen exacta (AltaExternaPersistente.PreimagenExterna), al
// documento y al expediente. La identidad procede del canal autenticado de
// la peticion; nunca de estos argumentos. Un error envuelto en
// ErrCapacidadNoDisponible es dependencia caida; cualquier otro, denegacion.
type AutorizadorRegistroExterno interface {
	AutorizarRegistroExterno(ctx context.Context, preimagen []byte, documentoID, expedienteRef string) (AutorizacionV3, error)
}

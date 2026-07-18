package almacen

import "errors"

var (
	ErrSolicitudAlmacenInvalida              = errors.New("vec: solicitud de almacen invalida")
	ErrInstruccionesCargaDirectaNoValidas    = errors.New("vec: instrucciones de carga directa no validas")
	ErrReciboCargaDirectaNoValido            = errors.New("vec: recibo de carga directa no valido")
	ErrSerializacionReciboCargaProhibida     = errors.New("vec: serializacion accidental del recibo de carga directa prohibida")
	ErrSerializacionSeudonimizacionProhibida = errors.New("vec: serializacion accidental de seudonimizacion de almacen prohibida")
	ErrSeudonimizacionAlmacenNoDisponible    = errors.New("vec: seudonimizacion de sujeto para almacen no disponible")
	ErrSerializacionContextoAlmacenProhibida = errors.New("vec: serializacion de contexto de almacen prohibida")
)

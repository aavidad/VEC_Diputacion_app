// Package recibomaterial concentra la validacion y la representacion canonica
// del recibo material de escritura. No conoce puertos, adaptadores ni estado.
package recibomaterial

import "errors"

var (
	// ErrReciboNoValido mantiene deliberadamente opacos todos los rechazos.
	ErrReciboNoValido = errors.New("vec: recibo material v2 de escritura no valido")
	// ErrAtestacionNoValida evita distinguir fallos de forma y autenticidad.
	ErrAtestacionNoValida = errors.New("vec: atestacion material v2 de almacen no valida")
	// ErrSerializacionProhibida impide volcar capacidades opacas por accidente.
	ErrSerializacionProhibida = errors.New("vec: serializacion generica de material v2 de almacen prohibida")
)

const (
	EsquemaPerfil      = "vec.almacen.perfil-capacidades-material.v2"
	EsquemaInstantanea = "vec.almacen.instantanea-objeto-material.v2"
	EsquemaRecibo      = "vec.almacen.recibo-escritura-material.v2"
	EsquemaVersion     = uint16(2)

	DominioPerfil = "perfil-capacidades-almacen-material-v2"
	DominioRecibo = "recibo-escritura-objeto-material-v2"

	EstadoNoInmovilizado = "no_inmovilizado"
	EstadoInmovilizado   = "inmovilizado"
	EstadoObjetoActivo   = "activo"

	AlgoritmoHMACSHA256 = "hmac-sha-256"
	AlgoritmoCOSESign1  = "cose-sign1"
	AccionEscribir      = "escribir"

	TextoRedactado = "[MATERIAL-ALMACEN-V2-REDACTADO]"
)

package ports

import "bytes"

// exportacionCanonicaRRHH conserva una única serialización y su huella. No es
// una autorización: el consumidor PostgreSQL debe verificar y consumir el
// material VEC dentro de la misma transacción que realiza la lectura.
type exportacionCanonicaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	dominio        string
	version        uint16
	bytesCanonicos []byte
	huellaSHA256   string
}

func (e exportacionCanonicaRRHH) Dominio() string {
	return e.dominio
}

func (e exportacionCanonicaRRHH) Version() uint16 {
	return e.version
}

// BytesCanonicos devuelve una copia defensiva apta para un parámetro binario
// SQL. Nunca debe interpolarse en una sentencia ni registrarse en bitácoras.
func (e exportacionCanonicaRRHH) BytesCanonicos() []byte {
	return bytes.Clone(e.bytesCanonicos)
}

func (e exportacionCanonicaRRHH) HuellaSHA256() string {
	return e.huellaSHA256
}

// Los cuatro tipos nominales impiden cruzar por accidente consulta, familia,
// detalle y alcance al construir los argumentos del adaptador.
// ExportacionCanonicaConsultaCuadroRRHH transporta la consulta de una página.
type ExportacionCanonicaConsultaCuadroRRHH struct {
	exportacionCanonicaRRHH
}

// ExportacionCanonicaFamiliaCuadroRRHH transporta los filtros estables de una
// navegación paginada, sin incluir el cursor.
type ExportacionCanonicaFamiliaCuadroRRHH struct {
	exportacionCanonicaRRHH
}

// ExportacionCanonicaConsultaDetalleRRHH transporta la referencia y la versión
// observada exigida por una consulta de detalle.
type ExportacionCanonicaConsultaDetalleRRHH struct {
	exportacionCanonicaRRHH
}

// ExportacionCanonicaAlcanceRRHH transporta únicamente el alcance organizativo
// mínimo concedido: organización, clase de ámbito y referencia opaca.
type ExportacionCanonicaAlcanceRRHH struct {
	exportacionCanonicaRRHH
}

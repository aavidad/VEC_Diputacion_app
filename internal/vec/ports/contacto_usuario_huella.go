package ports

import "context"

// SelladorHuellaPeticionContactoUsuario liga una intención al KMS. La salida
// es HMAC-SHA256 hexadecimal; el replay durable requiere conservar la versión
// de clave que la originó. La rotación no está implementada en el KMS actual.
type SelladorHuellaPeticionContactoUsuario interface {
	HuellaPeticionContactoUsuario(context.Context, string, uint64, []byte) (string, error)
}

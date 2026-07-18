package recibomaterial

import (
	"crypto/sha256"
	"crypto/subtle"
)

// SolicitudAtestacionValida comprueba dominio, copia logica y compromiso.
func SolicitudAtestacionValida(dominio string, mensaje []byte, huella [sha256.Size]byte) bool {
	esperada := sha256.Sum256(mensaje)
	return DominioAtestacionValido(dominio) && len(mensaje) > 0 &&
		huella != ([sha256.Size]byte{}) && subtle.ConstantTimeCompare(huella[:], esperada[:]) == 1
}

// AtestacionValida comprueba que la respuesta corresponde a la solicitud exacta.
func AtestacionValida(
	dominio string,
	mensaje []byte,
	huella [sha256.Size]byte,
	algoritmo, claveRef string,
	claveVersion uint32,
	dominioAtestado string,
	huellaAtestada [sha256.Size]byte,
	codigo []byte,
) bool {
	return SolicitudAtestacionValida(dominio, mensaje, huella) &&
		(algoritmo == AlgoritmoHMACSHA256 || algoritmo == AlgoritmoCOSESign1) &&
		AliasLogicoValido(claveRef, 256) && claveVersion > 0 && dominioAtestado == dominio &&
		subtle.ConstantTimeCompare(huellaAtestada[:], huella[:]) == 1 &&
		CodigoAtestacionValido(algoritmo, codigo)
}

// ResultadoLigado valida un resultado criptografico contra su vinculo exacto.
func ResultadoLigado(huellaPlan, huellaResultado, vinculoResultado [sha256.Size]byte) bool {
	return huellaResultado != ([sha256.Size]byte{}) && vinculoResultado != ([sha256.Size]byte{}) &&
		subtle.ConstantTimeCompare(vinculoResultado[:], huellaPlan[:]) == 1 &&
		!HuellasIguales(huellaResultado, huellaPlan)
}

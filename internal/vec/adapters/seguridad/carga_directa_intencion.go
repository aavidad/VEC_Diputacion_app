package seguridad

import (
	"encoding/hex"
)

func referenciaConsumoDeterministaCargaDirecta(
	clave claveHMACCargaDirecta, dominio, indiceHMAC, grupoHMAC, vinculoHMAC string,
) string {
	suma := calcularHMACCargaDirecta(clave, dominio+":v2", indiceHMAC, grupoHMAC, vinculoHMAC)
	defer borrarBytesCargaDirecta(suma)
	return dominio + ":" + hex.EncodeToString(suma)
}

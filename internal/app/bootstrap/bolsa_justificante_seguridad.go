package bootstrap

import (
	"net/http"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

// La frontera limita el canal y los perfiles ya publicados. El PDP comprueba
// después el permiso documental sobre la bolsa concreta en cada operación.
func fronteraJustificanteBolsaRRHHDesarrollo(perfiles []string) descriptorFronteraComunDesarrollo {
	return descriptorFronteraComunDesarrollo{
		Clave: "bolsa-justificante-renuncia-rrhh", Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
		PerfilesActivosRef: append([]string(nil), perfiles...),
		ClavePolitica:      "bolsa-justificante-renuncia-rrhh", ClaveCapacidad: "bolsa-justificante-renuncia-rrhh",
	}
}

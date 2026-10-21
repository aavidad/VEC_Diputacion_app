package bootstrap

import (
	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
	reglashttp "vec-diputacion-granada/internal/vec/reglas/httpinterno"
)

// rutaReglasVigentesDesarrollo identifica la consulta de reglas vigentes, que
// la frontera mTLS protege como lectura de RRHH, igual que Calendarios.
func rutaReglasVigentesDesarrollo(ruta string) bool {
	return ruta == reglashttp.RutaReglasVigentes
}

// nuevaRutaReglasVigentesDesarrollo compone la lectura de las reglas de Bolsa
// y Contratación temporal. Sin paquete de ejemplo declarado la ruta existe y
// cada catálogo aparece como «sin catálogo»: nunca con reglas supuestas.
func nuevaRutaReglasVigentesDesarrollo(compuestas reglasEjemploDesarrollo, rutas ...vechttp.RutaExacta) (vechttp.RutaExacta, error) {
	ajustesMontados := false
	for _, ruta := range rutas {
		if ruta.Ruta == ajusteshttp.Ruta && ruta.Manejador != nil {
			ajustesMontados = true
		}
	}
	manejador, err := reglashttp.NuevoManejador(
		reglashttp.Fuente{Modulo: reglas.ModuloBolsa, CatalogoID: reglas.CatalogoBolsa, Consulta: compuestas.bolsa},
		reglashttp.Fuente{Modulo: reglas.ModuloContratacionTemporal, CatalogoID: reglas.CatalogoContratacionTemporal, Consulta: compuestas.contratacionTemporal, EdicionDisponible: ajustesMontados},
	)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: reglashttp.RutaReglasVigentes, Manejador: manejador}, nil
}

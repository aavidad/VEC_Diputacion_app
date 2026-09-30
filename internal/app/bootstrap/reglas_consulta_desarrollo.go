package bootstrap

import (
	"io"
	"os"
	"strings"

	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	ajustespg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ajustesapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
	reglashttp "vec-diputacion-granada/internal/vec/reglas/httpinterno"
)

const envCTMotivosNegocioAjustes = "VEC_CT_MOTIVOS_AJUSTE_FILE"

// rutaReglasVigentesDesarrollo identifica la consulta de reglas vigentes, que
// la frontera mTLS protege como lectura de RRHH, igual que Calendarios.
func rutaReglasVigentesDesarrollo(ruta string) bool {
	return ruta == reglashttp.RutaReglasVigentes
}

// nuevaRutaReglasVigentesDesarrollo compone la lectura de las reglas de Bolsa
// y Contratación temporal. Sin paquete de ejemplo declarado la ruta existe y
// cada catálogo aparece como «sin catálogo»: nunca con reglas supuestas.
func nuevaRutaReglasVigentesDesarrollo(compuestas reglasEjemploDesarrollo) (vechttp.RutaExacta, error) {
	manejador, err := reglashttp.NuevoManejador(
		reglashttp.Fuente{Modulo: reglas.ModuloBolsa, CatalogoID: reglas.CatalogoBolsa, Consulta: compuestas.bolsa},
		reglashttp.Fuente{Modulo: reglas.ModuloContratacionTemporal, CatalogoID: reglas.CatalogoContratacionTemporal, Consulta: compuestas.contratacionTemporal},
	)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: reglashttp.RutaReglasVigentes, Manejador: manejador}, nil
}

// La escritura queda cerrada hasta que CT110 conserve la instantánea de cada
// plazo al abrir el tramo. La lectura sí consume una decisión V3 nominal.
func nuevaRutaConsultaAjustesCT(alta *dependenciasAltaContratacionTemporalDesarrollo,
	perfil *perfilFijoCTDesarrollo, fuente *reglas.Resolutor, reloj relojContratacionTemporalDesarrollo,
) (vechttp.RutaExacta, error) {
	if alta == nil || perfil == nil || fuente == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.proveedorMaterialConsultaAjustesReglas == nil ||
		!alta.postgresql.consultaAjustesReglasActiva ||
		perfil.perfilRef() == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef {
		return vechttp.RutaExacta{}, errMotivoAutorizacionAjustes
	}
	rutaMotivos := strings.TrimSpace(os.Getenv(envCTMotivosNegocioAjustes))
	if rutaMotivos == "" {
		return vechttp.RutaExacta{}, errMotivoAutorizacionAjustes
	}
	archivo, err := os.Open(rutaMotivos)
	if err != nil {
		return vechttp.RutaExacta{}, errMotivoAutorizacionAjustes
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(archivo, 4097))
	if err != nil || len(contenido) == 0 || len(contenido) > 4096 {
		return vechttp.RutaExacta{}, errMotivoAutorizacionAjustes
	}
	motivos, err := ajustesapp.LeerCatalogoMotivos(contenido)
	if err != nil {
		return vechttp.RutaExacta{}, errMotivoAutorizacionAjustes
	}
	proveedor := &proveedorConsultaAjustesCT{soporte: alta.soporte, pdp: alta.autorizador,
		material: alta.postgresql.proveedorMaterialConsultaAjustesReglas,
		motivo:   alta.postgresql.motivoConsultaAjustesReglas.Referencia, reloj: reloj}
	repositorio, err := ajustespg.NuevoRepositorioAjustesReglasCT(alta.postgresql.ejecucion, proveedor,
		organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	servicio, err := ajustesapp.NuevoServicio(repositorio, fuente, motivos, reloj)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	manejador, err := ajusteshttp.NuevoManejadorSoloLectura(proveedor, servicio)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: ajusteshttp.Ruta, Manejador: manejador}, nil
}

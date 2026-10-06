// Package telemetria contiene la observabilidad técnica de los procesos HTTP
// de VEC: el registro de acceso por petición, la ficha que acompaña a cada
// petición por su contexto y la identificación del binario que la atiende.
//
// Es una capa separada de la auditoría de uso de datos. No guarda identidad,
// cuerpos, cabeceras, parámetros de consulta ni valores de la ruta: solo la
// plantilla de la ruta, tiempos, tamaños y códigos cerrados. La auditoría
// nominal conserva su propia cadena, permisos y retención.
package telemetria

import (
	"runtime/debug"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

// Revision se fija en la compilación con
//
//	-ldflags "-X vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria.Revision=<hash>"
//
// Las compilaciones de despliegue usan -buildvcs=false, de modo que sin esta
// marca el binario no sabe de qué revisión procede.
var Revision string

// RevisionBinario devuelve la revisión marcada en la compilación o, si falta,
// la que Go incrusta desde el control de versiones. Lo no conforme pasa a
// "desconocida" con la misma regla que las incidencias técnicas.
func RevisionBinario() string {
	if v := domain.NormalizarVersionBinario(strings.TrimSpace(Revision)); v != domain.VersionBinarioDesconocida {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return domain.VersionBinarioDesconocida
	}
	for _, ajuste := range info.Settings {
		if ajuste.Key == "vcs.revision" {
			return domain.NormalizarVersionBinario(ajuste.Value)
		}
	}
	return domain.VersionBinarioDesconocida
}

// EntornoDe resuelve el entorno técnico. Un valor declarado expresamente
// (VEC_ENTORNO) manda y, si no pertenece a la lista cerrada, queda como
// "desconocido". Si no se declaró, se deduce del perfil de ejecución con el
// que el proceso ya arranca, que es una declaración de configuración y no una
// suposición.
func EntornoDe(declarado, perfilEjecucion string) string {
	declarado = strings.TrimSpace(declarado)
	if declarado != "" {
		return string(domain.NormalizarEntornoIncidenciaTecnica(declarado))
	}
	switch strings.ToLower(strings.TrimSpace(perfilEjecucion)) {
	case "desarrollo", "cidonia":
		return string(domain.EntornoIncidenciaDesarrollo)
	case "presentacion_rrhh", "presentacion":
		return string(domain.EntornoIncidenciaPresentacion)
	case "pruebas":
		return string(domain.EntornoIncidenciaPruebas)
	case "produccion":
		return string(domain.EntornoIncidenciaProduccion)
	default:
		return string(domain.EntornoIncidenciaDesconocido)
	}
}

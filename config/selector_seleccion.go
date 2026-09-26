package config

import "errors"

// Selección («Convoca integrado», fase 1): la solicitud de participación de
// una persona en una convocatoria y su consulta por RRHH. Como los demás
// selectores de despliegue, solo admite "true" y "false", exige la doble llave
// de desarrollo y, encendido, el catálogo de convocatorias de ejemplo. El
// arranque comprueba además que están instaladas sus migraciones (roles de
// Selección, AD3-89, AD3-90 y Selección 000001).
const (
	EnvSeleccionSolicitudesEnabled = "VEC_SELECCION_SOLICITUDES_ENABLED"
	// EnvSeleccionConvocatoriasSourcePath apunta al catálogo versionado de
	// convocatorias (plazo, turnos, requisitos, baremo y justificante) que
	// Selección publica al arrancar.
	EnvSeleccionConvocatoriasSourcePath = "VEC_SELECCION_CONVOCATORIAS_SOURCE_PATH"
)

var (
	ErrConfiguracionSeleccionSolicitudesSelector   = errors.New("config: selector de solicitudes de Seleccion invalido")
	ErrConfiguracionSeleccionSolicitudesActivacion = errors.New("config: solicitudes de Seleccion fuera del perfil de desarrollo o sin su catalogo")
)

// SeleccionSolicitudesDesarrolloActivo valida el selector sin abrir ficheros.
func (c Config) SeleccionSolicitudesDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.SeleccionSolicitudesEnabled,
		ErrConfiguracionSeleccionSolicitudesSelector, ErrConfiguracionSeleccionSolicitudesActivacion,
		catalogoRequerido{EnvSeleccionConvocatoriasSourcePath, c.SeleccionConvocatoriasSourcePath})
}

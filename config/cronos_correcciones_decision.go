package config

// EnvCronosCorreccionesDecisionEnabled selecciona las bandejas, decisiones
// e historia de correcciones de Cronos. No concede permisos.
const EnvCronosCorreccionesDecisionEnabled = "VEC_CRONOS_CORRECCIONES_DECISION_ENABLED"

// CronosCorreccionesDecisionDesarrolloActiva cierra sólo esta capacidad
// cuando el selector es inválido o Cronos empleado no está activo.
func (c Config) CronosCorreccionesDecisionDesarrolloActiva() bool {
	c = c.Normalize()
	activo, err := c.CronosEmpleadoDesarrolloActivo()
	return err == nil && activo && c.CronosCorreccionesDecisionEnabled == "true"
}

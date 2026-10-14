package config

// EnvCronosRecuperacionCorreccionEnabled selecciona la lectura del recibo
// original de una corrección propia. Requiere Cronos empleado y material
// privado propio; no concede permisos.
const EnvCronosRecuperacionCorreccionEnabled = "VEC_CRONOS_RECUPERACION_CORRECCION_ENABLED"

// CronosRecuperacionCorreccionDesarrolloActiva interpreta el selector de la
// capacidad opcional. Un valor inválido cierra sólo esta capacidad.
func (c Config) CronosRecuperacionCorreccionDesarrolloActiva() bool {
	c = c.Normalize()
	activo, err := c.CronosEmpleadoDesarrolloActivo()
	return err == nil && activo && c.CronosRecuperacionCorreccionEnabled == "true"
}

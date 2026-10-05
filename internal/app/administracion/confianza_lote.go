package administracion

// NuevaConfianzaLoteV3 compone la cadena V3 con una sola capacidad: la del lote
// ordinario de perfiles (AD190). Comparte PDP y fuentes con las demás cadenas
// del proceso; su clave HMAC es propia y la gobierna su procedimiento.
func NuevaConfianzaLoteV3(cfg ConfiguracionConfianzaPerfilesV3, deps DependenciasConfianzaPerfilesV3) (ConfianzaPerfilesV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{AudienciaLoteOrdinarioV3})
}

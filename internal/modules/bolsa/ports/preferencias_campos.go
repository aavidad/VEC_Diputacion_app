package ports

// CamposLecturaPreferenciasModalidades fija la proyección de ese tipo.
func CamposLecturaPreferenciasModalidades() []string {
	return []string{"version", "modalidades_guardadas", "modalidades_disponibles", "recibo_ref", "registrado_en"}
}

// CamposConcesionPreferenciasPropias amplía la concesión común. Cada lectura
// pide únicamente su proyección fija de bolsas o de modalidades.
func CamposConcesionPreferenciasPropias() []string {
	return []string{"version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas", "bolsas_retiradas", "revision_requerida", "modalidades_guardadas", "modalidades_disponibles", "recibo_ref", "registrado_en"}
}

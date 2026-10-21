package postgres

// consultaRRHHReintentable enumera las lecturas nominales que repiten el
// consumo completo ante una carrera de serialización (40001). El selector
// documental reutiliza el consumo del detalle y se reintenta igual que él.
func consultaRRHHReintentable(consulta string) bool {
	switch consulta {
	case consultaCuadroRRHHPostgreSQL, consultaDetalleRRHHPostgreSQL,
		consultaDetalleConBolsaRRHHPostgreSQL,
		consultaDetalleConSelectoresRRHHPostgreSQL,
		consultaDetalleConBolsaYSelectoresRRHHPostgreSQL,
		consultaSelectorIdentidadDocumentalRRHHPostgreSQL:
		return true
	}
	return false
}

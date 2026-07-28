package ports

// ExportarCanonReciboLecturaRRHHV2ParaVectorPostgreSQL es una frontera
// disponible solo durante las pruebas. Impide congelar bytes de un struct
// parcial: el Recibo debe haber superado antes toda la validación nominal V2.
func ExportarCanonReciboLecturaRRHHV2ParaVectorPostgreSQL(
	recibo ReciboLecturaRRHH,
) ([]byte, error) {
	if recibo.validarV2() != nil {
		return nil, ErrResultadoConsultaRRHHNoConfiable
	}
	return recibo.canonReciboLecturaRRHHV2()
}

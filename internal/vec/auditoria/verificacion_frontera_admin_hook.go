package auditoria

// VerificarCadenaFronteraAdminTecnicaV1 conserva la cadena anterior y coteja
// también la familia técnica de la frontera ADMIN, sin atribuirle actor V2.
func VerificarCadenaFronteraAdminTecnicaV1(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacion {
	return verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionFronteraAdminTecnicaV1)
}

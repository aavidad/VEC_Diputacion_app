package auditoria

// motivoErrorInstanteAuditoria propaga el fallo de parseo como motivo cerrado.
// El informe no conserva la fecha rechazada ni el texto de time.ParseError.
func motivoErrorInstanteAuditoria(err error) string {
	if err != nil {
		return MotivoInstanteAD171Invalido
	}
	return ""
}

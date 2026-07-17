package domain

// ClaveMotivoAutorizacionV2Valida comprueba exclusivamente el perfil opaco de
// la clave. No acredita que exista en un catalogo ni que haya sido generada con
// entropia suficiente; esas garantias pertenecen al servicio y al repositorio.
func ClaveMotivoAutorizacionV2Valida(clave string) bool {
	return claveOpacaMotivoAutorizacionV2Valida(clave)
}

package ports

// Los roles de cobertura amplían el mismo motor institucional Ed25519 de
// O3-03. No constituyen otra biblioteca de credenciales o raíces.
const (
	RolFuenteCobertura             RolAutoridadFuenteAnalisis = "fuente_cobertura"
	RolVerificadorCobertura        RolAutoridadFuenteAnalisis = "verificador_cobertura"
	RolPublicadorCatalogoCobertura RolAutoridadFuenteAnalisis = "publicador_catalogo_cobertura"
)

func rolAutoridadCoberturaValido(rol RolAutoridadFuenteAnalisis) bool {
	switch rol {
	case RolFuenteCobertura, RolVerificadorCobertura,
		RolPublicadorCatalogoCobertura:
		return true
	default:
		return false
	}
}

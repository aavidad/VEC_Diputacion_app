package administracion

// Las audiencias son las dos entradas cerradas publicadas por AUT43/AD185.
// Sus claves HMAC y raíz proceden de la misma configuración/KMS del proceso.
const (
	AudienciaUsuariosListarV3    = "vec.admin.usuarios.listar.v1"
	AudienciaUsuariosConsultarV3 = "vec.admin.usuarios.consultar.v1"
)

type ConfiguracionConfianzaUsuariosV3 = ConfiguracionConfianzaPerfilesV3
type DependenciasConfianzaUsuariosV3 = DependenciasConfianzaPerfilesV3
type ConfianzaUsuariosV3 = ConfianzaPerfilesV3

// NuevaConfianzaUsuariosV3 reutiliza toda la cadena PDP durable, COSE,
// verificador de confianza y HMAC. No consulta SQL, firma ni aprovisiona claves
// al construir; exige exactamente las dos audiencias del lector de usuarios.
func NuevaConfianzaUsuariosV3(cfg ConfiguracionConfianzaUsuariosV3, deps DependenciasConfianzaUsuariosV3) (ConfianzaUsuariosV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{AudienciaUsuariosListarV3, AudienciaUsuariosConsultarV3})
}

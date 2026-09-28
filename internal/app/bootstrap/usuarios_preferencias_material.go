package bootstrap

import usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"

const envUsuariosPreferenciasDesarrollo = "VEC_USUARIOS_PREFERENCIAS_ENABLED"

const (
	audienciaConsultaPreferenciasUsuarios      = usuariosports.AudienciaConsultarPreferencias
	audienciaActualizacionPreferenciasUsuarios = usuariosports.AudienciaActualizarPreferencias
)

// Cada efecto conserva su audiencia y clave HMAC derivada del gobierno V3
// común. El selector sólo incorpora ambas al catálogo si la composición
// autenticada de Usuarios se ha solicitado y puede completar su preflight.
func descriptoresMaterialPreferenciasUsuariosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: audienciaConsultaPreferenciasUsuarios, Dominio: "vec.usuarios.preferencias.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-consulta:", ProveedorNominal: "proveedor-material-usuarios-preferencias-consulta"},
		{Audiencia: audienciaActualizacionPreferenciasUsuarios, Dominio: "vec.usuarios.preferencias.actualizar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-actualizacion:", ProveedorNominal: "proveedor-material-usuarios-preferencias-actualizacion"},
	}
}

package bootstrap

import usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"

const envUsuariosPreferenciasDesarrollo = "VEC_USUARIOS_PREFERENCIAS_ENABLED"

const (
	audienciaConsultaPreferenciasUsuariosInterna      = usuariosports.AudienciaConsultarPreferenciasInterna
	audienciaActualizacionPreferenciasUsuariosInterna = usuariosports.AudienciaActualizarPreferenciasInterna
	audienciaConsultaPreferenciasUsuariosExterna      = usuariosports.AudienciaConsultarPreferenciasExterna
	audienciaActualizacionPreferenciasUsuariosExterna = usuariosports.AudienciaActualizarPreferenciasExterna
)

// Cada efecto conserva su audiencia y clave HMAC derivada del gobierno V3
// común. El selector sólo incorpora ambas al catálogo si la composición
// autenticada de Usuarios se ha solicitado y puede completar su preflight.
func descriptoresMaterialPreferenciasUsuariosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: audienciaConsultaPreferenciasUsuariosInterna, Dominio: "vec.usuarios.preferencias.consultar.interna.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-consulta-interna:", ProveedorNominal: "proveedor-material-usuarios-preferencias-consulta-interna"},
		{Audiencia: audienciaActualizacionPreferenciasUsuariosInterna, Dominio: "vec.usuarios.preferencias.actualizar.interna.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-actualizacion-interna:", ProveedorNominal: "proveedor-material-usuarios-preferencias-actualizacion-interna"},
		{Audiencia: audienciaConsultaPreferenciasUsuariosExterna, Dominio: "vec.usuarios.preferencias.consultar.externa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-consulta-externa:", ProveedorNominal: "proveedor-material-usuarios-preferencias-consulta-externa"},
		{Audiencia: audienciaActualizacionPreferenciasUsuariosExterna, Dominio: "vec.usuarios.preferencias.actualizar.externa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:usuarios-preferencias-actualizacion-externa:", ProveedorNominal: "proveedor-material-usuarios-preferencias-actualizacion-externa"},
	}
}

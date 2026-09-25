package bootstrap

import cronosapp "vec-diputacion-granada/internal/modules/cronos/application"

// Cronos se integra como Dietas en el gobierno V3 único de desarrollo: una
// clave HMAC derivada por audiencia de consumo (AD3-53 y AD3-70), con dominio y
// prefijo propios, bajo la misma raíz y el mismo publicador de Contratación.

// materialCronosDesdeCTDesarrollo transporta los ocho proveedores ya
// gobernados, uno por acción de la persona empleada. No expone claves.
type materialCronosDesdeCTDesarrollo struct {
	marcaje, disponibilidad, recibo, saldo              *proveedorMaterialAltaContratacionTemporalDesarrollo
	movimientos, correccion, permisos, solicitudPermiso *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (m materialCronosDesdeCTDesarrollo) completo() bool {
	return m.marcaje != nil && m.disponibilidad != nil && m.recibo != nil && m.saldo != nil &&
		m.movimientos != nil && m.correccion != nil && m.permisos != nil && m.solicitudPermiso != nil
}

// materialCronosDesdeProveedores asigna los proveedores en el orden de
// audienciasCronosEmpleadoDesarrollo.
func materialCronosDesdeProveedores(p [8]*proveedorMaterialAltaContratacionTemporalDesarrollo) materialCronosDesdeCTDesarrollo {
	return materialCronosDesdeCTDesarrollo{marcaje: p[0], disponibilidad: p[1], recibo: p[2], saldo: p[3],
		movimientos: p[4], correccion: p[5], permisos: p[6], solicitudPermiso: p[7]}
}

// cronosEmpleadoSolicitado sólo refleja el selector; la validación completa
// (doble llave de desarrollo) la hace la composición de las rutas.
func cronosEmpleadoSolicitado(selector string) bool { return selector == "true" }

func audienciasCronosEmpleadoDesarrollo() [8]string {
	return [8]string{
		cronosapp.AudienciaMarcajePropio,
		cronosapp.AudienciaDisponibilidadMarcajeRemoto,
		cronosapp.AudienciaRecuperacionMarcajeRemoto,
		cronosapp.AudienciaConsultaSaldoPropio,
		cronosapp.AudienciaConsultaMovimientosPropios,
		cronosapp.AudienciaSolicitudCorreccionPropia,
		cronosapp.AudienciaConsultaPermisosPropios,
		cronosapp.AudienciaSolicitudPermisoPropio,
	}
}

func descriptoresMaterialCronosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: cronosapp.AudienciaMarcajePropio, Dominio: "vec.cronos.marcaje-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-marcaje:", ProveedorNominal: "proveedor-material-cronos-marcaje"},
		{Audiencia: cronosapp.AudienciaDisponibilidadMarcajeRemoto, Dominio: "vec.cronos.marcaje-remoto-disponibilidad.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-disponibilidad:", ProveedorNominal: "proveedor-material-cronos-disponibilidad"},
		{Audiencia: cronosapp.AudienciaRecuperacionMarcajeRemoto, Dominio: "vec.cronos.marcaje-remoto-recibo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-recibo:", ProveedorNominal: "proveedor-material-cronos-recibo"},
		{Audiencia: cronosapp.AudienciaConsultaSaldoPropio, Dominio: "vec.cronos.saldo-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-saldo:", ProveedorNominal: "proveedor-material-cronos-saldo"},
		{Audiencia: cronosapp.AudienciaConsultaMovimientosPropios, Dominio: "vec.cronos.movimientos-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-movimientos:", ProveedorNominal: "proveedor-material-cronos-movimientos"},
		{Audiencia: cronosapp.AudienciaSolicitudCorreccionPropia, Dominio: "vec.cronos.correccion-propia.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion:", ProveedorNominal: "proveedor-material-cronos-correccion"},
		{Audiencia: cronosapp.AudienciaConsultaPermisosPropios, Dominio: "vec.cronos.permisos-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-permisos:", ProveedorNominal: "proveedor-material-cronos-permisos"},
		{Audiencia: cronosapp.AudienciaSolicitudPermisoPropio, Dominio: "vec.cronos.permiso-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-permiso:", ProveedorNominal: "proveedor-material-cronos-permiso"},
	}
}

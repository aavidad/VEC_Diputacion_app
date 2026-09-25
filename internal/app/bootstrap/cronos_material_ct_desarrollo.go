package bootstrap

import cronosapp "vec-diputacion-granada/internal/modules/cronos/application"

// Cronos se integra como Dietas en el gobierno V3 único de desarrollo: una
// clave HMAC derivada por audiencia de consumo (AD3-53), con dominio y
// prefijo propios, bajo la misma raíz y el mismo publicador de Contratación.

// materialCronosDesdeCTDesarrollo transporta los cuatro proveedores ya
// gobernados, uno por acción de la persona empleada. No expone claves.
type materialCronosDesdeCTDesarrollo struct {
	marcaje, disponibilidad, recibo, saldo *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (m materialCronosDesdeCTDesarrollo) completo() bool {
	return m.marcaje != nil && m.disponibilidad != nil && m.recibo != nil && m.saldo != nil
}

// cronosEmpleadoSolicitado sólo refleja el selector; la validación completa
// (doble llave de desarrollo) la hace la composición de las rutas.
func cronosEmpleadoSolicitado(selector string) bool { return selector == "true" }

func audienciasCronosEmpleadoDesarrollo() [4]string {
	return [4]string{
		cronosapp.AudienciaMarcajePropio,
		cronosapp.AudienciaDisponibilidadMarcajeRemoto,
		cronosapp.AudienciaRecuperacionMarcajeRemoto,
		cronosapp.AudienciaConsultaSaldoPropio,
	}
}

func descriptoresMaterialCronosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: cronosapp.AudienciaMarcajePropio, Dominio: "vec.cronos.marcaje-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-marcaje:", ProveedorNominal: "proveedor-material-cronos-marcaje"},
		{Audiencia: cronosapp.AudienciaDisponibilidadMarcajeRemoto, Dominio: "vec.cronos.marcaje-remoto-disponibilidad.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-disponibilidad:", ProveedorNominal: "proveedor-material-cronos-disponibilidad"},
		{Audiencia: cronosapp.AudienciaRecuperacionMarcajeRemoto, Dominio: "vec.cronos.marcaje-remoto-recibo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-recibo:", ProveedorNominal: "proveedor-material-cronos-recibo"},
		{Audiencia: cronosapp.AudienciaConsultaSaldoPropio, Dominio: "vec.cronos.saldo-propio.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-saldo:", ProveedorNominal: "proveedor-material-cronos-saldo"},
	}
}

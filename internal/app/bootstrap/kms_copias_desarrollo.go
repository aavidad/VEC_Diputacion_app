package bootstrap

import (
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

const dominioCopiasDesarrollo = "vec.kms.desarrollo.copias.componentes.v1"

// ProtectorCopiasDesarrollo usa el proveedor ya cargado sin exponer su clave.
func (c *ComposicionSeguridadDesarrollo) ProtectorCopiasDesarrollo(ref, version string, limite int64) (ports.Protector, error) {
	if c == nil || c.emisorKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return protectorCopiasDesdeEnvoltura(c.emisorKMS.claveEnvoltura, ref, version, limite)
}

// NuevoProtectorCopiasDesarrollo permite la CLI sintética offline con el
// material maestro externo existente. No crea material durante el arranque.
func NuevoProtectorCopiasDesarrollo(maestra [32]byte, ref, version string, limite int64) (ports.Protector, error) {
	if maestra == [32]byte{} {
		return nil, ErrKMSDesarrolloNoDisponible
	}
	envoltura := derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")
	defer clear(envoltura[:])
	return protectorCopiasDesdeEnvoltura(envoltura, ref, version, limite)
}
func protectorCopiasDesdeEnvoltura(envoltura [32]byte, ref, version string, limite int64) (ports.Protector, error) {
	k := derivarClaveDesarrollo(envoltura, dominioCopiasDesarrollo)
	defer clear(k[:])
	return adapter.NuevoProtectorJWE(k, ref, version, limite)
}

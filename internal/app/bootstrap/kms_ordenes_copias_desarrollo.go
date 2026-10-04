package bootstrap

import (
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

// La separación de clave impide que un componente archivado se pueda presentar
// como una orden de mantenimiento. Referencia y versión vienen de configuración.
const dominioOrdenesCopiasDesarrollo = "vec.kms.desarrollo.copias.orden.v1"

func (c *ComposicionSeguridadDesarrollo) ProtectorOrdenCopiasDesarrollo(ref, version string, limite int64) (ports.Protector, error) {
	if c == nil || c.emisorKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return protectorOrdenCopiasDesdeEnvoltura(c.emisorKMS.claveEnvoltura, ref, version, limite)
}

// NuevoProtectorOrdenCopiasDesarrollo conserva la derivación del proveedor
// existente para un ejecutor local con material externo; no genera claves.
func NuevoProtectorOrdenCopiasDesarrollo(maestra [32]byte, ref, version string, limite int64) (ports.Protector, error) {
	if maestra == [32]byte{} {
		return nil, ErrKMSDesarrolloNoDisponible
	}
	envoltura := derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")
	defer clear(envoltura[:])
	return protectorOrdenCopiasDesdeEnvoltura(envoltura, ref, version, limite)
}

func protectorOrdenCopiasDesdeEnvoltura(envoltura [32]byte, ref, version string, limite int64) (ports.Protector, error) {
	clave := derivarClaveDesarrollo(envoltura, dominioOrdenesCopiasDesarrollo)
	defer clear(clave[:])
	return adapter.NuevoProtectorJWE(clave, ref, version, limite)
}

package bootstrap

import docports "vec-diputacion-granada/internal/vec/documentos/ports"

// Documentos se integra como Cronos, Dietas y Personal B2 en el gobierno V3
// único de desarrollo: vec-server publica una clave HMAC derivada para la
// audiencia vec_documentos.operacion.v1 (admitida por AD3-60), con dominio y
// prefijo propios, bajo la misma raíz y el mismo publicador de Contratación.

// documentosSolicitados sólo refleja el selector; la validación completa
// (doble llave de desarrollo) la hace la composición de las rutas.
func documentosSolicitados(selector string) bool { return selector == "true" }

func descriptoresMaterialDocumentosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{{
		Audiencia:        docports.AudienciaV3,
		Dominio:          "vec.documentos.operacion.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:documentos-operacion:",
		ProveedorNominal: "proveedor-material-documentos-operacion",
	}}
}

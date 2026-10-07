package bootstrap

import meritos "vec-diputacion-granada/internal/modules/meritos/application"

// Declaración y consulta propia internas tienen claves de consumo separadas.
// Ser publicable no monta rutas ni concede un perfil funcional.
func descriptoresMaterialMeritosInternosDesarrollo() [2]descriptorMaterialConsumidorV3Desarrollo {
	_, declaracion := meritos.EspecificacionAutorizacion("meritos.hecho.declarar")
	return [2]descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: declaracion, Dominio: "vec.meritos.desarrollo.declarar.capacidad-v3", Prefijo: "clave:capacidad:meritos:declarar:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: meritos.AudienciaConsultaPropia, Dominio: "vec.meritos.desarrollo.consulta-propia.capacidad-v3", Prefijo: "clave:capacidad:meritos:consulta-propia:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}
}

package bootstrap

// El descriptor fija un dominio de derivación y prefijo propios. Se entrega
// sólo al publicador completo del gobierno común; no es una concesión.
func descriptorMaterialRPTUsosFixture() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        audienciaUsosCategoriasRPTFixture,
		Dominio:          "vec.catalogos.rpt.usos.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:rpt-usos:",
		ProveedorNominal: "proveedor-material-rpt-usos-fixture",
	}
}

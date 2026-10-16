package bootstrap

const (
	audienciaSubidaLocalActoModulo     = "vec_documentos.subida_local.acto_modulo.v1"
	audienciaSubidaLocalDescargaPropia = "vec_documentos.subida_local.descarga_propia.v1"
)

func descriptoresMaterialSubidaLocalExternaV3() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{
			Audiencia:        audienciaSubidaLocalActoModulo,
			Dominio:          "vec.documentos.subida-local.acto-modulo.desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:documentos-subida-local-acto:",
			ProveedorNominal: "proveedor-material-documentos-subida-local-acto",
		},
		{
			Audiencia:        audienciaSubidaLocalDescargaPropia,
			Dominio:          "vec.documentos.subida-local.descarga-propia.desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:documentos-subida-local-descarga:",
			ProveedorNominal: "proveedor-material-documentos-subida-local-descarga",
		},
	}
}

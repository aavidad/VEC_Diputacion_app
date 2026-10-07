package domain

func dimensionesBootstrapV3Validas(dimensiones []string) bool {
	return len(dimensiones) == 1 && dimensiones[0] == "organizacion_ref" ||
		len(dimensiones) == 2 && dimensiones[0] == "organizacion_ref" && dimensiones[1] == "unidad_ref"
}

func valoresBootstrapV3Validos(valores []string) bool {
	if len(valores) == 0 || len(valores) > 64 {
		return false
	}
	ultimo := ""
	for _, valor := range valores {
		if !textoFijoBootstrap(valor) || valor <= ultimo {
			return false
		}
		ultimo = valor
	}
	return true
}

func limitesAmbitoBootstrapV3Validos(rol RolGobernadoBootstrapAdministracionV3) bool {
	if !dimensionesBootstrapV3Validas(rol.DimensionesAmbito) || len(rol.AmbitosFijos) != len(rol.DimensionesAmbito) ||
		rol.UnidadRequerida && len(rol.DimensionesAmbito) != 2 {
		return false
	}
	for i, ambito := range rol.AmbitosFijos {
		if ambito.Clave != rol.DimensionesAmbito[i] || !valoresBootstrapV3Validos(ambito.Valores) ||
			ambito.Clave == "organizacion_ref" && len(ambito.Valores) != 1 {
			return false
		}
	}
	return true
}

func ambitosBootstrapV3Validos(ambitos []AmbitoBootstrapAdministracionV3, rol RolGobernadoBootstrapAdministracionV3) bool {
	if len(ambitos) < 1 || len(ambitos) > len(rol.DimensionesAmbito) || rol.UnidadRequerida && len(ambitos) != 2 {
		return false
	}
	for i, ambito := range ambitos {
		if ambito.Dimension != rol.DimensionesAmbito[i] || !evidenciaBootstrapValida(ambito.Fuente) ||
			!valoresBootstrapV3Validos(ambito.Valores) || len(ambito.Valores) != len(rol.AmbitosFijos[i].Valores) {
			return false
		}
		for j, valor := range ambito.Valores {
			if valor != rol.AmbitosFijos[i].Valores[j] {
				return false
			}
		}
	}
	return true
}

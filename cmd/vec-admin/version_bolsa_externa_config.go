package main

import (
	"encoding/json"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
)

// Configuración privada opcional: dos LOGIN y dos materiales propios.
type configuracionVersionBolsaExternaPrivada configuracionGobiernoRolesPrivada

func cargarConfiguracionVersionBolsaExternaPrivada(ruta string, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	efectos []efectoConfigurado, gobierno *configuracionGobiernoRolesPrivada,
	versionBolsa *configuracionVersionBolsaPrivada, b103 *configuracionVersionBolsaB103Privada) (configuracionVersionBolsaExternaPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionVersionBolsaExternaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionVersionBolsaExternaPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil ||
		validarConfiguracionVersionBolsaExternaPrivada(c, base, u, runtime, lote, plan, efectos, gobierno, versionBolsa, b103) != nil {
		return configuracionVersionBolsaExternaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionVersionBolsaExternaPrivada(c configuracionVersionBolsaExternaPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado,
	gobierno *configuracionGobiernoRolesPrivada, versionBolsa *configuracionVersionBolsaPrivada,
	b103 *configuracionVersionBolsaB103Privada) error {
	var otrasRutas []string
	var otrasConfianzas []json.RawMessage
	for _, otra := range []*configuracionGobiernoRolesPrivada{gobierno, convertirVersionBolsaGobierno(versionBolsa), convertirB103Gobierno(b103)} {
		if otra != nil {
			otrasRutas = append(otrasRutas, otra.PoolGobierno, otra.PoolCatalogo)
			otrasConfianzas = append(otrasConfianzas, otra.ConfianzaJSON)
		}
	}
	return validarConfiguracionGobiernoPrivada(configuracionGobiernoRolesPrivada(c), base, u, runtime, lote, plan, efectos,
		[2]string{pg.AudienciaVersionarRolBolsaExternaProponer, pg.AudienciaVersionarRolBolsaExternaAprobar},
		otrasRutas, otrasConfianzas)
}

func convertirVersionBolsaGobierno(c *configuracionVersionBolsaPrivada) *configuracionGobiernoRolesPrivada {
	if c == nil {
		return nil
	}
	v := configuracionGobiernoRolesPrivada(*c)
	return &v
}

func convertirB103Gobierno(c *configuracionVersionBolsaB103Privada) *configuracionGobiernoRolesPrivada {
	if c == nil {
		return nil
	}
	v := configuracionGobiernoRolesPrivada(*c)
	return &v
}

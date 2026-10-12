package main

import (
	"encoding/json"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
)

// El overlay B103 conserva la forma B1, con dos materiales y dos LOGIN nuevos.
// Se carga sólo desde VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE.
type configuracionVersionBolsaB103Privada configuracionGobiernoRolesPrivada

func cargarConfiguracionVersionBolsaB103Privada(ruta string, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	efectos []efectoConfigurado, gobierno *configuracionGobiernoRolesPrivada,
	versionBolsa *configuracionVersionBolsaPrivada) (configuracionVersionBolsaB103Privada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionVersionBolsaB103Privada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionVersionBolsaB103Privada
	if decodificarConfiguracionPrivada(b, &c) != nil ||
		validarConfiguracionVersionBolsaB103Privada(c, base, u, runtime, lote, plan, efectos, gobierno, versionBolsa) != nil {
		return configuracionVersionBolsaB103Privada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionVersionBolsaB103Privada(c configuracionVersionBolsaB103Privada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado,
	gobierno *configuracionGobiernoRolesPrivada, versionBolsa *configuracionVersionBolsaPrivada) error {
	var otrasRutas []string
	var otrasConfianzas []json.RawMessage
	if gobierno != nil {
		otrasRutas = append(otrasRutas, gobierno.PoolGobierno, gobierno.PoolCatalogo)
		otrasConfianzas = append(otrasConfianzas, gobierno.ConfianzaJSON)
	}
	if versionBolsa != nil {
		otrasRutas = append(otrasRutas, versionBolsa.PoolGobierno, versionBolsa.PoolCatalogo)
		otrasConfianzas = append(otrasConfianzas, versionBolsa.ConfianzaJSON)
	}
	return validarConfiguracionGobiernoPrivada(configuracionGobiernoRolesPrivada(c),
		base, u, runtime, lote, plan, efectos,
		[2]string{pg.AudienciaVersionarRolBolsaProponer, pg.AudienciaVersionarRolBolsaAprobar},
		otrasRutas, otrasConfianzas)
}

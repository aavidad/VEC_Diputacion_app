package main

import (
	"encoding/json"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
)

// Este overlay privado habilita sólo el gobierno nominal B1. Sus dos LOGIN y
// materiales son distintos de los usados para crear definiciones en #886.
type configuracionVersionBolsaPrivada configuracionGobiernoRolesPrivada

func cargarConfiguracionVersionBolsaPrivada(ruta string, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	efectos []efectoConfigurado, gobierno *configuracionGobiernoRolesPrivada) (configuracionVersionBolsaPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionVersionBolsaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionVersionBolsaPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil ||
		validarConfiguracionVersionBolsaPrivada(c, base, u, runtime, lote, plan, efectos, gobierno) != nil {
		return configuracionVersionBolsaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionVersionBolsaPrivada(c configuracionVersionBolsaPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado,
	gobierno *configuracionGobiernoRolesPrivada) error {
	var otrasRutas []string
	var otrasConfianzas []json.RawMessage
	if gobierno != nil {
		otrasRutas = []string{gobierno.PoolGobierno, gobierno.PoolCatalogo}
		otrasConfianzas = []json.RawMessage{gobierno.ConfianzaJSON}
	}
	return validarConfiguracionGobiernoPrivada(configuracionGobiernoRolesPrivada(c),
		base, u, runtime, lote, plan, efectos,
		[2]string{pg.AudienciaVersionarRolBolsaProponer, pg.AudienciaVersionarRolBolsaAprobar},
		otrasRutas, otrasConfianzas)
}

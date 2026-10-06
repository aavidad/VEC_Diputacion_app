package main

import (
	"encoding/json"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/domain"
)

// configuracionCargosPrivada tiene pool y LOGIN propios (exclusivo del grupo
// vec_personal_ejecutor, que exige Personal28/AD166) y una sola capacidad V3,
// la de publicación de cargos competenciales (conjunto 3 de AD204).
type configuracionCargosPrivada struct {
	Pool          string                           `json:"pool"`
	ConfianzaJSON json.RawMessage                  `json:"confianza"`
	Motivo        domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

func cargarConfiguracionCargosPrivada(ruta string, lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) (configuracionCargosPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionCargosPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionCargosPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionCargosPrivada(c, lote, plan, base, u, runtime) != nil {
		return configuracionCargosPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

// validarConfiguracionCargosPrivada exige pool y material propios, distintos de
// todos los demás (también del lote y del plan si están), una sola capacidad
// con la misma raíz y un motivo del catálogo común.
func validarConfiguracionCargosPrivada(c configuracionCargosPrivada, lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) error {
	confianzas, otrosPools := []json.RawMessage{base.ConfianzaJSON, u.ConfianzaJSON}, []string{}
	if lote != nil {
		if validarConfiguracionLotePrivada(*lote, base, u, runtime) != nil {
			return errConfiguracionPrivadaPerfiles
		}
		confianzas, otrosPools = append(confianzas, lote.ConfianzaJSON), append(otrosPools, lote.PoolLote)
	}
	if plan != nil {
		if validarConfiguracionPlanFirmaPrivada(*plan, lote, base, u, runtime) != nil {
			return errConfiguracionPrivadaPerfiles
		}
		confianzas, otrosPools = append(confianzas, plan.ConfianzaJSON), append(otrosPools, plan.Pool)
	}
	materiales := map[string]bool{}
	for _, raw := range confianzas {
		otros, err := decodificarMetadatosConfianzaPerfiles(raw)
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, e := range otros.EntradasCapacidad {
			materiales[e.MaterialArchivo] = true
		}
	}
	if !rutaPrivadaPerfilesValida(c.Pool) || contieneClavePrivadaInline(c.ConfianzaJSON) ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Motivo.CatalogoID != base.CatalogoMotivosID {
		return errConfiguracionPrivadaPerfiles
	}
	for _, ruta := range append([]string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos,
		base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, base.Pools.ActosADMIN,
		base.Pools.AuditoriaFrontera, base.Firmante.ClavePrivadaArchivo, base.Identidad.RutaConfiguracionHMAC,
		u.PoolLector, u.PoolIntentos, u.PoolSelector, u.PoolFronteraTecnica, runtime.PoolContexto, runtime.FuenteIdentificadoresArchivo}, otrosPools...) {
		if ruta == c.Pool {
			return errConfiguracionPrivadaPerfiles
		}
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil || len(meta.EntradasCapacidad) != 1 || meta.EntradasCapacidad[0].Audiencia != administracion.AudienciaCargoCompetencialV3 ||
		meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 || meta.Raiz.ClaveID != base.Firmante.ClaveID ||
		meta.Raiz.Audiencia != base.Firmante.Audiencia || meta.EntradasCapacidad[0].MaterialArchivo == c.Pool ||
		materiales[meta.EntradasCapacidad[0].MaterialArchivo] {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

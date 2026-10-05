package main

import (
	"encoding/json"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/domain"
)

// configuracionPlanFirmaPrivada tiene pool y LOGIN propios (grupo
// vec_plan_firma_gobierno_ejecutor, AD200) y una sola capacidad V3, la del
// gobierno del plan (conjunto 2 de AD202).
type configuracionPlanFirmaPrivada struct {
	Pool          string                           `json:"pool"`
	ConfianzaJSON json.RawMessage                  `json:"confianza"`
	Motivo        domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

func cargarConfiguracionPlanFirmaPrivada(ruta string, lote configuracionLotePrivada, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) (configuracionPlanFirmaPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionPlanFirmaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionPlanFirmaPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionPlanFirmaPrivada(c, lote, base, u, runtime) != nil {
		return configuracionPlanFirmaPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

// validarConfiguracionPlanFirmaPrivada exige pool y material propios, distintos
// de todos los demás, una sola capacidad (la del gobierno del plan) con la
// misma raíz y un motivo del catálogo común.
func validarConfiguracionPlanFirmaPrivada(p configuracionPlanFirmaPrivada, lote configuracionLotePrivada, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) error {
	if validarConfiguracionLotePrivada(lote, base, u, runtime) != nil {
		return errConfiguracionPrivadaPerfiles
	}
	materiales := map[string]bool{}
	for _, raw := range []json.RawMessage{base.ConfianzaJSON, u.ConfianzaJSON, lote.ConfianzaJSON} {
		otros, err := decodificarMetadatosConfianzaPerfiles(raw)
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, e := range otros.EntradasCapacidad {
			materiales[e.MaterialArchivo] = true
		}
	}
	if !rutaPrivadaPerfilesValida(p.Pool) || contieneClavePrivadaInline(p.ConfianzaJSON) ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || p.Motivo.CatalogoID != base.CatalogoMotivosID {
		return errConfiguracionPrivadaPerfiles
	}
	for _, ruta := range []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos,
		base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, base.Pools.ActosADMIN,
		base.Pools.AuditoriaFrontera, base.Firmante.ClavePrivadaArchivo, base.Identidad.RutaConfiguracionHMAC,
		u.PoolLector, u.PoolIntentos, u.PoolSelector, u.PoolFronteraTecnica, runtime.PoolContexto, runtime.FuenteIdentificadoresArchivo, lote.PoolLote} {
		if ruta == p.Pool {
			return errConfiguracionPrivadaPerfiles
		}
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(p.ConfianzaJSON)
	if err != nil || len(meta.EntradasCapacidad) != 1 || meta.EntradasCapacidad[0].Audiencia != administracion.AudienciaGobiernoPlanFirmaV3 ||
		meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 || meta.Raiz.ClaveID != base.Firmante.ClaveID ||
		meta.Raiz.Audiencia != base.Firmante.Audiencia || meta.EntradasCapacidad[0].MaterialArchivo == p.Pool ||
		materiales[meta.EntradasCapacidad[0].MaterialArchivo] {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

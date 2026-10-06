package main

import (
	"encoding/json"

	"vec-diputacion-granada/internal/vec/domain"
)

// configuracionEfectoPrivada es el archivo privado de un efecto nominal de
// vec-admin (cargos competenciales, certificados nominales): pool y LOGIN
// propios, exclusivos del grupo que exige la fachada del efecto, una sola
// capacidad V3 (la de su audiencia) y el motivo del catálogo común.
type configuracionEfectoPrivada struct {
	Pool          string                           `json:"pool"`
	ConfianzaJSON json.RawMessage                  `json:"confianza"`
	Motivo        domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

func cargarConfiguracionEfectoPrivada(ruta, audiencia string, otros []configuracionEfectoPrivada, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN) (configuracionEfectoPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionEfectoPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionEfectoPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionEfectoPrivada(c, audiencia, otros, lote, plan, base, u, runtime) != nil {
		return configuracionEfectoPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

// validarConfiguracionEfectoPrivada exige pool y material propios, distintos
// de todos los demás (también del lote, del plan y de los otros efectos), una
// sola capacidad de la audiencia del efecto con la misma raíz y un motivo del
// catálogo común.
func validarConfiguracionEfectoPrivada(c configuracionEfectoPrivada, audiencia string, otros []configuracionEfectoPrivada,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) error {
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
	for _, o := range otros {
		confianzas, otrosPools = append(confianzas, o.ConfianzaJSON), append(otrosPools, o.Pool)
	}
	materiales := map[string]bool{}
	for _, raw := range confianzas {
		m, err := decodificarMetadatosConfianzaPerfiles(raw)
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, e := range m.EntradasCapacidad {
			materiales[e.MaterialArchivo] = true
		}
	}
	if audiencia == "" || !rutaPrivadaPerfilesValida(c.Pool) || contieneClavePrivadaInline(c.ConfianzaJSON) ||
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
	if err != nil || len(meta.EntradasCapacidad) != 1 || meta.EntradasCapacidad[0].Audiencia != audiencia ||
		meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 || meta.Raiz.ClaveID != base.Firmante.ClaveID ||
		meta.Raiz.Audiencia != base.Firmante.Audiencia || meta.EntradasCapacidad[0].MaterialArchivo == c.Pool ||
		materiales[meta.EntradasCapacidad[0].MaterialArchivo] {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

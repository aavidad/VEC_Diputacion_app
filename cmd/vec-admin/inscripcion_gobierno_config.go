package main

import (
	"encoding/json"

	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

// Los archivos privados sólo nombran pools, confianza y motivos ya
// aprovisionados. Su presencia no instala SQL ni concede un rol al arrancar.
type configuracionGobiernoInscripcionPrivada struct {
	PoolGobierno  string                                      `json:"pool_gobierno"`
	PoolCatalogo  string                                      `json:"pool_catalogo"`
	ConfianzaJSON json.RawMessage                             `json:"confianza"`
	Motivos       map[string]domain.ReferenciaEntradaCatalogo `json:"motivos"`
}

func cargarConfiguracionGobiernoInscripcionPrivada(ruta string, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	efectos []efectoConfigurado, gobierno *configuracionGobiernoRolesPrivada) (configuracionGobiernoInscripcionPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionGobiernoInscripcionPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionGobiernoInscripcionPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil ||
		validarConfiguracionGobiernoInscripcionPrivada(c, base, u, runtime, lote, plan, efectos, gobierno) != nil {
		return configuracionGobiernoInscripcionPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionGobiernoInscripcionPrivada(c configuracionGobiernoInscripcionPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado,
	gobierno *configuracionGobiernoRolesPrivada) error {
	if !rutaPrivadaPerfilesValida(c.PoolGobierno) || !rutaPrivadaPerfilesValida(c.PoolCatalogo) ||
		c.PoolGobierno == c.PoolCatalogo || contieneClavePrivadaInline(c.ConfianzaJSON) || len(c.Motivos) != 2 ||
		!motivoGobiernoRolesValido(c.Motivos[postgres.AudienciaGobiernoInscripcionProponer], base.CatalogoMotivosID) ||
		!motivoGobiernoRolesValido(c.Motivos[postgres.AudienciaGobiernoInscripcionAprobar], base.CatalogoMotivosID) {
		return errConfiguracionPrivadaPerfiles
	}
	for clave := range c.Motivos {
		if clave != postgres.AudienciaGobiernoInscripcionProponer &&
			clave != postgres.AudienciaGobiernoInscripcionAprobar {
			return errConfiguracionPrivadaPerfiles
		}
	}
	rutas := []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos,
		base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN,
		base.Pools.ActosADMIN, base.Pools.AuditoriaFrontera, base.Firmante.ClavePrivadaArchivo,
		base.Identidad.RutaConfiguracionHMAC, u.PoolLector, u.PoolIntentos, u.PoolSelector,
		u.PoolFronteraTecnica, runtime.PoolContexto, runtime.FuenteIdentificadoresArchivo}
	confianzas := []json.RawMessage{base.ConfianzaJSON, u.ConfianzaJSON}
	if lote != nil {
		rutas = append(rutas, lote.PoolLote)
		confianzas = append(confianzas, lote.ConfianzaJSON)
	}
	if plan != nil {
		rutas = append(rutas, plan.Pool)
		confianzas = append(confianzas, plan.ConfianzaJSON)
	}
	for _, e := range efectos {
		rutas = append(rutas, e.cfg.Pool)
		confianzas = append(confianzas, e.cfg.ConfianzaJSON)
	}
	if gobierno != nil {
		rutas = append(rutas, gobierno.PoolGobierno, gobierno.PoolCatalogo)
		confianzas = append(confianzas, gobierno.ConfianzaJSON)
	}
	for _, ruta := range rutas {
		if ruta != "" && (ruta == c.PoolGobierno || ruta == c.PoolCatalogo) {
			return errConfiguracionPrivadaPerfiles
		}
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil || len(meta.EntradasCapacidad) != 2 ||
		meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 ||
		meta.Raiz.ClaveID != base.Firmante.ClaveID || meta.Raiz.Audiencia != base.Firmante.Audiencia {
		return errConfiguracionPrivadaPerfiles
	}
	materiales := map[string]bool{}
	for _, raw := range confianzas {
		otro, err := decodificarMetadatosConfianzaPerfiles(raw)
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, entrada := range otro.EntradasCapacidad {
			materiales[entrada.MaterialArchivo] = true
		}
	}
	vistas := map[string]bool{}
	for _, e := range meta.EntradasCapacidad {
		if (e.Audiencia != postgres.AudienciaGobiernoInscripcionProponer &&
			e.Audiencia != postgres.AudienciaGobiernoInscripcionAprobar) ||
			vistas[e.Audiencia] || e.MaterialArchivo == c.PoolGobierno || e.MaterialArchivo == c.PoolCatalogo ||
			materiales[e.MaterialArchivo] {
			return errConfiguracionPrivadaPerfiles
		}
		for _, ruta := range rutas {
			if ruta != "" && e.MaterialArchivo == ruta {
				return errConfiguracionPrivadaPerfiles
			}
		}
		vistas[e.Audiencia] = true
		materiales[e.MaterialArchivo] = true
	}
	if !vistas[postgres.AudienciaGobiernoInscripcionProponer] ||
		!vistas[postgres.AudienciaGobiernoInscripcionAprobar] {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

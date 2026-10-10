package main

import (
	"encoding/json"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

// Sólo rutas de material privado ya aprovisionado. Su presencia prepara el
// montaje ADMIN; no ejecuta mantenimiento AUT59, admite catálogo ni publica
// definiciones al arrancar.
type configuracionGobiernoRolesPrivada struct {
	PoolGobierno  string                                      `json:"pool_gobierno"`
	PoolCatalogo  string                                      `json:"pool_catalogo"`
	ConfianzaJSON json.RawMessage                             `json:"confianza"`
	Motivos       map[string]domain.ReferenciaEntradaCatalogo `json:"motivos"`
}

func cargarConfiguracionGobiernoRolesPrivada(ruta string, base configuracionPerfilesPrivada,
	u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN,
	lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada,
	efectos []efectoConfigurado) (configuracionGobiernoRolesPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionGobiernoRolesPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionGobiernoRolesPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil ||
		validarConfiguracionGobiernoRolesPrivada(c, base, u, runtime, lote, plan, efectos) != nil {
		return configuracionGobiernoRolesPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionGobiernoRolesPrivada(c configuracionGobiernoRolesPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado) error {
	return validarConfiguracionGobiernoPrivada(c, base, u, runtime, lote, plan, efectos,
		[2]string{pg.AudienciaGobiernoRolProponer, pg.AudienciaGobiernoRolAprobar}, nil, nil)
}

// La misma guarda verifica cada overlay Gov con audiencias y materiales
// disjuntos. Compartir un LOGIN o un material entre gobiernos falla cerrado.
func validarConfiguracionGobiernoPrivada(c configuracionGobiernoRolesPrivada,
	base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada,
	plan *configuracionPlanFirmaPrivada, efectos []efectoConfigurado,
	audiencias [2]string, otrasRutas []string, otrasConfianzas []json.RawMessage) error {
	if !rutaPrivadaPerfilesValida(c.PoolGobierno) || !rutaPrivadaPerfilesValida(c.PoolCatalogo) ||
		c.PoolGobierno == c.PoolCatalogo || contieneClavePrivadaInline(c.ConfianzaJSON) ||
		len(c.Motivos) != 2 ||
		!motivoGobiernoRolesValido(c.Motivos[audiencias[0]], base.CatalogoMotivosID) ||
		!motivoGobiernoRolesValido(c.Motivos[audiencias[1]], base.CatalogoMotivosID) {
		return errConfiguracionPrivadaPerfiles
	}
	for clave := range c.Motivos {
		if clave != audiencias[0] && clave != audiencias[1] {
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
	rutas = append(rutas, otrasRutas...)
	confianzas = append(confianzas, otrasConfianzas...)
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
		if (e.Audiencia != audiencias[0] && e.Audiencia != audiencias[1]) ||
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
	if !vistas[audiencias[0]] || !vistas[audiencias[1]] {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

func motivoGobiernoRolesValido(m domain.ReferenciaEntradaCatalogo, catalogo string) bool {
	return domain.ReferenciaMotivoAutorizacionV2Valida(m) && m.CatalogoID == catalogo
}

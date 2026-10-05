package main

import (
	"encoding/json"
	"regexp"

	"vec-diputacion-granada/internal/app/administracion"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
)

const modoLoteADMIN = "lote_v1"

// fuenteLotePrivada es el descriptor de una fuente propietaria de ámbito
// (organización o unidad) tal como lo dejó el arranque. AUT44 lo coteja con
// la fuente durable en la misma transacción del lote.
type fuenteLotePrivada struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type unidadLotePrivada struct {
	UnidadRef          string            `json:"unidad_ref"`
	FuenteOrganizacion fuenteLotePrivada `json:"fuente_organizacion"`
	FuenteUnidad       fuenteLotePrivada `json:"fuente_unidad"`
}

// configuracionLotePrivada abre el lote ordinario de perfiles y su preparación
// en el modo metadatos_v1. Tiene pool y LOGIN propios (grupo
// vec_admin_perfiles_lote_ejecutor) y una sola capacidad V3, la del lote. La
// organización, el proceso, el canal y los motivos de auditoría son los del
// overlay de usuarios: el lote no puede tener otra organización.
type configuracionLotePrivada struct {
	Modo          string                           `json:"modo"`
	PoolLote      string                           `json:"pool_lote"`
	ConfianzaJSON json.RawMessage                  `json:"confianza"`
	MotivoLote    domain.ReferenciaEntradaCatalogo `json:"motivo_lote"`
	Unidades      []unidadLotePrivada              `json:"unidades"`
	// MotivosCambio son los motivos que la pantalla ofrece al dar o quitar un
	// perfil, con la clave de texto que los nombra en los catálogos es/en.
	MotivosCambio []motivoCambioPrivado `json:"motivos_cambio"`
}

type motivoCambioPrivado struct {
	Motivo    domain.ReferenciaEntradaCatalogo `json:"motivo"`
	ClaveI18N string                           `json:"clave_i18n"`
}

func (c configuracionLotePrivada) motivosCambio() []api.MotivoLote {
	r := make([]api.MotivoLote, 0, len(c.MotivosCambio))
	for _, m := range c.MotivosCambio {
		r = append(r, api.MotivoLote{Motivo: api.Motivo{CatalogoID: m.Motivo.CatalogoID, CatalogoVersion: m.Motivo.CatalogoVersion,
			CatalogoHuellaSHA256: m.Motivo.CatalogoHuellaSHA256, EntradaClave: m.Motivo.EntradaClave}, ClaveI18N: m.ClaveI18N})
	}
	return r
}

var claveMotivoCambio = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

func cargarConfiguracionLotePrivada(ruta string, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) (configuracionLotePrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionLotePrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionLotePrivada
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionLotePrivada(c, base, u, runtime) != nil {
		return configuracionLotePrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionLotePrivada(c configuracionLotePrivada, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) error {
	// Con el lote, la preparación necesita su destino de auditoría propio.
	if _, ok := u.Destinos["preparar_lote_ordinario"]; !ok {
		return errConfiguracionPrivadaPerfiles
	}
	if c.Modo != modoLoteADMIN || !rutaPrivadaPerfilesValida(c.PoolLote) || contieneClavePrivadaInline(c.ConfianzaJSON) ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoLote) || c.MotivoLote.CatalogoID != base.CatalogoMotivosID {
		return errConfiguracionPrivadaPerfiles
	}
	// El pool del lote no comparte archivo con ningún otro pool ni secreto.
	for _, ruta := range []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos,
		base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, base.Pools.ActosADMIN,
		base.Pools.AuditoriaFrontera, base.Firmante.ClavePrivadaArchivo, base.Identidad.RutaConfiguracionHMAC,
		u.PoolLector, u.PoolIntentos, u.PoolSelector, u.PoolFronteraTecnica, runtime.PoolContexto, runtime.FuenteIdentificadoresArchivo} {
		if ruta == c.PoolLote {
			return errConfiguracionPrivadaPerfiles
		}
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil || len(meta.EntradasCapacidad) != 1 || meta.EntradasCapacidad[0].Audiencia != administracion.AudienciaLoteOrdinarioV3 ||
		meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 || meta.Raiz.ClaveID != base.Firmante.ClaveID ||
		meta.Raiz.Audiencia != base.Firmante.Audiencia || meta.EntradasCapacidad[0].MaterialArchivo == c.PoolLote {
		return errConfiguracionPrivadaPerfiles
	}
	// El material HMAC del lote es un archivo propio, distinto de los demás.
	materiales := map[string]bool{}
	for _, raw := range []json.RawMessage{base.ConfianzaJSON, u.ConfianzaJSON} {
		otros, err := decodificarMetadatosConfianzaPerfiles(raw)
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, e := range otros.EntradasCapacidad {
			materiales[e.MaterialArchivo] = true
		}
	}
	if materiales[meta.EntradasCapacidad[0].MaterialArchivo] {
		return errConfiguracionPrivadaPerfiles
	}
	if _, err := c.proveedorAmbitos(u.OrganizacionRef); err != nil {
		return errConfiguracionPrivadaPerfiles
	}
	if len(c.MotivosCambio) == 0 || len(c.MotivosCambio) > 16 {
		return errConfiguracionPrivadaPerfiles
	}
	motivos, claves := map[domain.ReferenciaEntradaCatalogo]bool{}, map[string]bool{}
	for _, m := range c.MotivosCambio {
		if m.Motivo.Validar() != nil || !claveMotivoCambio.MatchString(m.ClaveI18N) || motivos[m.Motivo] || claves[m.ClaveI18N] {
			return errConfiguracionPrivadaPerfiles
		}
		motivos[m.Motivo], claves[m.ClaveI18N] = true, true
	}
	return nil
}

// proveedorAmbitos traduce las unidades privadas al proveedor del adaptador,
// que valida forma, duplicados y que cada fuente sea de esa organización.
func (c configuracionLotePrivada) proveedorAmbitos(organizacion string) (pg.ProveedorAmbitosLote, error) {
	unidades := make([]pg.AmbitosFuenteLote, 0, len(c.Unidades))
	for _, x := range c.Unidades {
		unidades = append(unidades, pg.AmbitosFuenteLote{OrganizacionRef: organizacion, UnidadRef: x.UnidadRef,
			Descriptores: []pg.DimensionFuenteLote{
				{Dimension: "organizacion_ref", Valores: []string{organizacion}, Fuente: pg.FuenteDescriptorLote(x.FuenteOrganizacion)},
				{Dimension: "unidad_ref", Valores: []string{x.UnidadRef}, Fuente: pg.FuenteDescriptorLote(x.FuenteUnidad)}}})
	}
	return pg.NuevaFuenteAmbitosLotePrivada(pg.ConfiguracionProveedorAmbitosLote{OrganizacionRef: organizacion, Unidades: unidades})
}

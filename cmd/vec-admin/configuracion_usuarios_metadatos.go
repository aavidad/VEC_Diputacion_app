package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

const modoUsuariosMetadatos = "metadatos_v1"

type destinoUsuariosPrivado struct {
	Accion       string `json:"accion"`
	RecursoRef   string `json:"recurso_ref"`
	FinalidadRef string `json:"finalidad_ref"`
	TipoRecurso  string `json:"tipo_recurso"`
}

// Overlay privado y cerrado. El fichero legado conserva exactamente su
// formato y se usa sólo para los componentes comunes de identidad/arranque.
type configuracionUsuariosMetadatosPrivada struct {
	Modo                string                                      `json:"modo"`
	PoolLector          string                                      `json:"pool_lector"`
	PoolIntentos        string                                      `json:"pool_intentos"`
	PoolFronteraTecnica string                                      `json:"pool_frontera_tecnica"`
	PoolSelector        string                                      `json:"pool_selector"`
	ConfianzaJSON       json.RawMessage                             `json:"confianza"`
	OrganizacionRef     string                                      `json:"organizacion_ref"`
	UnidadRef           string                                      `json:"unidad_ref"`
	Proceso             string                                      `json:"proceso"`
	Canal               string                                      `json:"canal"`
	MotivosUsuarios     map[string]domain.ReferenciaEntradaCatalogo `json:"motivos_usuarios"`
	MotivoDenegado      domain.ReferenciaEntradaCatalogo            `json:"motivo_denegado"`
	MotivoError         domain.ReferenciaEntradaCatalogo            `json:"motivo_error"`
	PlazoAuditoriaMS    int                                         `json:"plazo_auditoria_ms"`
	Destinos            map[string]destinoUsuariosPrivado           `json:"destinos"`
}

var referenciaAmbitoUsuarios = regexp.MustCompile(`^[A-Za-z0-9_:-]{3,128}$`)
var procesoUsuarios = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
var recursoFijoUsuarios = regexp.MustCompile(`^administracion:[a-z0-9_:-]{1,185}$`)
var conjuntoUsuarios = regexp.MustCompile(`^conjunto_admin:[0-9a-f]{32}$`)

// clavesDestinoOpcionalesUsuarios sólo hacen falta con el lote montado
// (VEC_ADMIN_LOTE_CONFIG_FILE); un overlay anterior sigue siendo válido.
var clavesDestinoOpcionalesUsuarios = map[string]struct{}{"preparar_lote_ordinario": {}}

var clavesDestinoUsuarios = map[string]struct{}{
	"consultar": {}, "buscar_personas": {}, "consultar_persona": {}, "consultar_recibo": {},
	"escribir": {}, "aplicar_ordinario": {}, "proponer": {}, "cerrar_propuesta": {}, "aplicar_lote_ordinario": {},
}

func cargarConfiguracionUsuariosMetadatosPrivada(ruta string, base configuracionPerfilesPrivada) (configuracionUsuariosMetadatosPrivada, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionUsuariosMetadatosPrivada{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionUsuariosMetadatosPrivada
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionUsuariosMetadatosPrivada(c, base) != nil {
		return configuracionUsuariosMetadatosPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func referenciaConjuntoUsuarios(org, unidad string) string {
	b, _ := json.Marshal(struct {
		OrganizacionRef string `json:"organizacion_ref"`
		UnidadRef       string `json:"unidad_ref"`
	}{org, unidad})
	h := sha256.Sum256(append([]byte("vec.admin.conjunto-usuarios.v1\n"), b...))
	return "conjunto_admin:" + hex.EncodeToString(h[:16])
}

func validarConfiguracionUsuariosMetadatosPrivada(c configuracionUsuariosMetadatosPrivada, base configuracionPerfilesPrivada) error {
	if c.Modo != modoUsuariosMetadatos || !referenciaAmbitoUsuarios.MatchString(c.OrganizacionRef) || !referenciaAmbitoUsuarios.MatchString(c.UnidadRef) ||
		!procesoUsuarios.MatchString(c.Proceso) || c.Canal != "administracion_privilegiada" || c.PlazoAuditoriaMS <= 0 || c.PlazoAuditoriaMS > 2000 ||
		c.MotivoDenegado.Validar() != nil || c.MotivoError.Validar() != nil || c.MotivoDenegado.CatalogoID != base.CatalogoMotivosID || c.MotivoError.CatalogoID != base.CatalogoMotivosID ||
		len(c.MotivosUsuarios) != 2 || len(c.Destinos) != len(clavesDestinoUsuariosDe(c)) || contieneClavePrivadaInline(c.ConfianzaJSON) {
		return errConfiguracionPrivadaPerfiles
	}
	rutas := []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos, base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, base.Pools.AuditoriaFrontera, c.PoolLector, c.PoolIntentos, c.PoolSelector, c.PoolFronteraTecnica, base.Firmante.ClavePrivadaArchivo, base.Identidad.RutaConfiguracionHMAC}
	if base.Pools.ActosADMIN != "" {
		rutas = append(rutas, base.Pools.ActosADMIN)
	}
	vistas := map[string]bool{}
	for _, ruta := range rutas {
		if !rutaPrivadaPerfilesValida(ruta) || vistas[ruta] {
			return errConfiguracionPrivadaPerfiles
		}
		vistas[ruta] = true
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil || len(meta.EntradasCapacidad) != 2 || meta.Raiz.PublicaBase64 != base.Firmante.PublicaEsperadaBase64 || meta.Raiz.ClaveID != base.Firmante.ClaveID || meta.Raiz.Audiencia != base.Firmante.Audiencia {
		return errConfiguracionPrivadaPerfiles
	}
	baseMeta, err := decodificarMetadatosConfianzaPerfiles(base.ConfianzaJSON)
	if err != nil {
		return errConfiguracionPrivadaPerfiles
	}
	for _, entrada := range baseMeta.EntradasCapacidad {
		vistas[entrada.MaterialArchivo] = true
	}
	previstas := map[string]bool{administracion.AudienciaUsuariosListarV3: false, administracion.AudienciaUsuariosConsultarV3: false}
	for _, entrada := range meta.EntradasCapacidad {
		if _, ok := previstas[entrada.Audiencia]; !ok || previstas[entrada.Audiencia] || vistas[entrada.MaterialArchivo] {
			return errConfiguracionPrivadaPerfiles
		}
		previstas[entrada.Audiencia] = true
		vistas[entrada.MaterialArchivo] = true
	}
	for audiencia, vista := range previstas {
		motivo, ok := c.MotivosUsuarios[audiencia]
		if !vista || !ok || motivo.Validar() != nil || motivo.CatalogoID != base.CatalogoMotivosID {
			return errConfiguracionPrivadaPerfiles
		}
	}
	conjunto := referenciaConjuntoUsuarios(c.OrganizacionRef, c.UnidadRef)
	for clave := range clavesDestinoUsuariosDe(c) {
		d, ok := c.Destinos[clave]
		if !ok || !strings.HasPrefix(d.Accion, "administracion.") || !strings.HasPrefix(d.FinalidadRef, "gestion_") || strings.ContainsAny(d.Accion+d.FinalidadRef, "* \t\r\n") {
			return errConfiguracionPrivadaPerfiles
		}
		switch clave {
		case "buscar_personas":
			if d.TipoRecurso != "conjunto" || !conjuntoUsuarios.MatchString(d.RecursoRef) || d.RecursoRef != conjunto || d.Accion != "administracion.usuarios.listar" || d.FinalidadRef != "gestion_usuarios" {
				return errConfiguracionPrivadaPerfiles
			}
		case "consultar_persona":
			if d.TipoRecurso != "persona" || d.RecursoRef != "" || d.Accion != "administracion.usuarios.consultar" || d.FinalidadRef != "gestion_usuarios" {
				return errConfiguracionPrivadaPerfiles
			}
		default:
			if d.TipoRecurso != "fijo" || !recursoFijoUsuarios.MatchString(d.RecursoRef) {
				return errConfiguracionPrivadaPerfiles
			}
		}
	}
	return nil
}

// clavesDestinoUsuariosDe son las obligatorias más las opcionales presentes.
func clavesDestinoUsuariosDe(c configuracionUsuariosMetadatosPrivada) map[string]struct{} {
	claves := maps.Clone(clavesDestinoUsuarios)
	for clave := range clavesDestinoOpcionalesUsuarios {
		if _, ok := c.Destinos[clave]; ok {
			claves[clave] = struct{}{}
		}
	}
	return claves
}

func (c configuracionUsuariosMetadatosPrivada) destinosAuditoria() map[string]pg.DestinoFronteraNominal {
	r := make(map[string]pg.DestinoFronteraNominal, len(c.Destinos))
	for k, v := range c.Destinos {
		r[k] = pg.DestinoFronteraNominal{Accion: v.Accion, RecursoRef: v.RecursoRef, FinalidadRef: v.FinalidadRef, TipoRecurso: v.TipoRecurso}
	}
	return r
}

func (c configuracionUsuariosMetadatosPrivada) plazoAuditoria() time.Duration {
	return time.Duration(c.PlazoAuditoriaMS) * time.Millisecond
}

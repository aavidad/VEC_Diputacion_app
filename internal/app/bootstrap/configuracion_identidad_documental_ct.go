package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const envIdentidadDocumentalCTFile = "VEC_CT_IDENTIDAD_DOCUMENTAL_FILE"

var errIdentidadDocumentalCTNoDisponible = errors.New("contratación temporal: identidad documental no disponible")
var versionRolIdentidadDocumentalCT = regexp.MustCompile(`^rol:identidad_documental_ct_gestion_dev:v[1-9][0-9]{0,8}$`)

// Solo coordenadas de actos ya publicados y nombres de ficheros privados.
// El catálogo y el PDP vuelven a comprobar rol, asignación y ámbito.
type archivoIdentidadDocumentalCT struct {
	Esquema           string                         `json:"esquema"`
	PerfilActivoRef   string                         `json:"perfil_activo_ref"`
	VersionRolRef     string                         `json:"version_rol_ref"`
	ActoAsignacionRef string                         `json:"acto_asignacion_ref"`
	PrincipalID       string                         `json:"principal_id"`
	CuentaRef         string                         `json:"cuenta_ref"`
	CertificadoSHA256 string                         `json:"certificado_sha256"`
	UnidadRef         string                         `json:"unidad_ref"`
	Motivo            core.ReferenciaEntradaCatalogo `json:"motivo"`
	DSNFiles          map[string]string              `json:"dsn_files"`
}

var rolesPoolsIdentidadDocumentalCT = map[string]string{
	"bolsa":                "vec_bolsa_llamamientos_ejecutor",
	"fuente_autorizacion":  "vec_autorizacion_fuente",
	"motivos_autorizacion": "vec_autorizacion_motivos_evaluador",
}

func (c archivoIdentidadDocumentalCT) valida() bool {
	if c.Esquema != "vec.contratacion-temporal.identidad-documental-servidor.v1" ||
		!perfilActivoSeguridadComunValido(c.PerfilActivoRef) ||
		!versionRolIdentidadDocumentalCT.MatchString(c.VersionRolRef) ||
		c.ActoAsignacionRef == "" || c.PrincipalID == "" || c.CuentaRef == "" ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(c.CertificadoSHA256) ||
		c.UnidadRef == "" || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) ||
		len(c.DSNFiles) != len(rolesPoolsIdentidadDocumentalCT) {
		return false
	}
	for clave := range rolesPoolsIdentidadDocumentalCT {
		if c.DSNFiles[clave] == "" {
			return false
		}
	}
	return true
}

func leerConfiguracionIdentidadDocumentalCT(ruta string) (archivoIdentidadDocumentalCT, *os.Root, error) {
	var c archivoIdentidadDocumentalCT
	if !filepath.IsAbs(ruta) || dentroDeRepositorioGit(ruta) || validarArbolMaterialDesarrollo(filepath.Dir(ruta)) != nil {
		return c, nil, errIdentidadDocumentalCTNoDisponible
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return c, nil, errIdentidadDocumentalCTNoDisponible
	}
	b, err := leerArchivoIncorporacionV2(raiz, filepath.Base(ruta), 16<<10)
	if err != nil {
		raiz.Close()
		return c, nil, errIdentidadDocumentalCTNoDisponible
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || !c.valida() {
		raiz.Close()
		return archivoIdentidadDocumentalCT{}, nil, errIdentidadDocumentalCTNoDisponible
	}
	return c, raiz, nil
}

// Los DSN viven en ficheros separados. Cada LOGIN queda acreditado por el
// abridor existente; una configuración ausente deshabilita solo esta lectura.
func abrirPoolsIdentidadDocumentalCT(ctx context.Context, raiz *os.Root, c archivoIdentidadDocumentalCT) (map[string]*pgxpool.Pool, func(), error) {
	if ctx == nil || raiz == nil || !c.valida() {
		return nil, nil, errIdentidadDocumentalCTNoDisponible
	}
	pools := map[string]*pgxpool.Pool{}
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	usuarios := map[string]bool{}
	for clave, rol := range rolesPoolsIdentidadDocumentalCT {
		b, err := leerArchivoIncorporacionV2(raiz, c.DSNFiles[clave], 16<<10)
		if err != nil {
			cerrar()
			return nil, nil, errIdentidadDocumentalCTNoDisponible
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		config, err := pgxpool.ParseConfig(dsn)
		if err != nil || config.ConnConfig.User == "" || usuarios[config.ConnConfig.User] {
			cerrar()
			return nil, nil, errIdentidadDocumentalCTNoDisponible
		}
		usuarios[config.ConnConfig.User] = true
		pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-ct-identidad-documental-"+clave, rol)
		if err != nil {
			cerrar()
			return nil, nil, errIdentidadDocumentalCTNoDisponible
		}
		pools[clave] = pool
	}
	return pools, cerrar, nil
}

func concesionIdentidadDocumentalCTValida(c core.ConcesionRol) bool {
	return c.Accion == bp.AccionIdentidadDocumentoAceptacionCT && c.ModuloID == "bolsa" &&
		c.TipoRecurso == bp.TipoIdentidadDocumentoAceptacionCT &&
		len(c.Finalidades) == 1 && c.Finalidades[0] == bp.FinalidadIdentidadDocumentoAceptacionCT &&
		c.GarantiaMinima == core.AuthAssuranceHigh &&
		len(c.CamposPermitidos) == 4 && c.CamposPermitidos[0] == "apellidos" &&
		c.CamposPermitidos[1] == "documento_enmascarado" && c.CamposPermitidos[2] == "nombre" &&
		c.CamposPermitidos[3] == "version_fuente" && len(c.Obligaciones) == 0
}

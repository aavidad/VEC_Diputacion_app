package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/config"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const archivoGobiernoReglasBaremoHTTPV3 = "bolsa/gobierno-reglas-baremo-v3.json"

// Sólo configuración privada de composición. Los DSN permanecen en archivos
// privados separados; este documento no contiene permisos enviados por HTTP.
type configuracionGobiernoReglasBaremoHTTPV3 struct {
	MotivoIntentoDenegado vd.ReferenciaEntradaCatalogo `json:"motivo_intento_denegado"`
	MotivoIntentoError    vd.ReferenciaEntradaCatalogo `json:"motivo_intento_error"`
	Esquema               string                       `json:"esquema"`
	ConvocatoriaRef       string                       `json:"convocatoria_ref"`
	ExpedienteRef         string                       `json:"expediente_ref"`
	CatalogoMotivosID     string                       `json:"catalogo_motivos_id"`
	DSNFiles              map[string]string            `json:"dsn_files"`
	ProvisionarPerfil     bool                         `json:"provisionar_perfil"`
	AprobacionRef         string                       `json:"aprobacion_ref"`
	PreimagenPerfilSHA256 string                       `json:"preimagen_perfil_sha256"`
}

func leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg config.Config) (*configuracionGobiernoReglasBaremoHTTPV3, error) {
	if cfg.DevelopmentMaterialDir == "" {
		return nil, nil
	}
	if !filepath.IsAbs(cfg.DevelopmentMaterialDir) || dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	defer raiz.Close()
	if _, err := raiz.Lstat(archivoGobiernoReglasBaremoHTTPV3); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	b, err := leerArchivoIncorporacionV2(raiz, archivoGobiernoReglasBaremoHTTPV3, 16<<10)
	if err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var c configuracionGobiernoReglasBaremoHTTPV3
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.validar() != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	return &c, nil
}

func (c *configuracionGobiernoReglasBaremoHTTPV3) validar() error {
	if c == nil || c.MotivoIntentoDenegado.Validar() != nil || c.MotivoIntentoError.Validar() != nil || c.Esquema != "vec.bolsa.gobierno-reglas-baremo.configuracion.v3" ||
		c.ConvocatoriaRef == "" || c.ExpedienteRef == "" || c.CatalogoMotivosID == "" ||
		strings.TrimSpace(c.CatalogoMotivosID) != c.CatalogoMotivosID || strings.ContainsAny(c.CatalogoMotivosID, "*\r\n\t") ||
		len(c.CatalogoMotivosID) > 128 || len(c.DSNFiles) != 3 ||
		(!c.ProvisionarPerfil && (c.AprobacionRef != "" || c.PreimagenPerfilSHA256 != "")) ||
		((c.AprobacionRef == "") != (c.PreimagenPerfilSHA256 == "")) ||
		(c.PreimagenPerfilSHA256 != "" && !shaHexGobiernoV3(c.PreimagenPerfilSHA256)) {
		return app.ErrGobiernoV3NoDisponible
	}
	vistas := map[string]bool{}
	for _, nombre := range []string{"runtime", "fuente_autorizacion", "motivos_autorizacion"} {
		ruta := c.DSNFiles[nombre]
		if !filepath.IsLocal(ruta) || vistas[ruta] {
			return app.ErrGobiernoV3NoDisponible
		}
		vistas[ruta] = true
	}
	return nil
}

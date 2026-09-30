package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"vec-diputacion-granada/config"
)

var errAvisosExternos = errors.New("bootstrap: avisos externos no disponibles")

const esquemaAvisosExternos = "vec.avisos-externos.configuracion.v1"

var patronRefAvisosExternos = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{7,191}$`)
var patronIdiomaAvisosExternos = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)

type configuracionAvisosExternos struct {
	Esquema      string `json:"esquema"`
	Version      uint64 `json:"version"`
	ProductorRef string `json:"productor_ref"`
	Lote         int    `json:"lote"`
	Intervalo    string `json:"intervalo"`
	Idioma       string `json:"idioma"`
	URLPersonal  string `json:"url_personal"`
}

func leerConfiguracionAvisosExternos(cfg config.Config, relativa string) (configuracionAvisosExternos, error) {
	var c configuracionAvisosExternos
	f, err := os.Open(filepath.Join(cfg.DevelopmentMaterialDir, filepath.FromSlash(relativa)))
	if err != nil {
		return c, errAvisosExternos
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 16<<10+1))
	if err != nil || len(raw) > 16<<10 {
		return c, errAvisosExternos
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Esquema != esquemaAvisosExternos || c.Version == 0 || !patronRefAvisosExternos.MatchString(c.ProductorRef) {
		return c, errAvisosExternos
	}
	return c, nil
}
func (c configuracionAvisosExternos) intervaloValido() (time.Duration, error) {
	d, err := time.ParseDuration(c.Intervalo)
	if err != nil || d < time.Second || d > time.Hour || c.Lote < 1 || c.Lote > 100 || !patronIdiomaAvisosExternos.MatchString(c.Idioma) {
		return 0, errAvisosExternos
	}
	return d, nil
}

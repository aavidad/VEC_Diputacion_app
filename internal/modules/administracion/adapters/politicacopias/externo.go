//go:build linux

package politicacopias

import (
	"path/filepath"
	"strings"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
)

type ConfigExterna struct {
	Directorio        string   `json:"directorio"`
	RaicesRestauradas []string `json:"raices_restauradas"`
}

// AbrirExterno checks canonical paths before creating control files. The roots
// must be the complete trusted rollback inventory, not paths selected by HTTP.
// Mount topology and future root changes remain infrastructure obligations.
func AbrirExterno(c ConfigExterna) (*Archivo, error) {
	if !filepath.IsAbs(c.Directorio) || len(c.RaicesRestauradas) == 0 || len(c.RaicesRestauradas) > 64 {
		return nil, d.ErrEntrada
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(filepath.Clean(c.Directorio)))
	if e != nil {
		return nil, d.ErrDependencia
	}
	candidate := filepath.Join(parent, filepath.Base(filepath.Clean(c.Directorio)))
	for _, path := range c.RaicesRestauradas {
		if !filepath.IsAbs(path) {
			return nil, d.ErrEntrada
		}
		root, e := filepath.EvalSymlinks(path)
		if e != nil {
			return nil, d.ErrDependencia
		}
		rel, e := filepath.Rel(root, candidate)
		if e != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return nil, d.ErrEntrada
		}
	}
	return Abrir(candidate)
}

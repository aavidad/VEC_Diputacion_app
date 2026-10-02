package ensayofisicopg

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type cuentaTar struct {
	bytes    int64
	entradas int
}

// Dos pasadas sobre la copia SHA verificada: validar todo antes de extraer.
// No se admiten enlaces, rutas alternas, dispositivos ni entradas duplicadas.
func revisarTar(ctx context.Context, ruta string, c Configuracion, cuenta *cuentaTar) error {
	f, err := os.Open(ruta) // #nosec G304 -- copia TAR SHA verificada en scratch propio; no se reabre entrada externa.
	if err != nil {
		return errEntrada
	}
	defer f.Close()
	t := tar.NewReader(lectorContexto{ctx, f})
	vistos := map[string]byte{}
	padres := map[string]bool{}
	for {
		if ctx.Err() != nil {
			return errEntrada
		}
		h, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errEntrada
		}
		n := strings.TrimSuffix(h.Name, "/")
		if len(n) > 4096 || n == "" || n != path.Clean(n) || strings.ContainsAny(n, "\\\x00\r\n") || (n != "contenido" && !strings.HasPrefix(n, "contenido/")) || (h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir) || h.Linkname != "" || h.Mode&07000 != 0 || h.Size < 0 || (h.Typeflag == tar.TypeDir && h.Size != 0) {
			return errEntrada
		}
		if _, ok := vistos[n]; ok {
			return errEntrada
		}
		for p := path.Dir(n); p != "."; p = path.Dir(p) {
			if v, ok := vistos[p]; ok && v != tar.TypeDir {
				return errEntrada
			}
		}
		if h.Typeflag == tar.TypeReg && padres[n] {
			return errEntrada
		}
		for p := path.Dir(n); p != "."; p = path.Dir(p) {
			padres[p] = true
		}
		vistos[n] = h.Typeflag
		cuenta.entradas++
		if cuenta.entradas > c.LimiteEntradas || h.Size > c.LimiteExtraidoBytes-cuenta.bytes {
			return errEntrada
		}
		cuenta.bytes += h.Size
		if _, err = io.CopyN(io.Discard, t, h.Size); err != nil {
			return errEntrada
		}
	}
	if _, ok := vistos["contenido"]; !ok {
		return errEntrada
	}
	return nil
}

// destino y todos sus padres son nuevos, privados y no se exponen al runtime
// hasta acabar la extracción. Los modos se reducen a acceso del usuario local.
func extraerTar(ctx context.Context, ruta, destino string) error {
	f, err := os.Open(ruta) // #nosec G304 -- copia TAR SHA verificada en scratch propio; no se reabre entrada externa.
	if err != nil {
		return errEntrada
	}
	defer f.Close()
	root, err := os.OpenRoot(destino)
	if err != nil {
		return errEntrada
	}
	defer root.Close()
	t := tar.NewReader(lectorContexto{ctx, f})
	for {
		h, err := t.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errEntrada
		}
		n := filepath.FromSlash(strings.TrimSuffix(h.Name, "/"))
		if h.Typeflag == tar.TypeDir {
			if root.MkdirAll(n, 0700) != nil {
				return errEntrada
			}
			continue
		}
		if root.MkdirAll(filepath.Dir(n), 0700) != nil {
			return errEntrada
		}
		g, err := root.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600|os.FileMode(h.Mode&0100))
		if err != nil {
			return errEntrada
		}
		cantidad, escribir := io.CopyN(g, t, h.Size)
		cerrar := g.Close()
		if escribir != nil || cerrar != nil || cantidad != h.Size {
			return errEntrada
		}
	}
}

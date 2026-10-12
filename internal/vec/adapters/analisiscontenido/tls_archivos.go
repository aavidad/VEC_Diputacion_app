package analisiscontenido

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// leerArchivoTLS acepta solo un fichero regular absoluto, sin enlace final.
// La clave privada debe ser accesible únicamente al propietario.
func leerArchivoTLS(ruta string, clavePrivada bool) ([]byte, error) {
	if !filepath.IsAbs(ruta) {
		return nil, ErrConfiguracion
	}
	// #nosec G304 -- ruta de configuración privada, nunca de HTTP; O_NOFOLLOW, regular y límite estricto.
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrConfiguracion
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 1<<20 ||
		(clavePrivada && info.Mode().Perm()&0077 != 0) {
		return nil, ErrConfiguracion
	}
	contenido, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(contenido) < 1 || len(contenido) > 1<<20 {
		return nil, ErrConfiguracion
	}
	return contenido, nil
}

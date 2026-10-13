package bootstrap

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"time"
	"vec-diputacion-granada/internal/app/composicion/firmasct"
)

// El lector seguro ya usado por el material conserva además el modo exacto
// 0600 del registro personal. No importa la composición interna ni PKCS11.
func comprobarCRLFirmaVecCT(ruta string, hoja, autoridad *x509.Certificate, ahora time.Time) error {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return firmasct.ErrCRLFirmaNoDisponible
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return firmasct.ErrCRLFirmaNoDisponible
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil {
		return err
	}
	defer clear(contenido)
	actual, err := os.Lstat(ruta)
	if err != nil || !os.SameFile(info, actual) || actual.Mode().Perm() != 0600 {
		return firmasct.ErrCRLFirmaNoDisponible
	}
	lista, err := firmasct.LeerCRLFirmaR5(contenido)
	if err != nil {
		return err
	}
	return firmasct.ComprobarCRLFirmaR5(lista, hoja, autoridad, ahora)
}

package ensayofisicopg

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CrearRaizTemporal acepta una base de plataforma confiable, nunca una ruta web.
// Por defecto evita tmpfs compartido y crea una raíz exclusiva bajo /var/tmp.
// Una base configurada ha de ser privada, propia y sin enlaces; no se monta esa
// base, sino sólo la nueva raíz vacía que genera el ensayo.
func CrearRaizTemporal(base, prefijo string) (string, error) {
	if base == "" {
		base = "/var/tmp"
	}
	if prefijo != "vec-cs06f-" && prefijo != "vec-cs06l-" {
		return "", errEntrada
	}
	if base != filepath.Clean(base) || !filepath.IsAbs(base) || (base != "/var/tmp" && filepath.Dir(base) != "/var/tmp") {
		return "", errEntrada
	}
	i, err := os.Lstat(base)
	if err != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", errEntrada
	}
	if base != "/var/tmp" {
		stat, ok := i.Sys().(*syscall.Stat_t)
		if !ok || int(stat.Uid) != os.Getuid() || i.Mode().Perm() != 0700 {
			return "", errEntrada
		}
	}
	return os.MkdirTemp(base, prefijo)
}
func raizTemporalAdmitida(raiz string) bool {
	return filepath.Clean(raiz) == raiz && strings.HasPrefix(raiz, "/var/tmp/") && strings.HasPrefix(filepath.Base(raiz), "vec-cs06")
}

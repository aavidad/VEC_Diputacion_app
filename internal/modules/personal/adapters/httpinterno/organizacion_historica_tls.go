package httpinterno

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func cargarTLSOrganizacionHistorica(c ConfiguracionOrganizacionHistorica) (*tls.Config, error) {
	if !filepath.IsAbs(c.Directorio) || filepath.Clean(c.Directorio) != c.Directorio || dentroGitOrganizacionHistorica(c.Directorio) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	p, err := filepath.EvalSymlinks(c.Directorio)
	if err != nil || p != c.Directorio {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	info, err := os.Lstat(c.Directorio)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !propietarioTLSOH(info) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	raiz, err := os.OpenRoot(c.Directorio)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	defer raiz.Close()
	abierto, err := raiz.Stat(".")
	if err != nil || !os.SameFile(info, abierto) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	ca, err := leerTLSOrganizacionHistorica(raiz, c.AutoridadCA, false)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	cert, err := leerTLSOrganizacionHistorica(raiz, c.CertificadoCliente, false)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	key, err := leerTLSOrganizacionHistorica(raiz, c.ClaveCliente, true)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	defer clear(key)
	par, err := tls.X509KeyPair(cert, key)
	roots := x509.NewCertPool()
	if err != nil || !roots.AppendCertsFromPEM(ca) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{par}}, nil
}

func leerTLSOrganizacionHistorica(r *os.Root, nombre string, privado bool) ([]byte, error) {
	if !filepath.IsLocal(nombre) || nombre == "." || filepath.Clean(nombre) != nombre {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	f, err := r.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() || !propietarioTLSOH(s) || s.Size() < 1 || s.Size() > 1<<20 || s.Mode().Perm()&0022 != 0 || privado && s.Mode().Perm() != 0600 {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		clear(b)
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return b, nil
}

func dentroGitOrganizacionHistorica(p string) bool {
	for {
		_, err := os.Lstat(filepath.Join(p, ".git"))
		if err == nil || !os.IsNotExist(err) {
			return true
		}
		padre := filepath.Dir(p)
		if padre == p {
			return false
		}
		p = padre
	}
}

func propietarioTLSOH(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && (int64(s.Uid) == int64(os.Geteuid()) || s.Uid == 0)
}

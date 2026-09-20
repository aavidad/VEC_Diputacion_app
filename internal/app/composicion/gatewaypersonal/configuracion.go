package gatewaypersonal

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrConfiguracion = errors.New("gateway personal: configuracion invalida")

type Configuracion struct {
	Direccion, Origen, CertificadoTLS, ClaveTLS, CAClientes, CRLClientes, MapaCuentas, RaizWeb, DSN string
	ClaveHMAC                                                                                       []byte
}

func CargarConfiguracion() (Configuracion, error) {
	c := Configuracion{Direccion: os.Getenv("VEC_GATEWAY_PERSONAL_LISTEN"), Origen: os.Getenv("VEC_GATEWAY_PERSONAL_ORIGIN"), CertificadoTLS: os.Getenv("VEC_GATEWAY_PERSONAL_TLS_CERT"), ClaveTLS: os.Getenv("VEC_GATEWAY_PERSONAL_TLS_KEY"), CAClientes: os.Getenv("VEC_GATEWAY_PERSONAL_CLIENT_CA"), CRLClientes: os.Getenv("VEC_GATEWAY_PERSONAL_CLIENT_CRL"), MapaCuentas: os.Getenv("VEC_GATEWAY_PERSONAL_CERT_MAP"), RaizWeb: os.Getenv("VEC_AUTH_GATEWAY_WEB_ROOT"), DSN: os.Getenv("VEC_GATEWAY_PERSONAL_DSN")}
	clave := os.Getenv("VEC_GATEWAY_PERSONAL_SESSION_HMAC_FILE")
	if c.Direccion == "" || c.Origen == "" || c.CertificadoTLS == "" || c.ClaveTLS == "" || c.CAClientes == "" || c.CRLClientes == "" || c.MapaCuentas == "" || c.RaizWeb == "" || c.DSN == "" {
		return Configuracion{}, ErrConfiguracion
	}
	if !origenSeguro(c.Origen) || !directorioSeguro(c.RaizWeb) || !ficheroSeguro(c.ClaveTLS, true) || !ficheroSeguro(c.MapaCuentas, true) || !ficheroSeguro(clave, true) || !ficheroSeguro(c.CertificadoTLS, false) || !ficheroSeguro(c.CAClientes, false) || !ficheroSeguro(c.CRLClientes, false) {
		return Configuracion{}, ErrConfiguracion
	}
	material, err := os.ReadFile(clave)
	if err != nil || len(material) < 32 || len(material) > 64 {
		return Configuracion{}, ErrConfiguracion
	}
	c.ClaveHMAC = material
	c.RaizWeb, _ = filepath.EvalSymlinks(c.RaizWeb)
	return c, nil
}

func origenSeguro(s string) bool {
	u, e := url.Parse(s)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}
func directorioSeguro(s string) bool {
	if !filepath.IsAbs(s) {
		return false
	}
	i, e := os.Lstat(s)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return false
	}
	v, e := filepath.EvalSymlinks(s)
	return e == nil && v == filepath.Clean(s)
}
func ficheroSeguro(s string, privado bool) bool {
	if !filepath.IsAbs(s) {
		return false
	}
	i, e := os.Lstat(s)
	if e != nil || !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return !privado || i.Mode().Perm()&0o077 == 0
}

func (c Configuracion) TLS() (*tls.Config, *x509.RevocationList, map[string]string, error) {
	cert, err := tls.LoadX509KeyPair(c.CertificadoTLS, c.ClaveTLS)
	if err != nil {
		return nil, nil, nil, ErrConfiguracion
	}
	caPEM, err := os.ReadFile(c.CAClientes)
	if err != nil {
		return nil, nil, nil, ErrConfiguracion
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		return nil, nil, nil, ErrConfiguracion
	}
	crlPEM, err := os.ReadFile(c.CRLClientes)
	if err != nil {
		return nil, nil, nil, ErrConfiguracion
	}
	if bloque, _ := pem.Decode(crlPEM); bloque != nil {
		crlPEM = bloque.Bytes
	}
	crl, err := x509.ParseRevocationList(crlPEM)
	if err != nil {
		return nil, nil, nil, ErrConfiguracion
	}
	ahora := time.Now()
	if ahora.Before(crl.ThisUpdate) || !ahora.Before(crl.NextUpdate) {
		return nil, nil, nil, ErrConfiguracion
	}
	datos, err := os.ReadFile(c.MapaCuentas)
	if err != nil {
		return nil, nil, nil, ErrConfiguracion
	}
	var mapa map[string]string
	if json.Unmarshal(datos, &mapa) != nil || len(mapa) == 0 {
		return nil, nil, nil, ErrConfiguracion
	}
	limpio := make(map[string]string, len(mapa))
	for h, cuenta := range mapa {
		h = strings.ToLower(strings.TrimSpace(h))
		if len(h) != 64 || strings.TrimSpace(cuenta) == "" {
			return nil, nil, nil, ErrConfiguracion
		}
		if _, err := hex.DecodeString(h); err != nil {
			return nil, nil, nil, ErrConfiguracion
		}
		limpio[h] = strings.TrimSpace(cuenta)
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: raices, Certificates: []tls.Certificate{cert}}, crl, limpio, nil
}

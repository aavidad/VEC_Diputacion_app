package administracion

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrConfiguracion = errors.New("administracion: configuracion no valida")

type Configuracion struct {
	Entorno             string
	Escucha             string
	Host                string
	Audiencia           string
	EmisorIdentidad     string
	CertificadoServidor string
	ClaveServidor       string
	CAAdministracion    string
	CRLAdministracion   string
	RedesPermitidas     []string
	RetiradaEn          time.Time
}

// NuevoServidor solo monta la prueba de vida. Ninguna ruta de negocio queda
// publicada antes de disponer de rol y lectura ADMIN en V3.
func NuevoServidor(cfg Configuracion) (*http.Server, error) {
	if cfg.Entorno != "desarrollo" && cfg.Entorno != "cidonia" {
		// Produccion exige ademas Kerberos y concesion V3; todavia no hay
		// compositor ADMIN que pueda acreditarlos.
		return nil, ErrConfiguracion
	}
	if cfg.RetiradaEn.IsZero() || cfg.RetiradaEn.Location() != time.UTC ||
		!time.Now().Before(cfg.RetiradaEn) ||
		cfg.CertificadoServidor == "" || cfg.ClaveServidor == "" || cfg.CAAdministracion == "" ||
		cfg.CRLAdministracion == "" || cfg.Host == "" {
		return nil, ErrConfiguracion
	}
	superficie := httpseguridad.ConfiguracionSuperficie{
		Superficie:                          httpseguridad.SuperficieAdministracionPrivilegiada,
		ZonaRed:                             httpseguridad.ZonaRedAdministracion,
		DireccionEscucha:                    cfg.Escucha,
		Audiencia:                           cfg.Audiencia,
		EmisorIdentidad:                     cfg.EmisorIdentidad,
		RedesPermitidas:                     cfg.RedesPermitidas,
		DuracionMaximaAsercion:              time.Minute,
		EdadMaximaAutenticacion:             5 * time.Minute,
		MetodosAdmitidos:                    []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		FactoresRequeridos:                  []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados:           1,
		MinimoGruposCriptograficosDistintos: 1,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
		RequiereCuentaPrivilegiada:          true,
		PoliticaAdministracion:              httpseguridad.PoliticaAdministracionCertificadoTemporal,
		RetiradaPoliticaAdministracionEn:    cfg.RetiradaEn,
		CertificadoClienteDirecto:           true,
	}
	if err := superficie.Validar(); err != nil {
		return nil, ErrConfiguracion
	}
	red, err := httpseguridad.NuevaPoliticaRed(superficie)
	if err != nil {
		return nil, ErrConfiguracion
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertificadoServidor, cfg.ClaveServidor)
	if err != nil {
		return nil, ErrConfiguracion
	}
	ca, err := cargarCA(cfg.CAAdministracion)
	if err != nil {
		return nil, ErrConfiguracion
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	verificar := func(r *http.Request) bool {
		if !time.Now().Before(cfg.RetiradaEn) || r.Host != cfg.Host ||
			r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
			return false
		}
		direccion, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return false
		}
		ip, err := netip.ParseAddr(direccion)
		if err != nil || red.Autorizar(ip) != nil {
			return false
		}
		for _, cadena := range r.TLS.VerifiedChains {
			if len(cadena) > 1 && cadena[len(cadena)-1].Equal(ca) &&
				!cadena[0].IsCA && usoCliente(cadena[0]) &&
				certificadoVigente(cadena[0], ca, cfg.CRLAdministracion) {
				return true
			}
		}
		return false
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !verificar(r) {
			http.Error(w, "", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/livez" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})
	return &http.Server{
		Addr:              cfg.Escucha,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    raices,
		},
	}, nil
}

func certificadoVigente(hoja, ca *x509.Certificate, rutaCRL string) bool {
	datos, err := leerMaterial(rutaCRL)
	if err != nil {
		return false
	}
	bloque, resto := pem.Decode(datos)
	if bloque == nil || bloque.Type != "X509 CRL" || strings.TrimSpace(string(resto)) != "" {
		return false
	}
	crl, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil || crl.CheckSignatureFrom(ca) != nil ||
		time.Now().Before(crl.ThisUpdate) || !time.Now().Before(crl.NextUpdate) {
		return false
	}
	for _, revocado := range crl.RevokedCertificateEntries {
		if hoja.SerialNumber.Cmp(revocado.SerialNumber) == 0 {
			return false
		}
	}
	return true
}

func usoCliente(cert *x509.Certificate) bool {
	for _, uso := range cert.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}

func cargarCA(ruta string) (*x509.Certificate, error) {
	datos, err := leerMaterial(ruta)
	if err != nil {
		return nil, ErrConfiguracion
	}
	bloque, resto := pem.Decode(datos)
	if bloque == nil || bloque.Type != "CERTIFICATE" || strings.TrimSpace(string(resto)) != "" {
		return nil, ErrConfiguracion
	}
	ca, err := x509.ParseCertificate(bloque.Bytes)
	if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, ErrConfiguracion
	}
	return ca, nil
}

func leerMaterial(ruta string) ([]byte, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, ErrConfiguracion
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, ErrConfiguracion
	}
	defer raiz.Close()
	archivo, err := raiz.Open(filepath.Base(ruta))
	if err != nil {
		return nil, ErrConfiguracion
	}
	defer archivo.Close()
	estado, err := archivo.Stat()
	if err != nil || !estado.Mode().IsRegular() || estado.Size() > 1<<20 {
		return nil, ErrConfiguracion
	}
	datos, err := io.ReadAll(io.LimitReader(archivo, 1<<20+1))
	if err != nil || len(datos) > 1<<20 {
		return nil, ErrConfiguracion
	}
	return datos, nil
}

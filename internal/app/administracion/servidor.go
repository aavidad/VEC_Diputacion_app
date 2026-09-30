package administracion

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
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

var (
	ErrConfiguracion       = errors.New("administracion: configuracion no valida")
	errAccesoDenegado      = errors.New("administracion: acceso denegado")
	errParRemotoInvalido   = errors.New("administracion: par remoto invalido")
	errCadenaNoAdmitida    = errors.New("administracion: cadena cliente no admitida")
	errCRLNoDisponible     = errors.New("administracion: CRL no disponible")
	errCRLInvalida         = errors.New("administracion: CRL invalida")
	errCertificadoRevocado = errors.New("administracion: certificado revocado")
)

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
		return nil, fmt.Errorf("%w: %w", ErrConfiguracion, err)
	}
	red, err := httpseguridad.NuevaPoliticaRed(superficie)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfiguracion, err)
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertificadoServidor, cfg.ClaveServidor)
	if err != nil {
		return nil, fmt.Errorf("%w: certificado servidor: %w", ErrConfiguracion, err)
	}
	ca, err := cargarCA(cfg.CAAdministracion)
	if err != nil {
		return nil, fmt.Errorf("%w: CA ADMIN: %w", ErrConfiguracion, err)
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	verificar := func(r *http.Request) error {
		ahora := time.Now()
		if !ahora.Before(cfg.RetiradaEn) || r.Host != cfg.Host ||
			r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
			return errAccesoDenegado
		}
		direccion, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return fmt.Errorf("%w: %w", errParRemotoInvalido, err)
		}
		ip, err := netip.ParseAddr(direccion)
		if err != nil {
			return fmt.Errorf("%w: %w", errParRemotoInvalido, err)
		}
		if err := red.Autorizar(ip); err != nil {
			return fmt.Errorf("%w: %w", errAccesoDenegado, err)
		}
		for _, cadena := range r.TLS.VerifiedChains {
			if cadenaDirectaVigente(cadena, ca, ahora) {
				return comprobarCertificadoVigente(cadena[0], ca, cfg.CRLAdministracion, ahora)
			}
		}
		return errCadenaNoAdmitida
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if err := verificar(r); err != nil {
			switch {
			case errors.Is(err, errCRLNoDisponible):
				log.Print(errCRLNoDisponible)
			case errors.Is(err, errCRLInvalida):
				log.Print(errCRLInvalida)
			case errors.Is(err, errParRemotoInvalido):
				log.Print(errParRemotoInvalido)
			}
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

// Solo se admite una hoja emitida por la CA ADMIN. Si se aceptasen CA
// intermedias habria que comprobar la revocacion de cada eslabon.
func cadenaDirectaVigente(cadena []*x509.Certificate, ca *x509.Certificate, ahora time.Time) bool {
	if len(cadena) != 2 || cadena[0] == nil || cadena[1] == nil || ca == nil ||
		!cadena[1].Equal(ca) || cadena[0].IsCA || !usoCliente(cadena[0]) ||
		cadena[0].CheckSignatureFrom(ca) != nil {
		return false
	}
	for _, certificado := range cadena {
		if ahora.Before(certificado.NotBefore) || !ahora.Before(certificado.NotAfter) {
			return false
		}
	}
	return true
}

func comprobarCertificadoVigente(hoja, ca *x509.Certificate, rutaCRL string, ahora time.Time) error {
	datos, err := leerMaterial(rutaCRL)
	if err != nil {
		return fmt.Errorf("%w: %w", errCRLNoDisponible, err)
	}
	bloque, resto := pem.Decode(datos)
	if bloque == nil || bloque.Type != "X509 CRL" || strings.TrimSpace(string(resto)) != "" {
		return errCRLInvalida
	}
	crl, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil {
		return fmt.Errorf("%w: %w", errCRLInvalida, err)
	}
	if err := crl.CheckSignatureFrom(ca); err != nil {
		return fmt.Errorf("%w: firma: %w", errCRLInvalida, err)
	}
	if ahora.Before(crl.ThisUpdate) || !ahora.Before(crl.NextUpdate) {
		return errCRLInvalida
	}
	for _, revocado := range crl.RevokedCertificateEntries {
		if hoja.SerialNumber.Cmp(revocado.SerialNumber) == 0 {
			return errCertificadoRevocado
		}
	}
	return nil
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

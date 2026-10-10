package administracion

import (
	"context"
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

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
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

// falloConfiguracion conserva la causa para errors.Is/As sin exponer rutas de
// certificados o claves al registro de arranque.
type falloConfiguracion struct {
	clase string
	causa error
}

func (f falloConfiguracion) Error() string { return ErrConfiguracion.Error() + ": " + f.clase }
func (f falloConfiguracion) Unwrap() []error {
	if f.causa == nil {
		return []error{ErrConfiguracion}
	}
	return []error{ErrConfiguracion, f.causa}
}

// ClaseFallo devuelve la clase de un fallo de configuración del arranque ADMIN,
// o "" si el error no lleva una de la lista cerrada. Nunca devuelve la causa.
// Lista cerrada: dependencias, dependencias_runtime, registro_sesiones,
// revalidador, cuentas_is16, contexto_ca36, resolver_sesion, contexto_conexion,
// montaje_lecturas (ComponerServidorPerfiles); entorno, dependencias, host, ca,
// red, resolver_sesion (NuevoResolverSesionPerfiles); entorno, retirada,
// rutas_tls, host, superficie, red, tls, ca (nuevoServidor).
func ClaseFallo(err error) string {
	var f falloConfiguracion
	if !errors.As(err, &f) {
		return ""
	}
	switch f.clase {
	case "dependencias", "dependencias_runtime", "registro_sesiones", "revalidador", "cuentas_is16",
		"contexto_ca36", "resolver_sesion", "contexto_conexion", "montaje_lecturas", "entorno",
		"retirada", "rutas_tls", "host", "superficie", "red", "tls", "ca":
		return f.clase
	}
	return ""
}

// conClase conserva la clase que ya traiga err y, si no la trae, le pone clase.
func conClase(clase string, err error) error {
	if ClaseFallo(err) != "" {
		return err
	}
	return falloConfiguracion{clase: clase, causa: err}
}

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
	return nuevoServidor(cfg, nil)
}

// nuevoServidor conserva la misma frontera para todos los handlers nominales.
// El montaje no puede elegir rutas por configuración ni por petición.
func nuevoServidor(cfg Configuracion, perfiles *handlerPerfilesADMIN) (*http.Server, error) {
	if cfg.Entorno != "desarrollo" && cfg.Entorno != "cidonia" {
		// Produccion exige ademas Kerberos y concesion V3; todavia no hay
		// compositor ADMIN que pueda acreditarlos.
		return nil, falloConfiguracion{clase: "entorno"}
	}
	if cfg.RetiradaEn.IsZero() || cfg.RetiradaEn.Location() != time.UTC || !time.Now().Before(cfg.RetiradaEn) {
		return nil, falloConfiguracion{clase: "retirada"}
	}
	if cfg.CertificadoServidor == "" || cfg.ClaveServidor == "" || cfg.CAAdministracion == "" || cfg.CRLAdministracion == "" {
		return nil, falloConfiguracion{clase: "rutas_tls"}
	}
	host, hostValido := analizarHostAdmin(cfg.Host)
	if !hostValido {
		return nil, falloConfiguracion{clase: "host"}
	}
	superficie := httpseguridad.ConfiguracionSuperficie{
		Superficie:                          httpseguridad.SuperficieAdministracionPrivilegiada,
		ZonaRed:                             httpseguridad.ZonaRedAdministracion,
		DireccionEscucha:                    cfg.Escucha,
		Audiencia:                           cfg.Audiencia,
		EmisorIdentidad:                     cfg.EmisorIdentidad,
		RedesPermitidas:                     cfg.RedesPermitidas,
		DuracionMaximaAsercion:              time.Minute,
		EdadMaximaAutenticacion:             vidaAutenticacionConexionPerfiles,
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
		return nil, falloConfiguracion{clase: "superficie", causa: err}
	}
	red, err := httpseguridad.NuevaPoliticaRed(superficie)
	if err != nil {
		return nil, falloConfiguracion{clase: "red", causa: err}
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertificadoServidor, cfg.ClaveServidor)
	if err != nil {
		return nil, falloConfiguracion{clase: "tls", causa: err}
	}
	ca, err := cargarCA(cfg.CAAdministracion)
	if err != nil {
		return nil, falloConfiguracion{clase: "ca", causa: err}
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	verificar := func(r *http.Request) error {
		ahora := time.Now()
		if !ahora.Before(cfg.RetiradaEn) || r.Host != host.autoridad ||
			r.TLS == nil || !r.TLS.HandshakeComplete || r.TLS.DidResume ||
			len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) != 2 ||
			r.TLS.VerifiedChains[0][0] == nil ||
			len(r.TLS.PeerCertificates) != 1 || r.TLS.PeerCertificates[0] == nil ||
			!r.TLS.PeerCertificates[0].Equal(r.TLS.VerifiedChains[0][0]) {
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
		if perfiles != nil && perfiles.reloj != nil &&
			conexionPerfilesPorRenovar(r.Context(), perfiles.reloj.Ahora().UTC()) {
			// En HTTP/1.1 net/http cierra la conexión tras esta respuesta; la
			// siguiente petición llega por un handshake mTLS nuevo.
			w.Header().Set("Connection", "close")
		}
		ctx, err := ports.ConCorrelacionIncidenciasPeticion(r.Context())
		if err != nil {
			http.Error(w, "", http.StatusServiceUnavailable)
			return
		}
		r = r.WithContext(ctx)
		if err := verificar(r); err != nil {
			switch {
			case errors.Is(err, errCRLNoDisponible):
				log.Print(errCRLNoDisponible)
			case errors.Is(err, errCRLInvalida):
				log.Print(errCRLInvalida)
			case errors.Is(err, errParRemotoInvalido):
				log.Print(errParRemotoInvalido)
			}
			if perfiles != nil {
				codigo := "acceso_denegado"
				if errors.Is(err, errCRLNoDisponible) || errors.Is(err, errCRLInvalida) || errors.Is(err, errParRemotoInvalido) {
					codigo = "servicio_no_disponible"
				}
				if perfiles.auditor == nil || perfiles.auditor.RegistrarDenegacionADMIN(r.Context(), api.DenegacionADMIN{Codigo: codigo}) != nil {
					http.Error(w, "", http.StatusServiceUnavailable)
					return
				}
			}
			http.Error(w, "", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/livez" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if perfiles != nil && perfiles.atiende(r.URL.Path) {
			perfiles.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	var contextoConexion func(context.Context, net.Conn) context.Context
	if perfiles != nil {
		contextoConexion = perfiles.contextoConexion
	}
	return &http.Server{
		ConnContext:       contextoConexion,
		Addr:              cfg.Escucha,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       duracionMaximaPeticionADMIN,
		WriteTimeout:      duracionMaximaPeticionADMIN,
		// La renovación a los 3 minutos (Connection: close) más estos límites
		// de petición e inactividad garantizan que ninguna petición llegue por
		// una conexión cuya autenticación ya haya caducado.
		IdleTimeout: inactividadMaximaConexionADMIN,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			// La frontera rechaza toda sesión reanudada (DidResume): emitir
			// tickets solo provocaría denegaciones a navegadores legítimos.
			SessionTicketsDisabled: true,
			Certificates:           []tls.Certificate{cert},
			ClientAuth:             tls.RequireAndVerifyClientCert,
			ClientCAs:              raices,
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
	_, err := comprobarCertificadoVigenteHasta(hoja, ca, rutaCRL, ahora)
	return err
}

// comprobarCertificadoVigenteHasta comparte el único cotejo CRL del servidor
// con el resolver nominal. La evidencia de sesión no puede durar más que la CRL.
func comprobarCertificadoVigenteHasta(hoja, ca *x509.Certificate, rutaCRL string, ahora time.Time) (time.Time, error) {
	datos, err := leerMaterial(rutaCRL)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", errCRLNoDisponible, err)
	}
	bloque, resto := pem.Decode(datos)
	if bloque == nil || bloque.Type != "X509 CRL" || strings.TrimSpace(string(resto)) != "" {
		return time.Time{}, errCRLInvalida
	}
	crl, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", errCRLInvalida, err)
	}
	if err := crl.CheckSignatureFrom(ca); err != nil {
		return time.Time{}, fmt.Errorf("%w: firma: %w", errCRLInvalida, err)
	}
	if ahora.Before(crl.ThisUpdate) || !ahora.Before(crl.NextUpdate) {
		return time.Time{}, errCRLInvalida
	}
	for _, revocado := range crl.RevokedCertificateEntries {
		if hoja.SerialNumber.Cmp(revocado.SerialNumber) == 0 {
			return time.Time{}, errCertificadoRevocado
		}
	}
	return crl.NextUpdate.UTC(), nil
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

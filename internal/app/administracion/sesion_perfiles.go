package administracion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"net/netip"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	"vec-diputacion-granada/internal/vec/domain"
)

type resolvedorSesionPerfiles struct {
	cfg       Configuracion
	host      hostAdmin
	ca        []byte
	red       httpseguridad.PoliticaRed
	proveedor *adminperfiles.Proveedor
	reloj     httpseguridad.Reloj
}

// NuevoResolverSesionPerfiles reutiliza la frontera ADMIN directa de F. La
// cuenta y el perfil se resuelven después en las autoridades centrales.
func NuevoResolverSesionPerfiles(cfg Configuracion, deps adminperfiles.Dependencias) (*resolvedorSesionPerfiles, error) {
	host, hostValido := analizarHostAdmin(cfg.Host)
	if cfg.Entorno != "desarrollo" && cfg.Entorno != "cidonia" {
		return nil, falloConfiguracion{clase: "entorno"}
	}
	if deps.Reloj == nil {
		return nil, falloConfiguracion{clase: "dependencias"}
	}
	if !hostValido {
		return nil, falloConfiguracion{clase: "host"}
	}
	ca, err := cargarCA(cfg.CAAdministracion)
	if err != nil {
		return nil, falloConfiguracion{clase: "ca", causa: err}
	}
	config := superficieSesionPerfiles(cfg)
	red, err := httpseguridad.NuevaPoliticaRed(config)
	if err != nil {
		return nil, falloConfiguracion{clase: "red", causa: err}
	}
	proveedor, err := adminperfiles.Nuevo(config, deps)
	if err != nil {
		return nil, falloConfiguracion{clase: "resolver_sesion", causa: err}
	}
	return &resolvedorSesionPerfiles{cfg: cfg, host: host, ca: ca.Raw, red: red, proveedor: proveedor, reloj: deps.Reloj}, nil
}

// superficieSesionPerfiles fija la superficie ADMIN directa: certificado como
// único factor, garantía alta y cuenta privilegiada.
func superficieSesionPerfiles(cfg Configuracion) httpseguridad.ConfiguracionSuperficie {
	return httpseguridad.ConfiguracionSuperficie{
		Superficie:       httpseguridad.SuperficieAdministracionPrivilegiada,
		ZonaRed:          httpseguridad.ZonaRedAdministracion,
		DireccionEscucha: cfg.Escucha, Audiencia: cfg.Audiencia, EmisorIdentidad: cfg.EmisorIdentidad,
		RedesPermitidas:        cfg.RedesPermitidas,
		DuracionMaximaAsercion: time.Minute, EdadMaximaAutenticacion: vidaAutenticacionConexionPerfiles,
		MetodosAdmitidos:          []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		FactoresRequeridos:        []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados: 1, MinimoGruposCriptograficosDistintos: 1,
		GarantiaMinima: domain.AuthAssuranceHigh, RequiereCuentaPrivilegiada: true,
		PoliticaAdministracion:           httpseguridad.PoliticaAdministracionCertificadoTemporal,
		RetiradaPoliticaAdministracionEn: cfg.RetiradaEn, CertificadoClienteDirecto: true,
	}
}

func (s *resolvedorSesionPerfiles) ResolverSesionADMIN(ctx context.Context, r *http.Request) (api.SesionConfiable, error) {
	o, err := s.ObservarADMIN(ctx, r)
	if err != nil {
		return api.SesionConfiable{}, err
	}
	return s.proveedor.Resolver(ctx, r, o)
}

// ObservarADMIN se comparte con el selector de perfil activo. Devuelve solo
// hechos del canal verificado; no resuelve ni concede un perfil de negocio.
func (s *resolvedorSesionPerfiles) ObservarADMIN(ctx context.Context, r *http.Request) (adminperfiles.ObservacionADMIN, error) {
	var vacia adminperfiles.ObservacionADMIN
	if s == nil || ctx == nil || ctx.Err() != nil || r == nil || r.TLS == nil ||
		!r.TLS.HandshakeComplete || r.TLS.DidResume || r.Host != s.host.autoridad ||
		len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) != 2 ||
		len(r.TLS.PeerCertificates) != 1 || r.TLS.PeerCertificates[0] == nil {
		return vacia, api.ErrAutenticacionRequerida
	}
	// El certificado debe proceder del propio handshake; una cadena inyectada
	// en una petición o cabecera nunca basta.
	ca, err := cargarCA(s.cfg.CAAdministracion)
	if err != nil || !bytesIguales(ca.Raw, s.ca) || !r.TLS.PeerCertificates[0].Equal(r.TLS.VerifiedChains[0][0]) {
		return vacia, api.ErrAutenticacionRequerida
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !ahora.Before(s.cfg.RetiradaEn) || !cadenaDirectaVigente(r.TLS.VerifiedChains[0], ca, ahora) {
		return vacia, api.ErrAutenticacionRequerida
	}
	direccion, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return vacia, api.ErrAutenticacionRequerida
	}
	ip, err := netip.ParseAddr(direccion)
	if err != nil || s.red.Autorizar(ip) != nil {
		return vacia, api.ErrAutenticacionRequerida
	}
	for _, cabecera := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Remote-User", "X-Forwarded-Client-Cert", "X-Client-Cert", "X-SSL-Client-Cert"} {
		if len(r.Header.Values(cabecera)) != 0 {
			return vacia, api.ErrAutenticacionRequerida
		}
	}
	autenticada, err := autenticacionConexionPerfiles(ctx, r)
	if err != nil || autenticada.After(ahora) || !ahora.Before(autenticada.Add(vidaAutenticacionConexionPerfiles)) {
		return vacia, api.ErrAutenticacionRequerida
	}
	hoja := r.TLS.VerifiedChains[0][0]
	crlHasta, err := comprobarCertificadoVigenteHasta(hoja, ca, s.cfg.CRLAdministracion, ahora)
	if err != nil || !ahora.Before(crlHasta) {
		return vacia, api.ErrAutenticacionRequerida
	}
	certSHA, caSHA := sha256.Sum256(hoja.Raw), sha256.Sum256(ca.Raw)
	return adminperfiles.ObservacionADMIN{
		Entorno: s.cfg.Entorno, Host: s.host.nombre, Autoridad: s.host.autoridad, Audiencia: s.cfg.Audiencia,
		CertificadoSHA256: hex.EncodeToString(certSHA[:]), CASHA256: hex.EncodeToString(caSHA[:]),
		AutenticacionVerificadaEn: autenticada, RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: crlHasta.UTC().Truncate(time.Microsecond), CertificadoVigenteHasta: hoja.NotAfter.UTC().Truncate(time.Microsecond),
	}, nil
}

func bytesIguales(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diferencia byte
	for i := range a {
		diferencia |= a[i] ^ b[i]
	}
	return diferencia == 0
}

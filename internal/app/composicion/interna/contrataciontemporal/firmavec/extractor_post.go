package firmavec

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type EmisorAsercionCertificadoFirmaVecV2 interface {
	EmitirRegistroFirmaVecPreparada(context.Context, httpseguridad.AsercionProxyIdentidad) ([]byte, error)
}

// La implementación real consulta el registro privado y la CRL de la raíz.
type AcreditacionCertificadoFirmaVecV2 struct {
	PersonaRef string
	CuentaRef  string
}

type AcreditadorCertificadoFirmaVecV2 interface {
	AcreditarCertificadoFirmaVecV2(context.Context, *x509.Certificate, *x509.Certificate, time.Time) (AcreditacionCertificadoFirmaVecV2, error)
}

type extractorCertificadoFirmaVecV2 struct {
	emisor      EmisorAsercionCertificadoFirmaVecV2
	acreditador AcreditadorCertificadoFirmaVecV2
	emisorID    string
	audiencia   string
	retirada    time.Time
	reloj       vp.Reloj
}

func (e *extractorCertificadoFirmaVecV2) Extraer(r *http.Request) ([]byte, error) {
	if e == nil || r == nil || r.URL == nil || r.TLS == nil || r.Context().Err() != nil ||
		dependenciaNula(e.emisor) || dependenciaNula(e.acreditador) || dependenciaNula(e.reloj) ||
		r.Method != http.MethodPost || r.URL.Path != httpinterno.RutaRegistroFirmaVec ||
		!cabecerasCertificadoFirmaVecV2Validas(r.Header) ||
		!r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 ||
		len(r.TLS.PeerCertificates) == 0 || len(r.TLS.VerifiedChains) != 1 ||
		len(r.TLS.VerifiedChains[0]) < 2 ||
		len(r.TLS.PeerCertificates) > len(r.TLS.VerifiedChains[0]) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	for i, certificado := range r.TLS.PeerCertificates {
		if certificado == nil || r.TLS.VerifiedChains[0][i] == nil ||
			!bytes.Equal(certificado.Raw, r.TLS.VerifiedChains[0][i].Raw) {
			return nil, errIdentidadCertificadoFirmaVecNoDisponible
		}
	}
	hoja, ca := r.TLS.PeerCertificates[0], r.TLS.VerifiedChains[0][1]
	if hoja == nil || ca == nil || r.TLS.VerifiedChains[0][0] == nil ||
		!bytes.Equal(hoja.Raw, r.TLS.VerifiedChains[0][0].Raw) ||
		hoja.IsCA || hoja.KeyUsage&x509.KeyUsageDigitalSignature == 0 ||
		!certificadoClienteFirmaVecV2(hoja) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	ahora := e.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(hoja.NotBefore) || !ahora.Before(hoja.NotAfter) ||
		!ahora.Before(e.retirada) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	identidad, err := e.acreditador.AcreditarCertificadoFirmaVecV2(r.Context(), hoja, ca, ahora)
	if err != nil || identidad.PersonaRef == "" || identidad.CuentaRef == "" {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	canal, err := httpseguridad.ReferenciaCanalAsercionPasarela(*r.TLS, httpseguridad.SuperficieInternaCorporativa)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	hasta := ahora.Add(2 * time.Minute)
	if hoja.NotAfter.Before(hasta) {
		hasta = hoja.NotAfter.UTC().Truncate(time.Microsecond)
	}
	if e.retirada.Before(hasta) {
		hasta = e.retirada
	}
	if !ahora.Before(hasta) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	var aleatorio [24]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	sesionID := "ses_" + hex.EncodeToString(aleatorio[:])
	clear(aleatorio[:])
	huella := sha256.Sum256(hoja.Raw)
	credencial := "sha256:" + hex.EncodeToString(huella[:])
	asercion := httpseguridad.AsercionProxyIdentidad{Emisor: e.emisorID, Audiencia: e.audiencia,
		Superficie: httpseguridad.SuperficieInternaCorporativa,
		SujetoID:   identidad.PersonaRef,
		Cuenta:     httpseguridad.CuentaAcceso{ID: identidad.CuentaRef, SujetoVinculadoID: identidad.PersonaRef},
		SesionID:   sesionID, CanalVinculadoRef: canal,
		AutenticacionVerificadaEn: ahora, EmitidaEn: ahora, NoAntesDe: ahora,
		ExpiraEn: hasta, MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Factores: []httpseguridad.FactorAutenticacion{{Metodo: httpseguridad.MetodoCertificado,
			SujetoVinculadoID: identidad.PersonaRef, CredencialRef: "cert:" + credencial,
			EvidenciaRef:          "tls:verified:" + credencial,
			GrupoCriptograficoRef: "key:" + credencial, VerificadoEn: ahora}},
	}
	protegida, err := e.emisor.EmitirRegistroFirmaVecPreparada(r.Context(), asercion)
	if err != nil {
		clear(protegida)
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return protegida, nil
}

func certificadoClienteFirmaVecV2(c *x509.Certificate) bool {
	for _, uso := range c.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}

func cabecerasCertificadoFirmaVecV2Validas(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		if n == "authorization" || n == "proxy-authorization" || n == "cookie" ||
			n == strings.ToLower(httpseguridad.CabeceraAsercionPasarela) || n == "forwarded" ||
			n == "remote-user" || n == "x-remote-user" || n == "x-authenticated-user" || n == "x-user" ||
			strings.HasPrefix(n, "x-forwarded-") || strings.HasPrefix(n, "x-auth-") ||
			strings.HasPrefix(n, "x-identity-") || strings.HasPrefix(n, "x-client-") ||
			strings.HasPrefix(n, "x-ssl-") {
			return false
		}
	}
	return true
}

package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var errIdentidadCertificadoFirmaVecNoDisponible = errors.New("contratacion temporal: identidad certificado firma vec no disponible")

// Estos puertos pertenecen a las autoridades de la raíz interna. El
// preparador común fija el hash del cuerpo exacto una vez, con la cota propia
// de registro-vec; el extractor no lee ni modifica el cuerpo HTTP.
type PreparadorPeticionCertificadoFirmaVecV2 interface {
	PrepararPeticionFirmaVecV2(*http.Request) (*http.Request, httpseguridad.VinculoPeticionPasarela, error)
}

type VinculadorCertificadoFirmaVecV2 interface {
	AutenticarYVincular(context.Context, []byte) (context.Context, error)
}

type EmisorAsercionCertificadoFirmaVecV2 interface {
	Emitir(context.Context, httpseguridad.AsercionProxyIdentidad, httpseguridad.VinculoPeticionPasarela) ([]byte, error)
}

// La implementación real usa el registro privado de certificados de la
// misma raíz: confirma pertenencia, estado y CRL vigente antes de devolver
// las dos referencias. El emisor común las acredita otra vez al firmar.
type AcreditacionCertificadoFirmaVecV2 struct {
	PersonaRef string
	CuentaRef  string
}

type AcreditadorCertificadoFirmaVecV2 interface {
	AcreditarCertificadoFirmaVecV2(context.Context, *x509.Certificate, *x509.Certificate, time.Time) (AcreditacionCertificadoFirmaVecV2, error)
}

type ConfiguracionIdentidadCertificadoFirmaVecV2 struct {
	Fuente      identidadordinaria.ConfiguracionFuenteCertificadoTemporalConAsignacionVigente
	Vinculador  VinculadorCertificadoFirmaVecV2
	Preparador  PreparadorPeticionCertificadoFirmaVecV2
	Emisor      EmisorAsercionCertificadoFirmaVecV2
	Acreditador AcreditadorCertificadoFirmaVecV2
	Auditoria   vp.RegistradorAuditoriaFronteraRutaExacta
	Origen      string
	EmisorID    string
	Audiencia   string
	Reloj       vp.Reloj
}

type EntornoIdentidadCertificadoFirmaVecV2 struct {
	fuente     *identidadordinaria.FuenteCertificadoTemporal
	autoridad  *autoridadSesionFirmanteV2
	vinculador VinculadorCertificadoFirmaVecV2
	preparador PreparadorPeticionCertificadoFirmaVecV2
	extractor  *extractorCertificadoFirmaVecV2
	auditoria  vp.RegistradorAuditoriaFronteraRutaExacta
	origen     string
	host       string
}

// La raíz inyecta la misma instancia ServicioIdentidad en Fuente y en la
// fachada vinculadora. Una instancia distinta falla al extraer la cápsula;
// el puente también coteja el exportador TLS del request con la sesión común.
func NuevaIdentidadCertificadoFirmaVecV2(c ConfiguracionIdentidadCertificadoFirmaVecV2) (*EntornoIdentidadCertificadoFirmaVecV2, error) {
	if c.Fuente.Identidad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.Vinculador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.Preparador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.Emisor) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.Acreditador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.Auditoria) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.Reloj) ||
		c.EmisorID == "" || c.Audiencia == "" {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	origen, err := url.Parse(c.Origen)
	if err != nil || origen.Scheme != "https" || origen.Host == "" || origen.Hostname() == "" ||
		origen.User != nil || origen.Path != "" || origen.RawPath != "" || origen.RawQuery != "" ||
		origen.Fragment != "" || origen.Opaque != "" || origen.String() != c.Origen {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	c.Fuente.Reloj = c.Reloj
	fuente, err := identidadordinaria.NuevaFuenteCertificadoTemporalConAsignacionVigente(c.Fuente)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	autoridad, err := nuevaAutoridadSesionFirmanteV2Certificado(fuente, c.Reloj)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return &EntornoIdentidadCertificadoFirmaVecV2{fuente: fuente, autoridad: autoridad,
		vinculador: c.Vinculador, preparador: c.Preparador,
		extractor: &extractorCertificadoFirmaVecV2{emisor: c.Emisor, acreditador: c.Acreditador,
			emisorID: c.EmisorID, audiencia: c.Audiencia, retirada: c.Fuente.Politica.RetiradaEn, reloj: c.Reloj},
		auditoria: c.Auditoria, origen: c.Origen, host: origen.Host}, nil
}

// Envolver consume la cápsula de C4 exactamente una vez antes de preparar el
// contenedor CT. No acepta rutas ajenas ni identidades en el cuerpo/cabeceras.
func (e *EntornoIdentidadCertificadoFirmaVecV2) Envolver(siguiente http.Handler) (http.Handler, error) {
	if e == nil || e.fuente == nil || e.autoridad == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(siguiente) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return &puenteCertificadoFirmaVecV2{entorno: e, siguiente: siguiente}, nil
}

type puenteCertificadoFirmaVecV2 struct {
	entorno   *EntornoIdentidadCertificadoFirmaVecV2
	siguiente http.Handler
}

func (p *puenteCertificadoFirmaVecV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.entorno == nil || r == nil || r.URL == nil || w == nil {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path != httpinterno.RutaRegistroFirmaVec || r.URL.RawPath != "" ||
		r.URL.EscapedPath() != r.URL.Path || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.Opaque != "" || r.URL.User != nil || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" || r.RequestURI != r.URL.Path {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderPuenteCertificadoFirmaVecV2(w, http.StatusMethodNotAllowed)
		return
	}
	e := p.entorno
	if !origenFirmaVecV2Valido(r, e.origen, e.host) {
		p.denegar(w, r, http.StatusForbidden, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	if e.fuente == nil || e.autoridad == nil || e.extractor == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.preparador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.vinculador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.siguiente) {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusServiceUnavailable)
		return
	}
	if r.Context().Err() != nil || r.TLS == nil || !r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 {
		p.denegar(w, r, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	preparada, vinculo, err := e.preparador.PrepararPeticionFirmaVecV2(r)
	if err != nil || preparada == nil || preparada.Context() == nil || preparada.Context().Err() != nil ||
		preparada.Method != r.Method || preparada.Host != r.Host || preparada.URL == nil ||
		preparada.URL.RequestURI() != r.URL.RequestURI() || preparada.RequestURI != r.RequestURI ||
		!origenFirmaVecV2Valido(preparada, e.origen, e.host) ||
		vinculo.Metodo != http.MethodPost || vinculo.Ruta != httpinterno.RutaRegistroFirmaVec ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(vinculo.CuerpoSHA256) {
		p.denegar(w, r, http.StatusBadRequest, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	if preparada.Body != nil {
		defer preparada.Body.Close()
	}
	canalOriginal, canalValido := canalSesionFirmanteV2(r)
	canalPreparado, preparadoValido := canalSesionFirmanteV2(preparada)
	if !canalValido || !preparadoValido ||
		subtle.ConstantTimeCompare([]byte(canalOriginal), []byte(canalPreparado)) != 1 {
		p.denegar(w, r, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	preparada = peticionIdentidadConsultasContratacionTemporalDesarrollo(preparada)
	asercion, err := e.extractor.extraer(preparada, vinculo)
	if err != nil || len(asercion) == 0 {
		clear(asercion)
		p.denegar(w, preparada, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	defer clear(asercion)
	ctx, err := e.vinculador.AutenticarYVincular(preparada.Context(), asercion)
	if err != nil || ctx == nil || ctx.Err() != nil {
		p.denegar(w, preparada, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	ligada := preparada.WithContext(ctx)
	sesion, err := e.fuente.Abrir(ctx)
	canal, valido := canalSesionFirmanteV2(ligada)
	if err != nil || !valido || subtle.ConstantTimeCompare([]byte(sesion.CanalSHA256()), []byte(canal)) != 1 {
		p.denegar(w, ligada, http.StatusServiceUnavailable, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	lista, err := prepararPeticionFirmaVecV2(ligada)
	if err != nil || lista == nil {
		p.denegar(w, ligada, http.StatusServiceUnavailable, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	p.siguiente.ServeHTTP(w, lista)
}

func origenFirmaVecV2Valido(r *http.Request, origen, host string) bool {
	if r == nil || r.Host != host || len(r.Header.Values("Origin")) != 1 ||
		r.Header.Get("Origin") != origen || len(r.Header.Values("Referer")) > 1 ||
		len(r.Header.Values("Sec-Fetch-Site")) > 1 {
		return false
	}
	if sitio := r.Header.Get("Sec-Fetch-Site"); sitio != "" && sitio != "same-origin" {
		return false
	}
	if referencia := r.Header.Get("Referer"); referencia != "" {
		u, err := url.Parse(referencia)
		if err != nil || u.Scheme != "https" || u.Host != host || u.User != nil || u.Fragment != "" {
			return false
		}
	}
	return true
}

func (p *puenteCertificadoFirmaVecV2) denegar(w http.ResponseWriter, r *http.Request, estado int,
	motivo vp.MotivoAuditoriaFronteraRutaExacta,
) {
	if p == nil || p.entorno == nil || r == nil || p.entorno.auditoria == nil {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusServiceUnavailable)
		return
	}
	var correlacion [16]byte
	if _, err := rand.Read(correlacion[:]); err != nil {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusServiceUnavailable)
		return
	}
	orden := vp.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_" + hex.EncodeToString(correlacion[:]),
		Motivo: motivo, Superficie: vp.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
		Ruta: httpinterno.RutaRegistroFirmaVec}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(250*time.Millisecond))
	defer cancelar()
	if orden.Validar() != nil || p.entorno.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctx, orden) != nil {
		estado = http.StatusServiceUnavailable
	}
	responderPuenteCertificadoFirmaVecV2(w, estado)
}

func responderPuenteCertificadoFirmaVecV2(w http.ResponseWriter, estado int) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
}

type extractorCertificadoFirmaVecV2 struct {
	emisor      EmisorAsercionCertificadoFirmaVecV2
	acreditador AcreditadorCertificadoFirmaVecV2
	emisorID    string
	audiencia   string
	retirada    time.Time
	reloj       vp.Reloj
}

func (e *extractorCertificadoFirmaVecV2) extraer(r *http.Request,
	vinculo httpseguridad.VinculoPeticionPasarela,
) ([]byte, error) {
	if e == nil || r == nil || r.URL == nil || r.TLS == nil || r.Context().Err() != nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.emisor) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.acreditador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.reloj) ||
		r.Method != http.MethodPost || r.URL.Path != httpinterno.RutaRegistroFirmaVec ||
		vinculo.Metodo != r.Method || vinculo.Ruta != r.URL.RequestURI() ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(vinculo.CuerpoSHA256) ||
		!cabecerasCertificadoFirmaVecV2Validas(r.Header) ||
		!r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 ||
		len(r.TLS.PeerCertificates) != 1 || len(r.TLS.VerifiedChains) != 1 ||
		len(r.TLS.VerifiedChains[0]) < 2 {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
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
	protegida, err := e.emisor.Emitir(r.Context(), asercion, vinculo)
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

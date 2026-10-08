package firmavec

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var errIdentidadCertificadoFirmaVecNoDisponible = errors.New("contratacion temporal: identidad certificado firma vec no disponible")

// La raíz interna aporta su misma fachada y el emisor de la pasarela común.
// El vínculo del cuerpo permanece en el contexto privado de httpseguridad.
type VinculadorCertificadoFirmaVecV2 interface {
	AutenticarYVincular(context.Context, []byte) (context.Context, error)
}

type ConfiguracionIdentidadCertificadoFirmaVecV2 struct {
	Fuente      identidadordinaria.ConfiguracionFuenteCertificadoTemporalConAsignacionVigente
	Vinculador  VinculadorCertificadoFirmaVecV2
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
	sesion     *FuenteCertificadoTemporal
	vinculador VinculadorCertificadoFirmaVecV2
	extractor  *extractorCertificadoFirmaVecV2
	auditoria  vp.RegistradorAuditoriaFronteraRutaExacta
	origen     string
	host       string
}

// La raíz inyecta la misma instancia ServicioIdentidad en Fuente y en la
// fachada vinculadora. Una instancia distinta falla al extraer la cápsula;
// el puente también coteja el exportador TLS del request con la sesión común.
func NuevaIdentidadCertificadoFirmaVecV2(c ConfiguracionIdentidadCertificadoFirmaVecV2) (*EntornoIdentidadCertificadoFirmaVecV2, error) {
	if c.Fuente.Identidad == nil || dependenciaNula(c.Vinculador) ||
		dependenciaNula(c.Emisor) || dependenciaNula(c.Acreditador) ||
		dependenciaNula(c.Auditoria) || dependenciaNula(c.Reloj) ||
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
	sesion, err := NuevaFuenteCertificadoTemporal(fuente, c.Reloj)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return &EntornoIdentidadCertificadoFirmaVecV2{fuente: fuente, sesion: sesion,
		vinculador: c.Vinculador,
		extractor: &extractorCertificadoFirmaVecV2{emisor: c.Emisor, acreditador: c.Acreditador,
			emisorID: c.EmisorID, audiencia: c.Audiencia, retirada: c.Fuente.Politica.RetiradaEn, reloj: c.Reloj},
		auditoria: c.Auditoria, origen: c.Origen, host: origen.Host}, nil
}

// Envolver consume la cápsula de C4 una vez. El manejador CT inserta después
// su contenedor AUT56 para esta misma petición y revalida cada efecto.
func (e *EntornoIdentidadCertificadoFirmaVecV2) Envolver(siguiente http.Handler) (http.Handler, error) {
	if e == nil || e.fuente == nil || e.sesion == nil || dependenciaNula(siguiente) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return &puenteCertificadoFirmaVecV2{entorno: e, siguiente: siguiente}, nil
}

func (e *EntornoIdentidadCertificadoFirmaVecV2) FuenteSesionFirmanteV2() ports.FuenteSesionFirmanteV2 {
	if e == nil {
		return nil
	}
	return e.sesion
}

func (e *EntornoIdentidadCertificadoFirmaVecV2) FuenteComun() *identidadordinaria.FuenteCertificadoTemporal {
	if e == nil {
		return nil
	}
	return e.fuente
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
	if e.fuente == nil || e.sesion == nil || e.extractor == nil ||
		dependenciaNula(e.vinculador) || dependenciaNula(p.siguiente) {
		responderPuenteCertificadoFirmaVecV2(w, http.StatusServiceUnavailable)
		return
	}
	if r.Context().Err() != nil || r.TLS == nil || !r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 {
		p.denegar(w, r, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	preparada, err := httpseguridad.PrepararPeticionAsercionPasarela(r, httpseguridad.LimiteCuerpoRegistroFirmaVecPasarela)
	if err != nil || preparada == nil || preparada.Context() == nil || preparada.Context().Err() != nil ||
		preparada.Method != r.Method || preparada.Host != r.Host || preparada.URL == nil ||
		preparada.URL.RequestURI() != r.URL.RequestURI() || preparada.RequestURI != r.RequestURI ||
		!origenFirmaVecV2Valido(preparada, e.origen, e.host) {
		p.denegar(w, r, http.StatusBadRequest, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	if preparada.Body != nil {
		defer preparada.Body.Close()
	}
	canalOriginal, canalValido := canalFirmaVec(r)
	canalPreparado, preparadoValido := canalFirmaVec(preparada)
	if !canalValido || !preparadoValido ||
		subtle.ConstantTimeCompare([]byte(canalOriginal), []byte(canalPreparado)) != 1 {
		p.denegar(w, r, http.StatusUnauthorized, vp.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		return
	}
	asercion, err := e.extractor.Extraer(preparada)
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
	canal, valido := canalFirmaVec(ligada)
	if err != nil || !valido || subtle.ConstantTimeCompare([]byte(sesion.CanalSHA256()), []byte(canal)) != 1 {
		p.denegar(w, ligada, http.StatusServiceUnavailable, vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
		return
	}
	p.siguiente.ServeHTTP(w, ligada)
}

func canalFirmaVec(r *http.Request) (string, bool) {
	if r == nil || r.TLS == nil {
		return "", false
	}
	referencia, err := httpseguridad.ReferenciaCanalAsercionPasarela(*r.TLS, httpseguridad.SuperficieInternaCorporativa)
	if err != nil || referencia == "" {
		return "", false
	}
	suma := sha256.Sum256([]byte(referencia))
	return hex.EncodeToString(suma[:]), true
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

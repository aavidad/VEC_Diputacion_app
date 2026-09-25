// Package validadorautofirma traduce el puerto de verificacion de firma de
// Documentos al validador de AutofirmaV2 desplegado como servicio aparte.
//
// VEC no importa codigo de AutofirmaV2: este adaptador solo habla su API REST
// `POST /verify` (JSON con `content_base64` y, para firmas separadas,
// `original_content_base64`). La traduccion es deliberadamente conservadora:
// AutofirmaV2 no informa hoy de sello de tiempo ni expone un estado de
// revocacion explicito, por lo que el adaptador nunca devuelve una firma
// valida; como maximo `indeterminada`. Si detecta defectos concluyentes
// (integridad, certificado o confianza invalidos) devuelve `no_valida`.
package validadorautofirma

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	// RutaVerificacion es la ruta REST del validador de AutofirmaV2.
	RutaVerificacion = "/verify"

	tiempoPredeterminado = 20 * time.Second
	tiempoMaximo         = 60 * time.Second
	// La respuesta de AutofirmaV2 incluye detalles textuales que VEC descarta;
	// se acota antes de decodificar para no reservar memoria sin limite.
	maximaRespuesta   = 256 << 10
	minimoToken       = 32
	maximoToken       = 512
	maximoFirmantes   = 16
	nombreDocumento   = "documento"
	estadoNoInformado = "no_informado"
)

var (
	// ErrConfiguracion indica una configuracion privada incompleta o insegura.
	ErrConfiguracion = errors.New("documentos: configuracion del validador de firma invalida")
	errRedireccion   = errors.New("documentos: redireccion del validador prohibida")
)

// Configuracion procede de la configuracion privada de despliegue, nunca de
// Git ni de datos del navegador.
//
// URL es el origen exacto `https://host[:puerto]`, sin ruta ni consulta.
// CAPEM es la CA que emite el certificado del servidor (la CA local del
// validador o la del proxy que lo publique). NombreServidorTLS permite fijar
// el nombre verificado cuando difiere del host de la URL (por ejemplo
// `localhost`, que es el unico nombre del certificado local de AutofirmaV2).
// Debe existir al menos una credencial: Token (Bearer admitido por
// AutofirmaV2) o certificado cliente para mTLS (exigido por el proxy).
type Configuracion struct {
	URL                   string
	CAPEM                 []byte
	NombreServidorTLS     string
	Token                 []byte
	CertificadoClientePEM []byte
	ClaveClientePEM       []byte
	Timeout               time.Duration
}

// Cliente implementa ports.VerificadorFirma y ports.VerificadorFirmaMotivado.
// No tiene estado global: cada instancia posee su transporte.
type Cliente struct {
	url   string
	token string
	http  *http.Client
}

var (
	_ ports.VerificadorFirma         = (*Cliente)(nil)
	_ ports.VerificadorFirmaMotivado = (*Cliente)(nil)
)

// Nuevo valida la configuracion y construye un cliente con TLS 1.3, sin proxy
// ambiental, sin redirecciones y con limites de tiempo.
func Nuevo(config Configuracion) (*Cliente, error) {
	destino, err := url.Parse(config.URL)
	if err != nil || destino.Scheme != "https" || destino.Host == "" || destino.Opaque != "" ||
		destino.User != nil || destino.RawQuery != "" || destino.ForceQuery || destino.Fragment != "" ||
		(destino.Path != "" && destino.Path != "/") || strings.HasSuffix(destino.Hostname(), ".") {
		return nil, ErrConfiguracion
	}
	raices := x509.NewCertPool()
	if len(config.CAPEM) == 0 || !raices.AppendCertsFromPEM(config.CAPEM) {
		return nil, ErrConfiguracion
	}
	token := string(config.Token)
	if token != "" && !tokenValido(token) {
		return nil, ErrConfiguracion
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    raices,
		ServerName: strings.TrimSpace(config.NombreServidorTLS),
	}
	hayCertificado := len(config.CertificadoClientePEM) > 0 || len(config.ClaveClientePEM) > 0
	if hayCertificado {
		certificado, err := tls.X509KeyPair(config.CertificadoClientePEM, config.ClaveClientePEM)
		if err != nil {
			return nil, ErrConfiguracion
		}
		tlsConfig.Certificates = []tls.Certificate{certificado}
	}
	if token == "" && !hayCertificado {
		return nil, ErrConfiguracion
	}
	limite := config.Timeout
	if limite == 0 {
		limite = tiempoPredeterminado
	}
	if limite < time.Millisecond || limite > tiempoMaximo {
		return nil, ErrConfiguracion
	}
	transporte := &http.Transport{
		Proxy:                  nil,
		TLSClientConfig:        tlsConfig,
		TLSHandshakeTimeout:    limite,
		ResponseHeaderTimeout:  limite,
		DisableKeepAlives:      true,
		DisableCompression:     true,
		MaxResponseHeaderBytes: 16 << 10,
		ForceAttemptHTTP2:      false,
	}
	return &Cliente{
		url:   strings.TrimSuffix(config.URL, "/") + RutaVerificacion,
		token: token,
		http: &http.Client{
			Transport:     transporte,
			Timeout:       limite,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errRedireccion },
		},
	}, nil
}

func tokenValido(token string) bool {
	if len(token) < minimoToken || len(token) > maximoToken {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] <= ' ' || token[i] > '~' {
			return false
		}
	}
	return true
}

// peticionAutofirma reproduce solo los campos necesarios de la API de
// AutofirmaV2. El nombre es fijo: no se envia el nombre real del fichero.
type peticionAutofirma struct {
	Name           string `json:"name"`
	ContentBase64  string `json:"content_base64"`
	OriginalBase64 string `json:"original_content_base64,omitempty"`
}

// respuestaAutofirma decodifica solo lo necesario. Asunto, emisor, detalles
// libres y avisos del proveedor se descartan al decodificar.
type respuestaAutofirma struct {
	OK     bool                `json:"ok"`
	Valid  bool                `json:"valid"`
	Result *resultadoAutofirma `json:"result"`
}

type resultadoAutofirma struct {
	Valid           bool                `json:"valid"`
	Details         []string            `json:"details"`
	Integrity       aspectoAutofirma    `json:"integrity"`
	Certificate     aspectoAutofirma    `json:"certificate"`
	Trust           aspectoAutofirma    `json:"trust"`
	SignerSummaries []firmanteAutofirma `json:"signerSummaries"`
}

type aspectoAutofirma struct {
	Status string `json:"status"`
}

type firmanteAutofirma struct {
	Fingerprint string `json:"fingerprint"`
}

// Verificar implementa ports.VerificadorFirma. Devuelve el resultado tipado;
// solo `valida` con ValidarContra positivo habilitaria la firma.
func (c *Cliente) Verificar(ctx context.Context, s ports.SolicitudVerificacionFirma) (ports.ResultadoVerificacionFirma, error) {
	motivada, err := c.VerificarMotivado(ctx, s)
	if err != nil {
		return ports.ResultadoVerificacionFirma{}, err
	}
	return motivada.Resultado, nil
}

// VerificarMotivado implementa ports.VerificadorFirmaMotivado. Cualquier fallo
// de transporte, credencial o formato se traduce a `indeterminada` con motivo
// de catalogo, sin texto del proveedor. Solo una solicitud invalida es error.
func (c *Cliente) VerificarMotivado(ctx context.Context, s ports.SolicitudVerificacionFirma) (ports.VerificacionFirmaMotivada, error) {
	if c == nil || c.http == nil || ctx == nil || s.Validar() != nil {
		return ports.VerificacionFirmaMotivada{}, ports.ErrVerificacionFirmaInvalida
	}
	suma := sha256.Sum256(s.ContenidoFirmado)
	base := ports.ResultadoVerificacionFirma{
		Estado:               ports.EstadoVerificacionIndeterminada,
		HuellaOriginalSHA256: s.HuellaOriginalSHA256,
		// Huella calculada por VEC sobre los bytes enviados: AutofirmaV2 no
		// devuelve eco de huellas.
		HuellaFirmadoSHA256: hex.EncodeToString(suma[:]),
		SelloTiempoEstado:   estadoNoInformado,
		RevocacionEstado:    estadoNoInformado,
	}
	// PAdES lleva el original incrustado y AutofirmaV2 ignora el original
	// aportado; no se envia para minimizar datos.
	incrustada := bytes.HasPrefix(s.ContenidoFirmado, []byte("%PDF-"))
	peticion := peticionAutofirma{
		Name:          nombreDocumento,
		ContentBase64: base64.StdEncoding.EncodeToString(s.ContenidoFirmado),
	}
	if !incrustada {
		peticion.OriginalBase64 = base64.StdEncoding.EncodeToString(s.ContenidoOriginal)
	}
	recibida, motivo := c.llamar(ctx, peticion)
	if motivo != "" {
		return motivar(base, motivo), nil
	}
	return traducir(base, recibida, !incrustada), nil
}

func (c *Cliente) llamar(ctx context.Context, peticion peticionAutofirma) (respuestaAutofirma, ports.MotivoVerificacionFirma) {
	var cero respuestaAutofirma
	cuerpo, err := json.Marshal(peticion)
	if err != nil {
		return cero, ports.MotivoValidadorNoDisponible
	}
	solicitud, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(cuerpo))
	if err != nil {
		return cero, ports.MotivoValidadorNoDisponible
	}
	solicitud.Header.Set("Content-Type", "application/json")
	solicitud.Header.Set("Accept", "application/json")
	if c.token != "" {
		solicitud.Header.Set("Authorization", "Bearer "+c.token)
	}
	respuesta, err := c.http.Do(solicitud)
	if err != nil {
		return cero, ports.MotivoValidadorNoDisponible
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(respuesta.Body, maximaRespuesta))
		_ = respuesta.Body.Close()
	}()
	switch {
	case respuesta.StatusCode == http.StatusUnauthorized || respuesta.StatusCode == http.StatusForbidden:
		return cero, ports.MotivoCredencialRechazada
	case respuesta.StatusCode == http.StatusBadRequest || respuesta.StatusCode == http.StatusRequestEntityTooLarge ||
		respuesta.StatusCode == http.StatusUnprocessableEntity:
		return cero, ports.MotivoRechazadaPorValidador
	case respuesta.StatusCode != http.StatusOK:
		return cero, ports.MotivoValidadorNoDisponible
	}
	tipo, _, err := mime.ParseMediaType(respuesta.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" {
		return cero, ports.MotivoRespuestaNoInterpretable
	}
	contenido, err := io.ReadAll(io.LimitReader(respuesta.Body, maximaRespuesta+1))
	if err != nil {
		return cero, ports.MotivoValidadorNoDisponible
	}
	if len(contenido) > maximaRespuesta {
		return cero, ports.MotivoRespuestaNoInterpretable
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	var recibida respuestaAutofirma
	if lector.Decode(&recibida) != nil || lector.Decode(new(any)) != io.EOF {
		return cero, ports.MotivoRespuestaNoInterpretable
	}
	return recibida, ""
}

func motivar(r ports.ResultadoVerificacionFirma, m ports.MotivoVerificacionFirma) ports.VerificacionFirmaMotivada {
	r.Estado = m.EstadoAsociado()
	return ports.VerificacionFirmaMotivada{Resultado: r, Motivo: m}
}

// traducir aplica la politica de fallo cerrado. Precedencia: incoherencia de
// la respuesta; defectos concluyentes; y, para respuestas positivas, cada
// comprobacion que AutofirmaV2 no acredita.
func traducir(r ports.ResultadoVerificacionFirma, a respuestaAutofirma, separada bool) ports.VerificacionFirmaMotivada {
	res := a.Result
	if !a.OK || res == nil || res.Valid != a.Valid || len(res.SignerSummaries) > maximoFirmantes ||
		!estadoConocido(res.Integrity.Status) || !estadoConocido(res.Certificate.Status) ||
		!estadoConocido(res.Trust.Status) {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	switch {
	case res.Integrity.Status == "invalid":
		return motivar(r, ports.MotivoIntegridadNoValida)
	case res.Certificate.Status == "invalid":
		return motivar(r, ports.MotivoCertificadoNoValido)
	case res.Trust.Status == "invalid":
		return motivar(r, ports.MotivoConfianzaNoValida)
	case !res.Valid || res.Integrity.Status != "valid":
		// Negativo sin aspecto concluyente: no se inventa la causa.
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	// La firma solo esta ligada al original custodiado cuando AutofirmaV2 la
	// verifico en modo separado contra los bytes enviados por VEC.
	r.VinculoOriginal = separada && contieneMarcador(res.Details, "modo=detached")
	if len(res.SignerSummaries) == 1 && huellaValida(res.SignerSummaries[0].Fingerprint) {
		r.CertificadoHuellaSHA256 = res.SignerSummaries[0].Fingerprint
	}
	switch {
	case !r.VinculoOriginal:
		return motivar(r, ports.MotivoVinculoOriginalNoAcreditado)
	case r.CertificadoHuellaSHA256 == "":
		return motivar(r, ports.MotivoFirmanteNoIdentificado)
	case res.Certificate.Status != "valid":
		return motivar(r, ports.MotivoCertificadoNoAcreditado)
	case res.Trust.Status != "valid":
		return motivar(r, ports.MotivoConfianzaNoAcreditada)
	}
	// AutofirmaV2 no expone un estado de revocacion explicito ni verifica
	// sellos de tiempo; `certificate: valid` no prueba revocacion vigente.
	return motivar(r, ports.MotivoRevocacionNoAcreditada)
}

func estadoConocido(s string) bool {
	return s == "valid" || s == "invalid" || s == "warning" || s == "unknown"
}

func contieneMarcador(detalles []string, marcador string) bool {
	for _, d := range detalles {
		if d == marcador {
			return true
		}
	}
	return false
}

func huellaValida(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] < '0' || s[i] > '9') && (s[i] < 'a' || s[i] > 'f') {
			return false
		}
	}
	return true
}

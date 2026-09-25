// Package validadorautofirma traduce el puerto de verificacion de firma de
// Documentos al validador de AutofirmaV2 desplegado como servicio aparte en
// modo `-rest-solo-verificacion`.
//
// VEC no importa codigo de AutofirmaV2: este adaptador solo habla su API REST
// `POST /verify` (JSON con `content_base64` y `original_content_base64`) e
// interpreta exclusivamente el `dictamen` con contrato
// `autofirmav2.dictamen-verificacion.v1`. Los campos heredados (`valid`,
// `result`) se ignoran. Cualquier otro contrato, estado fuera de catalogo,
// huella de eco distinta o veredicto incoherente con sus aspectos se traduce
// a `indeterminada` / `respuesta_no_interpretable`.
//
// VEC no delega el veredicto: recalcula el suyo sobre los aspectos del
// dictamen con ports.PoliticaVerificacionFirmaV1 y solo lo acepta si coincide
// con el del validador, o si es mas estricto que un `valida` de este.
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
	// ContratoDictamen es el unico contrato de dictamen que se interpreta.
	ContratoDictamen = "autofirmav2.dictamen-verificacion.v1"

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
	OriginalBase64 string `json:"original_content_base64"`
}

// respuestaAutofirma decodifica solo el dictamen. Asunto, emisor, serie,
// motivos tecnicos, fuentes, fechas y los campos heredados se descartan.
type respuestaAutofirma struct {
	Dictamen *dictamenAutofirma `json:"dictamen"`
}

type dictamenAutofirma struct {
	Contrato                string              `json:"contrato"`
	Estado                  string              `json:"estado"`
	Motivo                  string              `json:"motivo"`
	Integridad              aspectoAutofirma    `json:"integridad"`
	Cadena                  aspectoAutofirma    `json:"cadena"`
	Certificado             aspectoAutofirma    `json:"certificado"`
	Revocacion              aspectoAutofirma    `json:"revocacion"`
	SelloTiempo             aspectoAutofirma    `json:"selloTiempo"`
	VinculoOriginal         aspectoAutofirma    `json:"vinculoOriginal"`
	HuellaFirmadoSHA256     string              `json:"huellaFirmadoSHA256"`
	HuellaOriginalSHA256    string              `json:"huellaOriginalSHA256"`
	CertificadoHuellaSHA256 string              `json:"certificadoHuellaSHA256"`
	Firmantes               []firmanteAutofirma `json:"firmantes"`
	Extensiones             extensionesDictamen `json:"extensiones"`
}

type aspectoAutofirma struct {
	Estado string `json:"estado"`
}

type firmanteAutofirma struct {
	CertificadoHuellaSHA256 string           `json:"certificadoHuellaSHA256"`
	Cadena                  aspectoAutofirma `json:"cadena"`
	Certificado             aspectoAutofirma `json:"certificado"`
	Revocacion              aspectoAutofirma `json:"revocacion"`
	SelloTiempo             aspectoAutofirma `json:"selloTiempo"`
}

type extensionesDictamen struct {
	RevocacionRemota  string `json:"revocacionRemota"`
	SelloTiempoRemoto string `json:"selloTiempoRemoto"`
}

// Catalogos cerrados del contrato v1. Son constantes de lectura: no se
// modifican en ejecucion.
var (
	estadosIntegridad  = []string{"valida", "parcial", "no_valida"}
	estadosCadena      = []string{"valida", "no_valida", "no_comprobada"}
	estadosCertificado = []string{"vigente", "no_vigente", "uso_no_permitido", "no_comprobado"}
	estadosRevocacion  = []string{ports.RevocacionVigente, ports.RevocacionRevocado, ports.RevocacionNoComprobada}
	estadosSello       = []string{ports.SelloTiempoNoPresente, ports.SelloTiempoValido,
		ports.SelloTiempoNoValido, ports.SelloTiempoNoComprobado}
	estadosVinculo   = []string{"acreditado", "no_acreditado", "no_aportado"}
	estadosExtension = []string{"desactivada", "activa"}
	motivosDictamen  = map[string]ports.MotivoVerificacionFirma{
		"verificada":                     ports.MotivoFirmaVerificada,
		"integridad_no_valida":           ports.MotivoIntegridadNoValida,
		"certificado_no_valido":          ports.MotivoCertificadoNoValido,
		"confianza_no_valida":            ports.MotivoConfianzaNoValida,
		"integridad_parcial":             ports.MotivoIntegridadParcial,
		"firmante_no_identificado":       ports.MotivoFirmanteNoIdentificado,
		"certificado_no_acreditado":      ports.MotivoCertificadoNoAcreditado,
		"confianza_no_acreditada":        ports.MotivoConfianzaNoAcreditada,
		"revocacion_no_acreditada":       ports.MotivoRevocacionNoAcreditada,
		"sello_tiempo_no_acreditado":     ports.MotivoSelloTiempoNoAcreditado,
		"vinculo_original_no_acreditado": ports.MotivoVinculoOriginalNoAcreditado,
	}
)

// Verificar implementa ports.VerificadorFirma. Devuelve el resultado tipado;
// solo `valida` con ValidarContra positivo habilita la firma.
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
		// Huella calculada por VEC sobre los bytes enviados; el eco del
		// validador solo se compara, nunca se adopta.
		HuellaFirmadoSHA256: hex.EncodeToString(suma[:]),
		SelloTiempoEstado:   estadoNoInformado,
		RevocacionEstado:    estadoNoInformado,
	}
	// El original se envia siempre, tambien en PAdES: AutofirmaV2 acredita el
	// vinculo cuando el original es prefijo exacto del PDF firmado y una firma
	// cubre todos sus bytes. Sin original no hay vinculo que acreditar.
	peticion := peticionAutofirma{
		Name:           nombreDocumento,
		ContentBase64:  base64.StdEncoding.EncodeToString(s.ContenidoFirmado),
		OriginalBase64: base64.StdEncoding.EncodeToString(s.ContenidoOriginal),
	}
	recibida, motivo := c.llamar(ctx, peticion)
	if motivo != "" {
		return motivar(base, motivo), nil
	}
	return traducir(base, recibida.Dictamen), nil
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

// traducir interpreta el dictamen con fallo cerrado. Primero valida contrato,
// catalogos y huellas de eco; despues recalcula el veredicto con la politica
// de VEC y lo contrasta con el declarado por el validador.
func traducir(r ports.ResultadoVerificacionFirma, d *dictamenAutofirma) ports.VerificacionFirmaMotivada {
	if !dictamenInterpretable(d) || d.HuellaFirmadoSHA256 != r.HuellaFirmadoSHA256 ||
		(d.HuellaOriginalSHA256 != "" && d.HuellaOriginalSHA256 != r.HuellaOriginalSHA256) {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	declarado := motivosDictamen[d.Motivo]
	if string(declarado.EstadoAsociado()) != d.Estado {
		return motivar(r, ports.MotivoRespuestaNoInterpretable)
	}
	sinAdoptar := r
	r.VinculoOriginal = d.VinculoOriginal.Estado == "acreditado"
	r.RevocacionEstado = d.Revocacion.Estado
	r.SelloTiempoEstado = d.SelloTiempo.Estado
	if len(d.Firmantes) == 1 && d.CertificadoHuellaSHA256 != "" {
		// La referencia del firmante es la huella de su certificado; su
		// correspondencia con la persona la resuelve la aplicacion.
		r.CertificadoHuellaSHA256 = d.CertificadoHuellaSHA256
		r.FirmanteRef = "ref:" + d.CertificadoHuellaSHA256
	}
	propio := veredictoVEC(d)
	switch {
	case propio == declarado:
		return motivar(r, propio)
	case declarado == ports.MotivoFirmaVerificada:
		// VEC es mas estricta que el validador (original no aportado o varios
		// firmantes): prevalece su motivo, nunca el positivo.
		return motivar(r, propio)
	default:
		// Un veredicto negativo que no casa con sus aspectos indica deriva
		// del contrato: no se adopta ningun aspecto de esa respuesta.
		return motivar(sinAdoptar, ports.MotivoRespuestaNoInterpretable)
	}
}

// veredictoVEC aplica ports.PoliticaVerificacionFirmaV1 sobre los aspectos
// agregados, con la misma precedencia que el contrato: defectos
// concluyentes antes que comprobaciones no concluidas.
func veredictoVEC(d *dictamenAutofirma) ports.MotivoVerificacionFirma {
	switch {
	case d.Integridad.Estado == "no_valida":
		return ports.MotivoIntegridadNoValida
	case d.Certificado.Estado == "no_vigente" || d.Certificado.Estado == "uso_no_permitido" ||
		d.Revocacion.Estado == ports.RevocacionRevocado:
		return ports.MotivoCertificadoNoValido
	case d.Cadena.Estado == "no_valida":
		return ports.MotivoConfianzaNoValida
	case d.Integridad.Estado != "valida":
		return ports.MotivoIntegridadParcial
	case len(d.Firmantes) == 0:
		return ports.MotivoFirmanteNoIdentificado
	case d.Certificado.Estado != "vigente":
		return ports.MotivoCertificadoNoAcreditado
	case d.Cadena.Estado != "valida":
		return ports.MotivoConfianzaNoAcreditada
	case d.Revocacion.Estado != ports.RevocacionVigente:
		return ports.MotivoRevocacionNoAcreditada
	case !ports.SelloTiempoAdmisible(d.SelloTiempo.Estado):
		return ports.MotivoSelloTiempoNoAcreditado
	case d.VinculoOriginal.Estado != "acreditado":
		return ports.MotivoVinculoOriginalNoAcreditado
	case len(d.Firmantes) != 1 || d.CertificadoHuellaSHA256 == "":
		return ports.MotivoFirmanteNoIdentificado
	default:
		return ports.MotivoFirmaVerificada
	}
}

// dictamenInterpretable exige contrato exacto, estados de catalogo en todos
// los aspectos y firmantes, y huellas bien formadas y coherentes.
func dictamenInterpretable(d *dictamenAutofirma) bool {
	if d == nil || d.Contrato != ContratoDictamen ||
		!en(d.Integridad.Estado, estadosIntegridad) || !en(d.Cadena.Estado, estadosCadena) ||
		!en(d.Certificado.Estado, estadosCertificado) || !en(d.Revocacion.Estado, estadosRevocacion) ||
		!en(d.SelloTiempo.Estado, estadosSello) || !en(d.VinculoOriginal.Estado, estadosVinculo) ||
		!en(d.Extensiones.RevocacionRemota, estadosExtension) ||
		!en(d.Extensiones.SelloTiempoRemoto, estadosExtension) ||
		len(d.Firmantes) > maximoFirmantes || !huellaValida(d.HuellaFirmadoSHA256) ||
		(d.HuellaOriginalSHA256 != "" && !huellaValida(d.HuellaOriginalSHA256)) {
		return false
	}
	if _, ok := motivosDictamen[d.Motivo]; !ok {
		return false
	}
	for _, f := range d.Firmantes {
		if !huellaValida(f.CertificadoHuellaSHA256) || !en(f.Cadena.Estado, estadosCadena) ||
			!en(f.Certificado.Estado, estadosCertificado) || !en(f.Revocacion.Estado, estadosRevocacion) ||
			!en(f.SelloTiempo.Estado, estadosSello) {
			return false
		}
	}
	switch {
	case d.CertificadoHuellaSHA256 == "":
		return len(d.Firmantes) != 1
	case len(d.Firmantes) != 1:
		return false
	default:
		return huellaValida(d.CertificadoHuellaSHA256) &&
			d.CertificadoHuellaSHA256 == d.Firmantes[0].CertificadoHuellaSHA256 &&
			d.CertificadoHuellaSHA256 != strings.Repeat("0", sha256.Size*2)
	}
}

func en(valor string, catalogo []string) bool {
	for _, c := range catalogo {
		if valor == c {
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

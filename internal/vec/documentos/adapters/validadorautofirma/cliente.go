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
// a `indeterminada` / `respuesta_no_interpretable`. Tambien lo son un
// dictamen con claves desconocidas o duplicadas, datos sobrantes, un eco del
// original ausente cuando el vinculo se evaluo, o agregados que no son el
// peor estado de sus firmantes (regla de agregacion del contrato v1).
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
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

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
	maximaRespuesta = 256 << 10
	minimoToken     = 32
	maximoToken     = 512
	maximoFirmantes = 16
	// maximaProfundidad acota el recorrido de claves duplicadas; el contrato
	// v1 y sus campos heredados no superan cuatro niveles.
	maximaProfundidad = 32
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

// respuestaAutofirma toma el dictamen en bruto; los campos heredados de
// primer nivel se ignoran. El dictamen se decodifica despues con
// DisallowUnknownFields contra el contrato v1 completo.
type respuestaAutofirma struct {
	Dictamen json.RawMessage `json:"dictamen"`
}

// dictamenAutofirma enumera todas las claves del contrato v1. Las que VEC no
// usa (formato, fecha de comprobacion, asunto, emisor, serie, motivos
// tecnicos, fuentes y fechas de aspecto) se aceptan como json.RawMessage para
// que DisallowUnknownFields no las rechace, y se descartan sin interpretarlas.
type dictamenAutofirma struct {
	Contrato                string              `json:"contrato"`
	Estado                  string              `json:"estado"`
	Motivo                  string              `json:"motivo"`
	Formato                 json.RawMessage     `json:"formato"`
	ComprobadoEn            json.RawMessage     `json:"comprobadoEn"`
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
	Estado string          `json:"estado"`
	Motivo json.RawMessage `json:"motivo"`
	Fuente json.RawMessage `json:"fuente"`
	Fecha  json.RawMessage `json:"fecha"`
}

type firmanteAutofirma struct {
	CertificadoHuellaSHA256 string           `json:"certificadoHuellaSHA256"`
	Serie                   json.RawMessage  `json:"serie"`
	Asunto                  json.RawMessage  `json:"asunto"`
	Emisor                  json.RawMessage  `json:"emisor"`
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
	estadosVinculo = []string{"acreditado", "no_acreditado", "no_aportado"}
	// Ordenes de agregacion del contrato v1, de peor a mejor. El agregado de
	// cada aspecto es el peor estado de sus firmantes; sin firmantes, el
	// estado por defecto del contrato.
	ordenCadena      = []string{"no_valida", "no_comprobada", "valida"}
	ordenCertificado = []string{"uso_no_permitido", "no_vigente", "no_comprobado", "vigente"}
	ordenRevocacion  = []string{ports.RevocacionRevocado, ports.RevocacionNoComprobada, ports.RevocacionVigente}
	ordenSello       = []string{ports.SelloTiempoNoValido, ports.SelloTiempoNoComprobado,
		ports.SelloTiempoNoPresente, ports.SelloTiempoValido}
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
	return traducir(base, recibida), nil
}

func (c *Cliente) llamar(ctx context.Context, peticion peticionAutofirma) (*dictamenAutofirma, ports.MotivoVerificacionFirma) {
	var cero *dictamenAutofirma
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
	dictamen, ok := decodificarRespuesta(contenido)
	if !ok {
		return cero, ports.MotivoRespuestaNoInterpretable
	}
	return dictamen, ""
}

// decodificarRespuesta acepta exactamente un objeto JSON de hasta
// maximaRespuesta bytes, sin claves duplicadas en ningun nivel (tampoco por
// plegado de mayusculas, que encoding/json haria coincidir) y con un dictamen
// sin claves ajenas al contrato v1. Un dictamen ausente o nulo devuelve nil,
// que traducir tambien trata como no interpretable.
func decodificarRespuesta(contenido []byte) (*dictamenAutofirma, bool) {
	if len(contenido) > maximaRespuesta || !sinClavesDuplicadas(contenido) {
		return nil, false
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	var recibida respuestaAutofirma
	if lector.Decode(&recibida) != nil || lector.Decode(new(any)) != io.EOF {
		return nil, false
	}
	if len(recibida.Dictamen) == 0 || string(recibida.Dictamen) == "null" {
		return nil, true
	}
	estricto := json.NewDecoder(bytes.NewReader(recibida.Dictamen))
	estricto.DisallowUnknownFields()
	dictamen := new(dictamenAutofirma)
	if estricto.Decode(dictamen) != nil || estricto.Decode(new(any)) != io.EOF {
		return nil, false
	}
	return dictamen, true
}

// sinClavesDuplicadas recorre el documento por tokens y exige un unico valor
// de nivel superior, profundidad acotada y claves unicas por objeto tras el
// plegado que aplica encoding/json al emparejar campos.
func sinClavesDuplicadas(contenido []byte) bool {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	if recorrerValor(lector, 0) != nil {
		return false
	}
	_, err := lector.Token()
	return err == io.EOF
}

// errEstructuraRespuesta es la causa cerrada de una respuesta cuyo recorrido
// por tokens no es admisible; cuando procede de encoding/json la conserva.
var errEstructuraRespuesta = errors.New("validadorautofirma: estructura de respuesta no admisible")

func recorrerValor(lector *json.Decoder, profundidad int) error {
	token, err := lector.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	if profundidad >= maximaProfundidad {
		return errEstructuraRespuesta
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]struct{})
		for lector.More() {
			token, err := lector.Token()
			if err != nil {
				return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
			}
			clave, esClave := token.(string)
			if !esClave {
				return errEstructuraRespuesta
			}
			plegada := plegarClave(clave)
			if _, repetida := vistas[plegada]; repetida {
				return errEstructuraRespuesta
			}
			vistas[plegada] = struct{}{}
			if err := recorrerValor(lector, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for lector.More() {
			if err := recorrerValor(lector, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return errEstructuraRespuesta
	}
	if _, err = lector.Token(); err != nil {
		return fmt.Errorf("%w: %w", errEstructuraRespuesta, err)
	}
	return nil
}

// plegarClave reproduce el plegado sin distincion de mayusculas con el que
// encoding/json asocia claves a campos, para detectar duplicados equivalentes.
func plegarClave(clave string) string {
	return strings.Map(func(r rune) rune { return unicode.ToUpper(unicode.ToLower(r)) }, clave)
}

func motivar(r ports.ResultadoVerificacionFirma, m ports.MotivoVerificacionFirma) ports.VerificacionFirmaMotivada {
	r.Estado = m.EstadoAsociado()
	return ports.VerificacionFirmaMotivada{Resultado: r, Motivo: m}
}

// traducir interpreta el dictamen con fallo cerrado. Primero valida contrato,
// catalogos y huellas de eco; despues recalcula el veredicto con la politica
// de VEC y lo contrasta con el declarado por el validador.
func traducir(r ports.ResultadoVerificacionFirma, d *dictamenAutofirma) ports.VerificacionFirmaMotivada {
	// VEC envia siempre el original: salvo que el validador declare no haberlo
	// recibido (`no_aportado`, sin eco), su eco es obligatorio y exacto.
	if !dictamenInterpretable(d) || d.HuellaFirmadoSHA256 != r.HuellaFirmadoSHA256 ||
		(d.VinculoOriginal.Estado == "no_aportado") != (d.HuellaOriginalSHA256 == "") ||
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
// concluyentes antes que comprobaciones no concluidas. Solo es seguro porque
// dictamenInterpretable ya exigio que cada agregado sea exactamente el peor
// estado de los firmantes: ningun defecto de un firmante queda oculto.
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
	// Coherencia exacta firmantes-agregados: con un firmante, sus cuatro
	// aspectos son los agregados; con varios, cada agregado es el peor.
	if d.Cadena.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Cadena.Estado }, ordenCadena, "no_comprobada") ||
		d.Certificado.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Certificado.Estado }, ordenCertificado, "no_comprobado") ||
		d.Revocacion.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.Revocacion.Estado }, ordenRevocacion, ports.RevocacionNoComprobada) ||
		d.SelloTiempo.Estado != agregado(d.Firmantes, func(f firmanteAutofirma) string { return f.SelloTiempo.Estado }, ordenSello, ports.SelloTiempoNoPresente) {
		return false
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

// agregado devuelve el peor estado de los firmantes segun orden (de peor a
// mejor), o porDefecto sin firmantes, como compone el contrato v1. Los
// estados ya se validaron contra catalogo, que coincide con el orden.
func agregado(firmantes []firmanteAutofirma, estado func(firmanteAutofirma) string, orden []string, porDefecto string) string {
	if len(firmantes) == 0 {
		return porDefecto
	}
	peor := len(orden)
	for _, f := range firmantes {
		for i, candidato := range orden {
			if candidato == estado(f) && i < peor {
				peor = i
			}
		}
	}
	if peor == len(orden) {
		return ""
	}
	return orden[peor]
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

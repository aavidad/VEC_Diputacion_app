// Package validadorautofirma consulta GrxFirma por TLS 1.3 y traduce solo el
// dictamen v2. Los indicadores heredados de la envoltura nunca acreditan firma.
package validadorautofirma

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
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
	// RutaVerificacion es la ruta REST del validador de GrxFirma.
	RutaVerificacion = "/v2/verify"
	// ContratoDictamen es el unico contrato de dictamen que se interpreta.
	ContratoDictamen = "autofirmav2.dictamen-verificacion.v2"

	tiempoPredeterminado = 20 * time.Second
	tiempoMaximo         = 60 * time.Second
	// La respuesta de GrxFirma incluye detalles textuales que VEC descarta;
	// se acota antes de decodificar para no reservar memoria sin limite.
	maximaRespuesta = 256 << 10
	minimoToken     = 32
	maximoToken     = 512
	// maximaProfundidad acota el recorrido de claves duplicadas; el contrato
	// v2 y la envoltura heredada tienen estructura acotada.
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
// `localhost`, que es el unico nombre del certificado local de GrxFirma).
// Debe existir al menos una credencial: Token (Bearer admitido por
// GrxFirma) o certificado cliente para mTLS (exigido por el proxy).
type Configuracion struct {
	URL                   string
	CAPEM                 []byte
	NombreServidorTLS     string
	Token                 []byte
	CertificadoClientePEM []byte
	ClaveClientePEM       []byte
	Timeout               time.Duration
	// Disponibilidad observa únicamente una respuesta 5xx o un dictamen
	// interpretable del origen autorizado. El receptor debe ser no bloqueante.
	// No recibe errores, credenciales ni contenido del documento.
	Disponibilidad func(context.Context, bool)
}

// Cliente implementa ports.VerificadorFirma y ports.VerificadorFirmaMotivado.
// No tiene estado global: cada instancia posee su transporte.
type Cliente struct {
	url            string
	token          string
	http           *http.Client
	disponibilidad func(context.Context, bool)
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
		url:            strings.TrimSuffix(config.URL, "/") + RutaVerificacion,
		token:          token,
		disponibilidad: config.Disponibilidad,
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
// GrxFirma. El nombre es fijo: no se envia el nombre real del fichero.
type peticionAutofirma struct {
	ContratoSolicitado string `json:"contrato_solicitado"`
	Name               string `json:"name"`
	ContentBase64      string `json:"content_base64"`
	OriginalBase64     string `json:"original_content_base64"`
}

var motivosDictamen = map[string]ports.MotivoVerificacionFirma{
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
	recibida, motivo, err := c.verificarDictamenV2(ctx, s)
	if err != nil {
		return ports.VerificacionFirmaMotivada{}, err
	}
	if motivo != "" {
		return motivar(base, motivo), nil
	}
	resultado := traducirV2(base, &recibida, s)
	if s.FormatoEsperado != "" && resultado.Motivo == ports.MotivoFirmaVerificada &&
		resultado.Resultado.Formato != s.FormatoEsperado {
		// El dictamen puede ser positivo para otro formato, pero ese positivo
		// no satisface el contrato de este consumidor. No adoptar sus aspectos.
		return motivar(base, ports.MotivoRespuestaNoInterpretable), nil
	}
	return resultado, nil
}

func (c *Cliente) llamar(ctx context.Context, peticion peticionAutofirma) (*dictamenV2, ports.MotivoVerificacionFirma) {
	var cero *dictamenV2
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
	case respuesta.StatusCode >= http.StatusInternalServerError && respuesta.StatusCode <= 599:
		c.observarDisponibilidad(ctx, false)
		return cero, ports.MotivoValidadorNoDisponible
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
	dictamen, err := decodificarRespuestaV2(contenido)
	if err != nil {
		return cero, ports.MotivoRespuestaNoInterpretable
	}
	return dictamen, ""
}

func motivar(r ports.ResultadoVerificacionFirma, m ports.MotivoVerificacionFirma) ports.VerificacionFirmaMotivada {
	r.Estado = m.EstadoAsociado()
	return ports.VerificacionFirmaMotivada{Resultado: r, Motivo: m}
}

// observarDisponibilidad no cambia la decisión de verificación. Una respuesta
// bien formada demuestra disponibilidad incluso si su dictamen es no_valida.
func (c *Cliente) observarDisponibilidad(ctx context.Context, disponible bool) {
	if c != nil && c.disponibilidad != nil && ctx != nil && ctx.Err() == nil {
		c.disponibilidad(ctx, disponible)
	}
}

package verificadorhttp

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
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	maximaRespuesta = 16 << 10
	tiempoMaximo    = 10 * time.Second
)

var ErrConfiguracion = errors.New("documentos: configuracion del verificador invalida")
var ErrVerificacionNoDisponible = errors.New("documentos: verificacion de firma no disponible")
var ErrRespuestaInvalida = errors.New("documentos: respuesta del verificador invalida")
var errRedireccion = errors.New("documentos: redireccion del verificador prohibida")

// Configuracion se obtiene de la configuracion privada de despliegue. El
// certificado cliente y la CA son obligatorios: TLS autentica ambos extremos.
// URL identifica el origen exacto del servicio, sin ruta, consulta ni usuario.
type Configuracion struct {
	URL                   string
	CAPEM                 []byte
	CertificadoClientePEM []byte
	ClaveClientePEM       []byte
	Timeout               time.Duration
}

// Cliente implementa el puerto mediante un servicio verificador separado.
// No acepta datos de un navegador como configuracion de destino o identidad.
type Cliente struct {
	url  string
	http *http.Client
}

var _ ports.VerificadorFirma = (*Cliente)(nil)

func Nuevo(config Configuracion) (*Cliente, error) {
	destino, err := url.Parse(config.URL)
	if err != nil || destino.Scheme != "https" || destino.Host == "" ||
		destino.User != nil || destino.RawQuery != "" || destino.Fragment != "" ||
		(destino.Path != "" && destino.Path != "/") ||
		strings.HasSuffix(destino.Host, ".") {
		return nil, ErrConfiguracion
	}
	if len(config.CAPEM) == 0 || len(config.CertificadoClientePEM) == 0 || len(config.ClaveClientePEM) == 0 {
		return nil, ErrConfiguracion
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(config.CAPEM) {
		return nil, ErrConfiguracion
	}
	certificado, err := tls.X509KeyPair(config.CertificadoClientePEM, config.ClaveClientePEM)
	if err != nil {
		return nil, ErrConfiguracion
	}
	limite := config.Timeout
	if limite == 0 {
		limite = tiempoMaximo
	}
	if limite < time.Millisecond || limite > tiempoMaximo {
		return nil, ErrConfiguracion
	}
	transporte := http.DefaultTransport.(*http.Transport).Clone()
	transporte.Proxy = nil
	transporte.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		RootCAs:      raices,
		Certificates: []tls.Certificate{certificado},
	}
	transporte.TLSHandshakeTimeout = limite
	transporte.ResponseHeaderTimeout = limite
	transporte.DisableKeepAlives = true
	return &Cliente{
		url: strings.TrimSuffix(config.URL, "/") + "/v1/verificaciones",
		http: &http.Client{
			Transport:     transporte,
			Timeout:       limite,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errRedireccion },
		},
	}, nil
}

type solicitudHTTP struct {
	VersionContrato      int    `json:"version_contrato"`
	DocumentoID          string `json:"documento_id"`
	Version              uint64 `json:"version"`
	HuellaOriginalSHA256 string `json:"huella_original_sha256"`
	HuellaFirmadoSHA256  string `json:"huella_firmado_sha256"`
	ContenidoOriginal    []byte `json:"contenido_original"`
	ContenidoFirmado     []byte `json:"contenido_firmado"`
}

type respuestaHTTP struct {
	VersionContrato         int    `json:"version_contrato"`
	Estado                  string `json:"estado"`
	HuellaOriginalSHA256    string `json:"huella_original_sha256"`
	HuellaFirmadoSHA256     string `json:"huella_firmado_sha256"`
	FirmanteRef             string `json:"firmante_ref"`
	CertificadoHuellaSHA256 string `json:"certificado_huella_sha256"`
	SelloTiempoEstado       string `json:"sello_tiempo_estado"`
	RevocacionEstado        string `json:"revocacion_estado"`
	VinculoOriginal         bool   `json:"vinculo_original"`
}

func (c *Cliente) Verificar(ctx context.Context, solicitud ports.SolicitudVerificacionFirma) (ports.ResultadoVerificacionFirma, error) {
	var cero ports.ResultadoVerificacionFirma
	if c == nil || c.http == nil || ctx == nil || solicitud.Validar() != nil {
		return cero, ports.ErrVerificacionFirmaInvalida
	}
	huellaFirmado := sha256.Sum256(solicitud.ContenidoFirmado)
	huellaFirmadoHex := hex.EncodeToString(huellaFirmado[:])
	cuerpo, err := json.Marshal(solicitudHTTP{
		VersionContrato:      1,
		DocumentoID:          solicitud.DocumentoID,
		Version:              solicitud.Version,
		HuellaOriginalSHA256: solicitud.HuellaOriginalSHA256,
		HuellaFirmadoSHA256:  huellaFirmadoHex,
		ContenidoOriginal:    solicitud.ContenidoOriginal,
		ContenidoFirmado:     solicitud.ContenidoFirmado,
	})
	if err != nil {
		return cero, ErrVerificacionNoDisponible
	}
	peticion, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(cuerpo))
	if err != nil {
		return cero, ErrVerificacionNoDisponible
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta, err := c.http.Do(peticion)
	if err != nil {
		return cero, ErrVerificacionNoDisponible
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusOK || respuesta.Header.Get("Content-Type") != "application/json" {
		return cero, ErrVerificacionNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(respuesta.Body, maximaRespuesta+1))
	if err != nil || len(contenido) > maximaRespuesta {
		return cero, ErrRespuestaInvalida
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var recibida respuestaHTTP
	if lector.Decode(&recibida) != nil || lector.Decode(new(any)) != io.EOF {
		return cero, ErrRespuestaInvalida
	}
	if recibida.VersionContrato != 1 || recibida.HuellaOriginalSHA256 != solicitud.HuellaOriginalSHA256 ||
		recibida.HuellaFirmadoSHA256 != huellaFirmadoHex {
		return cero, ErrRespuestaInvalida
	}
	resultado := ports.ResultadoVerificacionFirma{
		Estado:                  ports.EstadoVerificacionFirma(recibida.Estado),
		HuellaOriginalSHA256:    recibida.HuellaOriginalSHA256,
		HuellaFirmadoSHA256:     recibida.HuellaFirmadoSHA256,
		FirmanteRef:             recibida.FirmanteRef,
		CertificadoHuellaSHA256: recibida.CertificadoHuellaSHA256,
		SelloTiempoEstado:       recibida.SelloTiempoEstado,
		RevocacionEstado:        recibida.RevocacionEstado,
		VinculoOriginal:         recibida.VinculoOriginal,
	}
	if resultado.Estado != ports.EstadoVerificacionValida &&
		resultado.Estado != ports.EstadoVerificacionNoValida &&
		resultado.Estado != ports.EstadoVerificacionIndeterminada {
		return cero, ErrRespuestaInvalida
	}
	if resultado.Estado == ports.EstadoVerificacionValida && resultado.ValidarContra(solicitud) != nil {
		return cero, ErrRespuestaInvalida
	}
	return resultado, nil
}

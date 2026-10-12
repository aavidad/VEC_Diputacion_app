package analisiscontenido

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfiguracion = errors.New("analisiscontenido: configuracion invalida")
	ErrContenido     = errors.New("analisiscontenido: contenido no coincide")
	ErrRemoto        = errors.New("analisiscontenido: servicio no disponible")
)

const tamanoMaximoPDF = 32 << 20
const limiteMetadatos = 8 << 10
const limiteRespuestaHTTP = 32 << 10

// ConfiguracionCliente contiene rutas privadas de certificados, nunca su
// material. El nombre TLS debe figurar en el certificado del servidor.
type ConfiguracionCliente struct {
	Endpoint, NombreServidor, RutaCA, RutaCertificado, RutaClave string
	ConectorID                                                   string
}

type Cliente struct {
	endpoint, conectorID string
	http                 *http.Client
}

func NuevoCliente(c ConfiguracionCliente) (*Cliente, error) {
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		c.NombreServidor == "" || c.RutaCA == "" || c.RutaCertificado == "" || c.RutaClave == "" || !referenciaSegura(c.ConectorID, 128) {
		return nil, ErrConfiguracion
	}
	ca, err := leerArchivoTLS(c.RutaCA, false)
	if err != nil {
		return nil, ErrConfiguracion
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, ErrConfiguracion
	}
	certPEM, err := leerArchivoTLS(c.RutaCertificado, false)
	if err != nil {
		return nil, ErrConfiguracion
	}
	clavePEM, err := leerArchivoTLS(c.RutaClave, true)
	if err != nil {
		return nil, ErrConfiguracion
	}
	cert, err := tls.X509KeyPair(certPEM, clavePEM)
	if err != nil {
		return nil, ErrConfiguracion
	}
	transporte := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, Certificates: []tls.Certificate{cert}, ServerName: c.NombreServidor, MinVersion: tls.VersionTLS13},
		Proxy:           nil,
		MaxIdleConns:    2,
		MaxConnsPerHost: 2,
	}
	httpCliente := &http.Client{Transport: transporte, Timeout: 50 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Cliente{endpoint: strings.TrimRight(u.String(), "/"), conectorID: c.ConectorID, http: httpCliente}, nil
}

func (c *Cliente) Capacidades(ctx context.Context) (ports.CapacidadesAnalizadorContenido, error) {
	if c == nil || ctx == nil {
		return ports.CapacidadesAnalizadorContenido{}, ErrConfiguracion
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v1/capacidades", nil)
	if err != nil {
		return ports.CapacidadesAnalizadorContenido{}, ErrConfiguracion
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return ports.CapacidadesAnalizadorContenido{}, ErrRemoto
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ports.CapacidadesAnalizadorContenido{}, ErrRemoto
	}
	var capacidades ports.CapacidadesAnalizadorContenido
	if decodificarLimitado(resp.Body, limiteRespuestaHTTP, &capacidades) != nil || capacidades.ConectorID != c.conectorID ||
		ports.VerificarCapacidadesAnalizadorContenido(capacidades, ports.RequisitosAnalizadorContenido{AnalisisEnFlujo: true, CanalAutenticado: true, CifradoEnTransito: true, IdentidadMutua: true, ActualizacionFirmas: true, DetectaMalware: true, DetectaContenidoActivo: true, TamanoMinimo: 1}) != nil {
		return ports.CapacidadesAnalizadorContenido{}, ErrRemoto
	}
	return capacidades, nil
}

func (c *Cliente) Analizar(ctx context.Context, solicitud ports.SolicitudAnalizarContenido) (ports.ResultadoAnalisisContenido, error) {
	if c == nil || ctx == nil || solicitud.Validar() != nil || solicitud.MIME != "application/pdf" || solicitud.Tamano > tamanoMaximoPDF {
		return ports.ResultadoAnalisisContenido{}, ports.ErrSolicitudAnalisisContenidoInvalida
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil || proyeccion.ObjetoVinculado != solicitud.Objeto {
		return ports.ResultadoAnalisisContenido{}, ports.ErrSolicitudAnalisisContenidoInvalida
	}
	meta := solicitudRemota{
		Objeto: solicitud.Objeto, ConectorAlmacenID: solicitud.ConectorAlmacenID, Zona: solicitud.Zona, MIME: solicitud.MIME, Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256,
		CorrelacionRef: proyeccion.CorrelacionRef, AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad, Clasificacion: proyeccion.Clasificacion,
	}
	if meta.validar() != nil {
		return ports.ResultadoAnalisisContenido{}, ports.ErrSolicitudAnalisisContenidoInvalida
	}
	contenido, err := io.ReadAll(io.LimitReader(solicitud.Contenido, solicitud.Tamano+1))
	if err != nil || int64(len(contenido)) != solicitud.Tamano || huella(contenido) != strings.ToLower(solicitud.HuellaSHA256) {
		return ports.ResultadoAnalisisContenido{}, ErrContenido
	}
	var cuerpo bytes.Buffer
	m := multipart.NewWriter(&cuerpo)
	metadatos, err := json.Marshal(meta)
	if err != nil || len(metadatos) > limiteMetadatos {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	if err = m.WriteField("metadatos", string(metadatos)); err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	parte, err := m.CreateFormFile("archivo", "documento.pdf")
	if err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	if _, err = parte.Write(contenido); err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	if err = m.Close(); err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/analisis", &cuerpo)
	if err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrConfiguracion
	}
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return ports.ResultadoAnalisisContenido{}, ErrRemoto
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ports.ResultadoAnalisisContenido{}, ErrRemoto
	}
	var resultado ports.ResultadoAnalisisContenido
	if decodificarLimitado(resp.Body, limiteRespuestaHTTP, &resultado) != nil || resultado.ValidarContra(solicitud) != nil ||
		resultado.ConectorAnalizadorID != c.conectorID || resultado.VersionConector != 1 || resultado.BytesAnalizados > solicitud.Tamano {
		return ports.ResultadoAnalisisContenido{}, ports.ErrResultadoAnalisisContenidoInvalido
	}
	integra, err := evidenciaIntegra(resultado)
	if err != nil {
		return ports.ResultadoAnalisisContenido{}, errors.Join(ports.ErrResultadoAnalisisContenidoInvalido, err)
	}
	if !integra {
		return ports.ResultadoAnalisisContenido{}, ports.ErrResultadoAnalisisContenidoInvalido
	}
	return resultado, nil
}

func decodificarLimitado(r io.Reader, limite int64, destino any) error {
	b, err := io.ReadAll(io.LimitReader(r, limite+1))
	if err != nil || int64(len(b)) > limite {
		return ErrRemoto
	}
	if json.Unmarshal(b, destino) != nil {
		return ErrRemoto
	}
	return nil
}

func huella(b []byte) string { suma := sha256.Sum256(b); return hex.EncodeToString(suma[:]) }

func evidenciaIntegra(resultado ports.ResultadoAnalisisContenido) (bool, error) {
	esperada := resultado.HuellaEvidenciaSHA256
	resultado.HuellaEvidenciaSHA256 = ""
	bruto, err := json.Marshal(resultado)
	if err != nil {
		return false, err
	}
	actual := huella(bruto)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(esperada)) == 1, nil
}

var _ ports.AnalizadorContenido = (*Cliente)(nil)

// Package httpinterno consumes the nominal, audited Organisation query.
package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const rutaOrganizacionHistorica = "/api/vec/personal/organizacion-historica"

// MaximoBytesPagina is a required private transport limit supplied from the
// accredited server configuration. It does not declare source completeness.
type ConfiguracionOrganizacionHistorica struct {
	Origen, Directorio, AutoridadCA, CertificadoCliente, ClaveCliente string
	MaximoBytesPagina                                                 int64
}

type ClienteOrganizacionHistorica struct {
	origen     string
	cliente    *http.Client
	transporte *http.Transport
	maximo     int64
}

func NuevoClienteOrganizacionHistorica(c ConfiguracionOrganizacionHistorica) (*ClienteOrganizacionHistorica, error) {
	u, err := url.Parse(c.Origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.Origen || strings.TrimSpace(c.Origen) != c.Origen || u.Host != strings.ToLower(u.Host) || strings.HasSuffix(u.Hostname(), ".") || strings.HasSuffix(u.Host, ":") || c.MaximoBytesPagina < 1 || c.MaximoBytesPagina == math.MaxInt64 {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != p {
			return nil, domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	tlsConfig, err := cargarTLSOrganizacionHistorica(c)
	if err != nil {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	t := &http.Transport{TLSClientConfig: tlsConfig, DisableCompression: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxResponseHeaderBytes: 64 << 10, MaxIdleConnsPerHost: 1}
	client := &http.Client{Transport: t, Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &ClienteOrganizacionHistorica{origen: c.Origen, cliente: client, transporte: t, maximo: c.MaximoBytesPagina}, nil
}

func (c *ClienteOrganizacionHistorica) Cerrar() {
	if c != nil && c.transporte != nil {
		c.transporte.CloseIdleConnections()
	}
}

// No identity, organisation, profile, token or permission is transmitted by
// this client. The server fixes its actor/scope from the mTLS/F1 context; the
// expected organisation only constrains which response this consumer accepts.
func (c *ClienteOrganizacionHistorica) Consultar(ctx context.Context, s domain.SelectorOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio ports.ResultadoConsultaOrganizacionHistorica
	if c == nil || c.cliente == nil || ctx == nil || ctx.Err() != nil || s.Validar() != nil {
		return vacio, domain.ErrConsultaOrganizacionHistoricaInvalida
	}
	q := url.Values{"vigente_en": {string(s.VigenteEn)}, "conocido_en": {s.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z")}, "limite": {strconv.Itoa(s.Limite)}}
	for k, v := range map[string]string{"unidad_clave": s.UnidadClave, "version_rpt_ref": s.VersionRPTRef, "version_plantilla_ref": s.VersionPlantillaRef, "cursor": s.Cursor} {
		if v != "" {
			q.Set(k, v)
		}
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.origen+rutaOrganizacionHistorica+"?"+q.Encode(), nil)
	if err != nil {
		return vacio, domain.ErrConsultaOrganizacionHistoricaInvalida
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Cache-Control", "no-store")
	r.Header.Set("Pragma", "no-cache")
	resp, err := c.cliente.Do(r)
	if err != nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return vacio, domain.ErrConsultaOrganizacionHistoricaDenegada
	}
	media, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode != http.StatusOK || err != nil || media != "application/json" || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") || resp.Header.Get("Content-Encoding") != "" || len(resp.Header.Values("Set-Cookie")) != 0 || resp.Header.Get("Location") != "" || resp.ContentLength > c.maximo {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, c.maximo+1))
	if err != nil || int64(len(b)) > c.maximo || !utf8.Valid(b) || jsonUnicoOrganizacionHistorica(b) != nil || formaRespuestaOrganizacionHistorica(b) != nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	var envelope struct {
		Data ports.ResultadoConsultaOrganizacionHistorica `json:"data"`
	}
	if decoder.Decode(&envelope) != nil || decoder.Decode(new(any)) != io.EOF || validarResultadoOrganizacionHistorica(s, envelope.Data) != nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if ctx.Err() != nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return envelope.Data, nil
}

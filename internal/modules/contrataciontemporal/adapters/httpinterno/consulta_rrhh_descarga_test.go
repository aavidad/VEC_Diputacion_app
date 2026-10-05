package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type falloAuditadoDescargaPrueba struct{ causa error }

func (e falloAuditadoDescargaPrueba) Error() string { return "fallo auditado" }
func (e falloAuditadoDescargaPrueba) Unwrap() error { return e.causa }
func (e falloAuditadoDescargaPrueba) AcuseLecturaAuditada() vecports.AcuseIntentoAuditoria {
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_i_0123456789abcdef0123456789abcdef", Secuencia: 7,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: "correlacion_" + strings.Repeat("b", 32),
		RegistradaEn: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}
}

var _ ports.FalloLecturaAuditado = falloAuditadoDescargaPrueba{}

type registradorDescargaPrueba struct {
	solicitud      ports.SolicitudDescargaBorradorRRHH
	descargas      int
	fallos         []error
	errDescarga    error
	errFallo       error
	sinAnotarFallo bool
}

func (r *registradorDescargaPrueba) RegistrarDescarga(_ context.Context, s ports.SolicitudDescargaBorradorRRHH) (ports.ReciboDescargaBorradorRRHH, error) {
	r.descargas++
	r.solicitud = s
	if r.errDescarga != nil {
		return ports.ReciboDescargaBorradorRRHH{}, r.errDescarga
	}
	return ports.ReciboDescargaBorradorRRHH{DescargaRef: "descarga_borrador:" + strings.Repeat("c", 64),
		AuditoriaRef: "aud_v3_0123456789abcdef0123456789abcdef", DecisionRef: "decision:prueba",
		RegistradaEn: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}, nil
}

func (r *registradorDescargaPrueba) RegistrarFalloDescarga(_ context.Context, _ string, causa error) error {
	r.fallos = append(r.fallos, causa)
	if r.sinAnotarFallo {
		return ports.ErrDescargaBorradorRRHHSinAnotar
	}
	if r.errFallo != nil {
		return r.errFallo
	}
	return falloAuditadoDescargaPrueba{causa: causa}
}

func manejadorDescargaPrueba(t *testing.T, c *consultorDetalleRRHHPrueba, r *renderizadorBorradorRRHHPrueba, reg *registradorDescargaPrueba) http.Handler {
	t.Helper()
	h, err := NuevoManejadorConsultaDetalleRRHH(c, r)
	if err != nil {
		t.Fatal(err)
	}
	h, err = ConfigurarDescargaAuditadaConsultaDetalleRRHH(h, reg)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestDescargaBorradorConsumeDecisionLigadaAlArchivo(t *testing.T) {
	c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
	r := &renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4\nfixture descarga\n%%EOF\n")}
	reg := &registradorDescargaPrueba{}
	h := manejadorDescargaPrueba(t, c, r, reg)
	w := httptest.NewRecorder()
	p := peticionInformeRRHHPrueba()
	p.Header.Set("Accept", AcceptResolucionRRHH)
	h.ServeHTTP(w, p)
	suma := sha256.Sum256(r.contenido)
	if w.Code != http.StatusOK || reg.descargas != 1 || len(reg.fallos) != 0 || w.Body.String() != string(r.contenido) {
		t.Fatalf("descarga: estado=%d descargas=%d fallos=%d", w.Code, reg.descargas, len(reg.fallos))
	}
	s := reg.solicitud
	if s.ExpedienteRef != c.detalle.Resumen.ExpedienteRef || s.VersionExpediente != c.detalle.Resumen.Version ||
		s.Tipo != ports.BorradorResolucion || s.Formato != ports.FormatoBorradorRRHHPDF ||
		s.DocumentoSHA256 != hex.EncodeToString(suma[:]) || s.TamanoBytes != len(r.contenido) {
		t.Fatalf("la descarga no liga el archivo entregado: %+v", s)
	}
	if w.Header().Get("X-Audit-Ref") != "aud_v3_0123456789abcdef0123456789abcdef" || w.Header().Get("X-VEC-Documento-SHA256") != s.DocumentoSHA256 {
		t.Fatalf("cabeceras de acuse: %v", w.Header())
	}
}

func TestDescargaBorradorDenegadaNoEntregaArchivo(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"denegada auditada", falloAuditadoDescargaPrueba{causa: ports.ErrAutorizacionDenegada}, http.StatusForbidden, "acceso_denegado"},
		{"denegada sin registrador", ports.ErrAutorizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{"versión ausente", falloAuditadoDescargaPrueba{causa: ports.ErrDescargaBorradorRRHHVersionAusente}, http.StatusConflict, "documento_no_disponible"},
		{"registro caído", ports.ErrDescargaBorradorRRHHNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
			r := &renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4\nno se entrega\n%%EOF\n")}
			reg := &registradorDescargaPrueba{errDescarga: caso.err}
			h := manejadorDescargaPrueba(t, c, r, reg)
			w := httptest.NewRecorder()
			p := peticionInformeRRHHPrueba()
			p.Header.Set("Accept", AcceptResolucionRRHH)
			h.ServeHTTP(w, p)
			if w.Code != caso.estado || strings.Contains(w.Body.String(), "%PDF-") || w.Header().Get("Content-Disposition") != "" ||
				!strings.Contains(w.Body.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
			}
			var auditado ports.FalloLecturaAuditado
			if errors.As(caso.err, &auditado) != (w.Header().Get("X-Audit-Ref") != "") {
				t.Fatalf("acuse en cabecera incoherente: %v", w.Header())
			}
		})
	}
}

func TestDescargaBorradorFallosAnterioresQuedanAuditados(t *testing.T) {
	casos := []struct {
		nombre    string
		consulta  error
		contenido []byte
		render    error
		estado    int
	}{
		{"consulta no observable", application.ErrConsultaRRHHNoObservable, nil, nil, http.StatusNotFound},
		{"consulta no disponible", application.ErrConsultaRRHHNoDisponible, nil, nil, http.StatusServiceUnavailable},
		{"error al generar", nil, nil, errors.New("plantilla rota"), http.StatusInternalServerError},
		{"archivo no PDF", nil, []byte("no es PDF"), nil, http.StatusBadGateway},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba(), err: caso.consulta}
			r := &renderizadorBorradorRRHHPrueba{contenido: caso.contenido, err: caso.render}
			reg := &registradorDescargaPrueba{}
			h := manejadorDescargaPrueba(t, c, r, reg)
			w := httptest.NewRecorder()
			p := peticionInformeRRHHPrueba()
			p.Header.Set("Accept", AcceptResolucionRRHH)
			h.ServeHTTP(w, p)
			if w.Code != caso.estado || len(reg.fallos) != 1 || reg.descargas != 0 || w.Header().Get("X-Audit-Ref") == "" {
				t.Fatalf("estado=%d fallos=%d descargas=%d cabeceras=%v", w.Code, len(reg.fallos), reg.descargas, w.Header())
			}
			if caso.consulta != nil && !errors.Is(reg.fallos[0], caso.consulta) {
				t.Fatalf("causa perdida: %v", reg.fallos[0])
			}
		})
	}
	// Si el intento no se puede anotar, la respuesta es 503.
	c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba(), err: application.ErrConsultaRRHHNoObservable}
	reg := &registradorDescargaPrueba{errFallo: errors.New("registrador caído")}
	h := manejadorDescargaPrueba(t, c, &renderizadorBorradorRRHHPrueba{}, reg)
	w := httptest.NewRecorder()
	p := peticionInformeRRHHPrueba()
	p.Header.Set("Accept", AcceptResolucionRRHH)
	h.ServeHTTP(w, p)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("fallo sin anotar: estado=%d", w.Code)
	}
	// Sin nada que anotar (sin actor o sin registrador) se conserva el error.
	reg = &registradorDescargaPrueba{sinAnotarFallo: true}
	h = manejadorDescargaPrueba(t, c, &renderizadorBorradorRRHHPrueba{}, reg)
	w = httptest.NewRecorder()
	p = peticionInformeRRHHPrueba()
	p.Header.Set("Accept", AcceptResolucionRRHH)
	h.ServeHTTP(w, p)
	if w.Code != http.StatusNotFound || w.Header().Get("X-Audit-Ref") != "" {
		t.Fatalf("sin anotación: estado=%d", w.Code)
	}
}

func TestConsultaDetalleJSONNoUsaElRegistroDeDescargas(t *testing.T) {
	c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba(), err: application.ErrConsultaRRHHNoObservable}
	reg := &registradorDescargaPrueba{}
	h := manejadorDescargaPrueba(t, c, &renderizadorBorradorRRHHPrueba{}, reg)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, `{"expediente_ref":"expediente:ct:0001","version_observada":7}`))
	if w.Code != http.StatusNotFound || len(reg.fallos) != 0 || reg.descargas != 0 {
		t.Fatalf("la consulta JSON pasó por la descarga: estado=%d", w.Code)
	}
	if _, err := ConfigurarDescargaAuditadaConsultaDetalleRRHH(http.NotFoundHandler(), reg); err == nil {
		t.Fatal("manejador ajeno admitido")
	}
}

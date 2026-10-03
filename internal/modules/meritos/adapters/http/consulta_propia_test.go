package meritoshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type proveedorConsultaPrueba struct {
	solicitud application.SolicitudConsultaPropia
	err       error
	llamadas  int
	ctx       context.Context
	despues   func()
}

func (p *proveedorConsultaPrueba) SolicitudConsultaPropia(ctx context.Context) (application.SolicitudConsultaPropia, error) {
	p.llamadas++
	p.ctx = ctx
	if p.despues != nil {
		p.despues()
	}
	return p.solicitud, p.err
}

type lectorConsultaPrueba struct {
	resultado         ports.ResultadoConsultaPropia
	err               error
	llamadas          int
	solicitud         application.SolicitudConsultaPropia
	despues           func()
	intentos          int
	errAuditoria      error
	snapshotAuditoria application.SolicitudConsultaPropia
}

func (l *lectorConsultaPrueba) ConsultarActual(_ context.Context, s application.SolicitudConsultaPropia) (ports.ResultadoConsultaPropia, error) {
	l.llamadas++
	l.solicitud = s
	if l.despues != nil {
		l.despues()
	}
	return l.resultado, l.err
}

func (l *lectorConsultaPrueba) RegistrarFalloConsulta(_ context.Context, snapshot application.SolicitudConsultaPropia, causa error) error {
	l.intentos++
	l.snapshotAuditoria = snapshot
	if l.errAuditoria != nil {
		return l.errAuditoria
	}
	return causa
}

// Los dobles verifican exclusivamente traducción HTTP. No conceden acceso,
// no consumen V3 y no se usan en composición fuera de estas pruebas.
func consultaEscenarioHTTP(t *testing.T) (*proveedorConsultaPrueba, *lectorConsultaPrueba) {
	t.Helper()
	instante := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	persona, perfil := "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl"
	c, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: persona, PerfilRef: perfil,
		Accion: application.AccionConsultaPropia, Finalidad: application.FinalidadConsultaPropia,
		Campos: []string{"hecho_actual", "recibo_consulta"}, Obligaciones: []string{"auditar"}, DecisionRef: "decision:http:consulta",
		Recurso: vec.RecursoAutorizable{Referencia: "hecho:http:prueba", ModuloID: "meritos", Tipo: "hecho", Ambitos: map[string]string{"persona_ref": persona}}})
	if err != nil {
		t.Fatal(err)
	}
	d, err := c.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ctx, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, persona, perfil, vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	p := &proveedorConsultaPrueba{solicitud: application.SolicitudConsultaPropia{Vinculo: vinculo, Contexto: ctx, Correlacion: d.Correlacion, Motivo: d.ReferenciaMotivo, HechoRef: "hecho:proveedor"}}
	correlacion, _ := d.Correlacion.ValorCanonico()
	l := &lectorConsultaPrueba{resultado: ports.ResultadoConsultaPropia{Codigo: "obtenida",
		HechoActual: &ports.FichaHechoPropio{Referencia: "hecho:http:prueba", Version: 4, Tipo: "titulacion", ConceptoRef: "titulo:prueba", Denominacion: "Título de prueba",
			Procedencia: domain.Procedencia{FuenteRef: "fuente:prueba", Version: "1", HechoOrigenRef: "origen:prueba", CapturadaEn: instante.Format(time.RFC3339)},
			Vigencia:    domain.Vigencia{Desde: "2026-06-01"}, Estado: domain.Pendiente, Evidencias: []vec.ReferenciaDocumento{}},
		ReciboConsulta: &ports.ReciboConsultaPropia{Referencia: "recibo:http:prueba", HechoRef: "hecho:http:prueba", VersionConsultada: 4,
			DecisionRef: "decision:http:consulta", ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:http:consulta", CorrelacionRef: correlacion, ConsultadaEn: instante}}}
	return p, l
}

func consultaPeticionHTTP(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaConsultaPropia, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestConsultaHTTPSelectorMinimoYContextoConfiable(t *testing.T) {
	for _, codigo := range []string{"obtenida", "no_encontrada"} {
		t.Run(codigo, func(t *testing.T) {
			p, l := consultaEscenarioHTTP(t)
			if codigo == "no_encontrada" {
				l.resultado.Codigo, l.resultado.HechoActual, l.resultado.ReciboConsulta.VersionConsultada = codigo, nil, 0
			}
			r := consultaPeticionHTTP(`{"hecho_ref":"hecho:http:prueba"}`)
			r.Header.Set("X-Actor-Ref", "persona:ajena")
			r.Header.Set("Authorization", "valor_cliente_ignorado")
			r.Header.Set("Content-Type", "application/json; charset=utf-8")
			w := httptest.NewRecorder()
			NuevaConsultaPropia(p, l).ServeHTTP(w, r)
			var out ports.ResultadoConsultaPropia
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &out) != nil || !reflect.DeepEqual(out, l.resultado) {
				t.Fatal("respuesta directa inválida", w.Code, w.Body.String())
			}
			esperada := p.solicitud
			esperada.HechoRef = "hecho:http:prueba"
			if p.llamadas != 1 || l.llamadas != 1 || p.ctx != r.Context() || !reflect.DeepEqual(l.solicitud, esperada) {
				t.Fatal("se alteró el contexto confiable o el selector")
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Location") != "" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatal("respuesta conserva datos o cambia transporte")
			}
			for _, secreto := range []string{"persona_ref", "actor_ref", "declarante_ref", "valor_cliente_ignorado", "persona:ajena"} {
				if strings.Contains(w.Body.String(), secreto) {
					t.Fatal("respuesta expone identidad", secreto)
				}
			}
		})
	}
}

func TestConsultaHTTPRechazaCuerpoAjenoConIntentoNominal(t *testing.T) {
	for _, body := range []string{
		`{"hecho_ref":"hecho:http:prueba","persona_ref":"persona:ajena"}`,
		`{"actor_ref":"persona:ajena","hecho_ref":"hecho:http:prueba"}`,
		`{"hecho_ref":"hecho:http:prueba","contexto":{}}`,
		`{"hecho_ref":"hecho:http:prueba","version_esperada":4}`,
		`{"hecho_ref":"hecho:http:prueba","hecho_ref":"hecho:otro"}`,
		`{"HECHO_REF":"hecho:http:prueba"}`, `{"hecho_ref":null}`, `{"hecho_ref":4}`, `{"hecho_ref":""}`,
		`{"hecho_ref":"hecho:http:prueba"} {}`, `{}`, `[]`, `null`, ``, strings.Repeat(" ", 4097),
	} {
		t.Run(body[:min(len(body), 60)], func(t *testing.T) {
			p, l := consultaEscenarioHTTP(t)
			w := httptest.NewRecorder()
			NuevaConsultaPropia(p, l).ServeHTTP(w, consultaPeticionHTTP(body))
			if w.Code != http.StatusBadRequest || p.llamadas != 1 || l.llamadas != 0 || l.intentos != 1 ||
				!reflect.DeepEqual(l.snapshotAuditoria, p.solicitud) || strings.Contains(w.Body.String(), "hecho:http:prueba") {
				t.Fatal("entrada no mínima sin intento nominal o alcanza el lector", w.Code)
			}
		})
	}
}

func TestConsultaHTTPMetodoRutaYTipo(t *testing.T) {
	for _, caso := range []string{"get", "ruta", "query", "codificada", "sin_tipo", "tipo", "charset", "parametro", "encoding", "limite_stream"} {
		t.Run(caso, func(t *testing.T) {
			p, l := consultaEscenarioHTTP(t)
			r := consultaPeticionHTTP(`{"hecho_ref":"hecho:http:prueba"}`)
			estado := http.StatusBadRequest
			switch caso {
			case "get":
				r.Method, estado = http.MethodGet, http.StatusMethodNotAllowed
			case "ruta":
				r.URL.Path, estado = "/api/meritos/otra", http.StatusNotFound
			case "query":
				r.URL.RawQuery = "persona_ref=ajena"
			case "codificada":
				r.URL.RawPath, estado = "/api/meritos/hecho-propio/%63onsulta", http.StatusNotFound
			case "sin_tipo":
				r.Header.Del("Content-Type")
			case "tipo":
				r.Header.Set("Content-Type", "text/plain")
			case "charset":
				r.Header.Set("Content-Type", "application/json; charset=iso-8859-1")
			case "parametro":
				r.Header.Set("Content-Type", "application/json; actor=persona")
			case "encoding":
				r.Header.Set("Content-Encoding", "gzip")
			case "limite_stream":
				r = consultaPeticionHTTP(strings.Repeat(" ", 4097))
				r.ContentLength = -1
			}
			w := httptest.NewRecorder()
			NuevaConsultaPropia(p, l).ServeHTTP(w, r)
			router := estado == http.StatusMethodNotAllowed || estado == http.StatusNotFound
			expected := 1
			if router {
				expected = 0
			}
			if w.Code != estado || p.llamadas != expected || l.llamadas != 0 || l.intentos != expected {
				t.Fatal("ruta o formato inesperados alcanzan servicio", w.Code)
			}
			if caso == "get" && w.Header().Get("Allow") != http.MethodPost {
				t.Fatal("método admitido ausente")
			}
		})
	}
}

func TestConsultaHTTPDependenciasYErroresCierranSinDatos(t *testing.T) {
	for _, caso := range []string{"proveedor_nil", "proveedor_typednil", "lector_nil", "lector_typednil", "handler_nil", "proveedor_error", "proveedor_denegada", "lector_error", "lector_denegada", "denegada_con_datos", "resultado_invalido", "cancelada_inicial", "cancelada_proveedor", "cancelada_lector"} {
		t.Run(caso, func(t *testing.T) {
			p, l := consultaEscenarioHTTP(t)
			h := NuevaConsultaPropia(p, l)
			r := consultaPeticionHTTP(`{"hecho_ref":"hecho:http:prueba"}`)
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			r = r.WithContext(ctx)
			estado := http.StatusServiceUnavailable
			switch caso {
			case "proveedor_nil":
				h.proveedor = nil
			case "proveedor_typednil":
				h.proveedor = (*proveedorConsultaPrueba)(nil)
			case "lector_nil":
				h.lector = nil
			case "lector_typednil":
				h.lector = (*lectorConsultaPrueba)(nil)
			case "handler_nil":
				h = nil
			case "proveedor_error":
				p.err = errors.New("dato_privado")
			case "proveedor_denegada":
				p.err, estado = errors.Join(errors.New("dato_privado"), vec.ErrAutorizacionDenegada), http.StatusForbidden
			case "lector_error":
				l.err = errors.New("dato_privado")
			case "lector_denegada":
				l.err, estado = errors.Join(errors.New("dato_privado"), vec.ErrAutorizacionDenegada), http.StatusForbidden
			case "denegada_con_datos":
				l.resultado.Codigo, estado = "denegada", http.StatusForbidden
			case "resultado_invalido":
				l.resultado.ReciboConsulta = nil
			case "cancelada_inicial":
				cancel()
			case "cancelada_proveedor":
				p.despues = cancel
			case "cancelada_lector":
				l.despues = cancel
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != estado {
				t.Fatal("error se anunció como éxito o estado inesperado", w.Code)
			}
			for _, dato := range []string{"dato_privado", "hecho_actual", "recibo_consulta", "Título de prueba", "hecho:http:prueba"} {
				if strings.Contains(w.Body.String(), dato) {
					t.Fatal("error expone datos", dato)
				}
			}
			if strings.HasPrefix(caso, "proveedor_") || caso == "cancelada_inicial" || caso == "cancelada_proveedor" {
				if l.llamadas != 0 {
					t.Fatal("lector ejecutado sin proveedor confirmado")
				}
			}
		})
	}
}

func TestConsultaHTTPAuditoriaFalloEntradaNoDisponible(t *testing.T) {
	p, l := consultaEscenarioHTTP(t)
	l.errAuditoria = ports.ErrConsultaNoDisponible
	w := httptest.NewRecorder()
	NuevaConsultaPropia(p, l).ServeHTTP(w, consultaPeticionHTTP(`{"persona_ref":"no_aceptada"}`))
	if w.Code != http.StatusServiceUnavailable || l.intentos != 1 || l.llamadas != 0 || strings.Contains(w.Body.String(), "no_aceptada") {
		t.Fatal("fallo de auditoría devuelto como rechazo confirmado", w.Code)
	}
}

func TestConsultaHTTPErrorTecnicoCompuestoDa503(t *testing.T) {
	for _, fallo := range []error{ports.ErrConsultaNoDisponible, context.Canceled, context.DeadlineExceeded, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible} {
		p, l := consultaEscenarioHTTP(t)
		l.err = errors.Join(vec.ErrAutorizacionDenegada, fallo)
		w := httptest.NewRecorder()
		NuevaConsultaPropia(p, l).ServeHTTP(w, consultaPeticionHTTP(`{"hecho_ref":"hecho:http:prueba"}`))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatal("error técnico anunciado como403", w.Code)
		}
	}
}

func TestConsultaHTTPErrorPosLecturaDejaIntentoNominal(t *testing.T) {
	for _, caso := range []string{"cancelada", "resultado", "limite"} {
		t.Run(caso, func(t *testing.T) {
			p, l := consultaEscenarioHTTP(t)
			r := consultaPeticionHTTP(`{"hecho_ref":"hecho:http:prueba"}`)
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			r = r.WithContext(ctx)
			switch caso {
			case "cancelada":
				l.despues = cancel
			case "resultado":
				l.resultado.ReciboConsulta = nil
			case "limite":
				l.resultado.HechoActual.Denominacion = strings.Repeat("a", 65536)
			}
			w := httptest.NewRecorder()
			NuevaConsultaPropia(p, l).ServeHTTP(w, r)
			if w.Code != http.StatusServiceUnavailable || l.intentos != 1 || l.llamadas != 1 ||
				l.snapshotAuditoria.HechoRef != "hecho:http:prueba" ||
				l.snapshotAuditoria.Vinculo.ValidarPara(p.solicitud.Contexto) != nil {
				t.Fatal("error después de consultar sin intento nominal", w.Code)
			}
		})
	}
}

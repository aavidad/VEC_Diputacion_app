package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type lectorDocumentosRRHHPrueba struct {
	lecturas int
}

func (l *lectorDocumentosRRHHPrueba) ListarVigentesBolsa(_ context.Context, bolsa string) ([]ports.ConstitucionVigente, error) {
	l.lecturas++
	if bolsa != "bolsa:01" {
		return nil, nil
	}
	v := ports.ConstitucionVigente{ActaRef: "acta:importacion-convoca:01", ConfirmadaEn: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), TotalParticipaciones: 8}
	v.Bolsa.BolsaRef, v.Bolsa.Version = bolsa, 2
	v.Instantanea.InstantaneaRef, v.Instantanea.Version = "instantanea:01", 3
	v.Instantanea.ListadoDefinitivoRef, v.Instantanea.VersionListado = "listado:convoca:01", 1
	v.Instantanea.HuellaListadoSHA256 = strings.Repeat("a", 64)
	return []ports.ConstitucionVigente{v}, nil
}

func TestDocumentosBolsaRRHHRegistroYReciboAutorizados(t *testing.T) {
	lector := &lectorDocumentosRRHHPrueba{}
	servicio, err := bolsaapp.NuevoServicioDocumentosBolsa(lector)
	if err != nil {
		t.Fatal(err)
	}
	autorizadas, auditadas := 0, 0
	h := &bolsasRRHHDesarrollo{documentos: servicio, controlDocumentos: &controlLecturaDocumentosBolsa{
		Autorizar: func(_ context.Context, ref, _ string) error {
			autorizadas++
			if ref != "bolsa:01" {
				return errors.New("ajena")
			}
			return nil
		},
		Auditar: func(_ context.Context, ref, _ string, resultado error) error {
			auditadas++
			if ref != "bolsa:01" || resultado != nil {
				return errors.New("auditoria")
			}
			return nil
		},
	}}
	lista := httptest.NewRecorder()
	h.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:01/documentos?tipo=constitucion&texto=convoca&limite=1", nil))
	if lista.Code != http.StatusOK || autorizadas != 1 || auditadas != 1 || lector.lecturas != 1 {
		t.Fatalf("lista status=%d auth=%d audit=%d lecturas=%d body=%s", lista.Code, autorizadas, auditadas, lector.lecturas, lista.Body.String())
	}
	var respuesta struct {
		Data struct {
			Documentos []struct {
				DocumentoRef string `json:"documento_ref"`
				Accion       struct {
					Href string `json:"href"`
				} `json:"accion"`
			}
			Paginacion struct {
				Total int `json:"total"`
			} `json:"paginacion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lista.Body.Bytes(), &respuesta); err != nil || len(respuesta.Data.Documentos) != 1 || respuesta.Data.Paginacion.Total != 1 || respuesta.Data.Documentos[0].DocumentoRef != "acta:importacion-convoca:01" {
		t.Fatalf("contrato lista: %+v error=%v", respuesta, err)
	}
	recibo := httptest.NewRecorder()
	h.ServeHTTP(recibo, httptest.NewRequest(http.MethodGet, respuesta.Data.Documentos[0].Accion.Href, nil))
	if recibo.Code != http.StatusOK || !strings.Contains(recibo.Header().Get("Content-Disposition"), "attachment;") ||
		!strings.Contains(recibo.Body.String(), `"version_instantanea":3`) || !strings.Contains(recibo.Body.String(), `"estado_firma":"sin_firma_acreditada"`) || autorizadas != 2 || auditadas != 2 {
		t.Fatalf("recibo status=%d auth=%d audit=%d body=%s", recibo.Code, autorizadas, auditadas, recibo.Body.String())
	}
	// Un filtro inválido conserva el intento nominal y no consulta la fuente.
	invalida := httptest.NewRecorder()
	h.controlDocumentos.Auditar = func(_ context.Context, _ string, _ string, resultado error) error {
		auditadas++
		if !errors.Is(resultado, bolsaapp.ErrConsultaDocumentosBolsaInvalida) {
			return errors.New("causa de auditoria distinta")
		}
		return nil
	}
	h.ServeHTTP(invalida, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:01/documentos?tipo=inventado", nil))
	if invalida.Code != http.StatusBadRequest || autorizadas != 3 || auditadas != 3 || lector.lecturas != 2 {
		t.Fatalf("filtro inválido status=%d auth=%d audit=%d lecturas=%d", invalida.Code, autorizadas, auditadas, lector.lecturas)
	}
}

func TestDocumentosBolsaRRHHFallaCerradoAntesDeEntregar(t *testing.T) {
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:01/documentos", nil)) ||
		!rutaContextoAutorizacionContratacionTemporalDesarrollo(rutaBolsasRRHHDesarrollo+"/bolsa:01/documentos") {
		t.Fatal("documentos Bolsa fuera del guardián nominal de RRHH")
	}
	lector := &lectorDocumentosRRHHPrueba{}
	servicio, _ := bolsaapp.NuevoServicioDocumentosBolsa(lector)
	ruta := rutaBolsasRRHHDesarrollo + "/bolsa:01/documentos"
	h := &bolsasRRHHDesarrollo{documentos: servicio}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, ruta, nil))
	if res.Code != http.StatusServiceUnavailable || lector.lecturas != 0 {
		t.Fatalf("control ausente status=%d lecturas=%d", res.Code, lector.lecturas)
	}
	h.controlDocumentos = &controlLecturaDocumentosBolsa{Autorizar: func(context.Context, string, string) error {
		return errors.New("denegada")
	}, Auditar: func(context.Context, string, string, error) error { return nil }}
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, ruta, nil))
	if res.Code != http.StatusForbidden || lector.lecturas != 0 {
		t.Fatalf("denegada status=%d lecturas=%d", res.Code, lector.lecturas)
	}
	h.controlDocumentos = &controlLecturaDocumentosBolsa{Autorizar: func(context.Context, string, string) error {
		return nil
	}, Auditar: func(context.Context, string, string, error) error {
		return errors.New("caida")
	}}
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, ruta, nil))
	if res.Code != http.StatusServiceUnavailable || strings.Contains(res.Body.String(), "acta:importacion") {
		t.Fatalf("sin acuse status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestDocumentosBolsaRRHHSeleccionaSoporteLectorNominal(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	base := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: "rrhh_base", certificadoSHA256: strings.Repeat("a", 64)}
	lector := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: "rrhh_lector", certificadoSHA256: strings.Repeat("b", 64), lectorConsultasRRHH: true, tecnicoConsultaRRHH: true}
	lectorSinBolsa := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: "rrhh_consulta", certificadoSHA256: strings.Repeat("d", 64), lectorConsultasRRHH: true}
	resolver := resolverSoporteDocumentosBolsa(base, map[string]*soporteAltaContratacionTemporalDesarrollo{lector.principalID: lector, lectorSinBolsa.principalID: lectorSinBolsa})
	contexto := func(principal vecdomain.Principal, metodo, ruta string) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{
			sello: sello, ruta: ruta, metodo: metodo, principal: principal,
		})
	}
	ruta := rutaBolsasRRHHDesarrollo + "/bolsa:01/documentos"
	for _, caso := range []struct {
		soporte *soporteAltaContratacionTemporalDesarrollo
	}{{base}, {lector}} {
		seleccionado, err := resolver(contexto(principalDeSoportePrueba(caso.soporte), http.MethodGet, ruta))
		if err != nil || seleccionado != caso.soporte {
			t.Fatalf("técnico nominal %s: soporte=%p error=%v", caso.soporte.principalID, seleccionado, err)
		}
	}
	principal := principalDeSoportePrueba(lector)
	principal.Attributes["certificate_sha256"] = strings.Repeat("c", 64)
	if seleccionado, err := resolver(contexto(principal, http.MethodGet, ruta)); err == nil || seleccionado != nil {
		t.Fatalf("certificado ajeno seleccionó soporte: soporte=%p error=%v", seleccionado, err)
	}
	principal = principalDeSoportePrueba(lectorSinBolsa)
	principal.Roles = []string{"lector_rrhh"}
	if seleccionado, err := resolver(contexto(principal, http.MethodGet, ruta)); err == nil || seleccionado != nil {
		t.Fatalf("lector sin acceso Bolsa seleccionó soporte: soporte=%p error=%v", seleccionado, err)
	}
	principal = principalDeSoportePrueba(lector)
	for _, invalido := range []struct{ metodo, ruta string }{
		{http.MethodPost, ruta}, {http.MethodGet, rutaBolsasRRHHDesarrollo + "/bolsa:01/candidatos"},
	} {
		if seleccionado, err := resolver(contexto(principal, invalido.metodo, invalido.ruta)); err == nil || seleccionado != nil {
			t.Fatalf("ruta/método ajeno seleccionó soporte: soporte=%p error=%v", seleccionado, err)
		}
	}
}

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type lectorFichaPrueba struct {
	err      error
	llamadas int
	recibida ports.SolicitudConsultaConvocatoria
	cruzada  bool
}

func (l *lectorFichaPrueba) ConsultarExacta(_ context.Context, s ports.SolicitudConsultaConvocatoria) (ports.LecturaConvocatoria, error) {
	l.llamadas++
	l.recibida = s
	h := strings.Repeat("a", 64)
	correlacion, _ := s.Correlacion.ValorCanonico()
	r := ports.LecturaConvocatoria{
		Ficha: ports.FichaConvocatoria{ConvocatoriaID: s.Selector.ID, Secuencia: s.Selector.Secuencia, Revision: 1, FuenteRef: s.Selector.Referencia(), HuellaVersionSHA256: h,
			Bases:            []bolsadomain.ReferenciaDocumentoOficialConvocatoria{{Rol: "bases", PublicacionRef: "publicacion:bases", DocumentoRef: "documento:bases", VersionDocumento: 1, RepresentacionRef: "representacion:bases", HuellaContenidoSHA256: h, FirmaValidadaRef: "firma:bases", ReciboCustodiaRef: "custodia:bases"}},
			FlujoProceso:     bolsadomain.ReferenciaConfiguracionConvocatoria{ID: "flujo-prueba", Version: 1, HuellaContenidoSHA256: h},
			ReglasBaremacion: bolsadomain.ReferenciaConfiguracionConvocatoria{ID: "baremo:prueba", Version: 1, HuellaContenidoSHA256: h},
			FasesEstado:      "pendiente_fuente", ReferenciasCoberturaEstado: "pendiente_fuente"},
		Evidencia: ports.EvidenciaLecturaConvocatoria{ReciboRef: "recibo:prueba", DecisionRef: "decision:prueba", ConsumoHuellaSHA256: h, AuditoriaRef: "auditoria:prueba", CorrelacionRef: correlacion, ConsultadaEn: s.Actor.ResueltoEn},
	}
	if l.cruzada {
		r.Ficha.Secuencia++
	}
	return r, l.err
}

type generadorCorrelacionFicha struct{}

func (generadorCorrelacionFicha) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_" + strings.Repeat("a", 32), nil
}

func contextoPrueba(t *testing.T) ports.SolicitudConsultaConvocatoria {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionFicha{})
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudConsultaConvocatoria{Actor: actor, Correlacion: correlacion, Selector: bolsaports.SelectorVersionConvocatoriaExacta{ID: "convocatoria:servidor", Secuencia: 999}}
}

func handlerPrueba(t *testing.T, l *lectorFichaPrueba) http.Handler {
	t.Helper()
	h, err := NuevaFichaHandler(ConfigFicha{Lector: l, ResolverContexto: func(*http.Request) (ports.SolicitudConsultaConvocatoria, error) { return contextoPrueba(t), nil }, ValidarFrontera: func(*http.Request) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func peticion(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaFichaConvocatoria, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

const cuerpoValido = `{"convocatoria_id":"convocatoria:prueba","secuencia":2}`

func TestFichaTransportaVersionExactaEvidenciaYListaVacia(t *testing.T) {
	l := &lectorFichaPrueba{}
	w := httptest.NewRecorder()
	handlerPrueba(t, l).ServeHTTP(w, peticion(cuerpoValido))
	if w.Code != 200 || l.llamadas != 1 || l.recibida.Selector.ID != "convocatoria:prueba" || l.recibida.Selector.Secuencia != 2 {
		t.Fatalf("consulta: %d %s", w.Code, w.Body)
	}
	var r fichaJSON
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Evidencia.ReciboRef != "recibo:prueba" || r.Evidencia.AuditoriaRef != "auditoria:prueba" || r.Ficha.FuenteRef != "convocatoria:prueba#2" || r.Ficha.FasesEstado != "pendiente_fuente" {
		t.Fatal("se perdió evidencia o se dedujo fase")
	}
	if !strings.Contains(w.Body.String(), `"requisitos":[]`) || strings.Contains(w.Body.String(), "Actor") {
		t.Fatal("serialización no explícita")
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("persistencia de transporte")
	}
}

func TestFichaRechazaCamposExtraDuplicadosYTiposAntesDelLector(t *testing.T) {
	for _, c := range []string{`{"convocatoria_id":"convocatoria:prueba","secuencia":2,"actor":"intruso"}`, `{"convocatoria_id":"convocatoria:prueba","secuencia":2,"secuencia":3}`, `{"convocatoria_id":"convocatoria:prueba","secuencia":"2"}`, `{"convocatoria_id":"convocatoria:prueba","secuencia":0}`, cuerpoValido + `{}`, `null`, strings.Repeat("x", 5000)} {
		l := &lectorFichaPrueba{}
		w := httptest.NewRecorder()
		handlerPrueba(t, l).ServeHTTP(w, peticion(c))
		if w.Code != 400 || l.llamadas != 0 {
			t.Fatalf("entrada ambigua: %d", w.Code)
		}
	}
}

func TestFichaErroresSinResultadoParcialNiDetallesPrivados(t *testing.T) {
	for _, caso := range []struct {
		err     error
		estado  int
		cruzada bool
	}{{ports.ErrConsultaConvocatoriaDenegada, 403, false}, {ports.ErrConvocatoriaNoEncontrada, 404, false}, {ports.ErrConvocatoriaNoDisponible, 503, false}, {errors.New("ruta y secreto privado"), 503, false}, {nil, 503, true}} {
		l := &lectorFichaPrueba{err: caso.err, cruzada: caso.cruzada}
		w := httptest.NewRecorder()
		handlerPrueba(t, l).ServeHTTP(w, peticion(cuerpoValido))
		if w.Code != caso.estado || strings.Contains(w.Body.String(), "recibo:") || strings.Contains(w.Body.String(), "privado") || strings.Contains(w.Body.String(), "ficha") {
			t.Fatalf("dato parcial: %d %s", w.Code, w.Body)
		}
	}
}

func TestFichaExigeFronteraYContextoDelServidor(t *testing.T) {
	l := &lectorFichaPrueba{}
	for _, c := range []ConfigFicha{{Lector: l}, {Lector: l, ValidarFrontera: func(*http.Request) error { return nil }}, {ResolverContexto: func(*http.Request) (ports.SolicitudConsultaConvocatoria, error) { return contextoPrueba(t), nil }, ValidarFrontera: func(*http.Request) error { return nil }}} {
		if _, err := NuevaFichaHandler(c); err == nil {
			t.Fatal("aceptó autoridad ausente")
		}
	}
	for _, frontera := range []bool{true, false} {
		resuelto := false
		c := ConfigFicha{Lector: l, ResolverContexto: func(*http.Request) (ports.SolicitudConsultaConvocatoria, error) {
			resuelto = true
			return ports.SolicitudConsultaConvocatoria{}, errors.New("rechazo")
		}, ValidarFrontera: func(*http.Request) error {
			if frontera {
				return errors.New("rechazo")
			}
			return nil
		}}
		h, err := NuevaFichaHandler(c)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion(cuerpoValido))
		if w.Code != 403 || l.llamadas != 0 || resuelto == frontera {
			t.Fatal("cruzó frontera rechazada")
		}
	}
}

func TestFichaNoAdmiteRutasAlternativasNiContenidoNoJSON(t *testing.T) {
	for _, c := range []struct {
		ruta, metodo, tipo string
		estado             int
	}{{RutaFichaConvocatoria + "?actor=intruso", "POST", "application/json", 404}, {RutaFichaConvocatoria, "GET", "application/json", 405}, {RutaFichaConvocatoria, "POST", "text/plain", 415}, {RutaFichaConvocatoria, "POST", "application/json; evil=1", 415}} {
		l := &lectorFichaPrueba{}
		r := httptest.NewRequest(c.metodo, c.ruta, strings.NewReader(cuerpoValido))
		r.Header.Set("Content-Type", c.tipo)
		w := httptest.NewRecorder()
		handlerPrueba(t, l).ServeHTTP(w, r)
		if w.Code != c.estado || l.llamadas != 0 {
			t.Fatalf("transporte alternativo: %d", w.Code)
		}
	}
}

type escritorFichaFallo struct{ *httptest.ResponseRecorder }

func (escritorFichaFallo) Write([]byte) (int, error) {
	return 0, errors.New("detalle_privado_escritor")
}

func TestFichaRegistraFalloDeEscrituraSinDatosNiErrorPrivado(t *testing.T) {
	var salida bytes.Buffer
	anterior := log.Writer()
	log.SetOutput(&salida)
	defer log.SetOutput(anterior)
	for _, causa := range []error{nil, ports.ErrConvocatoriaNoDisponible} {
		salida.Reset()
		l := &lectorFichaPrueba{err: causa}
		w := escritorFichaFallo{httptest.NewRecorder()}
		handlerPrueba(t, l).ServeHTTP(w, peticion(cuerpoValido))
		if !strings.Contains(salida.String(), "seleccion_ficha_respuesta_no_entregada") || strings.Contains(salida.String(), "privado") || strings.Contains(salida.String(), "recibo:") || strings.Contains(salida.String(), "convocatoria:prueba") {
			t.Fatal("diagnóstico perdido o datos privados en el registro")
		}
	}
}

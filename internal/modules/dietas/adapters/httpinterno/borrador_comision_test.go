package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type autoridadBorradorPrueba struct {
	actor    vecdomain.ContextoActor
	err      error
	llamadas int
}

func (a *autoridadBorradorPrueba) ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error) {
	a.llamadas++
	return a.actor, a.err
}

type politicaBorradorPrueba struct {
	politica domain.PoliticaKilometraje
	err      error
}

func (p politicaBorradorPrueba) ResolverPoliticaKilometraje(context.Context, ports.SolicitudPoliticaKilometraje) (domain.PoliticaKilometraje, error) {
	return p.politica, p.err
}

type unidadBorradorHTTPPrueba struct {
	crear      ports.SolicitudCrearBorradorPropio
	recuperado domain.BorradorComision
	recibo     ports.ReciboBorradorComision
	pagina     ports.PaginaBorradoresPropios
	err        error
	llamadas   int
}

func (u *unidadBorradorHTTPPrueba) CrearBorradorPropio(_ context.Context, x ports.SolicitudCrearBorradorPropio) (ports.ReciboBorradorComision, error) {
	u.llamadas++
	u.crear = x
	return u.recibo, u.err
}
func (u *unidadBorradorHTTPPrueba) RecuperarBorradorPropio(context.Context, vecdomain.ContextoActor, string) (domain.BorradorComision, ports.ReciboBorradorComision, error) {
	u.llamadas++
	return u.recuperado, u.recibo, u.err
}
func (u *unidadBorradorHTTPPrueba) ListarBorradoresPropios(context.Context, vecdomain.ContextoActor, ports.ConsultaBorradoresPropios) (ports.PaginaBorradoresPropios, error) {
	u.llamadas++
	return u.pagina, u.err
}

func manejadorPrueba(t *testing.T, u *unidadBorradorHTTPPrueba, p politicaBorradorPrueba) (*ManejadorBorradorComision, *autoridadBorradorPrueba) {
	t.Helper()
	z, e := time.LoadLocation("Europe/Madrid")
	if e != nil {
		t.Fatal(e)
	}
	s, e := application.NuevoServicioBorradorComision(u, p, z)
	if e != nil {
		t.Fatal(e)
	}
	a := &autoridadBorradorPrueba{actor: actorHTTPBorrador(t)}
	m, e := NuevoManejadorBorradorComision(s, a)
	if e != nil {
		t.Fatal(e)
	}
	return m, a
}
func politicaPrueba() politicaBorradorPrueba {
	return politicaBorradorPrueba{politica: domain.PoliticaKilometraje{Referencia: "pol_km_001", Version: "v1", TarifaEURPorKM: "0.2500"}}
}
func reciboHTTPBorrador() ports.ReciboBorradorComision {
	return ports.ReciboBorradorComision{ComisionRef: "dietas:borrador:abcdefghijklmnop", ReciboRef: "rec_001", CorrelacionRef: "cor_001", Version: 1, RegistradoEn: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)}
}
func entradaJSON() string {
	return `{"operation_key":"abcdefghijklmnop","expected_version":0,"draft":{"politica_kilometraje_ref":"pol_km_001","politica_kilometraje_version":"v1","objeto":"Visita","fecha":"2026-01-02","fecha_fin":"2026-01-02","hora_inicio":"09:00","hora_fin":"10:00","vehiculo_propio":true,"ruta":{"fuente":"osrm_interno","version":"grafo-1","referencia":"ruta-1","catalogo_version":"catalogo-1","alternativa_ref":"alt-1","recomendada":true,"motivo_alternativa":"","kilometros":"10.0000","paradas":[{"codigo":"punto-1","nombre":"Origen","latitud":37.1,"longitud":-3.6},{"codigo":"punto-2","nombre":"Destino","latitud":37.2,"longitud":-3.5}],"tramos":[{"origen_codigo":"punto-1","destino_codigo":"punto-2","kilometros":"10.0000","duracion_minutos":12,"ajuste_kilometros":"0","motivo_ajuste":""}],"trazado":[[37.1,-3.6],[37.2,-3.5]],"liquidable":false},"gastos":{"manutencion_eur":"10.00","alojamiento_eur":"0.00","otros_eur":"1.00"}}}`
}
func borradorHTTPBorrador(t *testing.T, persona string) domain.BorradorComision {
	t.Helper()
	var x solicitudCrearBorradorJSON
	if e := jsonDecodifica(entradaJSON(), &x); e != nil {
		t.Fatal(e)
	}
	b, ok := aDominioEntrada(x.Borrador, persona)
	if !ok {
		t.Fatal("ruta no convertible")
	}
	z, _ := time.LoadLocation("Europe/Madrid")
	b, e := domain.PrepararBorrador(b, politicaPrueba().politica, z)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func jsonDecodifica(s string, d any) error { return json.Unmarshal([]byte(s), d) }

func TestManejadorBorradorCreaYRecuperaCompleto(t *testing.T) {
	actor := actorHTTPBorrador(t)
	u := &unidadBorradorHTTPPrueba{recibo: reciboHTTPBorrador()}
	u.recuperado = borradorHTTPBorrador(t, actor.PersonaRef)
	m, _ := manejadorPrueba(t, u, politicaPrueba())
	r := httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(entradaJSON()))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "abcdefghijklmnop")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || u.crear.Borrador.PersonaRef != actor.PersonaRef || u.crear.Borrador.Desglose.TotalEUR != "13.50" {
		t.Fatalf("POST no preparo borrador completo: %d %#v", w.Code, u.crear.Borrador)
	}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMisComisiones+"/dietas:borrador:abcdefghijklmnop", nil))
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "persona_ref") || !strings.Contains(w.Body.String(), `"total_eur":"13.50"`) {
		t.Fatalf("GET no proyecto borrador seguro: %d %s", w.Code, w.Body.String())
	}
}

type respuestaInterrumpida struct {
	cabeceras           http.Header
	estados, escrituras int
	err                 error
}

func (w *respuestaInterrumpida) Header() http.Header { return w.cabeceras }
func (w *respuestaInterrumpida) WriteHeader(int)     { w.estados++ }
func (w *respuestaInterrumpida) Write(p []byte) (int, error) {
	w.escrituras++
	return len(p) / 2, w.err
}

// Los dobles sólo prueban entrega HTTP después del éxito de la unidad durable;
// la atomicidad y la idempotencia se acreditan por separado con PostgreSQL.
func TestManejadorBorradorAbortaEntregaParcialYPermiteRecuperar(t *testing.T) {
	for _, operacion := range []string{"crear", "recuperar", "listar"} {
		for _, fallo := range []string{"error", "corto_sin_error"} {
			t.Run(operacion+"/"+fallo, func(t *testing.T) {
				actor := actorHTTPBorrador(t)
				recibo := reciboHTTPBorrador()
				b := borradorHTTPBorrador(t, actor.PersonaRef)
				u := &unidadBorradorHTTPPrueba{recibo: recibo, recuperado: b,
					pagina: ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{{Borrador: b.Resumen(), Recibo: recibo}}}}
				m, _ := manejadorPrueba(t, u, politicaPrueba())
				r := httptest.NewRequest(http.MethodGet, RutaMisComisiones, nil)
				if operacion == "crear" {
					r = httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(entradaJSON()))
					r.Header.Set("Content-Type", "application/json")
					r.Header.Set("Idempotency-Key", "abcdefghijklmnop")
				} else if operacion == "recuperar" {
					r = httptest.NewRequest(http.MethodGet, RutaMisComisiones+"/"+recibo.ComisionRef, nil)
				}
				w := &respuestaInterrumpida{cabeceras: make(http.Header)}
				if fallo == "error" {
					w.err = errors.New("transporte sintético interrumpido")
				}
				func() {
					defer func() {
						if p := recover(); p != http.ErrAbortHandler {
							t.Fatalf("entrega fallida no abortada: %v", p)
						}
					}()
					m.ServeHTTP(w, r)
				}()
				if w.estados != 1 || w.escrituras != 1 || u.llamadas != 1 {
					t.Fatal("se respondió o ejecutó de nuevo tras perder entrega")
				}
				recuperacion := httptest.NewRecorder()
				m.ServeHTTP(recuperacion, httptest.NewRequest(http.MethodGet, RutaMisComisiones+"/"+recibo.ComisionRef, nil))
				if recuperacion.Code != http.StatusOK || !strings.Contains(recuperacion.Body.String(), `"receipt_ref":"`+recibo.ReciboRef+`"`) {
					t.Fatal("recibo confirmado no recuperable tras fallo de entrega")
				}
				if operacion == "crear" && u.crear.ClaveOperacion != "abcdefghijklmnop" {
					t.Fatal("se alteró la clave original")
				}
			})
		}
	}
}

func TestManejadorBorradorRechazaInyeccionesYCredenciales(t *testing.T) {
	u := &unidadBorradorHTTPPrueba{recibo: reciboHTTPBorrador()}
	m, a := manejadorPrueba(t, u, politicaPrueba())
	for _, c := range []string{strings.Replace(entradaJSON(), `"objeto":"Visita"`, `"persona_ref":"per_inyectada","objeto":"Visita"`, 1), strings.Replace(entradaJSON(), `"gastos":{`, `"total_eur":"999.99","gastos":{`, 1)} {
		r := httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(c))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "abcdefghijklmnop")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || u.llamadas != 0 {
			t.Fatalf("inyeccion admitida: %d", w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodGet, RutaMisComisiones, nil)
	r.Header.Set("Cookie", "falsa=1")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || a.llamadas != 0 {
		t.Fatal("credencial web aceptada")
	}
}
func TestManejadorBorradorListaPaginadaYPoliticaNoDisponible(t *testing.T) {
	actor := actorHTTPBorrador(t)
	b := borradorHTTPBorrador(t, actor.PersonaRef)
	rec := reciboHTTPBorrador()
	u := &unidadBorradorHTTPPrueba{pagina: ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{{Borrador: b.Resumen(), Recibo: rec}}, Siguiente: ""}}
	m, _ := manejadorPrueba(t, u, politicaBorradorPrueba{err: errors.New("caida")})
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMisComisiones+"?limit=1", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatalf("lista/politica no disponible: %d %s", w.Code, w.Body.String())
	}
	for _, q := range []string{"?limit=01", "?limit=21", "?after=x", "?limit=1&limit=2", "?otra=1"} {
		w = httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMisComisiones+q, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("query insegura %q: %d", q, w.Code)
		}
	}
}
func TestManejadorBorradorResultadoIncierto(t *testing.T) {
	u := &unidadBorradorHTTPPrueba{recibo: reciboHTTPBorrador(), err: ports.ErrResultadoBorradorIncierto}
	m, _ := manejadorPrueba(t, u, politicaPrueba())
	r := httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(entradaJSON()))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "abcdefghijklmnop")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "dietas.error.resultado_incierto") {
		t.Fatalf("resultado incierto expuesto mal: %d %s", w.Code, w.Body.String())
	}
}
func TestManejadorBorradorRechazaAutoridadNulaTipada(t *testing.T) {
	u := &unidadBorradorHTTPPrueba{}
	z, _ := time.LoadLocation("Europe/Madrid")
	s, _ := application.NuevoServicioBorradorComision(u, politicaPrueba(), z)
	var a *autoridadBorradorPrueba
	if m, e := NuevoManejadorBorradorComision(s, a); m != nil || e == nil {
		t.Fatal("autoridad nula tipada admitida")
	}
}
func actorHTTPBorrador(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	ahora := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	a, e := vecdomain.NuevoContextoActor(cuenta, vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}, ahora)
	if e != nil {
		t.Fatal(e)
	}
	a.Instantanea.Vinculos = []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_0123456789abcdefghijkl", Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_0123456789abcdefghijkl", Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}
	return a
}

func TestListaLigeraYDetalleConservanRutaYAdvertencia(t *testing.T) {
	a := actorHTTPBorrador(t)
	b := borradorHTTPBorrador(t, a.PersonaRef)
	r := reciboHTTPBorrador()
	u := &unidadBorradorHTTPPrueba{recuperado: b, recibo: r, pagina: ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{{Borrador: b.Resumen(), Recibo: r}}}}
	m, _ := manejadorPrueba(t, u, politicaPrueba())
	for _, detalle := range []bool{false, true} {
		ruta := RutaMisComisiones
		if detalle {
			ruta += "/" + r.ComisionRef
		}
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusOK {
			t.Fatal("lectura falló", w.Code)
		}
		s := w.Body.String()
		if !strings.Contains(s, `"procedencia_ruta":"declarada_no_verificada"`) || !strings.Contains(s, `"revalidacion_ruta_requerida":true`) || !strings.Contains(s, `"ruta_etiquetas":`) {
			t.Fatal("ruta declarativa no identificada")
		}
		for _, campo := range []string{`"trazado":`, `"tramos":`, `"paradas":`} {
			if strings.Contains(s, campo) != detalle {
				t.Fatal("geometría incorrecta para operación", campo)
			}
		}
	}
	for _, campo := range []string{`"procedencia_ruta":"verificada",`, `"revalidacion_ruta_requerida":false,`, `"ruta_etiquetas":["inventada"],`} {
		cuerpo := strings.Replace(entradaJSON(), `"objeto":"Visita"`, campo+`"objeto":"Visita"`, 1)
		req := httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(cuerpo))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "abcdefghijklmnop")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatal("cliente aportó campos de proyección", w.Code)
		}
	}
}

func TestCuerpoBorradorFronteraConRutaDosMilPuntos(t *testing.T) {
	var entrada map[string]any
	if e := json.Unmarshal([]byte(entradaJSON()), &entrada); e != nil {
		t.Fatal(e)
	}
	ruta := entrada["draft"].(map[string]any)["ruta"].(map[string]any)
	trazado := make([][2]float64, 2000)
	for i := range trazado {
		trazado[i] = [2]float64{37.123456789, -3.123456789}
	}
	ruta["trazado"] = trazado
	cuerpo, e := json.Marshal(entrada)
	if e != nil || len(cuerpo) > int(limiteCuerpoBorradorComision) || len(cuerpo) < 50<<10 {
		t.Fatal("no se ejercita cuerpo próximo al límite", len(cuerpo))
	}
	for _, exceso := range []int{0, 1} {
		u := &unidadBorradorHTTPPrueba{recibo: reciboHTTPBorrador()}
		m, _ := manejadorPrueba(t, u, politicaPrueba())
		raw := string(cuerpo) + strings.Repeat(" ", int(limiteCuerpoBorradorComision)-len(cuerpo)+exceso)
		req := httptest.NewRequest(http.MethodPost, RutaMisComisiones, strings.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "abcdefghijklmnop")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)
		if exceso == 0 && (w.Code != http.StatusCreated || u.llamadas != 1 || len(u.crear.Borrador.Ruta.Trazado) != 2000 || !u.crear.Borrador.RevalidacionRutaRequerida) {
			t.Fatal("ruta completa en límite rechazada", w.Code)
		}
		if exceso == 1 && (w.Code != http.StatusBadRequest || u.llamadas != 0) {
			t.Fatal("exceso alcanzó persistencia", w.Code)
		}
	}
}

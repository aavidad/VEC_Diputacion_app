package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func solicitudResolucionPrueba() ports.SolicitudResolucionFormalizacion {
	return ports.SolicitudResolucionFormalizacion{ExpedienteRef: "expediente:prueba", VersionEsperada: 7, PropuestaRef: "propuesta:prueba",
		ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", NumeroResolucion: "EJ-2026/1", FechaResolucion: "2026-09-06",
		Motivo: "Revisión manual del ejercicio.", ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
}
func reciboResolucionPrueba(s ports.SolicitudResolucionFormalizacion) ports.ResultadoResolucionFormalizacion {
	return ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba",
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoSHA256: strings.Repeat("a", 64), DocumentoVersion: 7,
		ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba",
		VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

type autoridadResolucionPrueba struct {
	err      error
	llamadas int
}

func (a *autoridadResolucionPrueba) ResolverContextoResolucionFormalizacion(context.Context) error {
	a.llamadas++
	return a.err
}

type ejecutorResolucionPrueba struct {
	estado   string
	err      error
	llamadas int
}

func (e *ejecutorResolucionPrueba) RegistrarResolucionFormalizacion(_ context.Context, s ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	e.llamadas++
	if e.err != nil {
		return ports.ResultadoResolucionFormalizacion{}, e.err
	}
	r := reciboResolucionPrueba(s)
	r.Estado = e.estado
	return r, nil
}

type lectorResolucionHTTPPrueba struct {
	ejecutorResolucionPrueba
	p          ports.PreparacionResolucionFormalizacion
	lecturas   int
	errLectura error
}

func (e *lectorResolucionHTTPPrueba) ConsultarPreparacionResolucionFormalizacion(_ context.Context, ref string) (ports.PreparacionResolucionFormalizacion, error) {
	e.lecturas++
	return e.p, e.errLectura
}
func TestResolucionFormalizacionGETPreparacionYRecuperacion(t *testing.T) {
	s := solicitudResolucionPrueba()
	recibo := reciboResolucionPrueba(s)
	for _, v := range []uint64{7, 8} {
		p := ports.PreparacionResolucionFormalizacion{ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef, VersionEsperada: 7, VersionActual: v}
		if v == 8 {
			p.Recibo = &recibo
		}
		// Dos instancias nuevas de manejador: no dependen de POST ni sesión web.
		var anterior string
		for range 2 {
			e := &lectorResolucionHTTPPrueba{p: p}
			a := &autoridadResolucionPrueba{}
			h, _ := NuevoManejadorResolucionFormalizacion(a, e)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", RutaResolucionFormalizacion+"?expediente_ref="+s.ExpedienteRef, nil))
			if w.Code != 200 || e.llamadas != 0 || e.lecturas != 1 || a.llamadas != 1 || w.Header().Get("Set-Cookie") != "" ||
				!strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal(w.Code, w.Body.String())
			}
			var out struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if json.Unmarshal(w.Body.Bytes(), &out) != nil || len(out.Data) != 6 ||
				string(out.Data["esquema"]) != `"vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1"` {
				t.Fatal(w.Body.String())
			}
			if v == 7 && string(out.Data["recibo"]) != "null" {
				t.Fatal("recibo prematuro")
			}
			if v == 8 {
				want, _ := json.Marshal(proyectarResolucionFormalizacion(recibo))
				if string(out.Data["recibo"]) != string(want) {
					t.Fatal("recibo distinto de POST")
				}
			}
			if anterior != "" && anterior != w.Body.String() {
				t.Fatal("recuperación divergente")
			}
			anterior = w.Body.String()
		}
	}
}
func TestResolucionFormalizacionGETFronteraCerrada(t *testing.T) {
	for _, caso := range []string{"sin_lector", "denegado", "no_disponible", "recibo_ausente", "ajeno", "cookie", "actor", "duplicado", "propuesta", "version", "escape", "cuerpo", "cancelado"} {
		t.Run(caso, func(t *testing.T) {
			s := solicitudResolucionPrueba()
			e := &lectorResolucionHTTPPrueba{p: ports.PreparacionResolucionFormalizacion{ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef, VersionEsperada: 7, VersionActual: 7}}
			a := &autoridadResolucionPrueba{}
			var ejecutor EjecutorResolucionFormalizacion = e
			r := httptest.NewRequest("GET", RutaResolucionFormalizacion+"?expediente_ref="+s.ExpedienteRef, nil)
			status := 400
			switch caso {
			case "sin_lector":
				ejecutor = &ejecutorResolucionPrueba{}
				status = 503
			case "denegado":
				a.err = ports.ErrResolucionFormalizacionDenegada
				status = 403
			case "no_disponible":
				e.errLectura = errors.New("privado")
				status = 503
			case "recibo_ausente":
				e.p.VersionActual = 8
				status = 503
			case "ajeno":
				e.p.ExpedienteRef = "expediente:ajeno"
				status = 503
			case "cookie":
				r.Header.Set("Cookie", "no")
			case "actor":
				r.Header.Set("X-Actor", "no")
			case "duplicado":
				r.URL.RawQuery += "&expediente_ref=" + s.ExpedienteRef
			case "propuesta":
				r.URL.RawQuery += "&propuesta_ref=" + s.PropuestaRef
			case "version":
				r.URL.RawQuery += "&version_esperada=7"
			case "escape":
				r.URL.RawQuery = "expediente_ref=%ZZ"
			case "cuerpo":
				r = httptest.NewRequest("GET", r.URL.String(), strings.NewReader("{}"))
			case "cancelado":
				ctx, cancel := context.WithCancel(r.Context())
				cancel()
				r = r.WithContext(ctx)
				status = 503
			}
			h, _ := NuevoManejadorResolucionFormalizacion(a, ejecutor)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != status || e.llamadas != 0 || strings.Contains(w.Body.String(), `"data"`) ||
				strings.Contains(w.Body.String(), "privado") || (status == 400 && e.lecturas != 0) {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}

func cuerpoResolucionPrueba() string {
	s := solicitudResolucionPrueba()
	b, _ := json.Marshal(EntradaResolucionFormalizacion{s.ExpedienteRef, s.VersionEsperada, s.PropuestaRef, s.ClaveIdempotencia,
		s.NumeroResolucion, s.FechaResolucion, s.Motivo, s.ConfirmaRevisionPropuesta, s.ConfirmaEjercicioManual})
	return string(b)
}
func TestResolucionFormalizacionHTTPRegistroReplayYLimites(t *testing.T) {
	for _, estado := range []string{"registrada", "replay_registrada"} {
		a := &autoridadResolucionPrueba{}
		e := &ejecutorResolucionPrueba{estado: estado}
		h, err := NuevoManejadorResolucionFormalizacion(a, e)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, RutaResolucionFormalizacion, strings.NewReader(cuerpoResolucionPrueba()))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 201
		if estado == "replay_registrada" {
			want = 200
		}
		if w.Code != want {
			t.Fatal(w.Code, w.Body.String())
		}
		var out struct {
			Data map[string]any `json:"data"`
		}
		if json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatal("JSON")
		}
		if len(out.Data) != 17 || out.Data["firma_oficial"] != false || out.Data["eficacia_administrativa"] != false || out.Data["tipo_validacion"] != "manual_de_ejercicio" {
			t.Fatal(out.Data)
		}
		if w.Header().Get("Set-Cookie") != "" || a.llamadas != 1 || e.llamadas != 1 {
			t.Fatal("frontera incorrecta")
		}
	}
}
func TestResolucionFormalizacionHTTPContratoCerradoAntesDeAutoridad(t *testing.T) {
	b := cuerpoResolucionPrueba()
	for nombre, body := range map[string]string{
		"actor":     strings.TrimSuffix(b, "}") + ",\"actor\":\"rrhh\"}",
		"documento": strings.TrimSuffix(b, "}") + ",\"documento_resolucion_sha256\":\"abc\"}",
		"duplicado": strings.TrimSuffix(b, "}") + ",\"expediente_ref\":\"expediente:prueba\"}",
		"trailing":  b + " basura", "array": "[]", "null": "null",
		"fecha":        strings.Replace(b, "2026-09-06", "2026-02-30", 1),
		"confirmacion": strings.Replace(b, "\"confirma_ejercicio_manual\":true", "\"confirma_ejercicio_manual\":false", 1),
		"grande":       strings.Repeat(" ", 4097),
	} {
		t.Run(nombre, func(t *testing.T) {
			a := &autoridadResolucionPrueba{}
			e := &ejecutorResolucionPrueba{estado: "registrada"}
			h, _ := NuevoManejadorResolucionFormalizacion(a, e)
			r := httptest.NewRequest("POST", RutaResolucionFormalizacion, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 && w.Code != 422 {
				t.Fatal(w.Code, w.Body.String())
			}
			if e.llamadas != 0 || a.llamadas != 0 {
				t.Fatal("llamada con contrato inválido")
			}
		})
	}
	for _, header := range []string{"Cookie", "X-Actor", "Authorization"} {
		a := &autoridadResolucionPrueba{}
		e := &ejecutorResolucionPrueba{estado: "registrada"}
		h, _ := NuevoManejadorResolucionFormalizacion(a, e)
		r := httptest.NewRequest("POST", RutaResolucionFormalizacion, strings.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set(header, "prohibido")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || a.llamadas != 0 || e.llamadas != 0 {
			t.Fatal(header)
		}
	}
}
func TestResolucionFormalizacionHTTPNoOcultaFalloCon404(t *testing.T) {
	for _, caso := range []struct {
		err    error
		status int
	}{{ports.ErrResolucionFormalizacionDenegada, 403}, {ports.ErrResolucionFormalizacionEnConflicto, 409},
		{ports.ErrClaveResolucionFormalizacionUsada, 409}, {ports.ErrSolicitudResolucionFormalizacionInvalida, 422}, {errors.New("interno privado"), 503}} {
		a := &autoridadResolucionPrueba{}
		e := &ejecutorResolucionPrueba{err: caso.err}
		h, _ := NuevoManejadorResolucionFormalizacion(a, e)
		r := httptest.NewRequest("POST", RutaResolucionFormalizacion, strings.NewReader(cuerpoResolucionPrueba()))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.status || strings.Contains(w.Body.String(), "privado") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var a *autoridadResolucionPrueba
	if _, e := NuevoManejadorResolucionFormalizacion(a, &ejecutorResolucionPrueba{}); e == nil {
		t.Fatal("nil tipado admitido")
	}
}

func TestResolucionFormalizacionHTTPErroresContratoDelta2(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		for _, caso := range []struct {
			nombre string
			status int
			codigo string
			err    error
		}{
			{"contrato", 400, "peticion_no_valida", nil},
			{"autoridad", 403, "acceso_denegado", ports.ErrResolucionFormalizacionDenegada},
			{"permiso", 403, "acceso_denegado", ports.ErrAutorizacionDenegada},
			{"conflicto", 409, "conflicto", ports.ErrResolucionFormalizacionEnConflicto},
			{"clave_usada", 409, "conflicto", ports.ErrClaveResolucionFormalizacionUsada},
			{"contenido", 422, "contenido_no_valido", ports.ErrSolicitudResolucionFormalizacionInvalida},
			{"dependencia", 503, "servicio_no_disponible", errors.New("detalle interno privado")},
		} {
			t.Run(metodo+"/"+caso.nombre, func(t *testing.T) {
				a := &autoridadResolucionPrueba{}
				e := &lectorResolucionHTTPPrueba{}
				e.err, e.errLectura = caso.err, caso.err
				if caso.nombre == "autoridad" {
					a.err = caso.err
				}
				body := cuerpoResolucionPrueba()
				if caso.status == 400 {
					body = "{}"
				}
				if caso.status == 422 && metodo == http.MethodPost {
					// El cuerpo literal 422 procede del validador real, antes del puerto.
					body = strings.Replace(body, `"confirma_ejercicio_manual":true`, `"confirma_ejercicio_manual":false`, 1)
				}
				r := httptest.NewRequest(http.MethodPost, RutaResolucionFormalizacion, strings.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
				if metodo == http.MethodGet {
					r = httptest.NewRequest(metodo, RutaResolucionFormalizacion+"?expediente_ref=expediente:prueba", nil)
					if caso.status == 400 {
						r.URL.RawQuery += "&propuesta_ref=propuesta:privada"
					}
				}
				h, err := NuevoManejadorResolucionFormalizacion(a, e)
				if err != nil {
					t.Fatal(err)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				want := `{"error":{"clave_i18n":"api.contratacion_temporal.resolucion_formalizacion.error.` + caso.codigo + `","codigo":"` + caso.codigo + `","correlacion_ref":"corr_no_disponible"}}`
				if w.Code != caso.status || w.Body.String() != want {
					t.Fatalf("HTTP %d body=%s; esperado HTTP %d body=%s", w.Code, w.Body.String(), caso.status, want)
				}
				if len(w.Header().Values("Set-Cookie")) != 0 || strings.Contains(w.Body.String(), "privad") ||
					strings.Contains(w.Body.String(), `"data"`) || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
					t.Fatal("respuesta de error expone datos o sesión", w.Header(), w.Body.String())
				}
				sinPuerto := caso.status == 400 || caso.nombre == "autoridad" || (metodo == http.MethodPost && caso.status == 422)
				wantLecturas, wantRegistros := 0, 0
				if !sinPuerto {
					if metodo == http.MethodGet {
						wantLecturas = 1
					} else {
						wantRegistros = 1
					}
				}
				if e.lecturas != wantLecturas || e.llamadas != wantRegistros {
					t.Fatalf("llamadas inesperadas: lecturas=%d registros=%d", e.lecturas, e.llamadas)
				}
				if metodo == http.MethodPost && caso.status == 422 {
					if a.llamadas != 0 {
						t.Fatal("contenido inválido alcanzó autoridad")
					}
					t.Logf("HTTP 422 body literal del manejador real: %s", w.Body.String())
				}
			})
		}
	}
}

type consultaResolucionPDFPrueba struct {
	detalles map[uint64]ports.DetalleExpedienteRRHH
	llamadas []uint64
	fallo    bool
}

func tipoAcceptBorradorRRHHPrueba(tipo ports.TipoBorradorRRHH) string {
	switch tipo {
	case ports.BorradorInformeDefinitivo:
		return AcceptInformeDefinitivoRRHH
	case ports.BorradorResolucion:
		return AcceptResolucionRRHH
	case ports.BorradorDiligencia:
		return AcceptDiligenciaRRHH
	case ports.BorradorTomaPosesion:
		return AcceptTomaPosesionRRHH
	case ports.BorradorNotificacion:
		return AcceptNotificacionRRHH
	case ports.BorradorComunicacionCentro:
		return AcceptComunicacionCentroRRHH
	default:
		return ""
	}
}

func (c *consultaResolucionPDFPrueba) Consultar(_ context.Context, s ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	c.llamadas = append(c.llamadas, s.VersionObservada())
	if c.fallo {
		return ports.DetalleExpedienteRRHH{}, errors.New("dependencia")
	}
	detalle, ok := c.detalles[s.VersionObservada()]
	if !ok {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoObservable
	}
	return detalle.Clonar(), nil
}
func TestResolucionFormalizacionHTTPSeisPDFDesdeActual7A9ConOriginalV7(t *testing.T) {
	for _, perfil := range []struct {
		tipo   ports.TipoBorradorRRHH
		nombre string
	}{
		{ports.BorradorInformeDefinitivo, "informe-definitivo-borrador.pdf"}, {ports.BorradorResolucion, "resolucion-borrador.pdf"},
		{ports.BorradorDiligencia, "diligencia-borrador.pdf"}, {ports.BorradorTomaPosesion, "toma-posesion-borrador.pdf"},
		{ports.BorradorNotificacion, "notificacion-borrador.pdf"}, {ports.BorradorComunicacionCentro, "comunicacion-centro-borrador.pdf"},
	} {
		t.Run(string(perfil.tipo), func(t *testing.T) {
			original := detalleInformeRRHHPrueba()
			actual8 := original.Clonar()
			ultimo := actual8.Hitos[6]
			ultimo.Secuencia = 8
			ultimo.VersionExpediente = 8
			ultimo.AccionClave = "registrar_resolucion_formalizacion"
			ultimo.RealizadaEn = ultimo.RealizadaEn.Add(time.Minute)
			ultimo.FaseOrigen = "nombramiento"
			actual8.Hitos = append(actual8.Hitos, ultimo)
			actual8.Resumen.Version, actual8.Resumen.ActualizadoEn = 8, ultimo.RealizadaEn
			actual9 := actual8.Clonar()
			ultimo = actual9.Hitos[7]
			ultimo.Secuencia = 9
			ultimo.VersionExpediente = 9
			ultimo.AccionClave = "contratacion_temporal.anotacion_administrativa.registrar"
			ultimo.RealizadaEn = ultimo.RealizadaEn.Add(time.Minute)
			actual9.Hitos = append(actual9.Hitos, ultimo)
			actual9.Resumen.Version, actual9.Resumen.ActualizadoEn = 9, ultimo.RealizadaEn
			consultaActual := &consultaResolucionPDFPrueba{detalles: map[uint64]ports.DetalleExpedienteRRHH{7: original, 8: actual8, 9: actual9}}
			consultaOriginal := &consultaResolucionPDFPrueba{detalles: map[uint64]ports.DetalleExpedienteRRHH{7: original}}
			render := &renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4\nfixture de transporte")}
			render.alRenderizar = func(_ context.Context, d ports.DetalleExpedienteRRHH) {
				if d.Resumen.Version != 7 || len(d.Hitos) != 7 {
					t.Fatal("PDF reescrito desde v8")
				}
			}
			h, err := NuevoManejadorConsultaDetalleRRHHConOriginalPropuesta(consultaActual, consultaOriginal, render)
			if err != nil {
				t.Fatal(err)
			}
			for _, version := range []uint64{7, 8, 9} {
				w := httptest.NewRecorder()
				r := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, `{"expediente_ref":"expediente:ct:0001","version_observada":`+strconv.FormatUint(version, 10)+`}`)
				r.Header.Set("Accept", tipoAcceptBorradorRRHHPrueba(perfil.tipo))
				h.ServeHTTP(w, r)
				if w.Code != 200 || w.Header().Get("Content-Disposition") != `attachment; filename="`+perfil.nombre+`"` {
					t.Fatalf("v%d: %d %s", version, w.Code, w.Body.String())
				}
			}
			if got := consultaActual.llamadas; !reflect.DeepEqual(got, []uint64{7, 8, 7, 9}) || !reflect.DeepEqual(consultaOriginal.llamadas, []uint64{7}) || render.llamadas != 3 {
				t.Fatalf("lecturas no separadas: actual=%v original=%v render=%d", got, consultaOriginal.llamadas, render.llamadas)
			}
			consultaOriginal.fallo = true
			w := httptest.NewRecorder()
			r := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, `{"expediente_ref":"expediente:ct:0001","version_observada":9}`)
			r.Header.Set("Accept", tipoAcceptBorradorRRHHPrueba(perfil.tipo))
			h.ServeHTTP(w, r)
			if w.Code != 503 || render.llamadas != 3 {
				t.Fatal("fallo de antecedente histórico ocultado", w.Code)
			}
			consultaOriginal.fallo = false
			sinOriginal, err := NuevoManejadorConsultaDetalleRRHH(consultaActual, render)
			if err != nil {
				t.Fatal(err)
			}
			w = httptest.NewRecorder()
			r = nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, `{"expediente_ref":"expediente:ct:0001","version_observada":9}`)
			r.Header.Set("Accept", tipoAcceptBorradorRRHHPrueba(perfil.tipo))
			sinOriginal.ServeHTTP(w, r)
			if w.Code != 503 || render.llamadas != 3 {
				t.Fatal("ausencia de fachada histórica alteró o representó el PDF", w.Code)
			}
		})
	}
}

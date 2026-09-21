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

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

type generadorCorrelacionBorradorLlamamientoPrueba struct{}

func (generadorCorrelacionBorradorLlamamientoPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}

type preparadorBorradorLlamamientoDoble struct {
	crear, consultar                 puertosbolsa.SolicitudCrearBorradorLlamamiento
	solicitudConsulta                puertosbolsa.SolicitudConsultarBorradorLlamamiento
	errCrear, errConsultar           error
	entradaCrear                     EntradaCrearBorradorLlamamientoInterno
	entradaConsultar                 EntradaConsultarBorradorLlamamientoInterno
	contextoCrear, contextoConsultar context.Context
	llamadasConsultar                int
}

func (p *preparadorBorradorLlamamientoDoble) PrepararSolicitudCrearBorradorLlamamientoInterno(ctx context.Context, entrada EntradaCrearBorradorLlamamientoInterno) (puertosbolsa.SolicitudCrearBorradorLlamamiento, error) {
	p.contextoCrear, p.entradaCrear = ctx, entrada
	return p.crear, p.errCrear
}

func (p *preparadorBorradorLlamamientoDoble) PrepararSolicitudConsultarBorradorLlamamientoInterno(ctx context.Context, entrada EntradaConsultarBorradorLlamamientoInterno) (puertosbolsa.SolicitudConsultarBorradorLlamamiento, error) {
	p.llamadasConsultar++
	p.contextoConsultar, p.entradaConsultar = ctx, entrada
	return p.solicitudConsulta, p.errConsultar
}

type operadorBorradorLlamamientoDoble struct {
	crear, consultar                 puertosbolsa.ReciboBorradorLlamamiento
	errCrear, errConsultar           error
	solicitudCrear                   puertosbolsa.SolicitudCrearBorradorLlamamiento
	solicitudConsulta                puertosbolsa.SolicitudConsultarBorradorLlamamiento
	llamadasCrear, llamadasConsultar int
}

func (o *operadorBorradorLlamamientoDoble) Crear(_ context.Context, solicitud puertosbolsa.SolicitudCrearBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	o.llamadasCrear++
	o.solicitudCrear = solicitud
	return o.crear, o.errCrear
}

func (o *operadorBorradorLlamamientoDoble) Consultar(_ context.Context, solicitud puertosbolsa.SolicitudConsultarBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	o.llamadasConsultar++
	o.solicitudConsulta = solicitud
	return o.consultar, o.errConsultar
}

func solicitudBorradorLlamamientoPrueba(t *testing.T) (puertosbolsa.SolicitudCrearBorradorLlamamiento, puertosbolsa.SolicitudConsultarBorradorLlamamiento, puertosbolsa.ReciboBorradorLlamamiento) {
	t.Helper()
	ahora := time.Date(2026, 9, 20, 10, 30, 0, 0, time.UTC)
	resultado, vinculo, err := vecpruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionBorradorLlamamientoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "catalogo_motivos_borrador_llamamiento", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}
	contenido := dominiobolsa.ContenidoBorradorLlamamiento{Resumen: "Cobertura temporal de necesidad interna"}
	crear := puertosbolsa.SolicitudCrearBorradorLlamamiento{Vinculo: vinculo, ResultadoContexto: resultado, ClaveIdempotencia: "clave-prueba-001", Contenido: contenido, Correlacion: correlacion, Motivo: motivo}
	huella, err := dominiobolsa.HuellaComandoCrearBorradorLlamamiento(resultado.Contexto.PersonaRef, "unidad:rrhh", "ambito:rrhh", crear.ClaveIdempotencia, contenido)
	if err != nil {
		t.Fatal(err)
	}
	borrador, err := dominiobolsa.NuevoBorradorLlamamiento("borrador-llamamiento:alta:"+huella, resultado.Contexto.PersonaRef, "unidad:rrhh", "ambito:rrhh", contenido)
	if err != nil {
		t.Fatal(err)
	}
	recibo := puertosbolsa.ReciboBorradorLlamamiento{Referencia: "recibo:" + strings.Repeat("a", 64), Borrador: borrador, HuellaComandoSHA256: huella, RegistradoEn: ahora}
	consultar := puertosbolsa.SolicitudConsultarBorradorLlamamiento{Vinculo: vinculo, ResultadoContexto: resultado, BorradorRef: borrador.Referencia(), Correlacion: correlacion, Motivo: motivo}
	if errCrear, errConsultar, errRecibo := crear.Validar(), consultar.Validar(), recibo.Validar(); errCrear != nil || errConsultar != nil || errRecibo != nil {
		t.Fatalf("fixture de solicitud invalido: crear=%v consultar=%v recibo=%v borrador=%v ref=%q huella=%d instante=%v", errCrear, errConsultar, errRecibo, recibo.Borrador.Validar(), recibo.Referencia, len(recibo.HuellaComandoSHA256), recibo.RegistradoEn)
	}
	return crear, consultar, recibo
}

func nuevoHandlerBorradorLlamamientoPrueba(t *testing.T, p *preparadorBorradorLlamamientoDoble, o *operadorBorradorLlamamientoDoble) http.Handler {
	t.Helper()
	h, err := NuevoHandlerBorradorLlamamiento(p, o)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func peticionCrearBorradorLlamamiento() *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaBorradoresLlamamiento, strings.NewReader(`{"resumen":"Cobertura temporal de necesidad interna"}`))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "clave-prueba-001")
	return r
}

func TestHandlerBorradorLlamamientoCreaConIdentidadPreparadaYRespuestaMinima(t *testing.T) {
	crear, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	p := &preparadorBorradorLlamamientoDoble{crear: crear, solicitudConsulta: consultar}
	o := &operadorBorradorLlamamientoDoble{crear: recibo, consultar: recibo}
	r := peticionCrearBorradorLlamamiento()
	w := httptest.NewRecorder()
	nuevoHandlerBorradorLlamamientoPrueba(t, p, o).ServeHTTP(w, r)

	if w.Code != http.StatusCreated || o.llamadasCrear != 1 || p.contextoCrear != r.Context() || p.entradaCrear.Resumen != crear.Contenido.Resumen || p.entradaCrear.ClaveIdempotencia != crear.ClaveIdempotencia {
		t.Fatalf("alta incorrecta: estado=%d entradas=%+v llamadas=%d", w.Code, p.entradaCrear, o.llamadasCrear)
	}
	if o.solicitudCrear.ResultadoContexto.Contexto.PersonaRef != crear.ResultadoContexto.Contexto.PersonaRef || o.solicitudCrear.Vinculo.ValidarPara(crear.ResultadoContexto) != nil {
		t.Fatal("el preparador no conservó el contexto de servidor")
	}
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raiz); err != nil || len(raiz) != 1 || raiz["data"] == nil {
		t.Fatalf("respuesta no minima: %s / %v", w.Body.String(), err)
	}
	for _, prohibido := range []string{"propietario", "sesion", "decision", "ambito", "ad3", "huella_comando", crear.ResultadoContexto.Contexto.PersonaRef} {
		if strings.Contains(strings.ToLower(w.Body.String()), strings.ToLower(prohibido)) {
			t.Fatalf("filtracion de %q: %s", prohibido, w.Body.String())
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("cabeceras de no almacenamiento ausentes: %+v", w.Header())
	}
}

func TestHandlerBorradorLlamamientoReplayConsultaPropiaYGET(t *testing.T) {
	crear, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	recibo.ReintentoIdempotente = true
	p := &preparadorBorradorLlamamientoDoble{crear: crear, solicitudConsulta: consultar}
	o := &operadorBorradorLlamamientoDoble{crear: recibo, consultar: recibo}
	h := nuevoHandlerBorradorLlamamientoPrueba(t, p, o)
	post := httptest.NewRecorder()
	h.ServeHTTP(post, peticionCrearBorradorLlamamiento())
	if post.Code != http.StatusOK || !strings.Contains(post.Body.String(), `"reintento_idempotente":true`) {
		t.Fatalf("replay no conservado: %d %s", post.Code, post.Body.String())
	}
	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia(), nil))
	if get.Code != http.StatusOK || o.llamadasConsultar != 1 || p.entradaConsultar.BorradorRef != recibo.Borrador.Referencia() {
		t.Fatalf("consulta propia incorrecta: %d %+v", get.Code, p.entradaConsultar)
	}
	head := httptest.NewRecorder()
	h.ServeHTTP(head, httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia(), nil))
	if head.Code != http.StatusMethodNotAllowed || head.Header().Get("Allow") != http.MethodGet || o.llamadasConsultar != 1 || p.llamadasConsultar != 1 {
		t.Fatalf("HEAD no se denegó antes de resolver: estado=%d allow=%q preparador=%d operador=%d", head.Code, head.Header().Get("Allow"), p.llamadasConsultar, o.llamadasConsultar)
	}
}

func TestHandlerBorradorLlamamientoDetalleRechazaHEADSinInvocarCasoDeUso(t *testing.T) {
	_, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	preparador := &preparadorBorradorLlamamientoDoble{solicitudConsulta: consultar}
	operador := &operadorBorradorLlamamientoDoble{consultar: recibo}
	respuesta := httptest.NewRecorder()

	nuevoHandlerBorradorLlamamientoPrueba(t, preparador, operador).ServeHTTP(respuesta,
		httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia(), nil))

	if respuesta.Code != http.StatusMethodNotAllowed || respuesta.Header().Get("Allow") != http.MethodGet ||
		preparador.llamadasConsultar != 0 || operador.llamadasConsultar != 0 {
		t.Fatalf("HEAD alcanzó el caso de uso: estado=%d allow=%q preparador=%d operador=%d", respuesta.Code, respuesta.Header().Get("Allow"), preparador.llamadasConsultar, operador.llamadasConsultar)
	}
}

func TestRechazoHEADDetalleBorradorLlamamientoTerminaAntesDeLaCadena(t *testing.T) {
	llamadas := 0
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusTeapot)
	})
	filtro, err := EnvolverRechazoHEADDetalleBorradorLlamamiento(siguiente)
	if err != nil {
		t.Fatal(err)
	}
	referencia := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	respuesta := httptest.NewRecorder()
	filtro.ServeHTTP(respuesta, httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+referencia, nil))
	if respuesta.Code != http.StatusMethodNotAllowed || respuesta.Header().Get("Allow") != http.MethodGet ||
		respuesta.Body.Len() != 0 || llamadas != 0 {
		t.Fatalf("HEAD no terminó antes de la cadena: estado=%d allow=%q cuerpo=%q llamadas=%d", respuesta.Code, respuesta.Header().Get("Allow"), respuesta.Body.String(), llamadas)
	}
}

func TestRechazoHEADDetalleBorradorLlamamientoNoCapturaOtrasPeticiones(t *testing.T) {
	referencia := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	casos := []struct {
		nombre string
		r      *http.Request
	}{
		{"GET detalle", httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+referencia, nil)},
		{"POST colección", peticionCrearBorradorLlamamiento()},
		{"HEAD colección", httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento, nil)},
		{"HEAD referencia inválida", httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/invalida", nil)},
		{"HEAD subruta", httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+referencia+"/otra", nil)},
		{"HEAD query", httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+referencia+"?x=1", nil)},
		{"HEAD ajena", httptest.NewRequest(http.MethodHead, "/api/vec/ajena", nil)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			llamadas := 0
			filtro, err := EnvolverRechazoHEADDetalleBorradorLlamamiento(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusTeapot)
			}))
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			filtro.ServeHTTP(respuesta, caso.r)
			if respuesta.Code != http.StatusTeapot || llamadas != 1 {
				t.Fatalf("petición capturada: estado=%d llamadas=%d", respuesta.Code, llamadas)
			}
		})
	}
}

func TestRechazoHEADDetalleBorradorLlamamientoNoCapturaURLNoCanonica(t *testing.T) {
	referencia := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	nueva := func() *http.Request {
		return httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+referencia, nil)
	}
	casos := []struct {
		nombre string
		muta   func(*http.Request)
	}{
		{"opaque", func(r *http.Request) { r.URL.Opaque = r.URL.Path }},
		{"raw fragment", func(r *http.Request) { r.URL.RawFragment = "interno" }},
		{"raw path", func(r *http.Request) { r.URL.RawPath = r.URL.Path }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := nueva()
			caso.muta(r)
			llamadas := 0
			filtro, err := EnvolverRechazoHEADDetalleBorradorLlamamiento(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusTeapot)
			}))
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			filtro.ServeHTTP(respuesta, r)
			if respuesta.Code != http.StatusTeapot || llamadas != 1 {
				t.Fatalf("URL no canónica capturada: estado=%d llamadas=%d", respuesta.Code, llamadas)
			}
		})
	}
}

func TestRechazoHEADDetalleBorradorLlamamientoRechazaDependenciaNula(t *testing.T) {
	if filtro, err := EnvolverRechazoHEADDetalleBorradorLlamamiento(nil); filtro != nil || !errors.Is(err, ErrHandlerBorradorLlamamientoInvalido) {
		t.Fatalf("dependencia nula aceptada: %#v %v", filtro, err)
	}
}

func TestHandlerBorradorLlamamientoRespondeResumenEscapadoTrasReplay(t *testing.T) {
	crear, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	// 1.999 comillas caben en la entrada JSON de 4 KiB, pero sus escapes y los
	// metadatos hacen que la respuesta supere dicho límite. La salida no debe
	// reutilizar el presupuesto de entrada.
	contenido := dominiobolsa.ContenidoBorradorLlamamiento{Resumen: strings.Repeat(`"`, 1999)}
	crear.Contenido = contenido
	huella, err := dominiobolsa.HuellaComandoCrearBorradorLlamamiento(crear.ResultadoContexto.Contexto.PersonaRef, "unidad:rrhh", "ambito:rrhh", crear.ClaveIdempotencia, contenido)
	if err != nil {
		t.Fatal(err)
	}
	borrador, err := dominiobolsa.NuevoBorradorLlamamiento("borrador-llamamiento:alta:"+huella, crear.ResultadoContexto.Contexto.PersonaRef, "unidad:rrhh", "ambito:rrhh", contenido)
	if err != nil {
		t.Fatal(err)
	}
	recibo.Borrador, recibo.HuellaComandoSHA256 = borrador, huella
	consultar.BorradorRef = borrador.Referencia()
	if crear.Validar() != nil || consultar.Validar() != nil || recibo.Validar() != nil {
		t.Fatal("fixture escapado invalido")
	}
	cuerpo, err := json.Marshal(struct {
		Resumen string `json:"resumen"`
	}{Resumen: contenido.Resumen})
	if err != nil || len(cuerpo) > maximoCuerpoBorradorLlamamientoBytes {
		t.Fatalf("entrada de prueba fuera de presupuesto: bytes=%d err=%v", len(cuerpo), err)
	}
	preparador := &preparadorBorradorLlamamientoDoble{crear: crear, solicitudConsulta: consultar}
	operador := &operadorBorradorLlamamientoDoble{crear: recibo, consultar: recibo}
	handler := nuevoHandlerBorradorLlamamientoPrueba(t, preparador, operador)
	peticion := httptest.NewRequest(http.MethodPost, RutaBorradoresLlamamiento, strings.NewReader(string(cuerpo)))
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Idempotency-Key", crear.ClaveIdempotencia)
	alta := httptest.NewRecorder()
	handler.ServeHTTP(alta, peticion)
	if alta.Code != http.StatusCreated || alta.Body.Len() <= maximoCuerpoBorradorLlamamientoBytes || alta.Body.Len() > maximoRespuestaBorradorLlamamientoBytes {
		t.Fatalf("alta escapada no preservada: estado=%d bytes=%d cuerpo=%s", alta.Code, alta.Body.Len(), alta.Body.String())
	}
	var respuesta struct {
		Data struct {
			Resumen string `json:"resumen"`
		} `json:"data"`
	}
	if err := json.Unmarshal(alta.Body.Bytes(), &respuesta); err != nil || respuesta.Data.Resumen != contenido.Resumen {
		t.Fatalf("salida escapada no recuperable: err=%v resumen=%q", err, respuesta.Data.Resumen)
	}
	recibo.ReintentoIdempotente = true
	operador.crear = recibo
	replayRequest := httptest.NewRequest(http.MethodPost, RutaBorradoresLlamamiento, strings.NewReader(string(cuerpo)))
	replayRequest.Header.Set("Accept", "application/json")
	replayRequest.Header.Set("Content-Type", "application/json")
	replayRequest.Header.Set("Idempotency-Key", crear.ClaveIdempotencia)
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusOK || replay.Body.Len() <= maximoCuerpoBorradorLlamamientoBytes || !strings.Contains(replay.Body.String(), `"reintento_idempotente":true`) {
		t.Fatalf("replay escapado no recuperado: estado=%d bytes=%d", replay.Code, replay.Body.Len())
	}
}

func TestHandlerBorradorLlamamientoRechazaEntradasNoConfiablesSinPreparar(t *testing.T) {
	crear, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	cambiar := func(r *http.Request) { r.Header.Set("Authorization", "cliente-no-confiable") }
	casos := []struct {
		nombre    string
		construir func() *http.Request
	}{
		{"sin clave", func() *http.Request {
			r := peticionCrearBorradorLlamamiento()
			r.Header.Del("Idempotency-Key")
			return r
		}},
		{"dos claves", func() *http.Request {
			r := peticionCrearBorradorLlamamiento()
			r.Header["Idempotency-Key"] = []string{"clave-prueba-001", "clave-prueba-002"}
			return r
		}},
		{"clave fuera de contrato", func() *http.Request {
			r := peticionCrearBorradorLlamamiento()
			r.Header.Set("Idempotency-Key", "clave con espacio")
			return r
		}},
		{"campo extra", func() *http.Request {
			r := httptest.NewRequest(http.MethodPost, RutaBorradoresLlamamiento, strings.NewReader(`{"resumen":"Cobertura temporal de necesidad interna","persona_ref":"per_falsa"}`))
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "clave-prueba-001")
			return r
		}},
		{"authorization", func() *http.Request { r := peticionCrearBorradorLlamamiento(); cambiar(r); return r }},
		{"query en lectura", func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia()+"?otro=1", nil)
		}},
		{"cookie en lectura", func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia(), nil)
			r.Header.Set("Cookie", "x=y")
			return r
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			p := &preparadorBorradorLlamamientoDoble{crear: crear, solicitudConsulta: consultar}
			o := &operadorBorradorLlamamientoDoble{crear: recibo, consultar: recibo}
			w := httptest.NewRecorder()
			nuevoHandlerBorradorLlamamientoPrueba(t, p, o).ServeHTTP(w, caso.construir())
			if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound || o.llamadasCrear != 0 || o.llamadasConsultar != 0 || p.contextoCrear != nil || p.contextoConsultar != nil {
				t.Fatalf("entrada alcanzó preparador/operador: estado=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerBorradorLlamamientoClasificaDenegacionYNoFiltra(t *testing.T) {
	crear, consultar, recibo := solicitudBorradorLlamamientoPrueba(t)
	secreto := errors.New("secreto-interno-que-no-sale")
	p := &preparadorBorradorLlamamientoDoble{crear: crear, solicitudConsulta: consultar}
	o := &operadorBorradorLlamamientoDoble{crear: recibo, consultar: recibo, errConsultar: errors.Join(dominiovec.ErrAutorizacionDenegada, secreto)}
	w := httptest.NewRecorder()
	nuevoHandlerBorradorLlamamientoPrueba(t, p, o).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+recibo.Borrador.Referencia(), nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "acceso_denegado") || strings.Contains(w.Body.String(), secreto.Error()) {
		t.Fatalf("denegacion insegura: %d %s", w.Code, w.Body.String())
	}
}

func TestNuevoHandlerBorradorLlamamientoFallaCerrado(t *testing.T) {
	if h, err := NuevoHandlerBorradorLlamamiento(nil, nil); h != nil || !errors.Is(err, ErrHandlerBorradorLlamamientoInvalido) {
		t.Fatalf("constructor aceptó dependencias nulas: %#v %v", h, err)
	}
}

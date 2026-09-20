package bolsa

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type extractorB11Prueba struct {
	sobre    []byte
	err      error
	llamadas int
}

func (e *extractorB11Prueba) ExtraerSobrePeticionInterno(context.Context) ([]byte, error) {
	e.llamadas++
	return e.sobre, e.err
}

type autenticadorB11Prueba struct {
	err      error
	llamadas int
}

func (a *autenticadorB11Prueba) AutenticarCanalTLSMutuo(tls.ConnectionState) (httpseguridad.CanalProxyAutenticado, error) {
	a.llamadas++
	return httpseguridad.CanalProxyAutenticado{}, a.err
}

type resolvedorB11Prueba struct {
	err      error
	llamadas int
	metodo   string
	destino  string
	cuerpo   []byte
}

func (r *resolvedorB11Prueba) ResolverYVincular(
	ctx context.Context,
	_ []byte,
	_ httpseguridad.CanalProxyAutenticado,
	metodo, destino string,
	cuerpo []byte,
) (context.Context, error) {
	r.llamadas++
	r.metodo, r.destino, r.cuerpo = metodo, destino, append([]byte(nil), cuerpo...)
	if r.err != nil {
		return nil, r.err
	}
	return context.WithValue(ctx, claveContextoB11Prueba{}, "vinculado"), nil
}

type claveContextoB11Prueba struct{}

type correladorDenegacionB11Prueba struct {
	llamadas            int
	err                 error
	contextoCancelado   bool
	contextoConDeadline bool
}

func (c *correladorDenegacionB11Prueba) NuevaReferenciaCorrelacionAutorizacionV2(ctx context.Context) (string, error) {
	c.llamadas++
	c.contextoCancelado = ctx.Err() != nil
	_, c.contextoConDeadline = ctx.Deadline()
	return "correlacion_0123456789abcdef0123456789abcdef", c.err
}

func (*correladorDenegacionB11Prueba) NuevaClaveMotivoAutorizacionV2(context.Context) (string, error) {
	return "motivo_0123456789abcdef0123456789abcdef", nil
}

type registradorDenegacionB11Prueba struct {
	ordenes []puertosvec.OrdenDenegacionFronteraIdentidadV1
	err     error
}

func (r *registradorDenegacionB11Prueba) RegistrarDenegacionFronteraIdentidadV1(ctx context.Context, orden puertosvec.OrdenDenegacionFronteraIdentidadV1) error {
	if _, existe := ctx.Deadline(); !existe {
		return errors.New("auditoria sin plazo")
	}
	r.ordenes = append(r.ordenes, orden)
	return r.err
}

func TestMiddlewarePeticionSesionB11VinculaUnaVezYNoPropagaSobre(t *testing.T) {
	extractor := &extractorB11Prueba{sobre: []byte("sobre-protegido")}
	autenticador := &autenticadorB11Prueba{}
	resolvedor := &resolvedorB11Prueba{}
	denegaciones := &registradorDenegacionB11Prueba{}
	correlaciones := &correladorDenegacionB11Prueba{}
	llamadasHandler := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadasHandler++
		if r.Context().Value(claveContextoB11Prueba{}) != "vinculado" {
			t.Fatal("el handler no recibio el contexto vinculado")
		}
		if r.Header.Get("Set-Cookie") != "" {
			t.Fatal("la peticion no debe transportar cookie")
		}
		w.Header().Set("Set-Cookie", "prohibida=1")
		w.WriteHeader(http.StatusNoContent)
	})
	middleware, err := NuevoMiddlewarePeticionSesionB11(autenticador, resolvedor, extractor, denegaciones, correlaciones, handler)
	if err != nil {
		t.Fatalf("construir middleware: %v", err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusNoContent || llamadasHandler != 1 || extractor.llamadas != 1 ||
		autenticador.llamadas != 1 || resolvedor.llamadas != 1 {
		t.Fatalf("ruta B11 no siguio una sola cadena: codigo=%d handler=%d extractor=%d tls=%d resolver=%d", respuesta.Code, llamadasHandler, extractor.llamadas, autenticador.llamadas, resolvedor.llamadas)
	}
	if resolvedor.metodo != http.MethodGet || resolvedor.destino != rutaParticipacionesPropiasB11 || len(resolvedor.cuerpo) != 0 {
		t.Fatalf("peticion entregada al resolvedor incorrecta: metodo=%q destino=%q cuerpo=%q", resolvedor.metodo, resolvedor.destino, resolvedor.cuerpo)
	}
	if strings.TrimSpace(respuesta.Header().Get("Set-Cookie")) != "" {
		t.Fatal("la ruta emitio Set-Cookie")
	}
	if string(extractor.sobre) != strings.Repeat("\x00", len("sobre-protegido")) {
		t.Fatal("el sobre no quedo borrado tras el consumo")
	}
}

func TestMiddlewarePeticionSesionB11DeniegaEntradaClienteAntesDeConsumir(t *testing.T) {
	casos := []struct {
		nombre  string
		metodo  string
		destino string
		cuerpo  string
		mutar   func(*http.Request)
	}{
		{"metodo", http.MethodPost, rutaParticipacionesPropiasB11, "", nil},
		{"ruta", http.MethodGet, "/api/vec/bolsa/ajena", "", nil},
		{"query", http.MethodGet, rutaParticipacionesPropiasB11 + "?candidato=can_ajeno", "", nil},
		{"authorization", http.MethodGet, rutaParticipacionesPropiasB11, "", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ajeno") }},
		{"cookie", http.MethodGet, rutaParticipacionesPropiasB11, "", func(r *http.Request) { r.Header.Set("Cookie", "sesion=ajena") }},
		{"cuerpo", http.MethodGet, rutaParticipacionesPropiasB11, "x", nil},
		{"sin tls", http.MethodGet, rutaParticipacionesPropiasB11, "", func(r *http.Request) { r.TLS = nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			extractor := &extractorB11Prueba{sobre: []byte("sobre")}
			autenticador := &autenticadorB11Prueba{}
			resolvedor := &resolvedorB11Prueba{}
			middleware, err := NuevoMiddlewarePeticionSesionB11(autenticador, resolvedor, extractor, &registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler alcanzo una entrada no permitida")
			}))
			if err != nil {
				t.Fatalf("construir middleware: %v", err)
			}
			peticion := httptest.NewRequest(caso.metodo, caso.destino, strings.NewReader(caso.cuerpo))
			peticion.TLS = &tls.ConnectionState{}
			if caso.mutar != nil {
				caso.mutar(peticion)
			}
			respuesta := httptest.NewRecorder()
			middleware.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusUnauthorized || extractor.llamadas != 0 || autenticador.llamadas != 0 || resolvedor.llamadas != 0 {
				t.Fatalf("entrada denegada consumio capacidad: codigo=%d extractor=%d tls=%d resolvedor=%d", respuesta.Code, extractor.llamadas, autenticador.llamadas, resolvedor.llamadas)
			}
		})
	}
}

func TestMiddlewarePeticionSesionB11FallaCerradoCuandoFaltaSobreOFallaLaCadena(t *testing.T) {
	casos := []struct {
		nombre       string
		extractorErr error
		sobre        []byte
		autenticador error
		resolvedor   error
	}{
		{"sobre ausente", nil, nil, nil, nil},
		{"extractor denegado", errors.New("frontera no disponible"), nil, nil, nil},
		{"canal denegado", nil, []byte("sobre"), errors.New("tls no valido"), nil},
		{"asercion repetida", nil, []byte("sobre"), nil, errors.New("nonce consumido")},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			extractor := &extractorB11Prueba{sobre: caso.sobre, err: caso.extractorErr}
			autenticador := &autenticadorB11Prueba{err: caso.autenticador}
			resolvedor := &resolvedorB11Prueba{err: caso.resolvedor}
			middleware, err := NuevoMiddlewarePeticionSesionB11(autenticador, resolvedor, extractor, &registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler alcanzo una cadena invalida")
			}))
			if err != nil {
				t.Fatalf("construir middleware: %v", err)
			}
			peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
			peticion.TLS = &tls.ConnectionState{}
			respuesta := httptest.NewRecorder()
			middleware.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusUnauthorized {
				t.Fatalf("cadena invalida = %d", respuesta.Code)
			}
			if caso.autenticador != nil && extractor.llamadas != 0 {
				t.Fatalf("un canal TLS no autenticado alcanzo el extractor: llamadas=%d", extractor.llamadas)
			}
		})
	}
}

func TestMiddlewarePeticionSesionB11AuditaUnaDenegacionMinimaAntesDeResponder(t *testing.T) {
	denegaciones := &registradorDenegacionB11Prueba{}
	middleware, err := NuevoMiddlewarePeticionSesionB11(
		&autenticadorB11Prueba{}, &resolvedorB11Prueba{},
		&extractorB11Prueba{err: errors.New("sobre privado que no debe salir")},
		denegaciones, &correladorDenegacionB11Prueba{},
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler invocado") }),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnauthorized || len(denegaciones.ordenes) != 1 {
		t.Fatalf("denegacion no auditada una vez: codigo=%d ordenes=%d", respuesta.Code, len(denegaciones.ordenes))
	}
	orden := denegaciones.ordenes[0]
	if orden.Validar() != nil || orden.ActorRef != "" || orden.CanalRef != "" ||
		orden.Superficie != string(puertosvec.SuperficieDenegacionFronteraIdentidadV1ExternaPersonal) ||
		orden.RutaExacta != rutaParticipacionesPropiasB11 ||
		orden.Accion != string(puertosvec.AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias) ||
		orden.Motivo != string(puertosvec.MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida) {
		t.Fatalf("orden no minimizada: %+v", orden)
	}
}

func TestMiddlewarePeticionSesionB11ConservaDenegacionSiFallaAuditoriaYNoAlcanzaHandler(t *testing.T) {
	var salida bytes.Buffer
	antiguaSalida, antiguasBanderas := log.Writer(), log.Flags()
	log.SetOutput(&salida)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(antiguaSalida)
		log.SetFlags(antiguasBanderas)
	})
	denegaciones := &registradorDenegacionB11Prueba{err: errors.New("destino auditoria caido")}
	middleware, err := NuevoMiddlewarePeticionSesionB11(
		&autenticadorB11Prueba{err: errors.New("mTLS rechazado")}, &resolvedorB11Prueba{},
		&extractorB11Prueba{sobre: []byte("secreto")}, denegaciones, &correladorDenegacionB11Prueba{},
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler invocado") }),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnauthorized || len(denegaciones.ordenes) != 1 {
		t.Fatalf("fallo auditoria altero frontera: codigo=%d ordenes=%d", respuesta.Code, len(denegaciones.ordenes))
	}
	if texto := salida.String(); !strings.Contains(texto, "vec_b11_auditoria_denegacion_no_disponible") ||
		!strings.Contains(texto, "correlacion_0123456789abcdef0123456789abcdef") ||
		strings.Contains(texto, "destino auditoria caido") {
		t.Fatalf("señal de registrador no saneada: %q", texto)
	}
}

func TestMiddlewarePeticionSesionB11AuditaDenegacionDelHandlerSinDuplicarla(t *testing.T) {
	denegaciones := &registradorDenegacionB11Prueba{}
	middleware, err := NuevoMiddlewarePeticionSesionB11(
		&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
		denegaciones, &correladorDenegacionB11Prueba{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || len(denegaciones.ordenes) != 1 ||
		denegaciones.ordenes[0].Motivo != string(puertosvec.MotivoDenegacionFronteraIdentidadV1AccesoDenegado) {
		t.Fatalf("denegacion handler sin auditoria unica: codigo=%d ordenes=%+v", respuesta.Code, denegaciones.ordenes)
	}
}

func TestMiddlewarePeticionSesionB11NoEmiteSetCookieConClaveNoCanonica(t *testing.T) {
	middleware, err := NuevoMiddlewarePeticionSesionB11(
		&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
		&registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header()["set-cookie"] = []string{"prohibida=1"}
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	respuesta.Header()["set-cookie"] = []string{"preexistente=1"}
	middleware.ServeHTTP(respuesta, peticion)
	for nombre := range respuesta.Header() {
		if strings.EqualFold(nombre, "Set-Cookie") {
			t.Fatalf("Set-Cookie escapó con clave %q", nombre)
		}
	}
}

func TestMiddlewarePeticionSesionB11EliminaTrailersQuePodrianTransportarCookie(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodHead} {
		t.Run(metodo, func(t *testing.T) {
			middleware, err := NuevoMiddlewarePeticionSesionB11(
				&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
				&registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{},
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Trailer", "Set-Cookie")
					w.Header()[http.TrailerPrefix+"Set-Cookie"] = []string{"prohibida=1"}
					w.WriteHeader(http.StatusNoContent)
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			peticion := httptest.NewRequest(metodo, rutaParticipacionesPropiasB11, nil)
			peticion.TLS = &tls.ConnectionState{}
			respuesta := httptest.NewRecorder()
			respuesta.Header().Set("trailer", "Set-Cookie")
			respuesta.Header()[http.TrailerPrefix+"X-Preexistente"] = []string{"ajeno"}
			middleware.ServeHTTP(respuesta, peticion)
			for nombre := range respuesta.Header() {
				if cabeceraCookieOTrailerB11(nombre) {
					t.Fatalf("cookie o trailer escapó con clave %q", nombre)
				}
			}
		})
	}
}

func TestMiddlewarePeticionSesionB11HEADDenegadoNoEscribeCuerpoTempranoONormal(t *testing.T) {
	t.Run("temprano", func(t *testing.T) {
		middleware, err := NuevoMiddlewarePeticionSesionB11(
			&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
			&registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{},
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler invocado") }),
		)
		if err != nil {
			t.Fatal(err)
		}
		peticion := httptest.NewRequest(http.MethodHead, rutaParticipacionesPropiasB11, nil)
		peticion.Header.Set("Authorization", "prohibida")
		peticion.TLS = &tls.ConnectionState{}
		respuesta := httptest.NewRecorder()
		middleware.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusUnauthorized || respuesta.Body.Len() != 0 {
			t.Fatalf("HEAD temprano incorrecto: codigo=%d cuerpo=%q", respuesta.Code, respuesta.Body.String())
		}
	})
	t.Run("handler denegado", func(t *testing.T) {
		middleware, err := NuevoMiddlewarePeticionSesionB11(
			&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
			&registradorDenegacionB11Prueba{}, &correladorDenegacionB11Prueba{},
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("cuerpo que HEAD no entrega"))
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		peticion := httptest.NewRequest(http.MethodHead, rutaParticipacionesPropiasB11, nil)
		peticion.TLS = &tls.ConnectionState{}
		respuesta := httptest.NewRecorder()
		middleware.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusForbidden || respuesta.Body.Len() != 0 {
			t.Fatalf("HEAD del handler incorrecto: codigo=%d cuerpo=%q", respuesta.Code, respuesta.Body.String())
		}
	})
}

func TestMiddlewarePeticionSesionB11SenalaFalloDeAuditoriaSinFiltrarContenido(t *testing.T) {
	var salida bytes.Buffer
	antiguaSalida, antiguasBanderas := log.Writer(), log.Flags()
	log.SetOutput(&salida)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(antiguaSalida)
		log.SetFlags(antiguasBanderas)
	})
	correlador := &correladorDenegacionB11Prueba{err: errors.New("SECRETO_NO_REGISTRABLE")}
	middleware, err := NuevoMiddlewarePeticionSesionB11(
		&autenticadorB11Prueba{}, &resolvedorB11Prueba{}, &extractorB11Prueba{sobre: []byte("sobre")},
		&registradorDenegacionB11Prueba{}, correlador,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler invocado") }),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	peticion := httptest.NewRequest(http.MethodGet, rutaParticipacionesPropiasB11, nil).WithContext(ctx)
	peticion.TLS = &tls.ConnectionState{}
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, peticion)
	texto := salida.String()
	if respuesta.Code != http.StatusUnauthorized || correlador.llamadas != 1 || correlador.contextoCancelado || !correlador.contextoConDeadline ||
		!strings.Contains(texto, "vec_b11_auditoria_denegacion_no_disponible") ||
		strings.Contains(texto, "SECRETO_NO_REGISTRABLE") || strings.Contains(texto, rutaParticipacionesPropiasB11) {
		t.Fatalf("señal de auditoría incorrecta: codigo=%d llamadas=%d cancelado=%t plazo=%t salida=%q", respuesta.Code, correlador.llamadas, correlador.contextoCancelado, correlador.contextoConDeadline, texto)
	}
}

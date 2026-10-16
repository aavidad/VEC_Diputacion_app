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

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const claveIdempotenciaBorradorPrueba = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

type preparadorContextoBorradoresDoble struct {
	contexto gobiernoconvocatorias.ContextoOperacionBorrador
	err      error
	llamadas int
	recibido context.Context
}

func (p *preparadorContextoBorradoresDoble) PrepararContextoBorradoresInterno(
	ctx context.Context,
) (gobiernoconvocatorias.ContextoOperacionBorrador, error) {
	p.llamadas++
	p.recibido = ctx
	return p.contexto, p.err
}

type operadorBorradoresDoble struct {
	opciones              gobiernoconvocatorias.OpcionesBorradores
	lista                 gobiernoconvocatorias.ListaBorradores
	detalle               gobiernoconvocatorias.DetalleBorrador
	recibo                gobiernoconvocatorias.ProyeccionReciboBorrador
	errOpciones           error
	errLista              error
	errDetalle            error
	errCrear              error
	errActualizar         error
	llamadasOpciones      int
	llamadasLista         int
	llamadasDetalle       int
	llamadasCrear         int
	llamadasActualizar    int
	ultimaAlta            gobiernoconvocatorias.SolicitudAltaBorrador
	ultimaActualizacion   gobiernoconvocatorias.SolicitudActualizacionBorrador
	ultimoSelectorDetalle puertosbolsa.SelectorVersionConvocatoriaExacta
}

func (o *operadorBorradoresDoble) ObtenerOpciones(
	context.Context, gobiernoconvocatorias.ContextoOperacionBorrador,
) (gobiernoconvocatorias.OpcionesBorradores, error) {
	o.llamadasOpciones++
	return o.opciones, o.errOpciones
}

func (o *operadorBorradoresDoble) Listar(
	_ context.Context,
	_ gobiernoconvocatorias.ContextoOperacionBorrador,
	selector gobiernoconvocatorias.SelectorListaBorradores,
) (gobiernoconvocatorias.ListaBorradores, error) {
	o.llamadasLista++
	o.lista.Selector = selector
	return o.lista, o.errLista
}

func (o *operadorBorradoresDoble) ObtenerDetalle(
	_ context.Context,
	_ gobiernoconvocatorias.ContextoOperacionBorrador,
	selector puertosbolsa.SelectorVersionConvocatoriaExacta,
) (gobiernoconvocatorias.DetalleBorrador, error) {
	o.llamadasDetalle++
	o.ultimoSelectorDetalle = selector
	return o.detalle, o.errDetalle
}

func (o *operadorBorradoresDoble) Crear(
	_ context.Context,
	_ gobiernoconvocatorias.ContextoOperacionBorrador,
	solicitud gobiernoconvocatorias.SolicitudAltaBorrador,
) (gobiernoconvocatorias.ProyeccionReciboBorrador, error) {
	o.llamadasCrear++
	o.ultimaAlta = solicitud
	return o.recibo, o.errCrear
}

func (o *operadorBorradoresDoble) Actualizar(
	_ context.Context,
	_ gobiernoconvocatorias.ContextoOperacionBorrador,
	solicitud gobiernoconvocatorias.SolicitudActualizacionBorrador,
) (gobiernoconvocatorias.ProyeccionReciboBorrador, error) {
	o.llamadasActualizar++
	o.ultimaActualizacion = solicitud
	return o.recibo, o.errActualizar
}

func nuevoHandlerBorradoresPrueba(
	t *testing.T,
	preparador *preparadorContextoBorradoresDoble,
	operador *operadorBorradoresDoble,
) http.Handler {
	t.Helper()
	handler, err := NuevoHandlerBorradores(preparador, operador)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func contextoHTTPBorradorPrueba() gobiernoconvocatorias.ContextoOperacionBorrador {
	return gobiernoconvocatorias.ContextoOperacionBorrador{CorrelacionRef: "corr_servidor_0123456789abcdef"}
}

func reciboHTTPBorradorPrueba(accion string) gobiernoconvocatorias.ProyeccionReciboBorrador {
	return gobiernoconvocatorias.ProyeccionReciboBorrador{
		TransaccionRef: "txn_borrador_0123456789abcdef", Accion: accion,
		EstadoPrincipal: puertosbolsa.ReferenciaEstadoVersionConvocatoria{
			Referencia: "proceso:bolsa:auxiliar-2026#1", Revision: 3,
			HuellaEstadoSHA256: strings.Repeat("a", 64),
		},
		AuditoriaRef:    "aud_borrador_0123456789abcdef",
		EventoOutboxRef: "evt_borrador_0123456789abcdef",
		ConfirmadaEn:    time.Date(2026, 7, 18, 10, 30, 0, 123000000, time.UTC),
	}
}

func detalleHTTPBorradorPrueba(t *testing.T) gobiernoconvocatorias.DetalleBorrador {
	t.Helper()
	solicitud, err := solicitudAltaDesdePeticion(
		httptest.NewRecorder(),
		peticionMutacionBorrador(http.MethodPost, RutaBorradores, cuerpoAltaBorradorPrueba()),
	)
	if err != nil {
		t.Fatal(err)
	}
	referencia := func(nombre string, marca string) gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador {
		return gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador{
			Referencia: nombre, Version: 1, HuellaSHA256: strings.Repeat(marca, 64),
		}
	}
	return gobiernoconvocatorias.DetalleBorrador{
		Estado: puertosbolsa.ReferenciaEstadoVersionConvocatoria{
			Referencia: "proceso:bolsa:auxiliar-2026#1", Revision: 3,
			HuellaEstadoSHA256: strings.Repeat("a", 64),
		},
		CodigoVersionPublica: "v1", IdentificadorPublico: "bolsa-auxiliar-2026",
		Ambito: gobiernoconvocatorias.AmbitoLecturaBorrador{
			OrganizacionRef: "org_diputaciongranada", UnidadGestionRef: "uni_seleccionexterna",
		},
		ExpedienteRef: "expediente:seleccion:2026-001", Contenido: solicitud.Contenido,
		Configuracion: gobiernoconvocatorias.ConfiguracionLecturaBorrador{
			Catalogos: referencia("catalogos:bolsa", "1"), Calendario: referencia("calendario:bolsa", "2"),
			ReglasBaremacion: referencia("baremo:bolsa", "3"), FlujoProceso: referencia("convocatoria-bolsa", "4"),
			FlujoSolicitud: referencia("solicitud-bolsa", "5"), Plantilla: referencia("plantilla:bolsa:general", "6"),
			Documentos: []gobiernoconvocatorias.DocumentoLecturaBorrador{{
				Rol: "bases", PublicacionRef: "publicacion:bases:001", DocumentoRef: "documento:bases:001",
				VersionDocumento: 1, RepresentacionRef: "representacion:pdf:bases:001",
				HuellaContenidoSHA256: strings.Repeat("b", 64), FirmaValidadaRef: "firma:validada:bases:001",
				ReciboCustodiaRef: "custodia:bases:001",
			}},
		},
		Capacidades: gobiernoconvocatorias.CapacidadesFilaBorrador{Consultar: true, Actualizar: true},
	}
}

func cuerpoAltaBorradorPrueba() string {
	return `{"data":{"esquema":"vec.bolsa.borrador.crear.v1",` +
		`"plantilla_ref":"plantilla:bolsa:general","plantilla_version":2,` +
		`"plantilla_huella_sha256":"` + strings.Repeat("8", 64) + `",` +
		`"codigo_version_publica":"v1","identificador_publico":"bolsa-auxiliar-2026",` +
		`"expediente_ref":"expediente:seleccion:2026-001",` +
		`"contenido_editable":{"tipo":"bolsa","categorias":["auxiliar.administrativo"],` +
		`"titulo":"Bolsa de auxiliares","resumen":"Resumen público",` +
		`"descripcion":"Descripción completa",` +
		`"plazos":[{"referencia":"plazo:solicitud","tipo":"solicitud",` +
		`"titulo":"Presentación","descripcion":"Plazo de presentación",` +
		`"abre_en":"2026-07-20T08:00:00Z","cierra_en":"2026-07-30T23:59:59.123456Z"}],` +
		`"requisitos":[],"ayuda":[]},` +
		`"motivo_ref":"motivos_rrhh:crear_borrador","motivo_version":1,` +
		`"motivo_huella_sha256":"` + strings.Repeat("9", 64) + `"}}`
}

func cuerpoActualizacionBorradorPrueba() string {
	return `{"data":{"esquema":"vec.bolsa.borrador.actualizar.v1",` +
		`"contenido_editable":{"tipo":"bolsa","categorias":["auxiliar.administrativo"],` +
		`"titulo":"Bolsa de auxiliares actualizada","resumen":"Resumen público",` +
		`"descripcion":"Descripción completa",` +
		`"plazos":[{"referencia":"plazo:solicitud","tipo":"solicitud",` +
		`"titulo":"Presentación","descripcion":"Plazo de presentación",` +
		`"abre_en":"2026-07-20T08:00:00Z","cierra_en":"2026-07-30T23:59:59Z"}],` +
		`"requisitos":[],"ayuda":[]},` +
		`"motivo_ref":"motivos_rrhh:actualizar_borrador","motivo_version":1,` +
		`"motivo_huella_sha256":"` + strings.Repeat("9", 64) + `"}}`
}

func peticionMutacionBorrador(
	metodo, ruta, cuerpo string,
) *http.Request {
	peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Idempotency-Key", claveIdempotenciaBorradorPrueba)
	return peticion
}

func TestHandlerBorradoresAltaRealRespetaContratoYPermiteReplay(t *testing.T) {
	preparador := &preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}
	operador := &operadorBorradoresDoble{recibo: reciboHTTPBorradorPrueba(puertosbolsa.AccionCrearBorradorConvocatoria)}
	handler := nuevoHandlerBorradoresPrueba(t, preparador, operador)

	for intento := 1; intento <= 2; intento++ {
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticionMutacionBorrador(
			http.MethodPost, RutaBorradores, cuerpoAltaBorradorPrueba(),
		))
		if respuesta.Code != http.StatusCreated {
			t.Fatalf("intento %d: estado=%d: %s", intento, respuesta.Code, respuesta.Body.String())
		}
		if respuesta.Header().Get("ETag") != `"vec-borrador-v1.r3.sha256-`+strings.Repeat("a", 64)+`"` ||
			respuesta.Header().Get("Location") != RutaBorradores+"/proceso%3Abolsa%3Aauxiliar-2026/versiones/1" {
			t.Fatalf("cabeceras de alta no canónicas: ETag=%q Location=%q", respuesta.Header().Get("ETag"), respuesta.Header().Get("Location"))
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(respuesta.Body.Bytes(), &envelope); err != nil || len(envelope) != 1 || envelope["data"] == nil {
			t.Fatalf("envelope no cerrado: %s", respuesta.Body.String())
		}
		if !strings.Contains(respuesta.Body.String(), `"accion":"crear"`) ||
			strings.Contains(respuesta.Body.String(), "principal") || strings.Contains(respuesta.Body.String(), "motivo") {
			t.Fatalf("recibo público inseguro: %s", respuesta.Body.String())
		}
		comprobarCabecerasEstrictas(t, respuesta)
	}
	if operador.llamadasCrear != 2 || preparador.llamadas != 2 ||
		operador.ultimaAlta.ClaveIdempotencia != claveIdempotenciaBorradorPrueba ||
		operador.ultimaAlta.Contenido.Titulo != "Bolsa de auxiliares" {
		t.Fatalf("replay no llegó intacto: llamadas=%d/%d solicitud=%+v", preparador.llamadas, operador.llamadasCrear, operador.ultimaAlta)
	}
}

func TestHandlerBorradoresActualizacionConstruyeCASDesdeRutaYETag(t *testing.T) {
	preparador := &preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}
	operador := &operadorBorradoresDoble{recibo: reciboHTTPBorradorPrueba(puertosbolsa.AccionActualizarBorradorConvocatoria)}
	peticion := peticionMutacionBorrador(
		http.MethodPut,
		RutaBorradores+"/proceso%3Abolsa%3Aauxiliar-2026/versiones/1",
		cuerpoActualizacionBorradorPrueba(),
	)
	peticion.Header.Set("If-Match", `"vec-borrador-v1.r3.sha256-`+strings.Repeat("a", 64)+`"`)
	respuesta := httptest.NewRecorder()
	nuevoHandlerBorradoresPrueba(t, preparador, operador).ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusOK || operador.llamadasActualizar != 1 {
		t.Fatalf("estado=%d llamadas=%d: %s", respuesta.Code, operador.llamadasActualizar, respuesta.Body.String())
	}
	esperada := operador.ultimaActualizacion.Esperada
	if esperada.Referencia != "proceso:bolsa:auxiliar-2026#1" || esperada.Revision != 3 ||
		esperada.HuellaEstadoSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("CAS no ligado a ruta+ETag: %+v", esperada)
	}
	if respuesta.Header().Get("Location") != "" || !strings.Contains(respuesta.Body.String(), `"accion":"actualizar"`) {
		t.Fatalf("respuesta actualización incorrecta: %s", respuesta.Body.String())
	}
}

func TestHandlerBorradoresDistingueConflictoIdempotenciaYCASSinFiltrar(t *testing.T) {
	secreto := errors.New("dsn_postgres_supersecreto_interno")
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"idempotencia", errors.Join(puertosbolsa.ErrClaveIdempotenciaConvocatoriaReusada, secreto), http.StatusConflict, "clave_idempotencia_reutilizada"},
		{"cas", errors.Join(puertosbolsa.ErrCASVersionConvocatoriaEnConflicto, secreto), http.StatusPreconditionFailed, "estado_borrador_desactualizado"},
		{"inesperado", secreto, http.StatusInternalServerError, "error_interno"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			operador := &operadorBorradoresDoble{errActualizar: caso.err}
			peticion := peticionMutacionBorrador(
				http.MethodPut, RutaBorradores+"/proceso%3Abolsa%3Aauxiliar-2026/versiones/1",
				cuerpoActualizacionBorradorPrueba(),
			)
			peticion.Header.Set("If-Match", `"vec-borrador-v1.r3.sha256-`+strings.Repeat("a", 64)+`"`)
			respuesta := httptest.NewRecorder()
			nuevoHandlerBorradoresPrueba(t,
				&preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}, operador,
			).ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || !strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("estado=%d: %s", respuesta.Code, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "postgres") || strings.Contains(respuesta.Body.String(), "supersecreto") {
				t.Fatalf("causa filtrada: %s", respuesta.Body.String())
			}
			var raiz map[string]map[string]json.RawMessage
			if err := json.Unmarshal(respuesta.Body.Bytes(), &raiz); err != nil || len(raiz) != 1 ||
				len(raiz["error"]) != 2 || raiz["error"]["codigo"] == nil || raiz["error"]["correlacion_ref"] == nil {
				t.Fatalf("error no respeta envelope cerrado: %s", respuesta.Body.String())
			}
		})
	}
}

func TestHandlerBorradoresDeniegaPorDefectoAntesDelCasoDeUso(t *testing.T) {
	secreto := errors.New("ticket_kerberos_no_publicable")
	preparador := &preparadorContextoBorradoresDoble{err: errors.Join(ErrAutenticacionInternaAusente, secreto)}
	operador := &operadorBorradoresDoble{}
	respuesta := httptest.NewRecorder()
	nuevoHandlerBorradoresPrueba(t, preparador, operador).ServeHTTP(
		respuesta, httptest.NewRequest(http.MethodGet, RutaBorradoresOpciones, nil),
	)
	if respuesta.Code != http.StatusUnauthorized || preparador.llamadas != 1 ||
		operador.llamadasOpciones+operador.llamadasLista+operador.llamadasDetalle+
			operador.llamadasCrear+operador.llamadasActualizar != 0 {
		t.Fatalf("denegación incompleta: estado=%d llamadas=%+v", respuesta.Code, operador)
	}
	if strings.Contains(respuesta.Body.String(), "kerberos") || strings.Contains(respuesta.Body.String(), "ticket") {
		t.Fatalf("credencial filtrada: %s", respuesta.Body.String())
	}
	comprobarCabecerasEstrictas(t, respuesta)
}

func TestHandlerBorradoresConsultasDevuelvenEsquemasCerradosYListasNoNulas(t *testing.T) {
	limites := gobiernoconvocatorias.LimitesEdicionBorrador{
		MaximoCategorias: 1024, MaximoPlazos: 64, MaximoRequisitos: 256, MaximoDocumentos: 256,
		MaximoAyudas: 128, MaximoTitulo: 180, MaximoResumen: 500, MaximoDescripcion: 12_000,
		MaximoTituloPlazo: 180, MaximoDescripcionPlazo: 1_000, MaximoTituloRequisito: 180,
		MaximoDescripcionRequisito: 3_000, MaximoPreguntaAyuda: 300, MaximoRespuestaAyuda: 5_000,
	}
	operador := &operadorBorradoresDoble{
		opciones: gobiernoconvocatorias.OpcionesBorradores{Limites: limites},
		lista:    gobiernoconvocatorias.ListaBorradores{Total: 0, Elementos: nil},
	}
	handler := nuevoHandlerBorradoresPrueba(t,
		&preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}, operador,
	)

	opciones := httptest.NewRecorder()
	handler.ServeHTTP(opciones, httptest.NewRequest(http.MethodGet, RutaBorradoresOpciones, nil))
	if opciones.Code != http.StatusOK ||
		!strings.Contains(opciones.Body.String(), `"esquema":"vec.bolsa.borradores.opciones.v1"`) ||
		!strings.Contains(opciones.Body.String(), `"categorias":[]`) || strings.Contains(opciones.Body.String(), ":null") {
		t.Fatalf("opciones no canónicas: %d %s", opciones.Code, opciones.Body.String())
	}

	lista := httptest.NewRecorder()
	handler.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, RutaBorradores+"?limite=40", nil))
	if lista.Code != http.StatusOK ||
		!strings.Contains(lista.Body.String(), `"esquema":"vec.bolsa.borradores.lista.v1"`) ||
		!strings.Contains(lista.Body.String(), `"selector":{"limite":40}`) ||
		!strings.Contains(lista.Body.String(), `"elementos":[]`) || strings.Contains(lista.Body.String(), ":null") {
		t.Fatalf("lista no canónica: %d %s", lista.Code, lista.Body.String())
	}
	if operador.llamadasOpciones != 1 || operador.llamadasLista != 1 {
		t.Fatalf("consultas=%d/%d", operador.llamadasOpciones, operador.llamadasLista)
	}
}

func TestHandlerBorradoresDetalleExactoUsaRutaCanonizadaYETagFuerte(t *testing.T) {
	operador := &operadorBorradoresDoble{detalle: detalleHTTPBorradorPrueba(t)}
	handler := nuevoHandlerBorradoresPrueba(t,
		&preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}, operador,
	)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(
		http.MethodGet, RutaBorradores+"/proceso%3Abolsa%3Aauxiliar-2026/versiones/1", nil,
	))
	if respuesta.Code != http.StatusOK || operador.llamadasDetalle != 1 ||
		operador.ultimoSelectorDetalle.ID != "proceso:bolsa:auxiliar-2026" ||
		operador.ultimoSelectorDetalle.Secuencia != 1 {
		t.Fatalf("detalle=%d llamadas=%d selector=%+v: %s", respuesta.Code, operador.llamadasDetalle, operador.ultimoSelectorDetalle, respuesta.Body.String())
	}
	if respuesta.Header().Get("ETag") != `"vec-borrador-v1.r3.sha256-`+strings.Repeat("a", 64)+`"` ||
		!strings.Contains(respuesta.Body.String(), `"esquema":"vec.bolsa.borrador.detalle.v1"`) ||
		!strings.Contains(respuesta.Body.String(), `"configuracion_lectura"`) ||
		strings.Contains(respuesta.Body.String(), `"creada_por"`) {
		t.Fatalf("detalle inseguro: ETag=%q %s", respuesta.Header().Get("ETag"), respuesta.Body.String())
	}
}

func TestHandlerBorradoresRechazaLimitesJSONAmbiguoCookiesEIdentidadDeclarada(t *testing.T) {
	casos := []struct {
		nombre   string
		peticion func() *http.Request
		estado   int
	}{
		{"límite listado", func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaBorradores+"?limite=51", nil) }, http.StatusBadRequest},
		{"límite cuerpo declarado", func() *http.Request {
			p := peticionMutacionBorrador(http.MethodPost, RutaBorradores, `{}`)
			p.ContentLength = maximoCuerpoBorradorBytes + 1
			return p
		}, http.StatusRequestEntityTooLarge},
		{"clave JSON repetida", func() *http.Request {
			return peticionMutacionBorrador(http.MethodPost, RutaBorradores, `{"data":{},"data":{}}`)
		}, http.StatusBadRequest},
		{"clave JSON con capitalizacion ambigua", func() *http.Request {
			cuerpo := strings.Replace(cuerpoAltaBorradorPrueba(), `"data":`, `"Data":`, 1)
			return peticionMutacionBorrador(http.MethodPost, RutaBorradores, cuerpo)
		}, http.StatusBadRequest},
		{"clave JSON repetida sin distinguir mayusculas", func() *http.Request {
			return peticionMutacionBorrador(http.MethodPost, RutaBorradores, `{"data":{},"Data":{}}`)
		}, http.StatusBadRequest},
		{"campo identidad en cuerpo", func() *http.Request {
			cuerpo := strings.Replace(cuerpoAltaBorradorPrueba(), `"esquema":`, `"actor_ref":"per_admin","esquema":`, 1)
			return peticionMutacionBorrador(http.MethodPost, RutaBorradores, cuerpo)
		}, http.StatusBadRequest},
		{"cookie", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaBorradores+"?limite=40", nil)
			p.Header.Set("Cookie", "sesion=secreto")
			return p
		}, http.StatusBadRequest},
		{"cabecera de perfil", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaBorradores+"?limite=40", nil)
			p.Header.Set("X-VEC-Roles", "administrador")
			return p
		}, http.StatusBadRequest},
		{"identificador dinámico sin encodeURIComponent", func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaBorradores+"/proceso:bolsa:auxiliar-2026/versiones/1", nil)
		}, http.StatusNotFound},
		{"escape dinámico no canónico", func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaBorradores+"/proceso%3abolsa%3aauxiliar-2026/versiones/1", nil)
		}, http.StatusNotFound},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			preparador := &preparadorContextoBorradoresDoble{contexto: contextoHTTPBorradorPrueba()}
			operador := &operadorBorradoresDoble{}
			respuesta := httptest.NewRecorder()
			nuevoHandlerBorradoresPrueba(t, preparador, operador).ServeHTTP(respuesta, caso.peticion())
			if respuesta.Code != caso.estado || operador.llamadasCrear+operador.llamadasLista != 0 {
				t.Fatalf("estado=%d llamadas=%d/%d: %s", respuesta.Code, operador.llamadasCrear, operador.llamadasLista, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "secreto") || respuesta.Header().Get("Set-Cookie") != "" {
				t.Fatalf("entrada filtrada: %s", respuesta.Body.String())
			}
		})
	}
}

func TestNuevoHandlerBorradoresFallaCerradoConDependenciasNulas(t *testing.T) {
	var preparadorNulo *preparadorContextoBorradoresDoble
	var operadorNulo *operadorBorradoresDoble
	casos := []struct {
		preparador PreparadorContextoBorradoresInterno
		operador   OperadorBorradoresInternos
	}{
		{nil, &operadorBorradoresDoble{}}, {&preparadorContextoBorradoresDoble{}, nil},
		{preparadorNulo, &operadorBorradoresDoble{}}, {&preparadorContextoBorradoresDoble{}, operadorNulo},
	}
	for indice, caso := range casos {
		if handler, err := NuevoHandlerBorradores(caso.preparador, caso.operador); handler != nil ||
			!errors.Is(err, ErrHandlerBorradoresInvalido) {
			t.Fatalf("caso %d: handler=%v err=%v", indice, handler, err)
		}
	}
	var handlerNulo *HandlerBorradores
	respuesta := httptest.NewRecorder()
	handlerNulo.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, RutaBorradoresOpciones, nil))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("handler nil=%d", respuesta.Code)
	}
}

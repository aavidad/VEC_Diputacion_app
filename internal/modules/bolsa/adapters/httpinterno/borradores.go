package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaBorradoresOpciones = "/api/vec/bolsa/convocatorias/borradores/opciones"
	RutaBorradores         = "/api/vec/bolsa/convocatorias/borradores"
)

var (
	ErrHandlerBorradoresInvalido         = errors.New("bolsa http interno: handler de borradores invalido")
	ErrDependenciaBorradoresNoDisponible = errors.New("bolsa http interno: dependencia de borradores no disponible")
)

// PreparadorContextoBorradoresInterno solo recibe el contexto ya enriquecido
// por la frontera autenticadora. Al no recibir *http.Request, no puede elevar
// privilegios leyendo actor, persona, perfil o ámbito desde cabeceras, query o
// cuerpo enviados por el cliente.
type PreparadorContextoBorradoresInterno interface {
	PrepararContextoBorradoresInterno(context.Context) (
		gobiernoconvocatorias.ContextoOperacionBorrador,
		error,
	)
}

type OperadorBorradoresInternos interface {
	ObtenerOpciones(
		context.Context,
		gobiernoconvocatorias.ContextoOperacionBorrador,
	) (gobiernoconvocatorias.OpcionesBorradores, error)
	Listar(
		context.Context,
		gobiernoconvocatorias.ContextoOperacionBorrador,
		gobiernoconvocatorias.SelectorListaBorradores,
	) (gobiernoconvocatorias.ListaBorradores, error)
	ObtenerDetalle(
		context.Context,
		gobiernoconvocatorias.ContextoOperacionBorrador,
		puertosbolsa.SelectorVersionConvocatoriaExacta,
	) (gobiernoconvocatorias.DetalleBorrador, error)
	Crear(
		context.Context,
		gobiernoconvocatorias.ContextoOperacionBorrador,
		gobiernoconvocatorias.SolicitudAltaBorrador,
	) (gobiernoconvocatorias.ProyeccionReciboBorrador, error)
	Actualizar(
		context.Context,
		gobiernoconvocatorias.ContextoOperacionBorrador,
		gobiernoconvocatorias.SolicitudActualizacionBorrador,
	) (gobiernoconvocatorias.ProyeccionReciboBorrador, error)
}

type HandlerBorradores struct {
	preparador PreparadorContextoBorradoresInterno
	operador   OperadorBorradoresInternos
}

var (
	_ http.Handler               = (*HandlerBorradores)(nil)
	_ OperadorBorradoresInternos = (*gobiernoconvocatorias.FachadaBorradoresInternos)(nil)
)

func NuevoHandlerBorradores(
	preparador PreparadorContextoBorradoresInterno,
	operador OperadorBorradoresInternos,
) (http.Handler, error) {
	if dependenciaNula(preparador) || dependenciaNula(operador) {
		return nil, ErrHandlerBorradoresInvalido
	}
	return &HandlerBorradores{preparador: preparador, operador: operador}, nil
}

type claseRutaBorrador uint8

const (
	rutaBorradorDesconocida claseRutaBorrador = iota
	rutaBorradorOpciones
	rutaBorradorColeccion
	rutaBorradorDetalle
)

type rutaBorrador struct {
	clase    claseRutaBorrador
	selector puertosbolsa.SelectorVersionConvocatoriaExacta
}

func (h *HandlerBorradores) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlacion := nuevaCorrelacionErrorBorrador()
	if r != nil && r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r == nil || h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.operador) {
		responderErrorBorrador(w, http.StatusServiceUnavailable, "servicio_no_disponible", correlacion)
		return
	}
	ruta := reconocerRutaBorrador(r)
	if ruta.clase == rutaBorradorDesconocida {
		responderErrorBorrador(w, http.StatusNotFound, "recurso_no_encontrado", correlacion)
		return
	}
	permitidos := metodosRutaBorrador(ruta.clase)
	if !metodoPermitidoBorrador(r.Method, permitidos) {
		w.Header().Set("Allow", strings.Join(permitidos, ", "))
		responderErrorBorrador(w, http.StatusMethodNotAllowed, "metodo_no_permitido", correlacion)
		return
	}
	if !cabecerasComunesBorradorPermitidas(r) {
		responderErrorBorrador(w, http.StatusBadRequest, "peticion_no_permitida", correlacion)
		return
	}

	contexto, err := h.preparador.PrepararContextoBorradoresInterno(r.Context())
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	if correlacionBorradorValida(contexto.CorrelacionRef) {
		correlacion = contexto.CorrelacionRef
	}

	switch ruta.clase {
	case rutaBorradorOpciones:
		h.atenderOpciones(w, r, contexto, correlacion)
	case rutaBorradorColeccion:
		h.atenderColeccion(w, r, contexto, correlacion)
	case rutaBorradorDetalle:
		h.atenderDetalle(w, r, ruta.selector, contexto, correlacion)
	default:
		responderErrorBorrador(w, http.StatusNotFound, "recurso_no_encontrado", correlacion)
	}
}

func (h *HandlerBorradores) atenderOpciones(
	w http.ResponseWriter,
	r *http.Request,
	contexto gobiernoconvocatorias.ContextoOperacionBorrador,
	correlacion string,
) {
	if !entradaLecturaSinQueryBorrador(r) {
		responderErrorBorrador(w, http.StatusBadRequest, "peticion_no_permitida", correlacion)
		return
	}
	resultado, err := h.operador.ObtenerOpciones(r.Context(), contexto)
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	respuesta, err := nuevaRespuestaOpcionesBorrador(resultado)
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	responderJSONBorrador(w, http.StatusOK, respuesta, correlacion)
}

func (h *HandlerBorradores) atenderColeccion(
	w http.ResponseWriter,
	r *http.Request,
	contexto gobiernoconvocatorias.ContextoOperacionBorrador,
	correlacion string,
) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		selector, err := selectorListaDesdePeticion(r)
		if err != nil {
			responderErrorBorrador(w, http.StatusBadRequest, "selector_no_valido", correlacion)
			return
		}
		resultado, err := h.operador.Listar(r.Context(), contexto, selector)
		if err != nil {
			responderErrorBorradorClasificado(w, err, correlacion)
			return
		}
		respuesta, err := nuevaRespuestaListaBorradores(resultado, selector)
		if err != nil {
			responderErrorBorradorClasificado(w, err, correlacion)
			return
		}
		responderJSONBorrador(w, http.StatusOK, respuesta, correlacion)
		return
	}

	solicitud, err := solicitudAltaDesdePeticion(w, r)
	if err != nil {
		responderErrorEntradaBorrador(w, err, correlacion)
		return
	}
	recibo, err := h.operador.Crear(r.Context(), contexto, solicitud)
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	respuesta, etag, location, err := nuevaRespuestaReciboBorrador(recibo, "crear")
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Location", location)
	responderJSONBorrador(w, http.StatusCreated, respuesta, correlacion)
}

func (h *HandlerBorradores) atenderDetalle(
	w http.ResponseWriter,
	r *http.Request,
	selector puertosbolsa.SelectorVersionConvocatoriaExacta,
	contexto gobiernoconvocatorias.ContextoOperacionBorrador,
	correlacion string,
) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if !entradaLecturaSinQueryBorrador(r) {
			responderErrorBorrador(w, http.StatusBadRequest, "peticion_no_permitida", correlacion)
			return
		}
		resultado, err := h.operador.ObtenerDetalle(r.Context(), contexto, selector)
		if err != nil {
			responderErrorBorradorClasificado(w, err, correlacion)
			return
		}
		respuesta, etag, err := nuevaRespuestaDetalleBorrador(resultado, selector)
		if err != nil {
			responderErrorBorradorClasificado(w, err, correlacion)
			return
		}
		w.Header().Set("ETag", etag)
		responderJSONBorrador(w, http.StatusOK, respuesta, correlacion)
		return
	}

	solicitud, err := solicitudActualizacionDesdePeticion(w, r, selector)
	if err != nil {
		responderErrorEntradaBorrador(w, err, correlacion)
		return
	}
	recibo, err := h.operador.Actualizar(r.Context(), contexto, solicitud)
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	respuesta, etag, _, err := nuevaRespuestaReciboBorrador(recibo, "actualizar")
	if err != nil {
		responderErrorBorradorClasificado(w, err, correlacion)
		return
	}
	w.Header().Set("ETag", etag)
	responderJSONBorrador(w, http.StatusOK, respuesta, correlacion)
}

func reconocerRutaBorrador(r *http.Request) rutaBorrador {
	if r == nil || r.URL == nil {
		return rutaBorrador{}
	}
	escapada := r.URL.EscapedPath()
	if r.URL.Path == RutaBorradoresOpciones && escapada == RutaBorradoresOpciones {
		return rutaBorrador{clase: rutaBorradorOpciones}
	}
	if r.URL.Path == RutaBorradores && escapada == RutaBorradores {
		return rutaBorrador{clase: rutaBorradorColeccion}
	}
	prefijo := RutaBorradores + "/"
	if !strings.HasPrefix(r.URL.Path, prefijo) || !strings.HasPrefix(escapada, prefijo) {
		return rutaBorrador{}
	}
	partesEscapadas := strings.Split(strings.TrimPrefix(escapada, prefijo), "/")
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, prefijo), "/")
	if len(partesEscapadas) != 3 || len(partes) != 3 || !identificadorRutaBorradorValido(partes[0]) ||
		partes[1] != "versiones" || partesEscapadas[1] != "versiones" ||
		partesEscapadas[0] != codificarIdentificadorRutaBorrador(partes[0]) ||
		partesEscapadas[2] != partes[2] {
		return rutaBorrador{}
	}
	secuencia64, err := strconv.ParseUint(partes[2], 10, 53)
	if err != nil || secuencia64 == 0 || secuencia64 > uint64(^uint(0)>>1) {
		return rutaBorrador{}
	}
	secuencia := int(secuencia64)
	selector := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: partes[0], Secuencia: secuencia}
	if strconv.Itoa(secuencia) != partes[2] || selector.Validar() != nil {
		return rutaBorrador{}
	}
	return rutaBorrador{clase: rutaBorradorDetalle, selector: selector}
}

func identificadorRutaBorradorValido(identificador string) bool {
	if len(identificador) < 1 || len(identificador) > 160 {
		return false
	}
	primero := identificador[0]
	if !((primero >= 'A' && primero <= 'Z') || (primero >= 'a' && primero <= 'z') ||
		(primero >= '0' && primero <= '9')) {
		return false
	}
	for indice := 0; indice < len(identificador); indice++ {
		caracter := identificador[indice]
		if !((caracter >= 'A' && caracter <= 'Z') || (caracter >= 'a' && caracter <= 'z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' ||
			caracter == '.' || caracter == ':') {
			return false
		}
	}
	return true
}

// codificarIdentificadorRutaBorrador replica encodeURIComponent para el
// alfabeto cerrado de identificadores de versión. Así Location coincide con
// el cliente y una barra o escape alternativo nunca cambia el enrutamiento.
func codificarIdentificadorRutaBorrador(identificador string) string {
	var salida strings.Builder
	for indice := 0; indice < len(identificador); indice++ {
		caracter := identificador[indice]
		seguro := (caracter >= 'A' && caracter <= 'Z') || (caracter >= 'a' && caracter <= 'z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' || caracter == '.'
		if seguro {
			salida.WriteByte(caracter)
			continue
		}
		salida.WriteByte('%')
		const hexadecimal = "0123456789ABCDEF"
		salida.WriteByte(hexadecimal[caracter>>4])
		salida.WriteByte(hexadecimal[caracter&0x0f])
	}
	return salida.String()
}

func metodosRutaBorrador(clase claseRutaBorrador) []string {
	switch clase {
	case rutaBorradorOpciones:
		return []string{http.MethodGet, http.MethodHead}
	case rutaBorradorColeccion:
		return []string{http.MethodGet, http.MethodHead, http.MethodPost}
	case rutaBorradorDetalle:
		return []string{http.MethodGet, http.MethodHead, http.MethodPut}
	default:
		return nil
	}
}

func metodoPermitidoBorrador(metodo string, permitidos []string) bool {
	for _, permitido := range permitidos {
		if metodo == permitido {
			return true
		}
	}
	return false
}

func cabecerasComunesBorradorPermitidas(r *http.Request) bool {
	return !cabeceraPresente(r.Header, "Cookie") &&
		!cabeceraPresente(r.Header, "Proxy-Authorization") &&
		!cabeceraIdentidadHeredadaPresente(r.Header)
}

func entradaLecturaSinQueryBorrador(r *http.Request) bool {
	return r.URL.RawQuery == "" && !r.URL.ForceQuery && r.ContentLength == 0 &&
		len(r.TransferEncoding) == 0 && (r.Body == nil || r.Body == http.NoBody) &&
		!cabeceraPresente(r.Header, "Content-Type") &&
		!cabeceraPresente(r.Header, "Idempotency-Key") && !cabeceraPresente(r.Header, "If-Match")
}

type envelopeErrorBorrador struct {
	Error detalleErrorBorrador `json:"error"`
}

type detalleErrorBorrador struct {
	Codigo         string `json:"codigo"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func responderErrorBorradorClasificado(w http.ResponseWriter, err error, correlacion string) {
	estado, codigo := clasificarErrorBorrador(err)
	responderErrorBorrador(w, estado, codigo, correlacion)
}

func clasificarErrorBorrador(err error) (int, string) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		return http.StatusUnauthorized, "autenticacion_requerida"
	case esDependenciaBorradorNoDisponible(err):
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, puertosbolsa.ErrVersionGobernadaConvocatoriaNoEncontrada):
		return http.StatusNotFound, "borrador_no_encontrado"
	case errors.Is(err, puertosbolsa.ErrClaveIdempotenciaConvocatoriaReusada):
		return http.StatusConflict, "clave_idempotencia_reutilizada"
	case errors.Is(err, puertosbolsa.ErrCASVersionConvocatoriaEnConflicto):
		return http.StatusPreconditionFailed, "estado_borrador_desactualizado"
	case errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso),
		errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorIndeterminada):
		return http.StatusServiceUnavailable, "operacion_pendiente"
	case errors.Is(err, gobiernoconvocatorias.ErrSolicitudBorradorInvalida),
		errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida),
		errors.Is(err, dominiobolsa.ErrVersionConvocatoriaGobernadaInvalida):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	default:
		return http.StatusInternalServerError, "error_interno"
	}
}

func esDependenciaBorradorNoDisponible(err error) bool {
	return errors.Is(err, ErrDependenciaBorradoresNoDisponible) ||
		errors.Is(err, gobiernoconvocatorias.ErrFachadaBorradoresInvalida) ||
		errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) ||
		errors.Is(err, gobiernoconvocatorias.ErrConsultaBorradoresNoDisponible) ||
		errors.Is(err, puertosbolsa.ErrFuenteGobiernoConvocatoriasNoDisponible) ||
		errors.Is(err, aplicacionbolsa.ErrServicioConsultaConvocatoriaInvalido) ||
		errors.Is(err, puertosvec.ErrFuenteContextoActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible)
}

func responderErrorBorrador(w http.ResponseWriter, estado int, codigo, correlacion string) {
	if !correlacionBorradorValida(correlacion) {
		correlacion = nuevaCorrelacionErrorBorrador()
	}
	responderJSONBorrador(w, estado, envelopeErrorBorrador{Error: detalleErrorBorrador{
		Codigo: codigo, CorrelacionRef: correlacion,
	}}, correlacion)
}

func responderJSONBorrador(w http.ResponseWriter, estado int, valor any, correlacion string) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > maximoCuerpoBorradorBytes {
		estado = http.StatusInternalServerError
		w.Header().Del("ETag")
		w.Header().Del("Location")
		contenido, _ = json.Marshal(envelopeErrorBorrador{Error: detalleErrorBorrador{
			Codigo: "error_interno", CorrelacionRef: correlacion,
		}})
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func nuevaCorrelacionErrorBorrador() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "corr_error_generacion_no_disponible"
	}
	return "corr_" + hex.EncodeToString(bytes)
}

func correlacionBorradorValida(valor string) bool {
	if valor == "" || len(valor) > 180 || valor != strings.TrimSpace(valor) {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if !((caracter >= 'A' && caracter <= 'Z') || (caracter >= 'a' && caracter <= 'z') ||
			(caracter >= '0' && caracter <= '9') || strings.ContainsRune("._:/#@-", rune(caracter))) {
			return false
		}
	}
	return true
}

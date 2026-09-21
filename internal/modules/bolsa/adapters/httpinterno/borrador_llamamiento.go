package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const RutaBorradoresLlamamiento = "/api/vec/bolsa/llamamientos/borradores"

const (
	maximoCuerpoBorradorLlamamientoBytes    = 4 * 1024
	maximoRespuestaBorradorLlamamientoBytes = 16 * 1024
)

var (
	ErrHandlerBorradorLlamamientoInvalido         = errors.New("bolsa http interno: handler de borrador de llamamiento invalido")
	ErrDependenciaBorradorLlamamientoNoDisponible = errors.New("bolsa http interno: dependencia de borrador de llamamiento no disponible")
	errEntradaBorradorLlamamientoInvalida         = errors.New("bolsa http interno: entrada de borrador de llamamiento invalida")
	errEntradaBorradorLlamamientoDemasiadoGrande  = errors.New("bolsa http interno: entrada de borrador de llamamiento demasiado grande")
)

// EntradaCrearBorradorLlamamientoInterno solo contiene material controlado por
// el cliente. El preparador lo une a la identidad y contexto ya resueltos en
// servidor; por ello no recibe la peticion HTTP.
type EntradaCrearBorradorLlamamientoInterno struct {
	Resumen           string
	ClaveIdempotencia string
}

// EntradaConsultarBorradorLlamamientoInterno es el selector opaco procedente
// exclusivamente del segmento de ruta ya validado.
type EntradaConsultarBorradorLlamamientoInterno struct{ BorradorRef string }

type PreparadorBorradorLlamamientoInterno interface {
	PrepararSolicitudCrearBorradorLlamamientoInterno(context.Context, EntradaCrearBorradorLlamamientoInterno) (puertosbolsa.SolicitudCrearBorradorLlamamiento, error)
	PrepararSolicitudConsultarBorradorLlamamientoInterno(context.Context, EntradaConsultarBorradorLlamamientoInterno) (puertosbolsa.SolicitudConsultarBorradorLlamamiento, error)
}

type OperadorBorradorLlamamientoInterno interface {
	Crear(context.Context, puertosbolsa.SolicitudCrearBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error)
	Consultar(context.Context, puertosbolsa.SolicitudConsultarBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error)
}

type HandlerBorradorLlamamiento struct {
	preparador PreparadorBorradorLlamamientoInterno
	operador   OperadorBorradorLlamamientoInterno
}

var (
	_ http.Handler                       = (*HandlerBorradorLlamamiento)(nil)
	_ OperadorBorradorLlamamientoInterno = (*aplicacionbolsa.ServicioBorradorLlamamiento)(nil)
)

func NuevoHandlerBorradorLlamamiento(preparador PreparadorBorradorLlamamientoInterno, operador OperadorBorradorLlamamientoInterno) (http.Handler, error) {
	if dependenciaNula(preparador) || dependenciaNula(operador) {
		return nil, ErrHandlerBorradorLlamamientoInvalido
	}
	return &HandlerBorradorLlamamiento{preparador: preparador, operador: operador}, nil
}

// EnvolverRechazoHEADDetalleBorradorLlamamiento resuelve el contrato de
// método antes del dispatcher común. HEAD no obtiene identidad, contexto ni
// capacidad: para un detalle canónico solo anuncia que la operación admite
// GET y termina sin alcanzar la cadena funcional.
func EnvolverRechazoHEADDetalleBorradorLlamamiento(siguiente http.Handler) (http.Handler, error) {
	if dependenciaNula(siguiente) {
		return nil, ErrHandlerBorradorLlamamientoInvalido
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.Method == http.MethodHead {
			if _, clase := reconocerRutaBorradorLlamamiento(r); clase == rutaBorradorLlamamientoDetalle {
				contenido := []byte(`{"error":{"codigo":"metodo_no_permitido"}}`)
				aplicarCabeceras(w)
				w.Header().Set("Allow", http.MethodGet)
				w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
		}
		siguiente.ServeHTTP(w, r)
	}), nil
}

func (h *HandlerBorradorLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.operador) {
		responderErrorBorradorLlamamiento(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}

	referencia, clase := reconocerRutaBorradorLlamamiento(r)
	if clase == rutaBorradorLlamamientoDesconocida {
		responderErrorBorradorLlamamiento(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if !metodoBorradorLlamamientoPermitido(r.Method, clase) {
		w.Header().Set("Allow", metodosBorradorLlamamiento(clase))
		responderErrorBorradorLlamamiento(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}

	switch clase {
	case rutaBorradorLlamamientoColeccion:
		h.atenderCrear(w, r)
	case rutaBorradorLlamamientoDetalle:
		h.atenderConsultar(w, r, referencia)
	default:
		responderErrorBorradorLlamamiento(w, http.StatusNotFound, "recurso_no_encontrado")
	}
}

type claseRutaBorradorLlamamiento uint8

const (
	rutaBorradorLlamamientoDesconocida claseRutaBorradorLlamamiento = iota
	rutaBorradorLlamamientoColeccion
	rutaBorradorLlamamientoDetalle
)

var (
	patronReferenciaRutaBorradorLlamamiento    = regexp.MustCompile(`^borrador-llamamiento:alta:[0-9a-f]{64}$`)
	patronClaveIdempotenciaBorradorLlamamiento = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$`)
)

func reconocerRutaBorradorLlamamiento(r *http.Request) (string, claseRutaBorradorLlamamiento) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", rutaBorradorLlamamientoDesconocida
	}
	if r.URL.Path == RutaBorradoresLlamamiento && r.RequestURI == RutaBorradoresLlamamiento && r.URL.EscapedPath() == RutaBorradoresLlamamiento {
		return "", rutaBorradorLlamamientoColeccion
	}
	prefijo := RutaBorradoresLlamamiento + "/"
	if !strings.HasPrefix(r.URL.Path, prefijo) || r.RequestURI != r.URL.Path || r.URL.EscapedPath() != r.URL.Path {
		return "", rutaBorradorLlamamientoDesconocida
	}
	referencia := strings.TrimPrefix(r.URL.Path, prefijo)
	if referencia == "" || strings.ContainsRune(referencia, '/') || !patronReferenciaRutaBorradorLlamamiento.MatchString(referencia) {
		return "", rutaBorradorLlamamientoDesconocida
	}
	return referencia, rutaBorradorLlamamientoDetalle
}

func metodosBorradorLlamamiento(clase claseRutaBorradorLlamamiento) string {
	if clase == rutaBorradorLlamamientoColeccion {
		return http.MethodPost
	}
	return http.MethodGet
}

func metodoBorradorLlamamientoPermitido(metodo string, clase claseRutaBorradorLlamamiento) bool {
	if clase == rutaBorradorLlamamientoColeccion {
		return metodo == http.MethodPost
	}
	return metodo == http.MethodGet
}

func (h *HandlerBorradorLlamamiento) atenderCrear(w http.ResponseWriter, r *http.Request) {
	entrada, err := entradaCrearBorradorLlamamientoDesdePeticion(w, r)
	if err != nil {
		responderErrorEntradaBorradorLlamamiento(w, err)
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudCrearBorradorLlamamientoInterno(r.Context(), entrada)
	if err != nil {
		responderErrorBorradorLlamamientoClasificado(w, err)
		return
	}
	if solicitud.ClaveIdempotencia != entrada.ClaveIdempotencia || solicitud.Contenido.Resumen != entrada.Resumen || solicitud.Validar() != nil {
		responderErrorBorradorLlamamiento(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	recibo, err := h.operador.Crear(r.Context(), solicitud)
	if err != nil {
		responderErrorBorradorLlamamientoClasificado(w, err)
		return
	}
	h.responderRecibo(w, recibo, http.StatusCreated)
}

func (h *HandlerBorradorLlamamiento) atenderConsultar(w http.ResponseWriter, r *http.Request, referencia string) {
	if !entradaLecturaBorradorLlamamientoPermitida(r) {
		responderErrorBorradorLlamamiento(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	entrada := EntradaConsultarBorradorLlamamientoInterno{BorradorRef: referencia}
	solicitud, err := h.preparador.PrepararSolicitudConsultarBorradorLlamamientoInterno(r.Context(), entrada)
	if err != nil {
		responderErrorBorradorLlamamientoClasificado(w, err)
		return
	}
	if solicitud.BorradorRef != referencia || solicitud.Validar() != nil {
		responderErrorBorradorLlamamiento(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	recibo, err := h.operador.Consultar(r.Context(), solicitud)
	if err != nil {
		responderErrorBorradorLlamamientoClasificado(w, err)
		return
	}
	h.responderRecibo(w, recibo, http.StatusOK)
}

func (h *HandlerBorradorLlamamiento) responderRecibo(w http.ResponseWriter, recibo puertosbolsa.ReciboBorradorLlamamiento, alta int) {
	if recibo.Validar() != nil {
		responderErrorBorradorLlamamiento(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	estado := alta
	if recibo.ReintentoIdempotente {
		estado = http.StatusOK
	}
	responderJSONBorradorLlamamiento(w, estado, respuestaBorradorLlamamiento{Data: datosBorradorLlamamiento{
		BorradorRef: recibo.Borrador.Referencia(), Estado: string(recibo.Borrador.Estado()), Version: strconv.FormatUint(recibo.Borrador.Version(), 10),
		Resumen: recibo.Borrador.Contenido().Resumen, ReciboRef: recibo.Referencia,
		RegistradoEn: recibo.RegistradoEn.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano), ReintentoIdempotente: recibo.ReintentoIdempotente,
	}})
}

func entradaCrearBorradorLlamamientoDesdePeticion(w http.ResponseWriter, r *http.Request) (EntradaCrearBorradorLlamamientoInterno, error) {
	if !metadatosCrearBorradorLlamamientoPermitidos(r) {
		return EntradaCrearBorradorLlamamientoInterno{}, errEntradaBorradorLlamamientoInvalida
	}
	clave, err := cabeceraUnicaBorradorLlamamiento(r.Header, "Idempotency-Key")
	if err != nil || !patronClaveIdempotenciaBorradorLlamamiento.MatchString(clave) {
		return EntradaCrearBorradorLlamamientoInterno{}, errEntradaBorradorLlamamientoInvalida
	}
	var datos struct {
		Resumen string `json:"resumen"`
	}
	if err := decodificarCuerpoBorradorLlamamiento(w, r, &datos); err != nil {
		return EntradaCrearBorradorLlamamientoInterno{}, err
	}
	if datos.Resumen != strings.TrimSpace(datos.Resumen) || len(datos.Resumen) < 3 || len(datos.Resumen) > 2000 {
		return EntradaCrearBorradorLlamamientoInterno{}, errEntradaBorradorLlamamientoInvalida
	}
	return EntradaCrearBorradorLlamamientoInterno{Resumen: datos.Resumen, ClaveIdempotencia: clave}, nil
}

func metadatosCrearBorradorLlamamientoPermitidos(r *http.Request) bool {
	if r == nil || r.URL == nil || r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > maximoCuerpoBorradorLlamamientoBytes || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return false
	}
	if !cabeceraExactaBorradorLlamamiento(r.Header, "Accept", "application/json") || !cabeceraExactaBorradorLlamamiento(r.Header, "Content-Type", "application/json") {
		return false
	}
	for _, nombre := range []string{"Authorization", "Cookie", "Proxy-Authorization", "Forwarded", "Via", "Trailer", "TE", "Transfer-Encoding", "Content-Encoding", "Expect", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "Range"} {
		if cabeceraPresente(r.Header, nombre) {
			return false
		}
	}
	return !cabeceraIdentidadHeredadaPresente(r.Header)
}

func entradaLecturaBorradorLlamamientoPermitida(r *http.Request) bool {
	if r == nil || r.URL == nil || r.Body == nil || r.Body != http.NoBody || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return false
	}
	for _, nombre := range []string{"Authorization", "Cookie", "Proxy-Authorization", "Forwarded", "Via", "Trailer", "TE", "Transfer-Encoding", "Content-Encoding", "Expect", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "Range", "Idempotency-Key"} {
		if cabeceraPresente(r.Header, nombre) {
			return false
		}
	}
	return !cabeceraIdentidadHeredadaPresente(r.Header)
}

func cabeceraExactaBorradorLlamamiento(cabeceras http.Header, nombre, esperado string) bool {
	valor, err := cabeceraUnicaBorradorLlamamiento(cabeceras, nombre)
	return err == nil && valor == esperado
}

func cabeceraUnicaBorradorLlamamiento(cabeceras http.Header, nombre string) (string, error) {
	valores := make([]string, 0, 1)
	for recibido, lista := range cabeceras {
		if strings.EqualFold(recibido, nombre) {
			valores = append(valores, lista...)
		}
	}
	if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
		return "", errEntradaBorradorLlamamientoInvalida
	}
	return valores[0], nil
}

func decodificarCuerpoBorradorLlamamiento(w http.ResponseWriter, r *http.Request, destino any) error {
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpoBorradorLlamamientoBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var demasiadoGrande *http.MaxBytesError
		if errors.As(err, &demasiadoGrande) {
			return errEntradaBorradorLlamamientoDemasiadoGrande
		}
		return errEntradaBorradorLlamamientoInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return errEntradaBorradorLlamamientoInvalida
	}
	if len(contenido) > maximoCuerpoBorradorLlamamientoBytes {
		return errEntradaBorradorLlamamientoDemasiadoGrande
	}
	if err := validarJSONSinDuplicados(contenido); err != nil {
		if errors.Is(err, errEntradaBorradorDemasiadoGrande) {
			return errEntradaBorradorLlamamientoDemasiadoGrande
		}
		return errEntradaBorradorLlamamientoInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaBorradorLlamamientoInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaBorradorLlamamientoInvalida
	}
	return nil
}

func responderErrorEntradaBorradorLlamamiento(w http.ResponseWriter, err error) {
	if errors.Is(err, errEntradaBorradorLlamamientoDemasiadoGrande) {
		responderErrorBorradorLlamamiento(w, http.StatusRequestEntityTooLarge, "peticion_demasiado_grande")
		return
	}
	responderErrorBorradorLlamamiento(w, http.StatusBadRequest, "peticion_no_valida")
}

func responderErrorBorradorLlamamientoClasificado(errW http.ResponseWriter, err error) {
	estado, codigo := clasificarErrorBorradorLlamamiento(err)
	responderErrorBorradorLlamamiento(errW, estado, codigo)
}

func clasificarErrorBorradorLlamamiento(err error) (int, string) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		return http.StatusUnauthorized, "autenticacion_requerida"
	case errors.Is(err, puertosbolsa.ErrBorradorLlamamientoNoEncontrado):
		return http.StatusNotFound, "recurso_no_encontrado"
	case errors.Is(err, puertosbolsa.ErrClaveBorradorLlamamientoReutilizada):
		return http.StatusConflict, "clave_idempotencia_reutilizada"
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida), errors.Is(err, dominiobolsa.ErrBorradorLlamamientoInvalido):
		return http.StatusBadRequest, "peticion_no_valida"
	case errors.Is(err, ErrDependenciaBorradorLlamamientoNoDisponible),
		errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible),
		errors.Is(err, aplicacionbolsa.ErrServicioBorradorLlamamientoInvalido),
		errors.Is(err, puertosvec.ErrFuenteContextoActorNoDisponible),
		errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible),
		errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible),
		errors.Is(err, puertosvec.ErrRegistroDecisionNoDisponible),
		errors.Is(err, puertosvec.ErrRegistroDenegacionNoDisponible),
		errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}

type respuestaBorradorLlamamiento struct {
	Data datosBorradorLlamamiento `json:"data"`
}

type datosBorradorLlamamiento struct {
	BorradorRef          string `json:"borrador_ref"`
	Estado               string `json:"estado"`
	Version              string `json:"version"`
	Resumen              string `json:"resumen"`
	ReciboRef            string `json:"recibo_ref"`
	RegistradoEn         string `json:"registrado_en"`
	ReintentoIdempotente bool   `json:"reintento_idempotente"`
}

type envelopeErrorBorradorLlamamiento struct {
	Error detalleErrorBorradorLlamamiento `json:"error"`
}
type detalleErrorBorradorLlamamiento struct {
	Codigo string `json:"codigo"`
}

func responderErrorBorradorLlamamiento(w http.ResponseWriter, estado int, codigo string) {
	responderJSONBorradorLlamamiento(w, estado, envelopeErrorBorradorLlamamiento{Error: detalleErrorBorradorLlamamiento{Codigo: codigo}})
}

func responderJSONBorradorLlamamiento(w http.ResponseWriter, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > maximoRespuestaBorradorLlamamientoBytes {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

package httpinterno

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// B1: RRHH carga una bolsa desde el Excel de CONVOCA. POST vista-previa
// valida el fichero sin dejar rastro; POST cargas-convoca lo importa y
// constituye la bolsa con la decisión propia de la carga. El fichero viaja en
// base64 dentro de un JSON cerrado; las respuestas solo llevan códigos, que la
// pantalla traduce con su catálogo.
const (
	RutaVistaPreviaCargaConvoca = "/api/vec/bolsa/cargas-convoca/vista-previa"
	RutaConfirmarCargaConvoca   = "/api/vec/bolsa/cargas-convoca"

	EsquemaVistaPreviaCargaConvoca = "vec.bolsa.rrhh.carga_convoca.vista_previa.v1"
	EsquemaReciboCargaConvoca      = "vec.bolsa.rrhh.carga_convoca.recibo.v1"

	OperacionVistaPreviaCargaConvoca = "vista_previa"
	OperacionConfirmarCargaConvoca   = "confirmar"

	// MaximoCuerpoCargaConvoca cubre el fichero máximo en base64 más los
	// campos del JSON, y queda por debajo del límite global del servidor.
	MaximoCuerpoCargaConvoca = int64(aplicacionbolsa.MaximoBytesCargaConvoca/3*4 + 4096)
)

var claveCategoriaCargaConvoca = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,63}$`)

// ErrCategoriaCargaConvocaNoValida la devuelve el preparador cuando la clave
// no está en el catálogo RPT vigente.
var ErrCategoriaCargaConvocaNoValida = errors.New("bolsa http interno: categoria de carga no valida")

// EntradaConfirmarCargaConvoca es lo único que aporta el navegador.
type EntradaConfirmarCargaConvoca struct {
	NombreFichero   string
	Contenido       []byte
	CategoriaClave  string
	ExcluirConError bool
}

// PreparadorCargaConvoca autentica a RRHH en la frontera interna. La vista
// previa exige el mismo permiso que la carga, sin consumirlo; la confirmación
// añade el vínculo, el contexto, la correlación y el motivo del servidor y
// resuelve la categoría contra el catálogo RPT.
type PreparadorCargaConvoca interface {
	PrepararVistaPreviaCargaConvoca(context.Context) error
	PrepararConfirmacionCargaConvoca(context.Context, EntradaConfirmarCargaConvoca) (puertosbolsa.SolicitudConfirmarCargaConvoca, error)
}

type OperadorCargaConvoca interface {
	Previsualizar(ctx context.Context, nombre string, contenido []byte) (aplicacionbolsa.VistaPreviaCargaConvoca, error)
	Confirmar(ctx context.Context, solicitud puertosbolsa.SolicitudConfirmarCargaConvoca, excluirConErrores bool) (aplicacionbolsa.ResultadoCargaConvoca, error)
}

// AuditorIntentosCargaConvoca deja en la auditoría común cada intento que no
// termina bien (denegado o con error). Si no puede, la respuesta es 503.
type AuditorIntentosCargaConvoca interface {
	RegistrarIntentoFallidoCargaConvoca(ctx context.Context, operacion string, fallo error) error
}

type HandlerCargaConvoca struct {
	preparador PreparadorCargaConvoca
	operador   OperadorCargaConvoca
	auditor    AuditorIntentosCargaConvoca
}

func NuevoHandlerCargaConvoca(p PreparadorCargaConvoca, o OperadorCargaConvoca, a AuditorIntentosCargaConvoca) (http.Handler, error) {
	if dependenciaNula(p) || dependenciaNula(o) || dependenciaNula(a) {
		return nil, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return &HandlerCargaConvoca{preparador: p, operador: o, auditor: a}, nil
}

type cuerpoCargaConvoca struct {
	NombreFichero          string `json:"nombre_fichero"`
	ContenidoBase64        string `json:"contenido_base64"`
	Categoria              string `json:"categoria,omitempty"`
	ExcluirFilasConErrores bool   `json:"excluir_filas_con_errores,omitempty"`
}

func (h *HandlerCargaConvoca) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || dependenciaNula(h.preparador) || dependenciaNula(h.operador) || dependenciaNula(h.auditor) {
		responderCodigoCargaConvoca(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	ruta := r.URL.Path
	if (ruta != RutaVistaPreviaCargaConvoca && ruta != RutaConfirmarCargaConvoca) || r.URL.RawPath != "" ||
		r.URL.EscapedPath() != ruta || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderCodigoCargaConvoca(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderCodigoCargaConvoca(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	operacion := OperacionConfirmarCargaConvoca
	if ruta == RutaVistaPreviaCargaConvoca {
		operacion = OperacionVistaPreviaCargaConvoca
	}
	// La vista previa no necesita el cuerpo para comprobar el permiso: se
	// comprueba antes de leerlo.
	if operacion == OperacionVistaPreviaCargaConvoca {
		if err := h.preparador.PrepararVistaPreviaCargaConvoca(r.Context()); err != nil {
			h.fallar(w, r, operacion, err)
			return
		}
	}
	cuerpo, contenido, err := leerCuerpoCargaConvoca(w, r, operacion)
	if err != nil {
		h.fallar(w, r, operacion, err)
		return
	}
	defer borrarContenidoCargaConvoca(contenido)
	if operacion == OperacionVistaPreviaCargaConvoca {
		h.previsualizar(w, r, cuerpo, contenido)
		return
	}
	h.confirmar(w, r, cuerpo, contenido)
}

func (h *HandlerCargaConvoca) previsualizar(w http.ResponseWriter, r *http.Request, cuerpo cuerpoCargaConvoca, contenido []byte) {
	vista, err := h.operador.Previsualizar(r.Context(), cuerpo.NombreFichero, contenido)
	if err != nil {
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"data": vistaPreviaCargaConvocaJSON(vista)})
}

func (h *HandlerCargaConvoca) confirmar(w http.ResponseWriter, r *http.Request, cuerpo cuerpoCargaConvoca, contenido []byte) {
	solicitud, err := h.preparador.PrepararConfirmacionCargaConvoca(r.Context(), EntradaConfirmarCargaConvoca{
		NombreFichero: cuerpo.NombreFichero, Contenido: contenido, CategoriaClave: cuerpo.Categoria, ExcluirConError: cuerpo.ExcluirFilasConErrores})
	if err != nil {
		h.fallar(w, r, OperacionConfirmarCargaConvoca, err)
		return
	}
	resultado, err := h.operador.Confirmar(r.Context(), solicitud, cuerpo.ExcluirFilasConErrores)
	if err != nil {
		h.fallar(w, r, OperacionConfirmarCargaConvoca, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Recibo.Reutilizada {
		estado = http.StatusOK
	}
	responderJSON(w, estado, map[string]any{"data": reciboCargaConvocaJSON(resultado)})
}

// fallar registra el intento y responde con el código del fallo. Un intento
// que no se puede registrar no se responde como si nada: 503.
func (h *HandlerCargaConvoca) fallar(w http.ResponseWriter, r *http.Request, operacion string, fallo error) {
	estado, codigo := clasificarFalloCargaConvoca(fallo)
	if err := h.auditor.RegistrarIntentoFallidoCargaConvoca(r.Context(), operacion, fallo); err != nil {
		estado, codigo = http.StatusServiceUnavailable, "servicio_no_disponible"
	}
	responderCodigoCargaConvoca(w, estado, codigo)
}

// errEntradaCargaConvoca agrupa los rechazos del propio transporte.
var (
	errEntradaCargaConvoca        = errors.New("bolsa http interno: entrada de carga invalida")
	errCuerpoCargaConvocaExcesivo = errors.New("bolsa http interno: cuerpo de carga demasiado grande")
)

// leerCuerpoCargaConvoca aplica los límites antes de leer: tamaño declarado,
// lector acotado, JSON cerrado sin campos desconocidos y base64 estricto.
func leerCuerpoCargaConvoca(w http.ResponseWriter, r *http.Request, operacion string) (cuerpoCargaConvoca, []byte, error) {
	var cuerpo cuerpoCargaConvoca
	if r.Body == nil || r.Body == http.NoBody || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" ||
		len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" ||
		cabeceraPresente(r.Header, "Cookie") || cabeceraPresente(r.Header, "Authorization") ||
		cabeceraPresente(r.Header, "Proxy-Authorization") || cabeceraIdentidadHeredadaPresente(r.Header) || r.ContentLength <= 0 {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if r.ContentLength > MaximoCuerpoCargaConvoca {
		return cuerpo, nil, errCuerpoCargaConvocaExcesivo
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaximoCuerpoCargaConvoca))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cuerpo); err != nil {
		var demasiado *http.MaxBytesError
		if errors.As(err, &demasiado) {
			return cuerpo, nil, errCuerpoCargaConvocaExcesivo
		}
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if dec.Decode(&struct{}{}) != io.EOF || !aplicacionbolsa.NombreFicheroCargaConvocaValido(cuerpo.NombreFichero) ||
		cuerpo.ContenidoBase64 == "" || int64(len(cuerpo.ContenidoBase64)) > MaximoCuerpoCargaConvoca {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if operacion == OperacionVistaPreviaCargaConvoca && (cuerpo.Categoria != "" || cuerpo.ExcluirFilasConErrores) {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if operacion == OperacionConfirmarCargaConvoca && !claveCategoriaCargaConvoca.MatchString(cuerpo.Categoria) {
		return cuerpo, nil, ErrCategoriaCargaConvocaNoValida
	}
	if base64.StdEncoding.DecodedLen(len(cuerpo.ContenidoBase64)) > aplicacionbolsa.MaximoBytesCargaConvoca+2 {
		return cuerpo, nil, aplicacionbolsa.ErrFicheroCargaConvocaExcesivo
	}
	contenido, err := base64.StdEncoding.Strict().DecodeString(cuerpo.ContenidoBase64)
	cuerpo.ContenidoBase64 = ""
	if err != nil || len(contenido) == 0 {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if len(contenido) > aplicacionbolsa.MaximoBytesCargaConvoca {
		borrarContenidoCargaConvoca(contenido)
		return cuerpo, nil, aplicacionbolsa.ErrFicheroCargaConvocaExcesivo
	}
	return cuerpo, contenido, nil
}

func borrarContenidoCargaConvoca(contenido []byte) {
	for i := range contenido {
		contenido[i] = 0
	}
}

// clasificarFalloCargaConvoca traduce cada fallo a un estado y un código
// estable. Lo desconocido es indisponibilidad, nunca éxito ni permiso.
func clasificarFalloCargaConvoca(err error) (int, string) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		return http.StatusUnauthorized, "autenticacion_requerida"
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, errCuerpoCargaConvocaExcesivo), errors.Is(err, aplicacionbolsa.ErrFicheroCargaConvocaExcesivo):
		return http.StatusRequestEntityTooLarge, "fichero_demasiado_grande"
	case errors.Is(err, aplicacionbolsa.ErrFilasCargaConvocaExcesivas):
		return http.StatusUnprocessableEntity, "demasiadas_filas"
	case errors.Is(err, aplicacionbolsa.ErrFicheroCargaConvocaInvalido):
		return http.StatusUnprocessableEntity, "fichero_no_valido"
	case errors.Is(err, aplicacionbolsa.ErrCargaConvocaBloqueada):
		return http.StatusUnprocessableEntity, "fichero_no_cargable"
	case errors.Is(err, aplicacionbolsa.ErrCargaConvocaConErrores):
		return http.StatusUnprocessableEntity, "filas_con_errores_sin_aceptar"
	case errors.Is(err, ErrCategoriaCargaConvocaNoValida):
		return http.StatusUnprocessableEntity, "categoria_no_valida"
	case errors.Is(err, puertosbolsa.ErrConstitucionBolsaEnConflicto):
		return http.StatusConflict, "bolsa_en_conflicto"
	case errors.Is(err, errEntradaCargaConvoca):
		return http.StatusBadRequest, "peticion_no_valida"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}

func responderCodigoCargaConvoca(w http.ResponseWriter, estado int, codigo string) {
	responderJSON(w, estado, map[string]any{"error": map[string]string{"codigo": codigo}})
}

type incidenciaCargaConvocaJSON struct {
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type filaCargaConvocaJSON struct {
	Numero          int                          `json:"numero"`
	Estado          string                       `json:"estado"`
	Posicion        int                          `json:"posicion,omitempty"`
	Documento       string                       `json:"documento,omitempty"`
	PrimerApellido  string                       `json:"primer_apellido,omitempty"`
	SegundoApellido string                       `json:"segundo_apellido,omitempty"`
	Nombre          string                       `json:"nombre,omitempty"`
	Experiencia     string                       `json:"experiencia,omitempty"`
	Formacion       string                       `json:"formacion,omitempty"`
	Total           string                       `json:"total,omitempty"`
	Errores         []incidenciaCargaConvocaJSON `json:"errores"`
	Avisos          []string                     `json:"avisos"`
}

func vistaPreviaCargaConvocaJSON(v aplicacionbolsa.VistaPreviaCargaConvoca) map[string]any {
	filas := make([]filaCargaConvocaJSON, 0, len(v.Filas))
	for _, f := range v.Filas {
		fila := filaCargaConvocaJSON{Numero: f.Numero, Estado: f.Estado, Posicion: f.Posicion, Documento: f.Documento,
			PrimerApellido: f.PrimerApellido, SegundoApellido: f.SegundoApellido, Nombre: f.Nombre,
			Experiencia: f.Experiencia, Formacion: f.Formacion, Total: f.Total,
			Errores: make([]incidenciaCargaConvocaJSON, 0, len(f.Errores)), Avisos: append([]string{}, f.Avisos...)}
		for _, e := range f.Errores {
			fila.Errores = append(fila.Errores, incidenciaCargaConvocaJSON{Campo: e.Campo, Codigo: e.Codigo})
		}
		filas = append(filas, fila)
	}
	return map[string]any{
		"esquema": EsquemaVistaPreviaCargaConvoca, "huella_sha256": v.HuellaSHA256, "nombre_fichero": v.NombreFichero,
		"esquema_fichero": v.Esquema, "filas_leidas": v.FilasLeidas, "aceptadas": v.Aceptadas, "rechazadas": v.Rechazadas,
		"con_avisos": v.ConAvisos, "bloqueo": v.Bloqueo, "filas": filas,
	}
}

func reciboCargaConvocaJSON(r aplicacionbolsa.ResultadoCargaConvoca) map[string]any {
	pendientes := make([]map[string]any, 0, len(r.Recibo.PendientesRevision))
	for _, p := range r.Recibo.PendientesRevision {
		pendientes = append(pendientes, map[string]any{"fila": p.FilaNumero, "motivo": p.Motivo})
	}
	sustituidas := make([]map[string]any, 0, len(r.Recibo.SustituyeA))
	for _, s := range r.Recibo.SustituyeA {
		sustituidas = append(sustituidas, map[string]any{"bolsa_ref": s.BolsaRef, "version_bolsa": s.VersionBolsa})
	}
	return map[string]any{
		"esquema": EsquemaReciboCargaConvoca, "bolsa_ref": r.Recibo.BolsaRef, "version_bolsa": r.Recibo.VersionBolsa,
		"acta_ref": r.Recibo.ActaRef, "huella_sha256": r.HuellaSHA256, "reutilizada": r.Recibo.Reutilizada,
		"acta_reutilizada": r.ActaReutilizada, "filas_cargadas": r.FilasCargadas, "filas_excluidas": r.FilasExcluidas,
		"pendientes_revision": pendientes, "sustituye_a": sustituidas, "auditoria_ref": r.Recibo.AuditoriaRef,
		"confirmada_en": r.Recibo.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
	}
}

package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// B1: RRHH carga una bolsa desde el Excel de CONVOCA. POST vista-previa
// valida y registra la lectura sin constituir la bolsa. POST cargas-convoca
// importa y constituye la bolsa con la decisión propia. El fichero viaja en
// base64 dentro de un JSON cerrado; las respuestas solo llevan códigos, que la
// pantalla traduce con su catálogo.
const (
	RutaVistaPreviaCargaConvoca = "/api/vec/bolsa/cargas-convoca/vista-previa"
	RutaConfirmarCargaConvoca   = "/api/vec/bolsa/cargas-convoca"

	EsquemaVistaPreviaCargaConvoca = puertosbolsa.EsquemaVistaPreviaCargaConvocaV1
	EsquemaReciboCargaConvoca      = "vec.bolsa.rrhh.carga_convoca.recibo.v1"

	OperacionVistaPreviaCargaConvoca = "vista_previa"
	OperacionConfirmarCargaConvoca   = "confirmar"
	limiteDefectoVistaPreviaConvoca  = 50
	limiteMaximoVistaPreviaConvoca   = 100

	// MaximoCuerpoCargaConvoca cubre el fichero máximo en base64 más los
	// campos del JSON, y queda por debajo del límite global del servidor.
	MaximoCuerpoCargaConvoca = int64(aplicacionbolsa.MaximoBytesCargaConvoca/3*4 + 4096)
)

var claveCategoriaCargaConvoca = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,63}$`)

// ErrCategoriaCargaConvocaNoValida la devuelve el preparador cuando la clave
// no está en el catálogo RPT vigente.
var ErrCategoriaCargaConvocaNoValida = errors.New("bolsa http interno: categoria de carga no valida")
var errPaginacionCargaConvocaNoValida = errors.New("bolsa http interno: paginacion de carga no valida")

// EntradaConfirmarCargaConvoca es lo único que aporta el navegador.
type EntradaConfirmarCargaConvoca struct {
	NombreFichero   string
	Contenido       []byte
	CategoriaClave  string
	ExcluirConError bool
}

type EntradaVistaPreviaCargaConvoca struct {
	NombreFichero  string
	Contenido      []byte
	CategoriaClave string
	Pagina         puertosbolsa.PaginaVistaPreviaCargaConvoca
}

// PreparadorCargaConvoca aplica la frontera de acceso de cada operación. La
// lectura queda cerrada hasta disponer del consumidor nominal de consulta;
// confirmar conserva su vínculo, contexto y autorización V3 propios.
type PreparadorCargaConvoca interface {
	PrepararVistaPreviaCargaConvoca(context.Context) error
	PrepararSolicitudVistaPreviaCargaConvoca(context.Context, EntradaVistaPreviaCargaConvoca) (puertosbolsa.SolicitudVistaPreviaCargaConvoca, error)
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

// La preparación calcula y pagina en aplicación; el consumo V3/B93 confirma
// el apunte común antes de que HTTP pueda escribir la respuesta ya serializada.
type VistaPreviaAutorizadaCargaConvoca interface {
	Preparar(context.Context, puertosbolsa.SolicitudVistaPreviaCargaConvoca) (aplicacionbolsa.VistaPreviaCargaConvocaPreparada, error)
	Consumir(context.Context, aplicacionbolsa.VistaPreviaCargaConvocaPreparada) (puertosbolsa.AcuseVistaPreviaCargaConvoca, error)
}

type HandlerCargaConvoca struct {
	preparador PreparadorCargaConvoca
	operador   OperadorCargaConvoca
	auditor    AuditorIntentosCargaConvoca
	lectura    VistaPreviaAutorizadaCargaConvoca
}

type claveRecursoIntentoCargaConvoca struct{}

// El recurso del libro sólo se conoce después de decodificarlo. Se deriva de
// sus bytes y nunca de un identificador, nombre o cabecera del navegador.
func contextoRecursoIntentoCargaConvoca(ctx context.Context, contenido []byte) context.Context {
	huella := sha256.Sum256(contenido)
	return context.WithValue(ctx, claveRecursoIntentoCargaConvoca{}, "fichero:sha256:"+hex.EncodeToString(huella[:]))
}

// RecursoIntentoCargaConvoca devuelve la referencia opaca capturada por esta
// frontera; una entrada rechazada antes de decodificarla no tiene acta.
func RecursoIntentoCargaConvoca(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	ref, ok := ctx.Value(claveRecursoIntentoCargaConvoca{}).(string)
	return ref, ok && ref != ""
}

func NuevoHandlerCargaConvoca(p PreparadorCargaConvoca, o OperadorCargaConvoca,
	a AuditorIntentosCargaConvoca, lectura VistaPreviaAutorizadaCargaConvoca,
) (http.Handler, error) {
	if dependenciaNula(p) || dependenciaNula(o) || dependenciaNula(a) || dependenciaNula(lectura) {
		return nil, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return &HandlerCargaConvoca{preparador: p, operador: o, auditor: a, lectura: lectura}, nil
}

type cuerpoCargaConvoca struct {
	NombreFichero          string          `json:"nombre_fichero"`
	ContenidoBase64        string          `json:"contenido_base64"`
	Categoria              string          `json:"categoria,omitempty"`
	ExcluirFilasConErrores bool            `json:"excluir_filas_con_errores,omitempty"`
	Filtro                 json.RawMessage `json:"filtro,omitempty"`
	Limite                 json.RawMessage `json:"limite,omitempty"`
	Desplazamiento         json.RawMessage `json:"desplazamiento,omitempty"`
}

type paginaVistaPreviaCargaConvoca struct {
	filtro         string
	limite         int
	desplazamiento int
}

func (h *HandlerCargaConvoca) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || dependenciaNula(h.preparador) || dependenciaNula(h.operador) || dependenciaNula(h.auditor) || dependenciaNula(h.lectura) {
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
	r = r.WithContext(contextoRecursoIntentoCargaConvoca(r.Context(), contenido))
	pagina, err := paginaVistaPreviaDesdeCuerpo(cuerpo)
	if err != nil {
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, err)
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudVistaPreviaCargaConvoca(r.Context(), EntradaVistaPreviaCargaConvoca{
		NombreFichero: cuerpo.NombreFichero, Contenido: contenido, CategoriaClave: cuerpo.Categoria,
		Pagina: puertosbolsa.PaginaVistaPreviaCargaConvoca{Filtro: pagina.filtro, Limite: pagina.limite, Desplazamiento: pagina.desplazamiento},
	})
	if err != nil {
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, err)
		return
	}
	preparada, err := h.lectura.Preparar(r.Context(), solicitud)
	if err != nil {
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, err)
		return
	}
	respuesta, err := json.Marshal(map[string]any{"data": vistaPreviaCargaConvocaJSON(preparada)})
	if err != nil {
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, puertosbolsa.ErrVistaPreviaCargaConvocaNoDisponible)
		return
	}
	defer clear(respuesta)
	acuse, err := h.lectura.Consumir(r.Context(), preparada)
	if err != nil || acuse.Validar() != nil {
		if err == nil {
			err = puertosbolsa.ErrVistaPreviaCargaConvocaNoDisponible
		}
		h.fallar(w, r, OperacionVistaPreviaCargaConvoca, err)
		return
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(respuesta)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respuesta)
}

func (h *HandlerCargaConvoca) confirmar(w http.ResponseWriter, r *http.Request, cuerpo cuerpoCargaConvoca, contenido []byte) {
	r = r.WithContext(contextoRecursoIntentoCargaConvoca(r.Context(), contenido))
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
	if err := decodificarCuerpoCargaConvoca(dec, &cuerpo); err != nil {
		var demasiado *http.MaxBytesError
		if errors.As(err, &demasiado) {
			return cuerpo, nil, errCuerpoCargaConvocaExcesivo
		}
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if !aplicacionbolsa.NombreFicheroCargaConvocaValido(cuerpo.NombreFichero) ||
		cuerpo.ContenidoBase64 == "" || int64(len(cuerpo.ContenidoBase64)) > MaximoCuerpoCargaConvoca {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if operacion == OperacionVistaPreviaCargaConvoca && cuerpo.ExcluirFilasConErrores {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if operacion == OperacionConfirmarCargaConvoca &&
		(len(cuerpo.Filtro) != 0 || len(cuerpo.Limite) != 0 || len(cuerpo.Desplazamiento) != 0) {
		return cuerpo, nil, errEntradaCargaConvoca
	}
	if !claveCategoriaCargaConvoca.MatchString(cuerpo.Categoria) {
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

// Lee cada propiedad una sola vez. Rechaza duplicados, incluidos los campos
// de paginación: un mismo cuerpo no puede expresar dos páginas distintas.
func decodificarCuerpoCargaConvoca(dec *json.Decoder, cuerpo *cuerpoCargaConvoca) error {
	inicio, err := dec.Token()
	if err != nil {
		return err
	}
	if inicio != json.Delim('{') {
		return errEntradaCargaConvoca
	}
	vistos := make(map[string]bool, 7)
	for dec.More() {
		claveToken, err := dec.Token()
		if err != nil {
			return err
		}
		clave, ok := claveToken.(string)
		if !ok || vistos[clave] {
			return errEntradaCargaConvoca
		}
		vistos[clave] = true
		var destino any
		switch clave {
		case "nombre_fichero":
			destino = &cuerpo.NombreFichero
		case "contenido_base64":
			destino = &cuerpo.ContenidoBase64
		case "categoria":
			destino = &cuerpo.Categoria
		case "excluir_filas_con_errores":
			destino = &cuerpo.ExcluirFilasConErrores
		case "filtro":
			destino = &cuerpo.Filtro
		case "limite":
			destino = &cuerpo.Limite
		case "desplazamiento":
			destino = &cuerpo.Desplazamiento
		default:
			return errEntradaCargaConvoca
		}
		if err := dec.Decode(destino); err != nil {
			return err
		}
	}
	fin, err := dec.Token()
	if err != nil {
		return err
	}
	if fin != json.Delim('}') {
		return errEntradaCargaConvoca
	}
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return errEntradaCargaConvoca
	}
	return nil
}

func paginaVistaPreviaDesdeCuerpo(c cuerpoCargaConvoca) (paginaVistaPreviaCargaConvoca, error) {
	pagina := paginaVistaPreviaCargaConvoca{filtro: "todas", limite: limiteDefectoVistaPreviaConvoca}
	if len(c.Filtro) != 0 {
		if string(c.Filtro) == "null" || json.Unmarshal(c.Filtro, &pagina.filtro) != nil {
			return paginaVistaPreviaCargaConvoca{}, errEntradaCargaConvoca
		}
		switch pagina.filtro {
		case "todas", "aceptadas", "rechazadas", "con_avisos":
		default:
			return paginaVistaPreviaCargaConvoca{}, errPaginacionCargaConvocaNoValida
		}
	}
	if len(c.Limite) != 0 {
		limite, err := enteroPaginaCargaConvoca(c.Limite)
		if err != nil {
			return paginaVistaPreviaCargaConvoca{}, err
		}
		if limite < 1 || limite > limiteMaximoVistaPreviaConvoca {
			return paginaVistaPreviaCargaConvoca{}, errPaginacionCargaConvocaNoValida
		}
		pagina.limite = limite
	}
	if len(c.Desplazamiento) != 0 {
		desplazamiento, err := enteroPaginaCargaConvoca(c.Desplazamiento)
		if err != nil {
			return paginaVistaPreviaCargaConvoca{}, err
		}
		if desplazamiento < 0 || desplazamiento > aplicacionbolsa.MaximoFilasCargaConvoca {
			return paginaVistaPreviaCargaConvoca{}, errPaginacionCargaConvocaNoValida
		}
		pagina.desplazamiento = desplazamiento
	}
	return pagina, nil
}

func enteroPaginaCargaConvoca(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || (raw[0] != '-' && (raw[0] < '0' || raw[0] > '9')) {
		return 0, errEntradaCargaConvoca
	}
	valor, err := strconv.ParseInt(string(raw), 10, strconv.IntSize)
	if err != nil {
		return 0, errPaginacionCargaConvocaNoValida
	}
	return int(valor), nil
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
	case errors.Is(err, errPaginacionCargaConvocaNoValida):
		return http.StatusUnprocessableEntity, "paginacion_no_valida"
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

func vistaPreviaCargaConvocaJSON(preparada aplicacionbolsa.VistaPreviaCargaConvocaPreparada) map[string]any {
	v, pagina := preparada.Vista, preparada.Pagina
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
		"con_avisos": v.ConAvisos, "bloqueo": v.Bloqueo, "filtro": pagina.Filtro, "limite": pagina.Limite,
		"desplazamiento": pagina.Desplazamiento, "total_filtrado": preparada.TotalFiltrado, "filas": filas,
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

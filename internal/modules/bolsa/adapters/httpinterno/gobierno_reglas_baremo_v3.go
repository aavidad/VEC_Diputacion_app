package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"

	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	RutaAltaGobiernoReglasBaremoV3      = "/api/vec/bolsa/reglas-baremo/borradores/alta"
	RutaConsultaGobiernoReglasBaremoV3  = "/api/vec/bolsa/reglas-baremo/versiones/consultar"
	RutaRecuperarGobiernoReglasBaremoV3 = "/api/vec/bolsa/reglas-baremo/recibos/recuperar"
	maximoEntradaGobiernoReglasV3       = 256 * 1024
)

type AutoridadGobiernoReglasBaremoV3 interface {
	Credenciales(context.Context) (app.CredencialesGobiernoV3, error)
}

// La raíz liga este callback al registrador común ya compuesto. No recibe
// identidad, superficie, cuerpo ni material V3 del cliente.
type AuditarRechazoSesionGobiernoReglasV3 func(context.Context, string, error) error

// Registra errores de entrada una vez acreditada la sesión nominal.
type AuditarErrorGobiernoReglasV3 func(context.Context, error) error

type OperadorGobiernoReglasBaremoV3 interface {
	GuardarAltaBorrador(context.Context, app.CredencialesGobiernoV3, app.PeticionAltaBorradorV3) (ports.ResultadoAltaBorradorReglasV3, error)
	ConsultarExacta(context.Context, app.CredencialesGobiernoV3, app.PeticionConsultaExactaV3) (ports.ResultadoConsultaGobiernoReglasV3, error)
	RecuperarRecibo(context.Context, app.CredencialesGobiernoV3, app.PeticionRecuperarReciboV3) (ports.ResultadoRecuperacionGobiernoReglasV3, error)
}

// El handler traduce datos de negocio. La autoridad obtiene actor y perfil
// del contexto sellado de la frontera; ninguno se deserializa del formulario.
type HandlerGobiernoReglasBaremoV3 struct {
	autoridad    AutoridadGobiernoReglasBaremoV3
	operador     OperadorGobiernoReglasBaremoV3
	auditar      AuditarRechazoSesionGobiernoReglasV3
	auditarError AuditarErrorGobiernoReglasV3
}

func NuevoHandlerGobiernoReglasBaremoV3(a AutoridadGobiernoReglasBaremoV3, o OperadorGobiernoReglasBaremoV3, auditar AuditarRechazoSesionGobiernoReglasV3, errores ...AuditarErrorGobiernoReglasV3) (*HandlerGobiernoReglasBaremoV3, error) {
	if dependenciaNula(a) || dependenciaNula(o) || auditar == nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	if len(errores) > 1 || (len(errores) == 1 && errores[0] == nil) {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	h := &HandlerGobiernoReglasBaremoV3{autoridad: a, operador: o, auditar: auditar}
	if len(errores) == 1 {
		h.auditarError = errores[0]
	}
	return h, nil
}

func (h *HandlerGobiernoReglasBaremoV3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	aplicarCabeceras(w)
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" ||
		r.RequestURI != r.URL.Path || r.URL.EscapedPath() != r.URL.Path ||
		(r.URL.Path != RutaAltaGobiernoReglasBaremoV3 && r.URL.Path != RutaConsultaGobiernoReglasBaremoV3 && r.URL.Path != RutaRecuperarGobiernoReglasBaremoV3) {
		responderGobiernoReglasV3(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderGobiernoReglasV3(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.operador) || h.auditar == nil {
		responderGobiernoReglasV3(w, http.StatusServiceUnavailable, "gobierno_reglas_v3_no_disponible")
		return
	}
	credenciales, err := h.autoridad.Credenciales(r.Context())
	if err != nil {
		if errors.Is(err, app.ErrGobiernoV3NoAutenticado) || errors.Is(err, app.ErrGobiernoV3Prohibido) {
			if errAudit := h.auditar(r.Context(), r.URL.Path, err); errAudit != nil {
				errorGobiernoHTTPV3(w, app.ErrGobiernoV3NoDisponible)
				return
			}
		}
		errorGobiernoHTTPV3(w, err)
		return
	}
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || r.Header.Get("Content-Encoding") != "" {
		if h.auditarError != nil && h.auditarError(r.Context(), app.ErrGobiernoV3PeticionInvalida) != nil {
			errorGobiernoHTTPV3(w, app.ErrGobiernoV3NoDisponible)
		} else {
			responderGobiernoReglasV3(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		}
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximoEntradaGobiernoReglasV3)
	defer r.Body.Close()
	entrada, err := leerEntradaGobiernoReglasV3(r.Body, r.URL.Path)
	if err != nil {
		h.errorHTTP(w, r, err)
		return
	}
	var salida any
	estado := http.StatusOK
	switch r.URL.Path {
	case RutaAltaGobiernoReglasBaremoV3:
		p, err := entrada.alta()
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		resultado, err := h.operador.GuardarAltaBorrador(r.Context(), credenciales, p)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		salida, err = salidaAltaGobiernoReglasV3(resultado)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		if !resultado.Replay {
			estado = http.StatusCreated
		}
	case RutaConsultaGobiernoReglasBaremoV3:
		p, err := entrada.consulta()
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		resultado, err := h.operador.ConsultarExacta(r.Context(), credenciales, p)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		salida, err = salidaConsultaGobiernoReglasV3(resultado, p.Selector)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
	case RutaRecuperarGobiernoReglasBaremoV3:
		p, err := entrada.recuperacion()
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		resultado, err := h.operador.RecuperarRecibo(r.Context(), credenciales, p)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
		salida, err = salidaRecuperacionGobiernoReglasV3(resultado)
		if err != nil {
			h.errorHTTP(w, r, err)
			return
		}
	}
	canon, err := json.Marshal(salida)
	if err != nil {
		h.errorHTTP(w, r, ports.ErrConfirmacionReglasBaremoInvalida)
		return
	}
	w.WriteHeader(estado)
	_, _ = w.Write(canon)
}

func (h *HandlerGobiernoReglasBaremoV3) errorHTTP(w http.ResponseWriter, r *http.Request, err error) {
	if h.auditarError != nil && h.auditarError(r.Context(), err) != nil {
		err = app.ErrGobiernoV3NoDisponible
	}
	errorGobiernoHTTPV3(w, err)
}

func errorGobiernoHTTPV3(w http.ResponseWriter, err error) {
	estado, codigo := http.StatusServiceUnavailable, "gobierno_reglas_v3_no_disponible"
	var grande *http.MaxBytesError
	switch {
	case errors.Is(err, app.ErrGobiernoV3NoAutenticado):
		estado, codigo = http.StatusUnauthorized, "gobierno_reglas_v3_no_autenticado"
	case errors.Is(err, app.ErrGobiernoV3Prohibido):
		estado, codigo = http.StatusForbidden, "gobierno_reglas_v3_prohibido"
	case errors.As(err, &grande):
		estado, codigo = http.StatusRequestEntityTooLarge, "solicitud_demasiado_grande"
	case errors.Is(err, app.ErrGobiernoV3PeticionInvalida):
		estado, codigo = http.StatusBadRequest, "gobierno_reglas_v3_peticion_invalida"
	case errors.Is(err, ports.ErrClaveIdempotenciaReglasReutilizada), errors.Is(err, ports.ErrConflictoOCCReglasBaremo):
		estado, codigo = http.StatusConflict, "gobierno_reglas_v3_conflicto"
	case errors.Is(err, ports.ErrReglasBaremoNoEncontradas):
		estado, codigo = http.StatusNotFound, "gobierno_reglas_v3_no_encontrada"
	}
	responderGobiernoReglasV3(w, estado, codigo)
}

func responderGobiernoReglasV3(w http.ResponseWriter, estado int, codigo string) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}{Error: struct {
		Codigo string `json:"codigo"`
	}{codigo}})
}

var _ OperadorGobiernoReglasBaremoV3 = (*app.ServicioGobiernoV3)(nil)

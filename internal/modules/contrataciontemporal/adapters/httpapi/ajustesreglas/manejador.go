// Package ajustesreglas expone la edición CT por una ruta interna exacta.
// La frontera de identidad y red la compone bootstrap antes de registrar la
// ruta; este adaptador solo acepta un actor derivado de esa frontera.
package ajustesreglas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	Ruta         = "/api/vec/contratacion-temporal/reglas/ajustes"
	Esquema      = "vec.contratacion_temporal.reglas.ajustes.v1"
	maximoCuerpo = 32 << 10
)

type ResolverActor interface {
	ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error)
}

type Servicio interface {
	Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (app.Lectura, error)
	Publicar(context.Context, vecdomain.ContextoActor, app.Solicitud) (app.Resultado, error)
	Motivos() []app.Motivo
}

type Manejador struct {
	actor       ResolverActor
	servicio    Servicio
	soloLectura bool
}

func NuevoManejador(actor ResolverActor, servicio Servicio) (*Manejador, error) {
	if actor == nil || servicio == nil {
		return nil, app.ErrNoDisponible
	}
	return &Manejador{actor: actor, servicio: servicio}, nil
}

// NuevoManejadorSoloLectura permite consultar la edición sin activar el
// guardado mientras CT110 no conserva la instantánea del plazo en la misma
// transacción que inicia el tramo. El montaje decide cuándo existe esa
// dependencia durable; una petición no puede cambiar este modo.
func NuevoManejadorSoloLectura(actor ResolverActor, servicio Servicio) (*Manejador, error) {
	h, err := NuevoManejador(actor, servicio)
	if err != nil {
		return nil, err
	}
	h.soloLectura = true
	return h, nil
}

func (h *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || h.actor == nil || h.servicio == nil || r == nil || r.URL == nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.URL.Path != Ruta || r.URL.RawPath != "" || r.URL.EscapedPath() != Ruta || r.URL.ForceQuery ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" {
		fallo(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		fallo(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if cabecerasProhibidas(r.Header) || !origenPermitido(r) || r.Header.Get("Accept") != "application/json" {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if h.soloLectura && r.Method == http.MethodPost {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	actor, err := h.actor.ResolverContextoActor(r.Context())
	if errors.Is(err, app.ErrNoDisponible) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if err != nil || actor.Validar() != nil {
		fallo(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.Method == http.MethodGet {
		h.consultar(w, r, actor)
		return
	}
	h.publicar(w, r, actor)
}

func origenPermitido(r *http.Request) bool {
	origen := r.Header.Get("Origin")
	if origen == "" || origen == "https://"+r.Host {
		return true
	}
	return origen == "http://"+r.Host &&
		(r.Host == "localhost" || strings.HasPrefix(r.Host, "localhost:") ||
			r.Host == "127.0.0.1" || strings.HasPrefix(r.Host, "127.0.0.1:"))
}

func (h *Manejador) consultar(w http.ResponseWriter, r *http.Request, actor vecdomain.ContextoActor) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Body != nil && r.Body != http.NoBody {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	limite, antes, err := paginacion(r)
	if err != nil {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	lectura, err := h.servicio.Consultar(r.Context(), actor, limite, antes)
	if err != nil {
		falloServicio(w, err)
		return
	}
	estadoActivacion := lectura.Activacion.Estado
	if estadoActivacion != "activa" && estadoActivacion != "inactiva" && estadoActivacion != "sin_publicar" ||
		estadoActivacion != "activa" && (lectura.PuedeAjustar || len(lectura.Reglas) != 0) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	version := 0
	if lectura.Vigente != nil {
		version = lectura.Vigente.Version
	}
	if lectura.Programados == nil {
		lectura.Programados = []app.VersionProgramada{}
	}
	puedeAjustar := lectura.PuedeAjustar && !h.soloLectura && estadoActivacion == "activa"
	if lectura.Historial == nil {
		lectura.Historial = []app.CambioHistorico{}
	}
	reglasVista := make([]reglaVista, 0, len(lectura.Reglas))
	for _, regla := range lectura.Reglas {
		if regla.Edicion == nil && !regla.AjusteNoAplicable {
			continue
		}
		reglasVista = append(reglasVista, vistaRegla(regla))
	}
	responder(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": Esquema, "catalogo_id": reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal),
		"activacion":       map[string]string{"estado": estadoActivacion},
		"version_esperada": version, "puede_ajustar": puedeAjustar,
		"cabeza":      versionAjustesVista(lectura.Vigente, lectura.CabezaPublicadaEn),
		"vigente_hoy": versionAjustesVista(lectura.VigenteHoy, lectura.VigentePublicadaEn),
		"programados": lectura.Programados,
		"reglas":      reglasVista, "motivos": h.servicio.Motivos(),
		"historial": lectura.Historial, "hay_mas": lectura.HayMas,
		"hay_mas_programados": lectura.HayMasProgramados,
	}})
}

func versionAjustesVista(v *reglas.VersionAjustes, publicadaEn time.Time) any {
	if v == nil {
		return nil
	}
	return map[string]any{"version": v.Version, "huella_sha256": v.HuellaSHA256,
		"ajustes": v.Ajustes, "vigente_desde": v.VigenteDesde, "publicada_en": publicadaEn}
}

func (h *Manejador) publicar(w http.ResponseWriter, r *http.Request, actor vecdomain.ContextoActor) {
	if r.URL.RawQuery != "" || r.ContentLength < 1 || r.ContentLength > maximoCuerpo ||
		len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" ||
		r.Body == nil || r.Body == http.NoBody {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpo+1))
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > maximoCuerpo {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if err := jsonSinDuplicados(cuerpo); err != nil {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	var solicitud app.Solicitud
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	d.DisallowUnknownFields()
	if d.Decode(&solicitud) != nil || d.Decode(&struct{}{}) != io.EOF {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	resultado, err := h.servicio.Publicar(r.Context(), actor, solicitud)
	if err != nil {
		falloServicio(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Replay {
		estado = http.StatusOK
	}
	responder(w, estado, map[string]any{"data": map[string]any{
		"esquema": Esquema, "recibo": resultado.Recibo, "replay": resultado.Replay,
	}})
}

var errFormaSolicitud = errors.New("forma de solicitud invalida")

func paginacion(r *http.Request) (int, *int64, error) {
	const limiteInicial = 20
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return 0, nil, fmt.Errorf("leer paginacion: %w", err)
	}
	if len(q) > 2 {
		return 0, nil, errFormaSolicitud
	}
	limite := limiteInicial
	var antes *int64
	for clave, valores := range q {
		if len(valores) != 1 || valores[0] == "" {
			return 0, nil, errFormaSolicitud
		}
		switch clave {
		case "limite":
			n, err := strconv.Atoi(valores[0])
			if err != nil {
				return 0, nil, fmt.Errorf("leer limite: %w", err)
			}
			if n < 1 || n > 50 || strconv.Itoa(n) != valores[0] {
				return 0, nil, errFormaSolicitud
			}
			limite = n
		case "antes_de_version":
			n, err := strconv.ParseInt(valores[0], 10, 64)
			if err != nil {
				return 0, nil, fmt.Errorf("leer cursor: %w", err)
			}
			if n < 2 || n > 10_000_000 || strconv.FormatInt(n, 10) != valores[0] {
				return 0, nil, errFormaSolicitud
			}
			antes = &n
		default:
			return 0, nil, errFormaSolicitud
		}
	}
	return limite, antes, nil
}

type reglaVista struct {
	Clave             string            `json:"clave"`
	Etiqueta          string            `json:"etiqueta"`
	Unidad            string            `json:"unidad,omitempty"`
	Cantidad          *int              `json:"cantidad,omitempty"`
	Computo           string            `json:"computo,omitempty"`
	Valores           map[string]string `json:"valores,omitempty"`
	Edicion           *edicionVista     `json:"edicion,omitempty"`
	Ajuste            *ajusteVista      `json:"ajuste,omitempty"`
	AjusteNoAplicable bool              `json:"ajuste_no_aplicable"`
}

type edicionVista struct {
	Campos          []string         `json:"campos"`
	OpcionesUnidad  []reglas.Unidad  `json:"opciones_unidad"`
	OpcionesComputo []reglas.Computo `json:"opciones_computo"`
	CantidadMinima  int              `json:"cantidad_minima"`
	CantidadMaxima  int              `json:"cantidad_maxima"`
}

type ajusteVista struct {
	Version      int               `json:"version"`
	VigenteDesde string            `json:"vigente_desde"`
	Campos       map[string]string `json:"campos"`
}

func vistaRegla(r reglas.Regla) reglaVista {
	v := reglaVista{Clave: r.Clave, Etiqueta: r.Etiqueta, AjusteNoAplicable: r.AjusteNoAplicable}
	if r.Edicion != nil {
		v.Edicion = &edicionVista{Campos: append([]string{}, r.Edicion.Campos...),
			OpcionesUnidad:  append([]reglas.Unidad{}, r.Edicion.OpcionesUnidad...),
			OpcionesComputo: append([]reglas.Computo{}, r.Edicion.OpcionesComputo...),
			CantidadMinima:  r.Edicion.CantidadMinima,
			CantidadMaxima:  r.Edicion.CantidadMaxima}
	}
	if r.AjusteNoAplicable {
		// La base ya no es el valor efectivo. Ni siquiera se ofrece como 0.
		return v
	}
	v.Unidad, v.Cantidad, v.Computo = string(r.Unidad), &r.Cantidad, string(r.Computo)
	if r.Ajuste != nil {
		v.Ajuste = &ajusteVista{Version: r.Ajuste.Version,
			VigenteDesde: r.Ajuste.VigenteDesde.UTC().Format("2006-01-02T15:04:05.000000Z"),
			Campos:       r.Ajuste.Campos}
	}
	v.Valores = make(map[string]string, len(r.Edicion.Campos))
	for _, campo := range r.Edicion.Campos {
		switch campo {
		case reglas.CampoCantidad:
			v.Valores[campo] = strconv.Itoa(r.Cantidad)
		case reglas.CampoUnidad:
			v.Valores[campo] = string(r.Unidad)
		case reglas.CampoComputo:
			v.Valores[campo] = string(r.Computo)
		default:
			v.Valores[campo] = r.Atributos[campo]
		}
	}
	return v
}

func falloServicio(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
		fallo(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, app.ErrConflicto), errors.Is(err, reglas.ErrAjustesConflicto):
		fallo(w, http.StatusConflict, "version_o_clave_en_conflicto")
	case errors.Is(err, app.ErrEntradaInvalida), errors.Is(err, reglas.ErrAjusteInvalido):
		fallo(w, http.StatusUnprocessableEntity, "ajuste_invalido")
	default:
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}

func fallo(w http.ResponseWriter, estado int, codigo string) {
	responder(w, estado, map[string]any{"error": map[string]string{"codigo": codigo}})
}

func responder(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(cuerpo)
}

func cabecerasProhibidas(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		if n == "cookie" || n == "authorization" || n == "proxy-authorization" ||
			n == "forwarded" || n == "remote-user" || n == "idempotency-key" ||
			n == "x-http-method-override" || strings.HasPrefix(n, "x-forwarded-") ||
			strings.HasPrefix(n, "x-auth-") || strings.HasPrefix(n, "x-vec-") ||
			strings.Contains(n, "role") {
			return true
		}
	}
	return false
}

func jsonSinDuplicados(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	if err := valorUnico(d); err != nil {
		return err
	}
	_, err := d.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("leer resto JSON: %w", err)
	}
	return errFormaSolicitud
}

func valorUnico(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return fmt.Errorf("leer valor JSON: %w", err)
	}
	delimitador, esDelimitador := t.(json.Delim)
	if !esDelimitador {
		return nil
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]bool)
		for d.More() {
			clave, err := d.Token()
			if err != nil {
				return fmt.Errorf("leer clave JSON: %w", err)
			}
			k, ok := clave.(string)
			if !ok || vistas[k] {
				return errFormaSolicitud
			}
			if err := valorUnico(d); err != nil {
				return err
			}
			vistas[k] = true
		}
	case '[':
		for d.More() {
			if err := valorUnico(d); err != nil {
				return err
			}
		}
	default:
		return errFormaSolicitud
	}
	final, err := d.Token()
	if err != nil {
		return fmt.Errorf("leer cierre JSON: %w", err)
	}
	if final != json.Delim(map[json.Delim]rune{'{': '}', '[': ']'}[delimitador]) {
		return errFormaSolicitud
	}
	return nil
}

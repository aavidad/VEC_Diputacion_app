package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// Rutas del segundo corte de la persona empleada. La identidad, el empleado
// y la autorización V3 los aporta el resolver desde la frontera; el cliente
// sólo envía fechas, horas, permiso y la clave de su operación.
const (
	RutaConsultarMovimientosPropios = "/api/interna/cronos/movimientos/propio"
	RutaSolicitarCorreccionPropia   = "/api/interna/cronos/correcciones/propias"
	RutaConsultarPermisosPropios    = "/api/interna/cronos/permisos/propio"
	RutaSolicitarPermisoPropio      = "/api/interna/cronos/permisos/solicitudes"
)

type ResolverConsultaMovimientosPropios interface {
	ResolverConsultaMovimientosPropios(*http.Request) (ports.OrdenConsultaMovimientos, error)
}

type ResolverSolicitudCorreccionPropia interface {
	ResolverSolicitudCorreccionPropia(*http.Request) (ports.OrdenConsumoCorreccion, error)
}

type ResolverPermisosPropios interface {
	ResolverPermisosPropios(*http.Request) (ports.OrdenPermisosPropios, error)
}

// CasoUsoSolicitarOlvido es la única parte de las correcciones que ofrece
// este corte a la persona empleada.
type CasoUsoSolicitarOlvido interface {
	SolicitarOlvido(context.Context, ports.OrdenConsumoCorreccion, ports.SolicitudOlvidoMarcaje) (ports.ReciboCorreccion, error)
}

func cabecerasJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// decodificarCadenas lee un único objeto JSON de hasta 4 KiB cuyos campos son
// todos cadenas conocidas, sin duplicados ni campos extra.
func decodificarCadenas(w http.ResponseWriter, r *http.Request, obligatorios, opcionales []string) (map[string]string, bool) {
	if r.Header.Get("Content-Type") != "application/json" || r.Body == nil {
		return nil, false
	}
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	apertura, err := dec.Token()
	if err != nil || apertura != json.Delim('{') {
		return nil, false
	}
	admitidos := make(map[string]bool, len(obligatorios)+len(opcionales))
	for _, c := range append(append([]string{}, obligatorios...), opcionales...) {
		admitidos[c] = true
	}
	valores := make(map[string]string, len(admitidos))
	for dec.More() {
		token, err := dec.Token()
		nombre, ok := token.(string)
		if err != nil || !ok || !admitidos[nombre] {
			return nil, false
		}
		if _, repetido := valores[nombre]; repetido {
			return nil, false
		}
		var v string
		if dec.Decode(&v) != nil || len(v) > 256 {
			return nil, false
		}
		valores[nombre] = v
	}
	if cierre, err := dec.Token(); err != nil || cierre != json.Delim('}') {
		return nil, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	for _, c := range obligatorios {
		if _, ok := valores[c]; !ok {
			return nil, false
		}
	}
	return valores, true
}

func soloMetodo(w http.ResponseWriter, r *http.Request, metodo string) bool {
	if r.Method != metodo {
		w.Header().Set("Allow", metodo)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return false
	}
	return true
}

// ---- Movimientos ----

type ManejadorMovimientosPropios struct {
	resolver ResolverConsultaMovimientosPropios
	casoUso  ports.CasoUsoConsultarMovimientos
}

func NuevoManejadorMovimientosPropios(casoUso ports.CasoUsoConsultarMovimientos, resolver ResolverConsultaMovimientosPropios) (*ManejadorMovimientosPropios, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorMovimientosPropios{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorMovimientosPropios) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.Path != RutaConsultarMovimientosPropios || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	periodo, desde, hasta, err := parametrosSaldo(r.URL.RawQuery)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverConsultaMovimientosPropios(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	resultado, err := m.casoUso.ConsultarMovimientos(r.Context(), orden, periodo, desde, hasta)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaSaldoInvalida) || errors.Is(err, ports.ErrSolicitudCronosInvalida) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resultado)
}

// ---- Corrección de un olvido de marcaje ----

type ManejadorCorreccionPropia struct {
	resolver ResolverSolicitudCorreccionPropia
	casoUso  CasoUsoSolicitarOlvido
}

func NuevoManejadorCorreccionPropia(casoUso CasoUsoSolicitarOlvido, resolver ResolverSolicitudCorreccionPropia) (*ManejadorCorreccionPropia, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorCorreccionPropia{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorCorreccionPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.Path != RutaSolicitarCorreccionPropia || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	v, ok := decodificarCadenas(w, r, []string{"clave_operacion", "movimiento", "fecha_civil", "hora_pretendida"}, []string{"marcaje_original_ref"})
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverSolicitudCorreccionPropia(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.SolicitarOlvido(r.Context(), orden, ports.SolicitudOlvidoMarcaje{
		ClaveOperacion: v["clave_operacion"], MarcajeOriginalRef: v["marcaje_original_ref"], HuecoDeclarado: v["marcaje_original_ref"] == "",
		Movimiento: domain.PunchKind(v["movimiento"]), FechaCivil: v["fecha_civil"], HoraPretendida: v["hora_pretendida"],
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCorreccionInvalida), errors.Is(err, ports.ErrSolicitudCronosInvalida):
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		case errors.Is(err, ports.ErrCorreccionEnConflicto):
			errorJSON(w, http.StatusConflict, "conflicto")
		default:
			responderErrorAccesoCronos(w, err)
		}
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": map[string]any{
		"solicitud_ref": recibo.SolicitudRef, "actuacion_ref": recibo.ActuacionRef, "recibo_ref": recibo.ReciboRef,
		"estado": recibo.Estado, "version": recibo.Version, "instante_utc": recibo.InstanteUTC, "replay": recibo.Replay,
	}})
}

// ---- Permisos ----

type ManejadorPermisosPropios struct {
	resolver ResolverPermisosPropios
	casoUso  ports.CasoUsoPermisosPropios
}

func NuevoManejadorPermisosPropios(casoUso ports.CasoUsoPermisosPropios, resolver ResolverPermisosPropios) (*ManejadorPermisosPropios, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorPermisosPropios{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorPermisosPropios) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" || (r.URL.Path != RutaConsultarPermisosPropios && r.URL.Path != RutaSolicitarPermisoPropio) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaSolicitarPermisoPropio {
		m.solicitar(w, r)
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	anio, ok := parametroAnio(r.URL.RawQuery)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverPermisosPropios(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	resultado, err := m.casoUso.ConsultarPermisosPropios(r.Context(), orden, anio)
	if err != nil {
		if errors.Is(err, ports.ErrSolicitudCronosInvalida) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resultado)
}

func (m *ManejadorPermisosPropios) solicitar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	v, ok := decodificarCadenas(w, r, []string{"clave_operacion", "permiso_ref", "desde", "hasta"}, []string{"hora_inicio", "hora_fin"})
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverPermisosPropios(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.SolicitarPermisoPropio(r.Context(), orden, ports.PeticionPermisoPropio{
		PermisoRef: v["permiso_ref"], Desde: v["desde"], Hasta: v["hasta"], HoraInicio: v["hora_inicio"], HoraFin: v["hora_fin"],
		ClaveOperacion: v["clave_operacion"],
	})
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrSolicitudCronosInvalida):
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		case errors.Is(err, ports.ErrClaveOperacionEnConflicto):
			errorJSON(w, http.StatusConflict, "conflicto")
		case errors.Is(err, ports.ErrPermisoNoSolicitable):
			errorJSON(w, http.StatusUnprocessableEntity, "permiso_no_solicitable")
		case errors.Is(err, ports.ErrCalendarioNoPublicado):
			errorJSON(w, http.StatusUnprocessableEntity, "calendario_no_publicado")
		case errors.Is(err, ports.ErrPermisoFueraDeLimites):
			errorJSON(w, http.StatusUnprocessableEntity, "fuera_de_limites")
		case errors.Is(err, ports.ErrPermisoSolapado):
			errorJSON(w, http.StatusUnprocessableEntity, "solapado")
		default:
			responderErrorAccesoCronos(w, err)
		}
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}

// parametroAnio admite la consulta vacía (año en curso) o anio=AAAA.
func parametroAnio(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	if len(raw) > 16 {
		return 0, false
	}
	valores, err := url.ParseQuery(raw)
	if err != nil || len(valores) != 1 || len(valores["anio"]) != 1 || len(valores.Get("anio")) != 4 {
		return 0, false
	}
	anio, err := strconv.Atoi(valores.Get("anio"))
	if err != nil || anio < 2000 || anio > 2100 {
		return 0, false
	}
	return anio, true
}

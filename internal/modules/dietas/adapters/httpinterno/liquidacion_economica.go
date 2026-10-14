package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	dietasdomain "vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaLiquidacionesCircuito = RutaCircuito + "/"
	RutaLiquidaciones         = RutaBorradores + "/"
	maximoCuerpoLiquidacion   = 1024 * 1024
	maximoRevisiones          = dietasdomain.MaximoLineasLiquidacionPublicada
)

type RegistradorLiquidacionEconomicaHTTP interface {
	RegistrarPreparada(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudLiquidacionEconomica) (*dietasdomain.LiquidacionEconomica, error)
}

type ManejadorLiquidacionEconomica struct {
	preparador dietasports.PreparadorLiquidacionEconomicaHTTP
	identidad  dietasports.ResolutorRegistroLiquidacionEconomica
	registro   RegistradorLiquidacionEconomicaHTTP
	captura    dietasports.CapturadorLecturaLiquidacionEconomica
	lector     dietasports.LectorLiquidacionEconomica
}

func NuevoManejadorLiquidacionEconomica(preparador dietasports.PreparadorLiquidacionEconomicaHTTP, identidad dietasports.ResolutorRegistroLiquidacionEconomica, registro RegistradorLiquidacionEconomicaHTTP, captura dietasports.CapturadorLecturaLiquidacionEconomica, lector dietasports.LectorLiquidacionEconomica) (*ManejadorLiquidacionEconomica, error) {
	if dependenciaNula(preparador) || dependenciaNula(identidad) || dependenciaNula(registro) || dependenciaNula(captura) || dependenciaNula(lector) {
		return nil, ErrManejadorNoDisponible
	}
	return &ManejadorLiquidacionEconomica{preparador: preparador, identidad: identidad, registro: registro, captura: captura, lector: lector}, nil
}

func (m *ManejadorLiquidacionEconomica) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil || m == nil || dependenciaNula(m.preparador) || dependenciaNula(m.identidad) || dependenciaNula(m.registro) || dependenciaNula(m.captura) || dependenciaNula(m.lector) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Vec-Actor") != "" || r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if r.URL == nil || r.URL.EscapedPath() != r.URL.Path {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	ref, accion, version, ok := rutaLiquidacionEconomica(r.URL.Path)
	if !ok {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	switch accion {
	case "propuesta":
		m.propuesta(w, r, ref)
	case "registro":
		m.registrar(w, r, ref)
	case "lectura":
		m.leer(w, r, ref, version)
	default:
		responderError(w, http.StatusNotFound, "no_encontrada")
	}
}

func rutaLiquidacionEconomica(path string) (ref, accion string, version uint64, ok bool) {
	for _, prefijo := range []string{RutaLiquidacionesCircuito, RutaLiquidaciones} {
		resto, encontrado := strings.CutPrefix(path, prefijo)
		if !encontrado {
			continue
		}
		partes := strings.Split(resto, "/")
		if len(partes) < 2 || !referenciaCircuitoHTTP.MatchString(partes[0]) || partes[1] != "liquidaciones" {
			return "", "", 0, false
		}
		if len(partes) == 2 && prefijo == RutaLiquidacionesCircuito {
			return partes[0], "registro", 0, true
		}
		if len(partes) != 3 {
			return "", "", 0, false
		}
		if partes[2] == "propuesta" && prefijo == RutaLiquidacionesCircuito {
			return partes[0], "propuesta", 0, true
		}
		if partes[2] == "actual" {
			return partes[0], "lectura", 0, true
		}
		v, valido := versionLiquidacionHTTP(partes[2])
		if valido && v >= 2 {
			return partes[0], "lectura", v, true
		}
		return "", "", 0, false
	}
	return "", "", 0, false
}

func versionLiquidacionHTTP(texto string) (uint64, bool) {
	if len(texto) == 0 || len(texto) > 18 {
		return 0, false
	}
	v, err := strconv.ParseUint(texto, 10, 64)
	return v, err == nil && v > 0 && strconv.FormatUint(v, 10) == texto
}

func versionPropuestaLiquidacion(u *url.URL) (uint64, bool) {
	if u == nil || u.ForceQuery {
		return 0, false
	}
	valores, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(valores) != 1 || len(valores["version_esperada"]) != 1 {
		return 0, false
	}
	return versionLiquidacionHTTP(valores.Get("version_esperada"))
}

func (m *ManejadorLiquidacionEconomica) propuesta(w http.ResponseWriter, r *http.Request, ref string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	version, valida := versionPropuestaLiquidacion(r.URL)
	if !valida || !lecturaSinCuerpo(r) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	preparacion, err := m.preparador.PrepararLiquidacionEconomica(r.Context(), ref, version, nil)
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	if preparacion == nil || preparacion.Instantanea().ComisionRef != ref ||
		strconv.FormatInt(preparacion.Instantanea().ComisionVersion, 10) != strconv.FormatUint(version, 10) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responderJSON(w, http.StatusOK, struct {
		Propuesta dietasdomain.InstantaneaLiquidacionPublicada `json:"propuesta"`
	}{preparacion.Instantanea()})
}

type solicitudRegistroLiquidacionJSON struct {
	VersionEsperada   uint64                                  `json:"version_esperada"`
	ClaveIdempotencia string                                  `json:"clave_idempotencia"`
	Revisiones        []dietasdomain.RevisionLineaLiquidacion `json:"revisiones"`
}

func decodificarSolicitudLiquidacion(w http.ResponseWriter, r *http.Request, destino *solicitudRegistroLiquidacionJSON) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength < 1 || r.ContentLength > maximoCuerpoLiquidacion || len(r.TransferEncoding) != 0 ||
		r.Header.Get("Content-Type") != "application/json; charset=utf-8" || r.Header.Get("Accept") != "application/json" {
		return dietasdomain.ErrLiquidacionPublicada
	}
	decodificador := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoLiquidacion))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return dietasdomain.ErrLiquidacionPublicada
	}
	return nil
}

func (m *ManejadorLiquidacionEconomica) registrar(w http.ResponseWriter, r *http.Request, ref string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	var entrada solicitudRegistroLiquidacionJSON
	if err := decodificarSolicitudLiquidacion(w, r, &entrada); err != nil || entrada.VersionEsperada == 0 || entrada.VersionEsperada > 999999999999999999 ||
		!claveCircuitoHTTP.MatchString(entrada.ClaveIdempotencia) || len(entrada.Revisiones) == 0 || len(entrada.Revisiones) > maximoRevisiones {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	preparacion, err := m.preparador.PrepararLiquidacionEconomica(r.Context(), ref, entrada.VersionEsperada, entrada.Revisiones)
	if err != nil {
		if recuperada, ok := m.recuperarTrasPreparacionFallida(r.Context(), ref, entrada); ok {
			responderJSON(w, http.StatusOK, struct {
				Liquidacion dietasdomain.InstantaneaLiquidacionEconomica `json:"liquidacion"`
			}{recuperada})
			return
		}
		responderErrorLiquidacion(w, err)
		return
	}
	if preparacion == nil || preparacion.Instantanea().ComisionRef != ref || preparacion.Instantanea().ComisionVersion != int64(entrada.VersionEsperada) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	solicitud := dietasports.SolicitudLiquidacionEconomica{Decision: dietasports.SolicitudDecisionCircuito{
		Referencia: ref, Etapa: dietasdomain.EtapaLiquidacion, Decision: dietasdomain.DecisionAprobar,
		ClaveIdempotencia: entrada.ClaveIdempotencia, VersionEsperada: entrada.VersionEsperada}, Preparacion: preparacion}
	identidad, err := m.identidad.ResolverRegistroLiquidacionEconomica(r.Context(), solicitud)
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	solicitud.Decision.UnidadRef = identidad.UnidadCompetenciaRef
	liquidacion, err := m.registro.RegistrarPreparada(r.Context(), identidad, solicitud)
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	if liquidacion == nil {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responderJSON(w, http.StatusOK, struct {
		Liquidacion dietasdomain.InstantaneaLiquidacionEconomica `json:"liquidacion"`
	}{liquidacion.Instantanea()})
}

// La primera operación puede haber confirmado el acto aunque se perdiera su
// respuesta. La cola ya avanzada impide prepararla otra vez; la lectura nominal
// actual recupera únicamente el recibo durable de esa operación exacta.
func (m *ManejadorLiquidacionEconomica) recuperarTrasPreparacionFallida(ctx context.Context, ref string, entrada solicitudRegistroLiquidacionJSON) (dietasdomain.InstantaneaLiquidacionEconomica, bool) {
	var vacia dietasdomain.InstantaneaLiquidacionEconomica
	if ctx.Err() != nil {
		return vacia, false
	}
	captura, err := m.captura.CapturarLecturaLiquidacionEconomica(ctx, ref)
	if err != nil {
		return vacia, false
	}
	actor, err := vecdomain.RehidratarContextoActorVinculadoV2(captura.ContextoCanonico)
	if err != nil {
		return vacia, false
	}
	resultado, err := m.lector.Consultar(ctx, captura, dietasports.ConsultaLiquidacionEconomica{
		Referencia: ref, Version: entrada.VersionEsperada + 1,
	})
	if err != nil {
		return vacia, false
	}
	liquidacion, err := dietasdomain.RecuperarLiquidacionEconomica(resultado.Liquidacion)
	if err != nil {
		return vacia, false
	}
	s := liquidacion.Instantanea()
	if s.Economia.ComisionRef != ref || strconv.FormatInt(s.Economia.ComisionVersion, 10) != strconv.FormatUint(entrada.VersionEsperada, 10) ||
		s.Constancia.VersionEntrada != entrada.VersionEsperada || s.Constancia.VersionResultante != entrada.VersionEsperada+1 ||
		s.Constancia.DecisorPersonaRef != actor.PersonaRef || s.Constancia.ClaveIdempotencia != entrada.ClaveIdempotencia ||
		!mismasRevisionesLiquidacion(s.Economia.Lineas, entrada.Revisiones) {
		return vacia, false
	}
	return s, true
}

func mismasRevisionesLiquidacion(lineas []dietasdomain.LineaLiquidacionPropuesta, revisiones []dietasdomain.RevisionLineaLiquidacion) bool {
	if len(lineas) != len(revisiones) || len(lineas) == 0 {
		return false
	}
	vistas := make([]bool, len(lineas))
	for _, r := range revisiones {
		if r.Indice < 0 || r.Indice >= len(lineas) || vistas[r.Indice] {
			return false
		}
		vistas[r.Indice] = true
		l := lineas[r.Indice]
		if l.Indice != r.Indice || l.ReglaRef != r.ReglaRef ||
			l.ReconocidoPropuestoCentimos != r.ReconocidoPropuestoCentimos || l.MotivoCodigo != r.MotivoCodigo {
			return false
		}
	}
	return true
}

func (m *ManejadorLiquidacionEconomica) leer(w http.ResponseWriter, r *http.Request, ref string, version uint64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !lecturaSinCuerpo(r) || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	captura, err := m.captura.CapturarLecturaLiquidacionEconomica(r.Context(), ref)
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	resultado, err := m.lector.Consultar(r.Context(), captura, dietasports.ConsultaLiquidacionEconomica{Referencia: ref, Version: version})
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	if resultado.Liquidacion.Economia.ComisionRef != ref || (version != 0 && resultado.Liquidacion.Constancia.VersionResultante != version) ||
		resultado.Liquidacion.SnapshotSHA256 == "" {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responderJSON(w, http.StatusOK, struct {
		Liquidacion dietasdomain.InstantaneaLiquidacionEconomica `json:"liquidacion"`
	}{resultado.Liquidacion})
}

func responderErrorLiquidacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente):
		responderError(w, http.StatusForbidden, "competencia_sin_fuente")
	case errors.Is(err, dietasports.ErrAccesoCircuitoDenegado), errors.Is(err, dietasports.ErrAccesoBorradorDenegado):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, dietasports.ErrComisionNoEncontrada):
		responderError(w, http.StatusNotFound, "no_encontrada")
	case errors.Is(err, dietasports.ErrConflictoIdempotencia):
		responderError(w, http.StatusConflict, "conflicto_idempotencia")
	case errors.Is(err, dietasports.ErrVersionComisionConflicto):
		responderError(w, http.StatusConflict, "conflicto_version")
	case errors.Is(err, dietasports.ErrEstadoCircuitoConflicto):
		responderError(w, http.StatusConflict, "conflicto_estado")
	case errors.Is(err, dietasports.ErrRelacionNoDisponible):
		responderError(w, http.StatusServiceUnavailable, "relacion_no_disponible")
	case errors.Is(err, dietasdomain.ErrAlojamientoLiquidacionPendiente), errors.Is(err, dietasdomain.ErrLiquidacionPublicada), errors.Is(err, dietasports.ErrDocumentoNoDisponible):
		responderError(w, http.StatusConflict, "documento_no_disponible")
	case errors.Is(err, dietasports.ErrResultadoCircuitoIncierto):
		responderError(w, http.StatusServiceUnavailable, "resultado_incierto")
	default:
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
	}
}

var _ http.Handler = (*ManejadorLiquidacionEconomica)(nil)

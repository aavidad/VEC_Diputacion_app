package httpinterno

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const RutaCircuito = "/api/vec/dietas/comisiones/circuito"

// RutaCircuitoCompetencias informa de qué bandejas puede abrir el actor según
// la fuente gobernada de competencia, o de que esa fuente no existe.
const RutaCircuitoCompetencias = RutaCircuito + "/competencias"

var referenciaCircuitoHTTP = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
var claveCircuitoHTTP = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)

type ManejadorCircuito struct {
	identidades dietasports.ResolutorIdentidadEfectivaCircuito
	casoUso     dietasapp.CasoUsoCircuitoComision
}

func NuevoManejadorCircuito(identidades dietasports.ResolutorIdentidadEfectivaCircuito, casoUso dietasapp.CasoUsoCircuitoComision) (*ManejadorCircuito, error) {
	if dependenciaNula(identidades) || dependenciaNula(casoUso) {
		return nil, ErrManejadorNoDisponible
	}
	return &ManejadorCircuito{identidades: identidades, casoUso: casoUso}, nil
}

func (m *ManejadorCircuito) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil || m == nil || dependenciaNula(m.identidades) || dependenciaNula(m.casoUso) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("X-Vec-Actor") != "" || r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if r.URL == nil || r.URL.EscapedPath() != r.URL.Path {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	switch r.URL.Path {
	case RutaCircuito:
		m.listar(w, r)
		return
	case RutaCircuitoCompetencias:
		m.competencias(w, r)
		return
	}
	resto, ok := strings.CutPrefix(r.URL.Path, RutaCircuito+"/")
	if !ok {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if ref, decision := strings.CutSuffix(resto, "/decisiones"); decision && referenciaCircuitoHTTP.MatchString(ref) {
		m.decidir(w, r, ref)
		return
	}
	if referenciaCircuitoHTTP.MatchString(resto) {
		m.documento(w, r, resto)
		return
	}
	responderError(w, http.StatusNotFound, "no_encontrada")
}

type decisionCircuitoJSON struct {
	Etapa             domain.EtapaCircuito    `json:"etapa"`
	Decision          domain.DecisionCircuito `json:"decision"`
	Motivo            string                  `json:"motivo"`
	ClaveIdempotencia string                  `json:"clave_idempotencia"`
	VersionEsperada   uint64                  `json:"version_esperada"`
}

func (m *ManejadorCircuito) decidir(w http.ResponseWriter, r *http.Request, ref string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	var e decisionCircuitoJSON
	if err := decodificarSolicitud(w, r, &e); err != nil || e.Etapa.EstadoPendiente() == "" ||
		(e.Decision != domain.DecisionAprobar && e.Decision != domain.DecisionDevolver) ||
		!claveCircuitoHTTP.MatchString(e.ClaveIdempotencia) || e.VersionEsperada == 0 || e.VersionEsperada > 999999999999999999 ||
		len(e.Motivo) > 600 || !domain.TextoSinBordes(e.Motivo) ||
		(e.Decision == domain.DecisionDevolver && len(e.Motivo) < 3) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	s := dietasports.SolicitudDecisionCircuito{Referencia: ref, Etapa: e.Etapa, Decision: e.Decision, Motivo: e.Motivo, ClaveIdempotencia: e.ClaveIdempotencia, VersionEsperada: e.VersionEsperada}
	identidad, err := m.identidades.ResolverIdentidadEfectivaCircuito(r.Context(), dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: s})
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	// La unidad procede exclusivamente del contexto/Personal resuelto y ligado
	// al material V3. Nunca existe un selector libre en el cuerpo HTTP.
	s.UnidadRef = identidad.UnidadCompetenciaRef
	resultado, err := m.casoUso.Decidir(r.Context(), identidad, s)
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Recibo.Repeticion {
		estado = http.StatusOK
	}
	responderJSON(w, estado, struct {
		Comision dietasports.VistaComisionCircuito `json:"comision"`
		Recibo   reciboBorradorJSON                `json:"recibo"`
	}{resultado.Comision, reciboBorradorJSON{Referencia: resultado.Recibo.Referencia, Version: resultado.Recibo.Version, RegistradoEn: resultado.Recibo.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), Repeticion: resultado.Recibo.Repeticion}})
}

func (m *ManejadorCircuito) listar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !lecturaSinCuerpo(r) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	consulta, err := consultaCircuitoURL(r.URL)
	if err != nil {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	identidad, err := m.identidades.ResolverIdentidadEfectivaCircuito(r.Context(), dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionListarBandeja, Consulta: consulta})
	if errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente) {
		// Sin fuente gobernada no hay bandeja que mostrar ni acción posible.
		responderJSON(w, http.StatusOK, bandejaCircuitoJSON{Items: []dietasports.VistaComisionCircuito{}, Competencia: dietasports.FuenteCompetenciaSinFuente})
		return
	}
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	consulta.UnidadRef = identidad.UnidadCompetenciaRef
	pagina, err := m.casoUso.ListarPendientes(r.Context(), identidad, consulta)
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	if pagina.Items == nil {
		pagina.Items = []dietasports.VistaComisionCircuito{}
	}
	responderJSON(w, http.StatusOK, bandejaCircuitoJSON{Items: pagina.Items, SiguienteCursor: pagina.SiguienteCursor, Competencia: dietasports.FuenteCompetenciaAcreditada})
}

type bandejaCircuitoJSON struct {
	Items           []dietasports.VistaComisionCircuito `json:"items"`
	SiguienteCursor string                              `json:"siguiente_cursor,omitempty"`
	Competencia     string                              `json:"competencia"`
}

func (m *ManejadorCircuito) competencias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !lecturaSinCuerpo(r) || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	estado, err := m.identidades.EstadoCompetenciasCircuito(r.Context())
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	if estado.Etapas == nil {
		estado.Etapas = []domain.EtapaCircuito{}
	}
	if estado.Fuente != dietasports.FuenteCompetenciaSinFuente && estado.Fuente != dietasports.FuenteCompetenciaAcreditada ||
		(estado.Fuente == dietasports.FuenteCompetenciaSinFuente && len(estado.Etapas) != 0) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	for _, etapa := range estado.Etapas {
		if etapa.EstadoPendiente() == "" {
			responderError(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
	}
	responderJSON(w, http.StatusOK, estado)
}

func (m *ManejadorCircuito) documento(w http.ResponseWriter, r *http.Request, ref string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !lecturaSinCuerpo(r) || r.URL.ForceQuery {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	v, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(v) != 1 || len(v["etapa"]) != 1 || domain.EtapaCircuito(v.Get("etapa")).EstadoPendiente() == "" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	s := dietasports.SolicitudDocumentoCircuito{Referencia: ref, Etapa: domain.EtapaCircuito(v.Get("etapa"))}
	identidad, err := m.identidades.ResolverIdentidadEfectivaCircuito(r.Context(), dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: s})
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	s.UnidadRef = identidad.UnidadCompetenciaRef
	documento, err := m.casoUso.ConsultarDocumento(r.Context(), identidad, s)
	if err != nil {
		responderErrorCircuito(w, err)
		return
	}
	if documento.CodigosRuta == nil {
		documento.CodigosRuta = []string{}
	}
	responderJSON(w, http.StatusOK, struct {
		Comision dietasports.DocumentoCircuito `json:"comision"`
	}{documento})
}

func consultaCircuitoURL(u *url.URL) (dietasports.ConsultaBandejaCircuito, error) {
	var cero dietasports.ConsultaBandejaCircuito
	if u == nil || u.ForceQuery {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	v, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return cero, err
	}
	for k, xs := range v {
		if (k != "etapa" && k != "fecha_desde" && k != "fecha_hasta" && k != "limit" && k != "cursor") || len(xs) != 1 {
			return cero, domain.ErrDecisionCircuitoInvalida
		}
	}
	q := dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaCircuito(v.Get("etapa")), FechaDesde: v.Get("fecha_desde"), FechaHasta: v.Get("fecha_hasta"), Cursor: v.Get("cursor"), Limite: 20}
	if texto := v.Get("limit"); texto != "" {
		q.Limite, err = strconv.Atoi(texto)
		if err != nil || strconv.Itoa(q.Limite) != texto {
			return cero, domain.ErrDecisionCircuitoInvalida
		}
	}
	if q.Etapa.EstadoPendiente() == "" || q.Limite < 1 || q.Limite > 50 ||
		(q.FechaDesde != "" && !domain.FechaCircuitoValida(q.FechaDesde)) ||
		(q.FechaHasta != "" && !domain.FechaCircuitoValida(q.FechaHasta)) ||
		(q.FechaDesde != "" && q.FechaHasta != "" && q.FechaDesde > q.FechaHasta) ||
		(q.Cursor != "" && !referenciaCircuitoHTTP.MatchString(q.Cursor)) {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	return q, nil
}

func responderErrorCircuito(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente):
		responderError(w, http.StatusForbidden, "competencia_sin_fuente")
	case errors.Is(err, dietasports.ErrAccesoCircuitoDenegado):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, dietasports.ErrComisionNoEncontrada):
		responderError(w, http.StatusNotFound, "no_encontrada")
	case errors.Is(err, dietasports.ErrVersionComisionConflicto), errors.Is(err, dietasports.ErrEstadoCircuitoConflicto), errors.Is(err, dietasports.ErrConflictoIdempotencia), errors.Is(err, domain.ErrTransicionCircuitoInvalida), errors.Is(err, domain.ErrSeparacionCircuitoIncumplida):
		responderError(w, http.StatusConflict, "conflicto_estado")
	case errors.Is(err, domain.ErrDecisionCircuitoInvalida):
		responderError(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, dietasports.ErrResultadoCircuitoIncierto):
		responderError(w, http.StatusServiceUnavailable, "resultado_incierto")
	default:
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
	}
}

var _ http.Handler = (*ManejadorCircuito)(nil)

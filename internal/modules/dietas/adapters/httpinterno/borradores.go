// Package httpinterno publica la frontera HTTP interna de borradores de Dietas.
package httpinterno

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const RutaBorradores = "/api/vec/dietas/comisiones"
const maximoCuerpo = 16 * 1024

var (
	ErrManejadorNoDisponible = errors.New("dietas http interno: manejador no disponible")
	referenciaComisionHTTP   = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
	referenciaRelacionHTTP   = regexp.MustCompile(`^rel_[A-Za-z0-9_-]{22,128}$`)
)

type ManejadorBorradores struct {
	identidades dietasports.ResolutorIdentidadEfectivaBorrador
	casoUso     dietasapp.CasoUsoBorradorComision
}

func NuevoManejadorBorradores(identidades dietasports.ResolutorIdentidadEfectivaBorrador, casoUso dietasapp.CasoUsoBorradorComision) (*ManejadorBorradores, error) {
	if dependenciaNula(identidades) || dependenciaNula(casoUso) {
		return nil, ErrManejadorNoDisponible
	}
	return &ManejadorBorradores{identidades: identidades, casoUso: casoUso}, nil
}

func (m *ManejadorBorradores) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil || m == nil || dependenciaNula(m.identidades) || dependenciaNula(m.casoUso) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("X-Vec-Actor") != "" || r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	referencia, detalle, valida := reconocerRuta(r.URL)
	if !valida {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if detalle {
		m.atenderDetalle(w, r, referencia)
		return
	}
	m.atenderColeccion(w, r)
}

func (m *ManejadorBorradores) atenderColeccion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		consulta, relacion, err := consultaDesdeURL(r.URL)
		if err != nil || !lecturaSinCuerpo(r) {
			responderError(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		solicitud := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador, Consulta: consulta, RelacionRef: relacion}
		identidad, err := m.identidades.ResolverIdentidadEfectivaBorrador(r.Context(), solicitud)
		if err != nil {
			responderErrorClasificado(w, err)
			return
		}
		pagina, err := m.casoUso.ListarPropios(r.Context(), identidad, consulta)
		if err != nil {
			responderErrorClasificado(w, err)
			return
		}
		if pagina.Items == nil {
			pagina.Items = []dietasports.ResultadoBorradorComision{}
		}
		items := make([]resultadoBorradorJSON, len(pagina.Items))
		for indice, item := range pagina.Items {
			items[indice] = resultadoAJSON(item)
		}
		responderJSON(w, http.StatusOK, struct {
			Items           []resultadoBorradorJSON `json:"items"`
			SiguienteCursor string                  `json:"siguiente_cursor,omitempty"`
		}{items, pagina.SiguienteCursor})
	case http.MethodPost:
		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			responderError(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			responderError(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
			return
		}
		var entrada solicitudCrearJSON
		if err := decodificarSolicitud(w, r, &entrada); err != nil {
			responderError(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		solicitud := dietasports.SolicitudCrearBorradorPropio{
			ClaveIdempotencia: entrada.ClaveIdempotencia,
			FechaInicio:       entrada.FechaInicio,
			FechaFin:          entrada.FechaFin,
			Motivo:            entrada.Motivo,
			CodigosRuta:       entrada.CodigosRuta,
			RelacionRef:       entrada.RelacionRef,
		}
		operacion := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionCrearBorrador, Crear: solicitud, RelacionRef: solicitud.RelacionRef}
		identidad, err := m.identidades.ResolverIdentidadEfectivaBorrador(r.Context(), operacion)
		if err != nil {
			responderErrorClasificado(w, err)
			return
		}
		resultado, err := m.casoUso.CrearPropio(r.Context(), identidad, solicitud)
		if err != nil {
			responderErrorClasificado(w, err)
			return
		}
		estado := http.StatusCreated
		if resultado.Recibo.Repeticion {
			estado = http.StatusOK
		}
		responderJSON(w, estado, resultadoAJSON(resultado))
	default:
		w.Header().Set("Allow", "GET, POST")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
	}
}

type solicitudCrearJSON struct {
	ClaveIdempotencia string   `json:"clave_idempotencia"`
	FechaInicio       string   `json:"fecha_inicio"`
	FechaFin          string   `json:"fecha_fin"`
	Motivo            string   `json:"motivo"`
	CodigosRuta       []string `json:"codigos_ruta"`
	RelacionRef       string   `json:"relacion_ref"`
}

type reciboBorradorJSON struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	RegistradoEn string `json:"registrado_en"`
	Repeticion   bool   `json:"repeticion"`
}

type resultadoBorradorJSON struct {
	Comision domain.ComisionBorrador `json:"comision"`
	Recibo   reciboBorradorJSON      `json:"recibo"`
}

func resultadoAJSON(resultado dietasports.ResultadoBorradorComision) resultadoBorradorJSON {
	comision := resultado.Comision
	if comision.CodigosRuta == nil {
		comision.CodigosRuta = []string{}
	}
	return resultadoBorradorJSON{
		Comision: comision,
		Recibo: reciboBorradorJSON{
			Referencia:   resultado.Recibo.Referencia,
			Version:      resultado.Recibo.Version,
			RegistradoEn: resultado.Recibo.RegistradoEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
			Repeticion:   resultado.Recibo.Repeticion,
		},
	}
}

func (m *ManejadorBorradores) atenderDetalle(w http.ResponseWriter, r *http.Request, referencia string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !lecturaSinCuerpo(r) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(valores) > 1 || (len(valores) == 1 && len(valores["relacion_ref"]) != 1) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	relacion := valores.Get("relacion_ref")
	if relacion != "" && !referenciaRelacionHTTP.MatchString(relacion) {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	solicitud := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador, Referencia: referencia, RelacionRef: relacion}
	identidad, err := m.identidades.ResolverIdentidadEfectivaBorrador(r.Context(), solicitud)
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	resultado, err := m.casoUso.ObtenerPropio(r.Context(), identidad, referencia)
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	responderJSON(w, http.StatusOK, resultadoAJSON(resultado))
}

func reconocerRuta(u *url.URL) (string, bool, bool) {
	if u == nil || u.Path == "" || u.EscapedPath() != u.Path {
		return "", false, false
	}
	if u.Path == RutaBorradores {
		return "", false, true
	}
	prefijo := RutaBorradores + "/"
	if !strings.HasPrefix(u.Path, prefijo) {
		return "", false, false
	}
	referencia := strings.TrimPrefix(u.Path, prefijo)
	return referencia, true, referenciaComisionHTTP.MatchString(referencia)
}

func consultaDesdeURL(u *url.URL) (dietasports.ConsultaBorradoresPropios, string, error) {
	if u == nil || u.ForceQuery {
		return dietasports.ConsultaBorradoresPropios{}, "", errors.New("consulta invalida")
	}
	valores, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return dietasports.ConsultaBorradoresPropios{}, "", err
	}
	for clave, lista := range valores {
		if (clave != "limit" && clave != "cursor" && clave != "relacion_ref") || len(lista) != 1 {
			return dietasports.ConsultaBorradoresPropios{}, "", errors.New("consulta invalida")
		}
	}
	limite := 20
	if texto := valores.Get("limit"); texto != "" {
		limite, err = strconv.Atoi(texto)
		if err != nil || strconv.Itoa(limite) != texto || limite < 1 || limite > 50 {
			return dietasports.ConsultaBorradoresPropios{}, "", errors.New("limite invalido")
		}
	}
	consulta := dietasports.ConsultaBorradoresPropios{Limite: limite, Cursor: valores.Get("cursor")}
	if consulta.Cursor != "" && !referenciaComisionHTTP.MatchString(consulta.Cursor) {
		return consulta, "", errors.New("cursor invalido")
	}
	relacion := valores.Get("relacion_ref")
	if relacion != "" && !referenciaRelacionHTTP.MatchString(relacion) {
		return consulta, "", errors.New("relacion invalida")
	}
	return consulta, relacion, nil
}

func decodificarSolicitud(w http.ResponseWriter, r *http.Request, destino any) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength < 1 || r.ContentLength > maximoCuerpo || len(r.TransferEncoding) != 0 ||
		r.Header.Get("Content-Type") != "application/json; charset=utf-8" || r.Header.Get("Accept") != "application/json" {
		return errors.New("cuerpo invalido")
	}
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpo)
	decodificador := json.NewDecoder(lector)
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("contenido adicional")
	}
	return nil
}

func lecturaSinCuerpo(r *http.Request) bool {
	return r != nil && r.ContentLength == 0 && len(r.TransferEncoding) == 0 && (r.Body == nil || r.Body == http.NoBody) && r.Header.Get("Content-Type") == "" && r.Header.Get("Accept") == "application/json"
}

func responderErrorClasificado(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dietasports.ErrAccesoBorradorDenegado):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, dietasports.ErrRelacionNoValida):
		responderError(w, http.StatusUnprocessableEntity, "relacion_no_valida")
	case errors.Is(err, dietasports.ErrRelacionAmbigua):
		responderError(w, http.StatusConflict, "relacion_ambigua")
	case errors.Is(err, dietasports.ErrRelacionNoDisponible):
		responderError(w, http.StatusServiceUnavailable, "relacion_no_disponible")
	case errors.Is(err, dietasports.ErrComisionNoEncontrada):
		responderError(w, http.StatusNotFound, "no_encontrada")
	case errors.Is(err, dietasports.ErrConflictoIdempotencia):
		responderError(w, http.StatusConflict, "conflicto_idempotencia")
	case errors.Is(err, domain.ErrComisionBorradorInvalida):
		responderError(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, dietasports.ErrResultadoBorradorIncierto):
		responderError(w, http.StatusServiceUnavailable, "resultado_incierto")
	default:
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
	}
}

func responderError(w http.ResponseWriter, estado int, codigo string) {
	responderJSON(w, estado, struct {
		Error string `json:"error"`
	}{Error: "dietas.error." + codigo})
}

func responderJSON(w http.ResponseWriter, estado int, valor any) {
	if w == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}

func dependenciaNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	return (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface || v.Kind() == reflect.Func || v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && v.IsNil()
}

var _ http.Handler = (*ManejadorBorradores)(nil)

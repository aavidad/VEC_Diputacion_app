// Package httpinterno adapta el borrador propio de Dietas a su frontera HTTP.
package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const RutaMisComisiones = "/api/vec/dietas/mis-comisiones"
const limiteCuerpoBorradorComision int64 = dietasports.MaxBytesEntradaBorrador

var referenciaBorradorHTTP = regexp.MustCompile(`^dietas:borrador:[A-Za-z0-9_-]{16,128}$`)

// AutoridadContextoActor obtiene el contexto ya autenticado desde la composición
// del servidor. HTTP no acepta identidad, perfiles ni permisos del cliente.
type AutoridadContextoActor interface {
	ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error)
}

type ManejadorBorradorComision struct {
	servicio  *application.ServicioBorradorComision
	autoridad AutoridadContextoActor
}

func NuevoManejadorBorradorComision(servicio *application.ServicioBorradorComision, autoridad AutoridadContextoActor) (*ManejadorBorradorComision, error) {
	if servicio == nil || autoridadNula(autoridad) {
		return nil, errors.New("dietas: composicion http de borrador invalida")
	}
	return &ManejadorBorradorComision{servicio: servicio, autoridad: autoridad}, nil
}

type solicitudCrearBorradorJSON struct {
	ClaveOperacion  string              `json:"operation_key"`
	VersionEsperada uint64              `json:"expected_version"`
	Borrador        borradorEntradaJSON `json:"draft"`
}

// borradorEntradaJSON sólo contiene declaraciones del cliente. La política,
// instantes, desglose, estado y liquidabilidad los determina el caso de uso.
type borradorEntradaJSON struct {
	PoliticaReferencia string             `json:"politica_kilometraje_ref"`
	PoliticaVersion    string             `json:"politica_kilometraje_version"`
	Objeto             string             `json:"objeto"`
	FechaInicio        string             `json:"fecha"`
	FechaFin           string             `json:"fecha_fin"`
	HoraInicio         string             `json:"hora_inicio"`
	HoraFin            string             `json:"hora_fin"`
	VehiculoPropio     bool               `json:"vehiculo_propio"`
	Ruta               rutaBorradorJSON   `json:"ruta"`
	Gastos             gastosBorradorJSON `json:"gastos"`
}
type borradorComisionJSON struct {
	Objeto                    string                `json:"objeto"`
	FechaInicio               string                `json:"fecha"`
	FechaFin                  string                `json:"fecha_fin"`
	HoraInicio                string                `json:"hora_inicio"`
	HoraFin                   string                `json:"hora_fin"`
	ZonaHoraria               string                `json:"zona_horaria,omitempty"`
	Inicio                    string                `json:"inicio,omitempty"`
	Fin                       string                `json:"fin,omitempty"`
	VehiculoPropio            bool                  `json:"vehiculo_propio"`
	Ruta                      rutaBorradorJSON      `json:"ruta"`
	Gastos                    gastosBorradorJSON    `json:"gastos"`
	PoliticaKilometraje       *politicaBorradorJSON `json:"politica_kilometraje,omitempty"`
	Desglose                  *desgloseBorradorJSON `json:"desglose,omitempty"`
	Estado                    string                `json:"estado,omitempty"`
	ContextoPersonal          string                `json:"contexto_personal,omitempty"`
	Liquidable                *bool                 `json:"liquidable,omitempty"`
	RutaEtiquetas             []string              `json:"ruta_etiquetas"`
	ProcedenciaRuta           string                `json:"procedencia_ruta"`
	RevalidacionRutaRequerida bool                  `json:"revalidacion_ruta_requerida"`
}
type gastosBorradorJSON struct {
	ManutencionEUR string `json:"manutencion_eur"`
	AlojamientoEUR string `json:"alojamiento_eur"`
	OtrosEUR       string `json:"otros_eur"`
}
type politicaBorradorJSON struct {
	Referencia     string `json:"referencia"`
	Version        string `json:"version"`
	TarifaEURPorKM string `json:"tarifa_eur_km"`
}
type desgloseBorradorJSON struct {
	KilometrajeEUR string `json:"kilometraje_eur"`
	ManutencionEUR string `json:"manutencion_eur"`
	AlojamientoEUR string `json:"alojamiento_eur"`
	OtrosEUR       string `json:"otros_eur"`
	TotalEUR       string `json:"total_eur"`
}
type rutaBorradorJSON struct {
	Fuente            string           `json:"fuente"`
	Version           string           `json:"version"`
	Referencia        string           `json:"referencia"`
	CatalogoVersion   string           `json:"catalogo_version"`
	AlternativaRef    string           `json:"alternativa_ref"`
	Recomendada       bool             `json:"recomendada"`
	MotivoAlternativa string           `json:"motivo_alternativa"`
	Kilometros        string           `json:"kilometros"`
	Paradas           []paradaRutaJSON `json:"paradas,omitempty"`
	Tramos            []tramoRutaJSON  `json:"tramos,omitempty"`
	Trazado           [][]float64      `json:"trazado,omitempty"`
	Liquidable        bool             `json:"liquidable"`
}
type paradaRutaJSON struct {
	Codigo   string  `json:"codigo"`
	Nombre   string  `json:"nombre"`
	Latitud  float64 `json:"latitud"`
	Longitud float64 `json:"longitud"`
}
type tramoRutaJSON struct {
	OrigenCodigo     string `json:"origen_codigo"`
	DestinoCodigo    string `json:"destino_codigo"`
	Kilometros       string `json:"kilometros"`
	DuracionMinutos  int    `json:"duracion_minutos"`
	AjusteKilometros string `json:"ajuste_kilometros"`
	MotivoAjuste     string `json:"motivo_ajuste"`
}
type reciboBorradorJSON struct {
	ComisionRef    string `json:"commission_ref"`
	ReciboRef      string `json:"receipt_ref"`
	CorrelacionRef string `json:"correlation_ref"`
	Version        uint64 `json:"version"`
	RegistradoEn   string `json:"registered_at"`
	Repeticion     bool   `json:"repeated"`
}
type respuestaCrearBorradorJSON struct {
	Recibo reciboBorradorJSON `json:"receipt"`
}
type respuestaRecuperarBorradorJSON struct {
	Borrador borradorComisionJSON `json:"draft"`
	Recibo   reciboBorradorJSON   `json:"receipt"`
}
type politicaColeccionJSON struct {
	Disponible     bool   `json:"available"`
	Liquidable     bool   `json:"liquidable"`
	Referencia     string `json:"reference,omitempty"`
	Version        string `json:"version,omitempty"`
	TarifaEURPorKM string `json:"rate_eur_km,omitempty"`
}
type respuestaColeccionJSON struct {
	Items     []respuestaRecuperarBorradorJSON `json:"items"`
	Siguiente string                           `json:"next_cursor"`
	Politica  politicaColeccionJSON            `json:"policy"`
}

func (m *ManejadorBorradorComision) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararRespuesta(w)
	if m == nil || m.servicio == nil || autoridadNula(m.autoridad) || r == nil || r.URL == nil {
		escribirError(w, http.StatusServiceUnavailable, "dietas.error.borrador_no_disponible")
		return
	}
	if len(r.Header.Values("Authorization")) != 0 || len(r.Header.Values("Cookie")) != 0 {
		escribirError(w, http.StatusForbidden, "dietas.error.acceso_denegado")
		return
	}
	if r.URL.Fragment != "" || r.URL.RawFragment != "" {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == RutaMisComisiones && r.URL.RawQuery == "":
		m.crear(w, r)
	case r.Method == http.MethodGet && r.URL.Path == RutaMisComisiones:
		m.listar(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, RutaMisComisiones+"/") && r.URL.RawQuery == "":
		m.recuperar(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		escribirError(w, http.StatusMethodNotAllowed, "dietas.error.metodo_no_permitido")
	}
}
func (m *ManejadorBorradorComision) crear(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || !esJSON(r.Header.Get("Content-Type")) {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	defer r.Body.Close()
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, limiteCuerpoBorradorComision))
	d.DisallowUnknownFields()
	var entrada solicitudCrearBorradorJSON
	if d.Decode(&entrada) != nil || d.Decode(&struct{}{}) != io.EOF || entrada.ClaveOperacion == "" || r.Header.Get("Idempotency-Key") != entrada.ClaveOperacion {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	actor, ok := m.actor(w, r)
	if !ok {
		return
	}
	b, ok := aDominioEntrada(entrada.Borrador, actor.PersonaRef)
	if !ok {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	recibo, e := m.servicio.CrearPropio(r.Context(), dietasports.SolicitudCrearBorradorPropio{ContextoActor: actor, ClaveOperacion: entrada.ClaveOperacion, VersionEsperada: entrada.VersionEsperada, Borrador: b, PoliticaReferencia: entrada.Borrador.PoliticaReferencia, PoliticaVersion: entrada.Borrador.PoliticaVersion})
	if e != nil {
		m.errorServicio(w, e)
		return
	}
	estado := http.StatusCreated
	if recibo.Repeticion {
		estado = http.StatusOK
	}
	escribirJSON(w, estado, respuestaCrearBorradorJSON{aReciboJSON(recibo)})
}
func (m *ManejadorBorradorComision) recuperar(w http.ResponseWriter, r *http.Request) {
	ref := strings.TrimPrefix(r.URL.Path, RutaMisComisiones+"/")
	if !referenciaBorradorHTTP.MatchString(ref) || strings.Contains(ref, "/") {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	actor, ok := m.actor(w, r)
	if !ok {
		return
	}
	b, recibo, e := m.servicio.RecuperarPropio(r.Context(), actor, ref)
	if e != nil {
		m.errorServicio(w, e)
		return
	}
	escribirJSON(w, http.StatusOK, respuestaRecuperarBorradorJSON{desdeDominio(b), aReciboJSON(recibo)})
}
func (m *ManejadorBorradorComision) listar(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	q, ok := consultaDesdeURL(r.URL)
	if !ok {
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
		return
	}
	actor, ok := m.actor(w, r)
	if !ok {
		return
	}
	pagina, e := m.servicio.ListarPropios(r.Context(), actor, q)
	if e != nil {
		m.errorServicio(w, e)
		return
	}
	items := make([]respuestaRecuperarBorradorJSON, 0, len(pagina.Borradores))
	for _, v := range pagina.Borradores {
		items = append(items, respuestaRecuperarBorradorJSON{desdeDominio(v.Borrador), aReciboJSON(v.Recibo)})
	}
	politica := politicaColeccionJSON{Disponible: false, Liquidable: false}
	if p, e := m.servicio.PoliticaActual(r.Context(), actor); e == nil {
		politica = politicaColeccionJSON{Disponible: true, Liquidable: false, Referencia: p.Referencia, Version: p.Version, TarifaEURPorKM: p.TarifaEURPorKM}
	}
	escribirJSON(w, http.StatusOK, respuestaColeccionJSON{Items: items, Siguiente: pagina.Siguiente, Politica: politica})
}
func consultaDesdeURL(u *url.URL) (dietasports.ConsultaBorradoresPropios, bool) {
	if u == nil || len(u.RawQuery) > 512 {
		return dietasports.ConsultaBorradoresPropios{}, false
	}
	v, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return dietasports.ConsultaBorradoresPropios{}, false
	}
	for k, vs := range v {
		if (k != "limit" && k != "after") || len(vs) != 1 {
			return dietasports.ConsultaBorradoresPropios{}, false
		}
	}
	limite := 20
	if s, existe := v["limit"]; existe {
		n, e := strconv.Atoi(s[0])
		if e != nil || n < 1 || n > 20 || strconv.Itoa(n) != s[0] {
			return dietasports.ConsultaBorradoresPropios{}, false
		}
		limite = n
	}
	despues := ""
	if s, existe := v["after"]; existe {
		despues = s[0]
		if !referenciaBorradorHTTP.MatchString(despues) {
			return dietasports.ConsultaBorradoresPropios{}, false
		}
	}
	return dietasports.ConsultaBorradoresPropios{Limite: limite, Despues: despues}, true
}
func (m *ManejadorBorradorComision) actor(w http.ResponseWriter, r *http.Request) (vecdomain.ContextoActor, bool) {
	a, e := m.autoridad.ResolverContextoActor(r.Context())
	if e != nil || a.Validar() != nil {
		escribirError(w, http.StatusForbidden, "dietas.error.acceso_denegado")
		return vecdomain.ContextoActor{}, false
	}
	return a, true
}
func (m *ManejadorBorradorComision) errorServicio(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, dietasports.ErrAccesoBorradorDenegado):
		escribirError(w, http.StatusForbidden, "dietas.error.acceso_denegado")
	case errors.Is(e, dietasports.ErrConflictoBorrador):
		escribirError(w, http.StatusConflict, "dietas.error.conflicto_borrador")
	case errors.Is(e, dietasports.ErrBorradorNoEncontrado):
		escribirError(w, http.StatusNotFound, "dietas.error.borrador_no_encontrado")
	case errors.Is(e, dietasports.ErrResultadoBorradorIncierto):
		escribirError(w, http.StatusServiceUnavailable, "dietas.error.resultado_incierto")
	case errors.Is(e, dietasports.ErrPoliticaBorradorNoDisponible):
		escribirError(w, http.StatusServiceUnavailable, "dietas.error.politica_no_disponible")
	case errors.Is(e, domain.ErrBorradorComisionInvalido):
		escribirError(w, http.StatusBadRequest, "dietas.error.peticion_invalida")
	default:
		escribirError(w, http.StatusServiceUnavailable, "dietas.error.borrador_no_disponible")
	}
}
func aDominio(b borradorComisionJSON, p string) (domain.BorradorComision, bool) {
	if b.Ruta.Liquidable {
		return domain.BorradorComision{}, false
	}
	trazado := make([][2]float64, len(b.Ruta.Trazado))
	for i, v := range b.Ruta.Trazado {
		if len(v) != 2 {
			return domain.BorradorComision{}, false
		}
		trazado[i] = [2]float64{v[0], v[1]}
	}
	paradas := make([]domain.ParadaRuta, len(b.Ruta.Paradas))
	for i, v := range b.Ruta.Paradas {
		paradas[i] = domain.ParadaRuta{Codigo: v.Codigo, Nombre: v.Nombre, Latitud: v.Latitud, Longitud: v.Longitud}
	}
	tramos := make([]domain.TramoRuta, len(b.Ruta.Tramos))
	for i, v := range b.Ruta.Tramos {
		tramos[i] = domain.TramoRuta{OrigenCodigo: v.OrigenCodigo, DestinoCodigo: v.DestinoCodigo, Kilometros: v.Kilometros, DuracionMinutos: v.DuracionMinutos, AjusteKilometros: v.AjusteKilometros, MotivoAjuste: v.MotivoAjuste}
	}
	return domain.BorradorComision{PersonaRef: p, Objeto: b.Objeto, FechaInicio: b.FechaInicio, FechaFin: b.FechaFin, HoraInicio: b.HoraInicio, HoraFin: b.HoraFin, VehiculoPropio: b.VehiculoPropio, Ruta: domain.RutaVersionada{Fuente: b.Ruta.Fuente, Version: b.Ruta.Version, Referencia: b.Ruta.Referencia, CatalogoVersion: b.Ruta.CatalogoVersion, AlternativaRef: b.Ruta.AlternativaRef, Recomendada: b.Ruta.Recomendada, MotivoAlternativa: b.Ruta.MotivoAlternativa, Kilometros: b.Ruta.Kilometros, Paradas: paradas, Tramos: tramos, Trazado: trazado, Liquidable: false}, Gastos: domain.GastosComision{ManutencionEUR: b.Gastos.ManutencionEUR, AlojamientoEUR: b.Gastos.AlojamientoEUR, OtrosEUR: b.Gastos.OtrosEUR}}, true
}

func aDominioEntrada(b borradorEntradaJSON, p string) (domain.BorradorComision, bool) {
	return aDominio(borradorComisionJSON{Objeto: b.Objeto, FechaInicio: b.FechaInicio, FechaFin: b.FechaFin, HoraInicio: b.HoraInicio, HoraFin: b.HoraFin, VehiculoPropio: b.VehiculoPropio, Ruta: b.Ruta, Gastos: b.Gastos}, p)
}
func desdeDominio(b domain.BorradorComision) borradorComisionJSON {
	liquidable := false
	return borradorComisionJSON{Objeto: b.Objeto, FechaInicio: b.FechaInicio, FechaFin: b.FechaFin, HoraInicio: b.HoraInicio, HoraFin: b.HoraFin, ZonaHoraria: b.ZonaHoraria, Inicio: b.Inicio.UTC().Format(time.RFC3339Nano), Fin: b.Fin.UTC().Format(time.RFC3339Nano), VehiculoPropio: b.VehiculoPropio, Ruta: desdeRuta(b.Ruta), Gastos: gastosBorradorJSON{b.Gastos.ManutencionEUR, b.Gastos.AlojamientoEUR, b.Gastos.OtrosEUR}, PoliticaKilometraje: &politicaBorradorJSON{b.PoliticaKilometraje.Referencia, b.PoliticaKilometraje.Version, b.PoliticaKilometraje.TarifaEURPorKM}, Desglose: &desgloseBorradorJSON{b.Desglose.KilometrajeEUR, b.Desglose.ManutencionEUR, b.Desglose.AlojamientoEUR, b.Desglose.OtrosEUR, b.Desglose.TotalEUR}, Estado: b.Estado, ContextoPersonal: b.ContextoPersonal, Liquidable: &liquidable, RutaEtiquetas: append([]string(nil), b.RutaEtiquetas...), ProcedenciaRuta: b.ProcedenciaRuta, RevalidacionRutaRequerida: b.RevalidacionRutaRequerida}
}
func desdeRuta(r domain.RutaVersionada) rutaBorradorJSON {
	p := make([]paradaRutaJSON, len(r.Paradas))
	for i, v := range r.Paradas {
		p[i] = paradaRutaJSON{v.Codigo, v.Nombre, v.Latitud, v.Longitud}
	}
	t := make([]tramoRutaJSON, len(r.Tramos))
	for i, v := range r.Tramos {
		t[i] = tramoRutaJSON{v.OrigenCodigo, v.DestinoCodigo, v.Kilometros, v.DuracionMinutos, v.AjusteKilometros, v.MotivoAjuste}
	}
	tr := make([][]float64, len(r.Trazado))
	for i, v := range r.Trazado {
		tr[i] = []float64{v[0], v[1]}
	}
	return rutaBorradorJSON{Fuente: r.Fuente, Version: r.Version, Referencia: r.Referencia, CatalogoVersion: r.CatalogoVersion, AlternativaRef: r.AlternativaRef, Recomendada: r.Recomendada, MotivoAlternativa: r.MotivoAlternativa, Kilometros: r.Kilometros, Paradas: p, Tramos: t, Trazado: tr, Liquidable: false}
}
func aReciboJSON(r dietasports.ReciboBorradorComision) reciboBorradorJSON {
	return reciboBorradorJSON{r.ComisionRef, r.ReciboRef, r.CorrelacionRef, r.Version, r.RegistradoEn.UTC().Format(time.RFC3339Nano), r.Repeticion}
}
func esJSON(v string) bool {
	t, _, e := mime.ParseMediaType(v)
	return e == nil && t == "application/json"
}
func prepararRespuesta(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func escribirError(w http.ResponseWriter, e int, c string) {
	escribirJSON(w, e, map[string]string{"error": c})
}
func escribirJSON(w http.ResponseWriter, e int, r any) {
	w.WriteHeader(e)
	eJSON := json.NewEncoder(escritorJSONCompleto{w})
	eJSON.SetEscapeHTML(false)
	// La escritura puede fallar después del COMMIT. No enviar otra respuesta
	// ni dar por entregado el recibo: net/http aborta el transporte y el cliente
	// puede recuperar o repetir con la misma clave sin duplicar el borrador.
	if err := eJSON.Encode(r); err != nil {
		panic(http.ErrAbortHandler)
	}
}

// También detecta adaptadores que incumplen io.Writer devolviendo un fragmento
// sin error: una respuesta JSON parcial nunca cuenta como entrega completa.
type escritorJSONCompleto struct{ io.Writer }

func (w escritorJSONCompleto) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

func autoridadNula(a AutoridadContextoActor) bool {
	if a == nil {
		return true
	}
	v := reflect.ValueOf(a)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

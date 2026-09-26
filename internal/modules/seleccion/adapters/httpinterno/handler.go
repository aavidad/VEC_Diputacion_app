// Package httpinterno es el adaptador HTTP de la consulta de solicitudes de
// Selección en la superficie interna de RRHH (patrón «/consultas»: la
// consulta viaja en el cuerpo de un POST, nunca en la URL).
package httpinterno

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/httpcomun"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// Rutas de RRHH.
const (
	RutaConvocatorias    = "/api/vec/seleccion/solicitudes/convocatorias"
	RutaConsultas        = "/api/vec/seleccion/solicitudes/consultas"
	RutaDetalleConsultas = "/api/vec/seleccion/solicitudes/detalle/consultas"
	maximoCuerpo         = 4096
)

// Rutas enumera las rutas exactas de RRHH.
func Rutas() []string {
	return []string{RutaConvocatorias, RutaConsultas, RutaDetalleConsultas}
}

// EsRuta dice si la ruta pertenece a la consulta de RRHH.
func EsRuta(ruta string) bool {
	for _, r := range Rutas() {
		if r == ruta {
			return true
		}
	}
	return false
}

// OperacionEn describe cada método y ruta con la acción V3 que consume.
type OperacionEn struct {
	Metodo, Ruta, Accion string
}

// Operaciones enumera las operaciones de RRHH.
func Operaciones() []OperacionEn {
	return []OperacionEn{
		{http.MethodGet, RutaConvocatorias, ""},
		{http.MethodPost, RutaConsultas, ports.AccionConsultarSolicitudes},
		{http.MethodPost, RutaDetalleConsultas, ports.AccionConsultarDetalle},
	}
}

// Preparador es el puerto de identidad de RRHH (certificado corporativo hoy).
type Preparador interface {
	PrepararOrdenRRHH(*http.Request) (application.Orden, error)
}

// Casos son los casos de uso de RRHH.
type Casos interface {
	Convocatorias(context.Context) ([]domain.ConvocatoriaPublicada, error)
	Listar(context.Context, application.Orden, application.ConsultaListado) (application.PaginaRRHH, error)
	Detalle(context.Context, application.Orden, string) (application.FichaRRHH, error)
}

// Handler atiende las tres rutas exactas.
type Handler struct {
	preparador Preparador
	casos      Casos
}

// Nuevo exige sus dos dependencias.
func Nuevo(preparador Preparador, casos Casos) (*Handler, error) {
	if httpcomun.Nula(preparador) || httpcomun.Nula(casos) {
		return nil, ports.ErrNoDisponible
	}
	return &Handler{preparador: preparador, casos: casos}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || httpcomun.Nula(h.preparador) || httpcomun.Nula(h.casos) {
		httpcomun.Responder(w, http.StatusServiceUnavailable, httpcomun.Error("servicio_no_disponible"))
		return
	}
	switch {
	case httpcomun.RutaExacta(r, RutaConvocatorias):
		h.convocatorias(w, r)
	case httpcomun.RutaExacta(r, RutaConsultas):
		h.listar(w, r)
	case httpcomun.RutaExacta(r, RutaDetalleConsultas):
		h.detalle(w, r)
	default:
		httpcomun.Responder(w, http.StatusNotFound, httpcomun.Error("recurso_no_encontrado"))
	}
}

func (h *Handler) convocatorias(w http.ResponseWriter, r *http.Request) {
	if !httpcomun.Metodo(w, r, http.MethodGet) {
		return
	}
	if _, ok := httpcomun.LecturaPermitida(r); !ok {
		httpcomun.Responder(w, http.StatusBadRequest, httpcomun.Error("peticion_no_permitida"))
		return
	}
	if _, err := h.preparador.PrepararOrdenRRHH(r); err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	lista, err := h.casos.Convocatorias(r.Context())
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(httpcomun.ResumenConvocatorias(lista)))
}

type entradaListado struct {
	ConvocatoriaRef string `json:"convocatoria_ref"`
	Cursor          string `json:"cursor"`
	Limite          int    `json:"limite"`
}

type filaSalida struct {
	SolicitudRef       string `json:"solicitud_ref"`
	NumeroJustificante string `json:"numero_justificante"`
	NombreVisible      string `json:"nombre_visible"`
	DocumentoParcial   string `json:"documento_parcial"`
	Estado             string `json:"estado"`
	PresentadaEn       string `json:"presentada_en"`
	Puntuacion         string `json:"puntuacion_autobaremo"`
	Turno              string `json:"turno"`
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	if !httpcomun.Metodo(w, r, http.MethodPost) {
		return
	}
	var e entradaListado
	if _, ok := httpcomun.LeerEscritura(w, r, maximoCuerpo, &e, false); !ok {
		return
	}
	orden, err := h.preparador.PrepararOrdenRRHH(r)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	pagina, err := h.casos.Listar(r.Context(), orden, application.ConsultaListado{ConvocatoriaRef: e.ConvocatoriaRef, Cursor: e.Cursor, Limite: e.Limite})
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	filas := make([]filaSalida, 0, len(pagina.Filas))
	for _, f := range pagina.Filas {
		filas = append(filas, filaSalida{SolicitudRef: f.SolicitudRef, NumeroJustificante: f.NumeroJustificante, NombreVisible: f.NombreVisible,
			DocumentoParcial: f.DocumentoParcial, Estado: string(f.Estado), PresentadaEn: f.PresentadaEn, Puntuacion: httpcomun.Puntos(f.Puntuacion), Turno: f.Turno})
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(struct {
		Solicitudes     []filaSalida `json:"solicitudes"`
		CursorSiguiente string       `json:"cursor_siguiente"`
	}{filas, pagina.CursorSiguiente}))
}

type requisitoSalida struct {
	Clave           string `json:"clave"`
	Titulo          string `json:"titulo"`
	Estado          string `json:"estado"`
	Procedencia     string `json:"procedencia"`
	FechaReferencia string `json:"fecha_referencia"`
}

type meritoSalida struct {
	ClaveGrupo  string `json:"clave_grupo"`
	ClaveMerito string `json:"clave_merito"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Cantidad    string `json:"cantidad"`
	Unidad      string `json:"unidad"`
	Puntos      string `json:"puntos"`
}

type historiaSalida struct {
	Tipo    string `json:"tipo"`
	Version int    `json:"version"`
	En      string `json:"en"`
}

type fichaSalida struct {
	SolicitudRef       string                 `json:"solicitud_ref"`
	ConvocatoriaRef    string                 `json:"convocatoria_ref"`
	ConvocatoriaTitulo string                 `json:"convocatoria_titulo"`
	NumeroJustificante string                 `json:"numero_justificante"`
	Estado             string                 `json:"estado"`
	PresentadaEn       string                 `json:"presentada_en"`
	Turno              string                 `json:"turno"`
	Datos              domain.DatosPersonales `json:"datos"`
	Requisitos         []requisitoSalida      `json:"requisitos"`
	Meritos            []meritoSalida         `json:"meritos"`
	Puntuacion         string                 `json:"puntuacion_autobaremo"`
	Historia           []historiaSalida       `json:"historia"`
}

func (h *Handler) detalle(w http.ResponseWriter, r *http.Request) {
	if !httpcomun.Metodo(w, r, http.MethodPost) {
		return
	}
	var e struct {
		SolicitudRef string `json:"solicitud_ref"`
	}
	if _, ok := httpcomun.LeerEscritura(w, r, maximoCuerpo, &e, false); !ok {
		return
	}
	orden, err := h.preparador.PrepararOrdenRRHH(r)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	f, err := h.casos.Detalle(r.Context(), orden, e.SolicitudRef)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	salida := fichaSalida{SolicitudRef: f.Fila.SolicitudRef, ConvocatoriaRef: f.Fila.ConvocatoriaRef, ConvocatoriaTitulo: f.Convocatoria.Titulo,
		NumeroJustificante: f.Fila.NumeroJustificante, Estado: string(domain.EstadoPresentada), PresentadaEn: httpcomun.Instante(f.Fila.PresentadaEn),
		Turno: f.Fila.Turno, Datos: f.Datos, Puntuacion: httpcomun.Puntos(f.Fila.Puntuacion),
		Requisitos: make([]requisitoSalida, 0, len(f.Requisitos)), Meritos: make([]meritoSalida, 0, len(f.Meritos)), Historia: make([]historiaSalida, 0, len(f.Historia))}
	for _, q := range f.Requisitos {
		salida.Requisitos = append(salida.Requisitos, requisitoSalida{Clave: q.Clave, Titulo: q.Titulo, Estado: string(q.Estado), Procedencia: q.Procedencia, FechaReferencia: q.FechaReferencia})
	}
	for _, m := range f.Meritos {
		salida.Meritos = append(salida.Meritos, meritoSalida{ClaveGrupo: m.ClaveGrupo, ClaveMerito: m.ClaveMerito, Titulo: m.Titulo, Descripcion: m.Descripcion,
			Cantidad: m.Cantidad, Unidad: m.Unidad, Puntos: httpcomun.Puntos(m.Puntos)})
	}
	for _, x := range f.Historia {
		salida.Historia = append(salida.Historia, historiaSalida{Tipo: x.Tipo, Version: x.Version, En: httpcomun.Instante(x.En)})
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(salida))
}

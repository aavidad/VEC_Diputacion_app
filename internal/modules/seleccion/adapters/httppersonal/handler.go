// Package httppersonal es el adaptador HTTP de las solicitudes propias de la
// persona en la superficie personal (certificado mTLS hoy; Cl@ve o DNIe por
// otro Preparador). La identidad sale de la frontera, nunca del cuerpo.
package httppersonal

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/httpcomun"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// Rutas de la persona.
const (
	RutaMisSolicitudes = "/api/vec/seleccion/mis-solicitudes"
	RutaConvocatorias  = RutaMisSolicitudes + "/convocatorias"
	RutaConvocatoria   = RutaMisSolicitudes + "/convocatoria"
	RutaBorrador       = RutaMisSolicitudes + "/borrador"
	RutaPresentacion   = RutaMisSolicitudes + "/presentacion"
	maximoCuerpo       = 64 * 1024
)

// Rutas enumera las rutas exactas de la persona.
func Rutas() []string {
	return []string{RutaMisSolicitudes, RutaConvocatorias, RutaConvocatoria, RutaBorrador, RutaPresentacion}
}

// EsRuta dice si la ruta pertenece a las solicitudes propias.
func EsRuta(ruta string) bool {
	for _, r := range Rutas() {
		if r == ruta {
			return true
		}
	}
	return false
}

// OperacionEn describe cada combinación admitida de método y ruta, con la
// acción V3 que consume (vacía si la ruta solo sirve información pública de
// las bases). Lo usa la frontera para declarar y cotejar cada ruta.
type OperacionEn struct {
	Metodo, Ruta, Accion string
}

// Operaciones enumera todas las operaciones de la superficie personal.
func Operaciones() []OperacionEn {
	return []OperacionEn{
		{http.MethodGet, RutaConvocatorias, ""},
		{http.MethodGet, RutaConvocatoria, ""},
		{http.MethodGet, RutaMisSolicitudes, ports.AccionConsultarPropias},
		{http.MethodGet, RutaBorrador, ports.AccionConsultarPropias},
		{http.MethodPut, RutaBorrador, ports.AccionGuardarBorrador},
		{http.MethodPost, RutaPresentacion, ports.AccionPresentar},
	}
}

// Preparador es el puerto de identidad de la persona: traduce la petición
// autenticada por la frontera en una Orden. Sin identidad verificada debe
// devolver httpcomun.ErrAutenticacionAusente.
type Preparador interface {
	PrepararOrdenPersona(*http.Request) (application.Orden, error)
}

// Casos son los casos de uso de la persona.
type Casos interface {
	Convocatorias(context.Context) ([]domain.ConvocatoriaPublicada, error)
	Convocatoria(context.Context, string) (domain.ConvocatoriaPublicada, error)
	Listar(context.Context, application.Orden) ([]ports.ResumenSolicitudPropia, error)
	LeerBorrador(context.Context, application.Orden, string) (application.BorradorLeido, error)
	GuardarBorrador(context.Context, application.Orden, application.EntradaBorrador) (ports.ResultadoGuardado, error)
	Presentar(context.Context, application.Orden, application.EntradaPresentacion) (application.ReciboPresentacion, error)
}

// Handler atiende las cinco rutas exactas.
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
	var ruta string
	for _, candidata := range Rutas() {
		if httpcomun.RutaExacta(r, candidata) {
			ruta = candidata
		}
	}
	switch ruta {
	case RutaConvocatorias:
		h.convocatorias(w, r)
	case RutaConvocatoria:
		h.convocatoria(w, r)
	case RutaMisSolicitudes:
		h.listar(w, r)
	case RutaBorrador:
		if httpcomun.Metodo(w, r, http.MethodGet, http.MethodPut) {
			if r.Method == http.MethodGet {
				h.leerBorrador(w, r)
			} else {
				h.guardarBorrador(w, r)
			}
		}
	case RutaPresentacion:
		h.presentar(w, r)
	default:
		httpcomun.Responder(w, http.StatusNotFound, httpcomun.Error("recurso_no_encontrado"))
	}
}

// lectura valida un GET y resuelve la identidad.
func (h *Handler) lectura(w http.ResponseWriter, r *http.Request, parametros ...string) (application.Orden, map[string]string, bool) {
	if !httpcomun.Metodo(w, r, http.MethodGet) {
		return application.Orden{}, nil, false
	}
	valores, ok := httpcomun.LecturaPermitida(r, parametros...)
	if !ok {
		httpcomun.Responder(w, http.StatusBadRequest, httpcomun.Error("peticion_no_permitida"))
		return application.Orden{}, nil, false
	}
	orden, err := h.preparador.PrepararOrdenPersona(r)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return application.Orden{}, nil, false
	}
	return orden, valores, true
}

func (h *Handler) convocatorias(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.lectura(w, r); !ok {
		return
	}
	lista, err := h.casos.Convocatorias(r.Context())
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(httpcomun.ResumenConvocatorias(lista)))
}

// ConvocatoriaDetalle son las reglas publicadas de la convocatoria.
type ConvocatoriaDetalle struct {
	ConvocatoriaRef string `json:"convocatoria_ref"`
	Version         int    `json:"version"`
	Titulo          string `json:"titulo"`
	AbreEn          string `json:"abre_en"`
	CierraEn        string `json:"cierra_en"`
	Abierta         bool   `json:"abierta"`
	domain.ContenidoCanonico
}

// DetalleConvocatoria traduce una convocatoria publicada.
func DetalleConvocatoria(c domain.ConvocatoriaPublicada) ConvocatoriaDetalle {
	return ConvocatoriaDetalle{ConvocatoriaRef: c.Ref, Version: c.Version, Titulo: c.Titulo, AbreEn: httpcomun.Instante(c.AbreEn),
		CierraEn: httpcomun.Instante(c.CierraEn), Abierta: c.Abierta, ContenidoCanonico: c.Contenido()}
}

func (h *Handler) convocatoria(w http.ResponseWriter, r *http.Request) {
	_, valores, ok := h.lectura(w, r, "convocatoria_ref")
	if !ok {
		return
	}
	c, err := h.casos.Convocatoria(r.Context(), valores["convocatoria_ref"])
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(DetalleConvocatoria(c)))
}

type resumen struct {
	SolicitudRef       string  `json:"solicitud_ref"`
	ConvocatoriaRef    string  `json:"convocatoria_ref"`
	ConvocatoriaTitulo string  `json:"convocatoria_titulo"`
	Estado             string  `json:"estado"`
	Version            int     `json:"version"`
	PresentadaEn       *string `json:"presentada_en"`
	NumeroJustificante *string `json:"numero_justificante"`
	Puntuacion         string  `json:"puntuacion_autobaremo"`
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	orden, _, ok := h.lectura(w, r)
	if !ok {
		return
	}
	lista, err := h.casos.Listar(r.Context(), orden)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	filas := make([]resumen, 0, len(lista))
	for _, s := range lista {
		fila := resumen{SolicitudRef: s.SolicitudRef, ConvocatoriaRef: s.ConvocatoriaRef, ConvocatoriaTitulo: s.ConvocatoriaTitulo,
			Estado: string(s.Estado), Version: s.Version, Puntuacion: httpcomun.Puntos(s.Puntuacion)}
		if s.PresentadaEn != nil {
			t := httpcomun.Instante(*s.PresentadaEn)
			fila.PresentadaEn = &t
		}
		if s.NumeroJustificante != "" {
			n := s.NumeroJustificante
			fila.NumeroJustificante = &n
		}
		filas = append(filas, fila)
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(struct {
		Solicitudes []resumen `json:"solicitudes"`
	}{filas}))
}

type meritoES struct {
	ClaveGrupo  string `json:"clave_grupo"`
	ClaveMerito string `json:"clave_merito"`
	Descripcion string `json:"descripcion"`
	Cantidad    string `json:"cantidad"`
}

type borradorSalida struct {
	SolicitudRef    string                      `json:"solicitud_ref"`
	ConvocatoriaRef string                      `json:"convocatoria_ref"`
	Version         int                         `json:"version"`
	Estado          string                      `json:"estado"`
	Turno           string                      `json:"turno"`
	DatosCompletos  bool                        `json:"datos_completos"`
	Datos           domain.DatosPersonales      `json:"datos"`
	Requisitos      []domain.RequisitoDeclarado `json:"requisitos"`
	Meritos         []meritoES                  `json:"meritos"`
	Puntuacion      string                      `json:"puntuacion_autobaremo"`
	ActualizadaEn   string                      `json:"actualizada_en"`
}

func (h *Handler) leerBorrador(w http.ResponseWriter, r *http.Request) {
	orden, valores, ok := h.lectura(w, r, "convocatoria_ref")
	if !ok {
		return
	}
	b, err := h.casos.LeerBorrador(r.Context(), orden, valores["convocatoria_ref"])
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	meritos := make([]meritoES, 0, len(b.Borrador.Meritos))
	for _, m := range b.Borrador.Meritos {
		meritos = append(meritos, meritoES(m))
	}
	httpcomun.Responder(w, http.StatusOK, httpcomun.Datos(borradorSalida{SolicitudRef: b.Version.SolicitudRef, ConvocatoriaRef: b.Version.ConvocatoriaRef,
		Version: b.Version.Version, Estado: string(b.Version.Estado), Turno: b.Borrador.Turno, DatosCompletos: b.Version.DatosCompletos,
		Datos: b.Borrador.Datos, Requisitos: b.Borrador.Requisitos, Meritos: meritos, Puntuacion: httpcomun.Puntos(b.Version.Puntuacion),
		ActualizadaEn: httpcomun.Instante(b.Version.ActualizadaEn)}))
}

type entradaBorrador struct {
	ConvocatoriaRef string                      `json:"convocatoria_ref"`
	VersionEsperada *int                        `json:"version_esperada"`
	Turno           string                      `json:"turno"`
	Datos           domain.DatosPersonales      `json:"datos"`
	Requisitos      []domain.RequisitoDeclarado `json:"requisitos"`
	Meritos         []meritoES                  `json:"meritos"`
}

func (h *Handler) guardarBorrador(w http.ResponseWriter, r *http.Request) {
	var e entradaBorrador
	clave, ok := httpcomun.LeerEscritura(w, r, maximoCuerpo, &e, true)
	if !ok {
		return
	}
	if e.VersionEsperada == nil || e.ConvocatoriaRef == "" {
		httpcomun.Responder(w, http.StatusBadRequest, httpcomun.Error("datos_no_validos"))
		return
	}
	orden, err := h.preparador.PrepararOrdenPersona(r)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	meritos := make([]domain.MeritoDeclarado, 0, len(e.Meritos))
	for _, m := range e.Meritos {
		meritos = append(meritos, domain.MeritoDeclarado(m))
	}
	resultado, err := h.casos.GuardarBorrador(r.Context(), orden, application.EntradaBorrador{ConvocatoriaRef: e.ConvocatoriaRef,
		VersionEsperada: *e.VersionEsperada, Clave: clave,
		Borrador: domain.Borrador{Turno: e.Turno, Datos: e.Datos, Requisitos: e.Requisitos, Meritos: meritos}})
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	httpcomun.Responder(w, estadoCreacion(resultado.Reutilizada), httpcomun.Datos(struct {
		SolicitudRef   string `json:"solicitud_ref"`
		Version        int    `json:"version"`
		Estado         string `json:"estado"`
		Puntuacion     string `json:"puntuacion_autobaremo"`
		DatosCompletos bool   `json:"datos_completos"`
		Repetida       bool   `json:"repetida"`
	}{resultado.SolicitudRef, resultado.Version, string(domain.EstadoBorrador), httpcomun.Puntos(resultado.Puntuacion), resultado.DatosCompletos, resultado.Reutilizada}))
}

type entradaPresentacion struct {
	SolicitudRef           string `json:"solicitud_ref"`
	VersionEsperada        int    `json:"version_esperada"`
	DeclaracionResponsable bool   `json:"declaracion_responsable"`
}

func (h *Handler) presentar(w http.ResponseWriter, r *http.Request) {
	if !httpcomun.Metodo(w, r, http.MethodPost) {
		return
	}
	var e entradaPresentacion
	clave, ok := httpcomun.LeerEscritura(w, r, 4096, &e, true)
	if !ok {
		return
	}
	orden, err := h.preparador.PrepararOrdenPersona(r)
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	recibo, err := h.casos.Presentar(r.Context(), orden, application.EntradaPresentacion{SolicitudRef: e.SolicitudRef,
		VersionEsperada: e.VersionEsperada, Clave: clave, DeclaracionResponsable: e.DeclaracionResponsable})
	if err != nil {
		httpcomun.ResponderError(w, err)
		return
	}
	servicios := make(map[string]string, len(recibo.Servicios))
	for k, v := range recibo.Servicios {
		servicios[k] = string(v)
	}
	httpcomun.Responder(w, estadoCreacion(recibo.Reutilizada), httpcomun.Datos(struct {
		SolicitudRef       string            `json:"solicitud_ref"`
		Estado             string            `json:"estado"`
		NumeroJustificante string            `json:"numero_justificante"`
		PresentadaEn       string            `json:"presentada_en"`
		ReciboRef          string            `json:"recibo_ref"`
		Puntuacion         string            `json:"puntuacion_autobaremo"`
		Repetida           bool              `json:"repetida"`
		Servicios          map[string]string `json:"servicios"`
	}{recibo.SolicitudRef, string(domain.EstadoPresentada), recibo.NumeroJustificante, httpcomun.Instante(recibo.PresentadaEn), recibo.ReciboRef,
		httpcomun.Puntos(recibo.Puntuacion), recibo.Reutilizada, servicios}))
}

func estadoCreacion(repetida bool) int {
	if repetida {
		return http.StatusOK
	}
	return http.StatusCreated
}

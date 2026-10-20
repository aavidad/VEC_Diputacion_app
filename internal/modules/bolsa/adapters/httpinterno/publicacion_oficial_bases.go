package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// RutaPublicacionesOficialesBases lista las versiones vigentes que RRHH puede
// declarar (GET) y registra la declaración de su publicación oficial (POST).
const RutaPublicacionesOficialesBases = "/api/vec/bolsa/convocatorias/publicaciones-oficiales"

// ConsultorPublicacionesOficiales devuelve solo las versiones sobre las que el
// perfil activo tiene la concesión de declarar; la composición audita la lectura.
type ConsultorPublicacionesOficiales interface {
	ConsultarPublicacionesOficiales(context.Context) (ports.PaginaVersionesPublicacionOficial, error)
}

// PreparadorPublicacionOficial añade identidad, correlación y motivo de la
// frontera. Nunca cambia la declaración recibida.
type PreparadorPublicacionOficial interface {
	PrepararDeclaracionPublicacionOficial(context.Context, ports.DeclaracionPublicacionOficial) (ports.SolicitudDeclaracionPublicacionOficial, error)
}

type HandlerPublicacionesOficialesBases struct {
	consultor  ConsultorPublicacionesOficiales
	preparador PreparadorPublicacionOficial
	servicio   ports.ServicioPublicacionOficial
}

// Sin servicio (falta el consumidor del núcleo o su material) la ruta responde
// 503 y nunca acepta una declaración.
func NuevoHandlerPublicacionesOficialesBases(c ConsultorPublicacionesOficiales, p PreparadorPublicacionOficial, s ports.ServicioPublicacionOficial) http.Handler {
	return &HandlerPublicacionesOficialesBases{c, p, s}
}

type versionPublicacionOficialHTTP struct {
	ConvocatoriaRef        string                           `json:"convocatoria_ref"`
	Secuencia              int64                            `json:"secuencia"`
	IdentificadorPublico   string                           `json:"identificador_publico"`
	Titulo                 string                           `json:"titulo"`
	HuellaVersionSHA256    string                           `json:"huella_version_sha256"`
	PlazoInscripcionAbreEn *time.Time                       `json:"plazo_inscripcion_abre_en"`
	Publicacion            *ports.PublicacionOficialVigente `json:"publicacion"`
}

func (h *HandlerPublicacionesOficialesBases) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaPublicacionesOficialesBases || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		responderCodigoComprobacion(w, 404, "recurso_no_encontrado")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.consultar(w, r)
	case http.MethodPost:
		h.declarar(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderCodigoComprobacion(w, 405, "metodo_no_permitido")
	}
}

func (h *HandlerPublicacionesOficialesBases) consultar(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.consultor == nil {
		responderCodigoComprobacion(w, 503, "publicacion_oficial_no_habilitada")
		return
	}
	if r.ContentLength > 0 || len(r.TransferEncoding) != 0 {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	pagina, err := h.consultor.ConsultarPublicacionesOficiales(r.Context())
	if err != nil {
		responderErrorPublicacionOficial(w, err)
		return
	}
	items := make([]versionPublicacionOficialHTTP, 0, len(pagina.Items))
	for _, v := range pagina.Items {
		items = append(items, versionPublicacionOficialHTTP{ConvocatoriaRef: v.ConvocatoriaRef, Secuencia: v.Secuencia,
			IdentificadorPublico: v.IdentificadorPublico, Titulo: v.Titulo, HuellaVersionSHA256: v.HuellaVersionSHA256,
			PlazoInscripcionAbreEn: v.PlazoInscripcionAbreEn, Publicacion: v.Publicacion})
	}
	responderComprobacion(w, 200, map[string]any{"data": map[string]any{"items": items, "completa": pagina.Completa}})
}

func (h *HandlerPublicacionesOficialesBases) declarar(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.preparador == nil || h.servicio == nil {
		responderCodigoComprobacion(w, 503, "publicacion_oficial_no_habilitada")
		return
	}
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > ports.MaximoSolicitudPublicacionOficialBasesJSON ||
		len(r.TransferEncoding) != 0 || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" ||
		len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" || len(r.Header.Values("Idempotency-Key")) != 1 {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	var cuerpo struct {
		ConvocatoriaRef     string  `json:"convocatoria_ref"`
		HuellaVersionSHA256 string  `json:"huella_version_sha256"`
		Medio               string  `json:"medio"`
		BoletinNumero       *int64  `json:"boletin_numero"`
		AnuncioRef          string  `json:"anuncio_ref"`
		FechaPublicacion    string  `json:"fecha_publicacion"`
		Direccion           *string `json:"direccion"`
		SustituyeRef        *string `json:"sustituye_ref"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, ports.MaximoSolicitudPublicacionOficialBasesJSON+1))
	d.DisallowUnknownFields()
	if d.Decode(&cuerpo) != nil || d.Decode(new(any)) != io.EOF {
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
		return
	}
	declaracion := ports.DeclaracionPublicacionOficial{AnuncioRef: cuerpo.AnuncioRef, BoletinNumero: cuerpo.BoletinNumero,
		ClaveIdempotencia: r.Header.Get("Idempotency-Key"), ConvocatoriaRef: cuerpo.ConvocatoriaRef, Direccion: cuerpo.Direccion,
		FechaPublicacion: cuerpo.FechaPublicacion, HuellaVersionSHA256: cuerpo.HuellaVersionSHA256, Medio: cuerpo.Medio,
		SustituyeRef: cuerpo.SustituyeRef}
	q, err := h.preparador.PrepararDeclaracionPublicacionOficial(r.Context(), declaracion)
	if err != nil {
		responderErrorPublicacionOficial(w, err)
		return
	}
	if !mismaDeclaracionHTTP(q.Declaracion, declaracion) {
		responderCodigoComprobacion(w, 503, "publicacion_oficial_no_habilitada")
		return
	}
	recibo, err := h.servicio.Declarar(r.Context(), q)
	if err != nil {
		responderErrorPublicacionOficial(w, err)
		return
	}
	status := http.StatusCreated
	if recibo.Reutilizada {
		status = http.StatusOK
	}
	responderComprobacion(w, status, map[string]any{"data": recibo})
}

func mismaDeclaracionHTTP(a, b ports.DeclaracionPublicacionOficial) bool {
	x, errA := json.Marshal(a)
	y, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(x) == string(y)
}

func responderErrorPublicacionOficial(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrPublicacionOficialNoConfirmada):
		responderCodigoComprobacion(w, 503, "resultado_no_confirmado")
	case errors.Is(err, vd.ErrAutorizacionDenegada), errors.Is(err, vd.ErrPermissionDenied):
		responderCodigoComprobacion(w, 403, "acceso_denegado")
	case errors.Is(err, ports.ErrPublicacionOficialNoVigente):
		responderCodigoComprobacion(w, 409, "version_no_vigente")
	case errors.Is(err, ports.ErrPublicacionOficialConflicto):
		responderCodigoComprobacion(w, 409, "publicacion_en_conflicto")
	case errors.Is(err, ports.ErrPublicacionOficialInvalida):
		responderCodigoComprobacion(w, 400, "solicitud_invalida")
	case errors.Is(err, ports.ErrPublicacionOficialNoDisponible):
		responderCodigoComprobacion(w, 503, "publicacion_oficial_no_habilitada")
	default:
		responderCodigoComprobacion(w, 503, "servicio_no_disponible")
	}
}

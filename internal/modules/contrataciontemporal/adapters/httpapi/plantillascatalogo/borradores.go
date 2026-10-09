package plantillascatalogo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	pgplantillas "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaBorradores               = "/api/vec/contratacion-temporal/expedientes/borradores"
	RutaBorradoresDisponibles    = RutaBorradores + "/disponibles"
	EsquemaBorradoresDisponibles = "vec.contratacion-temporal.borradores-disponibles.v1"
	MIMEDOCX                     = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	maxCuerpoBorrador            = 4 << 10
	maxDocumentoBorrador         = 2 << 20
)

var claveTipoBorrador = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
var reciboPublicacionDocumental = regexp.MustCompile(`^recibo:[0-9a-f-]{36}$`)

// ConsultorDetalle reusa el caso de uso con concesión V3 y recibo de lectura.
type ConsultorDetalle interface {
	Consultar(context.Context, ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error)
}
type ProveedorPlantillas interface {
	ObtenerPlantillasDocumento(context.Context, plantillasapp.SolicitudDocumental, time.Time) (*informejuridico.PlantillasBorrador, string, error)
}

// PlantillasFijadas se usa únicamente en el generador de una petición cuyo
// catálogo publicado ya superó la consulta autorizada. Evita una segunda
// lectura PG después de comprobar el tipo disponible.
type PlantillasFijadas struct {
	Instantanea *informejuridico.PlantillasBorrador
}

func (p PlantillasFijadas) ObtenerPlantillas(ctx context.Context, _ time.Time) (*informejuridico.PlantillasBorrador, error) {
	if ctx == nil || ctx.Err() != nil || p.Instantanea == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return p.Instantanea, nil
}

// GeneradorConInstantanea recibe exactamente la versión publicada comprobada
// para esta petición. La composición adapta el renderizador vivo y lo fija a
// esa instantánea, para que una publicación concurrente no mezcle texto/ref.
type GeneradorConInstantanea interface {
	Generar(context.Context, *informejuridico.PlantillasBorrador, string, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error)
}
type GeneradorConInstantaneaFunc func(context.Context, *informejuridico.PlantillasBorrador, string, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error)

func (f GeneradorConInstantaneaFunc) Generar(ctx context.Context, p *informejuridico.PlantillasBorrador, formato string, tipo ports.TipoBorradorRRHH, d ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error) {
	return f(ctx, p, formato, tipo, d)
}

type DocumentoCatalogado struct {
	Contenido            []byte
	Formato              string
	Tipo                 ports.TipoBorradorRRHH
	PlantillaRef         string
	CatalogoRef          string
	CatalogoHuellaSHA256 string
}

type ManejadorBorradores struct {
	consultor  ConsultorDetalle
	plantillas ProveedorPlantillas
	generador  GeneradorConInstantanea
	ahora      func() time.Time
}

func NuevoManejadorBorradores(consultor ConsultorDetalle, plantillas ProveedorPlantillas, generador GeneradorConInstantanea, ahora func() time.Time) (*ManejadorBorradores, error) {
	if consultor == nil || plantillas == nil || generador == nil || ahora == nil {
		return nil, errors.New("borradores RRHH: dependencias no disponibles")
	}
	return &ManejadorBorradores{consultor: consultor, plantillas: plantillas, generador: generador, ahora: ahora}, nil
}

type peticionBorrador struct {
	ExpedienteRef    string  `json:"expediente_ref"`
	VersionObservada *uint64 `json:"version_observada"`
	Tipo             string  `json:"tipo,omitempty"`
	Formato          string  `json:"formato,omitempty"`
}
type tipoDisponible struct {
	Clave    string   `json:"clave"`
	Etiqueta string   `json:"etiqueta"`
	Formatos []string `json:"formatos"`
	// Disponible dice si el expediente ya permite preparar el documento; si
	// no, la pantalla lo muestra deshabilitado en lugar de dejar que falle.
	Disponible bool `json:"disponible"`
}
type listaDisponibles struct {
	Esquema              string           `json:"esquema"`
	CatalogoRef          string           `json:"catalogo_ref"`
	CatalogoHuellaSHA256 string           `json:"catalogo_huella_sha256"`
	ProcedenciaRef       string           `json:"procedencia_ref"`
	Tipos                []tipoDisponible `json:"tipos"`
}

func (h *ManejadorBorradores) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	preparar(w)
	if h == nil || h.consultor == nil || h.plantillas == nil || h.generador == nil || h.ahora == nil || r == nil || r.URL == nil {
		fallo(w, 503, "no_disponible")
		return
	}
	if !rutaBorradorValida(r) {
		fallo(w, 404, "no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		fallo(w, 405, "metodo_no_permitido")
		return
	}
	if !origenPermitido(r) {
		fallo(w, 403, "denegado")
		return
	}
	listar := r.URL.Path == RutaBorradoresDisponibles
	esperadoAccept := "application/pdf"
	if listar {
		esperadoAccept = "application/json"
	}
	if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != esperadoAccept && !(!listar && r.Header.Get("Accept") == MIMEDOCX) || r.Body == nil || r.ContentLength > maxCuerpoBorrador || len(r.Trailer) != 0 {
		fallo(w, 400, "entrada_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCuerpoBorrador+1))
	if err != nil || len(b) == 0 || len(b) > maxCuerpoBorrador || !utf8.Valid(b) || !jsonSinDuplicados(b) {
		fallo(w, 400, "entrada_invalida")
		return
	}
	var pedido peticionBorrador
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&pedido) != nil || dec.Decode(&struct{}{}) != io.EOF || pedido.VersionObservada == nil {
		fallo(w, 400, "entrada_invalida")
		return
	}
	if listar {
		if pedido.Tipo != "" || pedido.Formato != "" {
			fallo(w, 400, "entrada_invalida")
			return
		}
	} else if !claveTipoBorrador.MatchString(pedido.Tipo) || (pedido.Formato != "pdf" && pedido.Formato != "docx") || pedido.Formato == "pdf" && r.Header.Get("Accept") != "application/pdf" || pedido.Formato == "docx" && r.Header.Get("Accept") != MIMEDOCX {
		fallo(w, 400, "entrada_invalida")
		return
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(pedido.ExpedienteRef, *pedido.VersionObservada)
	if err != nil {
		fallo(w, 400, "entrada_invalida")
		return
	}
	if err = r.Context().Err(); err != nil {
		fallo(w, 503, "no_disponible")
		return
	}
	detalle, err := h.consultor.Consultar(r.Context(), solicitud)
	if err != nil {
		falloConsultaBorrador(w, err)
		return
	}
	if detalle.ValidarContenidoPublicablePara(solicitud) != nil {
		fallo(w, 503, "no_disponible")
		return
	}
	operacion := "descargar"
	if listar {
		operacion = "listar"
	}
	material := plantillasapp.SolicitudDocumental{
		Operacion: operacion, OrganizacionRef: detalle.Resumen.OrganizacionRef,
		ClaseAmbito: string(ports.AmbitoOrganizacionRRHH), AmbitoRef: detalle.Resumen.OrganizacionRef,
		ExpedienteRef: detalle.Resumen.ExpedienteRef, VersionObservada: detalle.Resumen.Version,
		ConsultaHuellaSHA256: detalle.Lectura.ConsultaHuellaSHA256(),
	}
	if !listar {
		material.Tipo = pedido.Tipo
		material.Formato = pedido.Formato
	}
	instantanea, procedencia, err := h.plantillas.ObtenerPlantillasDocumento(r.Context(), material, h.ahora().UTC())
	if errors.Is(err, pgplantillas.ErrDenegado) || errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		fallo(w, 403, "denegado")
		return
	}
	if err != nil || instantanea == nil || !reciboPublicacionDocumental.MatchString(procedencia) {
		fallo(w, 503, "no_disponible")
		return
	}
	if listar {
		h.listar(w, instantanea, procedencia, detalle)
		return
	}
	h.descargar(w, r, instantanea, procedencia, detalle, pedido)
}

func rutaBorradorValida(r *http.Request) bool {
	u := r.URL
	return (u.Path == RutaBorradores || u.Path == RutaBorradoresDisponibles) && u.RawQuery == "" && !u.ForceQuery && u.RawPath == "" && u.Scheme == "" && u.Host == "" && u.User == nil && u.Opaque == "" && u.Fragment == "" && u.RawFragment == "" && u.EscapedPath() == u.Path
}

func (h *ManejadorBorradores) listar(w http.ResponseWriter, p *informejuridico.PlantillasBorrador, procedencia string, d ports.DetalleExpedienteRRHH) {
	tipos := make([]tipoDisponible, 0, len(p.Tipos()))
	for _, tipo := range p.Tipos() {
		plantilla, ok := p.Plantilla(tipo)
		if !ok || !plantilla.AdmiteModalidad(d.Resumen.ModalidadClave) || !accionPlantillaCumplida(plantilla, d) {
			continue
		}
		tipos = append(tipos, tipoDisponible{Clave: string(tipo), Etiqueta: plantilla.Nombre, Formatos: []string{"pdf", "docx"},
			Disponible: informejuridico.BorradorPreparable(p, tipo, d)})
	}
	responder(w, 200, listaDisponibles{Esquema: EsquemaBorradoresDisponibles, CatalogoRef: p.Referencia(), CatalogoHuellaSHA256: p.Huella(), ProcedenciaRef: procedencia, Tipos: tipos})
}

func (h *ManejadorBorradores) descargar(w http.ResponseWriter, r *http.Request, p *informejuridico.PlantillasBorrador, procedencia string, d ports.DetalleExpedienteRRHH, pedido peticionBorrador) {
	tipo := ports.TipoBorradorRRHH(pedido.Tipo)
	plantilla, ok := p.Plantilla(tipo)
	if !ok || !plantilla.AdmiteModalidad(d.Resumen.ModalidadClave) || !accionPlantillaCumplida(plantilla, d) {
		fallo(w, 409, "documento_no_disponible")
		return
	}
	resultado, err := h.generador.Generar(r.Context(), p, pedido.Formato, tipo, d.Clonar())
	if errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		fallo(w, 409, "documento_no_disponible")
		return
	}
	if err != nil || r.Context().Err() != nil {
		fallo(w, 503, "no_disponible")
		return
	}
	contenido := resultado.Contenido
	if len(contenido) == 0 || len(contenido) > maxDocumentoBorrador || resultado.Formato != pedido.Formato || resultado.Tipo != tipo || resultado.PlantillaRef != plantilla.Referencia || resultado.CatalogoRef != p.Referencia() || resultado.CatalogoHuellaSHA256 != p.Huella() {
		fallo(w, 503, "no_disponible")
		return
	}
	mime := "application/pdf"
	extension := "pdf"
	firma := []byte("%PDF-")
	if pedido.Formato == "docx" {
		mime = MIMEDOCX
		extension = "docx"
		firma = []byte("PK\x03\x04")
	}
	if !bytes.HasPrefix(contenido, firma) {
		fallo(w, 503, "no_disponible")
		return
	}
	suma := sha256.Sum256(contenido)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", `attachment; filename="`+pedido.Tipo+`-borrador.`+extension+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.Header().Set("X-VEC-Catalogo-Ref", p.Referencia())
	w.Header().Set("X-VEC-Catalogo-Huella-SHA256", p.Huella())
	w.Header().Set("X-VEC-Plantilla-Procedencia-Ref", procedencia)
	w.Header().Set("X-VEC-Documento-SHA256", hex.EncodeToString(suma[:]))
	w.WriteHeader(200)
	_, _ = w.Write(contenido)
}

func accionPlantillaCumplida(p informejuridico.PlantillaBorrador, d ports.DetalleExpedienteRRHH) bool {
	if p.RequiereAccion == "" {
		return true
	}
	for _, h := range d.Hitos {
		if string(h.AccionClave) == p.RequiereAccion {
			return true
		}
	}
	return false
}
func falloConsultaBorrador(w http.ResponseWriter, err error) {
	if errors.Is(err, ctapp.ErrConsultaRRHHNoObservable) {
		fallo(w, 404, "no_encontrado")
		return
	}
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, pgplantillas.ErrDenegado) {
		fallo(w, 403, "denegado")
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		fallo(w, 504, "no_disponible")
		return
	}
	fallo(w, 503, "no_disponible")
}

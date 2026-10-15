package ports

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrCarteraNoDisponible    = errors.New("meritos.error.cartera_no_disponible")
	ErrCarteraDenegada        = errors.New("meritos.error.cartera_denegada")
	ErrCarteraSinSesion       = errors.New("meritos.error.cartera_sin_sesion")
	ErrCarteraConflicto       = errors.New("meritos.error.cartera_conflicto")
	ErrCarteraEntradaInvalida = errors.New("meritos.error.cartera_entrada_invalida")
)

const (
	AccionConsultarCartera  = "meritos.cartera.consultar_propia"
	AccionRetirarCartera    = "meritos.cartera.retirar_propio"
	FinalidadCartera        = "gestion_cartera_propia"
	ModuloCartera           = "meritos"
	TipoRecursoCartera      = "cartera_personal"
	AudienciaRetirarCartera = "vec_meritos.cartera.retirar_propio.v1"
)

func CamposConsultaCartera() []string {
	return []string{"denominacion", "disponibilidad", "evidencias", "hecho_ref", "motivos_retirada", "retirada", "tipo", "total", "version_hecho"}
}

type MotivoRetiradaCartera struct {
	Motivo   vec.ReferenciaEntradaCatalogo `json:"motivo"`
	Etiqueta string                        `json:"etiqueta"`
}

type DisponibilidadCartera struct {
	Estado  string `json:"estado"`
	Version uint64 `json:"version"`
}

type RetiradaCartera struct {
	ReciboRef string    `json:"recibo_ref"`
	Fecha     time.Time `json:"fecha"`
}

// FichaCartera incluye las referencias documentales de la versión real del
// hecho. Los binarios y los puntos no pertenecen a esta lectura.
type FichaCartera struct {
	HechoRef       string                    `json:"hecho_ref"`
	VersionHecho   int                       `json:"version_hecho"`
	Tipo           string                    `json:"tipo"`
	Denominacion   string                    `json:"denominacion"`
	Evidencias     []vec.ReferenciaDocumento `json:"evidencias"`
	Disponibilidad DisponibilidadCartera     `json:"disponibilidad"`
	Retirada       *RetiradaCartera          `json:"retirada,omitempty"`
}

type ReciboLecturaCartera struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      string    `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CorrelacionRef string    `json:"correlacion_ref"`
	ConsultadaEn   time.Time `json:"consultada_en"`
}

// PaginaCarteraInterna solo circula entre el repositorio y la aplicación.
// CursorSiguienteHechoRef nunca se expone sin el sello del servicio.
type PaginaCarteraInterna struct {
	Hechos                  []FichaCartera          `json:"hechos"`
	MotivosRetirada         []MotivoRetiradaCartera `json:"motivos_retirada"`
	Total                   uint64                  `json:"total"`
	CursorSiguienteHechoRef string                  `json:"cursor_siguiente_hecho_ref"`
	CorteEn                 time.Time               `json:"corte_en"`
	ReciboAcceso            ReciboLecturaCartera    `json:"recibo_acceso"`
}

type PaginaCartera struct {
	Hechos          []FichaCartera          `json:"hechos"`
	MotivosRetirada []MotivoRetiradaCartera `json:"motivos_retirada"`
	Total           uint64                  `json:"total"`
	CursorSiguiente string                  `json:"cursor_siguiente,omitempty"`
	Filtro          string                  `json:"filtro"`
	CorteEn         time.Time               `json:"corte_en"`
	ReciboAcceso    ReciboLecturaCartera    `json:"recibo_acceso"`
}

func (p PaginaCarteraInterna) Validar(corte time.Time, correlacion string, limite int,
	filtro, ultimoLeido string) error {
	if p.Hechos == nil || len(p.Hechos) > limite || p.Total < uint64(len(p.Hechos)) ||
		len(p.MotivosRetirada) < 1 || len(p.MotivosRetirada) > 128 ||
		p.CorteEn.IsZero() || !p.CorteEn.Equal(corte) ||
		p.ReciboAcceso.AuditoriaRef == "" || p.ReciboAcceso.Secuencia == "" ||
		!huellaSHA256CarteraValida(p.ReciboAcceso.HuellaSHA256) ||
		p.ReciboAcceso.CorrelacionRef != correlacion ||
		p.ReciboAcceso.ConsultadaEn.IsZero() || p.ReciboAcceso.ConsultadaEn.Before(corte) ||
		p.CursorSiguienteHechoRef != "" && !domain.ReferenciaValida(p.CursorSiguienteHechoRef) {
		return ErrCarteraNoDisponible
	}
	motivos := make(map[string]struct{}, len(p.MotivosRetirada))
	for _, m := range p.MotivosRetirada {
		clave := m.Motivo.Referencia()
		if m.Motivo.Validar() != nil || m.Motivo.CatalogoID != "motivos_meritos_cartera" ||
			!strings.HasPrefix(m.Motivo.EntradaClave, "motivo_") ||
			len(m.Etiqueta) > 200 || strings.TrimSpace(m.Etiqueta) == "" {
			return ErrCarteraNoDisponible
		}
		if _, repetido := motivos[clave]; repetido {
			return ErrCarteraNoDisponible
		}
		motivos[clave] = struct{}{}
	}
	ultimo := ultimoLeido
	for _, h := range p.Hechos {
		if !domain.ReferenciaValida(h.HechoRef) || h.VersionHecho < 1 || h.Denominacion == "" ||
			h.HechoRef <= ultimo || !tipoHechoCarteraValido(h.Tipo) ||
			filtro == "disponibles" && h.Disponibilidad.Estado != "disponible" ||
			filtro == "retirados" && h.Disponibilidad.Estado != "retirado" ||
			h.Disponibilidad.Estado == "disponible" && (h.Disponibilidad.Version != 0 || h.Retirada != nil) ||
			h.Disponibilidad.Estado == "retirado" && (h.Disponibilidad.Version == 0 || h.Retirada == nil ||
				!domain.ReferenciaValida(h.Retirada.ReciboRef) || h.Retirada.Fecha.IsZero() ||
				h.Retirada.Fecha.After(corte)) ||
			h.Disponibilidad.Estado != "disponible" && h.Disponibilidad.Estado != "retirado" {
			return ErrCarteraNoDisponible
		}
		if len(h.Evidencias) > 32 {
			return ErrCarteraNoDisponible
		}
		for _, evidencia := range h.Evidencias {
			if evidencia.Validar() != nil || !domain.ReferenciaValida(evidencia.ID) {
				return ErrCarteraNoDisponible
			}
		}
		ultimo = h.HechoRef
	}
	if p.CursorSiguienteHechoRef != "" && (len(p.Hechos) != limite || p.CursorSiguienteHechoRef != ultimo) {
		return ErrCarteraNoDisponible
	}
	return nil
}

func tipoHechoCarteraValido(tipo string) bool {
	switch tipo {
	case "titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro":
		return true
	default:
		return false
	}
}

func huellaSHA256CarteraValida(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

// CapturaLecturaCartera solo se crea tras evaluación positiva de sesión y
// concesión. PostgreSQL vuelve a cotejar sus bytes actuales antes de leer.
type CapturaLecturaCartera struct {
	personaRef, correlacionRef string
	contexto, vinculo, orden   []byte
	validaHasta                time.Time
}

func NuevaCapturaLecturaCartera(persona, correlacion string,
	contexto, vinculo, orden []byte, hasta time.Time) (CapturaLecturaCartera, error) {
	if !domain.ReferenciaValida(persona) || !domain.ReferenciaValida(correlacion) ||
		len(contexto) == 0 || len(contexto) > 131072 || len(vinculo) == 0 || len(vinculo) > 65536 ||
		len(orden) == 0 || len(orden) > 16384 || hasta.IsZero() {
		return CapturaLecturaCartera{}, ErrCarteraNoDisponible
	}
	return CapturaLecturaCartera{persona, correlacion, slices.Clone(contexto),
		slices.Clone(vinculo), slices.Clone(orden), hasta.UTC()}, nil
}

func (c CapturaLecturaCartera) PersonaRef() string       { return c.personaRef }
func (c CapturaLecturaCartera) CorrelacionRef() string   { return c.correlacionRef }
func (c CapturaLecturaCartera) ContextoCanonico() []byte { return slices.Clone(c.contexto) }
func (c CapturaLecturaCartera) VinculoCanonico() []byte  { return slices.Clone(c.vinculo) }
func (c CapturaLecturaCartera) OrdenJSON() []byte        { return slices.Clone(c.orden) }
func (c CapturaLecturaCartera) ValidaHasta() time.Time   { return c.validaHasta }

type LectorCartera interface {
	LeerCartera(context.Context, CapturaLecturaCartera) (PaginaCarteraInterna, error)
}

type AuditorIntentosCartera interface {
	RegistrarIntentoCartera(context.Context, CapturaLecturaCartera) error
}

type AuditorIntentosRetiradaCartera interface {
	RegistrarIntentoRetiradaCartera(context.Context, CapturaLecturaCartera) error
}

// OrdenRetiradaCartera transporta el comando canónico y el material V3 de una
// decisión nueva. La función propietaria coteja ambos y aplica CAS en una TX.
type OrdenRetiradaCartera struct {
	PersonaRef                    string
	HechoRef                      string
	HechoVersionEsperada          int
	DisponibilidadVersionEsperada uint64
	ClaveIdempotencia             string
	HuellaComandoSHA256           string
	ComandoCanonico               []byte
	Material                      vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboRetiradaCartera struct {
	Referencia          string                `json:"referencia"`
	HechoRef            string                `json:"hecho_ref"`
	VersionHecho        int                   `json:"version_hecho"`
	Disponibilidad      DisponibilidadCartera `json:"disponibilidad"`
	RetiradoEn          time.Time             `json:"retirado_en"`
	AuditoriaRef        string                `json:"auditoria_ref"`
	EventoRef           string                `json:"evento_ref"`
	ClaveIdempotencia   string                `json:"clave_idempotencia"`
	HuellaComandoSHA256 string                `json:"huella_comando_sha256"`
}

type ResultadoRetiradaCartera struct {
	Codigo       string                 `json:"codigo"`
	AuditoriaRef string                 `json:"auditoria_ref"`
	Recibo       *ReciboRetiradaCartera `json:"recibo"`
}

type ConfirmacionRetiradaCartera struct {
	Recibo             ReciboRetiradaCartera `json:"recibo"`
	Repetida           bool                  `json:"repetida"`
	AuditoriaAccesoRef string                `json:"auditoria_acceso_ref"`
}

type RepositorioRetiradaCartera interface {
	RetirarCartera(context.Context, OrdenRetiradaCartera) (ResultadoRetiradaCartera, error)
}

type ProveedorMaterialRetiradaCartera interface {
	EmitirMaterialRetiradaCartera(context.Context, vec.SolicitudAutorizacionLigadaV3,
		vec.ResultadoContextoActorRegistradoV2, vec.DecisionAutorizacionLigadaV3,
		vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

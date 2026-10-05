package administracionperfiles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// SufijoPreparacionLote completa /personas/{persona_ref}. La unidad va en la
// consulta (unidad_ref) y es la única clave admitida.
const SufijoPreparacionLote = "/preparacion-lote"

// ServicioLotesADMIN es la autoridad que monta vec-admin para el lote: aplicar
// y preparar. No incluye actos singulares ni propuestas.
type ServicioLotesADMIN interface {
	ServicioLotes
	PrepararLoteOrdinario(context.Context, domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error)
}

// NuevoHandlerUsuariosMetadatosConLote conserva las dos lecturas nominales de
// usuarios y abre sólo la preparación (GET) y el lote ordinario (POST). La
// organización viene de la configuración privada, nunca de la petición.
func NuevoHandlerUsuariosMetadatosConLote(origen, organizacion string, sesiones ResolvedorSesion,
	lecturas FuenteLecturas, catalogo ports.CatalogoRolesAdministrables, lotes ServicioLotesADMIN, auditor AuditorFrontera) (*Handler, error) {
	if !organizacionPrivadaLote.MatchString(organizacion) || dependenciaNula(catalogo) || dependenciaNula(lotes) {
		return nil, ErrConfiguracionIncompleta
	}
	h, err := NuevoHandlerUsuariosMetadatos(origen, sesiones, lecturas, auditor)
	if err != nil {
		return nil, err
	}
	h.soloLectura, h.catalogo, h.lotes, h.organizacionLote = false, catalogo, lotes, organizacion
	return h, nil
}

type AltaPreparacion struct {
	RolVersionRef             string    `json:"rol_version_ref"`
	Nombre                    string    `json:"nombre"`
	UnidadRequerida           bool      `json:"unidad_requerida"`
	VigenteHastaMaxima        time.Time `json:"vigente_hasta_maxima"`
	DuracionPropuestaSegundos int64     `json:"duracion_propuesta_segundos"`
	PerfilRef                 string    `json:"perfil_ref"`
	VinculoRef                string    `json:"vinculo_ref"`
	HuellaSHA256              string    `json:"huella_sha256"`
}

type BajaPreparacion struct {
	RolVersionRef  string    `json:"rol_version_ref"`
	Nombre         string    `json:"nombre"`
	PerfilRef      string    `json:"perfil_ref"`
	PerfilVersion  uint64    `json:"perfil_version"`
	VinculoRef     string    `json:"vinculo_ref"`
	VinculoVersion uint64    `json:"vinculo_version"`
	VigenteDesde   time.Time `json:"vigente_desde"`
	VigenteHasta   time.Time `json:"vigente_hasta"`
	HuellaSHA256   string    `json:"huella_sha256"`
}

// PreparacionLote trae lo que la pantalla copia en cada cambio del lote: la
// preimagen común (persona, cuenta, procedencia) y la propia de cada opción.
type PreparacionLote struct {
	OperacionRef            string            `json:"operacion_ref"`
	AuditoriaRef            string            `json:"auditoria_ref"`
	PreparadaEn             time.Time         `json:"preparada_en"`
	PersonaRef              string            `json:"persona_ref"`
	PersonaVersion          uint64            `json:"persona_version"`
	CuentaRef               string            `json:"cuenta_ref"`
	CuentaVersion           uint64            `json:"cuenta_version"`
	ProcedenciaRef          string            `json:"procedencia_ref"`
	ProcedenciaVersion      uint64            `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string            `json:"procedencia_huella_sha256"`
	UnidadRef               string            `json:"unidad_ref"`
	Altas                   []AltaPreparacion `json:"altas"`
	Bajas                   []BajaPreparacion `json:"bajas"`
	Truncado                bool              `json:"truncado"`
}

func preparacionDTO(p domain.PreparacionLoteAdministracionPerfiles) PreparacionLote {
	r := PreparacionLote{OperacionRef: p.OperacionRef, AuditoriaRef: p.AuditoriaRef, PreparadaEn: p.PreparadaEn,
		PersonaRef: p.PersonaRef, PersonaVersion: p.PersonaVersion, CuentaRef: p.CuentaRef, CuentaVersion: p.CuentaVersion,
		ProcedenciaRef: p.ProcedenciaRef, ProcedenciaVersion: p.ProcedenciaVersion,
		ProcedenciaHuellaSHA256: p.ProcedenciaHuellaSHA256, UnidadRef: p.UnidadRef, Truncado: p.Truncado,
		Altas: make([]AltaPreparacion, 0, len(p.Altas)), Bajas: make([]BajaPreparacion, 0, len(p.Bajas))}
	for _, a := range p.Altas {
		r.Altas = append(r.Altas, AltaPreparacion{RolVersionRef: a.RolVersionRef, Nombre: a.Nombre,
			UnidadRequerida: a.UnidadRequerida, VigenteHastaMaxima: a.VigenteHastaMaxima,
			DuracionPropuestaSegundos: int64(a.DuracionPropuesta / time.Second), PerfilRef: a.PerfilRef,
			VinculoRef: a.VinculoRef, HuellaSHA256: a.HuellaSHA256})
	}
	for _, b := range p.Bajas {
		r.Bajas = append(r.Bajas, BajaPreparacion{RolVersionRef: b.RolVersionRef, Nombre: b.Nombre, PerfilRef: b.PerfilRef,
			PerfilVersion: b.PerfilVersion, VinculoRef: b.VinculoRef, VinculoVersion: b.VinculoVersion,
			VigenteDesde: b.VigenteDesde, VigenteHasta: b.VigenteHasta, HuellaSHA256: b.HuellaSHA256})
	}
	return r
}

// rutaPreparacionLote devuelve la persona de /personas/{ref}/preparacion-lote.
func rutaPreparacionLote(p string) (string, bool) {
	if !strings.HasPrefix(p, PrefijoV1+"/personas/") || !strings.HasSuffix(p, SufijoPreparacionLote) {
		return "", false
	}
	ref := strings.TrimSuffix(strings.TrimPrefix(p, PrefijoV1+"/personas/"), SufijoPreparacionLote)
	return ref, true
}

func unidadPreparacionLote(raw string) (string, bool) {
	q, err := url.ParseQuery(raw)
	if err != nil || len(q) != 1 || len(q["unidad_ref"]) != 1 || !organizacionPrivadaLote.MatchString(q["unidad_ref"][0]) {
		return "", false
	}
	return q["unidad_ref"][0], true
}

func nuevaReferenciaPreparacion() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "prep_admin:" + hex.EncodeToString(b[:]), nil
}

func (h *Handler) getPreparacionLote(w http.ResponseWriter, r *http.Request, s SesionConfiable, persona string) {
	// Destino propio en la auditoría: una preparación no es aplicar un lote.
	const accion = "preparar_lote_ordinario"
	if h.lotes == nil || h.organizacionLote == "" {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "servicio_no_disponible", accion, "")
		return
	}
	unidad, ok := unidadPreparacionLote(r.URL.RawQuery)
	if !refOpaca(persona, "per_") || !ok || persona == s.Actor.PersonaRef {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	operacion, err := nuevaReferenciaPreparacion()
	if err != nil {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "servicio_no_disponible", accion, persona)
		return
	}
	solicitud := domain.SolicitudPreparacionLoteAdministracionPerfiles{OperacionRef: operacion,
		OrganizacionRef: h.organizacionLote, UnidadRef: unidad, PersonaRef: persona,
		Actor: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
		CorrelacionRef: s.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, persona)
		return
	}
	p, err := h.lotes.PrepararLoteOrdinario(r.Context(), solicitud)
	if err != nil {
		h.denegarErrorLote(w, r, s, err, accion, persona)
		return
	}
	if p.ValidarPara(solicitud) != nil {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "respuesta_incompatible", accion, persona)
		return
	}
	jsonRespuesta(w, http.StatusOK, struct {
		Preparacion PreparacionLote `json:"preparacion"`
	}{preparacionDTO(p)})
}

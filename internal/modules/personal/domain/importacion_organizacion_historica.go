package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrImportacionOrganizacionInvalida     = errors.New("personal: importacion de organizacion invalida")
	ErrImportacionOrganizacionNoDisponible = errors.New("personal: importacion de organizacion no disponible")
	ErrImportacionOrganizacionConflicto    = errors.New("personal: importacion de organizacion en conflicto")
	ErrImportacionOrganizacionDenegada     = errors.New("personal: importacion de organizacion denegada")
	patronClaveImportacion                 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	patronUUIDImportacion                  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

type FaseImportacionOrganizacion string

const (
	FasePrepararOrganizacion         FaseImportacionOrganizacion = "preparar"
	FaseConciliarOrganizacion        FaseImportacionOrganizacion = "conciliar"
	FasePublicarOrganizacion         FaseImportacionOrganizacion = "publicar"
	AudienciaImportacionOrganizacion                             = "vec_personal.organizacion_historica.importar.v1"
)

func (f FaseImportacionOrganizacion) Accion() string {
	switch f {
	case FasePrepararOrganizacion:
		return "personal.organizacion_historica.preparar"
	case FaseConciliarOrganizacion:
		return "personal.organizacion_historica.conciliar"
	case FasePublicarOrganizacion:
		return "personal.organizacion_historica.publicar"
	default:
		return ""
	}
}

type ReferenciaCatalogoImportacion struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	Revision     int    `json:"revision"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaCatalogoImportacion) Validar() error {
	if !referenciaImportacion(r.ID) || r.Version < 1 || r.Revision < 1 || !huellaImportacion(r.HuellaSHA256) {
		return ErrImportacionOrganizacionInvalida
	}
	return nil
}

// El manifiesto conserva las fechas declaradas por la fuente, sin inferir
// efectos de la fecha de publicacion ni elevar una lectura PDF a autoridad.
type ManifiestoImportacionOrganizacion struct {
	OrganismoRef            string                        `json:"organismo_ref"`
	Tipo                    string                        `json:"tipo"`
	VersionRef              string                        `json:"version_ref"`
	VersionRevision         int                           `json:"version_revision"`
	VersionPreviaRef        string                        `json:"version_previa_ref,omitempty"`
	FuenteRef               string                        `json:"fuente_ref"`
	FuenteVersion           string                        `json:"fuente_version"`
	FuenteHuellaSHA256      string                        `json:"fuente_huella_sha256"`
	DocumentoRef            string                        `json:"documento_ref"`
	CustodiaRef             string                        `json:"custodia_ref"`
	DiccionarioRef          string                        `json:"diccionario_ref"`
	ActoRef                 string                        `json:"acto_ref"`
	AprobadaEn              FechaCivil                    `json:"aprobada_en,omitempty"`
	PublicadaEn             FechaCivil                    `json:"publicada_en,omitempty"`
	EfectosDesde            FechaCivil                    `json:"efectos_desde,omitempty"`
	EfectosHasta            FechaCivil                    `json:"efectos_hasta,omitempty"`
	CatalogoUnidades        ReferenciaCatalogoImportacion `json:"catalogo_unidades"`
	CatalogoClasificaciones ReferenciaCatalogoImportacion `json:"catalogo_clasificaciones"`
}

func (m ManifiestoImportacionOrganizacion) Validar(preparacion bool) error {
	if !referenciaImportacion(m.OrganismoRef) || (m.Tipo != "rpt" && m.Tipo != "plantilla") ||
		!patronUUIDImportacion.MatchString(m.VersionRef) || m.VersionRevision < 1 ||
		(m.VersionPreviaRef != "" && (!patronUUIDImportacion.MatchString(m.VersionPreviaRef) || m.VersionPreviaRef == m.VersionRef)) ||
		!referenciaImportacion(m.FuenteRef) || !textoImportacion(m.FuenteVersion, 160) || !huellaImportacion(m.FuenteHuellaSHA256) ||
		m.CatalogoUnidades.Validar() != nil || m.CatalogoClasificaciones.Validar() != nil {
		return ErrImportacionOrganizacionInvalida
	}
	if !referenciaOpcionalImportacion(m.DocumentoRef) || !referenciaOpcionalImportacion(m.CustodiaRef) ||
		!referenciaOpcionalImportacion(m.DiccionarioRef) || !referenciaOpcionalImportacion(m.ActoRef) ||
		!fechaOpcionalImportacion(m.AprobadaEn) || !fechaOpcionalImportacion(m.PublicadaEn) ||
		!fechaOpcionalImportacion(m.EfectosDesde) || !fechaOpcionalImportacion(m.EfectosHasta) {
		return ErrImportacionOrganizacionInvalida
	}
	if m.EfectosHasta != "" && m.EfectosDesde != "" && !m.EfectosDesde.AntesDe(m.EfectosHasta) {
		return ErrImportacionOrganizacionInvalida
	}
	if !preparacion && (m.DocumentoRef == "" || m.CustodiaRef == "" || m.DiccionarioRef == "" || m.ActoRef == "" ||
		m.AprobadaEn == "" || m.PublicadaEn == "" || m.EfectosDesde == "") {
		return ErrImportacionOrganizacionInvalida
	}
	return nil
}

func (m ManifiestoImportacionOrganizacion) HuellaSHA256() (string, error) {
	if m.Validar(true) != nil {
		return "", ErrImportacionOrganizacionInvalida
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", ErrImportacionOrganizacionInvalida
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Cada fila conserva su identidad tecnica y el codigo literal de origen por
// separado. La fecha de conocimiento y la procedencia aprobada las fija SQL
// al publicar; no se admiten del cliente como hechos administrativos.
type HechoImportacionOrganizacion struct {
	Clase                    string     `json:"clase"`
	HechoRef                 string     `json:"hecho_ref"`
	Revision                 int        `json:"revision"`
	FilaFuenteRef            string     `json:"fila_fuente_ref"`
	PaginaFuente             int        `json:"pagina_fuente,omitempty"`
	OrganismoRef             string     `json:"organismo_ref"`
	UnidadRef                string     `json:"unidad_ref"`
	VigenteDesde             FechaCivil `json:"vigente_desde"`
	VigenteHasta             FechaCivil `json:"vigente_hasta,omitempty"`
	CodigoFuente             string     `json:"codigo_fuente,omitempty"`
	CodigoDatosReservaFuente string     `json:"codigo_datos_reserva_fuente,omitempty"`
	Denominacion             string     `json:"denominacion,omitempty"`
	ClasificacionRef         string     `json:"clasificacion_ref,omitempty"`
	CatalogoEntradaClave     string     `json:"catalogo_entrada_clave,omitempty"`
	TipoUnidad               string     `json:"tipo_unidad,omitempty"`
	PadreRef                 string     `json:"padre_ref,omitempty"`
	TipoRef                  string     `json:"tipo_ref,omitempty"`
	TipoRevision             int        `json:"tipo_revision,omitempty"`
	PlazaRef                 string     `json:"plaza_ref,omitempty"`
	PlazaRevision            int        `json:"plaza_revision,omitempty"`
	PuestoRef                string     `json:"puesto_ref,omitempty"`
	PuestoRevision           int        `json:"puesto_revision,omitempty"`
	Cantidad                 int        `json:"cantidad,omitempty"`
	Reconciliacion           string     `json:"reconciliacion,omitempty"`
	RegimenRef               string     `json:"regimen_ref,omitempty"`
	FormaProvisionRef        string     `json:"forma_provision_ref,omitempty"`
	NivelDestino             int        `json:"nivel_destino,omitempty"`
	EstadoEstructural        string     `json:"estado_estructural,omitempty"`
	DotacionPresupuestaria   string     `json:"dotacion_presupuestaria,omitempty"`
}

func (h HechoImportacionOrganizacion) Validar(m ManifiestoImportacionOrganizacion) error {
	if !patronUUIDImportacion.MatchString(h.HechoRef) || h.Revision < 1 ||
		!referenciaImportacion(h.FilaFuenteRef) || h.PaginaFuente < 0 || h.PaginaFuente > 100000 ||
		h.OrganismoRef != m.OrganismoRef || !referenciaImportacion(h.UnidadRef) ||
		h.VigenteDesde.Validar() != nil || !fechaOpcionalImportacion(h.VigenteHasta) ||
		(h.VigenteHasta != "" && !h.VigenteDesde.AntesDe(h.VigenteHasta)) ||
		!codigoOpcionalImportacion(h.CodigoFuente) || !codigoOpcionalImportacion(h.CodigoDatosReservaFuente) ||
		!referenciaOpcionalImportacion(h.ClasificacionRef) || !referenciaOpcionalImportacion(h.CatalogoEntradaClave) ||
		!referenciaOpcionalImportacion(h.RegimenRef) || !referenciaOpcionalImportacion(h.FormaProvisionRef) ||
		(h.Denominacion != "" && !textoImportacion(h.Denominacion, 300)) {
		return ErrImportacionOrganizacionInvalida
	}
	if h.PadreRef != "" && !patronUUIDImportacion.MatchString(h.PadreRef) ||
		h.TipoRef != "" && !patronUUIDImportacion.MatchString(h.TipoRef) ||
		h.PlazaRef != "" && !patronUUIDImportacion.MatchString(h.PlazaRef) ||
		h.PuestoRef != "" && !patronUUIDImportacion.MatchString(h.PuestoRef) {
		return ErrImportacionOrganizacionInvalida
	}
	switch h.Clase {
	case "nodo":
		if h.CatalogoEntradaClave == "" || h.Denominacion == "" ||
			(h.TipoUnidad != "delegacion" && h.TipoUnidad != "centro" && h.TipoUnidad != "puesto_responsabilidad") ||
			(h.TipoUnidad == "delegacion" && h.PadreRef != "") || m.Tipo != "rpt" ||
			h.CodigoFuente != "" || h.CodigoDatosReservaFuente != "" || h.ClasificacionRef != "" || h.TipoRef != "" || h.TipoRevision != 0 ||
			h.PlazaRef != "" || h.PlazaRevision != 0 || h.PuestoRef != "" || h.PuestoRevision != 0 || h.Cantidad != 0 || h.Reconciliacion != "" ||
			h.RegimenRef != "" || h.FormaProvisionRef != "" || h.NivelDestino != 0 || h.EstadoEstructural != "" || h.DotacionPresupuestaria != "" {
			return ErrImportacionOrganizacionInvalida
		}
	case "puesto_tipo":
		if m.Tipo != "rpt" || h.CodigoFuente == "" || h.Denominacion == "" || h.ClasificacionRef == "" || h.NivelDestino < 0 || h.NivelDestino > 30 ||
			h.CatalogoEntradaClave != "" || h.TipoUnidad != "" || h.PadreRef != "" || h.TipoRef != "" || h.TipoRevision != 0 ||
			h.PlazaRef != "" || h.PlazaRevision != 0 || h.PuestoRef != "" || h.PuestoRevision != 0 || h.Cantidad != 0 || h.Reconciliacion != "" ||
			h.EstadoEstructural != "" || h.DotacionPresupuestaria != "" {
			return ErrImportacionOrganizacionInvalida
		}
	case "dotacion":
		if m.Tipo != "rpt" || h.TipoRef == "" || h.TipoRevision < 1 || h.Cantidad < 1 || h.Cantidad > 100000 ||
			(h.Reconciliacion != "pendiente" && h.Reconciliacion != "parcial" && h.Reconciliacion != "reconciliada") ||
			h.CodigoFuente != "" || h.CodigoDatosReservaFuente != "" || h.Denominacion != "" || h.ClasificacionRef != "" || h.CatalogoEntradaClave != "" ||
			h.TipoUnidad != "" || h.PadreRef != "" || h.PlazaRef != "" || h.PlazaRevision != 0 || h.PuestoRef != "" || h.PuestoRevision != 0 ||
			h.RegimenRef != "" || h.FormaProvisionRef != "" || h.NivelDestino != 0 || h.EstadoEstructural != "" || h.DotacionPresupuestaria != "" {
			return ErrImportacionOrganizacionInvalida
		}
	case "plaza":
		if m.Tipo != "plantilla" || h.CodigoFuente == "" || h.ClasificacionRef == "" ||
			(h.EstadoEstructural != "vigente" && h.EstadoEstructural != "amortizada") ||
			(h.DotacionPresupuestaria != "acreditada" && h.DotacionPresupuestaria != "no_acreditada" && h.DotacionPresupuestaria != "desconocida") ||
			h.CodigoDatosReservaFuente != "" || h.Denominacion != "" || h.CatalogoEntradaClave != "" || h.TipoUnidad != "" || h.PadreRef != "" ||
			h.TipoRef != "" || h.TipoRevision != 0 || h.PlazaRef != "" || h.PlazaRevision != 0 || h.PuestoRef != "" || h.PuestoRevision != 0 ||
			h.Cantidad != 0 || h.Reconciliacion != "" || h.RegimenRef != "" || h.FormaProvisionRef != "" || h.NivelDestino != 0 {
			return ErrImportacionOrganizacionInvalida
		}
	case "puesto_individual":
		if m.Tipo != "rpt" || h.CodigoFuente == "" || h.TipoRef == "" || h.TipoRevision < 1 ||
			(h.EstadoEstructural != "vigente" && h.EstadoEstructural != "suprimido") ||
			h.CodigoDatosReservaFuente != "" || h.Denominacion != "" || h.ClasificacionRef != "" || h.CatalogoEntradaClave != "" || h.TipoUnidad != "" ||
			h.PadreRef != "" || h.PlazaRef != "" || h.PlazaRevision != 0 || h.PuestoRef != "" || h.PuestoRevision != 0 || h.Cantidad != 0 ||
			h.Reconciliacion != "" || h.RegimenRef != "" || h.FormaProvisionRef != "" || h.NivelDestino != 0 || h.DotacionPresupuestaria != "" {
			return ErrImportacionOrganizacionInvalida
		}
	case "vinculo":
		if h.PlazaRef == "" || h.PlazaRevision < 1 || h.PuestoRef == "" || h.PuestoRevision < 1 ||
			(h.Reconciliacion != "pendiente" && h.Reconciliacion != "confirmado" && h.Reconciliacion != "terminado") ||
			h.CodigoFuente != "" || h.CodigoDatosReservaFuente != "" || h.Denominacion != "" || h.ClasificacionRef != "" || h.CatalogoEntradaClave != "" ||
			h.TipoUnidad != "" || h.PadreRef != "" || h.TipoRef != "" || h.TipoRevision != 0 || h.Cantidad != 0 || h.RegimenRef != "" ||
			h.FormaProvisionRef != "" || h.NivelDestino != 0 || h.EstadoEstructural != "" || h.DotacionPresupuestaria != "" {
			return ErrImportacionOrganizacionInvalida
		}
	default:
		return ErrImportacionOrganizacionInvalida
	}
	return nil
}

type DecisionConciliacionOrganizacion struct {
	FilaFuenteRef string `json:"fila_fuente_ref"`
	Clase         string `json:"clase"`
	DestinoRef    string `json:"destino_ref,omitempty"`
	Resultado     string `json:"resultado"`
	Motivo        string `json:"motivo"`
	EvidenciaRef  string `json:"evidencia_ref"`
}

func (d DecisionConciliacionOrganizacion) Validar() error {
	if !referenciaImportacion(d.FilaFuenteRef) || !referenciaOpcionalImportacion(d.DestinoRef) ||
		!referenciaImportacion(d.EvidenciaRef) || !textoImportacion(d.Motivo, 2048) {
		return ErrImportacionOrganizacionInvalida
	}
	switch d.Clase {
	case "unidad", "clasificacion", "puesto_tipo", "dotacion", "plaza", "puesto_individual", "vinculo":
	default:
		return ErrImportacionOrganizacionInvalida
	}
	switch d.Resultado {
	case "vinculada":
		if d.DestinoRef == "" {
			return ErrImportacionOrganizacionInvalida
		}
	case "pendiente", "descartada":
		if d.DestinoRef != "" {
			return ErrImportacionOrganizacionInvalida
		}
	default:
		return ErrImportacionOrganizacionInvalida
	}
	return nil
}

// Actor y revision proceden del servidor y del ultimo recibo, respectivamente.
// El repositorio vuelve a comprobarlos bajo bloqueo antes de cualquier efecto.
type SolicitudImportacionOrganizacion struct {
	Fase              FaseImportacionOrganizacion        `json:"fase"`
	LoteRef           string                             `json:"lote_ref,omitempty"`
	RevisionEsperada  int64                              `json:"revision_esperada"`
	ClaveIdempotencia string                             `json:"clave_idempotencia"`
	CorrelacionRef    string                             `json:"correlacion_ref"`
	Manifiesto        ManifiestoImportacionOrganizacion  `json:"manifiesto"`
	Hechos            []HechoImportacionOrganizacion     `json:"hechos,omitempty"`
	Decisiones        []DecisionConciliacionOrganizacion `json:"decisiones,omitempty"`
	RevisorActorRef   string                             `json:"revisor_actor_ref,omitempty"`
	Actor             core.ContextoActor                 `json:"-"`
}

func (s SolicitudImportacionOrganizacion) Validar() error {
	if s.Fase.Accion() == "" || !patronClaveImportacion.MatchString(s.ClaveIdempotencia) ||
		!referenciaImportacion(s.CorrelacionRef) || s.Actor.Validar() != nil ||
		s.Manifiesto.Validar(s.Fase != FasePublicarOrganizacion) != nil {
		return ErrImportacionOrganizacionInvalida
	}
	if s.Fase == FasePrepararOrganizacion {
		if s.LoteRef != "" || s.RevisionEsperada != 0 || len(s.Decisiones) != 0 || s.RevisorActorRef != "" || len(s.Hechos) == 0 || len(s.Hechos) > 3000 {
			return ErrImportacionOrganizacionInvalida
		}
	} else if !referenciaImportacion(s.LoteRef) || s.RevisionEsperada < 1 {
		return ErrImportacionOrganizacionInvalida
	}
	if s.Fase == FasePrepararOrganizacion {
		vistos := make(map[string]struct{}, len(s.Hechos))
		for _, h := range s.Hechos {
			if h.Validar(s.Manifiesto) != nil {
				return ErrImportacionOrganizacionInvalida
			}
			k := h.Clase + "\x00" + h.HechoRef
			if _, ok := vistos[k]; ok {
				return ErrImportacionOrganizacionInvalida
			}
			vistos[k] = struct{}{}
		}
	} else if len(s.Hechos) != 0 {
		return ErrImportacionOrganizacionInvalida
	}
	if s.Fase == FasePublicarOrganizacion {
		if len(s.Decisiones) != 0 || !referenciaImportacion(s.RevisorActorRef) || s.RevisorActorRef != s.Actor.Principal.ID {
			return ErrImportacionOrganizacionInvalida
		}
	} else if s.RevisorActorRef != "" {
		return ErrImportacionOrganizacionInvalida
	}
	if s.Fase == FaseConciliarOrganizacion {
		if len(s.Decisiones) == 0 || len(s.Decisiones) > 1000 {
			return ErrImportacionOrganizacionInvalida
		}
		vistas := make(map[string]struct{}, len(s.Decisiones))
		for _, d := range s.Decisiones {
			if d.Validar() != nil {
				return ErrImportacionOrganizacionInvalida
			}
			k := d.Clase + "\x00" + d.FilaFuenteRef
			if _, ok := vistas[k]; ok {
				return ErrImportacionOrganizacionInvalida
			}
			vistas[k] = struct{}{}
		}
	} else if len(s.Decisiones) != 0 {
		return ErrImportacionOrganizacionInvalida
	}
	return nil
}

type MaterialImportacionOrganizacion struct {
	solicitud SolicitudImportacionOrganizacion
	canonico  []byte
	huella    string
	recurso   core.RecursoAutorizable
}

func NuevoMaterialImportacionOrganizacion(s SolicitudImportacionOrganizacion) (MaterialImportacionOrganizacion, error) {
	if s.Validar() != nil {
		return MaterialImportacionOrganizacion{}, ErrImportacionOrganizacionInvalida
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialImportacionOrganizacion{}, ErrImportacionOrganizacionInvalida
	}
	s.Actor = actor
	s.Hechos = append([]HechoImportacionOrganizacion(nil), s.Hechos...)
	s.Decisiones = append([]DecisionConciliacionOrganizacion(nil), s.Decisiones...)
	sort.Slice(s.Hechos, func(i, j int) bool {
		if s.Hechos[i].Clase != s.Hechos[j].Clase {
			return s.Hechos[i].Clase < s.Hechos[j].Clase
		}
		return s.Hechos[i].HechoRef < s.Hechos[j].HechoRef
	})
	sort.Slice(s.Decisiones, func(i, j int) bool {
		if s.Decisiones[i].Clase != s.Decisiones[j].Clase {
			return s.Decisiones[i].Clase < s.Decisiones[j].Clase
		}
		return s.Decisiones[i].FilaFuenteRef < s.Decisiones[j].FilaFuenteRef
	})
	contenido := struct {
		Esquema           string                             `json:"esquema"`
		Fase              FaseImportacionOrganizacion        `json:"fase"`
		LoteRef           string                             `json:"lote_ref"`
		RevisionEsperada  int64                              `json:"revision_esperada"`
		ClaveIdempotencia string                             `json:"clave_idempotencia"`
		CorrelacionRef    string                             `json:"correlacion_ref"`
		Manifiesto        ManifiestoImportacionOrganizacion  `json:"manifiesto"`
		Hechos            []HechoImportacionOrganizacion     `json:"hechos"`
		Decisiones        []DecisionConciliacionOrganizacion `json:"decisiones"`
		RevisorActorRef   string                             `json:"revisor_actor_ref"`
		ActorRef          string                             `json:"actor_ref"`
		ContextoRef       string                             `json:"contexto_ref"`
		PerfilRef         string                             `json:"perfil_ref"`
		PersonaVersion    uint64                             `json:"persona_version"`
		PerfilVersion     uint64                             `json:"perfil_version"`
	}{"vec.personal.importacion-organizacion.v1", s.Fase, s.LoteRef, s.RevisionEsperada, s.ClaveIdempotencia, s.CorrelacionRef, s.Manifiesto, s.Hechos, s.Decisiones, s.RevisorActorRef,
		s.Actor.Principal.ID, s.Actor.Instantanea.VinculoRef, s.Actor.PerfilActivoRef, s.Actor.Instantanea.PersonaVersion, s.Actor.Instantanea.PerfilVersion}
	canonico, err := json.Marshal(contenido)
	if err != nil {
		return MaterialImportacionOrganizacion{}, ErrImportacionOrganizacionInvalida
	}
	if len(canonico) > 4<<20 {
		return MaterialImportacionOrganizacion{}, ErrImportacionOrganizacionInvalida
	}
	h := sha256.Sum256(canonico)
	huella := hex.EncodeToString(h[:])
	referencia := s.LoteRef
	if referencia == "" {
		referencia = s.Manifiesto.OrganismoRef
	}
	recurso := core.RecursoAutorizable{Referencia: referencia, ModuloID: "personal", Tipo: "importacion_organizacion_historica",
		Ambitos:   map[string]string{"organismo_ref": s.Manifiesto.OrganismoRef},
		Atributos: map[string]string{"material_sha256": huella, "fase": string(s.Fase), "fuente_ref": s.Manifiesto.FuenteRef, "lote_ref": referencia}}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialImportacionOrganizacion{}, ErrImportacionOrganizacionInvalida
	}
	return MaterialImportacionOrganizacion{solicitud: s, canonico: canonico, huella: huella, recurso: recurso}, nil
}

func (m MaterialImportacionOrganizacion) Solicitud() SolicitudImportacionOrganizacion {
	s := m.solicitud
	actor, err := s.Actor.Clonar()
	if err != nil {
		return SolicitudImportacionOrganizacion{}
	}
	s.Actor = actor
	s.Hechos = append([]HechoImportacionOrganizacion(nil), s.Hechos...)
	s.Decisiones = append([]DecisionConciliacionOrganizacion(nil), s.Decisiones...)
	return s
}
func (m MaterialImportacionOrganizacion) Canonico() []byte     { return append([]byte(nil), m.canonico...) }
func (m MaterialImportacionOrganizacion) HuellaSHA256() string { return m.huella }
func (m MaterialImportacionOrganizacion) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = map[string]string{"organismo_ref": r.Ambitos["organismo_ref"]}
	r.Atributos = map[string]string{"material_sha256": r.Atributos["material_sha256"], "fase": r.Atributos["fase"], "fuente_ref": r.Atributos["fuente_ref"], "lote_ref": r.Atributos["lote_ref"]}
	return r
}

func referenciaImportacion(v string) bool {
	return len(v) >= 3 && len(v) <= 256 && v == strings.TrimSpace(v) && !strings.ContainsAny(v, "\x00\r\n\t")
}
func referenciaOpcionalImportacion(v string) bool { return v == "" || referenciaImportacion(v) }
func codigoOpcionalImportacion(v string) bool {
	return v == "" || (len(v) <= 160 && v == strings.TrimSpace(v) && !strings.ContainsAny(v, "\x00\r\n\t"))
}
func textoImportacion(v string, max int) bool {
	return v != "" && len(v) <= max && v == strings.TrimSpace(v) && !strings.ContainsAny(v, "\x00\r\n\t")
}
func huellaImportacion(v string) bool {
	return len(v) == 64 && patronHuellaOrganizacionHistorica.MatchString(v)
}
func fechaOpcionalImportacion(v FechaCivil) bool { return v == "" || v.Validar() == nil }
func InstanteImportacionValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}

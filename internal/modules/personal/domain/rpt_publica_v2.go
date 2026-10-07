package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrRPTPublicaV2Invalida     = errors.New("personal: consulta RPT publicada v2 invalida")
	ErrRPTPublicaV2Denegada     = errors.New("personal: consulta RPT publicada v2 denegada")
	ErrRPTPublicaV2NoDisponible = errors.New("personal: RPT publicada v2 no disponible")
	patronReferenciaRPTV2       = regexp.MustCompile(`^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$`)
	patronHuellaRPTV2           = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronCentroRPTV2           = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

const (
	AccionConsultaRPTPublicaV2    = "personal.rpt_publica.consultar"
	AudienciaConsultaRPTPublicaV2 = "vec_personal.rpt_publica.consultar.v2"
	FinalidadConsultaRPTPublicaV2 = "consultar_organizacion_publicada"
	EsquemaCandidatoRPTPublicaV2  = "vec.catalogo.rpt.candidato.v1"
	EstadoCandidatoRPTPublicaV2   = "preparacion_no_autoritativa"
	LimiteMaximoRPTPublicaV2      = 100
)

// La decisión debe cubrir el sobre de respuesta completo. Una concesión que
// permita solo metadatos no autoriza entregar items, totales ni evidencia.
func CamposRespuestaRPTPublicaV2() []string {
	return []string{
		"categorias_pendientes_grupo", "corte", "esquema", "estado", "evidencia", "fuente", "huella_sha256",
		"items", "limit", "offset", "publicacion_ref", "resumen", "total", "vista",
	}
}

// La proyección distingue referencias detectadas en el PDF de categorías
// gobernadas. Puestos y Dotacion de cada categoría solo cuentan filas con
// grupo acreditado por el preparador: no equivalen a todos los enlaces por
// CategoriaClave. Una alternativa nunca habilita un uso ni reparte dotación.
type CategoriaRPTPublicaV2 struct {
	Clave                                     string   `json:"clave"`
	Denominacion                              string   `json:"denominacion"`
	Origen                                    string   `json:"origen"`
	Grupos                                    []string `json:"grupos"`
	Escalas                                   []string `json:"escalas"`
	Puestos                                   int      `json:"puestos"`
	Dotacion                                  int      `json:"dotacion"`
	NivelDestinoMediana                       int      `json:"nivel_destino_mediana"`
	ComplementoEspecificoAnualCentimosMediana int      `json:"complemento_especifico_anual_centimos_mediana"`
}

type CategoriaPendienteRPTPublicaV2 struct {
	Denominacion string `json:"denominacion"`
	Origen       string `json:"origen"`
}

type PuestoRPTPublicoV2 struct {
	PuestoRPTPublico
	CategoriasClaves     []string                         `json:"categorias_claves"`
	CategoriasPendientes []CategoriaPendienteRPTPublicaV2 `json:"categorias_pendientes"`
}

type CatalogoRPTPublicaV2 struct {
	Esquema                   string                  `json:"esquema"`
	Estado                    string                  `json:"estado"`
	Fuente                    FuenteRPTPublica        `json:"fuente"`
	Resumen                   ResumenRPTPublica       `json:"resumen"`
	Categorias                []CategoriaRPTPublicaV2 `json:"categorias"`
	Puestos                   []PuestoRPTPublicoV2    `json:"puestos"`
	CategoriasPendientesGrupo []string                `json:"categorias_pendientes_grupo"`
}

// SnapshotRPTPublicaV2 es una única lectura validada del fichero. Corte es la
// fecha del documento publicado, no fecha de efectos ni de vigencia de puesto.
type SnapshotRPTPublicaV2 struct {
	PublicacionRef string
	Corte          string
	HuellaSHA256   string
	Catalogo       CatalogoRPTPublicaV2
}

func (s SnapshotRPTPublicaV2) Validar() error {
	if !patronReferenciaRPTV2.MatchString(s.PublicacionRef) || !fechaDocumentoRPTV2Valida(s.Corte) || !patronHuellaRPTV2.MatchString(s.HuellaSHA256) || s.Catalogo.Validar() != nil {
		return ErrRPTPublicaV2NoDisponible
	}
	return nil
}

func (c CatalogoRPTPublicaV2) Validar() error {
	if c.Esquema != EsquemaCandidatoRPTPublicaV2 || c.Estado != EstadoCandidatoRPTPublicaV2 ||
		!textoRPTPublico(c.Fuente.Documento, 1024) || !textoRPTPublico(c.Fuente.Importacion, 256) ||
		!fechaDocumentoRPTV2Valida(c.Fuente.GeneradoEn) || !textoRPTPublico(c.Fuente.Aviso, 4096) ||
		len(c.Categorias) == 0 || len(c.Categorias) > 10000 || len(c.Puestos) == 0 || len(c.Puestos) > 100000 {
		return ErrRPTPublicaV2NoDisponible
	}
	categorias, puestos, centros, pendientes := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, categoria := range c.Categorias {
		if !patronClaveRPTPublica.MatchString(categoria.Clave) || !textoRPTPublico(categoria.Denominacion, 512) ||
			(categoria.Origen != "categoria" && categoria.Origen != "denominacion") || len(categoria.Grupos) == 0 ||
			categoria.Puestos < 0 || categoria.Dotacion < 0 || categoria.NivelDestinoMediana < 0 || categoria.NivelDestinoMediana > 99 ||
			categoria.ComplementoEspecificoAnualCentimosMediana < 0 {
			return ErrRPTPublicaV2NoDisponible
		}
		if _, ok := categorias[categoria.Clave]; ok {
			return ErrRPTPublicaV2NoDisponible
		}
		categorias[categoria.Clave] = struct{}{}
		for _, lista := range [][]string{categoria.Grupos, categoria.Escalas} {
			for _, valor := range lista {
				if !textoRPTPublico(valor, 64) {
					return ErrRPTPublicaV2NoDisponible
				}
			}
		}
	}
	dotacion := 0
	for _, puesto := range c.Puestos {
		if puesto.PuestoRPTPublico.Validar() != nil {
			return ErrRPTPublicaV2NoDisponible
		}
		if _, ok := puestos[puesto.Codigo]; ok {
			return ErrRPTPublicaV2NoDisponible
		}
		puestos[puesto.Codigo] = struct{}{}
		centros[puesto.CentroCodigo] = struct{}{}
		dotacion += puesto.Dotacion
		refs := map[string]struct{}{}
		for _, clave := range puesto.CategoriasClaves {
			if _, ok := categorias[clave]; !ok {
				return ErrRPTPublicaV2NoDisponible
			}
			if _, ok := refs[clave]; ok {
				return ErrRPTPublicaV2NoDisponible
			}
			refs[clave] = struct{}{}
		}
		if puesto.CategoriaClave != "" && (len(puesto.CategoriasClaves) != 1 || puesto.CategoriaClave != puesto.CategoriasClaves[0]) {
			return ErrRPTPublicaV2NoDisponible
		}
		for _, pendiente := range puesto.CategoriasPendientes {
			if !textoRPTPublico(pendiente.Denominacion, 512) || (pendiente.Origen != "categoria" && pendiente.Origen != "denominacion") {
				return ErrRPTPublicaV2NoDisponible
			}
			pendientes[pendiente.Denominacion] = struct{}{}
		}
	}
	declaradas := map[string]struct{}{}
	for _, pendiente := range c.CategoriasPendientesGrupo {
		if !textoRPTPublico(pendiente, 512) || !contieneRPTV2(pendientes, pendiente) || contieneRPTV2(declaradas, pendiente) {
			return ErrRPTPublicaV2NoDisponible
		}
		declaradas[pendiente] = struct{}{}
	}
	if c.Resumen.Puestos != len(c.Puestos) || c.Resumen.Dotacion != dotacion || c.Resumen.Categorias != len(c.Categorias) ||
		c.Resumen.Centros != len(centros) || len(declaradas) != len(pendientes) {
		return ErrRPTPublicaV2NoDisponible
	}
	return nil
}

func contieneRPTV2(conjunto map[string]struct{}, clave string) bool {
	_, ok := conjunto[clave]
	return ok
}

func fechaDocumentoRPTV2Valida(v string) bool {
	fecha, err := time.Parse("2006-01-02", v)
	return err == nil && fecha.Format("2006-01-02") == v
}

type FiltroRPTPublicaV2 struct {
	Vista          string `json:"vista"`
	Q              string `json:"q"`
	Limite         int    `json:"limite"`
	Offset         int    `json:"offset"`
	CategoriaClave string `json:"categoria_clave"`
	CentroCodigo   string `json:"centro_codigo"`
}

func (f FiltroRPTPublicaV2) Validar() error {
	if (f.Vista != "categorias" && f.Vista != "puestos") || f.Q != strings.TrimSpace(f.Q) || !utf8.ValidString(f.Q) ||
		utf8.RuneCountInString(f.Q) > 100 || strings.ContainsAny(f.Q, "\x00\r\n\t") || f.Limite < 1 ||
		f.Limite > LimiteMaximoRPTPublicaV2 || f.Offset < 0 || f.Offset > 100000 ||
		(f.CategoriaClave != "" && !patronClaveRPTPublica.MatchString(f.CategoriaClave)) ||
		(f.CentroCodigo != "" && !patronCentroRPTV2.MatchString(f.CentroCodigo)) ||
		(f.Vista != "puestos" && (f.CategoriaClave != "" || f.CentroCodigo != "")) {
		return ErrRPTPublicaV2Invalida
	}
	return nil
}

type SolicitudConsultaRPTPublicaV2 struct {
	Actor    core.ContextoActor
	Filtro   FiltroRPTPublicaV2
	Snapshot SnapshotRPTPublicaV2
}

type MaterialConsultaRPTPublicaV2 struct {
	solicitud SolicitudConsultaRPTPublicaV2
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func NuevoMaterialConsultaRPTPublicaV2(s SolicitudConsultaRPTPublicaV2) (MaterialConsultaRPTPublicaV2, error) {
	if s.Actor.Validar() != nil || s.Filtro.Validar() != nil || s.Snapshot.Validar() != nil {
		return MaterialConsultaRPTPublicaV2{}, ErrRPTPublicaV2Invalida
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return MaterialConsultaRPTPublicaV2{}, ErrRPTPublicaV2Invalida
	}
	s.Actor = actor
	// El fichero se conserva en el servicio; el material sólo transporta su
	// identidad y el filtro. SQL liga estos bytes a la concesión y al recibo.
	canonico, err := json.Marshal(struct {
		Esquema          string             `json:"esquema"`
		PublicacionRef   string             `json:"publicacion_ref"`
		Corte            string             `json:"corte"`
		HuellaSHA256     string             `json:"huella_sha256"`
		Filtro           FiltroRPTPublicaV2 `json:"filtro"`
		ActorRef         string             `json:"actor_ref"`
		ContextoActorRef string             `json:"contexto_actor_ref"`
		ContextoVersion  uint64             `json:"contexto_version"`
		PersonaVersion   uint64             `json:"persona_version"`
		PerfilRef        string             `json:"perfil_ref"`
		PerfilVersion    uint64             `json:"perfil_version"`
	}{"vec.personal.rpt-publica.consulta.v2", s.Snapshot.PublicacionRef, s.Snapshot.Corte, s.Snapshot.HuellaSHA256,
		s.Filtro, actor.Principal.ID, actor.Instantanea.VinculoRef, actor.Instantanea.VinculoVersion,
		actor.Instantanea.PersonaVersion, actor.PerfilActivoRef, actor.Instantanea.PerfilVersion})
	if err != nil {
		return MaterialConsultaRPTPublicaV2{}, ErrRPTPublicaV2Invalida
	}
	suma := sha256.Sum256(canonico)
	recurso := core.RecursoAutorizable{Referencia: s.Snapshot.PublicacionRef, ModuloID: "personal", Tipo: "rpt_publica_publicacion",
		Ambitos: map[string]string{"publicacion_ref": s.Snapshot.PublicacionRef},
		Atributos: map[string]string{"publicacion_ref": s.Snapshot.PublicacionRef, "corte": s.Snapshot.Corte,
			"huella_sha256": s.Snapshot.HuellaSHA256, "vista": s.Filtro.Vista,
			"q_sha256": huellaTextoRPTV2(s.Filtro.Q), "limite": strconv.Itoa(s.Filtro.Limite),
			"offset": strconv.Itoa(s.Filtro.Offset), "material_sha256": hex.EncodeToString(suma[:]),
			"categoria_clave": filtroRPTV2ONinguno(s.Filtro.CategoriaClave), "centro_codigo": filtroRPTV2ONinguno(s.Filtro.CentroCodigo)}}
	if _, err := recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialConsultaRPTPublicaV2{}, ErrRPTPublicaV2Invalida
	}
	return MaterialConsultaRPTPublicaV2{solicitud: s, canonico: canonico, recurso: recurso}, nil
}

func huellaTextoRPTV2(v string) string {
	suma := sha256.Sum256([]byte(v))
	return hex.EncodeToString(suma[:])
}

func filtroRPTV2ONinguno(v string) string {
	if v == "" {
		return "sin_filtro"
	}
	return v
}

func (m MaterialConsultaRPTPublicaV2) Solicitud() SolicitudConsultaRPTPublicaV2 { return m.solicitud }
func (m MaterialConsultaRPTPublicaV2) Canonico() []byte                         { return append([]byte(nil), m.canonico...) }
func (m MaterialConsultaRPTPublicaV2) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = make(map[string]string, len(m.recurso.Ambitos))
	for k, v := range m.recurso.Ambitos {
		r.Ambitos[k] = v
	}
	r.Atributos = make(map[string]string, len(m.recurso.Atributos))
	for k, v := range m.recurso.Atributos {
		r.Atributos[k] = v
	}
	return r
}
func (m MaterialConsultaRPTPublicaV2) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

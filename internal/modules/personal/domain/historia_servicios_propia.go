package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionHistoriaServiciosPropia      = "personal.registro_empleado.servicios.historia_propia.consultar"
	AudienciaHistoriaServiciosPropia   = "vec_personal.registro_empleado.servicios.historia_propia.v1"
	FinalidadHistoriaServiciosPropia   = "consultar_historia_servicios_propios"
	TipoRecursoHistoriaServiciosPropia = "historia_servicios_propia"
	// Límite técnico de la primera versión, sin truncado ni cursor.
	LimiteHistoriaServiciosPropia = LimiteFilasFichaPropia
)

var (
	ErrHistoriaServiciosPropiaInvalida     = errors.New("personal.historia_servicios_propia.invalida")
	ErrHistoriaServiciosPropiaDenegada     = errors.New("personal.historia_servicios_propia.denegada")
	ErrHistoriaServiciosPropiaNoDisponible = errors.New("personal.historia_servicios_propia.no_disponible")
	ErrHistoriaServiciosPropiaExcedeLimite = errors.New("personal.historia_servicios_propia.excede_limite")
	patronServicioHistoriaPropia           = regexp.MustCompile(`^srv_[A-Za-z0-9_-]{22,128}$`)
)

var patronReciboHistoriaServiciosPropia = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)

func ReferenciaReciboHistoriaServiciosPropiaValida(ref string) bool {
	return patronReciboHistoriaServiciosPropia.MatchString(ref)
}

// AD8 deriva la PK durable AD1 de la huella del consumo confirmado. No se
// fabrica otro recibo ni se recupera una respuesta desde su huella.
func ReciboHistoriaServiciosPropiaLigado(recibo, auditoria, consumo string) bool {
	return ReferenciaReciboHistoriaServiciosPropiaValida(recibo) && recibo == auditoria && huellaRegistroDominioB2Valida(consumo) && auditoria == "aud_v3_"+consumo[:32]
}

// El periodo de efectos es [Desde,Hasta). ConocidoEn fija qué revisiones se
// conocían; no es la fecha de prestación ni el instante de esta consulta.
type CorteHistoriaServiciosPropia struct {
	Desde      FechaCivil `json:"efectos_desde"`
	Hasta      FechaCivil `json:"efectos_hasta"`
	ConocidoEn time.Time  `json:"conocido_en"`
}

func (c CorteHistoriaServiciosPropia) Validar() error {
	if c.Desde.Validar() != nil || c.Hasta.Validar() != nil || !c.Desde.AntesDe(c.Hasta) || !instanteRegistroB2Valido(c.ConocidoEn) {
		return ErrHistoriaServiciosPropiaInvalida
	}
	return nil
}

// Actor procede de la frontera confiable. No hay selector de Persona,
// empleado u organismo aportado por el cliente.
type SolicitudHistoriaServiciosPropia struct {
	Actor core.ContextoActor
	Corte CorteHistoriaServiciosPropia
}

type MaterialHistoriaServiciosPropia struct {
	actor    core.ContextoActor
	corte    CorteHistoriaServiciosPropia
	empleado string
	canonico []byte
	recurso  core.RecursoAutorizable
}

func NuevoMaterialHistoriaServiciosPropia(s SolicitudHistoriaServiciosPropia) (MaterialHistoriaServiciosPropia, error) {
	var cero MaterialHistoriaServiciosPropia
	if s.Corte.Validar() != nil {
		return cero, ErrHistoriaServiciosPropiaInvalida
	}
	// Sólo se reutiliza la resolución del vínculo propio, nunca su permiso.
	base, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Actor: s.Actor, Corte: CorteEmpleadoB2{VigenteEn: s.Corte.Desde, ConocidoEn: s.Corte.ConocidoEn}})
	if err != nil {
		if errors.Is(err, ErrFichaPropiaSinEmpleado) || errors.Is(err, ErrFichaPropiaAmbigua) {
			return cero, ErrHistoriaServiciosPropiaDenegada
		}
		return cero, ErrHistoriaServiciosPropiaInvalida
	}
	actor := base.Actor()
	contexto, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return cero, ErrHistoriaServiciosPropiaInvalida
	}
	contenido := struct {
		Esquema       string          `json:"esquema"`
		EmpleadoRef   string          `json:"empleado_ref"`
		Desde         FechaCivil      `json:"efectos_desde"`
		Hasta         FechaCivil      `json:"efectos_hasta"`
		ConocidoEn    string          `json:"conocido_en"`
		ContextoActor json.RawMessage `json:"contexto_actor"`
	}{"vec.personal.historia-servicios-propia.consulta.v1", base.EmpleadoRef(), s.Corte.Desde, s.Corte.Hasta, s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), contexto}
	canon, err := json.Marshal(contenido)
	if err != nil {
		return cero, ErrHistoriaServiciosPropiaInvalida
	}
	h := sha256.Sum256(canon)
	recurso := core.RecursoAutorizable{Referencia: base.EmpleadoRef(), ModuloID: "personal", Tipo: TipoRecursoHistoriaServiciosPropia,
		Ambitos: map[string]string{"empleado_ref": base.EmpleadoRef()}, Atributos: map[string]string{"operacion": "historia_servicios_propia", "efectos_desde": s.Corte.Desde.Texto(), "efectos_hasta": s.Corte.Hasta.Texto(), "conocido_en": contenido.ConocidoEn, "material_sha256": hex.EncodeToString(h[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return cero, ErrHistoriaServiciosPropiaInvalida
	}
	return MaterialHistoriaServiciosPropia{actor, s.Corte, base.EmpleadoRef(), canon, recurso}, nil
}

func (m MaterialHistoriaServiciosPropia) Actor() core.ContextoActor {
	a, _ := m.actor.Clonar()
	return a
}
func (m MaterialHistoriaServiciosPropia) Corte() CorteHistoriaServiciosPropia { return m.corte }
func (m MaterialHistoriaServiciosPropia) EmpleadoRef() string                 { return m.empleado }
func (m MaterialHistoriaServiciosPropia) Canonico() []byte                    { return append([]byte(nil), m.canonico...) }
func (m MaterialHistoriaServiciosPropia) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialHistoriaServiciosPropia) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// La revisión y la fuente tienen versiones distintas. Traza conserva efectos,
// conocimiento y acto B2; PeriodoDesde/Hasta son el servicio prestado inclusivo.
// No hay nombre, DNI, información reservada ni documento supuesto desde ActoRef.
type RevisionServicioPropio struct {
	ServicioRef     string          `json:"servicio_ref"`
	RelacionRef     string          `json:"relacion_ref"`
	PeriodoDesde    FechaCivil      `json:"periodo_desde"`
	PeriodoHasta    FechaCivil      `json:"periodo_hasta"`
	DiasReconocidos int64           `json:"dias_reconocidos"`
	Estado          string          `json:"estado"`
	Clase           string          `json:"clase"`
	Traza           TrazaEmpleadoB2 `json:"traza"`
}

type HistoriaServiciosPropia struct {
	EmpleadoRef string                       `json:"empleado_ref"`
	Corte       CorteHistoriaServiciosPropia `json:"corte"`
	// La cobertura procede de la fuente. Una lista vacía no la determina.
	Cobertura  string                   `json:"cobertura"`
	Revisiones []RevisionServicioPropio `json:"revisiones"`
}

func (h HistoriaServiciosPropia) ValidarPara(m MaterialHistoriaServiciosPropia) error {
	if len(h.Revisiones) > LimiteHistoriaServiciosPropia {
		return ErrHistoriaServiciosPropiaExcedeLimite
	}
	if h.Corte.Validar() != nil || h.EmpleadoRef != m.EmpleadoRef() || h.EmpleadoRef == "" || h.Corte.Desde != m.corte.Desde || h.Corte.Hasta != m.corte.Hasta || !h.Corte.ConocidoEn.Equal(m.corte.ConocidoEn) || h.Revisiones == nil || (h.Cobertura != "completa" && h.Cobertura != "parcial" && h.Cobertura != "no_acreditada") {
		return ErrHistoriaServiciosPropiaInvalida
	}
	ids := make(map[struct {
		ref     string
		version int64
	}]struct{})
	for i, s := range h.Revisiones {
		if !patronServicioHistoriaPropia.MatchString(s.ServicioRef) || !ReferenciaRelacionValida(s.RelacionRef) || s.PeriodoDesde.Validar() != nil || s.PeriodoHasta.Validar() != nil || !s.PeriodoDesde.AntesDe(s.PeriodoHasta) || s.DiasReconocidos < 0 || !textoFichaPropiaValido(s.Clase) || (s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") || s.Traza.ValidarEn(CorteEmpleadoB2{VigenteEn: h.Corte.Desde, ConocidoEn: h.Corte.ConocidoEn}) != nil || !s.Traza.Desde.AntesDe(h.Corte.Hasta) || (s.Traza.Hasta != "" && !h.Corte.Desde.AntesDe(s.Traza.Hasta)) {
			return ErrHistoriaServiciosPropiaInvalida
		}
		id := struct {
			ref     string
			version int64
		}{s.ServicioRef, s.Traza.Version}
		if _, ok := ids[id]; ok {
			return ErrHistoriaServiciosPropiaInvalida
		}
		ids[id] = struct{}{}
		// Orden total: efectos descendentes, conocimiento descendente,
		// referencia ascendente y revisión descendente. No se suprimen versiones.
		if i > 0 && !revisionServicioPropioAntes(h.Revisiones[i-1], s) {
			return ErrHistoriaServiciosPropiaInvalida
		}
	}
	return nil
}

func revisionServicioPropioAntes(a, b RevisionServicioPropio) bool {
	if a.Traza.Desde != b.Traza.Desde {
		return b.Traza.Desde.AntesDe(a.Traza.Desde)
	}
	if !a.Traza.RegistradaEn.Equal(b.Traza.RegistradaEn) {
		return a.Traza.RegistradaEn.After(b.Traza.RegistradaEn)
	}
	if a.ServicioRef != b.ServicioRef {
		return a.ServicioRef < b.ServicioRef
	}
	return a.Traza.Version > b.Traza.Version
}

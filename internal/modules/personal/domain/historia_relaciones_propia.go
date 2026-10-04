package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionHistoriaRelacionesPropia      = "personal.registro_empleado.relaciones.historia_propia.consultar"
	AudienciaHistoriaRelacionesPropia   = "vec_personal.registro_empleado.relaciones.historia_propia.v1"
	FinalidadHistoriaRelacionesPropia   = "consultar_historia_relaciones_propias"
	TipoRecursoHistoriaRelacionesPropia = "historia_relaciones_propia"
	// Límite técnico de la primera versión, sin truncado ni cursor.
	LimiteHistoriaRelacionesPropia = LimiteFilasFichaPropia
)

var (
	ErrHistoriaRelacionesPropiaInvalida     = errors.New("personal.historia_relaciones_propia.invalida")
	ErrHistoriaRelacionesPropiaDenegada     = errors.New("personal.historia_relaciones_propia.denegada")
	ErrHistoriaRelacionesPropiaNoDisponible = errors.New("personal.historia_relaciones_propia.no_disponible")
	ErrHistoriaRelacionesPropiaExcedeLimite = errors.New("personal.historia_relaciones_propia.excede_limite")
)

func ReferenciaReciboHistoriaRelacionesPropiaValida(ref string) bool {
	return ReferenciaReciboHistoriaServiciosPropiaValida(ref)
}

// AD8 deriva la PK durable AD1 de la huella del consumo confirmado. No se
// fabrica otro recibo ni se recupera una respuesta desde su huella.
func ReciboHistoriaRelacionesPropiaLigado(recibo, auditoria, consumo string) bool {
	return ReciboHistoriaServiciosPropiaLigado(recibo, auditoria, consumo)
}

// El periodo de efectos es [Desde,Hasta). ConocidoEn fija qué revisiones se
// conocían; no es el instante de esta consulta.
type CorteHistoriaRelacionesPropia struct {
	Desde      FechaCivil `json:"efectos_desde"`
	Hasta      FechaCivil `json:"efectos_hasta"`
	ConocidoEn time.Time  `json:"conocido_en"`
}

func (c CorteHistoriaRelacionesPropia) Validar() error {
	if c.Desde.Validar() != nil || c.Hasta.Validar() != nil || !c.Desde.AntesDe(c.Hasta) || !instanteRegistroB2Valido(c.ConocidoEn) {
		return ErrHistoriaRelacionesPropiaInvalida
	}
	return nil
}

// Actor procede de la frontera confiable. No hay selector de Persona,
// empleado u organismo aportado por el cliente.
type SolicitudHistoriaRelacionesPropia struct {
	Actor core.ContextoActor
	Corte CorteHistoriaRelacionesPropia
}

type MaterialHistoriaRelacionesPropia struct {
	actor    core.ContextoActor
	corte    CorteHistoriaRelacionesPropia
	empleado string
	canonico []byte
	recurso  core.RecursoAutorizable
}

func NuevoMaterialHistoriaRelacionesPropia(s SolicitudHistoriaRelacionesPropia) (MaterialHistoriaRelacionesPropia, error) {
	var cero MaterialHistoriaRelacionesPropia
	if s.Corte.Validar() != nil {
		return cero, ErrHistoriaRelacionesPropiaInvalida
	}
	// Sólo se reutiliza la resolución del vínculo propio, nunca su permiso.
	base, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Actor: s.Actor, Corte: CorteEmpleadoB2{VigenteEn: s.Corte.Desde, ConocidoEn: s.Corte.ConocidoEn}})
	if err != nil {
		if errors.Is(err, ErrFichaPropiaSinEmpleado) || errors.Is(err, ErrFichaPropiaAmbigua) {
			return cero, ErrHistoriaRelacionesPropiaDenegada
		}
		return cero, ErrHistoriaRelacionesPropiaInvalida
	}
	actor := base.Actor()
	contexto, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return cero, ErrHistoriaRelacionesPropiaInvalida
	}
	contenido := struct {
		Esquema       string          `json:"esquema"`
		EmpleadoRef   string          `json:"empleado_ref"`
		Desde         FechaCivil      `json:"efectos_desde"`
		Hasta         FechaCivil      `json:"efectos_hasta"`
		ConocidoEn    string          `json:"conocido_en"`
		ContextoActor json.RawMessage `json:"contexto_actor"`
	}{"vec.personal.historia-relaciones-propia.consulta.v1", base.EmpleadoRef(), s.Corte.Desde, s.Corte.Hasta, s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), contexto}
	canon, err := json.Marshal(contenido)
	if err != nil {
		return cero, ErrHistoriaRelacionesPropiaInvalida
	}
	h := sha256.Sum256(canon)
	recurso := core.RecursoAutorizable{Referencia: base.EmpleadoRef(), ModuloID: "personal", Tipo: TipoRecursoHistoriaRelacionesPropia,
		Ambitos: map[string]string{"empleado_ref": base.EmpleadoRef()}, Atributos: map[string]string{"operacion": "historia_relaciones_propia", "efectos_desde": s.Corte.Desde.Texto(), "efectos_hasta": s.Corte.Hasta.Texto(), "conocido_en": contenido.ConocidoEn, "material_sha256": hex.EncodeToString(h[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return cero, ErrHistoriaRelacionesPropiaInvalida
	}
	return MaterialHistoriaRelacionesPropia{actor, s.Corte, base.EmpleadoRef(), canon, recurso}, nil
}

func (m MaterialHistoriaRelacionesPropia) Actor() core.ContextoActor {
	a, _ := m.actor.Clonar()
	return a
}
func (m MaterialHistoriaRelacionesPropia) Corte() CorteHistoriaRelacionesPropia { return m.corte }
func (m MaterialHistoriaRelacionesPropia) EmpleadoRef() string                  { return m.empleado }
func (m MaterialHistoriaRelacionesPropia) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialHistoriaRelacionesPropia) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialHistoriaRelacionesPropia) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

// La revisión y la fuente tienen versiones distintas. Traza conserva efectos,
// conocimiento y acto B2. Las denominaciones proceden de fuentes acreditadas;
// vacío significa que la fuente no las conoce, nunca una inferencia de RPT.
// No hay nombre, DNI, información reservada ni documento supuesto desde ActoRef.
type RevisionRelacionPropia struct {
	RelacionRef string          `json:"relacion_ref"`
	Estado      string          `json:"estado"`
	Regimen     string          `json:"regimen"`
	Modalidad   string          `json:"modalidad"`
	Unidad      string          `json:"unidad"`
	Puesto      string          `json:"puesto"`
	Situacion   string          `json:"situacion"`
	Traza       TrazaEmpleadoB2 `json:"traza"`
}

type HistoriaRelacionesPropia struct {
	EmpleadoRef string                        `json:"empleado_ref"`
	Corte       CorteHistoriaRelacionesPropia `json:"corte"`
	// La cobertura procede de la fuente. Una lista vacía no la determina.
	Cobertura  string                   `json:"cobertura"`
	Revisiones []RevisionRelacionPropia `json:"revisiones"`
}

func (h HistoriaRelacionesPropia) ValidarPara(m MaterialHistoriaRelacionesPropia) error {
	if len(h.Revisiones) > LimiteHistoriaRelacionesPropia {
		return ErrHistoriaRelacionesPropiaExcedeLimite
	}
	if h.Corte.Validar() != nil || h.EmpleadoRef != m.EmpleadoRef() || h.EmpleadoRef == "" || h.Corte.Desde != m.corte.Desde || h.Corte.Hasta != m.corte.Hasta || !h.Corte.ConocidoEn.Equal(m.corte.ConocidoEn) || h.Revisiones == nil || (h.Cobertura != "completa" && h.Cobertura != "parcial" && h.Cobertura != "no_acreditada") {
		return ErrHistoriaRelacionesPropiaInvalida
	}
	ids := make(map[struct {
		ref     string
		version int64
	}]struct{})
	for i, s := range h.Revisiones {
		if !ReferenciaRelacionValida(s.RelacionRef) || !estadoRelacionB2Valido(s.Estado) || !textoFichaPropiaValido(s.Regimen) || !textoFichaPropiaValido(s.Modalidad) || !textoFichaPropiaValido(s.Unidad) || !textoFichaPropiaValido(s.Puesto) || !textoFichaPropiaValido(s.Situacion) || s.Traza.ValidarEn(CorteEmpleadoB2{VigenteEn: h.Corte.Desde, ConocidoEn: h.Corte.ConocidoEn}) != nil || !s.Traza.Desde.AntesDe(h.Corte.Hasta) || (s.Traza.Hasta != "" && !h.Corte.Desde.AntesDe(s.Traza.Hasta)) {
			return ErrHistoriaRelacionesPropiaInvalida
		}
		id := struct {
			ref     string
			version int64
		}{s.RelacionRef, s.Traza.Version}
		if _, ok := ids[id]; ok {
			return ErrHistoriaRelacionesPropiaInvalida
		}
		ids[id] = struct{}{}
		// Orden total: efectos descendentes, conocimiento descendente,
		// referencia ascendente y revisión descendente. No se suprimen versiones.
		if i > 0 && !revisionRelacionPropiaAntes(h.Revisiones[i-1], s) {
			return ErrHistoriaRelacionesPropiaInvalida
		}
	}
	return nil
}

func revisionRelacionPropiaAntes(a, b RevisionRelacionPropia) bool {
	if a.Traza.Desde != b.Traza.Desde {
		return b.Traza.Desde.AntesDe(a.Traza.Desde)
	}
	if !a.Traza.RegistradaEn.Equal(b.Traza.RegistradaEn) {
		return a.Traza.RegistradaEn.After(b.Traza.RegistradaEn)
	}
	if a.RelacionRef != b.RelacionRef {
		return a.RelacionRef < b.RelacionRef
	}
	return a.Traza.Version > b.Traza.Version
}

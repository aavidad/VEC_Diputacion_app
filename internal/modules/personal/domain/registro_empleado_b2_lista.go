package domain

import core "vec-diputacion-granada/internal/vec/domain"

const (
	AccionEmpleadosB2    = "personal.registro_empleado.empleados.consultar"
	AudienciaEmpleadosB2 = "vec_personal.registro_empleado.empleados.v1"
	LimiteEmpleadosB2    = 100
	maximoRelacionesB2   = 20
	maximoDenominacionB2 = 300
)

// SolicitudEmpleadosB2 pide la lista RRHH de empleados del organismo que el
// servidor asocia al actor. Sirve para elegir una ficha sin teclear
// referencias; Personal no guarda datos civiles y la lista no los contiene.
type SolicitudEmpleadosB2 struct {
	OrganismoRef string
	Corte        CorteEmpleadoB2
	Limite       int
	Cursor       string
	Actor        core.ContextoActor
}

func NuevoMaterialEmpleadosB2(s SolicitudEmpleadosB2) (MaterialConsultaRegistroEmpleadoB2, error) {
	if !patronReferenciaB2.MatchString(s.OrganismoRef) || s.Corte.Validar() != nil || s.Limite < 1 || s.Limite > LimiteEmpleadosB2 || (s.Cursor != "" && !patronCursorB2.MatchString(s.Cursor)) || s.Actor.Validar() != nil {
		return MaterialConsultaRegistroEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialConsultaB2("empleados", "", s.OrganismoRef, s.Corte, s.Limite, s.Cursor, s.Actor)
}

// RelacionVigenteEmpleadoB2 identifica al empleado por su relación vigente en
// la fecha del corte: unidad, puesto ocupado, régimen y modalidad publicados.
type RelacionVigenteEmpleadoB2 struct {
	RelacionRef           string          `json:"relacion_ref"`
	Estado                string          `json:"estado"`
	UnidadRef             string          `json:"unidad_ref"`
	UnidadDenominacion    string          `json:"unidad_denominacion"`
	PuestoDenominacion    string          `json:"puesto_denominacion"`
	RegimenDenominacion   string          `json:"regimen_denominacion"`
	ModalidadDenominacion string          `json:"modalidad_denominacion"`
	Traza                 TrazaEmpleadoB2 `json:"traza"`
}

type EmpleadoOrganismoB2 struct {
	EmpleadoRef string                      `json:"empleado_ref"`
	Relaciones  []RelacionVigenteEmpleadoB2 `json:"relaciones"`
}

type PaginaEmpleadosB2 struct {
	OrganismoRef    string                `json:"organismo_ref"`
	Corte           CorteEmpleadoB2       `json:"corte"`
	Limite          int                   `json:"limite"`
	Cursor          string                `json:"cursor"`
	CursorSiguiente string                `json:"cursor_siguiente"`
	Empleados       []EmpleadoOrganismoB2 `json:"empleados"`
}

func (p PaginaEmpleadosB2) ValidarPara(m MaterialConsultaRegistroEmpleadoB2) error {
	if m.Operacion() != "empleados" || p.OrganismoRef != m.OrganismoRef() || !corteRegistroB2Igual(p.Corte, m.Corte()) || p.Limite != m.Limite() || p.Cursor != m.Cursor() || p.Empleados == nil || len(p.Empleados) > p.Limite || (p.CursorSiguiente != "" && (!patronCursorB2.MatchString(p.CursorSiguiente) || p.CursorSiguiente == p.Cursor)) {
		return ErrRegistroEmpleadoB2Invalido
	}
	empleados, relaciones := map[string]struct{}{}, map[string]struct{}{}
	anterior := ""
	for _, e := range p.Empleados {
		if !ReferenciaEmpleadoValida(e.EmpleadoRef) || e.EmpleadoRef <= anterior || repetidoB2(empleados, e.EmpleadoRef) || e.Relaciones == nil || len(e.Relaciones) > maximoRelacionesB2 {
			return ErrRegistroEmpleadoB2Invalido
		}
		anterior = e.EmpleadoRef
		for _, r := range e.Relaciones {
			if !ReferenciaRelacionValida(r.RelacionRef) || (r.Estado != "vigente" && r.Estado != "suspendida") || !patronReferenciaB2.MatchString(r.UnidadRef) ||
				!denominacionB2Valida(r.UnidadDenominacion) || !denominacionB2Valida(r.PuestoDenominacion) || !denominacionB2Valida(r.RegimenDenominacion) || !denominacionB2Valida(r.ModalidadDenominacion) ||
				r.Traza.ValidarEn(p.Corte) != nil || p.Corte.VigenteEn.AntesDe(r.Traza.Desde) || (r.Traza.Hasta != "" && !p.Corte.VigenteEn.AntesDe(r.Traza.Hasta)) || repetidoB2(relaciones, r.RelacionRef) {
				return ErrRegistroEmpleadoB2Invalido
			}
		}
	}
	return nil
}

// Denominación visible opcional: vacía si la fuente organizativa no la tiene.
func denominacionB2Valida(s string) bool {
	if len(s) > maximoDenominacionB2*4 {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

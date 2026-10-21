package domain

// SolicitudCierreGobiernoRolPorReferencia llega del canal ADMIN con una
// referencia y huella de propuesta. El proponente y el RolID objetivo no
// proceden del cuerpo: la autoridad SQL los recupera de la propuesta
// inmutable bajo la decisión V3 de aprobación en la misma transacción.
type SolicitudCierreGobiernoRolPorReferencia struct {
	OperacionRef            string
	PropuestaRef            string
	PropuestaHuellaSHA256   string
	Aprobador               ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Decision                DecisionPropuestaAdministracionPerfiles
	Motivo                  ReferenciaEntradaCatalogo
	CorrelacionRef          string
}

func (s SolicitudCierreGobiernoRolPorReferencia) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "cierre_admin:") ||
		!ReferenciaAdministracionPerfilesValida(s.PropuestaRef, "propuesta_admin:") ||
		!huellaSHA256AutorizacionV3NoNula(s.PropuestaHuellaSHA256) ||
		s.Aprobador.Validar() != nil || s.Evidencia.ValidarPara(s.Aprobador) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil || s.Decision != DecisionAprobarPropuestaPerfil ||
		s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

// CompletarCierreGobiernoRolConMaterial sólo se usa tras recibir el material
// original desde SQL. ValidarPara comprueba después que el cierre y el recibo
// corresponden a esa propuesta exacta. Esta reconstrucción no autoriza nada.
func (s SolicitudCierreGobiernoRolPorReferencia) CompletarCierreGobiernoRolConMaterial(
	m MaterialPropuestaGobiernoPerfil) (SolicitudCierreGobiernoPerfil, error) {
	var vacia SolicitudCierreGobiernoPerfil
	h, err := m.HuellaSHA256()
	if s.Validar() != nil || err != nil || h != s.PropuestaHuellaSHA256 ||
		m.OperacionRef != s.PropuestaRef || m.Plan.Operacion != OperacionCrearPerfilGobernado ||
		m.Plan.Base != nil || m.Plan.DefinicionNueva == nil ||
		m.Plan.DefinicionNueva.Version != 1 || !PlanGobiernoRolNuevoAdmitido(m.Plan) ||
		m.Plan.Motivo != s.Motivo ||
		m.ProponentePersonaRef == s.Aprobador.PersonaRef {
		return vacia, ErrPlanGobiernoPerfilInvalido
	}
	completa := SolicitudCierreGobiernoPerfil{OperacionRef: s.OperacionRef,
		PropuestaRef: s.PropuestaRef, PropuestaHuellaSHA256: s.PropuestaHuellaSHA256,
		ProponentePersonaRef: m.ProponentePersonaRef, VersionRolObjetivoRef: m.Plan.VersionRolObjetivoRef,
		Aprobador: s.Aprobador, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
		Decision: s.Decision, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
	if completa.Validar() != nil {
		return vacia, ErrPlanGobiernoPerfilInvalido
	}
	return completa, nil
}

// PlanGobiernoRolNuevoAdmitido conserva la concesión ordinaria única y solo
// admite planes compuestos para los dos contratos exactos de Dietas.
func PlanGobiernoRolNuevoAdmitido(p PlanGobiernoPerfil) bool {
	if p.DefinicionNueva == nil {
		return false
	}
	switch p.DefinicionNueva.RolID {
	case RolIDPersonaDietasComisiones, RolIDPersonaDietasRutas:
		return planRolDietasExacto(p)
	default:
		return len(p.DefinicionNueva.Concesiones) == 1 && len(p.Selecciones) == 1
	}
}

// IntencionGobiernoRolNuevoAdmitida acota la intención antes de leer el
// catálogo: una única concesión ordinaria o el número exacto de concesiones de
// uno de los dos contratos de Dietas. El contenido lo coteja después
// PlanGobiernoRolNuevoAdmitido sobre el plan preparado.
func IntencionGobiernoRolNuevoAdmitida(s SolicitudPlanGobiernoPerfil) bool {
	if s.Publicacion == nil {
		return false
	}
	esperadas := 1
	switch s.Publicacion.RolPropuesto.RolID {
	case RolIDPersonaDietasComisiones:
		esperadas = len(accionesRolDietasComisiones)
	case RolIDPersonaDietasRutas:
		esperadas = len(accionesRolDietasRutas)
	}
	return len(s.Publicacion.Selecciones) == esperadas &&
		len(s.Publicacion.RolPropuesto.Concesiones) == esperadas
}

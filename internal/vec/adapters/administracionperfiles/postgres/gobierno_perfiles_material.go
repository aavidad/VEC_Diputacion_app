package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionGobiernoRolProponer    = "administracion.perfiles.definicion.proponer"
	AccionGobiernoRolAprobar     = "administracion.perfiles.definicion.aprobar"
	AudienciaGobiernoRolProponer = "vec_autorizacion.gobierno_rol_nuevo.propuesta.v1"
	AudienciaGobiernoRolAprobar  = "vec_autorizacion.gobierno_rol_nuevo.cierre.v1"
	FinalidadGobiernoRol         = "gobierno_definiciones_perfiles"
	atributoMaterialGobiernoRol  = "material_sha256"
)

type propuestaGobiernoRolEnvelope struct {
	Esquema        string `json:"esquema"`
	MaterialCanon  string `json:"material_canon"`
	MaterialSHA256 string `json:"material_sha256"`
	PlanSHA256     string `json:"plan_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
}

type cierreGobiernoRolEnvelope struct {
	Esquema               string                                         `json:"esquema"`
	OperacionRef          string                                         `json:"operacion_ref"`
	PropuestaRef          string                                         `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ActorPersonaRef       string                                         `json:"actor_persona_ref"`
	ActorPerfilRef        string                                         `json:"actor_perfil_ref"`
	AsignacionRef         string                                         `json:"asignacion_ref"`
	Motivo                domain.ReferenciaEntradaCatalogo               `json:"motivo"`
	CorrelacionRef        string                                         `json:"correlacion_ref"`
}

func materialPropuestaGobiernoRol(o domain.OrdenPropuestaGobiernoPerfil) (Efecto, error) {
	if o.Validar() != nil || o.Material.Plan.Operacion != domain.OperacionCrearPerfilGobernado ||
		o.Material.Plan.Base != nil || o.Material.Plan.DefinicionNueva == nil ||
		o.Material.Plan.DefinicionNueva.Version != 1 || len(o.Material.Plan.DefinicionNueva.Concesiones) != 1 ||
		len(o.Material.Plan.Selecciones) != 1 {
		return Efecto{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	planSHA, err := o.Material.Plan.HuellaSHA256()
	if err != nil {
		return Efecto{}, err
	}
	materialSHA, err := o.Material.HuellaSHA256()
	if err != nil {
		return Efecto{}, err
	}
	canon, err := json.Marshal(o.Material)
	if err != nil {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	b, err := json.Marshal(propuestaGobiernoRolEnvelope{
		Esquema: "administracion_gobierno_rol_nuevo_propuesta_v1", MaterialCanon: string(canon),
		MaterialSHA256: materialSHA, PlanSHA256: planSHA, CorrelacionRef: o.Solicitud.CorrelacionRef,
	})
	if err != nil || len(b) > 65536 {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return Efecto{Accion: AccionGobiernoRolProponer, Audiencia: AudienciaGobiernoRolProponer,
		Referencia: o.Material.Plan.VersionRolObjetivoRef, Material: b, CorrelacionAccesoRef: o.Solicitud.CorrelacionRef}, nil
}

func materialCierreGobiernoRolPorReferencia(s domain.SolicitudCierreGobiernoRolPorReferencia) (Efecto, error) {
	if s.Validar() != nil || s.Decision != domain.DecisionAprobarPropuestaPerfil {
		return Efecto{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	b, err := json.Marshal(cierreGobiernoRolEnvelope{
		Esquema: "administracion_gobierno_rol_nuevo_cierre_v1", OperacionRef: s.OperacionRef,
		PropuestaRef: s.PropuestaRef, PropuestaHuellaSHA256: s.PropuestaHuellaSHA256,
		Decision: s.Decision, ActorPersonaRef: s.Aprobador.PersonaRef, ActorPerfilRef: s.Aprobador.PerfilActivoRef,
		AsignacionRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Motivo: s.Motivo,
		CorrelacionRef: s.CorrelacionRef,
	})
	if err != nil || len(b) > 65536 {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return Efecto{Accion: AccionGobiernoRolAprobar, Audiencia: AudienciaGobiernoRolAprobar,
		Referencia: s.PropuestaRef, Material: b, CorrelacionAccesoRef: s.CorrelacionRef}, nil
}

func materialCierreGobiernoRol(s domain.SolicitudCierreGobiernoPerfil) (Efecto, error) {
	if s.Validar() != nil {
		return Efecto{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	return materialCierreGobiernoRolPorReferencia(domain.SolicitudCierreGobiernoRolPorReferencia{
		OperacionRef: s.OperacionRef, PropuestaRef: s.PropuestaRef,
		PropuestaHuellaSHA256: s.PropuestaHuellaSHA256, Aprobador: s.Aprobador,
		Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
		Decision: s.Decision, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef,
	})
}

// RecursoGobiernoRolNuevo deriva el recurso de bytes cerrados y de la
// asignación actual. La huella de material se recalcula en AUT60; el caller
// jamás aporta ámbitos ni atributos positivos.
func RecursoGobiernoRolNuevo(e Efecto, asignacion domain.AsignacionPerfil) (domain.RecursoAutorizable, error) {
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if len(e.Material) == 0 || len(e.Material) > 65536 || asignacion.Validar() != nil || len(asignacion.Ambitos) != 2 {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	ambitos := make(map[string]string, 2)
	for _, a := range asignacion.Ambitos {
		if len(a.Valores) != 1 || (a.Clave != "organizacion_ref" && a.Clave != "unidad_ref") {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		ambitos[a.Clave] = a.Valores[0]
	}
	if ambitos["organizacion_ref"] == "" || ambitos["unidad_ref"] == "" {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	var tipo string
	switch {
	case e.Accion == AccionGobiernoRolProponer && e.Audiencia == AudienciaGobiernoRolProponer:
		var p propuestaGobiernoRolEnvelope
		if decodificarGobiernoRol(e.Material, &p) != nil || p.Esquema != "administracion_gobierno_rol_nuevo_propuesta_v1" ||
			p.CorrelacionRef != e.CorrelacionAccesoRef || len(p.MaterialCanon) == 0 || len(p.MaterialCanon) > 60000 {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		var m domain.MaterialPropuestaGobiernoPerfil
		if decodificarGobiernoRol([]byte(p.MaterialCanon), &m) != nil || m.Plan.Operacion != domain.OperacionCrearPerfilGobernado ||
			m.Plan.DefinicionNueva == nil || m.Plan.DefinicionNueva.Version != 1 || m.Plan.Base != nil ||
			len(m.Plan.Selecciones) != 1 || len(m.Plan.DefinicionNueva.Concesiones) != 1 ||
			m.Plan.VersionRolObjetivoRef != e.Referencia {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		canon, err := json.Marshal(m)
		mh, eh := m.HuellaSHA256()
		ph, ep := m.Plan.HuellaSHA256()
		if err != nil || eh != nil || ep != nil || !bytes.Equal(canon, []byte(p.MaterialCanon)) ||
			mh != p.MaterialSHA256 || ph != p.PlanSHA256 {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		tipo = "definicion_rol"
	case e.Accion == AccionGobiernoRolAprobar && e.Audiencia == AudienciaGobiernoRolAprobar:
		var c cierreGobiernoRolEnvelope
		if decodificarGobiernoRol(e.Material, &c) != nil || c.Esquema != "administracion_gobierno_rol_nuevo_cierre_v1" ||
			c.Decision != domain.DecisionAprobarPropuestaPerfil || c.PropuestaRef != e.Referencia ||
			c.CorrelacionRef != e.CorrelacionAccesoRef || c.PropuestaHuellaSHA256 == "" ||
			!domain.ReferenciaAdministracionPerfilesValida(c.OperacionRef, "cierre_admin:") ||
			!domain.ReferenciaAdministracionPerfilesValida(c.PropuestaRef, "propuesta_admin:") {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		tipo = "propuesta_definicion_rol"
	default:
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	h := sha256.Sum256(e.Material)
	recurso := domain.RecursoAutorizable{Referencia: e.Referencia, ModuloID: "administracion", Tipo: tipo,
		Ambitos: ambitos, Atributos: map[string]string{atributoMaterialGobiernoRol: hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil || !asignacion.Cubre(recurso) {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	return recurso, nil
}

func decodificarGobiernoRol(b []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return falloDecodificacionGobiernoRol{}
	}
	return nil
}

type falloDecodificacionGobiernoRol struct{}

func (falloDecodificacionGobiernoRol) Error() string { return "gobierno_rol_material_invalido" }

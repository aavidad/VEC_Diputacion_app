package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionVersionarRolBolsaProponer    = "administracion.perfiles.version_bolsa.proponer"
	AccionVersionarRolBolsaAprobar     = "administracion.perfiles.version_bolsa.aprobar"
	AudienciaVersionarRolBolsaProponer = "vec_autorizacion.versionar_rol_bolsa.propuesta.v1"
	AudienciaVersionarRolBolsaAprobar  = "vec_autorizacion.versionar_rol_bolsa.cierre.v1"
	FinalidadVersionarRolBolsa         = "gobierno_definiciones_perfiles"
	esquemaVersionarRolBolsaPropuesta  = "administracion_version_rol_bolsa_propuesta_v1"
	esquemaVersionarRolBolsaCierre     = "administracion_version_rol_bolsa_cierre_v1"
)

type propuestaVersionarRolBolsaEnvelope struct {
	Esquema        string `json:"esquema"`
	MaterialCanon  string `json:"material_canon"`
	MaterialSHA256 string `json:"material_sha256"`
	PlanSHA256     string `json:"plan_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
}

type cierreVersionarRolBolsaEnvelope struct {
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

func materialPropuestaVersionarRolBolsa(o domain.OrdenPropuestaVersionarRolBolsa) (Efecto, error) {
	if o.Validar() != nil {
		return Efecto{}, domain.ErrVersionarRolBolsaInvalido
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
	b, err := json.Marshal(propuestaVersionarRolBolsaEnvelope{
		Esquema: esquemaVersionarRolBolsaPropuesta, MaterialCanon: string(canon),
		MaterialSHA256: materialSHA, PlanSHA256: planSHA, CorrelacionRef: o.Solicitud.CorrelacionRef,
	})
	if err != nil || len(b) > 65536 {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return Efecto{Accion: AccionVersionarRolBolsaProponer, Audiencia: AudienciaVersionarRolBolsaProponer,
		Referencia: o.Material.Plan.VersionRolObjetivoRef, Material: b,
		CorrelacionAccesoRef: o.Solicitud.CorrelacionRef}, nil
}

func materialCierreVersionarRolBolsa(s domain.SolicitudCierreVersionarRolBolsa) (Efecto, error) {
	if s.Validar() != nil {
		return Efecto{}, domain.ErrVersionarRolBolsaInvalido
	}
	b, err := json.Marshal(cierreVersionarRolBolsaEnvelope{
		Esquema: esquemaVersionarRolBolsaCierre, OperacionRef: s.OperacionRef,
		PropuestaRef: s.PropuestaRef, PropuestaHuellaSHA256: s.PropuestaHuellaSHA256,
		Decision: s.Decision, ActorPersonaRef: s.Aprobador.PersonaRef,
		ActorPerfilRef: s.Aprobador.PerfilActivoRef,
		AsignacionRef:  s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
		Motivo:         s.Motivo, CorrelacionRef: s.CorrelacionRef,
	})
	if err != nil || len(b) > 65536 {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return Efecto{Accion: AccionVersionarRolBolsaAprobar, Audiencia: AudienciaVersionarRolBolsaAprobar,
		Referencia: s.PropuestaRef, Material: b, CorrelacionAccesoRef: s.CorrelacionRef}, nil
}

// RecursoVersionarRolBolsa sólo deriva referencia, ámbito y huella del material
// canónico y de la asignación ya acreditada. El cliente no aporta positivos.
func RecursoVersionarRolBolsa(e Efecto, asignacion domain.AsignacionPerfil) (domain.RecursoAutorizable, error) {
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if len(e.Material) == 0 || len(e.Material) > 65536 || asignacion.Validar() != nil || len(asignacion.Ambitos) != 2 {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	ambitos := make(map[string]string, 2)
	for _, a := range asignacion.Ambitos {
		if len(a.Valores) != 1 || (a.Clave != "organizacion_ref" && a.Clave != "unidad_ref") || ambitos[a.Clave] != "" {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		ambitos[a.Clave] = a.Valores[0]
	}
	if ambitos["organizacion_ref"] == "" || ambitos["unidad_ref"] == "" {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	var ref, tipo string
	switch {
	case e.Accion == AccionVersionarRolBolsaProponer && e.Audiencia == AudienciaVersionarRolBolsaProponer:
		var p propuestaVersionarRolBolsaEnvelope
		if decodificarGobiernoRol(e.Material, &p) != nil || p.Esquema != esquemaVersionarRolBolsaPropuesta ||
			p.CorrelacionRef != e.CorrelacionAccesoRef || len(p.MaterialCanon) == 0 || len(p.MaterialCanon) > 60000 {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		var m domain.MaterialPropuestaVersionarRolBolsa
		if decodificarGobiernoRol([]byte(p.MaterialCanon), &m) != nil {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		canon, err := json.Marshal(m)
		mh, eh := m.HuellaSHA256()
		ph, ep := m.Plan.HuellaSHA256()
		if err != nil || eh != nil || ep != nil || !bytes.Equal(canon, []byte(p.MaterialCanon)) ||
			mh != p.MaterialSHA256 || ph != p.PlanSHA256 {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		ref, tipo = m.Plan.VersionRolObjetivoRef, "definicion_rol"
	case e.Accion == AccionVersionarRolBolsaAprobar && e.Audiencia == AudienciaVersionarRolBolsaAprobar:
		var c cierreVersionarRolBolsaEnvelope
		if decodificarGobiernoRol(e.Material, &c) != nil || c.Esquema != esquemaVersionarRolBolsaCierre ||
			c.Decision != domain.DecisionAprobarPropuestaPerfil || c.CorrelacionRef != e.CorrelacionAccesoRef ||
			!domain.ReferenciaAdministracionPerfilesValida(c.OperacionRef, "cierre_admin:") ||
			!domain.ReferenciaAdministracionPerfilesValida(c.PropuestaRef, "propuesta_admin:") ||
			c.PropuestaHuellaSHA256 == "" {
			return domain.RecursoAutorizable{}, errNoDisponible
		}
		ref, tipo = c.PropuestaRef, "propuesta_definicion_rol"
	default:
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	if e.Referencia != ref {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	h := sha256.Sum256(e.Material)
	recurso := domain.RecursoAutorizable{Referencia: ref, ModuloID: "administracion", Tipo: tipo,
		Ambitos: ambitos, Atributos: map[string]string{atributoMaterialGobiernoRol: hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil || !asignacion.Cubre(recurso) {
		return domain.RecursoAutorizable{}, errNoDisponible
	}
	return recurso, nil
}

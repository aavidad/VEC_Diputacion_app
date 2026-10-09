package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialGobiernoInscripcion = errors.New("material_gobierno_inscripcion_invalido")

const (
	AccionGobiernoInscripcionProponer    = "administracion.perfiles.version_inscripcion.proponer"
	AccionGobiernoInscripcionAprobar     = "administracion.perfiles.version_inscripcion.aprobar"
	AudienciaGobiernoInscripcionProponer = "vec_autorizacion.version_inscripcion.propuesta.v1"
	AudienciaGobiernoInscripcionAprobar  = "vec_autorizacion.version_inscripcion.cierre.v1"
)

type EfectoGobiernoInscripcion struct {
	Accion         string
	Audiencia      string
	RecursoRef     string
	CorrelacionRef string
	Material       []byte
}

type propuestaGobiernoInscripcionEnvelope struct {
	Esquema        string `json:"esquema"`
	MaterialCanon  string `json:"material_canon"`
	MaterialSHA256 string `json:"material_sha256"`
	PlanSHA256     string `json:"plan_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
}

type cierreGobiernoInscripcionEnvelope struct {
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

func MaterialPropuestaGobiernoInscripcion(o domain.OrdenPropuestaVersionInscripcion) (EfectoGobiernoInscripcion, error) {
	if o.Validar() != nil {
		return EfectoGobiernoInscripcion{}, ErrMaterialGobiernoInscripcion
	}
	planSHA, err := o.Material.Plan.HuellaSHA256()
	if err != nil {
		return EfectoGobiernoInscripcion{}, err
	}
	materialSHA, err := o.Material.HuellaSHA256()
	if err != nil {
		return EfectoGobiernoInscripcion{}, err
	}
	canon, err := json.Marshal(o.Material)
	if err != nil || len(canon) == 0 || len(canon) > 196608 {
		return EfectoGobiernoInscripcion{}, ErrMaterialGobiernoInscripcion
	}
	sobre, err := json.Marshal(propuestaGobiernoInscripcionEnvelope{
		Esquema: "administracion_version_inscripcion_propuesta_v1", MaterialCanon: string(canon),
		MaterialSHA256: materialSHA, PlanSHA256: planSHA, CorrelacionRef: o.Solicitud.CorrelacionRef,
	})
	if err != nil || len(sobre) == 0 || len(sobre) > 262144 {
		return EfectoGobiernoInscripcion{}, ErrMaterialGobiernoInscripcion
	}
	return EfectoGobiernoInscripcion{Accion: AccionGobiernoInscripcionProponer,
		Audiencia: AudienciaGobiernoInscripcionProponer, RecursoRef: o.Material.Plan.VersionRolObjetivoRef,
		CorrelacionRef: o.Solicitud.CorrelacionRef, Material: sobre}, nil
}

func MaterialCierreGobiernoInscripcion(s domain.SolicitudCierreVersionInscripcion) (EfectoGobiernoInscripcion, error) {
	if s.Validar() != nil {
		return EfectoGobiernoInscripcion{}, ErrMaterialGobiernoInscripcion
	}
	sobre, err := json.Marshal(cierreGobiernoInscripcionEnvelope{
		Esquema: "administracion_version_inscripcion_cierre_v1", OperacionRef: s.OperacionRef,
		PropuestaRef: s.PropuestaRef, PropuestaHuellaSHA256: s.PropuestaHuellaSHA256,
		Decision: s.Decision, ActorPersonaRef: s.Aprobador.PersonaRef,
		ActorPerfilRef: s.Aprobador.PerfilActivoRef,
		AsignacionRef:  s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
		Motivo:         s.Motivo, CorrelacionRef: s.CorrelacionRef,
	})
	if err != nil || len(sobre) == 0 || len(sobre) > 65536 {
		return EfectoGobiernoInscripcion{}, ErrMaterialGobiernoInscripcion
	}
	return EfectoGobiernoInscripcion{Accion: AccionGobiernoInscripcionAprobar,
		Audiencia: AudienciaGobiernoInscripcionAprobar, RecursoRef: s.PropuestaRef,
		CorrelacionRef: s.CorrelacionRef, Material: sobre}, nil
}

// RecursoGobiernoInscripcion exige la asignación ADMIN seleccionada y liga
// el recurso V3 a los bytes exactos de la propuesta/cierre. SQL vuelve a
// comprobar la asignación vigente y su concesión en la transacción.
func RecursoGobiernoInscripcion(e EfectoGobiernoInscripcion, asignacion domain.AsignacionPerfil) (domain.RecursoAutorizable, error) {
	if len(e.Material) == 0 || len(e.Material) > 262144 || asignacion.Validar() != nil ||
		len(asignacion.Ambitos) != 2 || !domain.ReferenciaCorrelacionAutorizacionV2Valida(e.CorrelacionRef) {
		return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
	}
	var tipo string
	switch {
	case e.Accion == AccionGobiernoInscripcionProponer && e.Audiencia == AudienciaGobiernoInscripcionProponer:
		var p propuestaGobiernoInscripcionEnvelope
		if decodificarGobiernoInscripcion(e.Material, &p) != nil ||
			p.Esquema != "administracion_version_inscripcion_propuesta_v1" ||
			p.CorrelacionRef != e.CorrelacionRef || len(p.MaterialCanon) == 0 {
			return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
		}
		var m domain.MaterialPropuestaVersionInscripcion
		if decodificarGobiernoInscripcion([]byte(p.MaterialCanon), &m) != nil ||
			m.Plan.VersionRolObjetivoRef != e.RecursoRef ||
			m.ProponentePersonaRef != asignacion.PrincipalID ||
			m.PerfilActivoRef != asignacion.PerfilActivoRef ||
			m.AsignacionPerfilRef != asignacion.Referencia() {
			return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
		}
		canon, err := json.Marshal(m)
		planSHA, pe := m.Plan.HuellaSHA256()
		materialSHA, me := m.HuellaSHA256()
		if err != nil || pe != nil || me != nil || !bytes.Equal(canon, []byte(p.MaterialCanon)) ||
			planSHA != p.PlanSHA256 || materialSHA != p.MaterialSHA256 {
			return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
		}
		tipo = "definicion_rol"
	case e.Accion == AccionGobiernoInscripcionAprobar && e.Audiencia == AudienciaGobiernoInscripcionAprobar:
		var c cierreGobiernoInscripcionEnvelope
		if decodificarGobiernoInscripcion(e.Material, &c) != nil ||
			c.Esquema != "administracion_version_inscripcion_cierre_v1" ||
			c.Decision != domain.DecisionAprobarPropuestaPerfil ||
			c.PropuestaRef != e.RecursoRef || c.CorrelacionRef != e.CorrelacionRef ||
			c.ActorPersonaRef != asignacion.PrincipalID ||
			c.ActorPerfilRef != asignacion.PerfilActivoRef ||
			c.AsignacionRef != asignacion.Referencia() || c.Motivo.Validar() != nil ||
			!huellaGobiernoInscripcionValida(c.PropuestaHuellaSHA256) ||
			!domain.ReferenciaAdministracionPerfilesValida(c.OperacionRef, "cierre_admin:") ||
			!domain.ReferenciaAdministracionPerfilesValida(c.PropuestaRef, "propuesta_admin:") {
			return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
		}
		tipo = "propuesta_definicion_rol"
	default:
		return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
	}
	ambitos := make(map[string]string, 2)
	for _, a := range asignacion.Ambitos {
		if (a.Clave != "organizacion_ref" && a.Clave != "unidad_ref") || len(a.Valores) != 1 {
			return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
		}
		ambitos[a.Clave] = a.Valores[0]
	}
	if len(ambitos) != 2 || ambitos["organizacion_ref"] == "" || ambitos["unidad_ref"] == "" {
		return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
	}
	h := sha256.Sum256(e.Material)
	recurso := domain.RecursoAutorizable{Referencia: e.RecursoRef, ModuloID: "administracion", Tipo: tipo,
		Ambitos: ambitos, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil || !asignacion.Cubre(recurso) {
		return domain.RecursoAutorizable{}, ErrMaterialGobiernoInscripcion
	}
	return recurso, nil
}

func huellaGobiernoInscripcionValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(b) == s
}

func decodificarGobiernoInscripcion(b []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return ErrMaterialGobiernoInscripcion
	}
	return nil
}

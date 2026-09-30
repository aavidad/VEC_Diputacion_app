package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	core "vec-diputacion-granada/internal/vec/domain"
)

const AudienciaPlanIncorporacionCT = "vec_personal.plan_incorporacion_ct.v1"

// DatosPlanIncorporacionCT fija referencias de las autoridades participantes.
// Personal no transforma la acreditación de Bolsa en identidad aportada por RRHH.
type DatosPlanIncorporacionCT struct {
	IdempotenciaRef                string                    `json:"idempotencia_ref"`
	OrigenCTRef                    string                    `json:"origen_ct_ref"`
	OrigenCTReciboRef              string                    `json:"origen_ct_recibo_ref"`
	OrigenCTHuellaSHA256           string                    `json:"origen_ct_huella_sha256"`
	ExpedienteRef                  string                    `json:"expediente_ref"`
	ExpedienteVersion              int64                     `json:"expediente_version"`
	OrganismoRef                   string                    `json:"organismo_ref"`
	UnidadRef                      string                    `json:"unidad_ref"`
	PersonaRef                     string                    `json:"persona_ref"`
	PersonaVersion                 int64                     `json:"persona_version"`
	FuenteBolsaRef                 string                    `json:"fuente_bolsa_ref"`
	FuenteBolsaVersion             int64                     `json:"fuente_bolsa_version"`
	FuenteBolsaReciboRef           string                    `json:"fuente_bolsa_recibo_ref"`
	FuenteBolsaHuellaSHA256        string                    `json:"fuente_bolsa_huella_sha256"`
	Regimen                        EntradaCatalogoEmpleadoB2 `json:"regimen"`
	Modalidad                      EntradaCatalogoEmpleadoB2 `json:"modalidad"`
	Desde                          FechaCivil                `json:"desde"`
	Hasta                          FechaCivil                `json:"hasta"`
	PlazaRef                       string                    `json:"plaza_ref"`
	PuestoRef                      string                    `json:"puesto_ref"`
	ClaseOcupacion                 string                    `json:"clase_ocupacion"`
	VersionPlantillaRef            string                    `json:"version_plantilla_ref"`
	VersionRPTRef                  string                    `json:"version_rpt_ref"`
	RevisionPlaza                  int64                     `json:"revision_plaza"`
	RevisionPuesto                 int64                     `json:"revision_puesto"`
	FuenteOrganizacionRef          string                    `json:"fuente_organizacion_ref"`
	FuenteOrganizacionHuellaSHA256 string                    `json:"fuente_organizacion_huella_sha256"`
	CatalogoRPTID                  string                    `json:"catalogo_rpt_id"`
	CatalogoRPTModulo              string                    `json:"catalogo_rpt_modulo"`
	CatalogoRPTCategoria           string                    `json:"catalogo_rpt_categoria"`
	CatalogoRPTVersion             int64                     `json:"catalogo_rpt_version"`
	CatalogoRPTHuellaSHA256        string                    `json:"catalogo_rpt_huella_sha256"`
	VinculoCTReciboRef             string                    `json:"vinculo_ct_recibo_ref"`
	Procedencia                    ProcedenciaActoEmpleadoB2 `json:"procedencia"`
}

func (d DatosPlanIncorporacionCT) Validar() error {
	for _, r := range []string{d.OrigenCTRef, d.OrigenCTReciboRef, d.ExpedienteRef, d.OrganismoRef, d.UnidadRef, d.FuenteBolsaRef, d.FuenteBolsaReciboRef, d.VersionPlantillaRef, d.VersionRPTRef, d.FuenteOrganizacionRef, d.CatalogoRPTID, d.CatalogoRPTModulo, d.CatalogoRPTCategoria, d.VinculoCTReciboRef} {
		if !patronReferenciaB2.MatchString(r) {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	for _, h := range []string{d.OrigenCTHuellaSHA256, d.FuenteBolsaHuellaSHA256, d.FuenteOrganizacionHuellaSHA256, d.CatalogoRPTHuellaSHA256} {
		if !huellaRegistroDominioB2Valida(h) {
			return ErrRegistroEmpleadoB2Invalido
		}
	}
	if (d.ClaseOcupacion != "temporal" && d.ClaseOcupacion != "provisional" && d.ClaseOcupacion != "titular") || !patronUUIDRegistroB2.MatchString(d.IdempotenciaRef) || !ReferenciaPersonaValida(d.PersonaRef) || d.PersonaVersion < 1 || d.ExpedienteVersion < 1 || d.FuenteBolsaVersion < 1 || d.CatalogoRPTVersion < 1 || d.RevisionPlaza < 1 || d.RevisionPuesto < 1 || !referenciaOrganizacionB2Valida(d.PlazaRef) || !referenciaOrganizacionB2Valida(d.PuestoRef) || d.Regimen.Validar() != nil || d.Modalidad.Validar() != nil || !intervaloActoB2Valido(d.Desde, d.Hasta) || d.Procedencia.Validar() != nil {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}
func (d DatosPlanIncorporacionCT) HuellaSHA256() string {
	b, _ := json.Marshal(d)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type PlanIncorporacionCT struct {
	PlanRef              string                   `json:"plan_ref"`
	ReciboRef            string                   `json:"recibo_ref"`
	Version              int64                    `json:"version"`
	HuellaSHA256         string                   `json:"huella_sha256"`
	Datos                DatosPlanIncorporacionCT `json:"datos"`
	Modo                 string                   `json:"modo"`
	EmpleadoExistenteRef string                   `json:"empleado_existente_ref"`
	ClaveAltaRelacion    string                   `json:"clave_alta_relacion"`
	ClaveOcupacion       string                   `json:"clave_ocupacion"`
	UsoRPTRef            string                   `json:"uso_rpt_ref"`
	ReservaRPTRef        string                   `json:"reserva_rpt_ref"`
	ConfirmacionRPTRef   string                   `json:"confirmacion_rpt_ref"`
}

func (p PlanIncorporacionCT) CalcularHuellaSHA256() string {
	h := sha256.Sum256([]byte(strings.Join([]string{p.Datos.HuellaSHA256(), p.PlanRef, p.ReciboRef, p.Modo, p.EmpleadoExistenteRef, p.ClaveAltaRelacion, p.ClaveOcupacion, p.UsoRPTRef, p.ReservaRPTRef, p.ConfirmacionRPTRef}, "|")))
	return hex.EncodeToString(h[:])
}
func (p PlanIncorporacionCT) Validar() error {
	if p.Datos.Validar() != nil || !patronReferenciaB2.MatchString(p.PlanRef) || !patronReferenciaB2.MatchString(p.ReciboRef) || p.Version != 1 || !patronUUIDRegistroB2.MatchString(p.ClaveAltaRelacion) || !patronUUIDRegistroB2.MatchString(p.ClaveOcupacion) || p.ClaveAltaRelacion == p.ClaveOcupacion || !patronReferenciaB2.MatchString(p.UsoRPTRef) || !patronReferenciaB2.MatchString(p.ReservaRPTRef) || !patronReferenciaB2.MatchString(p.ConfirmacionRPTRef) || (p.Modo != "alta_empleado" && p.Modo != "nueva_relacion") || (p.Modo == "alta_empleado" && p.EmpleadoExistenteRef != "") || (p.Modo == "nueva_relacion" && !ReferenciaEmpleadoValida(p.EmpleadoExistenteRef)) || p.HuellaSHA256 != p.CalcularHuellaSHA256() {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type SolicitudPlanIncorporacionCT struct {
	DatosPlanIncorporacionCT
	Actor core.ContextoActor `json:"-"`
}
type ConsultaPlanIncorporacionCT struct {
	PlanRef      string
	OrganismoRef string
	Actor        core.ContextoActor
}
type MaterialPlanIncorporacionCT struct {
	operacion    string
	datos        DatosPlanIncorporacionCT
	planRef      string
	organismoRef string
	actor        core.ContextoActor
	canonico     []byte
	recurso      core.RecursoAutorizable
}

func NuevoMaterialPrepararPlanIncorporacionCT(s SolicitudPlanIncorporacionCT) (MaterialPlanIncorporacionCT, error) {
	if s.DatosPlanIncorporacionCT.Validar() != nil {
		return MaterialPlanIncorporacionCT{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialPlanCT("preparar", s.IdempotenciaRef, s.OrganismoRef, s.DatosPlanIncorporacionCT, s.Actor)
}
func NuevoMaterialConsultarPlanIncorporacionCT(s ConsultaPlanIncorporacionCT, operacion string) (MaterialPlanIncorporacionCT, error) {
	if (operacion != "consultar" && operacion != "ejecutar" && operacion != "confirmar") || !patronReferenciaB2.MatchString(s.PlanRef) || !patronReferenciaB2.MatchString(s.OrganismoRef) {
		return MaterialPlanIncorporacionCT{}, ErrRegistroEmpleadoB2Invalido
	}
	return nuevoMaterialPlanCT(operacion, s.PlanRef, s.OrganismoRef, DatosPlanIncorporacionCT{}, s.Actor)
}
func nuevoMaterialPlanCT(op, ref, org string, d DatosPlanIncorporacionCT, a core.ContextoActor) (MaterialPlanIncorporacionCT, error) {
	actor, err := a.Clonar()
	if err != nil {
		return MaterialPlanIncorporacionCT{}, ErrRegistroEmpleadoB2Invalido
	}
	var datos *DatosPlanIncorporacionCT
	var negocioSHA string
	if op == "preparar" {
		datos = &d
		negocioSHA = d.HuellaSHA256()
	}
	b, err := json.Marshal(struct {
		Esquema      string                    `json:"esquema"`
		Operacion    string                    `json:"operacion"`
		PlanRef      string                    `json:"plan_ref"`
		OrganismoRef string                    `json:"organismo_ref"`
		Datos        *DatosPlanIncorporacionCT `json:"datos"`
		NegocioSHA   string                    `json:"negocio_sha256"`
		Actor        identidadActoB2           `json:"actor"`
	}{"vec.personal.plan-incorporacion-ct.v1", op, ref, org, datos, negocioSHA, identidadActoRegistroB2(actor)})
	if err != nil {
		return MaterialPlanIncorporacionCT{}, ErrRegistroEmpleadoB2Invalido
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{Referencia: ref, ModuloID: "personal", Tipo: "plan_incorporacion_ct", Ambitos: map[string]string{"objetivo_ref": ref, "organismo_ref": org}, Atributos: map[string]string{"operacion": op, "material_sha256": hex.EncodeToString(h[:])}}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialPlanIncorporacionCT{}, ErrRegistroEmpleadoB2Invalido
	}
	return MaterialPlanIncorporacionCT{op, d, ref, org, actor, b, r}, nil
}
func (m MaterialPlanIncorporacionCT) Operacion() string { return m.operacion }
func (m MaterialPlanIncorporacionCT) Accion() string {
	return "personal.plan_incorporacion_ct." + m.operacion
}
func (m MaterialPlanIncorporacionCT) PlanRef() string                 { return m.planRef }
func (m MaterialPlanIncorporacionCT) OrganismoRef() string            { return m.organismoRef }
func (m MaterialPlanIncorporacionCT) Datos() DatosPlanIncorporacionCT { return m.datos }
func (m MaterialPlanIncorporacionCT) Canonico() []byte                { return append([]byte(nil), m.canonico...) }
func (m MaterialPlanIncorporacionCT) Actor() core.ContextoActor       { a, _ := m.actor.Clonar(); return a }
func (m MaterialPlanIncorporacionCT) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialPlanIncorporacionCT) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

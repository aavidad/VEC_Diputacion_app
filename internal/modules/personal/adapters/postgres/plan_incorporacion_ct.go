package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const planIncorporacionCTSQL = `SELECT vec_personal.plan_incorporacion_ct_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

var _ ports.RepositorioPlanIncorporacionCT = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) PrepararPlan(ctx context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	return r.planCT(ctx, o, "preparar")
}
func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConsultarPlan(ctx context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	if o.Material.Operacion() != "consultar" && o.Material.Operacion() != "ejecutar" {
		return ports.EstadoPlanIncorporacionCT{}, domain.ErrRegistroEmpleadoB2Invalido
	}
	return r.planCT(ctx, o, o.Material.Operacion())
}
func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConfirmarPlan(ctx context.Context, o ports.OrdenPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	return r.planCT(ctx, o, "confirmar")
}
func (r *RepositorioRegistroEmpleadoB2PostgreSQL) planCT(ctx context.Context, o ports.OrdenPlanIncorporacionCT, op string) (ports.EstadoPlanIncorporacionCT, error) {
	var vacio ports.EstadoPlanIncorporacionCT
	m := o.Material
	a := o.Autorizacion
	x := a.ResumenCapacidad()
	h, err := m.HuellaSHA256()
	actor := m.Actor()
	if r == nil || m.Operacion() != op || err != nil || a.ValidarEstructura() != nil || a.PersonaVersion() != actor.Instantanea.PersonaVersion || a.PerfilVersion() != actor.Instantanea.PerfilVersion || x.Operacion() != m.Accion() || x.AudienciaConsumo() != domain.AudienciaPlanIncorporacionCT || x.EfectoRef() != m.Recurso().Referencia || x.EfectoHuellaSHA256() != h {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	estado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, planIncorporacionCTSQL, m.Canonico(), a, 64<<10, func(bruto []byte) (ports.EstadoPlanIncorporacionCT, error) {
		var s ports.EstadoPlanIncorporacionCT
		var forma map[string]json.RawMessage
		if verificarJSONOrganizacionHistorica(bruto) != nil || json.Unmarshal(bruto, &forma) != nil || !clavesRegistroB2(forma, []string{"plan", "estado", "recibo_alta_relacion", "recibo_ocupacion", "ejecucion_recibo_ref", "ejecucion_huella_sha256", "evidencia"}, nil) || decodificarJSONRegistroB2(bruto, &s) != nil || s.Plan.Validar() != nil || s.Plan.Datos.OrganismoRef != m.OrganismoRef() || !evidenciaRegistroB2Valida(s.Evidencia, a) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		if (op == "preparar" && s.Plan.Datos.HuellaSHA256() != m.Datos().HuellaSHA256()) || (op != "preparar" && s.Plan.PlanRef != m.PlanRef()) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		if !formaEstadoPlanCTValida(s) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return s, nil
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return estado, nil
}
func formaEstadoPlanCTValida(s ports.EstadoPlanIncorporacionCT) bool {
	r, o := s.ReciboAltaRelacion, s.ReciboOcupacion
	if r != nil {
		if !reciboPlanCTValido(*r) || r.RegistradoEn.After(s.Evidencia.ConsultadaEn) || r.Version != 1 {
			return false
		}
		if s.Plan.Modo == "alta_empleado" {
			if r.Tipo != "alta" || r.EfectoRef != s.Plan.Datos.PersonaRef || !proyeccionEmpleadoB2Patron.MatchString(r.ProyeccionRef) || r.HechoRef != "" {
				return false
			}
		} else if r.Tipo != "relacion" || r.EmpleadoRef != s.Plan.EmpleadoExistenteRef || r.EfectoRef != r.EmpleadoRef || r.ProyeccionRef != "" || r.HechoRef != r.RelacionRef {
			return false
		}
	}
	if o != nil && (r == nil || !reciboPlanCTValido(*o) || o.RegistradoEn.After(s.Evidencia.ConsultadaEn) || o.Version != 1 || o.Tipo != "ocupacion" || o.EmpleadoRef != r.EmpleadoRef || o.RelacionRef != r.RelacionRef || o.EfectoRef != r.EmpleadoRef || !hechoEmpleadoB2Patron.MatchString(o.HechoRef) || o.ProyeccionRef != "") {
		return false
	}
	switch s.Estado {
	case "preparado":
		if r != nil || o != nil {
			return false
		}
	case "relacion_registrada":
		if r == nil || o != nil {
			return false
		}
	case "ocupacion_registrada":
		if r == nil || o == nil {
			return false
		}
	case "ejecutado":
		if r == nil || o == nil || !referenciaEvidenciaB2.MatchString(s.EjecucionReciboRef) {
			return false
		}
		h := sha256.Sum256([]byte(s.Plan.HuellaSHA256 + "|" + r.ReciboRef + "|" + o.ReciboRef))
		return s.EjecucionHuellaSHA256 == hex.EncodeToString(h[:])
	default:
		return false
	}
	return s.EjecucionReciboRef == "" && s.EjecucionHuellaSHA256 == ""
}
func reciboPlanCTValido(r ports.ReciboActoRegistroEmpleadoB2) bool {
	_, off := r.RegistradoEn.Zone()
	return reciboActoEmpleadoB2Patron.MatchString(r.ReciboRef) && domain.ReferenciaEmpleadoValida(r.EmpleadoRef) && domain.ReferenciaRelacionValida(r.RelacionRef) && !r.EficaciaAdministrativa && !r.FirmaOficial && !r.RegistradoEn.IsZero() && off == 0 && r.RegistradoEn.Nanosecond()%1000 == 0 && referenciaEvidenciaB2.MatchString(r.DecisionRef) && referenciaEvidenciaB2.MatchString(r.AuditoriaRef) && huellaEvidenciaB2.MatchString(r.ConsumoHuellaSHA256)
}

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ResolverSeleccion(ctx context.Context, o ports.OrdenPlanIncorporacionCT) (ports.ResultadoSeleccionPlanIncorporacionCT, error) {
	var vacio ports.ResultadoSeleccionPlanIncorporacionCT
	m, a := o.Material, o.Autorizacion
	x := a.ResumenCapacidad()
	actor := m.Actor()
	h, e := m.HuellaSHA256()
	if r == nil || m.Operacion() != "seleccionar" || e != nil || a.ValidarEstructura() != nil || a.PersonaVersion() != actor.Instantanea.PersonaVersion || a.PerfilVersion() != actor.Instantanea.PerfilVersion || x.Operacion() != m.Accion() || x.AudienciaConsumo() != domain.AudienciaPlanIncorporacionCT || x.EfectoRef() != m.Recurso().Referencia || x.EfectoHuellaSHA256() != h {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, e := ejecutarRegistroEmpleadoB2(ctx, r.pool, planIncorporacionCTSQL, m.Canonico(), a, 16<<10, func(b []byte) (ports.ResultadoSeleccionPlanIncorporacionCT, error) {
		var s ports.ResultadoSeleccionPlanIncorporacionCT
		var forma map[string]json.RawMessage
		if verificarJSONOrganizacionHistorica(b) != nil || json.Unmarshal(b, &forma) != nil || !clavesRegistroB2(forma, []string{"seleccion", "evidencia"}, nil) || decodificarJSONRegistroB2(b, &s) != nil || s.Seleccion.ValidarPara(m) != nil || !evidenciaRegistroB2Valida(s.Evidencia, a) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return s, nil
	})
	if e != nil {
		return vacio, errorDominioRegistroEmpleadoB2(e)
	}
	return resultado, nil
}

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConsultarClasesOcupacion(ctx context.Context, o ports.OrdenPlanIncorporacionCT) (ports.ResultadoClasesOcupacionCT, error) {
	var vacio ports.ResultadoClasesOcupacionCT
	m, a := o.Material, o.Autorizacion
	x := a.ResumenCapacidad()
	actor := m.Actor()
	h, e := m.HuellaSHA256()
	if r == nil || m.Operacion() != "clases_ocupacion" || e != nil || a.ValidarEstructura() != nil || a.PersonaVersion() != actor.Instantanea.PersonaVersion || a.PerfilVersion() != actor.Instantanea.PerfilVersion || x.Operacion() != m.Accion() || x.AudienciaConsumo() != domain.AudienciaPlanIncorporacionCT || x.EfectoRef() != m.Recurso().Referencia || x.EfectoHuellaSHA256() != h {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, e := ejecutarRegistroEmpleadoB2(ctx, r.pool, planIncorporacionCTSQL, m.Canonico(), a, 16<<10, func(b []byte) (ports.ResultadoClasesOcupacionCT, error) {
		var s ports.ResultadoClasesOcupacionCT
		var forma map[string]json.RawMessage
		if verificarJSONOrganizacionHistorica(b) != nil || json.Unmarshal(b, &forma) != nil || !clavesRegistroB2(forma, []string{"catalogo", "evidencia"}, nil) || decodificarJSONRegistroB2(b, &s) != nil || s.Catalogo.Validar() != nil || !evidenciaRegistroB2Valida(s.Evidencia, a) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return s, nil
	})
	if e != nil {
		return vacio, errorDominioRegistroEmpleadoB2(e)
	}
	return resultado, nil
}

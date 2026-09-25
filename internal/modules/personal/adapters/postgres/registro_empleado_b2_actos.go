package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maxRespuestaActoEmpleadoB2 = 32 << 10

var _ ports.RepositorioActosRegistroEmpleadoB2 = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) RegistrarEmpleadoRRHH(ctx context.Context, o ports.OrdenAltaEmpleadoB2) (ports.ResultadoAltaEmpleadoB2, error) {
	var vacio ports.ResultadoAltaEmpleadoB2
	if r == nil || !materialActoEmpleadoB2Valido(o.Material, o.Autorizacion, "alta", domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, registrarEmpleadoB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaActoEmpleadoB2, func(bruto []byte) (ports.ResultadoAltaEmpleadoB2, error) {
		var z ports.ResultadoAltaEmpleadoB2
		if formaReciboActoB2(bruto, "alta") != nil || decodificarJSONRegistroB2(bruto, &z) != nil || !reciboActoEmpleadoB2Valido(z.Recibo, z.AccesoActual, o.Material, o.Autorizacion, true) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) RegistrarHechoEmpleadoRRHH(ctx context.Context, o ports.OrdenHechoEmpleadoB2) (ports.ResultadoHechoEmpleadoB2, error) {
	var vacio ports.ResultadoHechoEmpleadoB2
	if r == nil || !materialActoEmpleadoB2Valido(o.Material, o.Autorizacion, "hecho", domain.AccionHechoEmpleadoB2, domain.AudienciaHechoEmpleadoB2) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, registrarHechoEmpleadoB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaActoEmpleadoB2, func(bruto []byte) (ports.ResultadoHechoEmpleadoB2, error) {
		var z ports.ResultadoHechoEmpleadoB2
		if formaReciboActoB2(bruto, "hecho") != nil || decodificarJSONRegistroB2(bruto, &z) != nil || !reciboActoEmpleadoB2Valido(z.Recibo, z.AccesoActual, o.Material, o.Autorizacion, false) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func materialActoEmpleadoB2Valido(m domain.MaterialActoRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, tipo, accion, audiencia string) bool {
	if m.Tipo() != tipo || a.ValidarEstructura() != nil || len(m.Canonico()) == 0 {
		return false
	}
	actor := m.Actor()
	recurso := m.Recurso()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && recurso.Referencia == m.Referencia() && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == accion && x.AudienciaConsumo() == audiencia && x.EfectoRef() == recurso.Referencia && x.EfectoHuellaSHA256() == h
}

func formaReciboActoB2(bruto []byte, tipo string) error {
	if len(bruto) == 0 || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) || verificarJSONOrganizacionHistorica(bruto) != nil {
		return errRegistroEmpleadoB2NoDisponible
	}
	var raiz map[string]json.RawMessage
	if json.Unmarshal(bruto, &raiz) != nil || !clavesRegistroB2(raiz, []string{"recibo", "acceso_actual"}, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	var recibo, acceso map[string]json.RawMessage
	if json.Unmarshal(raiz["recibo"], &recibo) != nil || json.Unmarshal(raiz["acceso_actual"], &acceso) != nil ||
		!clavesRegistroB2(acceso, []string{"decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en", "estado_replay"}, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	obligatorias := []string{"recibo_ref", "empleado_ref", "relacion_ref", "tipo", "version", "registrado_en", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "eficacia_administrativa", "firma_oficial"}
	if tipo == "alta" {
		obligatorias = append(obligatorias, "proyeccion_ref")
	} else {
		obligatorias = append(obligatorias, "hecho_ref")
	}
	if !clavesRegistroB2(recibo, obligatorias, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	return nil
}

var reciboActoEmpleadoB2Patron = regexp.MustCompile(`^perrec_[0-9a-f]{32}$`)
var proyeccionEmpleadoB2Patron = regexp.MustCompile(`^pep_[A-Za-z0-9_-]{22,128}$`)
var hechoEmpleadoB2Patron = regexp.MustCompile(`^(rel|ocu|srv|sit)_[A-Za-z0-9_-]{22,128}$`)

func reciboActoEmpleadoB2Valido(r ports.ReciboActoRegistroEmpleadoB2, acceso ports.AccesoActualRegistroEmpleadoB2, m domain.MaterialActoRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, alta bool) bool {
	x := a.ResumenCapacidad()
	_, offset := r.RegistradoEn.Zone()
	if !reciboActoEmpleadoB2Patron.MatchString(r.ReciboRef) || !domain.ReferenciaEmpleadoValida(r.EmpleadoRef) || !domain.ReferenciaRelacionValida(r.RelacionRef) ||
		r.Version < 1 || r.RegistradoEn.IsZero() || offset != 0 || r.RegistradoEn.Nanosecond()%1000 != 0 ||
		r.EficaciaAdministrativa || r.FirmaOficial || !referenciaEvidenciaB2.MatchString(r.DecisionRef) || r.EfectoRef != x.EfectoRef() || !huellaEvidenciaB2.MatchString(r.ConsumoHuellaSHA256) || !referenciaEvidenciaB2.MatchString(r.AuditoriaRef) {
		return false
	}
	_, accesoOffset := acceso.ConsultadaEn.Zone()
	if (acceso.EstadoReplay != "registrado" && acceso.EstadoReplay != "replay") || acceso.DecisionRef != x.DecisionRef() || acceso.EfectoRef != x.EfectoRef() ||
		!huellaEvidenciaB2.MatchString(acceso.ConsumoHuellaSHA256) || !referenciaEvidenciaB2.MatchString(acceso.AuditoriaRef) ||
		acceso.ConsultadaEn.IsZero() || accesoOffset != 0 || acceso.ConsultadaEn.Nanosecond()%1000 != 0 ||
		acceso.ConsultadaEn.Before(x.EmitidaEn()) || !acceso.ConsultadaEn.Before(x.ExpiraEn()) || r.RegistradoEn.After(acceso.ConsultadaEn) {
		return false
	}
	if acceso.EstadoReplay == "registrado" && (r.DecisionRef != acceso.DecisionRef || r.ConsumoHuellaSHA256 != acceso.ConsumoHuellaSHA256 || r.AuditoriaRef != acceso.AuditoriaRef || !r.RegistradoEn.Equal(acceso.ConsultadaEn)) {
		return false
	}
	if alta {
		return r.Tipo == "alta" && r.HechoRef == "" && proyeccionEmpleadoB2Patron.MatchString(r.ProyeccionRef)
	}
	if r.ProyeccionRef != "" || !hechoEmpleadoB2Patron.MatchString(r.HechoRef) || r.EmpleadoRef != m.Referencia() {
		return false
	}
	var material struct {
		Tipo        string `json:"tipo"`
		RelacionRef string `json:"relacion_ref"`
	}
	if json.Unmarshal(m.Canonico(), &material) != nil || material.Tipo != r.Tipo {
		return false
	}
	if material.Tipo != "relacion" && r.RelacionRef != material.RelacionRef {
		return false
	}
	return true
}

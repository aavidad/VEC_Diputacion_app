package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaCatalogoEmpleadoB2SQL  = `SELECT vec_personal.consultar_catalogo_empleado_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	cambioCatalogoEmpleadoB2SQL    = `SELECT vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	maxRespuestaCatalogoEmpleadoB2 = 256 << 10
)

var _ ports.RepositorioCatalogosRegistroEmpleadoB2 = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConsultarRRHH(ctx context.Context, o ports.OrdenCatalogoEmpleadoB2) (ports.ResultadoConsultaCatalogoEmpleadoB2, error) {
	var vacio ports.ResultadoConsultaCatalogoEmpleadoB2
	if r == nil || o.Material.Operacion() != "consultar" || !autorizacionCatalogoEmpleadoB2Valida(o.Material, o.Autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaCatalogoEmpleadoB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaCatalogoEmpleadoB2, func(bruto []byte) (ports.ResultadoConsultaCatalogoEmpleadoB2, error) {
		var z ports.ResultadoConsultaCatalogoEmpleadoB2
		if !formaConsultaCatalogoEmpleadoB2(bruto) || decodificarJSONRegistroB2(bruto, &z) != nil {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		for _, e := range z.Entradas {
			if e.Validar() != nil || e.OrganismoRef != o.Material.OrganismoRef() || e.Tipo != o.Material.Tipo() ||
				(o.Material.Estado() != "" && e.Estado != o.Material.Estado()) {
				return vacio, errRegistroEmpleadoB2NoDisponible
			}
		}
		if z.OrganismoRef != o.Material.OrganismoRef() || !evidenciaCatalogoEmpleadoB2Valida(z.Evidencia, o) || len(z.Entradas) > o.Material.Limite() ||
			!cursorCatalogoEmpleadoB2Valido(z, o.Material) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) CambiarRRHH(ctx context.Context, o ports.OrdenCatalogoEmpleadoB2) (ports.ResultadoCambioCatalogoEmpleadoB2, error) {
	var vacio ports.ResultadoCambioCatalogoEmpleadoB2
	if r == nil || (o.Material.Operacion() != "publicar" && o.Material.Operacion() != "retirar") ||
		!autorizacionCatalogoEmpleadoB2Valida(o.Material, o.Autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, cambioCatalogoEmpleadoB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaCatalogoEmpleadoB2, func(bruto []byte) (ports.ResultadoCambioCatalogoEmpleadoB2, error) {
		var z ports.ResultadoCambioCatalogoEmpleadoB2
		if !formaCambioCatalogoEmpleadoB2(bruto) || decodificarJSONRegistroB2(bruto, &z) != nil ||
			z.Entrada.Validar() != nil || z.Entrada.OrganismoRef != o.Material.OrganismoRef() ||
			z.Entrada.Tipo != o.Material.Tipo() || z.Entrada.Ref != o.Material.Ref() ||
			z.Entrada.Version != o.Material.Version() || z.Entrada.Revision != o.Material.Revision() ||
			z.Entrada.Estado != map[string]string{"publicar": "publicada", "retirar": "retirada"}[o.Material.Operacion()] ||
			!entradaCatalogoEmpleadoB2IgualMaterial(z.Entrada, o.Material) || !reciboCatalogoEmpleadoB2Valido(z, o) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func autorizacionCatalogoEmpleadoB2Valida(m domain.MaterialCatalogoEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil || len(m.Canonico()) == 0 {
		return false
	}
	actor := m.Actor()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	accion := "personal.registro_empleado.catalogo." + m.Operacion()
	audiencia := "vec_personal.registro_empleado.catalogo." + m.Operacion() + ".v1"
	return err == nil && a.PersonaVersion() == actor.Instantanea.PersonaVersion &&
		a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == accion &&
		x.AudienciaConsumo() == audiencia && x.EfectoRef() == m.Recurso().Referencia &&
		x.EfectoHuellaSHA256() == h
}

func formaConsultaCatalogoEmpleadoB2(bruto []byte) bool {
	var raiz map[string]json.RawMessage
	if !jsonCatalogoEmpleadoB2Valido(bruto) || json.Unmarshal(bruto, &raiz) != nil ||
		!clavesRegistroB2(raiz, []string{"organismo_ref", "entradas", "cursor_siguiente", "evidencia"}, nil) ||
		len(raiz["entradas"]) == 0 || raiz["entradas"][0] != '[' {
		return false
	}
	var evidencia map[string]json.RawMessage
	return json.Unmarshal(raiz["evidencia"], &evidencia) == nil &&
		clavesRegistroB2(evidencia, []string{"decision_ref", "auditoria_ref", "consumo_huella_sha256", "efecto_ref", "consultada_en"}, nil)
}

func formaCambioCatalogoEmpleadoB2(bruto []byte) bool {
	var raiz map[string]json.RawMessage
	if !jsonCatalogoEmpleadoB2Valido(bruto) || json.Unmarshal(bruto, &raiz) != nil ||
		!clavesRegistroB2(raiz, []string{"entrada", "recibo", "acceso_actual"}, nil) {
		return false
	}
	var entrada, recibo, acceso map[string]json.RawMessage
	return json.Unmarshal(raiz["entrada"], &entrada) == nil && json.Unmarshal(raiz["recibo"], &recibo) == nil &&
		json.Unmarshal(raiz["acceso_actual"], &acceso) == nil &&
		clavesRegistroB2(entrada, []string{"organismo_ref", "tipo", "ref", "version", "revision", "estado", "denominacion", "huella_sha256", "vigente_desde", "vigente_hasta"}, nil) &&
		clavesRegistroB2(recibo, []string{"decision_ref", "auditoria_ref", "consumo_huella_sha256", "registrado_en"}, nil) &&
		clavesRegistroB2(acceso, []string{"decision_ref", "auditoria_ref", "consumo_huella_sha256", "registrado_en", "estado_replay"}, nil)
}

func jsonCatalogoEmpleadoB2Valido(bruto []byte) bool {
	return len(bruto) != 0 && !bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) &&
		verificarJSONOrganizacionHistorica(bruto) == nil
}

func evidenciaCatalogoEmpleadoB2Valida(e ports.EvidenciaCatalogoEmpleadoB2, o ports.OrdenCatalogoEmpleadoB2) bool {
	x := o.Autorizacion.ResumenCapacidad()
	return e.DecisionRef == x.DecisionRef() && e.EfectoRef == o.Material.Recurso().Referencia &&
		referenciaEvidenciaB2.MatchString(e.AuditoriaRef) && huellaEvidenciaB2.MatchString(e.ConsumoHuellaSHA256) &&
		instanteCatalogoEmpleadoB2Valido(e.ConsultadaEn) && !e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func cursorCatalogoEmpleadoB2Valido(r ports.ResultadoConsultaCatalogoEmpleadoB2, m domain.MaterialCatalogoEmpleadoB2) bool {
	ultimoRef, ultimaVersion := m.CursorRef(), m.CursorVersion()
	for _, e := range r.Entradas {
		if e.Ref < ultimoRef || e.Ref == ultimoRef && e.Version <= ultimaVersion {
			return false
		}
		ultimoRef, ultimaVersion = e.Ref, e.Version
	}
	return r.CursorSiguiente == nil || len(r.Entradas) > 0 &&
		r.CursorSiguiente.Ref == ultimoRef && r.CursorSiguiente.Version == ultimaVersion
}

func entradaCatalogoEmpleadoB2IgualMaterial(e domain.EntradaCatalogoRegistroEmpleadoB2, m domain.MaterialCatalogoEmpleadoB2) bool {
	var datos struct {
		Denominacion string `json:"denominacion"`
		Huella       string `json:"huella_sha256"`
		Desde        string `json:"vigente_desde"`
		Hasta        string `json:"vigente_hasta"`
	}
	return json.Unmarshal(m.Canonico(), &datos) == nil && e.Denominacion == datos.Denominacion &&
		e.HuellaSHA256 == datos.Huella && e.VigenteDesde.Texto() == datos.Desde && e.VigenteHasta.Texto() == datos.Hasta
}

func reciboCatalogoEmpleadoB2Valido(r ports.ResultadoCambioCatalogoEmpleadoB2, o ports.OrdenCatalogoEmpleadoB2) bool {
	x := o.Autorizacion.ResumenCapacidad()
	a := r.AccesoActual
	recibo := r.Recibo
	if (a.EstadoReplay != "registrado" && a.EstadoReplay != "replay") || a.DecisionRef != x.DecisionRef() ||
		!referenciaEvidenciaB2.MatchString(a.AuditoriaRef) || !huellaEvidenciaB2.MatchString(a.ConsumoHuellaSHA256) ||
		!instanteCatalogoEmpleadoB2Valido(a.RegistradoEn) || a.RegistradoEn.Before(x.EmitidaEn()) || !a.RegistradoEn.Before(x.ExpiraEn()) ||
		!referenciaEvidenciaB2.MatchString(recibo.DecisionRef) || !referenciaEvidenciaB2.MatchString(recibo.AuditoriaRef) ||
		!huellaEvidenciaB2.MatchString(recibo.ConsumoHuellaSHA256) || !instanteCatalogoEmpleadoB2Valido(recibo.RegistradoEn) ||
		recibo.RegistradoEn.After(a.RegistradoEn) {
		return false
	}
	return a.EstadoReplay != "registrado" || recibo.DecisionRef == a.DecisionRef &&
		recibo.AuditoriaRef == a.AuditoriaRef && recibo.ConsumoHuellaSHA256 == a.ConsumoHuellaSHA256 &&
		recibo.RegistradoEn.Equal(a.RegistradoEn)
}

func instanteCatalogoEmpleadoB2Valido(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}

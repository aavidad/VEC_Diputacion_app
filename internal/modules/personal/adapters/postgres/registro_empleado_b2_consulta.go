package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maxRespuestaFichaEmpleadoB2 = 512 << 10
const maxRespuestaVacantesB2 = 256 << 10

var _ ports.RepositorioRegistroEmpleadoB2 = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConsultarFichaRRHH(ctx context.Context, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	var vacio ports.ResultadoFichaEmpleadoB2
	if r == nil || !materialFichaB2Valido(o.Material, o.Autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaFichaEmpleadoB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaFichaEmpleadoB2, func(bruto []byte) (ports.ResultadoFichaEmpleadoB2, error) {
		return decodificarFichaEmpleadoB2(bruto, o)
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ListarVacantesRRHH(ctx context.Context, o ports.OrdenVacantesB2) (ports.ResultadoVacantesB2, error) {
	var vacio ports.ResultadoVacantesB2
	if r == nil || !materialVacantesB2Valido(o.Material, o.Autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaVacantesB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaVacantesB2, func(bruto []byte) (ports.ResultadoVacantesB2, error) {
		return decodificarVacantesB2(bruto, o)
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func materialFichaB2Valido(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if m.Operacion() != "ficha" {
		return false
	}
	reconstruido, err := domain.NuevoMaterialFichaEmpleadoB2(domain.SolicitudFichaEmpleadoB2{EmpleadoRef: m.EmpleadoRef(), Corte: m.Corte(), Actor: m.Actor()})
	return err == nil && bytes.Equal(reconstruido.Canonico(), m.Canonico()) && autorizacionConsultaEmpleadoB2Valida(m, a, domain.AccionFichaEmpleadoB2, domain.AudienciaFichaEmpleadoB2)
}

func materialVacantesB2Valido(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if m.Operacion() != "vacantes" {
		return false
	}
	reconstruido, err := domain.NuevoMaterialVacantesB2(domain.SolicitudVacantesB2{OrganismoRef: m.OrganismoRef(), Corte: m.Corte(), Limite: m.Limite(), Cursor: m.Cursor(), Actor: m.Actor()})
	return err == nil && bytes.Equal(reconstruido.Canonico(), m.Canonico()) && autorizacionConsultaEmpleadoB2Valida(m, a, domain.AccionVacantesB2, domain.AudienciaVacantesB2)
}

func autorizacionConsultaEmpleadoB2Valida(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, audiencia string) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	actor := m.Actor()
	return err == nil && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == accion && x.AudienciaConsumo() == audiencia && x.EfectoRef() == m.Recurso().Referencia && x.EfectoHuellaSHA256() == h
}

func decodificarFichaEmpleadoB2(bruto []byte, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	var vacio ports.ResultadoFichaEmpleadoB2
	if err := formaRespuestaRegistroB2(bruto, "ficha"); err != nil {
		return vacio, err
	}
	var r ports.ResultadoFichaEmpleadoB2
	if err := decodificarJSONRegistroB2(bruto, &r); err != nil || r.Ficha.ValidarPara(o.Material) != nil || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

func decodificarVacantesB2(bruto []byte, o ports.OrdenVacantesB2) (ports.ResultadoVacantesB2, error) {
	var vacio ports.ResultadoVacantesB2
	if err := formaRespuestaRegistroB2(bruto, "pagina"); err != nil {
		return vacio, err
	}
	var r ports.ResultadoVacantesB2
	if err := decodificarJSONRegistroB2(bruto, &r); err != nil || r.Pagina.ValidarPara(o.Material) != nil || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

func decodificarJSONRegistroB2(bruto []byte, destino any) error {
	lector := json.NewDecoder(bytes.NewReader(bruto))
	lector.DisallowUnknownFields()
	if err := lector.Decode(destino); err != nil {
		return err
	}
	if lector.Decode(new(any)) != io.EOF {
		return errRegistroEmpleadoB2NoDisponible
	}
	return nil
}

func formaRespuestaRegistroB2(bruto []byte, clave string) error {
	if len(bruto) == 0 || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) || verificarJSONOrganizacionHistorica(bruto) != nil {
		return errRegistroEmpleadoB2NoDisponible
	}
	var raiz map[string]json.RawMessage
	if json.Unmarshal(bruto, &raiz) != nil || !clavesRegistroB2(raiz, []string{clave, "evidencia"}, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	var datos, evidencia map[string]json.RawMessage
	if json.Unmarshal(raiz[clave], &datos) != nil || json.Unmarshal(raiz["evidencia"], &evidencia) != nil || !clavesRegistroB2(evidencia, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	if clave == "ficha" {
		if !clavesRegistroB2(datos, []string{"empleado_ref", "persona_ref", "corte", "version", "relaciones", "ocupaciones", "situaciones", "servicios", "eficacia_administrativa", "firma_oficial"}, nil) {
			return errRegistroEmpleadoB2NoDisponible
		}
		for _, nombre := range []string{"relaciones", "ocupaciones", "situaciones", "servicios"} {
			if len(datos[nombre]) == 0 || datos[nombre][0] != '[' {
				return errRegistroEmpleadoB2NoDisponible
			}
		}
	} else {
		if !clavesRegistroB2(datos, []string{"organismo_ref", "corte", "limite", "cursor", "cobertura", "vacantes"}, []string{"cursor_siguiente"}) || len(datos["vacantes"]) == 0 || datos["vacantes"][0] != '[' {
			return errRegistroEmpleadoB2NoDisponible
		}
	}
	var corte map[string]json.RawMessage
	if json.Unmarshal(datos["corte"], &corte) != nil || !clavesRegistroB2(corte, []string{"vigente_en", "conocido_en"}, nil) {
		return errRegistroEmpleadoB2NoDisponible
	}
	return nil
}

func clavesRegistroB2(datos map[string]json.RawMessage, obligatorias, opcionales []string) bool {
	if datos == nil {
		return false
	}
	permitidas := make(map[string]bool, len(obligatorias)+len(opcionales))
	for _, clave := range obligatorias {
		permitidas[clave] = true
		if _, ok := datos[clave]; !ok {
			return false
		}
	}
	for _, clave := range opcionales {
		permitidas[clave] = true
	}
	for clave := range datos {
		if !permitidas[clave] {
			return false
		}
	}
	return true
}

var referenciaEvidenciaB2 = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
var huellaEvidenciaB2 = regexp.MustCompile(`^[a-f0-9]{64}$`)

func evidenciaRegistroB2Valida(e ports.EvidenciaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	return referenciaEvidenciaB2.MatchString(e.ReciboRef) && referenciaEvidenciaB2.MatchString(e.AuditoriaRef) && huellaEvidenciaB2.MatchString(e.ConsumoHuellaSHA256) &&
		e.DecisionRef == x.DecisionRef() && e.EfectoRef == x.EfectoRef() && !e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0 &&
		!e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func errorDominioRegistroEmpleadoB2(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, errRegistroEmpleadoB2Denegado) {
		return domain.ErrRegistroEmpleadoB2Denegado
	}
	if errors.Is(err, errRegistroEmpleadoB2NoEncontrado) {
		return domain.ErrRegistroEmpleadoB2NoEncontrado
	}
	if errors.Is(err, errRegistroEmpleadoB2Cobertura) {
		return domain.ErrCoberturaVacantesB2NoAcreditada
	}
	if errors.Is(err, errRegistroEmpleadoB2Conflicto) {
		return domain.ErrRegistroEmpleadoB2Conflicto
	}
	if errors.Is(err, errRegistroEmpleadoB2Invalido) {
		return domain.ErrRegistroEmpleadoB2Invalido
	}
	return domain.ErrRegistroEmpleadoB2NoDisponible
}

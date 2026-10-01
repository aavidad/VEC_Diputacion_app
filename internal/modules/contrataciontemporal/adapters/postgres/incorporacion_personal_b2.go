package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type FuentePlanNominalB2PostgreSQL struct{ pool iniciadorTransacciones }

func NuevaFuentePlanNominalB2PostgreSQL(p *pgxpool.Pool) (*FuentePlanNominalB2PostgreSQL, error) {
	if p == nil {
		return nil, ports.ErrPlanNominalB2NoDisponible
	}
	return &FuentePlanNominalB2PostgreSQL{p}, nil
}

var _ ports.RepositorioPlanNominalB2 = (*FuentePlanNominalB2PostgreSQL)(nil)

func (f *FuentePlanNominalB2PostgreSQL) ejecutar(ctx context.Context, accion string, b []byte, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, validar func([]byte) error, variante ...string) error {
	if ctx == nil || f == nil || dependenciaNula(f.pool) || len(b) == 0 || len(b) > 65536 {
		return ports.ErrPlanNominalB2NoDisponible
	}
	var selector struct {
		UnidadRef       string                              `json:"unidad_ref"`
		UnidadCTRef     string                              `json:"unidad_ct_ref"`
		Material        *domain.PlanIncorporacionPersonalB2 `json:"material"`
		OrganizacionRef string                              `json:"organizacion_ref"`
		ExpedienteRef   string                              `json:"expediente_ref"`
		Solicitud       *ports.SolicitudPlanNominalB2       `json:"solicitud"`
	}
	if json.Unmarshal(b, &selector) != nil {
		return ports.ErrPlanNominalB2Invalido
	}
	org, exp := selector.OrganizacionRef, selector.ExpedienteRef
	unidad := selector.UnidadRef
	if selector.UnidadCTRef != "" {
		unidad = selector.UnidadCTRef
	}
	if selector.Material != nil {
		unidad = selector.Material.UnidadCTRef
	}
	if selector.Solicitud != nil {
		org, exp = selector.Solicitud.OrganizacionRef, selector.Solicitud.ExpedienteRef
	}
	audiencia := map[string]string{ports.AccionRegistrarPlanNominalB2: ports.AudienciaRegistrarPlanNominalB2, ports.AccionLeerPlanNominalB2: ports.AudienciaLeerPlanNominalB2, ports.AccionConfirmarOrigenB2: ports.AudienciaConfirmarOrigenB2}[accion]
	if audiencia == "" || !capacidadParaVinculoRPT(a, accion, audiencia, exp, "contratacion_temporal", ports.TipoRecursoPlanNominalB2, map[string]string{"organizacion_ref": org, "unidad_ref": unidad}, b) {
		return ports.ErrPlanNominalB2Denegado
	}
	args := append([]any{string(b)}, argumentosCapacidadVinculoRPT(a)...)
	defer func() {
		for _, v := range args {
			if x, ok := v.([]byte); ok {
				clear(x)
			}
		}
	}()
	fn := map[string]string{ports.AccionRegistrarPlanNominalB2: "registrar_plan_nominal_b2_v1", ports.AccionLeerPlanNominalB2: "leer_plan_nominal_b2_v1", ports.AccionConfirmarOrigenB2: "confirmar_origen_incorporacion_b2_v1"}[accion]
	if len(variante) > 0 {
		switch variante[0] {
		case "origen":
			if accion != ports.AccionLeerPlanNominalB2 {
				return ports.ErrPlanNominalB2Denegado
			}
			fn = "leer_origen_incorporacion_b2_v1"
		case "antecedentes":
			if accion != ports.AccionLeerPlanNominalB2 {
				return ports.ErrPlanNominalB2Denegado
			}
			fn = "leer_antecedentes_plan_b2_v1"
		default:
			return ports.ErrPlanNominalB2Invalido
		}
	}
	var err error
	for intento := 0; intento < maximoIntentosSeguimiento; intento++ {
		err = func() error {
			tx, e := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				return e
			}
			defer revertirTransaccion(tx)
			if e = configurarTransaccionSeguimiento(ctx, tx); e != nil {
				return e
			}
			var raw []byte
			// fn pertenece al conjunto cerrado anterior; nunca se recibe del canal.
			if e = tx.QueryRow(ctx, "SELECT vec_contratacion_temporal."+fn+"($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text", args...).Scan(&raw); e != nil {
				return e
			}
			defer clear(raw)
			if resultadoPlanB2Ausente(raw) {
				if e = tx.Commit(ctx); e != nil {
					return e
				}
				return ports.ErrPlanNominalB2NoEncontrado
			}
			if e = validar(raw); e != nil {
				return e
			}
			return tx.Commit(ctx)
		}()
		if err == nil || ctx.Err() != nil || !errorPostgreSQLReintentable(err) {
			break
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return ports.ErrPlanNominalB2Denegado
		case "22023":
			return ports.ErrPlanNominalB2Invalido
		case "23505", "40001", "55000":
			return ports.ErrPlanNominalB2Conflicto
		}
		return ports.ErrPlanNominalB2NoDisponible
	}
	return err
}

func resultadoPlanB2Ausente(raw []byte) bool {
	return raw == nil || string(raw) == "null"
}

func (f *FuentePlanNominalB2PostgreSQL) RegistrarPlanNominalB2(ctx context.Context, m ports.RegistroPlanNominalB2, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ContratoPlanNominalB2, error) {
	var c ports.ContratoPlanNominalB2
	if m.Material.Validar() != nil {
		return c, ports.ErrPlanNominalB2Invalido
	}
	b, e := domain.CanonicoPlanPersonalB2(m)
	if e != nil {
		return c, e
	}
	sha, _ := domain.SHA256PlanPersonalB2(m)
	e = f.ejecutar(ctx, ports.AccionRegistrarPlanNominalB2, b, a, func(raw []byte) error {
		if decodificarJSONEstricto(raw, &c) != nil || c.Material != m.Material || c.PlanSHA256 != sha || !contratoSQLB2Valido(c) {
			return ports.ErrPlanNominalB2NoDisponible
		}
		return nil
	})
	if e != nil {
		return ports.ContratoPlanNominalB2{}, e
	}
	return c, nil
}
func (f *FuentePlanNominalB2PostgreSQL) LeerContratoPlanNominal(ctx context.Context, org, exp string, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, unidad ...string) (ports.ContratoPlanNominalB2, error) {
	var c ports.ContratoPlanNominalB2
	b, e := domain.CanonicoPlanPersonalB2(map[string]string{"organizacion_ref": org, "expediente_ref": exp, "unidad_ref": unidadPlanSQLB2(unidad)})
	if e != nil {
		return c, e
	}
	e = f.ejecutar(ctx, ports.AccionLeerPlanNominalB2, b, a, func(raw []byte) error {
		if decodificarJSONEstricto(raw, &c) != nil || c.Material.OrganizacionRef != org || c.Material.ExpedienteRef != exp || !contratoSQLB2Valido(c) {
			return ports.ErrPlanNominalB2NoDisponible
		}
		return nil
	})
	if e != nil {
		return ports.ContratoPlanNominalB2{}, e
	}
	return c, nil
}
func contratoSQLB2Valido(c ports.ContratoPlanNominalB2) bool {
	if c.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || c.PlanVersion != 1 || c.IntencionVersion != 1 || c.Material.Validar() != nil || !domain.UUIDPlanPersonalB2Valido(c.IdempotenciaPersonalUUID) || !domain.HuellaPlanPersonalB2Valida(c.PlanSHA256) || !domain.InstanteUTCCanonico(c.RegistradoEn) {
		return false
	}
	for _, r := range []string{c.PlanRef, c.PlanReciboRef, c.IntencionRef, c.IntencionReciboRef, c.SolicitudPersonalRef} {
		if !domain.ReferenciaOpacaValida(r) {
			return false
		}
	}
	h, e := domain.SHA256PlanPersonalB2(ports.RegistroPlanNominalB2{Solicitud: c.Solicitud, Material: c.Material})
	return e == nil && h == c.PlanSHA256
}
func (f *FuentePlanNominalB2PostgreSQL) ConfirmarOrigenIncorporacionB2(ctx context.Context, m ports.ConfirmacionOrigenIncorporacionB2, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.OrigenIncorporacionPersonalB2, error) {
	var o ports.OrigenIncorporacionPersonalB2
	b, e := domain.CanonicoPlanPersonalB2(m)
	if e != nil {
		return o, e
	}
	e = f.ejecutar(ctx, ports.AccionConfirmarOrigenB2, b, a, func(raw []byte) error {
		if decodificarJSONEstricto(raw, &o) != nil || o.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || o.Confirmacion != m || o.FirmaOficial || o.EficaciaAdministrativa || !domain.InstanteUTCCanonico(o.RegistradoEn) || !domain.ReferenciaOpacaValida(o.ReciboRef) || !domain.ReferenciaOpacaValida(o.AuditoriaRef) || !domain.ReferenciaOpacaValida(o.OutboxRef) {
			return ports.ErrPlanNominalB2NoDisponible
		}
		return nil
	})
	if e != nil {
		return ports.OrigenIncorporacionPersonalB2{}, e
	}
	return o, nil
}

func (f *FuentePlanNominalB2PostgreSQL) LeerOrigenIncorporacionB2(ctx context.Context, org, exp string, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, unidad ...string) (ports.OrigenIncorporacionPersonalB2, bool, error) {
	var o ports.OrigenIncorporacionPersonalB2
	b, _ := domain.CanonicoPlanPersonalB2(map[string]string{"organizacion_ref": org, "expediente_ref": exp, "unidad_ref": unidadPlanSQLB2(unidad)})
	e := f.ejecutar(ctx, ports.AccionLeerPlanNominalB2, b, a, func(raw []byte) error {
		if decodificarJSONEstricto(raw, &o) != nil || o.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || o.Confirmacion.OrganizacionRef != org || o.Confirmacion.ExpedienteRef != exp || o.FirmaOficial || o.EficaciaAdministrativa || !domain.InstanteUTCCanonico(o.RegistradoEn) {
			return ports.ErrPlanNominalB2NoDisponible
		}
		return nil
	}, "origen")
	if errors.Is(e, ports.ErrPlanNominalB2NoEncontrado) {
		return ports.OrigenIncorporacionPersonalB2{}, false, nil
	}
	if e != nil {
		return ports.OrigenIncorporacionPersonalB2{}, false, e
	}
	return o, true, nil
}
func (f *FuentePlanNominalB2PostgreSQL) LeerAntecedentesPlanB2(ctx context.Context, org, exp string, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, unidad ...string) (ports.AntecedentesPlanNominalB2, error) {
	var o ports.AntecedentesPlanNominalB2
	b, _ := domain.CanonicoPlanPersonalB2(map[string]string{"organizacion_ref": org, "expediente_ref": exp, "unidad_ref": unidadPlanSQLB2(unidad)})
	e := f.ejecutar(ctx, ports.AccionLeerPlanNominalB2, b, a, func(raw []byte) error {
		if decodificarJSONEstricto(raw, &o) != nil || o.OrganizacionRef != org || o.ExpedienteRef != exp || !domain.VersionPlanPersonalB2Valida(o.VersionExpediente) || !domain.VersionPlanPersonalB2Valida(o.AnalisisVersion) || !domain.HuellaPlanPersonalB2Valida(o.AnalisisSHA256) || o.Vinculo == nil || o.Vinculo.Validar() != nil {
			return ports.ErrPlanNominalB2NoDisponible
		}
		return nil
	}, "antecedentes")
	if e != nil {
		return ports.AntecedentesPlanNominalB2{}, e
	}
	return o, nil
}

func unidadPlanSQLB2(unidades []string) string {
	if len(unidades) != 1 {
		return ""
	}
	return unidades[0]
}

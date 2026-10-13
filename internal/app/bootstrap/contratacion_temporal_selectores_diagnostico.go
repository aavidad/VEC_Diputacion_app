package bootstrap

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Los argumentos de diagnóstico proceden exclusivamente de literales de los
// puntos de composición, nunca de configuración, SQL o identidad. Se conserva
// la causa para errors.Is/As, pero el registro no imprime su mensaje libre.
func diagnosticarSelectorCTArranque(selector, componente, etapa, requisito string, centinela, causa error) error {
	slog.Error("contratacion temporal: selector no compuesto",
		"selector", selector, "etapa", etapa, "requisito", requisito,
		"causa", causaFalloPostgreSQLCTDesarrollo(causa))
	return marcarFalloComponenteArranque(componente+"_"+etapa,
		&falloFuenteConfiguracionArranque{causa: errors.Join(centinela, causa)})
}

func requisitoSeguimientoCTArranque(etapa string) string {
	switch etapa {
	case "catalogo_causas":
		return "VEC_CT_CAUSAS_CESE_SOURCE_PATH"
	case "catalogo_motivos":
		return "VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH"
	case "catalogo_reglas", "resolutor":
		return "VEC_CT_REGLAS_SOURCE_PATH"
	case "perfil_reincorporacion_ct153":
		return "CT153: perfil_reincorporacion_ct153_v1()=organizacion_ref; EXECUTE del LOGIN ejecutor"
	case "perfil_fijo_reincorporacion":
		return "VEC_CT_PROVISION_PERFILES_RRHH_APROBACION; VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES: CAS propio"
	case "instantanea":
		return "sesion_nominal_RRHH; contexto_reincorporacion_registrado; gobierno_motivos"
	case "proveedor_material":
		return "VEC_CT_GOBIERNO_DATABASE_URL: gobierno_audiencias_AD92_AD98"
	default:
		return "dependencias_seguimiento_CT"
	}
}

// Una consulta identifica cada objeto ausente; no se aplica ninguna migración
// ni se leen datos de expedientes. Los requisitos son etiquetas compiladas.
func comprobarMigracionesReincorporacionCTArranque(ctx context.Context, pool *pgxpool.Pool) error {
	var presentes [6]bool
	err := pool.QueryRow(ctx, `SELECT
		to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()') IS NOT NULL,
		to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NOT NULL,
		to_regprocedure('vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)') IS NOT NULL,
		to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
		to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
		to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(
		&presentes[0], &presentes[1], &presentes[2], &presentes[3], &presentes[4], &presentes[5])
	fallar := func(requisito string, causa error) error {
		return diagnosticarSelectorCTArranque(envCTReincorporacionTitularEnabled, "ct_reincorporacion", "migraciones",
			requisito, errSeguimientoCeseDesarrolloNoDisponible, causa)
	}
	if err != nil {
		return fallar("VEC_CT_DATABASE_URL: consulta_catalogo_postgresql", err)
	}
	for i, requisito := range [...]string{
		"CT153: perfil_reincorporacion_ct153_v1()",
		"CT130: reincorporacion_titular_v1",
		"CT130: preparar_reincorporacion_titular_v1(jsonb)",
		"CT130: confirmar_reincorporacion_titular_v1",
		"CT134: leer_antecedente_reincorporacion_titular_atestada_v1",
		"AD98: registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada",
	} {
		if !presentes[i] {
			return fallar(requisito, nil)
		}
	}
	return nil
}

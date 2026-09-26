package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Incorporación acreditada de CT (dudas 11 y 12 de RRHH, CT 000124 y AD3-88)
// en el perfil de desarrollo: confirmación de GINPIX por RRHH, cierre con el
// número de esa confirmación y confirmación de la incorporación por el
// centro. Va tras VEC_CT_INCORPORACION_ACREDITADA_ENABLED y sobre el
// seguimiento de cese.

// ErrIncorporacionAcreditadaMigracionesNoDisponibles detiene el arranque si
// el selector está encendido y falta alguna migración de la que depende.
var ErrIncorporacionAcreditadaMigracionesNoDisponibles = errors.New("bootstrap: incorporación acreditada de CT sin sus migraciones")

var (
	ErrIncorporacionAcreditadaFaltaAD388   = fmt.Errorf("%w: falta AD3-88 (consumidores de GINPIX, del centro y de la no incorporación)", ErrIncorporacionAcreditadaMigracionesNoDisponibles)
	ErrIncorporacionAcreditadaFaltaCT124   = fmt.Errorf("%w: falta CT 000124 (incorporación acreditada)", ErrIncorporacionAcreditadaMigracionesNoDisponibles)
	errIncorporacionAcreditadaComprobacion = fmt.Errorf("%w: no se pudo comprobar el catálogo", ErrIncorporacionAcreditadaMigracionesNoDisponibles)
)

// consultaMigracionesIncorporacionAcreditada se ejecuta con el LOGIN ejecutor
// de CT: exige que las fachadas de CT 000124 existan y pueda ejecutarlas; de
// AD3-88 comprueba en el catálogo sus fachadas, que ese LOGIN no ejecuta.
const consultaMigracionesIncorporacionAcreditada = `SELECT
 (SELECT count(*)=3 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN ('registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada',
     'registrar_y_consumir_incorporacion_centro_v3_atestada','registrar_y_consumir_no_incorporacion_ct_v3_atestada')),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_contratacion_temporal.preparar_confirmacion_ginpix_v1(jsonb)',
  'vec_contratacion_temporal.confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.consultar_incorporaciones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.expediente_del_centro_v1(text,text,text)',
  'vec_contratacion_temporal.confirmar_incorporacion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text)',
  'vec_contratacion_temporal.preparar_no_incorporacion_v1(jsonb)',
  'vec_contratacion_temporal.confirmar_no_incorporacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(bigint,text,integer)']) f)`

// comprobarMigracionesIncorporacionAcreditadaDesarrollo se llama al arrancar,
// solo con el selector encendido, antes de publicar sus claves.
func comprobarMigracionesIncorporacionAcreditadaDesarrollo(ctx context.Context, ct *pgxpool.Pool) error {
	if ct == nil {
		return errIncorporacionAcreditadaComprobacion
	}
	var ad388, ct124 bool
	if err := ct.QueryRow(ctx, consultaMigracionesIncorporacionAcreditada).Scan(&ad388, &ct124); err != nil {
		return errIncorporacionAcreditadaComprobacion
	}
	switch {
	case !ad388:
		return ErrIncorporacionAcreditadaFaltaAD388
	case !ct124:
		return ErrIncorporacionAcreditadaFaltaCT124
	}
	return nil
}

// incorporacionAcreditadaSolicitada lee el selector ya validado en la raíz.
func incorporacionAcreditadaSolicitada(cfg config.Config) bool {
	activo, err := cfg.CTIncorporacionAcreditadaDesarrolloActivo()
	if err != nil {
		slog.Error("incorporación acreditada de CT no compuesta: selector inválido", "causa", err)
		return false
	}
	return activo
}

// descriptorMaterialConfirmacionGINPIXDesarrollo es la audiencia de GINPIX;
// la del centro reutiliza el material de las peticiones de centro.
func descriptorMaterialConfirmacionGINPIXDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaConsumoConfirmacionGINPIXV1,
		Dominio: "vec.ct.confirmacion-ginpix.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-confirmacion-ginpix:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
}

// descriptorMaterialNoIncorporacionDesarrollo: audiencia propia de la no
// incorporación (AD3-88).
func descriptorMaterialNoIncorporacionDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaConsumoNoIncorporacionV1,
		Dominio: "vec.ct.no-incorporacion.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-no-incorporacion:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
}

// Atributos de c22: «etiqueta_<motivo>», «consecuencia_<motivo>» y
// «segunda_persona» («si» exige que resuelva otra persona).
const (
	prefijoEtiquetaNoIncorporacion     = "etiqueta_"
	prefijoConsecuenciaNoIncorporacion = "consecuencia_"
	atributoSegundaPersona             = "segunda_persona"
	valorSegundaPersonaExigida         = "si"
)

// ReglaNoIncorporacion lee la regla c22 vigente (motivos, consecuencia de
// cada uno en Bolsa y segregación); la ampara esa misma regla con el motivo
// de su ruta. Sin la regla, la operación no está disponible.
func (f fuenteReglasSeguimientoDesarrollo) ReglaNoIncorporacion(ctx context.Context, instante time.Time) (ports.ReglaNoIncorporacion, ports.PoliticaOperacionSeguimiento, error) {
	regla, err := f.reglas.Regla(ctx, reglas.CTNoIncorporacion)
	if err != nil {
		return ports.ReglaNoIncorporacion{}, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	r := ports.ReglaNoIncorporacion{SegundaPersona: regla.Atributos[atributoSegundaPersona] == valorSegundaPersonaExigida}
	for _, clave := range regla.Elementos() {
		r.Motivos = append(r.Motivos, ports.MotivoNoIncorporacion{Clave: clave, Etiqueta: regla.Atributos[prefijoEtiquetaNoIncorporacion+clave],
			ConsecuenciaClave: regla.Atributos[prefijoConsecuenciaNoIncorporacion+clave]})
	}
	if !r.Valida() {
		return ports.ReglaNoIncorporacion{}, ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	return r, politicaSeguimientoDesarrollo(regla, httpinterno.RutaNoIncorporaciones, instante), nil
}

// PoliticaConfirmacionGINPIX: la confirmación la ampara la regla c10 (la que
// exige GINPIX confirmado), con el motivo de su propia ruta.
func (f fuenteReglasSeguimientoDesarrollo) PoliticaConfirmacionGINPIX(ctx context.Context, instante time.Time) (ports.PoliticaOperacionSeguimiento, error) {
	regla, err := f.reglas.Regla(ctx, reglas.CTCierreExpediente)
	if err != nil {
		return ports.PoliticaOperacionSeguimiento{}, errSeguimientoCeseDesarrolloNoDisponible
	}
	return politicaSeguimientoDesarrollo(regla, httpinterno.RutaConfirmacionesGINPIX, instante), nil
}

// Valor del atributo de c10 que admite el cierre administrativo sin cese.
const cierreSinCeseAdmitidoCatalogo = "admitido"

// admisionCierreSinCeseDesarrollo decide con la regla c10 vigente si se
// ofrece el cierre administrativo sin cese. Sin catálogo de reglas se
// conserva la conducta anterior (se ofrece).
func admisionCierreSinCeseDesarrollo(resolutor *reglas.Resolutor) func(context.Context) (bool, error) {
	if resolutor == nil {
		return nil
	}
	return func(ctx context.Context) (bool, error) {
		regla, err := resolutor.Regla(ctx, reglas.CTCierreExpediente)
		if err != nil {
			return false, err
		}
		return regla.Atributos[reglas.AtributoCierreSinCese] == cierreSinCeseAdmitidoCatalogo, nil
	}
}

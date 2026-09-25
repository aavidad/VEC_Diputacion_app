#!/usr/bin/env bash
# Ensambla Dietas desde fuentes canonicas. Imprime SQL para psql con una sola
# transaccion y la marca `:finalizar;`, que quien ejecuta sustituye por
# ROLLBACK (ensayo) o COMMIT.
#
#   04_dietas_migraciones.sh                paquete R1 (sin cambios): requiere
#                                           el nucleo AD3-48 y las migraciones
#                                           de main ya instalados.
#   04_dietas_migraciones.sh --incremental  paquete D2-D7 idempotente: Dietas
#                                           000005-000011, AD3-59/80/75 y
#                                           Personal 000012/000013 en orden
#                                           causal. Cada migracion va
#                                           tras una comprobacion de su marca
#                                           en el catalogo y se omite si ya
#                                           esta instalada; nunca se reaplica.
#
# Requisitos del incremental que NO instala (tienen su propio paquete): R1
# (Dietas 000001-000004), AD3-51/52, Personal 000010/000011 y Composicion
# 000010a. Si faltan, la precondicion de la migracion que los exige
# aborta la transaccion completa (55000) sin dejar rastro.
#
# Orden verificado contra las precondiciones SQL de cada migracion:
#   000005 (tras 000004)
#   AD3-59 (tras AD3-52; crea las fachadas documento/consulta/circuito y D7)
#   Personal 000012 (exige las fachadas D7 de AD3-59) -> 000013 (tras 000012)
#   000006 (exige las fachadas documento y consulta de AD3-59 y la
#           revalidacion de asignacion de Personal 000012)
#   000007 (exige 000006 y las fachadas circuito/bandeja/prelectura de AD3-59)
#   AD3-80 (exige AD3-59; crea la fachada del revisor)
#   000008 (exige 000007 y la fachada de AD3-80)
#   AD3-75 (exige los cuerpos exactos de AD3-59 y AD3-80; ninguna migracion
#           Dietas la exige ni ella exige ninguna de Dietas)
#   000009 (exige 000008) -> 000010 (exige 000009) -> 000011 (exige 000010)
# 000010 y 000011 van siempre juntas y seguidas, en la misma transaccion:
# 000010 sola proyecta la devolucion sin cotejar el campo de la decision V3.
# Con AD3-75 antes de 000009 la lista con devolucion sigue cerrada en Dietas
# (sus cotejos la rechazan) hasta 000011. Despues: politica V3
# (politica_dietas_d5_d6.py) y, por ultimo, el binario.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)

volcar() {
  local relativa=$1 fichero
  fichero=$repo/deploy/postgresql/$relativa
  printf -- '-- INICIO deploy/postgresql/%s\n' "$relativa"
  grep -vE '^\\set ON_ERROR_STOP on$|^BEGIN;$|^COMMIT;$' -- "$fichero"
  printf -- '-- FIN deploy/postgresql/%s\n' "$relativa"
}

if [[ $# -eq 0 ]]; then
  migraciones=(
    dietas_borradores/roles_up.sql
    personal/migraciones/000007_relacion_empleado_dietas.up.sql
    autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas.up.sql
    autorizacion_atestada_v3/migraciones/000050_acceso_rutas_dietas.up.sql
    personal/migraciones/000008_consulta_relaciones_propias_dietas.up.sql
    personal/migraciones/000009_asignacion_dietas.up.sql
    dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql
    dietas_borradores/migraciones/000002_tarifas_provisionales.up.sql
    dietas_borradores/migraciones/000003_consulta_tarifas_provisionales.up.sql
    dietas_borradores/migraciones/000004_calculo_comision.up.sql
  )
  printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
  for relativa in "${migraciones[@]}"; do
    volcar "$relativa"
  done
  printf '%s\n' ':finalizar;'
  exit 0
fi

if [[ $# -ne 1 || $1 != --incremental ]]; then
  echo 'uso: 04_dietas_migraciones.sh [--incremental]' >&2
  exit 2
fi

firma_v3='(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
firma_dietas='(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
ad3=vec_autorizacion_atestada_v3
# migracion|marca SQL, verdadera si la migracion ya esta instalada.
incremental=(
  "dietas_borradores/migraciones/000005_auditoria_frontera.up.sql|to_regprocedure('vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)') IS NOT NULL"
  "autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql|to_regprocedure('$ad3.registrar_y_consumir_dietas_documento_v3_atestada$firma_v3') IS NOT NULL"
  "personal/migraciones/000012_asignacion_dietas.up.sql|to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NOT NULL"
  "personal/migraciones/000013_auditoria_frontera_asignacion_dietas.up.sql|to_regclass('vec_personal.auditoria_frontera_asignacion_dietas') IS NOT NULL"
  "dietas_borradores/migraciones/000006_documento_comision.up.sql|to_regclass('vec_dietas.comision_revision') IS NOT NULL"
  "dietas_borradores/migraciones/000007_circuito_comision.up.sql|to_regclass('vec_dietas.cola_circuito_comision') IS NOT NULL"
  "autorizacion_atestada_v3/migraciones/000080_consumidor_revisor_documento_dietas.up.sql|to_regprocedure('$ad3.registrar_y_consumir_dietas_revisor_documento_v3_atestada$firma_v3') IS NOT NULL"
  "dietas_borradores/migraciones/000008_revision_circuito_comision.up.sql|to_regprocedure('vec_dietas.consultar_documento_circuito_v1$firma_dietas') IS NOT NULL"
  "autorizacion_atestada_v3/migraciones/000075_campo_devolucion_documento_dietas.up.sql|EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('$ad3.registrar_y_consumir_dietas_documento_v3_atestada$firma_v3') AND strpos(prosrc,'campos_devolucion')>0)"
  "dietas_borradores/migraciones/000009_otros_gastos_justificados.up.sql|to_regclass('vec_dietas.tipo_otro_gasto_provisional') IS NOT NULL"
  "dietas_borradores/migraciones/000010_devolucion_reenvio_comision.up.sql|to_regprocedure('vec_dietas.devolucion_vigente_comision_v1(text,bigint)') IS NOT NULL"
  "dietas_borradores/migraciones/000011_campo_devolucion_decision.up.sql|EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_dietas.autorizar_documento_v2$firma_dietas') AND strpos(prosrc,'vec.dietas.campo_devolucion')>0)"
)

printf '%s\n' '\set ON_ERROR_STOP on' 'BEGIN;'
for entrada in "${incremental[@]}"; do
  relativa=${entrada%%|*}
  marca=${entrada#*|}
  # Cada migracion fija con SET LOCAL su rol y su entorno; se restablecen para
  # que la marca y la siguiente precondicion se evaluen como la sesion DBA.
  printf '%s\n' 'RESET ROLE;' 'RESET ALL;'
  printf 'SELECT (%s) AS vec_dietas_instalada \\gset\n' "$marca"
  printf '%s\n' '\if :vec_dietas_instalada'
  printf '\\echo %s\n' "'OMITIDA, ya instalada: deploy/postgresql/$relativa'"
  printf '%s\n' '\else'
  volcar "$relativa"
  printf '%s\n' '\endif'
done
# 000010 sin 000011 expondria la devolucion sin cotejar la decision V3.
cat <<SQL
RESET ROLE;
RESET ALL;
DO \$tanda\$ BEGIN
 IF to_regprocedure('vec_dietas.devolucion_vigente_comision_v1(text,bigint)') IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_dietas.autorizar_documento_v2$firma_dietas')
      AND strpos(prosrc,'vec.dietas.campo_devolucion')>0)
 THEN RAISE EXCEPTION 'Dietas 000010 sin 000011' USING ERRCODE='55000'; END IF;
END \$tanda\$;
SQL
printf '%s\n' ':finalizar;'

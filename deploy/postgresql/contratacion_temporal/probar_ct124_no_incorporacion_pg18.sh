#!/usr/bin/env bash
# Ensayo de la cadena «aceptación → no incorporación → baja en Bolsa →
# siguiente llamamiento» (AD3-88, CT124 y Bolsa 000042) en PostgreSQL 18.4
# desechable sobre la estructura real restaurada de la principal (volcado con
# datos sintéticos). Instala la cadena previa si el volcado no la trae
# (AD3-82/83; CT110/111/113/115/116/119/121; Bolsa 000019-000039); comprueba
# ROLLBACK, UP, doble UP, DOWN exacto (núcleo AD3, continuación CT119/CT121 y
# su restricción, cierre CT115, origen de versión y guardado de Bolsa) y UP
# otra vez; recorre con dobles explícitos de las fachadas AD3 la no
# incorporación del expediente A (aceptado, nombrado y sin incorporación), su
# publicación, la baja por la bandeja de Bolsa, el antecedente de la
# continuación, el siguiente llamamiento en Bolsa y la confirmación de la
# continuación; reinicia PostgreSQL, repite los materiales (mismos recibos,
# ninguna fila nueva) y comprueba que los DOWN se niegan con historia.
# Uso: probar_ct124_no_incorporacion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct124ni-<pid> y se borran al terminar.
# VEC_CT124_CONSERVAR=1 deja el contenedor para depurar (borrarlo a mano).
# Con VEC_CT124_GO=1 publica el puerto solo en 127.0.0.1 y, en lugar de la
# cadena SQL, ejecuta las pruebas de contrato Go↔SQL de CT y de Bolsa.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct124ni-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
[[ -n ${VEC_CT124_CONSERVAR:-} ]] || trap limpiar EXIT
mkdir -p "$datos"
red=(--network none)
[[ ${VEC_CT124_GO:-} == 1 ]] && red=(-p 127.0.0.1::5432)
docker run -d --rm "${red[@]}" --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then break; fi
  sleep 0.5
done
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
igual() { [[ $1 == "$2" ]] || { echo "FALLO $3" >&2; exit 1; }; ok "$3"; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() { # $1 marca final, $2... ficheros en una sola sesión
  local marca=$1 salida; shift
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -8 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
bolsa=$repo/deploy/postgresql/bolsa_llamamientos/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
echo '== Cadena previa (si falta): AD3-82/83, CT110-CT121, Bolsa 000019-000039'
if [[ $(escalar "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]; then
  run <"$ad3/000082_consumidor_cese_cierre_contratacion_temporal.up.sql"
  run <"$ad3/000083_consumidor_modificacion_tras_nombramiento_ct.up.sql"
fi
if [[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL") == t ]]; then
  run <"$ct/000113_publicacion_contratos_bolsa.up.sql"
  run <"$ct/000115_cese_y_cierre_expediente.up.sql"
  run <"$ct/000116_modificacion_tras_nombramiento.up.sql"
fi
if [[ $(escalar "SELECT to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]; then
  run <"$ct/000110_fase_desde_cuadro_rrhh.up.sql"
  run <"$ct/000111_plazo_respuesta_llamamiento.up.sql"
  run <"$ct/000119_continuacion_tras_expiracion.up.sql"
  run <"$ct/000121_sucesor_tras_expiracion.up.sql"
fi
if [[ $(escalar "SELECT to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL") == t ]]; then
  for m in 000019 000021 000024 000025 000026 000028 000031 000032 000033 000034 000035 000037 000039; do
    run <"$(ls "$bolsa/${m}"_*.up.sql)"
  done
fi

nucleo="SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))||md5(pg_get_constraintdef(c.oid)) FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
estado_ct="SELECT md5(string_agg(pg_get_functiondef(p.oid)||coalesce(p.proacl::text,'')||coalesce(p.proconfig::text,''),'' ORDER BY p.oid::regprocedure::text))
  ||(SELECT md5(string_agg(pg_get_constraintdef(oid),'' ORDER BY conname)) FROM pg_constraint
      WHERE conname IN ('expediente_version_integral_origen_version_check','continuacion_confirmacion_completa'))
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal'
   AND p.proname IN ('preparar_cierre_expediente_v1','confirmar_cierre_expediente_v1','continuar_llamamiento_rrhh_v2',
     'registrar_comunicacion_llamamiento_local_v1','registrar_respuesta_recibida_rrhh_v1','consultar_justificante_respuesta_recibida_rrhh_v1',
     'registrar_resolucion_manual_respuesta_rrhh_v1','leer_expediente_aviso_confirmado_v1')"
guardado="SELECT md5(pg_get_functiondef(p.oid))||coalesce(p.proacl::text,'') FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure"
inicial_nucleo=$(escalar "$nucleo"); inicial_ct=$(escalar "$estado_ct"); inicial_guardado=$(escalar "$guardado")

echo '== AD3-88: ROLLBACK, UP, doble UP, DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql" | run
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'ROLLBACK AD3-88 conserva el núcleo'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql"; ok 'UP AD3-88'
con_ad388=$(escalar "$nucleo")
falla_con "$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql" 'preimagen incompatible'; ok 'doble UP AD3-88 rechazado'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.down.sql"
igual "$(escalar "$nucleo")" "$inicial_nucleo" 'DOWN AD3-88 restaura exactamente núcleo y audiencias'
run <"$ad3/000088_consumidor_incorporacion_acreditada_ct.up.sql"
igual "$(escalar "$nucleo")" "$con_ad388" 'UP tras DOWN reproduce el núcleo'
igual "$(escalar "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" f 'solo el propietario de CT invoca la fachada de no incorporación'

echo '== CT124: ROLLBACK, UP, doble UP, DOWN exacto, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$ct/000124_incorporacion_acreditada.up.sql" | run
igual "$(escalar "$estado_ct")" "$inicial_ct" 'ROLLBACK CT124 conserva cierre, continuación, restricción y origen'
run <"$ct/000124_incorporacion_acreditada.up.sql"; ok 'UP CT124'
con_ct124=$(escalar "$estado_ct")
[[ $con_ct124 != "$inicial_ct" ]] || { echo 'FALLO: CT124 no amplió cierre ni continuación' >&2; exit 1; }
falla_con "$ct/000124_incorporacion_acreditada.up.sql" 'CT124 ya instalada'; ok 'doble UP CT124 rechazado'
run <"$ct/000124_incorporacion_acreditada.down.sql"
igual "$(escalar "$estado_ct")" "$inicial_ct" 'DOWN CT124 restaura exactamente las ocho funciones, su ACL, la restricción y el origen'
run <"$ct/000124_incorporacion_acreditada.up.sql"
igual "$(escalar "$estado_ct")" "$con_ct124" 'UP tras DOWN reproduce CT124'
igual "$(escalar "SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal'
  AND p.proname IN ('preparar_no_incorporacion_v1','confirmar_no_incorporacion_v1','leer_no_incorporaciones_bolsa_v1')
  AND p.prosecdef AND has_function_privilege('vec_contratacion_temporal_ejecutor',p.oid,'EXECUTE') AND NOT has_function_privilege('public',p.oid,'EXECUTE')")" 3 'tres fachadas definidoras de la no incorporación solo para el ejecutor'

echo '== Bolsa 000042: ROLLBACK, UP, doble UP, DOWN exacto, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$bolsa/000042_no_incorporacion_ct.up.sql" | run
igual "$(escalar "$guardado")" "$inicial_guardado" 'ROLLBACK Bolsa 000042 conserva el guardado'
run <"$bolsa/000042_no_incorporacion_ct.up.sql"; ok 'UP Bolsa 000042'
con_b42=$(escalar "$guardado")
falla_con "$bolsa/000042_no_incorporacion_ct.up.sql" 'ya instalada'; ok 'doble UP Bolsa 000042 rechazado'
run <"$bolsa/000042_no_incorporacion_ct.down.sql"
igual "$(escalar "$guardado")" "$inicial_guardado" 'DOWN Bolsa 000042 restaura exactamente el guardado y su ACL'
run <"$bolsa/000042_no_incorporacion_ct.up.sql"
igual "$(escalar "$guardado")" "$con_b42" 'UP tras DOWN reproduce Bolsa 000042'
igual "$(escalar "SELECT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb)','EXECUTE')
  AND NOT has_function_privilege('public','vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb)','EXECUTE')
  AND NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.no_incorporacion_registrada_b42(text)','EXECUTE')
  AND NOT has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.no_incorporacion_bolsa','SELECT')")" t 'bandeja solo por su función; tabla y auxiliar cerrados'

echo '== Transacciones (dobles explícitos de las fachadas AD3)'
b='expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
salida=$(docker exec -i "$nombre" psql -X -q -At -v ON_ERROR_STOP=1 -v exp_a="$b" -U postgres -d postgres <"$pruebas/ct115_ct116_fixture_pg18.sql" 2>&1) || { echo "$salida" >&2; exit 1; }
prueba 'fixture no incorporación OK' "$pruebas/ct124_no_incorporacion_fixture.sql"
if [[ ${VEC_CT124_GO:-} == 1 ]]; then
  echo '== Contrato Go↔SQL de CT y de Bolsa (dobles explícitos de las fachadas AD3)'
  # La consulta del seguimiento lee también las propuestas (CT128).
  run <"$ct/000128_propuesta_sucesor_no_incorporacion.up.sql"
  puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
  a='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
  org=$(escalar "SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref='$a' AND version=7")
  evento="$datos-evento.json"
  (cd "$repo" && VEC_CT124NI_PG_DSN="postgres://vec_ct115_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT124NI_ORG="$org" \
    VEC_CT124NI_EXP="$a" VEC_CT124NI_EVENTO="$evento" TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestNoIncorporacionPostgreSQLContratoGoSQL -v \
    ./internal/modules/contrataciontemporal/adapters/postgres/ 2>&1 | tail -8)
  (cd "$repo" && VEC_B42_PG_DSN="postgres://vec_b42_runtime@127.0.0.1:$puerto/postgres?sslmode=disable" VEC_CT124NI_EVENTO="$evento" \
    TMPDIR=${TMPDIR:-/dev/shm} go test -count=1 -run TestBandejaNoIncorporacionesPostgreSQLContratoGoSQL -v ./internal/modules/bolsa/adapters/postgres/ 2>&1 | tail -8)
  rm -f "$evento"
  echo 'ENSAYO GO COMPLETO'
  exit 0
fi
prueba 'cadena no incorporación OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_no_incorporacion_cadena.sql"

echo '== Reinicio de PostgreSQL y repetición de los mismos materiales'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
prueba 'reinicio no incorporación OK' "$pruebas/ct124_utilidades.sql" "$pruebas/ct124_no_incorporacion_reinicio.sql"
falla_con "$ct/000124_incorporacion_acreditada.down.sql" 'no admitido con historia'; ok 'DOWN CT124 rechazado con historia'
falla_con "$bolsa/000042_no_incorporacion_ct.down.sql" 'reversión denegada'; ok 'DOWN Bolsa 000042 rechazado con historia'
falla_con "$ad3/000088_consumidor_incorporacion_acreditada_ct.down.sql" 'DOWN no admitido'; ok 'DOWN AD3-88 rechazado con CT124 instalada'
echo 'Cadena de no incorporación verificada'

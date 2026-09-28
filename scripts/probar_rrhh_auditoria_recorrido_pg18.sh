#!/usr/bin/env bash
# Recorrido de Auditoría RRHH sobre una copia SINTÉTICA en PG18 desechable.
# Uso: script GLOBALS_SQL BASE_PG_DUMP EXPEDIENTE_CT PARTICIPACION_BOLSA
# La preimagen debe tener AD3-90, CT131 y Bolsa47, sin AD3-91/CT132/Bolsa48.
# El operador aporta un hook local que arranca/detiene la aplicación aislada:
# VEC_AUDITORIA_APP_HOOK=/ruta/privada/script (recibe start|stop).
# El hook recibe VEC_AUDITORIA_PG_SOCKET y configura perfiles/certificados
# sintéticos; VEC_AUDITORIA_HTTP_* se describen en rrhh_auditoria/recorrido_http.py.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta pg_dumpall --globals-only sintético}
volcado=${2:?falta pg_dump -Fc sintético}
exp_ct=${3:?falta expediente CT sintético}
exp_bolsa=${4:?falta participación Bolsa sintética}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan volcados sintéticos' >&2; exit 2; }
[[ ${VEC_AUDITORIA_DATOS_SINTETICOS:-} == 1 ]] || { echo 'Indicar VEC_AUDITORIA_DATOS_SINTETICOS=1 tras verificar el volcado' >&2; exit 2; }
[[ -x ${VEC_AUDITORIA_APP_HOOK:-} ]] || { echo 'Falta hook ejecutable de aplicación aislada' >&2; exit 2; }
[[ $exp_ct =~ ^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$ && $exp_bolsa =~ ^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$ ]] || { echo 'Referencias sintéticas inválidas' >&2; exit 2; }
for herramienta in docker python3 go; do
  command -v "$herramienta" >/dev/null || { echo "Falta $herramienta" >&2; exit 2; }
done

nombre="vec-auditoria-pg18-$$"
datos="/dev/shm/$nombre"
socket="/dev/shm/$nombre-socket"
evidencia="/dev/shm/$nombre-evidencia.json"
cache_go="/dev/shm/$nombre-go-cache"
app_iniciada=0
limpiar() {
  if (( app_iniciada )); then "$VEC_AUDITORIA_APP_HOOK" stop >/dev/null 2>&1 || true; fi
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  # PG18 crea /var/lib/postgresql/18/docker como root dentro del bind mount.
  # Un rm del usuario deja basura en /dev/shm; limpiar desde la misma imagen.
  docker run --rm --network none -v /dev/shm:/limpiar postgres:18.4 \
    rm -rf "/limpiar/$nombre" "/limpiar/$nombre-socket" >/dev/null 2>&1 || true
  rm -rf -- "$datos" "$socket" "$evidencia" "$cache_go" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos" "$socket"
chmod 1777 "$socket"
export VEC_AUDITORIA_PG_SOCKET="$socket"
export VEC_AUDITORIA_EXP_CT="$exp_ct" VEC_AUDITORIA_EXP_BOLSA="$exp_bolsa"
export VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT="$exp_ct"
export VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA="$exp_bolsa"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" -v "$socket:/var/run/postgresql" postgres:18.4 >/dev/null
esperar() {
  for _ in $(seq 1 180); do
    if docker exec "$nombre" pg_isready -q -U postgres -d postgres; then return 0; fi
    sleep 0.5
  done
  echo 'PG18 no arrancó' >&2; return 1
}
esperar
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]] || { echo 'Volumen anónimo inesperado' >&2; exit 1; }
psql() { docker exec -i "$nombre" psql -X -q -A -t -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
scalar() { printf '%s\n' "$1" | psql "${@:2}"; }
igual() { [[ $1 == "$2" ]] || { printf 'FALLO %s: %s != %s\n' "$3" "$1" "$2" >&2; exit 1; }; printf 'OK %s\n' "$3"; }

[[ $(scalar 'SHOW server_version') == 18.4* ]] || { echo 'Se exige PostgreSQL 18.4' >&2; exit 2; }
echo 'OK PostgreSQL 18.4'
echo 'Restaurando únicamente la copia sintética en PG18 efímero'
# El rol postgres ya lo crea la imagen. Ningún error restante se ignora.
sed -e '/^CREATE ROLE postgres;$/d' -e '/^ALTER ROLE postgres WITH /d' "$globales" | psql >/dev/null
docker exec -i "$nombre" pg_restore --exit-on-error -U postgres -d postgres <"$volcado" >/dev/null

ad3="$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000091_consumidor_consulta_auditoria_rrhh.up.sql"
ct="$repo/deploy/postgresql/contratacion_temporal/migraciones/000132_consulta_auditoria_ct.up.sql"
bolsa="$repo/deploy/postgresql/bolsa_llamamientos/migraciones/000048_consulta_auditoria_participacion.up.sql"
preimagen="SELECT
 to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL
 AND to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NOT NULL
 AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 AND to_regprocedure('vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 AND to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL;"
igual "$(scalar "$preimagen")" t 'preimagen AD3-90/CT131/Bolsa47 sin las tres migraciones'
huella_previa=$(scalar "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)) || ':' || md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure));")
for migracion in "$ad3" "$ct" "$bolsa"; do
  sed 's/^COMMIT;$/ROLLBACK;/' "$migracion" | psql >/dev/null
  case "$migracion" in
    "$ad3") deteccion="SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL;" ;;
    "$ct") deteccion="SELECT to_regprocedure('vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL;" ;;
    "$bolsa") deteccion="SELECT to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL;" ;;
  esac
  igual "$(scalar "$deteccion")" t "ROLLBACK $(basename "$migracion") sin objeto nuevo"
  if [[ $migracion == "$ad3" ]]; then
    igual "$(scalar "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)) || ':' || md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure));")" "$huella_previa" 'ROLLBACK AD3-91 conserva funciones previas'
  fi
  psql <"$migracion" >/dev/null
  igual "$(scalar "$deteccion")" f "UP $(basename "$migracion") detectado"
  echo "OK UP $(basename "$migracion")"
  if psql <"$migracion" >/dev/null 2>&1; then echo "FALLO doble UP $(basename "$migracion")" >&2; exit 1; fi
done
[[ $huella_previa != "$(scalar "SELECT md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)) || ':' || md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure));")" ]] || { echo 'AD3-91 no cambió consumidores' >&2; exit 1; }

acl="SELECT has_function_privilege('vec_contratacion_temporal_consultor_rrhh','vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_consultor_rrhh','vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE');"
igual "$(scalar "$acl")" t 'ACL nominal CT y Bolsa sin suma de perfiles'
historia_ct=$(scalar "SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_ct';" -v "exp_ct=$exp_ct")
historia_bolsa=$(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=:'exp_bolsa') + (SELECT count(*) FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE participacion_ref=:'exp_bolsa');" -v "exp_bolsa=$exp_bolsa")
(( historia_ct >= 2 && historia_bolsa >= 2 )) || { echo 'Se requieren al menos dos hechos sintéticos por fuente para paginar' >&2; exit 2; }
auditoria_ref_cuenta() {
  scalar "SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE efecto_ref IN (:'exp_ct',:'exp_bolsa');" \
    -v "exp_ct=$exp_ct" -v "exp_bolsa=$exp_bolsa"
}
auditoria_pre=$(auditoria_ref_cuenta)

echo 'Ejecutando pruebas Go del manejador, fuentes y composición'
(cd "$repo" && TMPDIR=/dev/shm GOCACHE="$cache_go" GOMAXPROCS=2 go test -count=1 ./internal/vec/auditoria ./internal/modules/contrataciontemporal/adapters/auditoriaconsulta ./internal/modules/bolsa/adapters/auditoriaconsulta ./internal/app/bootstrap)
"$VEC_AUDITORIA_APP_HOOK" start
app_iniciada=1
python3 "$repo/scripts/rrhh_auditoria/recorrido_http.py" before "$evidencia"
auditoria_antes_reinicio=$(auditoria_ref_cuenta)
igual "$auditoria_antes_reinicio" "$((auditoria_pre + 6))" 'seis lecturas autorizadas dejan seis auditorías V3 exactas'
"$VEC_AUDITORIA_APP_HOOK" stop
app_iniciada=0
docker restart "$nombre" >/dev/null
esperar
igual "$(scalar "$acl")" t 'ACL conservada tras reiniciar PG18'
igual "$(scalar "SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_ct';" -v "exp_ct=$exp_ct")" "$historia_ct" 'historia CT conservada'
igual "$(scalar "SELECT (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=:'exp_bolsa') + (SELECT count(*) FROM vec_bolsa_llamamientos.traza_valor_participacion WHERE participacion_ref=:'exp_bolsa');" -v "exp_bolsa=$exp_bolsa")" "$historia_bolsa" 'historia Bolsa conservada'
igual "$(auditoria_ref_cuenta)" "$auditoria_antes_reinicio" 'auditoría V3 conservada'
"$VEC_AUDITORIA_APP_HOOK" start
app_iniciada=1
python3 "$repo/scripts/rrhh_auditoria/recorrido_http.py" after "$evidencia"
auditoria_final=$(auditoria_ref_cuenta)
igual "$auditoria_final" "$((auditoria_antes_reinicio + 6))" 'seis lecturas tras restart generan auditoría nueva'
echo 'OK GET opciones, POST CT/Bolsa, paginación, 403 ajeno y recuperación con auditoría V3 durable'

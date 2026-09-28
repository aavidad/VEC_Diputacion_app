#!/usr/bin/env bash
# PostgreSQL 18 efímero: sintaxis, recibos CT56, replay, conflicto, concurrencia
# y reinicio. Fixture de V3 sintético, no equivale al ensayo de preimagen real.
set -Eeuo pipefail
raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4}
contenedor="vec-ct138-${PPID}-${RANDOM}"
datos="/dev/shm/$contenedor"
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run --rm -v /dev/shm:/limpiar --entrypoint rm "$imagen" -rf "/limpiar/$contenedor" >/dev/null 2>&1 || true
}
if [[ ${VEC_CT138_CONSERVAR:-} != 1 ]]; then trap limpiar EXIT; fi
mkdir -p "$datos"
docker run -d --rm --network none --name "$contenedor" \
  -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" \
  -v "$raiz:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 90); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
psql_login() { docker exec -i "$contenedor" psql -X -q -At -v ON_ERROR_STOP=1 -U vec_ct138_login -d postgres "$@"; }
ct=/repo/deploy/postgresql/contratacion_temporal
pruebas=$ct/pruebas_sql
psql_super -f "$pruebas/ct138_fixture_minimo_pg18.sql" >/dev/null
psql_super -f "$ct/migraciones/000056_respuesta_recibida_rrhh.up.sql" >/dev/null
psql_super -f "$pruebas/ct138_preimagen_sintetica_pg18.sql" >/dev/null
psql_super -f "$pruebas/ct138_sondas_minimas_pg18.sql" >/dev/null
firma="vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
huella_previa=$(psql_super -At -c "SELECT md5(pg_get_functiondef('$firma'::regprocedure))")
consulta() {
  local llamamiento=$1 comunicacion=$2 clave=$3 huella=$4 version=$5
  psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct138_registrar_sintetico(vec_contratacion_temporal.ct138_material_sintetico('$llamamiento','$comunicacion','$clave',repeat('$huella',64)),$version)::text; COMMIT;" | grep '^{'
}
legacy=$(consulta llamamiento:legado comunicacion:legada 11111111-1111-4111-8111-111111111111 a 1)
[[ -n $legacy ]] || { echo 'CT56 no creó recibo'; exit 1; }
psql_super -f "$ct/migraciones/000138_respuesta_recibida_semantica.up.sql" >/dev/null
[[ $(psql_super -At -c "SELECT md5(pg_get_functiondef('$firma'::regprocedure))") == "$huella_previa" ]] \
  || { echo 'CT138 alteró CT56'; exit 1; }
[[ $(psql_super -At -c "SELECT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('public','vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] \
  || { echo 'ACL CT138 incorrecta'; exit 1; }
replay_legacy=$(consulta llamamiento:legado comunicacion:legada 99999999-9999-4999-8999-999999999999 a 2)
nuevo=$(consulta llamamiento:nuevo comunicacion:nueva '' b 2)
replay_nuevo=$(consulta llamamiento:nuevo comunicacion:nueva 88888888-8888-4888-8888-888888888888 b 2)
rechazar_cruce_identidad() {
  local llamamiento=$1 comunicacion=$2 clave=$3 huella=$4 actor=$5 perfil=$6
  if psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct138_registrar_sintetico(vec_contratacion_temporal.ct138_material_sintetico('$llamamiento','$comunicacion','$clave',repeat('$huella',64)),2,false,'$actor','$perfil'); COMMIT;" >/tmp/vec-ct138-identidad-$$ 2>&1; then
    echo 'CT138 reveló un recibo a otra identidad'; exit 1
  fi
  grep -q 'replay de respuesta denegado' /tmp/vec-ct138-identidad-$$
  rm -f /tmp/vec-ct138-identidad-$$
}
rechazar_cruce_identidad llamamiento:legado comunicacion:legada '' a actor:ajeno perfil:sintetico
rechazar_cruce_identidad llamamiento:legado comunicacion:legada '' a actor:sintetico perfil:ajeno
rechazar_cruce_identidad llamamiento:nuevo comunicacion:nueva '' b actor:ajeno perfil:sintetico
rechazar_cruce_identidad llamamiento:nuevo comunicacion:nueva '' b actor:sintetico perfil:ajeno
python3 - "$legacy" "$replay_legacy" "$nuevo" "$replay_nuevo" <<'PY'
import json,sys
a,b,c,d=map(json.loads,sys.argv[1:])
for primero,replay in ((a,b),(c,d)):
    assert primero['Estado']=='registrada_por_rrhh'
    assert replay['Estado']=='replay_registrada_por_rrhh'
    assert {k:v for k,v in primero.items() if k!='Estado'}=={k:v for k,v in replay.items() if k!='Estado'}
assert c['Solicitud']['ClaveIdempotencia']!=''
assert c['Solicitud']['ClaveIdempotencia']!='88888888-8888-4888-8888-888888888888'
PY
if consulta llamamiento:nuevo comunicacion:nueva '' c 2 >/tmp/vec-ct138-conflicto-$$ 2>&1; then
  echo 'CT138 admitió contenido distinto'; exit 1
fi
grep -q 'contenido de respuesta divergente' /tmp/vec-ct138-conflicto-$$
rm -f /tmp/vec-ct138-conflicto-$$
if consulta llamamiento:nuevo comunicacion:nueva-otra '' b 2 >/tmp/vec-ct138-conflicto-$$ 2>&1; then
  echo 'CT138 admitió otra comunicación para el mismo candidato'; exit 1
fi
grep -q 'contenido de respuesta divergente' /tmp/vec-ct138-conflicto-$$
rm -f /tmp/vec-ct138-conflicto-$$
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh') == 2 ]]
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.historia_respuesta_recibida_rrhh') == 2 ]]
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.outbox_respuesta_recibida_rrhh') == 2 ]]
consulta llamamiento:concurrente comunicacion:concurrente '' d 2 >/tmp/vec-ct138-uno-$$ 2>&1 & p1=$!
consulta llamamiento:concurrente comunicacion:concurrente '' d 2 >/tmp/vec-ct138-dos-$$ 2>&1 & p2=$!
estado1=0
estado2=0
wait "$p1" || estado1=$?
wait "$p2" || estado2=$?
# El segundo intento serializable puede recibir 40001. Ningún error ajeno
# se toma por éxito, ni se atribuye un replay sin leer su recibo.
concurrente=$(consulta llamamiento:concurrente comunicacion:concurrente '' d 2)
python3 - "$estado1" /tmp/vec-ct138-uno-$$ "$estado2" /tmp/vec-ct138-dos-$$ "$concurrente" <<'PY'
import json,re,sys
salidas=[]
for estado,ruta in ((int(sys.argv[1]),sys.argv[2]),(int(sys.argv[3]),sys.argv[4])):
    contenido=open(ruta,encoding='utf-8').read()
    if estado==0:
        filas=[linea for linea in contenido.splitlines() if linea.startswith('{')]
        assert len(filas)==1, f'resultado concurrente ausente o duplicado: {contenido}'
        salidas.append(json.loads(filas[0]))
    else:
        assert re.search(r'serialización de respuesta|could not serialize access',contenido), contenido
assert sum(r['Estado']=='registrada_por_rrhh' for r in salidas)==1, salidas
assert all(r['Estado'] in ('registrada_por_rrhh','replay_registrada_por_rrhh') for r in salidas)
replay=json.loads(sys.argv[5])
assert replay['Estado']=='replay_registrada_por_rrhh'
for recibo in salidas:
    assert {k:v for k,v in recibo.items() if k!='Estado'}=={k:v for k,v in replay.items() if k!='Estado'}
PY
rm -f /tmp/vec-ct138-uno-$$ /tmp/vec-ct138-dos-$$
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh') == 3 ]]
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.historia_respuesta_recibida_rrhh') == 3 ]]
[[ $(psql_super -At -c 'SELECT count(*) FROM vec_contratacion_temporal.outbox_respuesta_recibida_rrhh') == 3 ]]
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 90); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 1
done
[[ $(consulta llamamiento:nuevo comunicacion:nueva '' b 2) == "$replay_nuevo" ]]
if psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct138_registrar_sintetico(vec_contratacion_temporal.ct138_material_sintetico('llamamiento:nuevo','comunicacion:nueva','',repeat('b',64)),2,true); COMMIT;" >/tmp/vec-ct138-denegado-$$ 2>&1; then
  echo 'CT138 admitió decisión ajena'; exit 1
fi
grep -q 'autorización de respuesta divergente' /tmp/vec-ct138-denegado-$$
rm -f /tmp/vec-ct138-denegado-$$
printf 'CT138 sintético PostgreSQL 18: CT56 intacta, ACL, replay, identidad, conflicto, concurrencia, reinicio y denegación OK\n'

#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de AD3-61 (fachadas D7b/D7c) sobre la
# PREIMAGEN REAL del núcleo hasta AD3-52: una base VEC restaurada, no un stub.
# Contenedores sin red, borrados al salir.
#
# Uso: dietas_d7bc_ad3_000061_pg18.sh BASE.dump ROLES.sql
#   BASE.dump  pg_dump -Fc de una base VEC con AD3-52 instalada, sin AD3-53,
#              AD3-59 ni AD3-61 (clon sintético; nunca datos reales).
#   ROLES.sql  pg_dumpall --roles-only de la misma instancia; las contraseñas
#              se descartan antes de entrar al contenedor.
#
# Recorre dos órdenes, siempre en serie: 53→59→Personal 12/13→61 y
# 59→Personal 12/13→61→53. En cada uno comprueba: Personal 14/15 se cierran
# sin AD3-61; ROLLBACK de AD3-61 sin rastro; COMMIT; segunda aplicación
# rechazada sin cambios; DOWN rechazado; el consumo CT real repetido es
# idéntico antes y después y no crea filas; Personal 14/15 se instalan sobre
# las fachadas reales; las cinco operaciones D7b/D7c superan guarda,
# prevalidación y ligadura con el login exacto y se detienen en la clave de
# capacidad; un login con dos grupos o de otro módulo se rechaza en la guarda;
# y las funciones públicas de Personal llegan al núcleo por la fachada real.
set -Eeuo pipefail
base=${1:?falta BASE.dump (pg_dump -Fc con AD3-52)}
roles=${2:?falta ROLES.sql (pg_dumpall --roles-only)}
[[ -s $base && -s $roles ]] || { echo 'Base o roles vacíos' >&2; exit 2; }
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
pg=$repo/deploy/postgresql
mig=$pg/autorizacion_atestada_v3/migraciones
m53=$mig/000053_consumidores_cronos_empleado.up.sql
m59=$mig/000059_consumidor_documento_dietas.up.sql
m61=$mig/000061_consumidor_competencias_rectificacion_dietas.up.sql
m61down=$mig/000061_consumidor_competencias_rectificacion_dietas.down.sql
p12=$pg/personal/migraciones/000012_asignacion_dietas.up.sql
p13=$pg/personal/migraciones/000013_auditoria_frontera_asignacion_dietas.up.sql
p14=$pg/personal/migraciones/000014_competencias_asignacion_dietas.up.sql
p15=$pg/personal/migraciones/000015_solicitud_rectificacion_dietas.up.sql
tmp=$(mktemp -d)
contenedores=()
limpiar() {
  for c in "${contenedores[@]}"; do docker rm -f "$c" >/dev/null 2>&1 || true; done
  rm -rf -- "$tmp"
}
trap limpiar EXIT INT TERM
sed -E "s/ PASSWORD '[^']*'//; s/ PASSWORD [^ ;]+//" "$roles" | grep -vE '^(CREATE|ALTER) ROLE postgres( |;)' > "$tmp/roles.sql"
if grep -qi 'password' "$tmp/roles.sql"; then echo 'No se pudieron retirar las contraseñas' >&2; exit 2; fi

fallo() { echo "FALLO [$orden]: $*" >&2; exit 1; }
ok() { echo "OK [$orden] $*"; }
sql() { docker exec -i "$C" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
val() { docker exec "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
firma10='(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
firma() {
  val "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef(c.oid))||
   (SELECT md5(string_agg(p.oid::regprocedure::text||coalesce(p.proacl::text,'')||md5(p.prosrc),'|' ORDER BY p.oid::regprocedure::text))
      FROM pg_proc p WHERE p.pronamespace::regnamespace::text IN ('vec_autorizacion_atestada_v3','vec_personal'))||
   (SELECT count(*) FROM pg_class k WHERE k.relnamespace='vec_personal'::regnamespace)
   FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
}
consumos() {
  val "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)"
}
# Ejecuta una sentencia como el login indicado y devuelve la fila o el error.
como() {
  { docker exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U "$1" -d postgres 2>&1 || true; } <<SQL | sed -n 's/^.*ERROR: *//p;/|/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
$2;
COMMIT;
SQL
}
llamar() { como "$1" "SELECT * FROM prueba59.consumir_$2('$3','$4')"; }
esperar() {
  for _ in $(seq 1 120); do
    if docker exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.5
      docker exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1 && return 0
    fi
    sleep 0.3
  done
  fallo 'PostgreSQL no disponible'
}
aplicar() { sql -o /dev/null < "$1" || fallo "$2"; ok "$2 aplicada"; }
aplicar53() {
  sql -o /dev/null -c 'CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOBYPASSRLS; CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOBYPASSRLS'
  aplicar "$m53" AD3-53
}

campos_comp='["asignacion_ref","auditoria_ad3_ref","cardinalidad","competencias","consultada_en","consumo_huella_sha256","decision_ref","efecto_ref","recibo_ref","relacion_ref","rol","unidad_ref","version","vigente_desde"]'
campos_sol='["asignacion_ref","auditoria_ad3_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado","recibo_ref","registrada_en","solicitud_ref","version_origen"]'
campos_lista='["administrativo_persona_ref","asignacion_actual","asignacion_ref","auditoria_ad3_ref","campos_a_revisar","cardinalidad","centro_ref","consultada_en","consumo_huella_sha256","decision_ref","detalle_solicitado","efecto_ref","empleado_ref","estado","fecha_referencia","grupo_dieta","motivo_revision","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","solicitud_ref","solicitudes","unidad_ref","version","version_origen","vigente_desde"]'
fc=registrar_y_consumir_competencias_asignacion_dietas_v3_atestada
fr=registrar_y_consumir_rectificacion_dietas_v3_atestada
# caso|fachada|operación|audiencia|recurso|finalidad|campos
casos=(
 "c1|$fc|personal.asignacion_dietas.competencias_consultar|vec_personal.asignacion_dietas.competencias.v1|asignacion_dietas_competencias|tramitar_dietas_asignadas|$campos_comp"
 "r1|$fr|personal.asignacion_dietas.rectificacion.solicitar|vec_personal.asignacion_dietas.rectificacion.solicitar.v1|rectificacion_asignacion_dietas|solicitar_rectificacion_dietas|$campos_sol"
 "r2|$fr|personal.asignacion_dietas.rectificacion.propia.consultar|vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1|rectificacion_asignacion_dietas|consultar_rectificacion_dietas_propia|$campos_sol"
 "r3|$fr|personal.asignacion_dietas.rectificacion.resolver|vec_personal.asignacion_dietas.rectificacion.resolver.v1|rectificacion_asignacion_dietas|resolver_rectificacion_dietas|$campos_sol"
 "r4|$fr|personal.asignacion_dietas.rectificacion.competente.consultar|vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1|rectificaciones_competentes_dietas|consultar_rectificaciones_dietas_competentes|$campos_lista"
)

ensayar() {
  orden=$1
  C="vec-ad3-61-${orden}-$$"
  contenedores+=("$C")
  docker run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
  esperar
  [[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'
  sql -o /dev/null < "$tmp/roles.sql" || fallo 'roles'
  docker exec -i "$C" pg_restore -U postgres -d postgres --exit-on-error --single-transaction < "$base" || fallo 'restauración'
  [[ $(val "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada$firma10') IS NOT NULL
     AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada$firma10') IS NULL
     AND to_regprocedure('vec_autorizacion_atestada_v3.$fc$firma10') IS NULL
     AND to_regprocedure('vec_autorizacion_atestada_v3.$fr$firma10') IS NULL
     AND to_regrole('vec_personal_d7_ejecutor') IS NULL") == t ]] \
    || fallo 'la base no es la preimagen AD3-52 sin AD3-53/59/61'
  ok 'preimagen real hasta AD3-52 restaurada'

  sql -o /dev/null < "$dir/dietas_documento_ad3_000059_sondas.sql" || fallo 'sondas AD3-59'
  sql -o /dev/null < "$dir/dietas_d7bc_ad3_000061_sondas.sql" || fallo 'sondas AD3-61'
  sql -o /dev/null <<'SQL' || fallo 'login CT'
CREATE ROLE vec_prueba61_ct LOGIN INHERIT NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_prueba61_ct WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba61_ct;
GRANT EXECUTE ON FUNCTION prueba59.consumir_ct(text,text) TO vec_prueba61_ct;
SQL
  local ct_antes ct_despues filas antes despues
  ct_antes=$(llamar vec_prueba61_ct ct registrar_y_consumir_decision_v3_atestada ct_alta)
  [[ $ct_antes == decision:*'|f' ]] || fallo "consumo CT previo: $ct_antes"
  filas=$(consumos)
  ok "consumo CT antes: ${ct_antes%%|*}, consumo_nuevo=f"

  [[ $orden == 53-59-61 ]] && aplicar53
  aplicar "$m59" AD3-59
  aplicar "$p12" 'Personal 000012'
  aplicar "$p13" 'Personal 000013'

  # Sin AD3-61, D7b y D7c fallan cerradas antes de crear nada.
  for f in "$p14" "$p15"; do
    antes=$(firma)
    if sql -o /dev/null < "$f" 2>/dev/null; then fallo "$(basename "$f") instaló sin AD3-61"; fi
    [[ $(firma) == "$antes" ]] || fallo "$(basename "$f") dejó objetos"
  done
  ok 'Personal 000014/000015 cerradas sin AD3-61'

  antes=$(firma)
  sed '$s/^COMMIT;$/ROLLBACK;/' "$m61" | sql -o /dev/null || fallo 'AD3-61 en ROLLBACK'
  [[ $(firma) == "$antes" ]] || fallo 'ROLLBACK de AD3-61 dejó rastro'
  ok 'ROLLBACK de AD3-61 sin rastro en núcleo, audiencias, funciones ni ACL'
  aplicar "$m61" AD3-61
  despues=$(firma)
  if sql -o /dev/null < "$m61" 2>/dev/null; then fallo 'segunda aplicación de AD3-61 aceptada'; fi
  [[ $(firma) == "$despues" ]] || fallo 'la segunda aplicación alteró el estado'
  if sql -o /dev/null < "$m61down" 2>/dev/null; then fallo 'DOWN de AD3-61 admitido'; fi
  [[ $(firma) == "$despues" ]] || fallo 'el DOWN alteró el estado'
  ok 'segunda aplicación y DOWN de AD3-61 rechazados sin cambios'
  [[ $orden == 59-61-53 ]] && aplicar53

  local def p
  def=$(val "SELECT pg_get_functiondef('$nucleo'::regprocedure)")
  for p in competencias_asignacion_dietas_personal rectificacion_dietas_personal \
           documento_dietas_mutacion consulta_asignacion_dietas_personal correccion_asignacion_dietas_personal \
           alta_inicial_asignacion_dietas_personal cronos_marcaje_propio cronos_saldo_propio \
           importacion_organizacion_historica_personal; do
    [[ $(grep -c "IS DISTINCT FROM '$p'" <<<"$def") == 1 ]] || fallo "exclusión de $p no única"
  done
  [[ $(grep -c "p_perfil_mutacion IN ('competencias_asignacion_dietas_personal','rectificacion_dietas_personal')" <<<"$def") == 1 ]] \
    || fallo 'guarda D7b/D7c no única'
  grep -q "'vec_personal_d7_ejecutor'::regrole" <<<"$def" && fallo 'literal ::regrole del grupo D7 en el núcleo'
  grep -q "'vec_cronos_v1_ejecutor'" <<<"$def" || fallo 'guarda Cronos perdida'
  ok 'núcleo con exclusiones únicas de D7b/D7c, D7, Dietas, Cronos y AD3-52; grupo D7 cotejado por nombre'

  ct_despues=$(llamar vec_prueba61_ct ct registrar_y_consumir_decision_v3_atestada ct_alta)
  [[ $ct_despues == "$ct_antes" ]] || fallo "CT tras AD3-61 distinto: $ct_despues"
  [[ $(consumos) == "$filas" ]] || fallo 'el consumo CT repetido creó filas'
  ok 'consumo CT tras AD3-61 idéntico y sin filas nuevas'

  aplicar "$p14" 'Personal 000014 sobre la fachada real'
  aplicar "$p15" 'Personal 000015 sobre la fachada real'
  for p in "$fc" "$fr"; do
    [[ $(val "SELECT pg_get_userbyid(p.proowner)||'|'||p.prosecdef||'|'||(strpos(p.prosrc,'consumir_decision_mutacion_v3_interna')>0)
       FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.$p$firma10'::regprocedure") == 'vec_autorizacion_atestada_v3_propietario|true|true' ]] \
      || fallo "$p no es la fachada real AD3-61"
    [[ $(val "SELECT string_agg(pg_get_userbyid(a.grantee)||':'||a.is_grantable,',') FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid='vec_autorizacion_atestada_v3.$p$firma10'::regprocedure AND a.grantee<>p.proowner") == 'vec_personal_propietario:false' ]] \
      || fallo "ACL de $p"
  done
  ok 'fachadas reales: propietario AD3, SECURITY DEFINER, sin PUBLIC y solo vec_personal_propietario'

  sql -o /dev/null <<'SQL' || fallo 'logins'
CREATE ROLE prueba61_otro NOLOGIN;
CREATE ROLE vec_prueba61_d7 LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba61_d7 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba61_doble LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba61_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_dietas_ejecutor TO vec_prueba61_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba61_d7_otro LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba61_d7_otro WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT prueba61_otro TO vec_prueba61_d7_otro WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba61_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba61_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59, prueba61 TO vec_prueba61_d7, vec_prueba61_doble, vec_prueba61_d7_otro, vec_prueba61_dietas;
GRANT SELECT ON prueba59.material, prueba61.material_personal TO vec_prueba61_d7, vec_prueba61_d7_otro;
GRANT EXECUTE ON FUNCTION prueba59.consumir_personal(text,text) TO vec_prueba61_d7, vec_prueba61_doble, vec_prueba61_d7_otro, vec_prueba61_dietas;
GRANT EXECUTE ON FUNCTION prueba61.competencias(text), prueba61.competentes(text) TO vec_prueba61_d7, vec_prueba61_d7_otro;
SQL

  local linea caso fachada op aud tipo fin campos r
  for linea in "${casos[@]}"; do
    IFS='|' read -r caso fachada op aud tipo fin campos <<<"$linea"
    sql -o /dev/null -c "SELECT prueba61.preparar('$caso','$op','$aud','$tipo','$fin','$campos'::jsonb,
       'prueba61:$caso',encode(sha256(convert_to('prueba61:$caso','UTF8')),'hex'))" || fallo "material $caso"
    r=$(llamar vec_prueba61_d7 personal "$fachada" "$caso")
    [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "$op con login exacto: $r"
    r=$(llamar vec_prueba61_doble personal "$fachada" "$caso")
    [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "$op con dos grupos: $r"
    r=$(llamar vec_prueba61_d7_otro personal "$fachada" "$caso")
    [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "$op con grupo D7 y otro grupo: $r"
    r=$(llamar vec_prueba61_dietas personal "$fachada" "$caso")
    [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "$op con login de Dietas: $r"
    ok "$op: login exacto supera guarda y ligadura; dos grupos y login ajeno rechazados"
  done

  # La fachada cierra antes del núcleo lo que no es su contrato exacto.
  sql -o /dev/null -c "SELECT prueba61.preparar('x1','personal.asignacion_dietas.competencias_consultar',
     'vec_personal.asignacion_dietas.competencias.v1','asignacion_dietas_competencias','otra_finalidad',
     '$campos_comp'::jsonb,'prueba61:x1',encode(sha256(convert_to('prueba61:x1','UTF8')),'hex'))" || fallo 'material x1'
  r=$(llamar vec_prueba61_d7 personal "$fc" x1)
  [[ $r == 'AD3-61: competencias Personal denegadas' ]] || fallo "finalidad ajena aceptada: $r"
  r=$(llamar vec_prueba61_d7 personal "$fr" c1)
  [[ $r == 'AD3-61: operación de rectificación denegada' ]] || fallo "competencias por la fachada D7c: $r"
  r=$(llamar vec_prueba61_d7 personal "$fc" r4)
  [[ $r == 'AD3-61: competencias Personal denegadas' ]] || fallo "lista D7c por la fachada D7b: $r"
  sql -o /dev/null -c "SELECT prueba61.preparar('x2','personal.asignacion_dietas.rectificacion.solicitar',
     'vec_personal.asignacion_dietas.rectificacion.solicitar.v1','rectificacion_asignacion_dietas',
     'solicitar_rectificacion_dietas','$campos_lista'::jsonb,'prueba61:x2',
     encode(sha256(convert_to('prueba61:x2','UTF8')),'hex'))" || fallo 'material x2'
  r=$(llamar vec_prueba61_d7 personal "$fr" x2)
  [[ $r == 'AD3-61: rectificación Personal denegada' ]] || fallo "campos ampliados aceptados: $r"
  ok 'finalidad, operación cruzada y campos ampliados se deniegan en la fachada'

  # Extremo a extremo: función pública de Personal → fachada real → núcleo.
  sql -o /dev/null -c "SELECT prueba61.preparar_lista('pc','vec.personal.asignacion-dietas.competencias.v1',
     'personal.asignacion_dietas.competencias_consultar','vec_personal.asignacion_dietas.competencias.v1',
     'asignacion_dietas_competencias','tramitar_dietas_asignadas','$campos_comp'::jsonb)" || fallo 'material pc'
  sql -o /dev/null -c "SELECT prueba61.preparar_lista('pr','vec.personal.rectificaciones-dietas.competentes.v1',
     'personal.asignacion_dietas.rectificacion.competente.consultar','vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1',
     'rectificaciones_competentes_dietas','consultar_rectificaciones_dietas_competentes','$campos_lista'::jsonb)" || fallo 'material pr'
  r=$(como vec_prueba61_d7 "SELECT prueba61.competencias('pc')")
  [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "Personal 000014 → AD3-61: $r"
  r=$(como vec_prueba61_d7 "SELECT prueba61.competentes('pr')")
  [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "Personal 000015 → AD3-61: $r"
  r=$(como vec_prueba61_d7_otro "SELECT prueba61.competentes('pr')")
  [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "Personal 000015 con grupo adicional: $r"
  ok 'Personal 000014/000015 llegan al núcleo por la fachada real; grupo adicional rechazado en el núcleo'

  [[ $(consumos) == "$filas" ]] || fallo 'las sondas crearon consumos'
  ok 'ninguna sonda consumió'
}

ensayar 53-59-61
ensayar 59-61-53
echo 'PG18.4: AD3-61 sobre preimagen real hasta AD3-52, en serie con AD3-53/59 y Personal 12-15: ROLLBACK, COMMIT, repetición, DOWN, CT, logins y Personal 14/15 sobre fachadas reales verificados.'

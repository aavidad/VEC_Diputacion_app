#!/usr/bin/env bash
# F1-10: carrera real entre una rectificación de la comprobación y una reserva.
# SOLO en una copia desechable: confirma datos ficticios (b140_preparar_caso.sql)
# para que dos sesiones los vean. Nunca contra una base compartida.
# Uso: PSQL="docker exec -i <contenedor> psql -X -U postgres -d postgres" \
#      B140_COPIA_DESECHABLE=si ./b140_carrera_rectificacion_reserva.sh
set -euo pipefail
: "${PSQL:?Falta PSQL con el psql de superusuario de la copia}"
[ "${B140_COPIA_DESECHABLE:-}" = si ] || { echo 'Exige B140_COPIA_DESECHABLE=si' >&2; exit 3; }
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
fallos=0

origen_sql="SELECT jsonb_build_object(
 'organizacion_ref',c->>'organizacion_ref','expediente_ref',c->>'expediente_ref',
 'necesidad_ref',c->>'necesidad_ref','unidad_ref',c->>'unidad_ref',
 'categoria_ref',c->>'categoria_ref','version_expediente',c->'version_expediente',
 'acta_ref',b.acta_ref,'bolsa_ref',b.bolsa_ref,'version_bolsa',b.version_bolsa,
 'version_instantanea',b.version_instantanea,'version_politica',b.version_politica,
 'criterios_ofrecimiento',b.criterios_ofrecimiento,
 'orden_seleccionado',(SELECT min((x->>'orden_acta')::bigint) FROM jsonb_array_elements(b.entradas) x
   WHERE x->>'orden_vigente'='1'),
 'lectura',jsonb_build_object('criterios_ofrecimiento',b.criterios_ofrecimiento,'entradas',
   (SELECT jsonb_agg(jsonb_set(x.valor,'{candidato_ref}',to_jsonb(coalesce(x.valor->>'candidato_ref','')))
    ORDER BY x.n) FROM jsonb_array_elements(b.entradas) WITH ORDINALITY x(valor,n))))
FROM (SELECT :'ctx'::jsonb AS c) k,
LATERAL vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
 c->>'categoria_ref',c->>'modalidad_ofrecida',c->>'causa_ofrecida',c) b;"

leer_origen() { # $1 contexto -> origen actual
  printf '%s\n' "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;" \
    "SET LOCAL ROLE vec_bolsa_llamamientos_propietario;" "$origen_sql" "COMMIT;" |
    $PSQL -v ON_ERROR_STOP=1 -Atq -v ctx="$1"
}

# Una reserva que espera $2 s entre su instantánea y el cotejo.
reserva() { # $1 origen, $2 espera antes, $3 espera después
  printf '%s\n' "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;" \
    "SET LOCAL ROLE vec_contratacion_temporal_propietario;" \
    "SELECT 1 FROM vec_contratacion_temporal.expediente_alta LIMIT 1;" \
    "SELECT pg_sleep($2);" \
    "SELECT vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(:'origen'::jsonb);" \
    "SELECT pg_sleep($3);" "COMMIT;" |
    $PSQL -v ON_ERROR_STOP=1 -Atq -v origen="$1" 2>&1
}

# Registro de un acto con los mismos pasos de control que B133 tras B140.
rectificar() { # $1 contexto, $2 participación, $3 versión nueva, $4 espera inicial
  printf '%s\n' "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;" \
    "SET LOCAL ROLE vec_bolsa_llamamientos_propietario;" \
    "SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version LIMIT 1;" \
    "SELECT pg_sleep($4);" \
    "SELECT set_config('b140.ctx',:'ctx',true),set_config('b140.part',:'part',true),set_config('b140.v',:'v',true);" \
    "DO \$r\$ DECLARE c jsonb:=current_setting('b140.ctx')::jsonb; p text:=current_setting('b140.part');
 v bigint:=current_setting('b140.v')::bigint; b record; po record; e jsonb; ref text; canon jsonb; BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:comprobacion:'||
  jsonb_build_array(c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',p)::text,0));
 INSERT INTO vec_bolsa_llamamientos.version_comprobacion_reserva_v1 VALUES(
  c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',p,v-1)
 ON CONFLICT (organizacion_ref,expediente_ref,necesidad_ref,participacion_ref)
 DO UPDATE SET version=vec_bolsa_llamamientos.version_comprobacion_reserva_v1.version;
 SELECT * INTO STRICT b FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  c->>'categoria_ref',c->>'modalidad_ofrecida',c->>'causa_ofrecida');
 SELECT x INTO STRICT e FROM jsonb_array_elements(b.entradas) x WHERE x->>'participacion_ref'=p;
 SELECT version,huella_sha256 INTO STRICT po FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=b.bolsa_ref ORDER BY version DESC LIMIT 1;
 canon:=jsonb_build_object('ambito_ref',c->>'ambito_ref','bolsa_ref',b.bolsa_ref,
  'categoria_ref',c->>'categoria_ref','causa_ofrecida',c->>'causa_ofrecida',
  'expediente_ref',c->>'expediente_ref','huella_politica_ofertas_sha256',po.huella_sha256,
  'modalidad_ofrecida',c->>'modalidad_ofrecida','necesidad_ref',c->>'necesidad_ref',
  'orden_acta',(e->>'orden_acta')::bigint,'organizacion_ref',c->>'organizacion_ref',
  'participacion_ref',p,'unidad_ref',c->>'unidad_ref','version_bolsa',b.version_bolsa,
  'version_expediente',(c->>'version_expediente')::bigint,
  'version_instantanea',b.version_instantanea,'version_politica_ofertas',po.version);
 ref:='comprobacion:'||encode(sha256(convert_to('b140:carrera:'||v||':'||p,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
  comprobacion_ref,organizacion_ref,expediente_ref,necesidad_ref,bolsa_ref,participacion_ref,
  version,version_esperada,contexto,politica,hechos,resultado,material_sha256,actor_ref,
  perfil_ref,clave_idempotencia,recibo_ref,evento_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(ref,c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',b.bolsa_ref,p,v,v-1,
  canon,'{}','{}','{\"resultado\":\"pendiente\",\"motivo_clave\":\"antecedentes_incompletos\"}',
  repeat('a',64),'per_fixture','prf_fixture','b140:carrera:'||v,'recibo:'||ref,'evento:'||ref,
  'decision:b140:carrera:'||v,'auditoria:b140:carrera:'||v,clock_timestamp());
 UPDATE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g SET version=v
 WHERE g.organizacion_ref=c->>'organizacion_ref' AND g.expediente_ref=c->>'expediente_ref'
  AND g.necesidad_ref=c->>'necesidad_ref' AND g.participacion_ref=p AND g.version=v-1;
 IF NOT FOUND THEN RAISE EXCEPTION 'B140: versión de control divergente' USING ERRCODE='40001'; END IF;
END \$r\$;" "COMMIT;" |
    $PSQL -v ON_ERROR_STOP=1 -Atq -v ctx="$1" -v part="$2" -v v="$3" 2>&1
}

espera() { # $1 nombre, $2 salida, $3 patrón esperado
  if grep -Eq "$3" <<<"$2"; then echo "OK   $1"; else echo "FALLO $1: $2"; fallos=$((fallos+1)); fi
}
sin_error() { # $1 nombre, $2 salida
  if grep -q 'ERROR' <<<"$2"; then echo "FALLO $1: $2"; fallos=$((fallos+1)); else echo "OK   $1"; fi
}

# 0. Datos ficticios confirmados y contexto.
{ echo "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;"
  echo "SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC';"
  cat "$dir/b140_preparar_caso.sql"
  echo "SELECT (SELECT valor FROM pg_temp.b140_caso WHERE nombre='ctx')::text;"
  echo "SELECT (SELECT valor->>'participacion_ref' FROM pg_temp.b140_caso WHERE nombre='pendiente');"
  echo "COMMIT;"; } | $PSQL -v ON_ERROR_STOP=1 -Atq > "$tmp/caso"
ctx=$(sed -n 1p "$tmp/caso"); part=$(sed -n 2p "$tmp/caso")
[ -n "$ctx" ] && [ -n "$part" ] || { echo 'Preparación incompleta' >&2; exit 1; }
origen0=$(leer_origen "$ctx")

# 1. La rectificación se confirma después de la instantánea de la reserva.
reserva "$origen0" 3 0 > "$tmp/a1" & pa=$!
sleep 1; r1=$(rectificar "$ctx" "$part" 1 0 || true); wait $pa || true
sin_error "rectificación confirmada antes del cotejo" "$r1"
espera "la reserva con instantánea anterior se reintenta" "$(cat "$tmp/a1")" 'could not serialize|B140: acto concurrente'

# 2. La reserva coteja primero; la rectificación concurrente debe reintentar.
origen1=$(leer_origen "$ctx")
reserva "$origen1" 0 3 > "$tmp/a2" & pa=$!
sleep 1; r2=$(rectificar "$ctx" "$part" 2 0 || true); wait $pa || true
espera "la reserva que cotejó primero se confirma" "$(cat "$tmp/a2")" '^t$'
sin_error "la reserva que cotejó primero no falla" "$(cat "$tmp/a2")"
espera "la rectificación concurrente se reintenta" "$r2" 'could not serialize|B140: versión de control divergente'
r3=$(rectificar "$ctx" "$part" 2 0 || true)
sin_error "el reintento de la rectificación se confirma" "$r3"

# 3. Una reserva con la lectura anterior a la rectificación se detiene.
r4=$(reserva "$origen1" 0 0 || true)
espera "lectura anterior a la rectificación rechazada" "$r4" 'B140: acto o selección cambiados'

echo "b140_carrera_rectificacion_reserva: $fallos fallos"
[ "$fallos" -eq 0 ]

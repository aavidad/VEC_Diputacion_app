#!/usr/bin/env bash
# Usuarios 000004/000005 (5.08b) en PostgreSQL 18.4 efímero, sin red y con
# una V3 sintética de forma. No acredita COSE, KMS ni SMTP.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-correos-$$"
datos="${TMPDIR:-/tmp}/$contenedor"
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --pull never --network none -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
 rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$contenedor" \
 -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break
 sleep 0.5
done
sleep 2
docker exec "$contenedor" pg_isready -q -U postgres
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
psql_interna() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres "$@"; }
psql_externa() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_externa -d postgres "$@"; }
sql="$raiz/deploy/postgresql/usuarios_vec"
ad3="$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones"
psql_pg < "$sql/roles_up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/preimagen_ad3_sintetica.sql" >/dev/null
psql_pg < "$ad3/000106_consumidor_preferencias_propias.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000001_preferencias_base.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000002_operaciones_preferencias.up.sql" >/dev/null
psql_pg < "$sql/roles_000003_up.sql" >/dev/null
psql_pg < "$sql/migraciones/000003_auditoria_frontera_preferencias.up.sql" >/dev/null
psql_pg < "$ad3/000107_consumidor_correos_usuarios.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000004_correos_propios.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000005_frontera_correos.up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/correos_vector_material.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/correos_operaciones_sinteticas.sql" >/dev/null

# ACL: los ejecutores sólo tienen las seis fachadas; ninguna tabla.
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF has_table_privilege('vec_usuarios_prueba_interna','vec_usuarios.correos_direccion','SELECT')
    OR has_table_privilege('vec_usuarios_prueba_externa','vec_usuarios.correos_envio','UPDATE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.validar_material_correos(text,bytea,bytea,numeric,numeric)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.reservar_envio_correos(text,text,text,text,text,text,timestamptz)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_prueba_externa','vec_usuarios.confirmar_envio_correo_v1(text,text,text,boolean)','EXECUTE')
    OR has_function_privilege('vec_ct_prueba','vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios' AND tablename LIKE 'correos_%' AND roles<>ARRAY['vec_usuarios_propietario']::name[])
    OR position('mis-correos' IN pg_get_functiondef('vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::regprocedure))=0
 THEN RAISE EXCEPTION 'ACL/RLS correos incompatible'; END IF;
END $prueba$;
SQL

A=correo:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
B=correo:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
salida=$(psql_interna -At <<SQL 2>&1 >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO \$prueba\$
DECLARE r jsonb; g jsonb; alta jsonb; i int;
BEGIN
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF (g->>'version')::int<>0 OR g->'correos'<>'[]'::jsonb THEN RAISE EXCEPTION 'GET inicial'; END IF;
 IF public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001','$A','recuperar') IS NOT NULL
 THEN RAISE EXCEPTION 'recuperación ausente devolvió recibo'; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',0,'correo-prueba-v3-falsa-01','$A','recuperar',NULL,true);
  RAISE EXCEPTION 'V3 falsa reveló clave ausente';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 alta:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001','$A');
 IF (alta->>'version')::int<>1 OR alta->>'correo_ref'<>'$A' OR alta->>'replay'<>'false'
    OR jsonb_array_length(alta->'envios')<>1 OR alta->'envios'->0->>'tipo'<>'verificacion'
    OR alta->'envios'->0->>'reserva_ref' !~ '^reserva:[0-9a-f]{32}$'
    OR alta->'envios'->0->'sobre'->>'cifrado_hex'<>repeat('bb',32)
 THEN RAISE EXCEPTION 'alta inicial %',alta; END IF;
 r:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001','$A','recuperar');
 IF r->>'recibo_ref'<>alta->>'recibo_ref' OR r->>'replay'<>'true' OR r->'envios'<>'[]'::jsonb
 THEN RAISE EXCEPTION 'replay alta %',r; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001','$A','recuperar',NULL,false,'interna_corporativa','igualdad-v1','','otra');
  RAISE EXCEPTION 'clave con otra huella aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',1,'correo-prueba-duplicada-01','$B','aplicar',NULL,false,'interna_corporativa','igualdad-v1','$A');
  RAISE EXCEPTION 'dirección duplicada admitida';
 EXCEPTION WHEN SQLSTATE 'P1411' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',0,'correo-prueba-version-vieja','$B');
  RAISE EXCEPTION 'versión antigua admitida';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.activar',1,'correo-prueba-activar-pend','$A');
  RAISE EXCEPTION 'pendiente activado';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verif-mal-01','$A','aplicar',false);
 IF r->>'valido'<>'false' OR (r->>'intentos_restantes')::int<>4 THEN RAISE EXCEPTION 'intento fallido %',r; END IF;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verif-mal-01','$A','recuperar');
 IF r->>'resultado'<>'codigo_invalido' THEN RAISE EXCEPTION 'replay intento fallido %',r; END IF;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verif-mal-01','$A','aplicar',false);
 IF r->'replay'->>'resultado'<>'codigo_invalido' THEN RAISE EXCEPTION 'reintento fallido consumió intento %',r; END IF;
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF (g->'correos'->0->'codigo'->>'intentos_restantes')::int<>4 THEN RAISE EXCEPTION 'intentos en consulta %',g; END IF;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verif-bien-1','$A','aplicar',true);
 IF (r->>'version')::int<>2 OR r->>'replay'<>'false' THEN RAISE EXCEPTION 'verificación %',r; END IF;
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF g->'correos'->0->>'estado'<>'verificado' OR g->'correos'->0->>'activo'<>'true' OR g->'correos'->0->'codigo'<>'null'::jsonb
 THEN RAISE EXCEPTION 'primer verificado no activo %',g; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.verificar',2,'correo-prueba-verif-otra-1','$A','aplicar',true);
  RAISE EXCEPTION 'código usado dos veces';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 r:=public.probar_correos('vec.correos.anadir',2,'correo-prueba-anadir-0002','$B');
 IF (r->>'version')::int<>3 THEN RAISE EXCEPTION 'segunda alta %',r; END IF;
 r:=public.probar_correos('vec.correos.reenviar',3,'correo-prueba-reenvia-01','$B');
 IF (r->>'version')::int<>4 OR jsonb_array_length(r->'envios')<>1 THEN RAISE EXCEPTION 'reenvío %',r; END IF;
 r:=public.probar_correos('vec.correos.verificar',4,'correo-prueba-verif-bien-2','$B','aplicar',true);
 IF (r->>'version')::int<>5 THEN RAISE EXCEPTION 'verificación B %',r; END IF;
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF g->'correos'->1->>'activo'<>'false' THEN RAISE EXCEPTION 'segundo verificado activado %',g; END IF;
 r:=public.probar_correos('vec.correos.activar',5,'correo-prueba-activar-b01','$B');
 IF (r->>'version')::int<>6 OR jsonb_array_length(r->'envios')<>1 OR r->'envios'->0->>'tipo'<>'aviso_cambio'
    OR r->'envios'->0->>'correo_ref'<>'$A' THEN RAISE EXCEPTION 'activar %',r; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.retirar',6,'correo-prueba-retira-act','$B');
  RAISE EXCEPTION 'activa retirada';
 EXCEPTION WHEN SQLSTATE 'P1413' THEN NULL; END;
 r:=public.probar_correos('vec.correos.retirar',6,'correo-prueba-retira-a01','$A');
 IF (r->>'version')::int<>7 THEN RAISE EXCEPTION 'retirar %',r; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.reenviar',7,'correo-prueba-reenvia-ret','$A');
  RAISE EXCEPTION 'reenvío a retirado';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF jsonb_array_length(g->'correos')<>1 OR g->'correos'->0->>'correo_ref'<>'$B' OR (g->>'version')::int<>7
 THEN RAISE EXCEPTION 'consulta final %',g; END IF;
 -- Tras retirarla, la misma dirección puede volver a añadirse.
 r:=public.probar_correos('vec.correos.anadir',7,'correo-prueba-anadir-0003','correo:cccccccccccccccccccccccccccccccc','aplicar',NULL,false,'interna_corporativa','igualdad-v1','$A');
 IF (r->>'version')::int<>8 THEN RAISE EXCEPTION 'realta %',r; END IF;
 r:=public.probar_correos('vec.correos.anadir',8,'correo-prueba-anadir-0004','correo:dddddddddddddddddddddddddddddddd');
 IF (r->>'version')::int<>9 THEN RAISE EXCEPTION 'quinto código %',r; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',9,'correo-prueba-limite-hora','correo:ffffffffffffffffffffffffffffffff');
  RAISE EXCEPTION 'límite de códigos por hora no aplicado';
 EXCEPTION WHEN SQLSTATE 'P1429' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',9,'correo-prueba-rotacion-01','correo:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee','aplicar',NULL,false,'interna_corporativa','igualdad-v2');
  RAISE EXCEPTION 'rotación de igualdad sin reindexado';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 -- Cinco intentos fallidos agotan el código: el sexto, aun correcto, es P1410.
 FOR i IN 1..5 LOOP
  r:=public.probar_correos('vec.correos.verificar',9,'correo-prueba-agota-000'||i,'correo:cccccccccccccccccccccccccccccccc','aplicar',false);
  IF (r->>'intentos_restantes')::int<>5-i THEN RAISE EXCEPTION 'intentos %',r; END IF;
 END LOOP;
 BEGIN
  PERFORM public.probar_correos('vec.correos.verificar',9,'correo-prueba-agota-0006','correo:cccccccccccccccccccccccccccccccc','aplicar',true);
  RAISE EXCEPTION 'código agotado admitido';
 EXCEPTION WHEN SQLSTATE 'P1410' THEN NULL; END;
 g:=public.probar_correos('vec.correos.consultar',0,'','');
 IF (SELECT count(*) FROM jsonb_array_elements(g->'correos') x WHERE x->>'correo_ref'='correo:cccccccccccccccccccccccccccccccc' AND x->'codigo'='null'::jsonb)<>1
 THEN RAISE EXCEPTION 'código agotado presentado como vigente %',g; END IF;
 RAISE NOTICE 'ENVIO % %', alta->'envios'->0->>'envio_ref', alta->'envios'->0->>'reserva_ref';
END \$prueba\$;
COMMIT;
SQL
)
echo "$salida" | grep -o 'ENVIO correo_envio:[0-9a-f]* reserva:[0-9a-f]*' | cut -d' ' -f2- > /dev/shm/.vec-508b-envio-$$
# La otra superficie ve el mismo conjunto de la persona con su propio LOGIN;
# material de una superficie con el LOGIN de la otra se deniega.
psql_externa -At <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $prueba$ DECLARE g jsonb; BEGIN
 g:=public.probar_correos('vec.correos.consultar',0,'','','aplicar',NULL,false,'externa_personal');
 IF (g->>'version')::int<>9 OR jsonb_array_length(g->'correos')<>3 THEN RAISE EXCEPTION 'consulta externa %',g; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.consultar',0,'','');
  RAISE EXCEPTION 'material interno admitido con LOGIN externo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $prueba$;
COMMIT;
SQL
# El superusuario de prueba no está sujeto a RLS: sólo lee la reserva.
# La reserva en claro solo la conoce quien ejecutó el alta: se toma de su recibo.
read -r envio reserva < /dev/shm/.vec-508b-envio-$$; rm -f /dev/shm/.vec-508b-envio-$$
[[ $envio =~ ^correo_envio:[0-9a-f]{32}$ && $reserva =~ ^reserva:[0-9a-f]{32}$ ]]
# Otra superficie no puede anotar el envío; la propia sólo una vez.
confirmar() { # $1 función psql, $2 reserva, $3 aceptado
 "$1" -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_usuarios.confirmar_envio_correo_v1('per_ABCDEFGHIJKLMNOPQRSTUV','$envio','$2',$3); COMMIT;" 2>&1 || true
}
confirmar psql_externa "$reserva" true | grep -q 'envío desconocido o ya anotado'
confirmar psql_interna 'reserva:00000000000000000000000000000000' true | grep -q 'envío desconocido o ya anotado'
confirmar psql_interna "$reserva" true | grep -qx t
confirmar psql_interna "$reserva" false | grep -q 'envío desconocido o ya anotado'
# Fuera de SERIALIZABLE no hay operación.
{ psql_interna -At -c "SELECT public.probar_correos('vec.correos.consultar',0,'','')" 2>&1 || true; } | grep -q 'material correo denegado'
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.correos_contexto)<>0
 THEN RAISE EXCEPTION 'contexto residual'; END IF;
 IF (SELECT count(*) FROM vec_usuarios.correos_historia)<>9
    OR (SELECT count(*) FROM vec_usuarios.correos_recibo)<>9
    OR (SELECT count(*) FROM vec_usuarios.correos_intento_fallido)<>6
    OR (SELECT count(*) FROM vec_usuarios.correos_envio)<>6
    OR (SELECT count(*) FROM vec_usuarios.correos_envio WHERE estado='aceptado')<>1
    OR (SELECT count(*) FROM vec_usuarios.correos_direccion WHERE activo)<>1
 THEN RAISE EXCEPTION 'historia, recibos o envíos inesperados'; END IF;
END $prueba$;
SQL
for sentencia in "UPDATE vec_usuarios.correos_historia SET estado_resultante='retirado'" \
  "DELETE FROM vec_usuarios.correos_recibo" "DELETE FROM vec_usuarios.correos_direccion" \
  "UPDATE vec_usuarios.correos_direccion SET cifrado=cifrado||'\\x00'::bytea" \
  "TRUNCATE vec_usuarios.correos_envio CASCADE"; do
 if psql_pg -c "$sentencia" >/dev/null 2>&1; then
  echo "historia mutable: $sentencia" >&2; exit 1
 fi
done
echo 'correos_pg18: OK'

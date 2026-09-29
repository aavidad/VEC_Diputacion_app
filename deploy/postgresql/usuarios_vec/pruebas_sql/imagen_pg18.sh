#!/usr/bin/env bash
# Usuarios 000006/000007 y Documentos 000007 (5.08c) en PostgreSQL 18.4
# efímero, sin red y con una V3 sintética de forma. No acredita COSE ni KMS.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-imagen-$$"
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
doc="$raiz/deploy/postgresql/documentos"
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
psql_pg < "$ad3/000108_consumidor_imagen_usuarios.up.sql" >/dev/null
psql_pg < "$doc/roles_up.sql" >/dev/null
psql_pg < "$doc/migraciones/000007_imagen_personal.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000006_imagen_propia.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000007_frontera_imagen.up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/imagen_vector_material.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/imagen_operaciones_sinteticas.sql" >/dev/null

# ACL: los ejecutores solo tienen las cuatro fachadas; ni tablas propias ni
# nada de Documentos. Usuarios solo alcanza las tres funciones documentales.
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF has_table_privilege('vec_usuarios_prueba_interna','vec_usuarios.imagen_actual','SELECT')
    OR has_table_privilege('vec_usuarios_prueba_externa','vec_usuarios.imagen_recibo','INSERT')
    OR has_table_privilege('vec_usuarios_prueba_interna','vec_documentos.imagen_personal','SELECT')
    OR has_table_privilege('vec_usuarios_propietario','vec_documentos.imagen_personal','SELECT')
    OR has_table_privilege('vec_usuarios_propietario','vec_documentos.imagen_personal_historia','SELECT')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_documentos.abrir_imagen_personal_v1(text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_externa','vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.validar_material_imagen(text,bytea,bytea,numeric,numeric)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_prueba_externa','vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_ct_prueba','vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.abrir_imagen_personal_v1(text,text)','EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios' AND tablename LIKE '%imagen%' AND roles<>ARRAY['vec_usuarios_propietario']::name[])
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_documentos' AND roles<>ARRAY['vec_documentos_propietario']::name[])
    OR position('mi-imagen' IN pg_get_functiondef('vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::regprocedure))=0
 THEN RAISE EXCEPTION 'ACL/RLS de imagen incompatible'; END IF;
END $prueba$;
SQL
# Un ejecutor no puede llamar a Documentos saltándose la V3 de Usuarios.
{ psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_documentos.abrir_imagen_personal_v1('per_ABCDEFGHIJKLMNOPQRSTUV','docimg_00000000000000000000000000000000'); COMMIT;" 2>&1 || true; } | grep -qi 'permission denied\|permiso denegado'

F1=ffd8ffe000104a46494600010101ffd9
F2=ffd8ffe000104a46494600020202ffd9
psql_interna -At <<SQL >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO \$prueba\$
DECLARE r jsonb; g jsonb; a jsonb; icono constant jsonb:='{"modo":"icono","paleta":"morado","icono":"hoja"}';
 foto constant jsonb:='{"modo":"foto","paleta":"azul","icono":""}';
BEGIN
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'existe'<>'false' OR (g->>'version')::int<>0 OR g->'eleccion'<>'{"modo":"iniciales","paleta":"azul","icono":""}'::jsonb OR g->>'foto_hex' IS NOT NULL
 THEN RAISE EXCEPTION 'GET inicial %',g; END IF;
 IF public.probar_imagen('recuperar',0,'imagen-prueba-icono-0001',icono) IS NOT NULL
 THEN RAISE EXCEPTION 'recuperación ausente devolvió recibo'; END IF;
 BEGIN
  PERFORM public.probar_imagen('recuperar',0,'imagen-prueba-v3-falsa-01',icono,NULL,true);
  RAISE EXCEPTION 'V3 falsa reveló clave ausente';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 a:=public.probar_imagen('guardar',0,'imagen-prueba-icono-0001',icono);
 IF (a->>'version')::int<>1 OR a->>'replay'<>'false' OR a->>'foto_nueva'<>'false' OR a->>'recibo_ref' !~ '^img_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'elección de icono %',a; END IF;
 r:=public.probar_imagen('recuperar',0,'imagen-prueba-icono-0001',icono);
 IF r->>'recibo_ref'<>a->>'recibo_ref' OR r->>'replay'<>'true' THEN RAISE EXCEPTION 'replay %',r; END IF;
 BEGIN
  PERFORM public.probar_imagen('recuperar',0,'imagen-prueba-icono-0001','{"modo":"icono","paleta":"verde","icono":"hoja"}');
  RAISE EXCEPTION 'clave con otra huella aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_imagen('guardar',0,'imagen-prueba-version-vie',icono);
  RAISE EXCEPTION 'versión antigua admitida';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_imagen('guardar',1,'imagen-prueba-sin-foto-01',foto);
  RAISE EXCEPTION 'modo foto sin foto admitido';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_imagen('guardar',1,'imagen-prueba-divergente',foto,decode('$F1','hex'),false,'interna_corporativa',NULL,repeat('b',64));
  RAISE EXCEPTION 'foto distinta de su huella admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_imagen('guardar',1,'imagen-prueba-no-jpeg-01',foto,decode('89504e470d0a1a0a','hex'));
  RAISE EXCEPTION 'contenido que no es JPEG custodiado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_imagen('guardar',1,'imagen-prueba-paleta-lib',jsonb_build_object('modo','iniciales','paleta','rojo','icono',''));
  RAISE EXCEPTION 'paleta fuera del vocabulario admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 a:=public.probar_imagen('guardar',1,'imagen-prueba-foto-00001',foto,decode('$F1','hex'));
 IF (a->>'version')::int<>2 OR a->>'foto_nueva'<>'true' OR a->>'foto_retirada'<>'false' THEN RAISE EXCEPTION 'foto %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'foto_hex'<>'$F1' OR g#>>'{eleccion,modo}'<>'foto' THEN RAISE EXCEPTION 'la foto no vuelve a su titular %',g; END IF;
 a:=public.probar_imagen('guardar',2,'imagen-prueba-conserva-01','{"modo":"foto","paleta":"verde","icono":""}');
 IF (a->>'version')::int<>3 OR a->>'foto_nueva'<>'false' OR a->>'foto_retirada'<>'false' THEN RAISE EXCEPTION 'conservar foto %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'foto_hex'<>'$F1' OR g#>>'{eleccion,paleta}'<>'verde' THEN RAISE EXCEPTION 'foto conservada %',g; END IF;
 a:=public.probar_imagen('guardar',3,'imagen-prueba-foto-00002',foto,decode('$F2','hex'));
 IF (a->>'version')::int<>4 OR a->>'foto_nueva'<>'true' OR a->>'foto_retirada'<>'true' THEN RAISE EXCEPTION 'sustitución %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'foto_hex'<>'$F2' THEN RAISE EXCEPTION 'la sustituta no se ve %',g; END IF;
 a:=public.probar_imagen('guardar',4,'imagen-prueba-inicial-01','{"modo":"iniciales","paleta":"gris","icono":""}');
 IF (a->>'version')::int<>5 OR a->>'foto_retirada'<>'true' THEN RAISE EXCEPTION 'quitar foto %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'foto_hex' IS NOT NULL OR g#>>'{eleccion,modo}'<>'iniciales' OR (g->>'version')::int<>5 THEN RAISE EXCEPTION 'foto tras quitarla %',g; END IF;
END \$prueba\$;
COMMIT;
SQL
# La otra superficie ve el mismo estado de la persona con su propio LOGIN;
# el material de una superficie con el LOGIN de la otra se deniega.
psql_externa -At <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $prueba$ DECLARE g jsonb; BEGIN
 g:=public.probar_imagen('consultar',0,'',NULL,NULL,false,'externa_personal');
 IF (g->>'version')::int<>5 THEN RAISE EXCEPTION 'consulta externa %',g; END IF;
 BEGIN
  PERFORM public.probar_imagen('consultar',0,'',NULL);
  RAISE EXCEPTION 'material interno admitido con LOGIN externo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $prueba$;
COMMIT;
SQL
# Fuera de SERIALIZABLE no hay operación.
{ psql_interna -At -c "SELECT public.probar_imagen('consultar',0,'',NULL)" 2>&1 || true; } | grep -q 'material imagen denegado'
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.imagen_contexto)<>0 THEN RAISE EXCEPTION 'contexto residual'; END IF;
 IF (SELECT count(*) FROM vec_usuarios.imagen_historia)<>5 OR (SELECT count(*) FROM vec_usuarios.imagen_recibo)<>5
    OR (SELECT count(*) FROM vec_usuarios.imagen_historia WHERE foto_retirada_ref IS NOT NULL)<>2
 THEN RAISE EXCEPTION 'historia o recibos de Usuarios inesperados'; END IF;
 -- Documentos: dos fotos custodiadas y retiradas, sin bytes; ninguna viva.
 IF (SELECT count(*) FROM vec_documentos.imagen_personal)<>2
    OR (SELECT count(*) FROM vec_documentos.imagen_personal WHERE estado='retirada' AND contenido IS NULL)<>2
    OR (SELECT count(*) FROM vec_documentos.imagen_personal_historia)<>4
    OR (SELECT count(*) FROM vec_documentos.imagen_personal_historia WHERE operacion_ref !~ '^img_[0-9a-f]{32}$')<>0
 THEN RAISE EXCEPTION 'custodia documental inesperada'; END IF;
 -- Cada retirada queda enlazada al recibo de Usuarios que la provocó.
 IF (SELECT count(*) FROM vec_documentos.imagen_personal_historia h JOIN vec_usuarios.imagen_recibo r
      ON r.recibo_ref=h.operacion_ref)<>4
 THEN RAISE EXCEPTION 'historia documental sin su operación de Usuarios'; END IF;
END $prueba$;
SQL
for sentencia in "UPDATE vec_usuarios.imagen_historia SET foto_ref=NULL" \
  "DELETE FROM vec_usuarios.imagen_recibo" "TRUNCATE vec_usuarios.imagen_actual CASCADE" \
  "UPDATE vec_documentos.imagen_personal SET contenido='\\xffd8ffd9'::bytea" \
  "DELETE FROM vec_documentos.imagen_personal" "UPDATE vec_documentos.imagen_personal_historia SET evento='retirada'" \
  "TRUNCATE vec_documentos.imagen_personal_historia CASCADE"; do
 if psql_pg -c "$sentencia" >/dev/null 2>&1; then
  echo "historia mutable: $sentencia" >&2; exit 1
 fi
done
echo 'imagen_pg18: OK'

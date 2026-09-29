#!/usr/bin/env bash
# Fase 1, paso 3: Documentos 000008, Usuarios 000008, roles_000009, AD3-109 y
# Usuarios 000009 en PostgreSQL 18.4 efímero, sin red y con la V3 sintética
# de forma. Siembra datos de la MISMA persona desde los dos portales (el caso
# que antes se compartía), ensaya cada migración con ROLLBACK (la base queda
# idéntica), provoca un fallo a mitad (tampoco deja rastro), migra y comprueba:
#  - que la historia y los recibos no pierden ni cambian nada;
#  - que cada portal ve solo lo suyo (preferencias, imagen y correos);
#  - que un LOGIN de un portal no puede leer ni escribir lo del otro.
# No acredita COSE, KMS, SMTP ni el binario.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-superficie-$$"
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
# pg_dump 18 añade una llave aleatoria en \restrict/\unrestrict: no cuenta.
huella_base() { { docker exec "$contenedor" pg_dumpall -U postgres --roles-only; docker exec "$contenedor" pg_dump -U postgres -d postgres; } | grep -v '^\\\(un\)\?restrict ' | sha256sum | cut -d' ' -f1; }
sql="$raiz/deploy/postgresql/usuarios_vec"
ad3="$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones"
doc="$raiz/deploy/postgresql/documentos"
for f in "$sql/roles_up.sql" "$sql/pruebas_sql/preimagen_ad3_sintetica.sql" "$ad3/000106_consumidor_preferencias_propias.up.sql" \
 "$sql/migraciones/000001_preferencias_base.up.sql" "$sql/migraciones/000002_operaciones_preferencias.up.sql" \
 "$sql/roles_000003_up.sql" "$sql/migraciones/000003_auditoria_frontera_preferencias.up.sql" \
 "$ad3/000107_consumidor_correos_usuarios.up.sql" "$sql/migraciones/000004_correos_propios.up.sql" \
 "$sql/migraciones/000005_frontera_correos.up.sql" "$ad3/000108_consumidor_imagen_usuarios.up.sql" \
 "$doc/roles_up.sql" "$doc/migraciones/000007_imagen_personal.up.sql" \
 "$sql/migraciones/000006_imagen_propia.up.sql" "$sql/migraciones/000007_frontera_imagen.up.sql" \
 "$sql/pruebas_sql/operaciones_sinteticas.sql" "$sql/pruebas_sql/imagen_operaciones_sinteticas.sql" \
 "$sql/pruebas_sql/correos_operaciones_sinteticas.sql"; do
 psql_pg < "$f" >/dev/null
done

F1=ffd8ffe000104a46494600010101ffd9
F2=ffd8ffe000104a46494600020202ffd9
F3=ffd8ffe000104a46494600030303ffd9
# 1. Siembra con el modelo antiguo: la misma persona alterna de portal.
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_preferencias('put',0,'pref-int-00000001','{\"idioma\":\"es\",\"tamano_texto\":\"normal\",\"alto_contraste\":false,\"tema\":\"claro\",\"inicio\":\"cuadro\",\"filas\":20,\"aviso_correo_tareas\":false,\"aviso_correo_plazos\":false}'); COMMIT;" >/dev/null
psql_externa -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_preferencias('put',1,'pref-ext-00000001','{\"idioma\":\"en\",\"tamano_texto\":\"grande\",\"alto_contraste\":true,\"tema\":\"oscuro\",\"inicio\":\"bolsas\",\"filas\":50,\"aviso_correo_tareas\":true,\"aviso_correo_plazos\":false}',false,'externa_personal'); COMMIT;" >/dev/null
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_preferencias('put',2,'pref-int-00000002','{\"idioma\":\"es\",\"tamano_texto\":\"muy_grande\",\"alto_contraste\":false,\"tema\":\"sistema\",\"inicio\":\"peticiones\",\"filas\":100,\"aviso_correo_tareas\":false,\"aviso_correo_plazos\":true}'); COMMIT;" >/dev/null
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_imagen('guardar',0,'imagen-int-00000001','{\"modo\":\"foto\",\"paleta\":\"azul\",\"icono\":\"\"}',decode('$F1','hex')); COMMIT;" >/dev/null
psql_externa -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_imagen('guardar',1,'imagen-ext-00000001','{\"modo\":\"foto\",\"paleta\":\"verde\",\"icono\":\"\"}',NULL,false,'externa_personal'); COMMIT;" >/dev/null
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_imagen('guardar',2,'imagen-int-00000002','{\"modo\":\"foto\",\"paleta\":\"gris\",\"icono\":\"\"}'); COMMIT;" >/dev/null
A=correo:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
B=correo:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_correos('vec.correos.anadir',0,'correo-int-000001','$A'); COMMIT;" >/dev/null
psql_externa -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_correos('vec.correos.anadir',1,'correo-ext-000001','$B','aplicar',NULL,false,'externa_personal'); COMMIT;" >/dev/null
psql_externa -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_correos('vec.correos.verificar',2,'correo-ext-000002','$B','aplicar',true,false,'externa_personal'); COMMIT;" >/dev/null
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_correos('vec.correos.verificar',3,'correo-int-000002','$A','aplicar',false); COMMIT;" >/dev/null
psql_interna -At -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_correos('vec.correos.verificar',3,'correo-int-000003','$A','aplicar',true); COMMIT;" >/dev/null
# Antes de migrar, el fallo que se corrige: el portal externo ve lo interno.
psql_externa -At <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $prueba$ DECLARE g jsonb; BEGIN
 g:=public.probar_preferencias('get',0,'','{"idioma":"es","tamano_texto":"normal","alto_contraste":false,"tema":"claro","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}',false,'externa_personal');
 IF (g->>'version')::int<>3 OR g#>>'{valores,tema}'<>'sistema' THEN RAISE EXCEPTION 'preimagen compartida no reproducida %',g; END IF;
 g:=public.probar_correos('vec.correos.consultar',0,'',NULL,'aplicar',NULL,false,'externa_personal');
 IF jsonb_array_length(g->'correos')<>2 THEN RAISE EXCEPTION 'preimagen de correos no reproducida %',g; END IF;
END $prueba$;
COMMIT;
SQL

# 2. Cada migración en ROLLBACK deja la base idéntica; después se aplica.
lista=(
 "$doc/migraciones/000008_imagen_personal_superficie.up.sql"
 "$sql/migraciones/000008_superficie_preferencias_imagen.up.sql"
 "$sql/roles_000009_up.sql"
 "$ad3/000109_correos_por_poblacion.up.sql"
 "$sql/migraciones/000009_correos_por_poblacion.up.sql"
)
for f in "${lista[@]}"; do
 antes=$(huella_base)
 if [[ "$(grep -c '^COMMIT;$' "$f")" != 1 ]]; then echo "sin COMMIT único: $f" >&2; exit 1; fi
 sed 's/^COMMIT;$/ROLLBACK;/' "$f" | psql_pg >/dev/null
 if [[ "$(huella_base)" != "$antes" ]]; then echo "ROLLBACK dejó rastro: $f" >&2; exit 1; fi
 if [[ "$f" == *usuarios_vec/migraciones/000008_* ]]; then
  # Fallo a mitad: una decisión V3 desaparece dentro de la misma transacción.
  salida=$(sed "s/^LOCK TABLE vec_usuarios.preferencias_actual,/DELETE FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 WHERE decision_ref=(SELECT min(decision_ref) FROM vec_usuarios.preferencias_historia);\n&/" "$f" | psql_pg 2>&1 >/dev/null || true)
  grep -q 'decisión sin superficie deducible' <<<"$salida" || { echo "fallo inyectado no detectado: $salida" >&2; exit 1; }
  if [[ "$(huella_base)" != "$antes" ]]; then echo "el fallo dejó rastro" >&2; exit 1; fi
 fi
 psql_pg < "$f" >/dev/null
done
# Repetir una migración ya aplicada se rechaza sin cambios.
antes=$(huella_base)
for f in "${lista[@]}"; do
 if psql_pg < "$f" >/dev/null 2>&1; then echo "reaplicación admitida: $f" >&2; exit 1; fi
done
[[ "$(huella_base)" == "$antes" ]]
for f in "$sql/migraciones/000008_superficie_preferencias_imagen.down.sql" "$sql/migraciones/000009_correos_por_poblacion.down.sql" \
 "$ad3/000109_correos_por_poblacion.down.sql"; do
 if psql_pg < "$f" >/dev/null 2>&1; then echo "DOWN admitido: $f" >&2; exit 1; fi
done
psql_pg < "$sql/pruebas_sql/correos_poblacion_operaciones_sinteticas.sql" >/dev/null

# 3. Reparto de lo existente: nada se pierde y cada portal conserva lo suyo.
psql_pg <<SQL >/dev/null
DO \$prueba\$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.preferencias_historia)<>3 OR (SELECT count(*) FROM vec_usuarios.preferencias_recibo)<>3
    OR (SELECT string_agg(superficie||':'||version,',' ORDER BY version) FROM vec_usuarios.preferencias_historia)
       <>'interna_corporativa:1,externa_personal:2,interna_corporativa:3'
    OR (SELECT string_agg(superficie||':'||version||':'||(valores->>'tema'),',' ORDER BY superficie) FROM vec_usuarios.preferencias_actual)
       <>'externa_personal:2:oscuro,interna_corporativa:3:sistema'
 THEN RAISE EXCEPTION 'reparto de preferencias'; END IF;
 -- La foto la subió el portal interno: el externo vuelve a iniciales (verde)
 -- con una versión anotada; la foto sigue viva solo en el interno.
 IF (SELECT count(*) FROM vec_usuarios.imagen_historia)<>4 OR (SELECT count(*) FROM vec_usuarios.imagen_recibo)<>3
    OR (SELECT string_agg(superficie||':'||version||':'||(eleccion->>'modo')||':'||(eleccion->>'paleta')||':'||(foto_ref IS NOT NULL),',' ORDER BY superficie)
        FROM vec_usuarios.imagen_actual)<>'externa_personal:3:iniciales:verde:false,interna_corporativa:3:foto:gris:true'
    OR (SELECT count(*) FROM vec_usuarios.imagen_historia WHERE decision_ref='migracion:usuarios:000008' AND superficie='externa_personal' AND version=3)<>1
    OR (SELECT string_agg(superficie||':'||estado,',') FROM vec_documentos.imagen_personal)<>'interna_corporativa:activa'
 THEN RAISE EXCEPTION 'reparto de imagen'; END IF;
 -- Cada dirección va con el portal donde se añadió, con toda su historia.
 IF (SELECT string_agg(correo_ref||':'||estado||':'||activo,',') FROM vec_usuarios_correos_interno.correos_direccion)<>'$A:verificado:false'
    OR (SELECT string_agg(correo_ref||':'||estado||':'||activo,',') FROM vec_usuarios_correos_externo.correos_direccion)<>'$B:verificado:true'
    OR (SELECT string_agg(version::text,',' ORDER BY version) FROM vec_usuarios_correos_interno.correos_historia)<>'1,4'
    OR (SELECT string_agg(version::text,',' ORDER BY version) FROM vec_usuarios_correos_externo.correos_historia)<>'2,3'
    OR (SELECT version FROM vec_usuarios_correos_interno.correos_conjunto)<>4
    OR (SELECT version FROM vec_usuarios_correos_externo.correos_conjunto)<>3
    OR (SELECT count(*) FROM vec_usuarios_correos_interno.correos_intento_fallido)<>1
    OR (SELECT count(*) FROM vec_usuarios_correos_externo.correos_intento_fallido)<>0
    OR (SELECT count(*) FROM vec_usuarios_correos_interno.correos_envio)<>1 OR (SELECT count(*) FROM vec_usuarios_correos_externo.correos_envio)<>1
    OR (SELECT count(*) FROM vec_usuarios_correos_interno.correos_recibo)+(SELECT count(*) FROM vec_usuarios_correos_externo.correos_recibo)<>4
    OR to_regclass('vec_usuarios.correos_direccion') IS NOT NULL
 THEN RAISE EXCEPTION 'reparto de correos'; END IF;
END \$prueba\$;
SQL

# 4. Después: cada portal lee y escribe solo lo suyo.
psql_externa -At <<SQL >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO \$prueba\$ DECLARE g jsonb; a jsonb; BEGIN
 g:=public.probar_preferencias('get',0,'','{"idioma":"es","tamano_texto":"normal","alto_contraste":false,"tema":"claro","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}',false,'externa_personal');
 IF (g->>'version')::int<>2 OR g#>>'{valores,tema}'<>'oscuro' THEN RAISE EXCEPTION 'preferencias externas %',g; END IF;
 -- La clave de una operación interna no existe para el portal externo.
 IF public.probar_preferencias('rec',2,'pref-int-00000002','{"idioma":"es","tamano_texto":"muy_grande","alto_contraste":false,"tema":"sistema","inicio":"peticiones","filas":100,"aviso_correo_tareas":false,"aviso_correo_plazos":true}',false,'externa_personal') IS NOT NULL
 THEN RAISE EXCEPTION 'recibo interno visible desde fuera'; END IF;
 BEGIN
  PERFORM public.probar_preferencias('put',3,'pref-ext-version-in','{"idioma":"es","tamano_texto":"normal","alto_contraste":false,"tema":"claro","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}',false,'externa_personal');
  RAISE EXCEPTION 'el externo escribió con la versión interna';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 a:=public.probar_preferencias('put',2,'pref-ext-00000002','{"idioma":"es","tamano_texto":"normal","alto_contraste":true,"tema":"claro","inicio":"bolsas","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}',false,'externa_personal');
 IF (a->>'version')::int<>3 THEN RAISE EXCEPTION 'guardado externo %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL,NULL,false,'externa_personal');
 IF g->>'foto_hex' IS NOT NULL OR g#>>'{eleccion,modo}'<>'iniciales' OR (g->>'version')::int<>3 THEN RAISE EXCEPTION 'imagen externa %',g; END IF;
 a:=public.probar_imagen('guardar',3,'imagen-ext-00000002','{"modo":"foto","paleta":"morado","icono":""}',decode('$F2','hex'),false,'externa_personal');
 IF (a->>'version')::int<>4 OR a->>'foto_nueva'<>'true' OR a->>'foto_retirada'<>'false' THEN RAISE EXCEPTION 'foto externa %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL,NULL,false,'externa_personal');
 IF g->>'foto_hex'<>'$F2' THEN RAISE EXCEPTION 'foto externa no vuelve %',g; END IF;
 g:=public.probar_correos_poblacion('vec.correos.consultar',0,'',NULL,'aplicar',NULL,false,'externa_personal');
 IF (g->>'version')::int<>3 OR jsonb_array_length(g->'correos')<>1 OR g#>>'{correos,0,correo_ref}'<>'$B' THEN RAISE EXCEPTION 'correos externos %',g; END IF;
 IF public.probar_correos_poblacion('vec.correos.anadir',1,'correo-int-000001','$A','recuperar',NULL,false,'externa_personal') IS NOT NULL
 THEN RAISE EXCEPTION 'recibo de correo interno visible desde fuera'; END IF;
 a:=public.probar_correos_poblacion('vec.correos.anadir',3,'correo-ext-000003','correo:cccccccccccccccccccccccccccccccc','aplicar',NULL,false,'externa_personal');
 IF (a->>'version')::int<>4 THEN RAISE EXCEPTION 'alta externa %',a; END IF;
 -- El LOGIN externo con material externo contra el esquema interno: denegado.
 BEGIN
  PERFORM public.probar_correos_poblacion('vec.correos.consultar',0,'',NULL,'aplicar',NULL,false,'externa_personal',
   'igualdad-v1','','','vec_usuarios_correos_interno');
  RAISE EXCEPTION 'el LOGIN externo entró en el esquema interno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 -- Y con material interno: también (la superficie no casa con el LOGIN).
 BEGIN
  PERFORM public.probar_preferencias('get',0,'','{"idioma":"es","tamano_texto":"normal","alto_contraste":false,"tema":"claro","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}');
  RAISE EXCEPTION 'material interno admitido con LOGIN externo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END \$prueba\$;
COMMIT;
SQL
psql_interna -At <<SQL >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO \$prueba\$ DECLARE g jsonb; a jsonb; BEGIN
 g:=public.probar_preferencias('get',0,'','{"idioma":"es","tamano_texto":"normal","alto_contraste":false,"tema":"claro","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}');
 IF (g->>'version')::int<>3 OR g#>>'{valores,tema}'<>'sistema' THEN RAISE EXCEPTION 'preferencias internas cambiadas por fuera %',g; END IF;
 a:=public.probar_preferencias('put',3,'pref-int-00000003','{"idioma":"en","tamano_texto":"normal","alto_contraste":false,"tema":"oscuro","inicio":"cuadro","filas":50,"aviso_correo_tareas":true,"aviso_correo_plazos":true}');
 IF (a->>'version')::int<>4 THEN RAISE EXCEPTION 'guardado interno %',a; END IF;
 g:=public.probar_imagen('consultar',0,'',NULL);
 IF g->>'foto_hex'<>'$F1' OR g#>>'{eleccion,paleta}'<>'gris' THEN RAISE EXCEPTION 'foto interna %',g; END IF;
 a:=public.probar_imagen('guardar',3,'imagen-int-00000003','{"modo":"foto","paleta":"azul","icono":""}',decode('$F3','hex'));
 IF (a->>'version')::int<>4 OR a->>'foto_retirada'<>'true' THEN RAISE EXCEPTION 'sustitución interna %',a; END IF;
 g:=public.probar_correos_poblacion('vec.correos.consultar',0,'',NULL);
 IF (g->>'version')::int<>4 OR jsonb_array_length(g->'correos')<>1 OR g#>>'{correos,0,correo_ref}'<>'$A' THEN RAISE EXCEPTION 'correos internos %',g; END IF;
 BEGIN
  PERFORM public.probar_correos_poblacion('vec.correos.consultar',0,'',NULL,'aplicar',NULL,false,'interna_corporativa',
   'igualdad-v1','','','vec_usuarios_correos_externo');
  RAISE EXCEPTION 'el LOGIN interno entró en el esquema externo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END \$prueba\$;
COMMIT;
SQL
# La sustitución interna retiró F1 y no tocó la foto externa.
psql_pg -At -c "SELECT string_agg(superficie||':'||estado||':'||encode(contenido,'hex'),',' ORDER BY superficie,estado) FROM vec_documentos.imagen_personal WHERE estado='activa'" \
 | grep -qx "externa_personal:activa:$F2,interna_corporativa:activa:$F3"
psql_pg -At -c "SELECT (SELECT count(*) FROM vec_documentos.imagen_personal WHERE estado='retirada' AND contenido IS NULL)=1" | grep -qx t

# 5. Privilegios directos: ningún LOGIN alcanza el esquema del otro portal
# ni las tablas de ninguno; el propietario común ya no alcanza correos.
{ psql_externa -At -c "SELECT vec_usuarios_correos_interno.confirmar_envio_correo_v1('per_ABCDEFGHIJKLMNOPQRSTUV','correo_envio:00000000000000000000000000000000','reserva:00000000000000000000000000000000',true)" 2>&1 || true; } | grep -qi 'permission denied\|permiso denegado'
{ psql_interna -At -c "SELECT count(*) FROM vec_usuarios_correos_externo.correos_direccion" 2>&1 || true; } | grep -qi 'permission denied\|permiso denegado'
{ psql_interna -At -c "SELECT count(*) FROM vec_usuarios_correos_interno.correos_direccion" 2>&1 || true; } | grep -qi 'permission denied\|permiso denegado'
{ psql_externa -At -c "SELECT count(*) FROM vec_usuarios.preferencias_actual" 2>&1 || true; } | grep -qi 'permission denied\|permiso denegado'
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF has_schema_privilege('vec_usuarios_propietario','vec_usuarios_correos_interno','USAGE')
    OR has_schema_privilege('vec_usuarios_propietario','vec_usuarios_correos_externo','USAGE')
    OR has_schema_privilege('vec_usuarios_correos_interno_propietario','vec_usuarios_correos_externo','USAGE')
    OR has_schema_privilege('vec_usuarios_correos_externo_propietario','vec_usuarios_correos_interno','USAGE')
    OR has_schema_privilege('vec_usuarios_prueba_externa','vec_usuarios_correos_interno','USAGE')
    OR has_schema_privilege('vec_usuarios_prueba_interna','vec_usuarios_correos_externo','USAGE')
    OR NOT has_schema_privilege('vec_usuarios_prueba_interna','vec_usuarios_correos_interno','USAGE')
    OR has_function_privilege('vec_usuarios_propietario','vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_documentos.superficie_sesion_imagen_v1()','EXECUTE')
 THEN RAISE EXCEPTION 'privilegios cruzados'; END IF;
 IF (SELECT count(*) FROM vec_usuarios.contexto_transaccion)+(SELECT count(*) FROM vec_usuarios.imagen_contexto)
   +(SELECT count(*) FROM vec_usuarios_correos_interno.correos_contexto)+(SELECT count(*) FROM vec_usuarios_correos_externo.correos_contexto)<>0
 THEN RAISE EXCEPTION 'contexto residual'; END IF;
END $prueba$;
SQL
# 6. Historia de solo adición también en los esquemas nuevos y en la columna nueva.
for sentencia in "UPDATE vec_usuarios.preferencias_historia SET superficie='externa_personal'" \
  "DELETE FROM vec_usuarios.imagen_recibo" "UPDATE vec_usuarios_correos_interno.correos_historia SET version=9" \
  "DELETE FROM vec_usuarios_correos_externo.correos_recibo" "TRUNCATE vec_usuarios_correos_externo.correos_historia CASCADE" \
  "UPDATE vec_documentos.imagen_personal SET superficie='externa_personal'"; do
 if psql_pg -c "$sentencia" >/dev/null 2>&1; then
  echo "historia mutable: $sentencia" >&2; exit 1
 fi
done
echo 'superficie_pg18: OK'

#!/usr/bin/env bash
# Aspirantes 000001 y AD3-110 en PostgreSQL 18.4 efímero, sin red, con los
# datos en /dev/shm y una V3 sintética de forma. No acredita COSE ni KMS.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4}
contenedor="vec-aspirantes-ficha-$$"
datos=$(mktemp -d "${VEC_PG_TMP:-/dev/shm}/vec-aspirantes-XXXX")
chmod 0777 "$datos"
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --pull never --network none -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
 rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
# Con --go publica el puerto solo en 127.0.0.1 y, tras las pruebas SQL,
# recorre Go → PostgreSQL con el adaptador real (ficha_pg18_test.go).
red=(--network none)
[[ ${1:-} == --go ]] && red=(-p 127.0.0.1::5432)
docker run -d --rm --pull never "${red[@]}" --name "$contenedor" \
 -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break
 sleep 0.5
done
sleep 2
docker exec "$contenedor" pg_isready -q -U postgres
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
psql_como() { local u=$1; shift; docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U "$u" -d postgres "$@"; }
sql="$raiz/deploy/postgresql/aspirantes"
ad3="$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones"
psql_pg -c "REVOKE CREATE ON DATABASE postgres FROM PUBLIC" >/dev/null
psql_pg < "$sql/roles_up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/preimagen_ad3_sintetica.sql" >/dev/null
psql_pg < "$ad3/000110_consumidor_aspirantes.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000001_ficha_propia.up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/vector_material.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/operaciones_sinteticas.sql" >/dev/null
# Una segunda instalación no se admite: la preimagen ya no coincide.
if psql_pg < "$ad3/000110_consumidor_aspirantes.up.sql" >/dev/null 2>&1; then echo "AD3-110 reinstalada"; exit 1; fi
if psql_pg < "$sql/migraciones/000001_ficha_propia.down.sql" >/dev/null 2>&1; then echo "DOWN admitido"; exit 1; fi

psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF has_table_privilege('vec_aspirantes_prueba_externa','vec_aspirantes.valor','SELECT')
    OR has_table_privilege('vec_aspirantes_prueba_externa','vec_aspirantes.acceso','INSERT')
    OR has_function_privilege('vec_aspirantes_prueba_externa','vec_aspirantes.validar_material(text,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_aspirantes_prueba_externa','vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_aspirantes_prueba_externa','vec_aspirantes.consultar_ficha_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_ct_prueba','vec_aspirantes.consultar_ficha_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'aspirantes_ficha_rectificar')=0
    OR (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check') NOT LIKE '%vec_aspirantes.ficha.alta.externa_personal.v1%'
 THEN RAISE EXCEPTION 'ACL o parche incompatibles'; END IF;
END $prueba$;
SQL

I1=$(printf '1%.0s' $(seq 64)); I2=$(printf '2%.0s' $(seq 64)); I3=$(printf '3%.0s' $(seq 64))
A1=asp_AAAAAAAAAAAAAAAAAAAAAA; D1=aspdoc_AAAAAAAAAAAAAAAAAAAAAA
A2=asp_BBBBBBBBBBBBBBBBBBBBBA; D2=aspdoc_BBBBBBBBBBBBBBBBBBBBBA
psql_como vec_aspirantes_prueba_externa <<SQL >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO \$prueba\$
DECLARE r jsonb; g jsonb; alta jsonb; x jsonb;
BEGIN
 g:=public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1');
 IF g<>'{"estado":"sin_ficha"}'::jsonb THEN RAISE EXCEPTION 'consulta inicial %',g; END IF;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',1,'rect-lucia-00000001','$I1');
  RAISE EXCEPTION 'rectificar sin ficha';
 EXCEPTION WHEN SQLSTATE 'P1404' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1',NULL,true);
  RAISE EXCEPTION 'V3 falsa aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1',NULL,false,'','per_ABCDEFGHIJKLMNOPQRSTUV','vec_aspirantes.ficha.alta.externa_personal.v1');
  RAISE EXCEPTION 'audiencia ajena aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 alta:=public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-lucia-00000001','$I1',public.ficha_prueba('$A1','$D1','$I1'));
 IF (alta->>'version')::int<>1 OR alta->>'replay'<>'false' OR alta->>'recibo_ref' !~ '^asprec_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'alta %',alta; END IF;
 r:=public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-lucia-00000001','$I1',public.ficha_prueba('$A1','$D1','$I1'));
 IF r->>'recibo_ref'<>alta->>'recibo_ref' OR r->>'replay'<>'true' THEN RAISE EXCEPTION 'replay alta %',r; END IF;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-lucia-00000001','$I1',public.ficha_prueba('$A1','$D1','$I1'),false,'otra');
  RAISE EXCEPTION 'clave con otra huella aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-lucia-00000002','$I1',public.ficha_prueba('$A2','$D2','$I1'));
  RAISE EXCEPTION 'segunda ficha con el mismo documento';
 EXCEPTION WHEN SQLSTATE 'P1411' THEN NULL; END;
 g:=public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1');
 IF g->>'estado'<>'activa' OR g->>'aspirante_ref'<>'$A1' OR (g->>'version')::int<>1 OR jsonb_array_length(g->'valores')<>4
    OR g->'documento'->>'tipo'<>'dni' OR g->'documento'->'sobre'->>'cifrado_hex'<>repeat('bb',25)
    OR g->'documento'->'indice'->>'valor'<>'$I1' OR g->>'acceso_ref' !~ '^aspacc_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'consulta con ficha %',g; END IF;
 -- Otra persona con otro documento no ve esta ficha.
 g:=public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I2',NULL,false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
 IF g->>'estado'<>'sin_ficha' THEN RAISE EXCEPTION 'fuga entre fichas %',g; END IF;
 r:=public.probar_ficha('vec.aspirantes.ficha.rectificar',1,'rect-lucia-00000001','$I1',public.cambio_prueba(2,'dato_nuevo','movil'));
 IF (r->>'version')::int<>2 OR r->>'replay'<>'false' THEN RAISE EXCEPTION 'rectificar %',r; END IF;
 r:=public.probar_ficha('vec.aspirantes.ficha.rectificar',1,'rect-lucia-00000001','$I1',public.cambio_prueba(2,'dato_nuevo','movil'));
 IF r->'replay'->>'replay'<>'true' OR (r->'replay'->>'version')::int<>2 THEN RAISE EXCEPTION 'replay rectificar %',r; END IF;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',1,'rect-lucia-00000002','$I1',public.cambio_prueba(2,'dato_nuevo','domicilio'));
  RAISE EXCEPTION 'versión antigua aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000003','$I1',public.cambio_prueba(3,'dato_nuevo','telefono'));
  RAISE EXCEPTION 'dato nuevo sobre un valor existente';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000004','$I1',public.cambio_prueba(3,'cambio_de_dato','domicilio'));
  RAISE EXCEPTION 'cambio sobre un dato que no había';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000005','$I1',public.cambio_prueba(3,'cambio_de_dato','codigo_postal',true));
  RAISE EXCEPTION 'retirada de un dato ausente';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000006','$I1',public.cambio_prueba(4,'cambio_de_dato','telefono'));
  RAISE EXCEPTION 'versión del valor distinta de la de la ficha';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000007','$I1',
   jsonb_set(public.cambio_prueba(3,'cambio_de_dato','telefono'),'{valores,0,campo}','"nombre"'));
  RAISE EXCEPTION 'identidad rectificada por el titular';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 r:=public.probar_ficha('vec.aspirantes.ficha.rectificar',2,'rect-lucia-00000008','$I1',public.cambio_prueba(3,'correccion_de_error','telefono'));
 IF (r->>'version')::int<>3 THEN RAISE EXCEPTION 'corrección %',r; END IF;
 r:=public.probar_ficha('vec.aspirantes.ficha.rectificar',3,'rect-lucia-00000009','$I1',public.cambio_prueba(4,'cambio_de_dato','movil',true));
 IF (r->>'version')::int<>4 THEN RAISE EXCEPTION 'retirada %',r; END IF;
 g:=public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1');
 SELECT x2 INTO x FROM jsonb_array_elements(g->'valores') x2 WHERE x2->>'campo'='movil';
 IF (g->>'version')::int<>4 OR x->'sobre'<>'null'::jsonb OR (x->>'version')::int<>4 THEN RAISE EXCEPTION 'consulta final %',g; END IF;
 SELECT x2 INTO x FROM jsonb_array_elements(g->'valores') x2 WHERE x2->>'campo'='telefono';
 IF (x->>'version')::int<>3 OR x->'sobre'->>'cifrado_hex'<>repeat('15',25) THEN RAISE EXCEPTION 'teléfono vigente %',x; END IF;
 -- Altas inválidas con otro documento.
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-karim-00000001','$I2',
   public.ficha_prueba('$A2','$D2','$I2')#-'{valores,1}',false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
  RAISE EXCEPTION 'alta sin primer apellido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-karim-00000002','$I2',
   public.ficha_prueba('$A2','$D2','$I3'),false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
  RAISE EXCEPTION 'alta con índice distinto del material';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-karim-00000003','$I2',
   jsonb_set(public.ficha_prueba('$A2','$D2','$I2'),'{valores,3,campo}','"discapacidad"'),false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
  RAISE EXCEPTION 'campo de categoría especial admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-karim-00000004','$I2',
   jsonb_set(public.ficha_prueba('$A2','$D2','$I2'),'{documento,pais}','"FR"'),false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
  RAISE EXCEPTION 'DNI de otro país admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 r:=public.probar_ficha('vec.aspirantes.ficha.alta',0,'alta-karim-00000005','$I2',
   jsonb_set(jsonb_set(public.ficha_prueba('$A2','$D2','$I2'),'{documento,tipo}','"nie"'),'{valores}',(public.ficha_prueba('$A2','$D2','$I2')->'valores')-2),
   false,'','per_ZYXWVUTSRQPONMLKJIHGFE');
 IF (r->>'version')::int<>1 THEN RAISE EXCEPTION 'alta con un solo apellido %',r; END IF;
END \$prueba\$;
COMMIT;
SQL

# Recuento, inmutabilidad y aislamiento, como superusuario.
psql_pg <<SQL >/dev/null
DO \$prueba\$ DECLARE n int; BEGIN
 IF (SELECT count(*) FROM vec_aspirantes.ficha)<>2 OR (SELECT count(*) FROM vec_aspirantes.historia)<>5
    OR (SELECT count(*) FROM vec_aspirantes.recibo)<>5 OR (SELECT count(*) FROM vec_aspirantes.evento_salida)<>5
    OR (SELECT count(*) FROM vec_aspirantes.acceso)<>2 OR (SELECT count(*) FROM vec_aspirantes.contexto)<>0
    OR (SELECT count(*) FROM vec_aspirantes.valor WHERE aspirante_ref='$A1')<>7
    OR (SELECT motivo FROM vec_aspirantes.historia WHERE aspirante_ref='$A1' AND version=3)<>'correccion_de_error'
    OR (SELECT campos FROM vec_aspirantes.acceso ORDER BY ocurrido_en DESC LIMIT 1)<>ARRAY['documento','nombre','primer_apellido','segundo_apellido','telefono']
 THEN RAISE EXCEPTION 'recuento'; END IF;
 BEGIN UPDATE vec_aspirantes.valor SET estado='retirado',clave_ref=NULL,nonce=NULL,cifrado=NULL WHERE aspirante_ref='$A1' AND campo='telefono' AND version=1;
  RAISE EXCEPTION 'valor modificado'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN DELETE FROM vec_aspirantes.historia; RAISE EXCEPTION 'historia borrada'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN DELETE FROM vec_aspirantes.acceso; RAISE EXCEPTION 'acceso borrado'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN UPDATE vec_aspirantes.ficha SET version=9 WHERE aspirante_ref='$A1'; RAISE EXCEPTION 'salto de versión'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN DELETE FROM vec_aspirantes.ficha; RAISE EXCEPTION 'ficha borrada'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN TRUNCATE vec_aspirantes.indice_documento CASCADE; RAISE EXCEPTION 'índice truncado'; EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END \$prueba\$;
SQL

# Un LOGIN con otra membresía vec_ no pasa; tampoco fuera de SERIALIZABLE.
salida=$(psql_como vec_aspirantes_prueba_cruzada -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1');" 2>&1 || true)
grep -q 'denegado' <<<"$salida" || { echo "sesión cruzada: $salida"; exit 1; }
salida=$(psql_como vec_aspirantes_prueba_externa -c "SELECT public.probar_ficha('vec.aspirantes.ficha.consultar',0,'','$I1');" 2>&1 || true)
grep -q 'denegado' <<<"$salida" || { echo "sin SERIALIZABLE: $salida"; exit 1; }
salida=$(psql_pg -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_aspirantes.consultar_ficha_propia_v1('{}','x','x','x','x',1,1,'x','x','x','x');" 2>&1 || true)
grep -q 'denegado' <<<"$salida" || { echo "superusuario: $salida"; exit 1; }
# Contratación no alcanza el parseo del núcleo con un perfil de Aspirantes.
salida=$(psql_como vec_ct_prueba -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('aspirantes_ficha_consultar','{}','{}','x','x',1,1,convert_to('sonda_ct','UTF8'),'x','x','x');" 2>&1 || true)
grep -q 'sesión denegada' <<<"$salida" || { echo "guarda de sesión: $salida"; exit 1; }
echo "Aspirantes 000001 y AD3-110: PG18 OK"
if [[ ${1:-} == --go ]]; then
 psql_pg -c "CREATE ROLE vec_aspirantes_prueba_go LOGIN INHERIT NOBYPASSRLS; GRANT vec_aspirantes_ejecutor_externo TO vec_aspirantes_prueba_go WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;" >/dev/null
 puerto=$(docker port "$contenedor" 5432/tcp | head -1 | sed 's/.*://')
 (cd "$raiz" && VEC_ASPIRANTES_PG18_DSN="postgres://vec_aspirantes_prueba_go@127.0.0.1:$puerto/postgres?sslmode=disable" \
  go test -count=1 -v -run PG18 ./internal/modules/aspirantes/adapters/postgres/)
 echo "Aspirantes Go → PG18 OK"
fi

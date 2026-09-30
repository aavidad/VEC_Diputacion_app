\set ON_ERROR_STOP on
-- BORRADOR NO INSTALABLE. AD3-133 sustituirá hacia delante AD3-121/122.
-- Entrada causal: H6/AD132; roles Catálogos, Cat1, Cat2, AD117, Cat3 y
-- AD126 de #222 instaladas una sola vez. AD134/136 son posteriores.
-- Falta medir en PG18 la preimagen GLOBAL de ambos linajes completos: función,
-- cuerpo, definición, configuración, propietario, ACL, OID y dependencias;
-- CHECK de audiencias, membresías técnicas y TEMP de PUBLIC. Ninguna huella
-- de las candidatas anteriores sirve para esta postimagen.
-- La versión final sólo cambiará search_path de funciones pendientes a
-- pg_catalog,pg_temp. No cambiará cuerpos, concesiones, OID ni historia.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo', 0));
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000133', 0));
DO $borrador$
DECLARE
    nucleo pg_catalog.oid := pg_catalog.to_regprocedure(
      'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
    cuerpo pg_catalog.text;
    configuracion pg_catalog.text[];
    -- Las dos matrices contienen el conjunto completo y heterogéneo de los
    -- objetos reales, no combinaciones individuales de huellas. Se
    -- completan con el inventario PG18 exacto de cada linaje post-#222.
    matriz_a pg_catalog.jsonb := NULL;
    matriz_b pg_catalog.jsonb := NULL;
    pendientes_a pg_catalog.jsonb := NULL;
    pendientes_b pg_catalog.jsonb := NULL;
    frontera_a pg_catalog.jsonb := NULL;
    frontera_b pg_catalog.jsonb := NULL;
    frontera_actual pg_catalog.jsonb;
    matriz_elegida pg_catalog.jsonb;
    pendientes_elegidos pg_catalog.jsonb;
    actual pg_catalog.jsonb;
    filas pg_catalog.jsonb;
    e pg_catalog.jsonb;
    f pg_catalog.oid;
    antes pg_catalog.pg_proc%ROWTYPE;
    despues pg_catalog.pg_proc%ROWTYPE;
    dependencias_antes pg_catalog.jsonb;
    dependencias_despues pg_catalog.jsonb;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('server_version_num')::pg_catalog.int4
          NOT BETWEEN 180000 AND 189999
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR nucleo IS NULL THEN
        RAISE EXCEPTION 'AD3-133: migrador, PG18 o prefijo #222 ausente'
          USING ERRCODE = '55000';
    END IF;
    SELECT p.prosrc, p.proconfig INTO STRICT cuerpo, configuracion
      FROM pg_catalog.pg_proc AS p WHERE p.oid = nucleo;
    IF pg_catalog.encode(pg_catalog.sha256(
         pg_catalog.convert_to(cuerpo, 'UTF8')), 'hex')
         IS DISTINCT FROM '1542976c6948408364a36d71bea0066c33cc84ecff67ca1aacafdfc264eaeef0'
       OR configuracion IS DISTINCT FROM
          ARRAY['search_path=pg_catalog, pg_temp', 'lock_timeout=2s'] THEN
        RAISE EXCEPTION 'AD3-133: núcleo AD126 no coincide con #222'
          USING ERRCODE = '55000';
    END IF;
    -- Antes de permitir la postimagen, cotejar también en esta transacción el
    -- CHECK de audiencias, ACL de tipos compuestos, membresías y TEMP/PUBLIC
    -- medidos en ambas historias. La prueba focal obtiene el inventario íntegro.
    IF matriz_a IS NULL OR matriz_b IS NULL
       OR pendientes_a IS NULL OR pendientes_b IS NULL
       OR frontera_a IS NULL OR frontera_b IS NULL THEN
        RAISE EXCEPTION 'AD3-133: borrador sin matrices globales PG18 post-#222'
          USING ERRCODE = '55000';
    END IF;
SELECT pg_catalog.jsonb_agg(entrada ORDER BY entrada->>'firma') INTO actual FROM (
WITH sig(firma) AS (SELECT x->>'firma' FROM pg_catalog.jsonb_array_elements(matriz_a) x)
select jsonb_build_object(
 'firma',s.firma,'sha_prosrc',encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),
 'owner',pg_get_userbyid(p.proowner),'language',l.lanname,
 'argtypes',(select jsonb_agg(format_type(x,null) order by i) from unnest(p.proargtypes::oid[]) with ordinality t(x,i)),
 'allargtypes',(select jsonb_agg(format_type(x,null) order by i) from unnest(p.proallargtypes) with ordinality t(x,i)),
 'return',format_type(p.prorettype,null),'variadic',case when p.provariadic=0 then null else format_type(p.provariadic,null) end,
 'pg_proc',to_jsonb(p)-array['oid','pronamespace','proowner','prolang','proargtypes','proallargtypes','prorettype','provariadic','prosrc','proacl'],
 'acl_effective',coalesce((select jsonb_agg(jsonb_build_object('grantee',coalesce(gr.rolname,'PUBLIC'),'grantor',go.rolname,'privilege',a.privilege_type,'grantable',a.is_grantable) order by coalesce(gr.rolname,'PUBLIC'),go.rolname,a.privilege_type,a.is_grantable) from aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a left join pg_roles gr on gr.oid=a.grantee left join pg_roles go on go.oid=a.grantor),'[]'::jsonb),
 'pg_depend',coalesce((select jsonb_agg(jsonb_build_object('class',d.classid::regclass::text,'objsubid',d.objsubid,'refclass',d.refclassid::regclass::text,'refobjsubid',d.refobjsubid,'deptype',d.deptype,'target',to_jsonb(i)) order by d.classid::regclass::text,d.objsubid,d.refclassid::regclass::text,d.refobjsubid,d.deptype,i.identity) from pg_depend d cross join lateral pg_identify_object(d.refclassid,d.refobjid,d.refobjsubid) i where d.classid='pg_proc'::regclass and d.objid=p.oid),'[]'::jsonb),
 'pg_shdepend',coalesce((select jsonb_agg(jsonb_build_object('class',d.classid::regclass::text,'objsubid',d.objsubid,'refclass',d.refclassid::regclass::text,'refobjsubid',0,'deptype',d.deptype,'target',to_jsonb(i)) order by d.classid::regclass::text,d.objsubid,d.refclassid::regclass::text,d.deptype,i.identity) from pg_shdepend d cross join lateral pg_identify_object(d.refclassid,d.refobjid,0) i where d.classid='pg_proc'::regclass and d.objid=p.oid and d.dbid=(select oid from pg_database where datname=current_database())),'[]'::jsonb)
) from sig s join pg_proc p on p.oid=to_regprocedure(s.firma) join pg_language l on l.oid=p.prolang
) AS conjunto(entrada);
    IF pg_catalog.jsonb_array_length(actual) IS DISTINCT FROM
       pg_catalog.jsonb_array_length(matriz_a)
       OR pg_catalog.jsonb_array_length(matriz_a) IS DISTINCT FROM
          pg_catalog.jsonb_array_length(matriz_b)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
           WHERE p.pronamespace=pg_catalog.to_regnamespace(
             'vec_autorizacion_atestada_v3'))
          IS DISTINCT FROM pg_catalog.jsonb_array_length(matriz_a)
       OR (actual IS DISTINCT FROM matriz_a AND actual IS DISTINCT FROM matriz_b) THEN
        RAISE EXCEPTION 'AD3-133: preimagen global incompatible'
          USING ERRCODE = '55000';
    END IF;
    SELECT pg_catalog.jsonb_build_object(
      'audiencias',(SELECT pg_catalog.jsonb_build_object(
        'definicion',pg_catalog.pg_get_constraintdef(c.oid,true),
        'metadatos',pg_catalog.to_jsonb(c)-ARRAY[
          'oid','connamespace','conrelid','conbin'])
        FROM pg_catalog.pg_constraint AS c
        WHERE c.conrelid=pg_catalog.to_regclass(
          'vec_autorizacion_atestada_v3.clave_capacidad_version')
          AND c.conname='clave_capacidad_version_audiencia_consumo_check'),
      'tipos',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'nombre',t.typname,'propietario',pg_catalog.pg_get_userbyid(t.typowner),
        'acl',pg_catalog.coalesce((SELECT pg_catalog.jsonb_agg(
           pg_catalog.jsonb_build_object(
             'grantee',pg_catalog.coalesce(r.rolname,'PUBLIC'),
             'privilege',a.privilege_type,'grantable',a.is_grantable)
           ORDER BY pg_catalog.coalesce(r.rolname,'PUBLIC'),
                    a.privilege_type,a.is_grantable)
           FROM pg_catalog.aclexplode(pg_catalog.coalesce(
             t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
           LEFT JOIN pg_catalog.pg_roles AS r ON r.oid=a.grantee),
           '[]'::pg_catalog.jsonb)) ORDER BY t.typname)
         FROM pg_catalog.pg_type AS t
         WHERE t.typnamespace=pg_catalog.to_regnamespace(
           'vec_autorizacion_atestada_v3') AND t.typtype='c'),
      'membresias',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'rol',g.rolname,'miembro',m.rolname,'admin',a.admin_option,
        'inherit',a.inherit_option,'set',a.set_option)
        ORDER BY g.rolname,m.rolname)
        FROM pg_catalog.pg_auth_members AS a
        JOIN pg_catalog.pg_roles AS g ON g.oid=a.roleid
        JOIN pg_catalog.pg_roles AS m ON m.oid=a.member
        WHERE pg_catalog.left(g.rolname,4)='vec_'
           OR pg_catalog.left(m.rolname,4)='vec_'),
      'public_temp',(SELECT pg_catalog.coalesce(pg_catalog.bool_or(
        a.grantee=0 AND a.privilege_type='TEMPORARY'),false)
        FROM pg_catalog.pg_database AS d
        LEFT JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
          d.datacl,pg_catalog.acldefault('d',d.datdba))) AS a ON true
        WHERE d.datname=pg_catalog.current_database())) INTO frontera_actual;
    IF frontera_actual->'audiencias' IS NULL
       OR frontera_actual->'audiencias'='null'::pg_catalog.jsonb
       OR frontera_actual->>'public_temp' IS DISTINCT FROM 'false'
       OR frontera_actual IS DISTINCT FROM
          CASE WHEN actual=matriz_a THEN frontera_a ELSE frontera_b END THEN
       RAISE EXCEPTION 'AD3-133: CHECK, tipos, membresías o TEMP incompatibles'
         USING ERRCODE='55000';
    END IF;
    matriz_elegida := CASE WHEN actual=matriz_a THEN matriz_a ELSE matriz_b END;
    pendientes_elegidos := CASE WHEN actual=matriz_a THEN pendientes_a ELSE pendientes_b END;
    IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements_text(pendientes_elegidos) AS x(firma)
               WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(matriz_elegida) AS m(obj)
                                 WHERE m.obj->>'firma'=x.firma))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p
                  CROSS JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
                    p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
                  WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
                    AND (a.grantee=0 OR p.proowner<>
                      'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_type AS t
                  CROSS JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
                    t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
                  WHERE t.typnamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
                    AND t.typtype='c' AND a.grantee=0) THEN
       RAISE EXCEPTION 'AD3-133: firmas pendientes, propietario o ACL incompatibles'
         USING ERRCODE='55000';
    END IF;
 -- Capturar el catálogo completo con sus OID locales antes del primer ALTER.
 SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(p) ORDER BY p.oid)
 INTO filas FROM pg_catalog.pg_proc p
 WHERE p.oid IN(SELECT pg_catalog.to_regprocedure(x->>'firma') FROM pg_catalog.jsonb_array_elements(matriz_elegida) x);
 SELECT pg_catalog.jsonb_build_object(
 'locales',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(filas) x)),
 'compartidas',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
 FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(filas) x))) INTO dependencias_antes;
 FOR e IN SELECT x FROM pg_catalog.jsonb_array_elements(filas) x LOOP
  f:=(e->>'oid')::pg_catalog.oid;
  SELECT p.* INTO STRICT antes FROM pg_catalog.pg_proc p WHERE p.oid=f;
  IF pg_catalog.to_jsonb(antes) IS DISTINCT FROM e THEN
   RAISE EXCEPTION 'AD3-133: preimagen concurrente incompatible' USING ERRCODE='55000';
  END IF;
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements_text(pendientes_elegidos) x
                WHERE x=f::pg_catalog.regprocedure::pg_catalog.text) THEN
   IF NOT (antes.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']) THEN
    RAISE EXCEPTION 'AD3-133: función fuera del cambio no cerrada' USING ERRCODE='55000';
   END IF;
   CONTINUE; -- AD126 ya cerró el núcleo: no se reaplica el ALTER.
  END IF;
  IF NOT (antes.proconfig @> ARRAY['search_path=pg_catalog']) THEN
   RAISE EXCEPTION 'AD3-133: search_path pendiente incompatible' USING ERRCODE='55000';
  END IF;
  SELECT pg_catalog.array_agg(CASE WHEN opcion='search_path=pg_catalog' THEN 'search_path=pg_catalog, pg_temp' ELSE opcion END ORDER BY indice)
  INTO configuracion FROM pg_catalog.unnest(antes.proconfig) WITH ORDINALITY AS opciones(opcion,indice);
  EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',f::pg_catalog.regprocedure);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=f;
  IF pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM e-'proconfig'
  OR despues.proconfig IS DISTINCT FROM configuracion THEN
   RAISE EXCEPTION 'AD3-133: postimagen incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
 SELECT pg_catalog.jsonb_build_object(
 'locales',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(filas) x)),
 'compartidas',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
 FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(filas) x))) INTO dependencias_despues;
 IF dependencias_despues IS DISTINCT FROM dependencias_antes THEN
  RAISE EXCEPTION 'AD3-133: dependencias alteradas' USING ERRCODE='55000';
 END IF;
END $borrador$;
COMMIT;

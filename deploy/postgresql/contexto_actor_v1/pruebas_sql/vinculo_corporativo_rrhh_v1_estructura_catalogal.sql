-- Contrato catalogal focal C2.2-B1. Se ejecuta despues de 000004 up.
SET search_path = pg_catalog;
SET timezone = 'UTC';

DO $contrato_catalogal$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid :=
      'vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass;
    actual constant oid :=
      'vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass;
    rol oid;
BEGIN
    IF (SELECT count(*) FROM pg_class
         WHERE oid IN (versiones,actual) AND relkind='r'
           AND relpersistence='p' AND relowner=propietario
           AND relreplident='d'
           AND relrowsecurity AND relforcerowsecurity) <> 2 THEN
        RAISE EXCEPTION 'tablas, propiedad o RLS no exactos';
    END IF;

    IF (SELECT count(*) FROM pg_attribute WHERE attrelid=versiones
         AND attnum>0 AND NOT attisdropped) <> 25
       OR (SELECT count(*) FROM pg_attribute WHERE attrelid=actual
            AND attnum>0 AND NOT attisdropped) <> 5
       OR (SELECT count(*) FROM pg_constraint WHERE conrelid=versiones) <> 50
       OR (SELECT count(*) FROM pg_constraint WHERE conrelid=actual) <> 10
       OR (SELECT count(*) FROM pg_constraint
            WHERE conrelid IN (versiones,actual)) <> 60
       OR (SELECT count(*) FROM pg_constraint
            WHERE connamespace=esquema AND conname IN
              ('perfil_versiones_persona_uq',
               'vinculo_contexto_versiones_actor_uq')) <> 2
       OR (SELECT count(*) FROM pg_constraint
            WHERE connamespace=esquema AND (
              (conrelid IN (versiones,actual) AND contype <> 'n')
              OR conname IN ('perfil_versiones_persona_uq',
                             'vinculo_contexto_versiones_actor_uq')
            )) <> 32 THEN
        RAISE EXCEPTION 'columnas o restricciones no exactas';
    END IF;

    IF (SELECT count(*) FROM pg_constraint
         WHERE conrelid IN (versiones,actual) AND contype='f'
           AND confmatchtype='f' AND confupdtype='a' AND confdeltype='a'
           AND NOT condeferrable AND NOT condeferred AND convalidated) <> 7
       OR (SELECT count(*) FROM pg_index
            WHERE indrelid IN (versiones,actual)) <> 3
       OR (SELECT count(*) FROM pg_index i
            JOIN pg_class x ON x.oid=i.indexrelid
            JOIN pg_am am ON am.oid=x.relam
            JOIN (VALUES
              ('vinculo_corporativo_versiones_pk',true,
               'vinculo_corporativo_ref,version'),
              ('vinculo_corporativo_versiones_actual_uq',false,
               'cuenta_ref,superficie,uso,vinculo_corporativo_ref,version'),
              ('vinculo_corporativo_actual_pk',true,'cuenta_ref,superficie,uso'),
              ('perfil_versiones_persona_uq',false,
               'perfil_ref,version,persona_ref'),
              ('vinculo_contexto_versiones_actor_uq',false,
               'vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref')
            ) e(nombre,primario,columnas) ON e.nombre=x.relname
            WHERE i.indrelid IN (versiones,actual,
              'vec_contexto_actor_v1.perfil_versiones'::regclass,
              'vec_contexto_actor_v1.vinculo_contexto_versiones'::regclass)
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.relpersistence='p' AND x.relreplident='n'
              AND x.reloptions IS NULL AND am.amname='btree'
              AND i.indisunique AND NOT i.indnullsnotdistinct
              AND i.indisprimary=e.primario AND NOT i.indisexclusion
              AND i.indimmediate AND NOT i.indisclustered AND i.indisvalid
              AND NOT i.indcheckxmin AND i.indisready AND i.indislive
              AND NOT i.indisreplident AND i.indnkeyatts=i.indnatts
              AND (SELECT string_agg(pg_get_indexdef(
                    i.indexrelid,posicion,false),',' ORDER BY posicion)
                     FROM generate_series(1,i.indnatts) posicion)=e.columnas) <> 5
       OR (SELECT count(*) FROM pg_trigger
            WHERE tgrelid IN (versiones,actual) AND NOT tgisinternal) <> 5
       OR (SELECT count(*) FROM pg_trigger
            WHERE tgrelid IN (versiones,actual) AND tgisinternal) <> 16 THEN
        RAISE EXCEPTION 'indices, FKs o disparadores no exactos';
    END IF;

    IF (SELECT count(*) FROM pg_policy
         WHERE polrelid IN (versiones,actual)
           AND polname='acceso_propietario_exacto' AND polcmd='*'
           AND polpermissive AND polroles=ARRAY[propietario]::oid[]
           AND polqual IS NOT NULL AND polwithcheck IS NOT NULL) <> 2
       OR EXISTS (SELECT 1 FROM pg_attribute
                   WHERE attrelid IN (versiones,actual) AND attnum>0
                     AND NOT attisdropped AND attacl IS NOT NULL)
       OR EXISTS (
         SELECT 1 FROM pg_class c,
              LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
          WHERE c.oid IN (versiones,actual) AND a.grantee=0
            AND a.privilege_type IN ('SELECT','INSERT','UPDATE','DELETE','TRUNCATE')
       )
       OR has_table_privilege('vec_contexto_actor_v1_runtime',versiones,'SELECT')
       OR has_table_privilege('vec_contexto_actor_v1_runtime',actual,'SELECT') THEN
        RAISE EXCEPTION 'politicas o ACL no exactas';
    END IF;

    IF EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid IN (versiones,actual)
         AND a.attnum>0 AND NOT a.attisdropped AND (a.attacl IS NOT NULL
           OR a.atthasdef OR a.attstorage<>(SELECT typstorage FROM pg_type
                                            WHERE oid=a.atttypid)
           OR a.attcompression<>''))
       OR (SELECT count(*) FROM pg_type t WHERE t.oid IN (
            (SELECT reltype FROM pg_class WHERE oid=versiones),
            (SELECT reltype FROM pg_class WHERE oid=actual),
            (SELECT typarray FROM pg_type WHERE typrelid=versiones),
            (SELECT typarray FROM pg_type WHERE typrelid=actual))
            AND t.typowner=propietario AND (
              (t.typrelid IN (versiones,actual) AND NOT EXISTS (
                SELECT 1 FROM aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
                 WHERE a.grantee<>propietario))
              OR (t.typrelid=0 AND t.typacl IS NULL AND t.typelem IN (
                (SELECT reltype FROM pg_class WHERE oid=versiones),
                (SELECT reltype FROM pg_class WHERE oid=actual))))) <> 4
       OR has_type_privilege('vec_contexto_actor_v1_runtime',
            (SELECT reltype FROM pg_class WHERE oid=versiones),'USAGE')
       OR has_type_privilege('vec_contexto_actor_v1_runtime',
            (SELECT typarray FROM pg_type WHERE typrelid=versiones),'USAGE') THEN
        RAISE EXCEPTION 'columnas o tipos no exactos';
    END IF;

    IF (SELECT count(*) FROM pg_class t JOIN pg_class p ON p.reltoastrelid=t.oid
         JOIN pg_am am ON am.oid=t.relam WHERE p.oid IN (versiones,actual)
           AND t.relkind='t' AND t.relowner=propietario AND t.reltablespace=0
           AND t.relpersistence='p' AND t.relreplident='n'
           AND t.reloptions IS NULL AND am.amname='heap') <> 2
       OR (SELECT count(*) FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid
            JOIN pg_class x ON x.oid=i.indexrelid JOIN pg_am am ON am.oid=x.relam
            WHERE t.oid IN (
              (SELECT reltoastrelid FROM pg_class WHERE oid=versiones),
              (SELECT reltoastrelid FROM pg_class WHERE oid=actual))
              AND i.indisunique AND NOT i.indnullsnotdistinct
              AND i.indisprimary AND NOT i.indisexclusion AND i.indimmediate
              AND NOT i.indisclustered AND i.indisvalid AND NOT i.indcheckxmin
              AND i.indisready AND i.indislive AND NOT i.indisreplident
              AND i.indnkeyatts=2 AND i.indnatts=2
              AND (SELECT string_agg(pg_get_indexdef(
                    i.indexrelid,posicion,false),',' ORDER BY posicion)
                     FROM generate_series(1,i.indnatts) posicion)
                  = 'chunk_id,chunk_seq'
              AND x.relowner=propietario AND x.reltablespace=0
              AND x.relpersistence='p' AND x.relreplident='n'
              AND x.reloptions IS NULL AND am.amname='btree') <> 2 THEN
        RAISE EXCEPTION 'TOAST o sus indices no exactos';
    END IF;

    FOR rol IN SELECT oid FROM pg_roles WHERE rolname IN (
      'vec_contexto_actor_v1_runtime',
      'vec_contexto_actor_corporativo_rrhh_selector','c22b_consumidor',
      'c22b_login','c22b_pdp','c22b_autorizacion','c22b_contratacion','c22b_bolsa'
    ) LOOP
      IF has_table_privilege(rol,versiones,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
         OR has_table_privilege(rol,actual,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
         OR has_type_privilege(rol,(SELECT reltype FROM pg_class WHERE oid=versiones),'USAGE')
         OR has_type_privilege(rol,(SELECT reltype FROM pg_class WHERE oid=actual),'USAGE')
         OR has_type_privilege(rol,(SELECT typarray FROM pg_type WHERE typrelid=versiones),'USAGE')
         OR has_type_privilege(rol,(SELECT typarray FROM pg_type WHERE typrelid=actual),'USAGE') THEN
        RAISE EXCEPTION 'acceso efectivo inesperado para rol %', rol::regrole;
      END IF;
    END LOOP;
END
$contrato_catalogal$;

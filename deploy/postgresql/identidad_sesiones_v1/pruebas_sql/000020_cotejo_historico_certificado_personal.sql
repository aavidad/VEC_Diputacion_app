\set ON_ERROR_STOP on
-- Solo en clon PG18 con IS20 instalada; lecturas de historia real y ROLLBACK.
-- No crea autenticaciones, políticas, actos ni recibos de muestra.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
DO $test$
DECLARE
 f oid:='vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)'::regprocedure;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
 a oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 r record; interna jsonb; admin jsonb; alterada jsonb;
 politica_ref text; politica_huella text; verificada_en timestamptz;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=o
       AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
                             'statement_timeout=5s','lock_timeout=2s']::text[])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x WHERE p.oid=f
          AND (x.grantee NOT IN(o,a) OR x.grantor<>o
               OR x.privilege_type<>'EXECUTE' OR x.is_grantable))
    OR has_function_privilege(a,f,'EXECUTE') IS NOT TRUE
    OR has_function_privilege('vec_identidad_sesiones_v1_revalidador',f,'EXECUTE')
 THEN RAISE EXCEPTION 'IS20: ACL o definición distinta'; END IF;

 -- El vínculo canónico viene de decisiones durables; no se reconstruye a mano.
 FOR r IN
   SELECT d.vinculo
     FROM (SELECT convert_from(decision_canonica,'UTF8')::jsonb
                    ->'vinculo_autenticacion_actor' AS vinculo
             FROM vec_autorizacion.decision_concedida_contexto_actor_v3) d
     JOIN vec_autorizacion.sesion_autenticacion_v1 b
       ON b.autenticacion_ref=d.vinculo->>'autenticacion_ref'
      AND b.sesion_ref=d.vinculo->>'sesion_ref'
      AND b.autenticacion_huella_sha256=d.vinculo->>'autenticacion_huella_sha256'
    WHERE d.vinculo->>'superficie'='interna_corporativa'
      AND d.vinculo->>'metodo_observado'='certificado'
      AND d.vinculo->>'garantia_observada'='sustancial'
      AND d.vinculo->>'cuenta_privilegiada'='false'
      AND vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
          b.politica_garantia_ref,b.politica_garantia_huella_sha256,
          b.autenticacion_verificada_en) IS TRUE
    ORDER BY b.sesion_emitida_en DESC
 LOOP
   BEGIN
     IF vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
          convert_to(r.vinculo::text,'UTF8')) IS TRUE THEN
       interna:=r.vinculo;
       EXIT;
     END IF;
   EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL;
   END;
 END LOOP;
 IF interna IS NULL THEN RAISE EXCEPTION 'IS20: falta sesión interna nominal apta'; END IF;
 politica_ref:=interna->>'politica_garantia_ref';
 politica_huella:=interna->>'politica_garantia_huella_sha256';
 verificada_en:=(interna->>'autenticacion_verificada_en')::timestamptz;
 IF vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
      politica_ref,politica_huella,verificada_en) IS NOT TRUE
    OR vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
      'pga_'||repeat('9',22),politica_huella,verificada_en) IS NOT FALSE
    OR vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
      politica_ref,repeat('f',64),verificada_en) IS NOT FALSE
    OR vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
      politica_ref,politica_huella,verificada_en-interval '200 years') IS NOT FALSE
 THEN RAISE EXCEPTION 'IS20: política histórica no discrimina referencia, huella o fecha'; END IF;

 alterada:=jsonb_set(interna,'{politica_garantia_ref}',to_jsonb('pga_'||repeat('9',22)));
 BEGIN
   PERFORM vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
     convert_to(alterada::text,'UTF8'));
   RAISE EXCEPTION 'IS20: aceptó política alterada';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL; END;
 alterada:=jsonb_set(interna,'{politica_garantia_huella_sha256}',to_jsonb(repeat('f',64)));
 BEGIN
   PERFORM vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
     convert_to(alterada::text,'UTF8'));
   RAISE EXCEPTION 'IS20: aceptó huella alterada';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL; END;
 alterada:=jsonb_set(interna,'{autenticacion_verificada_en}',
   to_jsonb((verificada_en-interval '200 years')::text));
 BEGIN
   PERFORM vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
     convert_to(alterada::text,'UTF8'));
   RAISE EXCEPTION 'IS20: aceptó fecha alterada';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL; END;

 -- ADMIN conserva garantía alta. Se coteja una sesión ADMIN original y se
 -- rechaza un vínculo alterado que pretendiera presentarla como sustancial.
 FOR r IN
   SELECT convert_from(decision_canonica,'UTF8')::jsonb
          ->'vinculo_autenticacion_actor' AS vinculo
     FROM vec_autorizacion.decision_concedida_contexto_actor_v3
    WHERE convert_from(decision_canonica,'UTF8')::jsonb
          #>>'{vinculo_autenticacion_actor,superficie}'='administracion_privilegiada'
    ORDER BY registrada_en DESC
 LOOP
   BEGIN
     IF vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
          convert_to(r.vinculo::text,'UTF8')) IS TRUE THEN
       admin:=r.vinculo;
       EXIT;
     END IF;
   EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL;
   END;
 END LOOP;
 IF admin IS NULL OR admin->>'garantia_observada' IS DISTINCT FROM 'alto'
    OR admin->>'cuenta_privilegiada' IS DISTINCT FROM 'true' THEN
   RAISE EXCEPTION 'IS20: falta sesión ADMIN alta apta';
 END IF;
 alterada:=jsonb_set(admin,'{garantia_observada}','"sustancial"'::jsonb);
 BEGIN
   PERFORM vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
     convert_to(alterada::text,'UTF8'));
   RAISE EXCEPTION 'IS20: aceptó ADMIN sustancial';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P0002' THEN NULL; END;
END $test$;
ROLLBACK;

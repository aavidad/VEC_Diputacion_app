\set ON_ERROR_STOP on
-- Comprobación estructural sobre el clon aislado autorizado, después del UP.
-- No fabrica permisos, decisiones ni material de autorización V3.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $acl$ DECLARE t text; f record; BEGIN
 FOREACH t IN ARRAY ARRAY['hecho_identidad','hecho_version','operacion','auditoria_operacion','outbox','acceso_operacion'] LOOP
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_meritos.'||t) AND c.relrowsecurity AND c.relforcerowsecurity AND c.relowner='vec_meritos_propietario'::regrole)
 OR has_table_privilege('vec_meritos_ejecutor','vec_meritos.'||t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 OR EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=to_regclass('vec_meritos.'||t) AND 0=ANY(p.polroles))
 THEN RAISE EXCEPTION 'test: tabla con ACL/RLS incompatible %',t; END IF;
 END LOOP;
 FOR f IN SELECT p.* FROM pg_proc p WHERE p.pronamespace='vec_meritos'::regnamespace LOOP
 IF EXISTS (SELECT 1 FROM aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
 OR (f.prosecdef AND (f.proowner<>'vec_meritos_propietario'::regrole OR NOT 'search_path=pg_catalog, pg_temp'=ANY(f.proconfig)))
 THEN RAISE EXCEPTION 'test: función con ACL/config incompatible %',f.proname; END IF;
 IF f.proname<>'operar_hecho_v1' AND has_function_privilege('vec_meritos_ejecutor',f.oid,'EXECUTE') THEN
 RAISE EXCEPTION 'test: ejecutor accede a función auxiliar %',f.proname; END IF;
 END LOOP;
END $acl$;
SET LOCAL ROLE vec_meritos_propietario;
DO $estructura$
DECLARE h jsonb; o jsonb; m jsonb; mala jsonb; valor jsonb; n integer;
BEGIN
 h:='{"referencia":"hecho:ensayo","persona_ref":"persona:externa","version":1,"tipo":"curso_superacion","concepto_ref":"concepto:curso","denominacion":"Curso sintético","horas":12,"procedencia":{"fuente_ref":"fuente:ensayo","version":"v1","hecho_origen_ref":"origen:curso","capturada_en":"2026-10-01T12:00:00Z"},"vigencia":{"desde":"2026-10-01"},"estado":"declarado","evidencias":[]}'::jsonb;
 m:=jsonb_build_object('catalogo_id','catalogo:motivos','catalogo_version',1,'catalogo_huella_sha256',repeat('a',64),'entrada_clave','declaracion');
 o:=jsonb_build_object('esquema','vec.meritos.hecho.operacion.v1','accion','meritos.hecho.declarar','actor_ref','persona:externa','clave_idempotencia','clave:ensayo','version_esperada',0,'hecho',h,'motivo',m,'fecha_corte','2026-10-01');
 valor:=vec_meritos.validar_comando_v1(convert_to(o::text,'UTF8'));
 IF valor<>o THEN RAISE EXCEPTION 'test: comando válido cambiado'; END IF;
 -- Referencia documental válida no permite acreditar un hecho.
 FOR mala IN SELECT value FROM jsonb_array_elements(jsonb_build_array(
 jsonb_set(o,'{hecho,estado}','"acreditado"'),
 jsonb_set(o,'{hecho,revision}','{"referencia":"revision:propia","actor_ref":"persona:externa","motivo_ref":"motivo:propio","fecha":"2026-10-01T12:00:00Z"}'),
 jsonb_set(o,'{hecho,evidencias}','[{"id":"documento:uno","version":1},{"id":"documento:uno","version":1}]'),
 jsonb_set(o,'{hecho,procedencia,capturada_en}','"2026-02-30T12:00:00Z"'),
 jsonb_set(o,'{hecho,vigencia,desde}','"2026-02-30"'),
 jsonb_set(o,'{hecho,persona_ref}','null'),
 jsonb_set(o,'{hecho,persona_ref}','1'),
 jsonb_set(o,'{hecho,tipo}','null'),
 jsonb_set(o,'{hecho,version}','2'),
 jsonb_set(o,'{hecho,puntos}','100'),
 jsonb_set(o,'{motivo,catalogo_huella_sha256}','"invalida"'),
 jsonb_set(o,'{version_esperada}','-1'),
 jsonb_set(o,'{accion}','"meritos.hecho.verificar"')
 )) LOOP
 BEGIN
 PERFORM vec_meritos.validar_comando_v1(convert_to(mala::text,'UTF8'));
 RAISE EXCEPTION 'test: aceptó comando inválido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 END LOOP;
 -- Sin contexto nominal, incluso el dueño no lee ni añade historia.
 SELECT count(*) INTO n FROM vec_meritos.hecho_identidad;
 IF n<>0 THEN RAISE EXCEPTION 'test: RLS ausente sin contexto'; END IF;
 BEGIN
 INSERT INTO vec_meritos.hecho_identidad(hecho_ref,persona_ref,fuente_ref,origen_ref,tipo,declarante_ref)
 VALUES('hecho:ensayo','persona:externa','fuente:ensayo','origen:curso','curso_superacion','persona:externa');
 RAISE EXCEPTION 'test: RLS aceptó alta sin contexto';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $estructura$;
ROLLBACK;
SELECT 'MERITOS-ESTRUCTURA-ACL-OK';

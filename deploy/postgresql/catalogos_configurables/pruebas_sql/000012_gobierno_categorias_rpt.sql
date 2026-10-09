\set ON_ERROR_STOP on
-- Prueba transaccional en PostgreSQL 18 desechable: CC1, roles CC12, CC12.
-- No consume V3: AD232 prueba esa frontera por separado.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='2min';
DO $test$
DECLARE
 a constant text:='principal:preparador-cc12'; b constant text:='principal:revisor-cc12';
 perfil_a constant text:='perfil:preparacion-cc12'; perfil_b constant text:='perfil:revision-cc12';
 conf_sha text; conf_sha2 text; conf_sha_alterna text; conf_recibo jsonb; conf_replay jsonb; desde timestamptz(6):=pg_catalog.clock_timestamp()-interval '1 day';
 hasta timestamptz(6):=pg_catalog.clock_timestamp()+interval '1 day';
 doc text; contenido jsonb; meta jsonb; prep jsonb; h text; res jsonb; repetido jsonb;
 v_antes bigint; fuente bytea:=pg_catalog.convert_to('fuente privada sintetica de ejercicio','UTF8');
BEGIN
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid='vec_catalogos_configurables.efectuar_gobierno_rpt(text,text,jsonb,bytea,jsonb,text,bigint,text,text,text,text,text,text)'::pg_catalog.regprocedure
     AND x.grantee=0 AND x.privilege_type='EXECUTE')
   OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
   'vec_catalogos_configurables.efectuar_gobierno_rpt(text,text,jsonb,bytea,jsonb,text,bigint,text,text,text,text,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'CC12: ACL incorrecta'; END IF;
 IF pg_catalog.has_function_privilege('vec_catalogos_configurables_rpt_consumidor',
   'vec_catalogos_configurables.efectuar_gobierno_rpt(text,text,jsonb,bytea,jsonb,text,bigint,text,text,text,text,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'CC12: grupo tecnico puede efectuar sin V3'; END IF;
 IF pg_catalog.has_function_privilege('vec_catalogos_configurables_rpt_consumidor',
   'vec_catalogos_configurables.provisionar_perfiles_rpt(text,text,bigint,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,boolean,text)','EXECUTE') THEN
  RAISE EXCEPTION 'CC12: grupo tecnico puede proveer perfiles sin V3'; END IF;
 IF pg_catalog.pg_has_role('vec_autorizacion_atestada_v3_propietario',
   'vec_catalogos_configurables_rpt_consumidor','MEMBER') THEN
  RAISE EXCEPTION 'CC12: membresia AD3 impropia'; END IF;
 doc:=pg_catalog.jsonb_build_object('id','rpt.categoria.ejercicio','version',1,'modulo_id','personal',
   'fuente_ref','fuente:ejercicio-cc12','estado','publicado','creado_por',a,'publicado_por',b,
   'entradas',pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('clave','categoria.ejercicio',
     'etiqueta','Categoria de ejercicio','atributos',pg_catalog.jsonb_build_object(
       'estado','habilitada','organizacion_ref','organizacion:ejercicio-cc12'))))::text;
 contenido:=pg_catalog.jsonb_build_object('accion','publicar','catalogo_id','rpt.categoria.ejercicio',
   'modulo_id','personal','version',1,'documento_canonico',doc,
   'documento_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc,'UTF8')),'hex'),
   'preimagenes_control','{}'::jsonb,'preimagenes_huella_sha256','',
   'fuente_ref','fuente:ejercicio-cc12','motivo_ref','motivos:1:ejercicio_cc12');
 meta:=pg_catalog.jsonb_build_object('fuente_ref','fuente:ejercicio-cc12','clase','ejercicio',
   'procedencia_ref','procedencia:ejercicio-cc12','custodia_ref','custodia:ejercicio-cc12',
   'organizacion_ref','organizacion:ejercicio-cc12','vigente_desde',desde,'vigente_hasta',hasta,
   'sha256',pg_catalog.encode(pg_catalog.sha256(fuente),'hex'));
 prep:=vec_catalogos_configurables.preparar_propuesta_rpt(contenido,fuente,meta);
 contenido:=prep->'contenido'; h:=prep->>'huella_sha256';
 BEGIN
  PERFORM vec_catalogos_configurables.efectuar_gobierno_rpt('proponer','propuesta:cc12',contenido,fuente,meta,
    h,0,a,perfil_a,'decision:cc12-a',repeat('a',64),'recibo:cc12-a','motivos:1:ejercicio_cc12');
  RAISE EXCEPTION 'CC12: sin competencia se admitio propuesta';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- Fixture sintético en transacción revertida: no es autoridad administrativa.
 conf_sha:=vec_catalogos_configurables.preparar_provision_perfiles_rpt(perfil_a,perfil_b,1,
   'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12',desde,hasta,true);
 BEGIN
  PERFORM vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',NULL,'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12',repeat('e',64),'recibo:perfil-cc12',
    desde,hasta,true,conf_sha);
  RAISE EXCEPTION 'CC12: fuente sin huella aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 conf_recibo:=vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12',repeat('e',64),'recibo:perfil-cc12',
    desde,hasta,true,conf_sha);
 conf_replay:=vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12-nueva',repeat('d',64),'recibo:perfil-cc12',
    desde,hasta,true,conf_sha);
 IF conf_recibo->>'recibo_ref'<>conf_replay->>'recibo_ref'
    OR conf_recibo->>'registrada_en'<>conf_replay->>'registrada_en'
    OR (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_perfiles_historia)<>1
    OR (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_perfiles_outbox)<>1 THEN
  RAISE EXCEPTION 'CC12: replay de provision duplicado'; END IF;
 conf_sha_alterna:=vec_catalogos_configurables.preparar_provision_perfiles_rpt(perfil_a,perfil_b,1,
   'fuente:perfil-cc12',repeat('a',64),'custodia:perfil-cc12','acto:perfil-cc12',desde,hasta,true);
 BEGIN
  PERFORM vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',repeat('a',64),'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12-otra',repeat('d',64),'recibo:perfil-cc12',
    desde,hasta,true,conf_sha_alterna);
  RAISE EXCEPTION 'CC12: recibo reutilizado con otra fuente';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12-conflicto',repeat('c',64),'recibo:perfil-cc12-conflicto',
    desde,hasta,true,conf_sha);
  RAISE EXCEPTION 'CC12: CAS obsoleto aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 res:=vec_catalogos_configurables.efectuar_gobierno_rpt('proponer','propuesta:cc12',contenido,fuente,meta,
    h,0,a,perfil_a,'decision:cc12-a',repeat('a',64),'recibo:cc12-a','motivos:1:ejercicio_cc12');
 IF res->>'estado'<>'propuesta' OR (res->>'revision')::bigint<>1 THEN RAISE EXCEPTION 'CC12: propuesta incorrecta'; END IF;
 BEGIN
  UPDATE vec_catalogos_configurables.rpt_propuesta SET fuente_bytes=pg_catalog.convert_to('otra fuente','UTF8')
   WHERE propuesta_ref='propuesta:cc12';
  RAISE EXCEPTION 'CC12: fuente mutable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.preparar_propuesta_rpt(
    pg_catalog.jsonb_set(contenido,'{accion}','"deshabilitar"'::jsonb),fuente,meta);
  RAISE EXCEPTION 'CC12: deshabilitacion no abierta en C1';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 repetido:=vec_catalogos_configurables.efectuar_gobierno_rpt('proponer','propuesta:cc12',contenido,fuente,meta,
    h,0,a,perfil_a,'decision:cc12-a2',repeat('b',64),'recibo:cc12-a2','motivos:1:ejercicio_cc12');
 IF repetido->>'recibo_ref'<>'recibo:cc12-a' THEN RAISE EXCEPTION 'CC12: replay de propuesta duplicado'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.efectuar_gobierno_rpt('aprobar','propuesta:cc12',NULL,NULL,NULL,
    h,1,a,perfil_b,'decision:cc12-mal',repeat('c',64),'recibo:cc12-mal','motivos:1:ejercicio_cc12');
  RAISE EXCEPTION 'CC12: autoaprobacion admitida';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 res:=vec_catalogos_configurables.efectuar_gobierno_rpt('aprobar','propuesta:cc12',NULL,NULL,NULL,
    h,1,b,perfil_b,'decision:cc12-b',repeat('b',64),'recibo:cc12-b','motivos:1:ejercicio_cc12');
 IF res->>'estado'<>'aprobada' THEN RAISE EXCEPTION 'CC12: aprobacion incorrecta'; END IF;
 SELECT pg_catalog.count(*) INTO v_antes FROM vec_catalogos_configurables.rpt_gobierno_outbox;
 res:=vec_catalogos_configurables.efectuar_gobierno_rpt('confirmar','propuesta:cc12',NULL,NULL,NULL,
    h,2,b,perfil_b,'decision:cc12-c',repeat('c',64),'recibo:cc12-c','motivos:1:ejercicio_cc12');
 IF res->>'estado'<>'confirmada' OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion
   WHERE catalogo_id='rpt.categoria.ejercicio' AND recibo_ref='recibo:cc12-c') THEN
  RAISE EXCEPTION 'CC12: publicacion no atomica'; END IF;
 repetido:=vec_catalogos_configurables.efectuar_gobierno_rpt('confirmar','propuesta:cc12',NULL,NULL,NULL,
    h,2,b,perfil_b,'decision:cc12-c2',repeat('d',64),'recibo:cc12-c2','motivos:1:ejercicio_cc12');
 IF repetido->>'recibo_ref'<>'recibo:cc12-c' OR
    (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_gobierno_outbox)<>v_antes+1 OR
    (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_gobierno_historia)<>3 THEN
  RAISE EXCEPTION 'CC12: confirmacion duplicada'; END IF;
 conf_sha2:=vec_catalogos_configurables.preparar_provision_perfiles_rpt(perfil_a,perfil_b,2,
   'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12-v2',desde,hasta,true);
 PERFORM vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,1,conf_sha,
   'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12-v2',
   'principal:administrador-cc12','decision:perfil-cc12-v2',repeat('c',64),'recibo:perfil-cc12-v2',
   desde,hasta,true,conf_sha2);
 IF (SELECT version FROM vec_catalogos_configurables.rpt_perfiles_competencia WHERE clave)<>2
    OR (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_perfiles_historia)<>2
    OR (SELECT pg_catalog.count(*) FROM vec_catalogos_configurables.rpt_perfiles_outbox)<>2 THEN
  RAISE EXCEPTION 'CC12: provision CAS no durable'; END IF;
 conf_replay:=vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,0,NULL,
    'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12',
    'principal:administrador-cc12','decision:perfil-cc12-nueva',repeat('d',64),'recibo:perfil-cc12',
    desde,hasta,true,conf_sha);
 IF conf_replay->>'recibo_ref'<>'recibo:perfil-cc12' OR conf_replay->>'version'<>'1' THEN
  RAISE EXCEPTION 'CC12: recibo antiguo perdido'; END IF;
 BEGIN
  PERFORM vec_catalogos_configurables.provisionar_perfiles_rpt(perfil_a,perfil_b,1,conf_sha,
    'fuente:perfil-cc12',repeat('f',64),'custodia:perfil-cc12','acto:perfil-cc12-v2',
    'principal:administrador-cc12','decision:perfil-cc12-v3',repeat('a',64),'recibo:perfil-cc12-v3',
    desde,hasta,true,conf_sha2);
  RAISE EXCEPTION 'CC12: CAS repetido aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
END $test$;
ROLLBACK;

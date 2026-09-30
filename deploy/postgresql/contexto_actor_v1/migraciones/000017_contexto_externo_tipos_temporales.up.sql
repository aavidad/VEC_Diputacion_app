\set ON_ERROR_STOP on
-- CTX17: tipos temporales fuera del circuito externo CTX15.
-- Conserva OID, cuerpos, firmas, propietarios, ACL e historia.
-- No instala CTX15 ni modifica sus fuentes. Sin DOWN con historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:externo:000017',0));
DO $correctiva$
DECLARE esperado record; antes record; despues record; propietario oid;
BEGIN
 propietario:=pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR propietario IS NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.contexto_externo_versiones') IS NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.registros_contexto_externo_v2') IS NULL THEN
  RAISE EXCEPTION 'CTX17: requiere CTX15 instalada y migrador autorizado' USING ERRCODE='55000';
 END IF;
 -- Inventario causal cerrado: las funciones de CTX15 y sus auxiliares externos.
 -- El circuito interno y las autoridades de otros módulos quedan fuera.
 FOR esperado IN SELECT * FROM (VALUES
  ('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)', '1e95045c30d40d55df20c123aafe8e514d4bebccb421850a15c52795b41ace08', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_autorizacion_propietario']::text[]),
  ('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)', '75fcb514ca5b330f8ab7921965edc6fa045adb18bbaa8ee6b01747821dca1c64', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_autorizacion_propietario']::text[]),
  ('vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1()', '6c64f71b1d90d809e3ace1bffceedee183b4b65542f22324482e7035307b66b4', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_candidato_externo']::text[]),
  ('vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1()', 'd5c1a0d2b608c28b712d4cc4a25182c7bc0d5f7221321b631d05acff4bf42495', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_usuarios_externo']::text[]),
  ('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamp with time zone,timestamp with time zone)', '9d5469703af26216a88e55e5f984cbee890bd5cdf145cc78d49e6c6f7ddd0ec1', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_autorizacion_propietario']::text[]),
  ('vec_contexto_actor_v1.acreditar_uso_registro_contexto_externo_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamp with time zone,timestamp with time zone)', '3d15a68e6647450bb4a1d4fbc72cb466aa8165fb868831e76a4f53d950c2d3e4', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(jsonb,timestamp with time zone)', 'a359d4b08ddad6c9c24056fbfc1a124757371344324d51e97310ed59c37f43a4', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.controlar_mutacion_contexto_externo_v1()', '06b91776b1fb05372fc087cbb96ac526d7e1c9793bc6f154561826a6f27a70a9', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.exigir_familia_contexto_externo_v1(text)', 'b877e35db69e095cb23bbf94279e55bfc685a6067075d776f3cb101c49c41a6f', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1()', '6e550be40520d09b1bd37749c79a61826b35efece724fb1e070d7889937424ca', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.exigir_runtime_usuarios_externo_v1()', 'a265b2dab8193e9e9567146195a6e8707e4c9d937a2d54c1334a0a7924f9f248', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(jsonb,numeric)', 'b1f8f2eb1e41dc43ee3b2575505d32e68507f98ae937eba5f5c210c559f9eeb5', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.instante_valido(timestamp with time zone)', 'c7a607ca83ef1b80821dbdf9640bfc07ec5a56c6868f56e37184fd53a44703e1', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)', 'aa17f0608fb82fd8d8d1e5734e1d8c0677de393d6664f100c62bf39e7494990d', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_autorizacion_propietario']::text[]),
  ('vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)', '0a65515047b5ddb079fe9f7bbb31c72f8de5639516f7b7af2d7774d5a43485a7', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_autorizacion_propietario']::text[]),
  ('vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1(text)', '46e280840e0e70c18a746973c5ddddb4c25b8b852de546bf3b7a0b51a9a7e853', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(oid,oid,oid,oid[])', 'adad4d80d892e7cab5c30f43abd214ad6abba95e42d5998e2ba9a51abb1f6175', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.procedencia_valida(text,numeric,text,text)', 'd764763669685d8b667770c7e83a9911c18486ec0bcd40db23682cee281becfa', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(text,numeric,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,timestamp with time zone,timestamp with time zone,text)', '73816ce45676e68201195de5cecbecd65d035b1a0a89ae688834cbc3b5803ca9', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.publicar_provision_candidato_externo_v1(text,numeric,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamp with time zone,timestamp with time zone,text)', '73816ce45676e68201195de5cecbecd65d035b1a0a89ae688834cbc3b5803ca9', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(jsonb,numeric,text,text)', '96c4f753168f30631be52e0add1568f5209e1a1a569f600798636a407a76e797', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.rechazar_mutacion_historia()', '8468903d08f988f2202bf77e9817fdf6a6547c36ce17b71c51f561c8aaf2d4b0', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.rechazar_truncado()', 'a8c6e4d98179f51ffe0dd4556016685083a430d84509bfa7f857c33b47610720', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(text,text,text,text,timestamp with time zone)', '6155c60b9fa23827cda0e73621e25e66a2031e8fcb2efde7cf5a1545271f53f1', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_candidato_externo']::text[]),
  ('vec_contexto_actor_v1.reconciliar_contexto_externo_v2(text,text,text,text,timestamp with time zone,text)', 'c3b054fc361df0b7f7a429099c8473bdf675fdb411611b859f7e7b30e5017f7e', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1(text,text,text,text,timestamp with time zone)', '247265e0140514a7f006fc67a41d4a815c36095930fdc63f4f464819ba434ca1', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_usuarios_externo']::text[]),
  ('vec_contexto_actor_v1.referencia_operacion_valida(text,text)', 'db2bd4a4014289b5ca113b02da6d520231b80928dff0f56e786801560f5d56c4', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.referencia_valida(text,text)', '228163c32df353932c0fa7223ab74ce42966e764202171002ee43aed0a3b7385', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(text,text,text,text,timestamp with time zone)', '3d32e1bd7f1004ff29a42e4bde55175b0bcd0932846df1fd18f672394d350af3', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_candidato_externo']::text[]),
  ('vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1(text,text,text,text,timestamp with time zone)', '75e5f392dd928ee2294faf2d2dfbe6c78fe59e2732719e83728230384c1a85cb', true, ARRAY['vec_contexto_actor_v1_propietario', 'vec_contexto_actor_v1_usuarios_externo']::text[]),
  ('vec_contexto_actor_v1.resolver_y_registrar_contexto_externo_v2(text,text,text,text,timestamp with time zone,text)', '645235215196a9ad72365d43aef5942c5ad1aeb73de6204ccfcea148ea09e86f', true, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(jsonb)', '152de8e8752cb7656879a3f0e9018e61fb27eb2e756b3b3f868b7659a142fdc2', false, ARRAY['vec_contexto_actor_v1_propietario']::text[]),
  ('vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(text,text,text,timestamp with time zone)', 'fd7de3ef3eff05a25aa5550dbf868f8ee9f5f11e58f2b3fc0ad9005124807783', true, ARRAY['vec_contexto_actor_v1_propietario']::text[])
 ) AS inventario(firma,huella,definidor,ejecutores) LOOP
  SELECT p.* INTO antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(esperado.firma);
  IF NOT FOUND OR antes.proowner IS DISTINCT FROM propietario
     OR antes.prosecdef IS DISTINCT FROM esperado.definidor
     OR antes.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
     OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperado.huella
     OR (SELECT count(*) FROM pg_catalog.aclexplode(coalesce(antes.proacl,pg_catalog.acldefault('f',antes.proowner))))<>pg_catalog.cardinality(esperado.ejecutores)
     OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(antes.proacl,pg_catalog.acldefault('f',antes.proowner))) acl
         LEFT JOIN pg_catalog.pg_roles r ON r.oid=acl.grantee
         WHERE acl.grantor<>propietario OR acl.privilege_type<>'EXECUTE' OR acl.is_grantable
            OR r.rolname IS NULL OR r.rolname<>ALL(esperado.ejecutores)) THEN
   RAISE EXCEPTION 'CTX17: preimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
  EXECUTE pg_catalog.format('ALTER FUNCTION %s SET search_path=pg_catalog,pg_temp',antes.oid::regprocedure);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF despues.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
     OR pg_catalog.to_jsonb(despues)-'proconfig' IS DISTINCT FROM pg_catalog.to_jsonb(antes)-'proconfig' THEN
   RAISE EXCEPTION 'CTX17: postimagen incompatible para %',esperado.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $correctiva$;
COMMIT;

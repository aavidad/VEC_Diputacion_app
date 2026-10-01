\set ON_ERROR_STOP on
-- Ejecutar después de la cadena completa B2; no crea consumos.
-- La preimagen final se midió tras AD130 y persistió tras CT155/CT156 y reinicio.
BEGIN;
SET TRANSACTION READ ONLY;
SET LOCAL search_path=pg_catalog, pg_temp;
DO $prueba$
DECLARE
 firma text;
 beneficiario text;
 f oid;
 p record;
 nucleo oid;
 fuente text;
 config text[];
 propietario oid;
 definidora boolean;
 tipo "char";
 volatilidad "char";
 paralelo "char";
 acl aclitem[];
BEGIN
 FOR firma,beneficiario IN
  SELECT * FROM (VALUES
   ('vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_contratacion_temporal_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_bolsa_llamamientos_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_bolsa_llamamientos_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_personal_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_contratacion_temporal_propietario')
  ) AS esperadas(firma,beneficiario)
 LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL THEN RAISE EXCEPTION 'B2: fachada ausente %',firma; END IF;
  SELECT proowner,prosecdef,proconfig,proacl INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR p.prosecdef IS NOT TRUE
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
     OR NOT has_function_privilege(beneficiario,f,'EXECUTE')
     OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE a.grantee NOT IN (p.proowner,beneficiario::regrole::oid)
          OR a.grantor IS DISTINCT FROM p.proowner
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
  THEN RAISE EXCEPTION 'B2: configuración o ACL divergente %',firma; END IF;
 END LOOP;

 nucleo:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF nucleo IS NULL THEN RAISE EXCEPTION 'B2: núcleo ausente'; END IF;
 SELECT prosrc,proconfig,proowner,prosecdef,prokind,provolatile,proparallel,proacl
 INTO STRICT fuente,config,propietario,definidora,tipo,volatilidad,paralelo,acl
 FROM pg_proc WHERE oid=nucleo;
 IF config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR propietario IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR definidora IS NOT TRUE OR tipo<>'f' OR volatilidad<>'v' OR paralelo<>'u'
    OR encode(sha256(convert_to(pg_get_functiondef(nucleo),'UTF8')),'hex')
       IS DISTINCT FROM 'dd5a4a795f5b03ebaf4a0f437d4416e20cffae5450dae04fc8dad721c67a1db4'
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex')
       IS DISTINCT FROM '96f3f3c3b078d2f01c0b731aa39ddefce23ae48d284af15c958a1f2c99d1de89'
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee=propietario AND a.grantor=propietario
         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
       WHERE a.grantee<>propietario OR a.grantor<>propietario
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
          d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=nucleo)
       IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',nucleo,
           'objsubid',0,'refclassid','pg_language'::regclass::oid,
           'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
           'refobjsubid',0,'deptype','n'),
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',nucleo,
           'objsubid',0,'refclassid','pg_namespace'::regclass::oid,
           'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
           'refobjsubid',0,'deptype','n'))
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,
          d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=nucleo)
       IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
           'classid','pg_proc'::regclass::oid,'objid',nucleo,'objsubid',0,
           'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex')
        FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
          AND c.conname='clave_capacidad_version_audiencia_consumo_check'
          AND c.contype='c' AND c.convalidated)
       IS DISTINCT FROM '5ad406c6e428cdc1eb827fa7f5dd86d7406add833cad3deffb25b082789dec0f'
    OR strpos(fuente,'documentos.firmado.custodiar')=0
    OR strpos(fuente,'vinculo_categoria_rpt_ct')=0
    OR strpos(fuente,'consulta_persona_aceptacion_ct_bolsa')=0
    OR strpos(fuente,'consulta_anclaje_aceptacion_ct_bolsa')=0
    OR strpos(fuente,'p_perfil_mutacion IS NOT DISTINCT FROM ''registro_empleado_b2''')=0
    OR strpos(fuente,'vec_personal.plan_incorporacion_ct.v1')=0
    OR strpos(fuente,'personal.plan_incorporacion_ct.clases_ocupacion')=0
    OR strpos(fuente,'incorporacion_personal_ct')=0
 THEN RAISE EXCEPTION 'B2: núcleo incompleto'; END IF;
END $prueba$;
ROLLBACK;

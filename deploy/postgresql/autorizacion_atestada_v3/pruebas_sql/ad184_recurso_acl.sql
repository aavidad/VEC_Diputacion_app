\set ON_ERROR_STOP on
-- Transportes sintéticos para el parser; no claves ni capacidades favorables.
-- Ejecutar sólo tras AUT42 y AD184 reales, en clon y ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
CREATE TEMP TABLE ad184_vector(material text,contexto text,huella text);
INSERT INTO ad184_vector VALUES ('{"esquema":"vec.persona.denominacion.publicar.v1","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version_esperada":0,"procedencia_ref":"prc_cccccccccccccccccccccccccccccccc","procedencia_version":1,"procedencia_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","procedencia_autoridad":"no_autoritativa","sobre_sha256":"733383a01e4a25530c2985e4a1bbb820ce9b122efde41bfaf75ad39fc5680e9d","sobre":{"Esquema":"vec.persona.denominacion.aead.v1","PersonaRef":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","ClaveRef":"clave:denominacion:cifrado:prueba:v1","Version":1,"Nonce":"AAAAAAAAAAAAAAAA","Cifrado":"AAAAAAAAAAAAAAAAAAAAAAA=","Indice":{"AmbitoRef":"ambito:denominacion:prueba","NormaRef":"norma:denominacion:prueba:v1","NormaSHA256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","ClaveRef":"clave:denominacion:busqueda:prueba:v1","Tokens":["AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="]}},"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"}}','{"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"},"atributos":{"material_sha256":"66a0c2c19a8d8658353b6685ba7b47006d28898d76dd9eb30e64f71513b8ceea","procedencia_autoridad":"no_autoritativa","procedencia_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","procedencia_version":"1"}}','979b62c5e85ac67c5db4cae8c2cd04f0d83e618a3078c248ec61a15167629c9f'),('{"esquema":"vec.persona.denominacion.leer.v1","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version":1,"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"}}','{"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"},"atributos":{"material_sha256":"cc2c49b0a1409c936b453d0675a978b2a29d08f682fe73789fe7796f30f490b6"}}','8ca1e459795f57dd96bc4c7e0b88c09a80be9fd73c3723680cf2dc30ac05c023');
GRANT SELECT ON TABLE pg_temp.ad184_vector TO vec_autorizacion_propietario;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $vectores$
DECLARE v record;r jsonb;mutado text;f oid;
BEGIN
 FOR v IN SELECT * FROM pg_temp.ad184_vector LOOP
  r:=vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(v.material);
  IF vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(v.material::jsonb) IS DISTINCT FROM v.material THEN RAISE EXCEPTION 'AD184: canon_material_divergente';END IF;
  IF r->>'contexto_canonico' IS DISTINCT FROM v.contexto OR r->>'contexto_sha256' IS DISTINCT FROM v.huella
  THEN RAISE EXCEPTION 'AD184: vector_contexto_divergente';END IF;
  mutado:=(v.material::jsonb||'{"actor_ref":"inventado"}'::jsonb)::text;
  BEGIN PERFORM vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(mutado);RAISE EXCEPTION 'AD184: material_abierto';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
  mutado:=jsonb_set(v.material::jsonb,'{ambitos,unidad_ref}','"unidad_distinta"')::text;
  IF vec_autorizacion_atestada_v3.recurso_denominacion_persona_v1(vec_autorizacion_atestada_v3.canon_material_denominacion_persona_v1(mutado::jsonb))->>'contexto_sha256'=v.huella THEN RAISE EXCEPTION 'AD184: ambito_no_ligado';END IF;
 END LOOP;
END $vectores$;
RESET ROLE;
DO $acl$
DECLARE f oid;
BEGIN
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN('registrar_y_consumir_denominacion_persona_v3_atestada','cotejar_consumo_denominacion_persona_v3_atestada')LOOP
  IF NOT has_function_privilege('vec_contexto_actor_v1_propietario',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=f AND(a.grantee NOT IN(p.proowner,'vec_contexto_actor_v1_propietario'::regrole)OR a.is_grantable OR a.privilege_type<>'EXECUTE'))THEN RAISE EXCEPTION 'AD184: ACL_abierta';END IF;
 END LOOP;
END $acl$;
ROLLBACK;

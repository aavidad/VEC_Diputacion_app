\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000071:canon:intencion:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $precondicion$
DECLARE firma text; dependencia oid;
BEGIN
 IF current_setting('server_encoding') <> 'UTF8' THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='intencion CT requiere UTF8';
 END IF;
 FOREACH firma IN ARRAY ARRAY[
  'vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(jsonb)',
  'vec_contratacion_temporal.nodo_incorporacion_canonico_v2(jsonb,text)'
 ] LOOP
  dependencia:=to_regprocedure(firma);
  IF dependencia IS NULL OR NOT EXISTS (
   SELECT 1 FROM pg_proc p WHERE p.oid=dependencia
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prokind='f' AND p.provolatile='i' AND NOT p.prosecdef
  ) THEN
   RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='intencion CT requiere codec CT70 propietario';
  END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname=ANY(ARRAY[
   'intencion_registro_incorporacion_canonica_v2','intencion_registro_incorporacion_sha256_v2'])) THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='intencion CT existente o sobrecarga incompatible';
 END IF;
END $precondicion$;

-- API CT72: entrada EXACTA de Material.MaterialCanonico(), no Datos(), ni una
-- intención declarada. CT70 valida también las trazas renovables ANTES de
-- excluirlas. Esta función pura no concede permiso ni acredita origen/commit.
-- Orden de campos idéntico a ports.IntencionRegistroIncorporacionV2.
CREATE FUNCTION vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(entrada jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE v jsonb; canon text;
BEGIN
 PERFORM vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(entrada);
 v:=entrada->'Vinculo';
 canon:='{"Esquema":"vec.contratacion-temporal.incorporacion.ejercicio.intencion.v2","Confirmacion":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada->'Confirmacion','confirmacion')
  ||',"VersionActualExpediente":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada->'VersionActualExpediente','seguro1')
  ||',"Preparacion":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada->'Preparacion','preparacion')
  ||',"Identidad":{"Principal":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'principal_id','per_')
  ||',"Perfil":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'perfil_activo_ref','prf_')
  ||',"Cuenta":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'cuenta_ref','cta_')
  ||',"CuentaOrdinaria":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'cuenta_ordinaria_ref','cta_')
  ||',"Privilegiada":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'cuenta_privilegiada','bool')
  ||',"Superficie":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'superficie','superficie')
  ||',"Autoridad":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->'autoridad_efectiva','=autoridad_maestra_acreditada')
  ||'},"MotivoV3":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada->'MotivoV3','motivo')
  ||',"Personal":'
  ||vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada->'Personal','personal')
  ||',"EjercicioSintetico":true,"FirmaOficial":false,"EficaciaAdministrativa":false}';
 RETURN convert_to(canon,'UTF8');
END $$;

CREATE FUNCTION vec_contratacion_temporal.intencion_registro_incorporacion_sha256_v2(entrada jsonb) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
BEGIN ATOMIC
 SELECT encode(pg_catalog.sha256(vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(entrada)),'hex');
END;

-- Sin SECURITY DEFINER ni grants a runtimes; elimina privilegios por defecto.
DO $acl$
DECLARE f record; g record;
BEGIN
 FOR f IN SELECT p.oid,p.proowner FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname=ANY(ARRAY[
   'intencion_registro_incorporacion_canonica_v2','intencion_registro_incorporacion_sha256_v2']) LOOP
  IF f.proowner<>'vec_contratacion_temporal_propietario'::regrole THEN
   RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='owner intencion CT incompatible';
  END IF;
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
  FOR g IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid=f.oid AND a.grantee<>0 AND a.grantee<>f.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.oid::regprocedure,pg_get_userbyid(g.grantee));
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;

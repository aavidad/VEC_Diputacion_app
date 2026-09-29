\set ON_ERROR_STOP on
-- Fixture sintética solo en clon desechable con CTX15 vigente.
-- La migración AUT19 depende de AUT16; esta prueba integrada usa CTX15.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contexto$
DECLARE snapshot jsonb; c jsonb; k text; prefijo text; h text;
BEGIN
 snapshot:=jsonb_build_object('provision_ref','pce_aut19_sintetica_abcdefghijklmnopqrstuv','poblacion','candidato','estado','activo');
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  prefijo:=CASE k WHEN 'cuenta' THEN 'cta_' WHEN 'persona' THEN 'per_' WHEN 'perfil' THEN 'prf_' WHEN 'contexto' THEN 'vca_' ELSE 'vin_' END;
  c:=jsonb_build_object('referencia',prefijo||'aut19_sintetica_abcdefghijklmnopqrstuv','version',1,
   'procedencia_ref','prc_aut19_sintetica_abcdefghijklmnopqrstuv','procedencia_version',1,'procedencia_huella_sha256',repeat('a',64),
   'procedencia_autoridad','autoridad_maestra_acreditada','estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z');
  IF k='perfil' THEN c:=jsonb_set(c,'{referencia}','"prf_aut19_sintetico_abcdefghijklmnopqrstuv"'::jsonb); END IF;
  IF k='vinculo_candidato' THEN c:=c||jsonb_build_object('candidato_ref','can_aut19_sintetico_abcdefghijklmnopqrstuv'); END IF;
  snapshot:=snapshot||jsonb_build_object(k,c);
 END LOOP;
 h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot,1);
 PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(snapshot,0,NULL,h);
END $contexto$;
RESET ROLE;
-- Añadir la salida de go run del generador contiguo: termina con ROLLBACK.

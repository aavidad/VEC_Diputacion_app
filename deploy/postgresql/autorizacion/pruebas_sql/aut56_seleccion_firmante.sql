\set ON_ERROR_STOP on
-- Vector de AUT56. Como superusuario, sobre un clon desechable con AUT35,
-- CA25, Personal28/29/37/38 y AUT56, y un nodo orgánico vigente. Todo en
-- ROLLBACK. Siembra una identidad, su certificado (CA25), un cargo con su
-- enlace por TIPO (Personal) y una asignación del rol del paso, y comprueba:
-- la selección positiva, sus negativas, que AUT35 coteja por tipo, y que la
-- decisión de firma sigue ligada al documento exacto: con el mismo cargo, una
-- decisión hecha para el original A no sirve para el original B (CT172 la
-- rechaza antes del consumidor V3). Cada caso imprime «OK <caso>»; un fallo
-- aborta.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $latido$ BEGIN
 IF to_regclass('vec_autorizacion_atestada_v3.sellado_auditoria_v5') IS NOT NULL THEN
  UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp(); END IF;
END $latido$;
CREATE FUNCTION pg_temp.exigir(r text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF r IS NULL OR r NOT LIKE 'OK %' THEN RAISE EXCEPTION '%',coalesce(r,'FALLO sin resultado'); END IF; RETURN r; END $f$;
SELECT n.nodo_ref::text AS nodo,n.revision AS rev,n.organismo_ref AS org,n.unidad_ref AS uni
 FROM vec_personal.org_nodo_historia n
 JOIN vec_contexto_actor_v1.organizacion_actual o ON o.organizacion_ref=n.organismo_ref
 WHERE NOT n.retirado AND n.revision=(SELECT max(x.revision) FROM vec_personal.org_nodo_historia x WHERE x.nodo_ref=n.nodo_ref)
  AND n.vigente_desde<=current_date AND (n.vigente_hasta IS NULL OR n.vigente_hasta>current_date)
 ORDER BY n.nodo_ref LIMIT 1 \gset

-- Identidad sintética (patrón del vector CA25) y su certificado.
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES('prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES('cta_aut56_sintetica_aaaaaaaaaaaa',1,'prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),
 'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '100 days');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_aut56_sintetica_aaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES('per_aut56_sintetica_bbbbbbbbbbbb',1,'prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),
 'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '100 days');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_aut56_sintetica_bbbbbbbbbbbb',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES('prf_aut56_sintetico_cccccccccccc',1,'per_aut56_sintetica_bbbbbbbbbbbb','prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),
 'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '100 days');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_aut56_sintetico_cccccccccccc',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES('vca_aut56_sintetico_dddddddddddd',1,'cta_aut56_sintetica_aaaaaaaaaaaa','prf_aut56_sintetico_cccccccccccc',
 'per_aut56_sintetica_bbbbbbbbbbbb','prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),
 'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '100 days');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_aut56_sintetico_dddddddddddd',1);
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 (vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,
 organizacion_ref,organizacion_version,organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,organizacion_procedencia_autoridad,
 superficie,uso,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 SELECT 'vcr_aut56_sintetico_eeeeeeeeeeee',1,'cta_aut56_sintetica_aaaaaaaaaaaa',1,'per_aut56_sintetica_bbbbbbbbbbbb',1,'prf_aut56_sintetico_cccccccccccc',1,
  'vca_aut56_sintetico_dddddddddddd',1,o.organizacion_ref,o.version,o.procedencia_ref,o.procedencia_version,o.procedencia_huella_sha256,o.procedencia_autoridad,
  'interna_corporativa','consulta_rrhh','prc_aut56_sintetica_aaaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '100 days'
 FROM vec_contexto_actor_v1.organizacion_actual a JOIN vec_contexto_actor_v1.organizacion_versiones o USING(organizacion_ref,version)
 WHERE a.organizacion_ref=:'org';
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES('cta_aut56_sintetica_aaaaaaaaaaaa','interna_corporativa','consulta_rrhh','vcr_aut56_sintetico_eeeeeeeeeeee',1);
RESET ROLE;
SELECT to_char(date_trunc('second',clock_timestamp()-interval '10 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS".000000Z"') AS cdesde,
 to_char(date_trunc('second',clock_timestamp()+interval '90 days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS".000000Z"') AS chasta,
 repeat('7',64) AS der \gset
SELECT vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(convert_to(format(
 '{"esquema":%s,"clave":%s,"vinculo_ref":%s,"version":1,"certificado_der_sha256":%s,"cuenta_ref":%s,"persona_ref":%s,"vinculo_cuenta_persona_ref":%s,"organizacion_ref":%s,"estado":"vigente","vigente_desde":%s,"vigente_hasta":%s,"evidencia_ref":%s,"evidencia_sha256":%s,"preimagen_ref":"","preimagen_version":0,"preimagen_sha256":"none"}',
 to_json('vec.contexto-actor.certificado-firmante.publicacion.v2'::text),to_json(repeat('a',32)),to_json('vcc_aut56_sintetico_ffffffffffff'::text),
 to_json(:'der'::text),to_json('cta_aut56_sintetica_aaaaaaaaaaaa'::text),to_json('per_aut56_sintetica_bbbbbbbbbbbb'::text),
 to_json('vca_aut56_sintetico_dddddddddddd'::text),to_json(:'org'::text),to_json(:'cdesde'::text),to_json(:'chasta'::text),
 to_json('evi_aut56_sintetica_aaaaaaaaaaaa'::text),to_json(repeat('9',64))),'UTF8'),'decision:aut56','aud_aut56')->>'estado' AS cert_estado \gset
SELECT pg_temp.exigir(CASE WHEN :'cert_estado'='vigente' THEN 'OK certificado_sembrado' ELSE 'FALLO certificado_sembrado' END);

-- Cargo y enlace por tipo (como el vector de Personal38).
SET LOCAL ROLE vec_personal_propietario;
INSERT INTO vec_personal.cargo_competencial_historia(cargo_ref,version,huella_sha256,organizacion_ref,unidad_ref,puesto_ref,puesto_revision,
 organo_ref,organo_revision,denominacion_catalogo_ref,estado,vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,
 fuente_ref,fuente_version,fuente_huella_sha256,publicada_en,decision_ref,auditoria_ref,recibo_ref)
VALUES('car_AUT56AAAAAAAAAAAAAAAAAAAAA',1,repeat('1',64),:'org',:'uni',NULL,NULL,:'nodo'::uuid,:rev,'cargo:aut56:jefatura','vigente',
 now()-interval '1 day',now()+interval '100 days','acto:aut56:nombramiento',1,repeat('2',64),'fuente:aut56',1,repeat('3',64),now(),'decision:aut56','aud_aut56','percar_aut56_c');
INSERT INTO vec_personal.cargo_competencial_actual VALUES('car_AUT56AAAAAAAAAAAAAAAAAAAAA',1,repeat('1',64));
INSERT INTO vec_personal.enlace_cargo_competencial_historia(enlace_ref,version,huella_sha256,cargo_ref,cargo_version,persona_ref,clase,
 titular_enlace_ref,titular_enlace_version,titular_enlace_sha256,delegante_persona_ref,accion_ref,recurso_ref,finalidad_ref,estado,
 vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,fuente_ref,fuente_version,fuente_huella_sha256,
 requiere_enlace_laboral,empleado_ref,ocupacion_ref,ocupacion_revision,publicada_en,decision_ref,auditoria_ref,recibo_ref)
VALUES('enc_AUT56TITULARAAAAAAAAAAAAA',1,repeat('4',64),'car_AUT56AAAAAAAAAAAAAAAAAAAAA',1,'per_aut56_sintetica_bbbbbbbbbbbb','titular',NULL,NULL,NULL,NULL,
 'contratacion_temporal.documento.firma_vec.registrar','firma_vec_documento_contratacion_temporal','gestionar_contratacion_temporal','vigente',
 now()-interval '1 hour',now()+interval '50 days','acto:aut56:nombramiento',1,repeat('2',64),'fuente:aut56',1,repeat('3',64),false,NULL,NULL,NULL,now(),'decision:aut56','aud_aut56','percar_aut56_t');
INSERT INTO vec_personal.enlace_cargo_competencial_actual VALUES('enc_AUT56TITULARAAAAAAAAAAAAA',1,repeat('4',64));
RESET ROLE;

-- Asignaciones. La fachada sólo elige la única asignación activa y vigente,
-- con rol publicado y control habilitado, y ámbitos exactos (organización y
-- unidad del paso). Primero se siembran asignaciones que incumplen una sola
-- condición cada una: con cualquiera de esos filtros quitado, la selección
-- dejaría de denegarse. Después se añade la buena y luego otra buena.
SELECT r.version_rol_ref AS vrol,r.rol_id AS rol FROM vec_autorizacion.version_rol r
 JOIN vec_autorizacion.control_vigencia_version_rol_actual c ON c.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=c.version_rol_ref AND v.revision=c.revision
 WHERE r.documento->>'estado'='publicada' AND v.estado='habilitada' AND r.rol_id='tecnico_rrhh_desarrollo' ORDER BY r.version DESC LIMIT 1 \gset
SELECT to_char(date_trunc('second',now()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS emitida \gset
SELECT set_config('aut56.vrol',:'vrol',true),set_config('aut56.emitida',:'emitida',true),set_config('aut56.org',:'org',true),
 set_config('aut56.uni',:'uni',true),set_config('aut56.rol',:'rol',true),set_config('aut56.der',:'der',true);
-- Rol propio con el control retirado, para el caso «control no habilitado».
-- Copia las concesiones del rol real: la tabla exige concesiones válidas.
SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
VALUES('rol:aut56_rol_retirado:v1','aut56_rol_retirado',1,repeat('6',64),:'emitida'::timestamptz,
 jsonb_build_object('rol_id','aut56_rol_retirado','version',1,'nombre','Prueba AUT56','estado','publicada',
  'concesiones',(SELECT x.documento->'concesiones' FROM vec_autorizacion.version_rol x WHERE x.version_rol_ref=:'vrol'),
  'publicada_por','prueba:aut56','publicada_en',:'emitida'));
INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
VALUES('rol:aut56_rol_retirado:v1',1,'retirada',repeat('5',64),:'emitida'::timestamptz,
 jsonb_build_object('version_rol_ref','rol:aut56_rol_retirado:v1','revision',1,'estado','retirada','actualizado_por','prueba:aut56','actualizado_en',:'emitida'));
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
VALUES('rol:aut56_rol_retirado:v1',1,now(),'prueba:aut56','acto:aut56');
-- Rol propio sin publicar con el control habilitado: sólo el filtro de
-- «rol publicado» lo deja fuera.
INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
VALUES('rol:aut56_rol_borrador:v1','aut56_rol_borrador',1,repeat('4',64),:'emitida'::timestamptz,
 jsonb_build_object('rol_id','aut56_rol_borrador','version',1,'nombre','Prueba AUT56','estado','borrador',
  'concesiones',(SELECT x.documento->'concesiones' FROM vec_autorizacion.version_rol x WHERE x.version_rol_ref=:'vrol'),
  'publicada_por','prueba:aut56','publicada_en',:'emitida'));
INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
VALUES('rol:aut56_rol_borrador:v1',1,'habilitada',repeat('3',64),:'emitida'::timestamptz,
 jsonb_build_object('version_rol_ref','rol:aut56_rol_borrador:v1','revision',1,'estado','habilitada','actualizado_por','prueba:aut56','actualizado_en',:'emitida'));
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
VALUES('rol:aut56_rol_borrador:v1',1,now(),'prueba:aut56','acto:aut56');
RESET ROLE;
CREATE FUNCTION pg_temp.asignar(id text,perfil text,vrol text DEFAULT current_setting('aut56.vrol'),estado text DEFAULT 'activa',
 hasta interval DEFAULT interval '30 days',ambitos jsonb DEFAULT NULL,desde interval DEFAULT interval '-60 days') RETURNS void LANGUAGE plpgsql AS $f$
DECLARE doc jsonb;
BEGIN
 doc:=jsonb_build_object('asignacion_id',id,'version',1,'perfil_activo_ref',perfil,'principal_id','per_aut56_sintetica_bbbbbbbbbbbb',
  'version_rol_ref',vrol,'estado',estado,'emitida_en',current_setting('aut56.emitida'),
  'vigente_desde',to_char((now()+desde) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'vigente_hasta',to_char((now()+hasta) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'ambitos',coalesce(ambitos,jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(current_setting('aut56.org'))),
   jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(current_setting('aut56.uni'))))));
 SET LOCAL ROLE vec_autorizacion_propietario;
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||id||':v1',id,1,perfil,'per_aut56_sintetica_bbbbbbbbbbbb',vrol,repeat('8',64),current_setting('aut56.emitida')::timestamptz,doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
 VALUES(perfil,'asignacion:'||id||':v1',now(),'prueba:aut56','acto:aut56');
 RESET ROLE;
END $f$;
-- Sustituye la v1 activa por una v2 revocada, que pasa a ser la actual: la
-- v1 sigue en la historia y sólo el cruce con el puntero actual la descarta.
CREATE FUNCTION pg_temp.revocar_v2(id text,perfil text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE doc jsonb;
BEGIN
 SELECT documento||jsonb_build_object('version',2,'estado','revocada') INTO STRICT doc
  FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref='asignacion:'||id||':v1';
 SET LOCAL ROLE vec_autorizacion_propietario;
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 SELECT 'asignacion:'||id||':v2',id,2,perfil,principal_id,version_rol_ref,repeat('7',64),emitida_en,doc
  FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref='asignacion:'||id||':v1';
 UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref='asignacion:'||id||':v2',actualizada_en=now()
  WHERE perfil_activo_ref=perfil;
 RESET ROLE;
END $f$;
CREATE ROLE prueba_aut56_ct LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_aut56_ct WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_aut56_ajeno LOGIN;
GRANT vec_personal_ejecutor TO prueba_aut56_ajeno WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_aut56_mezcla LOGIN;
GRANT vec_contratacion_temporal_ejecutor,vec_autorizacion_propietario TO prueba_aut56_mezcla;
-- Devuelve la selección o el mensaje exacto de la denegación.
CREATE FUNCTION pg_temp.sel(der text DEFAULT current_setting('aut56.der'),rol text DEFAULT current_setting('aut56.rol'),
 org text DEFAULT current_setting('aut56.org'),uni text DEFAULT current_setting('aut56.uni'),
 tipo text DEFAULT 'firma_vec_documento_contratacion_temporal') RETURNS jsonb LANGUAGE plpgsql AS $f$
BEGIN
 RETURN vec_autorizacion.seleccionar_firmante_plan_ct_v1(der,'car_AUT56AAAAAAAAAAAAAAAAAAAAA',rol,'contratacion_temporal.documento.firma_vec.registrar',
  tipo,'gestionar_contratacion_temporal',org,uni);
EXCEPTION WHEN insufficient_privilege THEN RETURN jsonb_build_object('denegado',SQLERRM);
END $f$;
CREATE FUNCTION pg_temp.denegado(r jsonb,mensaje text,caso text) RETURNS text LANGUAGE sql AS $f$
 SELECT CASE WHEN r->>'denegado'=mensaje THEN 'OK '||caso ELSE 'FALLO '||caso||' '||r::text END
$f$;
-- Asignaciones que incumplen una condición cada una.
SELECT pg_temp.asignar('asg_aut56caducadaaaaaaaaaaaaaaaaaa','prf_aut56_caducada_aaaaaaaaaaaaaa',hasta=>interval '-1 day');
SELECT pg_temp.asignar('asg_aut56revocadaaaaaaaaaaaaaaaaaa','prf_aut56_revocada_aaaaaaaaaaaaaa',estado=>'revocada');
SELECT pg_temp.asignar('asg_aut56otraunidadaaaaaaaaaaaaaaa','prf_aut56_otraunidad_aaaaaaaaaaaa',ambitos=>jsonb_build_array(
 jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(current_setting('aut56.org'))),
 jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array('unidad:aut56:ajena'))));
SELECT pg_temp.asignar('asg_aut56sinunidadaaaaaaaaaaaaaaaa','prf_aut56_sinunidad_aaaaaaaaaaaaa',ambitos=>jsonb_build_array(
 jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(current_setting('aut56.org')))));
SELECT pg_temp.asignar('asg_aut56dimextraaaaaaaaaaaaaaaaaa','prf_aut56_dimextra_aaaaaaaaaaaaaa',ambitos=>jsonb_build_array(
 jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(current_setting('aut56.org'))),
 jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(current_setting('aut56.uni'))),
 jsonb_build_object('clave','centro_ref','valores',jsonb_build_array('centro:aut56'))));
SELECT pg_temp.asignar('asg_aut56dosvaloresaaaaaaaaaaaaaaa','prf_aut56_dosvalores_aaaaaaaaaaaa',ambitos=>jsonb_build_array(
 jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(current_setting('aut56.org'))),
 jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array(current_setting('aut56.uni'),'unidad:aut56:ajena'))));
SELECT pg_temp.asignar('asg_aut56retiradoaaaaaaaaaaaaaaaaa','prf_aut56_retirado_aaaaaaaaaaaaaa',vrol=>'rol:aut56_rol_retirado:v1');
SELECT pg_temp.asignar('asg_aut56futuraaaaaaaaaaaaaaaaaaaa','prf_aut56_futura_aaaaaaaaaaaaaaaa',desde=>interval '1 day');
SELECT pg_temp.asignar('asg_aut56sustituidaaaaaaaaaaaaaaaa','prf_aut56_sustituida_aaaaaaaaaaaa');
SELECT pg_temp.revocar_v2('asg_aut56sustituidaaaaaaaaaaaaaaaa','prf_aut56_sustituida_aaaaaaaaaaaa');
SELECT pg_temp.asignar('asg_aut56borradoraaaaaaaaaaaaaaaaa','prf_aut56_borrador_aaaaaaaaaaaaaa',vrol=>'rol:aut56_rol_borrador:v1');

SET SESSION AUTHORIZATION prueba_aut56_ct;
SET LOCAL timezone='UTC';
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(),'seleccion_firmante_ausente_o_ambigua','solo_asignaciones_invalidas'));
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(rol=>'aut56_rol_retirado'),'seleccion_firmante_ausente_o_ambigua','control_retirado_denegado'));
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(rol=>'aut56_rol_borrador'),'seleccion_firmante_ausente_o_ambigua','rol_sin_publicar_denegado'));
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(der=>repeat('6',64)),'seleccion_firmante_certificado_no_admitido','certificado_ajeno_denegado'));
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(org=>'org_aut56ajenaaaaaaaaaaaaaaaaaaa'),'seleccion_firmante_certificado_no_admitido','organizacion_del_certificado_distinta'));
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(tipo=>'documento:aut56:original-a'),'cargo_localizacion_ausente_o_ambigua','enlace_por_documento_no_selecciona'));
RESET SESSION AUTHORIZATION;
-- La asignación buena.
SELECT pg_temp.asignar('asg_aut56aaaaaaaaaaaaaaaaaaaaaaaaaa','prf_aut56_sintetico_cccccccccccc');
SET SESSION AUTHORIZATION prueba_aut56_ct;
SELECT pg_temp.sel()::text AS s \gset
SELECT pg_temp.exigir(CASE WHEN (:'s'::jsonb)->>'enlace_ejercicio_ref'='enc_AUT56TITULARAAAAAAAAAAAAA'
 AND (:'s'::jsonb)->>'perfil_activo_ref'='prf_aut56_sintetico_cccccccccccc' AND (:'s'::jsonb)->>'persona_ref'='per_aut56_sintetica_bbbbbbbbbbbb'
 AND (:'s'::jsonb)#>>'{asignacion,referencia}'='asignacion:asg_aut56aaaaaaaaaaaaaaaaaaaaaaaaaa:v1'
 AND (:'s'::jsonb)#>>'{asignacion,version}'='1' AND (:'s'::jsonb)#>>'{asignacion,huella_sha256}'=repeat('8',64)
 AND (:'s'::jsonb)#>>'{rol,referencia}'=:'vrol' AND (:'s'::jsonb)#>>'{control_rol,referencia}'=:'vrol'
 AND (:'s'::jsonb)#>>'{control_rol,revision}' IS NOT NULL
 AND (:'s'::jsonb)->>'cuenta_ref'='cta_aut56_sintetica_aaaaaaaaaaaa'
 AND (:'s'::jsonb)#>>'{vinculo_certificado,referencia}'='vcc_aut56_sintetico_ffffffffffff' AND (:'s'::jsonb)#>>'{vinculo_certificado,version}'='1'
 THEN 'OK seleccion_positiva' ELSE 'FALLO seleccion_positiva '||:'s' END);
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(rol=>'otro_rol_aut56'),'seleccion_firmante_ausente_o_ambigua','otro_rol_denegado'));
RESET SESSION AUTHORIZATION;
-- Una segunda asignación válida del mismo rol hace la selección ambigua.
SELECT pg_temp.asignar('asg_aut56bbbbbbbbbbbbbbbbbbbbbbbbbb','prf_aut56_sintetico_segundo_cccc');
SET SESSION AUTHORIZATION prueba_aut56_ct;
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(),'seleccion_firmante_ausente_o_ambigua','dos_asignaciones_ambiguas'));
RESET SESSION AUTHORIZATION;
-- Guardas: un LOGIN que además es propietario de AUT no la ejecuta, y otro
-- grupo no tiene EXECUTE.
SET SESSION AUTHORIZATION prueba_aut56_mezcla;
SELECT pg_temp.exigir(pg_temp.denegado(pg_temp.sel(),'seleccion_firmante_denegada','login_con_propietario_aut_denegado'));
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut56_ajeno;
DO $ajeno$ BEGIN
 BEGIN PERFORM vec_autorizacion.seleccionar_firmante_plan_ct_v1(repeat('7',64),'car_x','r_x','a.b','t_x','f_x','org_x','uni_x');
  RAISE EXCEPTION 'FALLO ajeno_sin_execute';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ajeno$;
RESET SESSION AUTHORIZATION;
SELECT 'OK ajeno_sin_execute';
-- La vía directa de CT172 queda cerrada para el ejecutor.
SELECT pg_temp.exigir(CASE WHEN NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
 'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN 'OK ct172_directa_cerrada' ELSE 'FALLO ct172_directa_cerrada' END);

-- AUT35 coteja el enlace por tipo.
SELECT pg_temp.exigir(CASE WHEN strpos(pg_get_functiondef('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure),
 $m$per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'tipo_recurso'$m$)>0
 AND strpos(pg_get_functiondef('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure),
 $m$per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'recurso_autorizable_ref'$m$)=0
 THEN 'OK aut35_por_tipo' ELSE 'FALLO aut35_por_tipo' END);

-- La decisión sigue ligada al documento exacto: con el mismo cargo, una
-- decisión para el original A, presentada con el material del original B, se
-- rechaza en CT172 antes del consumidor V3; con A sí llega al consumidor.
-- Como el vector de CT172, una identidad técnica mixta hace que el núcleo V3
-- deniegue de forma determinista (42501) en cuanto lo alcanza.
CREATE ROLE prueba_aut56_mixto LOGIN;
GRANT vec_contratacion_temporal_ejecutor,vec_personal_ejecutor TO prueba_aut56_mixto;
-- La vía directa está cerrada para el ejecutor; sólo dentro de esta
-- transacción (ROLLBACK) se le concede a esta identidad de prueba para
-- comprobar la atadura al documento en CT172.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) TO prueba_aut56_mixto;
DO $v3$ BEGIN
 IF to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) TO prueba_aut56_mixto;
 END IF;
END $v3$;
RESET ROLE;
SET SESSION AUTHORIZATION prueba_aut56_mixto;
DO $documento$
DECLARE base jsonb;b jsonb;evidencia jsonb;desc_a jsonb;desc_b jsonb;bd_a bytea;bd_b bytea;contexto_h text;decision bytea;capacidad bytea;traza text;
 v3 boolean:=to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL;
 registrar text;
 vacio bytea:=convert_to('{}','UTF8');
BEGIN
 evidencia:=jsonb_build_array(jsonb_build_object('Orden',1,'ByteRange',jsonb_build_array(0,1050,1150,50),
  'RevisionHuellaSHA256',repeat('d',64),'ContenidoFirmadoHuellaSHA256',repeat('4',64),'RevisionLongitud',1200,
  'CubreDocumentoCompletoHastaAqui',true,'FirmanteRef','ref:'||repeat('e',64),'CertificadoHuellaSHA256',repeat('e',64),
  'IntegridadEstado','valida','CadenaEstado','valida','CertificadoEstado','vigente','RevocacionEstado','vigente',
  'SelloTiempoEstado','no_presente','TipoFirma','aprobacion','NivelDocMDP',NULL,
  'CambiosDesdeAnterior',jsonb_build_object('Estado','permitidos','Detalle',jsonb_build_array('firma_anadida'))));
 base:=jsonb_build_object(
   'Via','certificado_vec',
   'OrganizacionRef','organizacion:ct172:negativas',
   'ExpedienteRef','expediente:ct172:negativas',
   'VersionExpediente',1,
   'Documento','informe_definitivo',
   'CatalogoRef','vec.circuito:1',
   'CatalogoHuella',repeat('a',64),
   'PasoRef','vec.circuito:1:informe.p1',
   'PasoOrden',1,
   'Secuencia',1,
   'HistoriaRevision',0,
   'HistoriaHuella',repeat('b',64),
   'OriginalRef','documento:ct172:original',
   'OriginalVersion',1,
   'OriginalHuella',repeat('c',64),
   'FirmadoHuella',repeat('d',64),
   'CertificadoHuella',repeat('e',64),
   'FirmanteRef','ref:'||repeat('e',64),
   'FirmantePrincipalRef','per_ct172_firmante',
   'PerfilFirmanteRef','perfil:ct:jefatura',
   'CargoFirmante','ct_cargo_jefatura',
   'UnidadFirmanteRef','unidad:ct172:negativas',
   'PerfilActivoFirmanteRef','prf_ct172_firmante',
   'PuestoFirmanteRef',NULL,
   'AmbitoFirmanteRef',NULL)||
  jsonb_build_object(
   'AsignacionFirmanteRef','asignacion:ct172:negativas',
   'AsignacionFirmanteVersion',1,
   'AsignacionFirmanteHuella',repeat('f',64),
   'VersionRolFirmanteRef','rol:ct_cargo_jefatura:v1',
   'VersionRolFirmanteHuella',repeat('1',64),
   'ControlVigenciaFirmanteRef','rol:ct_cargo_jefatura:v1',
   'ControlVigenciaFirmanteRevision',1,
   'ControlVigenciaFirmanteHuella',repeat('2',64),
   'AsignacionVigenteDesde','2026-01-01T00:00:00Z',
   'AsignacionVigenteHasta','2027-01-01T00:00:00Z',
   'ActoCompetenciaRef',NULL,
   'DelegacionRef',NULL,
   'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v2',
   'RevocacionEstado','vigente',
   'SelloTiempoEstado','no_presente',
   'ClaveIdempotencia','clave-ct172-negativa-0001',
   'DocumentoCustodiaRef','documento:ct172:firmado',
   'DocumentoCustodiaVersion',1,
   'CatalogoVersion',1,
   'RolIDFirmante','ct_cargo_jefatura',
   'PerfilActivoOperadorRef','prf_ct172_firmante',
   'CuentaFirmanteRef','cuenta:ct172:negativas',
   'VinculoCredencialFirmanteRef','vinculo:ct172:negativas',
   'VinculoCredencialFirmanteRevision',1,
   'VinculoCredencialFirmanteHuella',repeat('3',64))||
  jsonb_build_object(
   'FirmaAnteriorRef',NULL,
   'ReciboAnteriorRef',NULL,
   'EntradaDocumentoRef','documento:ct172:original',
   'EntradaDocumentoVersion',1,
   'EntradaDocumentoLongitud',1000,
   'EntradaDocumentoHuella',repeat('c',64),
   'OrdenFirmaPDF',1,
   'ByteRange',jsonb_build_array(0,1050,1150,50),
   'RevisionHuellaSHA256',repeat('d',64),
   'ContenidoFirmadoHuellaSHA256',repeat('4',64),
   'RevisionLongitud',1200,
   'EvidenciaFirmasCanonica',evidencia,
   'EvidenciaFirmasHuellaSHA256',encode(sha256(convert_to(evidencia::text,'UTF8')),'hex'));
 -- Firmante, perfil, organización y unidad de la asignación sembrada (con
 -- AD206 la huella sale de esa asignación; sin AD206, sólo de la organización).
 base:=base||jsonb_build_object('OrganizacionRef',current_setting('aut56.org'),'UnidadFirmanteRef',current_setting('aut56.uni'),
  'FirmantePrincipalRef','per_aut56_sintetica_bbbbbbbbbbbb','PerfilActivoFirmanteRef','prf_aut56_sintetico_cccccccccccc',
  'PerfilActivoOperadorRef','prf_aut56_sintetico_cccccccccccc');
 -- Mismo cargo, mismo paso y misma persona; sólo cambia el original.
 b:=base||jsonb_build_object('OriginalRef','documento:ct172:original-b','OriginalHuella',repeat('e',64),
  'EntradaDocumentoRef','documento:ct172:original-b','EntradaDocumentoHuella',repeat('e',64));
 desc_a:=jsonb_build_object('esquema','vec.competencia-firmante.constructor-ct.v1',
  'certificado_der_sha256',base->'CertificadoHuella',
  'seleccion',jsonb_build_object('perfil_esperado_ref',base->'PerfilFirmanteRef','perfil_activo_ref',base->'PerfilActivoFirmanteRef',
   'rol_id',base->'RolIDFirmante','cargo_ref','cargo:ct172:negativas','enlace_ejercicio_ref','enlace:ct172:negativas'),
  'recurso',jsonb_build_object('organizacion_ref',base->'OrganizacionRef','unidad_ref',base->'UnidadFirmanteRef',
   'expediente_ref',base->'ExpedienteRef','documento_ref',base->'OriginalRef','recurso_autorizable_ref',base->'OriginalRef',
   'modulo_id','contratacion_temporal','tipo_recurso','documento','recurso_contexto_sha256',repeat('5',64),
   'original',jsonb_build_object('referencia',base->'OriginalRef','version',base->'OriginalVersion','huella_sha256',base->'OriginalHuella'),
   'pdf_raiz_sha256',base->'OriginalHuella',
   'firmado',jsonb_build_object('referencia',base->'DocumentoCustodiaRef','version',base->'DocumentoCustodiaVersion','huella_sha256',base->'FirmadoHuella'),
   'pdf_firmado_sha256',base->'FirmadoHuella','numero_firmas',1,'entrada_revision',NULL),
  'accion','contratacion_temporal.documento.firmar','finalidad','formalizar',
  'motivo',jsonb_build_object('catalogo_id','vec.motivos','catalogo_version',1,'catalogo_huella_sha256',repeat('6',64),'entrada_clave','firma'),
  'circuito',jsonb_build_object('referencia',base->'CatalogoRef','version',base->'CatalogoVersion','huella_sha256',base->'CatalogoHuella'),
  'paso_ref',base->'PasoRef','paso_orden',1,'fecha_historica',NULL);
 desc_b:=jsonb_set(jsonb_set(jsonb_set(desc_a,'{recurso,documento_ref}','"documento:ct172:original-b"'),'{recurso,recurso_autorizable_ref}','"documento:ct172:original-b"'),
  '{recurso,original}',jsonb_build_object('referencia','documento:ct172:original-b','version',1,'huella_sha256',repeat('e',64)));
 desc_b:=jsonb_set(desc_b,'{recurso,pdf_raiz_sha256}',to_jsonb(repeat('e',64)));
 bd_a:=convert_to(desc_a::text,'UTF8'); bd_b:=convert_to(desc_b::text,'UTF8');
 -- Decisión y capacidad para el original A (huella de contexto de A).
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||current_setting('aut56.org')||'"'||
  CASE WHEN v3 THEN ',"unidad_ref":"'||current_setting('aut56.uni')||'"' ELSE '' END||'},"atributos":{"descriptor_firma_sha256":"'||
  encode(sha256(bd_a),'hex')||'","material_sha256":"'||encode(sha256(convert_to(base::text,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 decision:=convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firma_vec.registrar','modulo_id','contratacion_temporal',
  'tipo_recurso','firma_vec_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
  'recurso_ref','operacion-firma-vec-ct:'||(base->>'ClaveIdempotencia'),'contexto_recurso_huella_sha256',contexto_h,
  'principal_id',base->'FirmantePrincipalRef','perfil_activo_ref',base->'PerfilActivoOperadorRef',
  'asignacion_ref','asignacion:asg_aut56aaaaaaaaaaaaaaaaaaaaaaaaaa:v1','asignacion_huella_sha256',repeat('8',64),'version_rol_ref',current_setting('aut56.vrol'),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 capacidad:=convert_to(jsonb_build_object('suite','no_acreditada','operacion','contratacion_temporal.documento.firma_vec.registrar',
  'audiencia_consumo','vec_contratacion_temporal.firma_vec.v2','efecto_ref','operacion-firma-vec-ct:'||(base->>'ClaveIdempotencia'),
  'huella_efecto_sha256',contexto_h,'huella_decision_sha256',encode(sha256(decision),'hex'))::text,'UTF8');
 registrar:=CASE WHEN v3 THEN 'registrar_firma_verificada_v3' ELSE 'registrar_firma_verificada_v2' END;
 -- Control: con A la fachada llega al consumidor V3 (que la deniega por no
 -- estar atestada); así la negativa de B no se debe a otra cosa.
 BEGIN
  EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)',registrar) USING base::text,'2026-10-03T00:00:00Z'::timestamptz,capacidad,decision,
   vacio,vacio,1::numeric,1::numeric,vacio,vacio,vacio,vacio,bd_a;
  RAISE EXCEPTION 'FALLO control_a_llega_al_consumidor (aceptado)';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  GET STACKED DIAGNOSTICS traza=PG_EXCEPTION_CONTEXT;
  IF strpos(traza,'consumir_decision_mutacion_v3_interna')=0 THEN RAISE EXCEPTION 'FALLO control_a_llega_al_consumidor'; END IF;
 END;
 RAISE NOTICE 'OK control_a_llega_al_consumidor';
 -- Original B con la decisión de A: divergente antes del consumidor.
 BEGIN
  EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)',registrar) USING b::text,'2026-10-03T00:00:00Z'::timestamptz,capacidad,decision,
   vacio,vacio,1::numeric,1::numeric,vacio,vacio,vacio,vacio,bd_b;
  RAISE EXCEPTION 'FALLO otro_original_denegado (aceptado)';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  GET STACKED DIAGNOSTICS traza=PG_EXCEPTION_CONTEXT;
  IF strpos(traza,'consumir_decision_mutacion_v3_interna')<>0 THEN RAISE EXCEPTION 'FALLO otro_original_denegado (llegó al consumidor)'; END IF;
 END;
 RAISE NOTICE 'OK otro_original_denegado';
END $documento$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

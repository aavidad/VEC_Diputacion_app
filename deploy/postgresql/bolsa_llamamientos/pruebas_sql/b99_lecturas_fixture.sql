-- Se añade al cuerpo de b96_solicitud_idempotencia_fixture.sql antes de su
-- ROLLBACK final. Sólo en PostgreSQL 18 desechable con BC9, CC11, B96 y B99.
DO $b99$
DECLARE
 s record; propias jsonb; detalle jsonb; convs jsonb; lista jsonb;
 carta jsonb; pendientes bigint; esperados bigint;
BEGIN
 SELECT solicitud_ref,persona_ref,convocatoria_ref,unidad_ref,ambito_ref
 INTO s FROM vec_bolsa_llamamientos.solicitud_inscripcion ORDER BY presentada_en DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'B99 fixture: falta solicitud B96'; END IF;
 propias:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
  true,s.persona_ref,jsonb_build_object('estado','','convocatoria_ref','','cursor','','limite',20),
  'es','externa_personal',NULL,NULL);
 IF (propias->>'total')::bigint<1 OR EXISTS(
   SELECT 1 FROM jsonb_array_elements(propias->'solicitudes') x
   WHERE coalesce(x->>'convocatoria_titulo','')='')
 THEN RAISE EXCEPTION 'B99 fixture: título ausente en lista propia'; END IF;
 detalle:=vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
  true,s.persona_ref,s.solicitud_ref,'en','externa_personal',NULL,NULL);
 IF coalesce(detalle->>'convocatoria_titulo','')=''
 THEN RAISE EXCEPTION 'B99 fixture: título ausente en detalle inglés'; END IF;
 convs:=vec_bolsa_llamamientos.listar_convocatorias_rrhh_inscripcion_interna_v1(
  s.unidad_ref,s.ambito_ref,'',20,'es');
 SELECT x INTO carta FROM jsonb_array_elements(convs->'convocatorias') x
 WHERE x->>'convocatoria_ref'=s.convocatoria_ref;
 IF carta IS NULL THEN RAISE EXCEPTION 'B99 fixture: convocatoria ausente'; END IF;
 pendientes:=(carta->>'pendientes')::bigint;
 lista:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
  false,s.persona_ref,
  jsonb_build_object('estado','pendiente','convocatoria_ref',s.convocatoria_ref,'cursor','','limite',20),
  'es','interna_corporativa',s.unidad_ref,s.ambito_ref);
 IF (lista->>'total')::bigint<>pendientes
 THEN RAISE EXCEPTION 'B99 fixture: recuento %, lista %',
  pendientes,lista->>'total'; END IF;
 SELECT count(*) INTO esperados
 FROM vec_bolsa_llamamientos.solicitud_inscripcion q
 JOIN LATERAL (SELECT version,estado FROM vec_bolsa_llamamientos.solicitud_inscripcion_version v
  WHERE v.solicitud_ref=q.solicitud_ref ORDER BY version DESC LIMIT 1) actual ON true
 JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
  ON r.solicitud_ref=q.solicitud_ref AND r.version=actual.version
 WHERE q.convocatoria_ref=s.convocatoria_ref AND q.unidad_ref=s.unidad_ref
  AND q.ambito_ref=s.ambito_ref AND actual.estado='pendiente';
 IF pendientes<>esperados
 THEN RAISE EXCEPTION 'B99 fixture: recuento %, filas %',pendientes,esperados; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(lista->'solicitudes') x
   WHERE coalesce(x->>'convocatoria_titulo','')='')
 THEN RAISE EXCEPTION 'B99 fixture: título ausente en lista RRHH'; END IF;
 lista:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
  false,s.persona_ref,
  jsonb_build_object('estado','pendiente','convocatoria_ref',s.convocatoria_ref,'cursor','','limite',20),
  'es','interna_corporativa','unidad_sin_acceso',s.ambito_ref);
 IF (lista->>'total')::bigint<>0
 THEN RAISE EXCEPTION 'B99 fixture: ámbito ajeno visible'; END IF;
END $b99$;
ROLLBACK;

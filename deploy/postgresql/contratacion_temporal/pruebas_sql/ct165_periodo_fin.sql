\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE
    cerrado jsonb := '{"inicio":"2026-10-02T00:00:00Z","fin":"2026-10-31T00:00:00Z"}';
    abierto jsonb := '{"inicio":"2026-10-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"regla_ref":"regla:modalidad:sustitucion","catalogo_version":2,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fecha_fin":"no_aplica","causa_fin":"reincorporacion_titular"}}';
    solicitud jsonb;
    canon text;
BEGIN
    IF vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(cerrado) IS NOT TRUE
       OR vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(abierto) IS NOT TRUE
       OR vec_contratacion_temporal.periodo_previsto_estructural_v1(
          '{"inicio":"2026-10-02T00:00:00Z","fin":null}') IS NOT FALSE
       OR vec_contratacion_temporal.periodo_previsto_estructural_v1(
          '{"inicio":"2026-10-02T00:00:00Z","fin":"2026-10-31T00:00:00Z","causa_fin":"reincorporacion_titular"}') IS NOT FALSE
       OR vec_contratacion_temporal.periodo_previsto_estructural_v1(
          '{"inicio":"2026-10-02T00:00:00Z","causa_fin":"Texto libre"}') IS NOT FALSE
       OR vec_contratacion_temporal.periodo_previsto_estructural_v1(
          '{"inicio":"2026-10-02T00:00:00Z","causa_fin":"reincorporacion_titular"}') IS NOT FALSE THEN
        RAISE EXCEPTION 'CT165 periodo estructural incorrecto';
    END IF;
    solicitud := pg_catalog.jsonb_build_object(
      'centro_ref','centro:sintetico','contacto_ref','persona:sintetica',
      'categoria_ref','categoria:sintetica','grupo_subgrupo','C2',
      'motivo_clave','sustitucion','detalle','Caso sintético',
      'periodo',cerrado,
      'rc',pg_catalog.jsonb_build_object('existe',false,'numero','','fecha','',
         'importe',pg_catalog.jsonb_build_object('centimos',0,'moneda','EUR'),
         'documento_ref',''),
      'documentos_adjuntos','[]'::jsonb,'observaciones',''
    );
    canon := vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(solicitud);
    IF canon IS NULL OR pg_catalog.strpos(canon, '"fin":"2026-10-31T00:00:00Z"')=0
       OR pg_catalog.strpos(canon, '"causa_fin"')<>0 THEN
        RAISE EXCEPTION 'CT165 canon legado alterado';
    END IF;
    canon := vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(
        pg_catalog.jsonb_set(solicitud,'{periodo}',abierto,false));
    IF canon IS NULL OR pg_catalog.strpos(canon,'"causa_fin":"reincorporacion_titular"')=0
       OR pg_catalog.strpos(canon,'"politica_fin":{"regla_ref":"regla:modalidad:sustitucion","catalogo_version":2')=0
       OR pg_catalog.strpos(canon,'"fin"')<>0 THEN
        RAISE EXCEPTION 'CT165 canon con causa incorrecto';
    END IF;
END
$prueba$;
ROLLBACK;

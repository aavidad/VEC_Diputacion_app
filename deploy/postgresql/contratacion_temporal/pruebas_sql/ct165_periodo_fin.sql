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
    analisis jsonb;
    canon text;
    huella text;
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
    analisis := pg_catalog.jsonb_build_object(
      'modalidad_clave','sustitucion','categoria_ref','categoria:sintetica',
      'grupo_subgrupo','C2','causa_clave','reincorporacion_titular',
      'periodo',abierto,'porcentaje_jornada',10000,
      'entrada_rc_esperada',pg_catalog.jsonb_build_object(
        'referencia','entrada:rc:sintetica','huella_sha256',repeat('4',64)),
      'actuacion_registro',pg_catalog.jsonb_build_object(
        'secuencia',2,'version_expediente',2,
        'accion_clave','contratacion_temporal.analisis.registrar',
        'fase_destino','solicitud','recibo_ref','recibo:analisis:sintetico'),
      'validacion_rc',pg_catalog.jsonb_build_object(
        'resultado','no_requerida','entrada_ref','entrada:rc:sintetica',
        'huella_entrada_sha256',repeat('4',64),
        'fuente_ref','fuente:presupuestaria:sintetica',
        'recibo_ref','recibo:fuente:rc:sintetico',
        'validada_en','2026-10-02T00:00:00Z','motivo','tramite_ordinario')
    );
    huella := vec_contratacion_temporal.huella_analisis_derivado_v2(analisis);
    IF huella IS NULL OR huella !~ '^[0-9a-f]{64}$'
       OR huella=vec_contratacion_temporal.huella_analisis_derivado_v2(
          pg_catalog.jsonb_set(analisis,'{periodo,politica_fin,catalogo_version}','3'::jsonb,false)) THEN
        RAISE EXCEPTION 'CT165 huella no vincula versión de política';
    END IF;
END
$prueba$;
ROLLBACK;

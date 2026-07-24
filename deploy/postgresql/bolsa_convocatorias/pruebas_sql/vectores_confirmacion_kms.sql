-- Contrato cruzado Go/PostgreSQL. El runner copia el mismo fixture versionado
-- que consume vectores_dorados_borradores_test.go al contenedor efimero. Las
-- funciones y migraciones de produccion no leen ni dependen de estos ficheros.
BEGIN;

CREATE TEMP TABLE pg_temp.vectores_kms_confirmacion_borrador_v1 (
    manifiesto jsonb NOT NULL,
    aad_canonica text NOT NULL,
    acreditacion jsonb NOT NULL,
    preimagen_atestacion text NOT NULL,
    preimagen_revalidacion text NOT NULL
) ON COMMIT DROP;

INSERT INTO pg_temp.vectores_kms_confirmacion_borrador_v1
SELECT pg_read_file('/tmp/vec-vectores-kms-borrador-v1/manifest.json')::jsonb,
       rtrim(pg_read_file(
           '/tmp/vec-vectores-kms-borrador-v1/aad_canonica.json'
       ), E'\n'),
       pg_read_file(
           '/tmp/vec-vectores-kms-borrador-v1/acreditacion_postgresql.json'
       )::jsonb,
       rtrim(pg_read_file(
           '/tmp/vec-vectores-kms-borrador-v1/preimagen_atestacion.json'
       ), E'\n'),
       rtrim(pg_read_file(
           '/tmp/vec-vectores-kms-borrador-v1/preimagen_revalidacion.json'
       ), E'\n');

GRANT SELECT ON pg_temp.vectores_kms_confirmacion_borrador_v1
    TO vec_bolsa_convocatorias_propietario;

SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $vectores$
DECLARE
    fixture record;
    manifiesto jsonb;
    aad jsonb;
    acreditacion jsonb;
    aad_reordenada jsonb;
    acreditacion_reordenada jsonb;
BEGIN
    SELECT * INTO STRICT fixture
      FROM pg_temp.vectores_kms_confirmacion_borrador_v1;
    manifiesto := fixture.manifiesto;
    aad := fixture.aad_canonica::jsonb;
    acreditacion := fixture.acreditacion;

    IF manifiesto ->> 'esquema' <>
          'vec.pruebas.bolsa.vectores-kms-confirmacion-borrador.v1'
       OR aad ->> 'huella_version_sha256' <>
          manifiesto ->> 'huella_version_sha256'
       OR aad ->> 'huella_material_sha256' <>
          manifiesto ->> 'huella_material_sha256'
       OR aad ->> 'huella_decision_politica_sha256' <>
          manifiesto ->> 'huella_decision_politica_sha256'
       OR aad ->> 'huella_evidencia_perfil_sha256' <>
          manifiesto ->> 'huella_evidencia_perfil_sha256'
       OR acreditacion ->> 'huella_aad' <>
          manifiesto ->> 'huella_aad_sha256'
       OR acreditacion ->> 'huella_envoltura_sha256' <>
          manifiesto ->> 'huella_envoltura_sha256'
       OR acreditacion ->> 'huella_sobre_sha256' <>
          manifiesto ->> 'huella_sobre_sha256'
       OR acreditacion ->> 'huella_cuerpo_recibo_sha256' <>
          manifiesto ->> 'huella_cuerpo_recibo_sha256'
       OR acreditacion ->> 'huella_acreditacion_sha256' <>
          manifiesto ->> 'huella_acreditacion_sha256'
       OR acreditacion -> 'firma_atestacion_kms' ->>
              'huella_preimagen_sha256' <>
          manifiesto ->> 'huella_preimagen_atestacion_sha256'
       OR acreditacion -> 'firma_revalidacion_kms' ->>
              'huella_preimagen_sha256' <>
          manifiesto ->> 'huella_preimagen_revalidacion_sha256'
       OR acreditacion -> 'firma_atestacion_kms' ->>
              'firma_base64url_sin_relleno' <>
          manifiesto ->> 'firma_atestacion_base64url_sin_relleno'
       OR acreditacion -> 'firma_revalidacion_kms' ->>
              'firma_base64url_sin_relleno' <>
          manifiesto ->> 'firma_revalidacion_base64url_sin_relleno' THEN
        RAISE EXCEPTION 'fixture KMS internamente divergente';
    END IF;

    SELECT jsonb_object_agg(clave, valor ORDER BY clave DESC)
      INTO aad_reordenada
      FROM jsonb_each(aad) AS e(clave, valor);
    SELECT jsonb_object_agg(clave, valor ORDER BY clave DESC)
      INTO acreditacion_reordenada
      FROM jsonb_each(acreditacion) AS e(clave, valor);

    IF (acreditacion ->> 'revalidacion_solicitada_en')::timestamptz >=
          (acreditacion ->> 'confirmada_en')::timestamptz THEN
        RAISE EXCEPTION
            'vector KMS confundio preparacion real con not-before';
    END IF;

    IF vec_bolsa_convocatorias.aad_canonica_borrador_v1(aad)
          IS DISTINCT FROM convert_to(fixture.aad_canonica, 'UTF8')
       OR encode(sha256(convert_to(fixture.aad_canonica, 'UTF8')), 'hex') <>
          manifiesto ->> 'huella_aad_sha256'
       OR vec_bolsa_convocatorias.aad_canonica_borrador_v1(aad_reordenada)
          IS DISTINCT FROM convert_to(fixture.aad_canonica, 'UTF8') THEN
        RAISE EXCEPTION 'vector dorado AAD Go/PostgreSQL divergente';
    END IF;

    IF vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
           acreditacion
       ) IS DISTINCT FROM convert_to(fixture.preimagen_atestacion, 'UTF8')
       OR encode(sha256(convert_to(
              fixture.preimagen_atestacion, 'UTF8'
          )), 'hex') <>
          manifiesto ->> 'huella_preimagen_atestacion_sha256'
       OR vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
           acreditacion
       ) IS DISTINCT FROM convert_to(fixture.preimagen_revalidacion, 'UTF8')
       OR encode(sha256(convert_to(
              fixture.preimagen_revalidacion, 'UTF8'
          )), 'hex') <>
          manifiesto ->> 'huella_preimagen_revalidacion_sha256'
       OR encode(sha256(
              vec_bolsa_convocatorias.firma_base64url_borrador_v1(
                  acreditacion -> 'firma_atestacion_kms' ->>
                      'firma_base64url_sin_relleno'
              )
          ), 'hex') <>
          manifiesto ->> 'huella_firma_atestacion_sha256'
       OR encode(sha256(
              vec_bolsa_convocatorias.firma_base64url_borrador_v1(
                  acreditacion -> 'firma_revalidacion_kms' ->>
                      'firma_base64url_sin_relleno'
              )
          ), 'hex') <>
          manifiesto ->> 'huella_firma_revalidacion_sha256'
       OR encode(sha256(
              vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
                  acreditacion
              )
          ), 'hex') <>
          manifiesto ->> 'huella_acreditacion_sha256'
       OR vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
              acreditacion_reordenada
          ) IS DISTINCT FROM
          vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
              acreditacion
          ) THEN
        RAISE EXCEPTION 'vectores dorados KMS Go/PostgreSQL divergentes';
    END IF;

    -- Negativos: omision, sustitucion y reordenacion no pueden confundirse.
    IF vec_bolsa_convocatorias.aad_canonica_borrador_v1(
           aad - 'perfil_cifrado_ref'
       ) IS NOT NULL
       OR vec_bolsa_convocatorias.aad_canonica_borrador_v1(
           jsonb_set(aad, '{version_ref}', '"otro#1"'::jsonb)
       ) IS NOT DISTINCT FROM convert_to(fixture.aad_canonica, 'UTF8')
       OR vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
           acreditacion - 'recibo_ref'
       ) IS NOT NULL
       OR encode(sha256(
              vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
                  jsonb_set(acreditacion, '{revision_confirmada}', '3'::jsonb)
              )
          ), 'hex') = manifiesto ->> 'huella_acreditacion_sha256'
       OR vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
              jsonb_set(
                  acreditacion, '{atestacion_ref}', '"atestacion:otra"'::jsonb
              )
          ) IS NOT DISTINCT FROM
          convert_to(fixture.preimagen_atestacion, 'UTF8')
       OR vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
              jsonb_set(
                  acreditacion, '{comprobacion_kms_ref}',
                  '"comprobacion:kms:otra"'::jsonb
              )
          ) IS NOT DISTINCT FROM
          convert_to(fixture.preimagen_revalidacion, 'UTF8') THEN
        RAISE EXCEPTION 'un negativo criptografico se acepto como vector dorado';
    END IF;
END
$vectores$;

ROLLBACK;

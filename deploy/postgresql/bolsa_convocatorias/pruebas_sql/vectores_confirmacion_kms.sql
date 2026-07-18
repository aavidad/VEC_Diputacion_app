-- Vectores cruzados con vectores_dorados_borradores_test.go. Los literales no
-- se regeneran desde las funciones SQL: cambiar uno exige versionar el
-- contrato criptografico de forma deliberada en Go y PostgreSQL.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $vectores$
DECLARE
    aad jsonb := $json$
{"esquema":"bolsa.convocatoria.borrador.aad.v1","version_ref":"proceso:bolsa:auxiliar-2026-1#1","version_revision":1,"huella_version_sha256":"c99741e60ea930a7ff55e0ff85fddee6a4d4b547ba84cb2711548d89b0171d50","esquema_material":"bolsa.convocatoria.intencion.v2","huella_material_sha256":"1d9559ac3a4587ec9e4c71c50008c8e8c60b7debae8a6b5de096fa63f9237190","perfil_cifrado_ref":"perfil:cifrado:borradores:v1","perfil_cifrado_version":1,"huella_perfil_cifrado_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","algoritmo_aead":"A256GCM","algoritmo_envoltura_clave":"A256KW","evidencia_perfil_ref":"evidencia:resolucion-perfil-cifrado:001","evidencia_perfil_version":1,"huella_evidencia_perfil_sha256":"d69bc655b54f22213a2d03811fd92c07da2296c06ad6d534b2d8186cb959e936","decision_politica_ref":"decision:politica-cifrado:borradores:001","decision_politica_version":1,"huella_decision_politica_sha256":"20ca8cc2c31dc1c9e3a8bfa9063109eab1507aa7578a121d4f09b824a0441e8e","localizador_esquema":2,"localizador_dominio":"localizador","localizador_clave_ref":"clave:hmac:convocatorias:localizador:v3","localizador_generacion":3,"localizador_hmac_sha256":"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe","huella_solicitud_esquema":2,"huella_solicitud_dominio":"huella_solicitud","huella_solicitud_clave_ref":"clave:hmac:convocatorias:huella:v3","huella_solicitud_generacion":3,"huella_solicitud_hmac_sha256":"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927","revision_diario":1,"cercado_diario":1,"arrendamiento_inicia_en":"2026-07-18T09:00:00.007Z","arrendamiento_vence_en":"2026-07-18T09:02:00.007Z","atestacion_sellado_ref":"atestacion:motivo:001","atestacion_sellado_version":1,"huella_atestacion_sellado_sha256":"5555555555555555555555555555555555555555555555555555555555555555","token_consumo_sellado_ref":"consumo:motivo:001","huella_correlacion_sha256":"b7f3f6944cba8392fdc5245ffb3ea2def4fc6e7ca132ffd635e894b3c11eba93","procedencia_esquema":"vec.acto.procedencia.v1","perfil_ejecucion":"pruebas","autoridad_acto":"autoritativo","proveedor_procedencia_ref":"proveedor-pruebas","migrable_produccion":true}
$json$::jsonb;
    aad_esperada text := $canon$
{"esquema":"bolsa.convocatoria.borrador.aad.v1","version_ref":"proceso:bolsa:auxiliar-2026-1#1","version_revision":1,"huella_version_sha256":"c99741e60ea930a7ff55e0ff85fddee6a4d4b547ba84cb2711548d89b0171d50","esquema_material":"bolsa.convocatoria.intencion.v2","huella_material_sha256":"1d9559ac3a4587ec9e4c71c50008c8e8c60b7debae8a6b5de096fa63f9237190","perfil_cifrado_ref":"perfil:cifrado:borradores:v1","perfil_cifrado_version":1,"huella_perfil_cifrado_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","algoritmo_aead":"A256GCM","algoritmo_envoltura_clave":"A256KW","evidencia_perfil_ref":"evidencia:resolucion-perfil-cifrado:001","evidencia_perfil_version":1,"huella_evidencia_perfil_sha256":"d69bc655b54f22213a2d03811fd92c07da2296c06ad6d534b2d8186cb959e936","decision_politica_ref":"decision:politica-cifrado:borradores:001","decision_politica_version":1,"huella_decision_politica_sha256":"20ca8cc2c31dc1c9e3a8bfa9063109eab1507aa7578a121d4f09b824a0441e8e","localizador_esquema":2,"localizador_dominio":"localizador","localizador_clave_ref":"clave:hmac:convocatorias:localizador:v3","localizador_generacion":3,"localizador_hmac_sha256":"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe","huella_solicitud_esquema":2,"huella_solicitud_dominio":"huella_solicitud","huella_solicitud_clave_ref":"clave:hmac:convocatorias:huella:v3","huella_solicitud_generacion":3,"huella_solicitud_hmac_sha256":"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927","revision_diario":1,"cercado_diario":1,"arrendamiento_inicia_en":"2026-07-18T09:00:00.007Z","arrendamiento_vence_en":"2026-07-18T09:02:00.007Z","atestacion_sellado_ref":"atestacion:motivo:001","atestacion_sellado_version":1,"huella_atestacion_sellado_sha256":"5555555555555555555555555555555555555555555555555555555555555555","token_consumo_sellado_ref":"consumo:motivo:001","huella_correlacion_sha256":"b7f3f6944cba8392fdc5245ffb3ea2def4fc6e7ca132ffd635e894b3c11eba93","procedencia_esquema":"vec.acto.procedencia.v1","perfil_ejecucion":"pruebas","autoridad_acto":"autoritativo","proveedor_procedencia_ref":"proveedor-pruebas","migrable_produccion":true}
$canon$;
    acreditacion jsonb := $json$
{"acreditacion_ref":"acreditacion:kms:confirmacion:001","arrendamiento_inicia_en":"2026-07-18T09:00:00.007000Z","arrendamiento_vence_en":"2026-07-18T09:02:00.007000Z","atestacion_emitida_en":"2026-07-18T09:00:00.011000Z","atestacion_ref":"atestacion:kms:borrador:001","atestacion_valida_hasta":"2026-07-18T09:04:00.011000Z","cercado":1,"clave_maestra_ref":"clave:kms:borradores:v1","comprobacion_kms_ref":"comprobacion:kms:persistencia:001","confirmacion_solicitada_en":"2026-07-18T09:00:00.012000Z","confirmada_en":"2026-07-18T09:00:00.012003Z","esquema":"bolsa.convocatoria.borrador.acreditacion-kms-confirmacion.v1","estado":"confirmada","firma_atestacion_kms":{"algoritmo_firma":"Ed25519","firma_base64url_sin_relleno":"ejv2we_0n4wzR2t6161HISw99BPQmHutxJ6SfD8fFgBaFJGh7fBitqqNvJno6j7ue8GA-j18GQNuaUvYud5GBQ","huella_clave_publica_sha256":"dae7b96f2766f6ec4a82cb806b5de86f8a80bb55b2525fb5899f025fc7ee1453","huella_preimagen_sha256":"19208c645928cf9c9ded5966f2a9d03dab9f13ecff33d4f43cf739de4b5262a0","verificador_ref":"verificador:kms-emisor-prueba:v1"},"firma_revalidacion_kms":{"algoritmo_firma":"Ed25519","firma_base64url_sin_relleno":"Nk02lywVzSBjWrjbmE-tZx0WWcnRjjAP3mb3fokl6lQr1Ag7foTAW9Pdh_cKYPNmtire-Bgwx_HQD05K3cxaAQ","huella_clave_publica_sha256":"b48c7a6250118e1556efed4b0c837f67c4557d835fc0f24bab4d8c5bae50919c","huella_preimagen_sha256":"9767b5af7654b5ee9963cef30d5869f3a1beffbe54c3cb18f57efcaecb18bf06","verificador_ref":"verificador:kms-revalidacion-prueba:v1"},"huella_aad":"95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c","huella_acreditacion_sha256":"ca54a859c80cf90b1dbd17214b797d34261f6d8c2427e588cd7d5f5688299b83","huella_comprobacion_kms_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","huella_cuerpo_recibo_sha256":"899dffa8f86829470c203320ce37945f2750171732f2c53bda7263e698a7cfd8","huella_envoltura_sha256":"b49d7d4067999aec587566fdd08d06c910e2037cfe0fd7b81874b4e954fb650d","huella_sobre_sha256":"328f371674c16b2a178f829d9fe01b19aebcff2b168b481b79c3b59a6fc1975c","identidad_primaria":{"huella_solicitud":{"clave_ref":"clave:hmac:convocatorias:huella:v3","dominio":"huella_solicitud","generacion_clave":3,"hmac_sha256":"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927","version_esquema":2},"localizador":{"clave_ref":"clave:hmac:convocatorias:localizador:v3","dominio":"localizador","generacion_clave":3,"hmac_sha256":"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe","version_esquema":2}},"perfil":{"algoritmo_aead":"A256GCM","algoritmo_envoltura_clave":"A256KW","huella_contenido_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","referencia":"perfil:cifrado:borradores:v1","version":1},"procedencia":{"autoridad":"autoritativo","esquema":"vec.acto.procedencia.v1","migrable_produccion":true,"perfil_ejecucion":"pruebas","proveedor_ref":"proveedor-pruebas"},"recibo_ref":"recibo:convocatoria:001","revalidacion_solicitada_en":"2026-07-18T09:00:00.012001Z","revalidada_en":"2026-07-18T09:00:00.012002Z","revision_confirmada":2,"revision_reserva":1,"transaccion_ref":"transaccion:convocatoria:001","verificador_ref":"verificador:acreditacion-kms:v1","version_acreditacion":1,"version_atestacion":1,"version_clave":1}
$json$::jsonb;
    preimagen_a_esperada text := $canon$
{"Esquema":"bolsa.convocatoria.borrador.atestacion-kms.v1","AtestacionRef":"atestacion:kms:borrador:001","Estado":"vigente","PerfilRef":"perfil:cifrado:borradores:v1","HuellaPerfil":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","AlgoritmoAEAD":"A256GCM","AlgoritmoEnvoltura":"A256KW","ClaveRef":"clave:kms:borradores:v1","HuellaAAD":"95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c","HuellaEnvoltura":"b49d7d4067999aec587566fdd08d06c910e2037cfe0fd7b81874b4e954fb650d","HuellaSobre":"328f371674c16b2a178f829d9fe01b19aebcff2b168b481b79c3b59a6fc1975c","VerificadorRef":"verificador:kms-emisor-prueba:v1","AlgoritmoFirma":"Ed25519","HuellaClavePublica":"dae7b96f2766f6ec4a82cb806b5de86f8a80bb55b2525fb5899f025fc7ee1453","ProcedenciaEsquema":"vec.acto.procedencia.v1","PerfilEjecucion":"pruebas","Autoridad":"autoritativo","ProveedorProcedenciaRef":"proveedor-pruebas","VersionAtestacion":1,"PerfilVersion":1,"VersionClave":1,"MigrableProduccion":true,"EmitidaEn":"2026-07-18T09:00:00.011Z","ValidaHasta":"2026-07-18T09:04:00.011Z"}
$canon$;
    preimagen_b_esperada text := $canon$
{"Esquema":"bolsa.convocatoria.borrador.revalidacion-kms.v1","AtestacionRef":"atestacion:kms:borrador:001","Estado":"autorizada","HuellaAAD":"95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c","HuellaCuerpoRecibo":"899dffa8f86829470c203320ce37945f2750171732f2c53bda7263e698a7cfd8","ComprobacionRef":"comprobacion:kms:persistencia:001","HuellaComprobacion":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","AlgoritmoFirma":"Ed25519","VerificadorRef":"verificador:kms-revalidacion-prueba:v1","HuellaClavePublica":"b48c7a6250118e1556efed4b0c837f67c4557d835fc0f24bab4d8c5bae50919c","AlgoritmoAtestacion":"Ed25519","VerificadorAtestacion":"verificador:kms-emisor-prueba:v1","HuellaClaveAtestacion":"dae7b96f2766f6ec4a82cb806b5de86f8a80bb55b2525fb5899f025fc7ee1453","HuellaPreimagenAtestacion":"19208c645928cf9c9ded5966f2a9d03dab9f13ecff33d4f43cf739de4b5262a0","HuellaFirmaAtestacion":"aad85bf858ca95e9b6401ffb9a6f458818a628666ee00120f0fa90ae9158d6d2","VersionAtestacion":1,"Revision":1,"Cercado":1,"Identidad":{"localizador":{"version_esquema":2,"dominio":"localizador","clave_ref":"clave:hmac:convocatorias:localizador:v3","generacion_clave":3,"valor_hmac_sha256":"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe"},"huella_solicitud":{"version_esquema":2,"dominio":"huella_solicitud","clave_ref":"clave:hmac:convocatorias:huella:v3","generacion_clave":3,"valor_hmac_sha256":"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927"}},"ArrendamientoVenceEn":"2026-07-18T09:02:00.007Z","ConfirmacionSolicitadaEn":"2026-07-18T09:00:00.012Z","RevalidacionSolicitadaEn":"2026-07-18T09:00:00.012001Z","ComprobadaEn":"2026-07-18T09:00:00.012002Z"}
$canon$;
    aad_reordenada jsonb;
    acreditacion_reordenada jsonb;
BEGIN
    aad_esperada := btrim(aad_esperada, E'\n\r');
    preimagen_a_esperada := btrim(preimagen_a_esperada, E'\n\r');
    preimagen_b_esperada := btrim(preimagen_b_esperada, E'\n\r');
    SELECT jsonb_object_agg(clave, valor ORDER BY clave DESC)
      INTO aad_reordenada
      FROM jsonb_each(aad) AS e(clave, valor);
    SELECT jsonb_object_agg(clave, valor ORDER BY clave DESC)
      INTO acreditacion_reordenada
      FROM jsonb_each(acreditacion) AS e(clave, valor);

    IF vec_bolsa_convocatorias.aad_canonica_borrador_v1(aad)
          IS DISTINCT FROM convert_to(aad_esperada, 'UTF8')
       OR encode(sha256(convert_to(aad_esperada, 'UTF8')), 'hex') <>
          '95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c'
       OR vec_bolsa_convocatorias.aad_canonica_borrador_v1(aad_reordenada)
          IS DISTINCT FROM convert_to(aad_esperada, 'UTF8') THEN
        RAISE EXCEPTION 'vector dorado AAD Go/PostgreSQL divergente';
    END IF;
    IF vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
           acreditacion
       ) IS DISTINCT FROM convert_to(preimagen_a_esperada, 'UTF8')
       OR encode(sha256(convert_to(preimagen_a_esperada, 'UTF8')), 'hex') <>
          '19208c645928cf9c9ded5966f2a9d03dab9f13ecff33d4f43cf739de4b5262a0'
       OR vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
           acreditacion
       ) IS DISTINCT FROM convert_to(preimagen_b_esperada, 'UTF8')
       OR encode(sha256(convert_to(preimagen_b_esperada, 'UTF8')), 'hex') <>
          '9767b5af7654b5ee9963cef30d5869f3a1beffbe54c3cb18f57efcaecb18bf06'
       OR encode(sha256(
              vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
                  acreditacion
              )
          ), 'hex') <>
          'ca54a859c80cf90b1dbd17214b797d34261f6d8c2427e588cd7d5f5688299b83'
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
       ) IS NOT DISTINCT FROM convert_to(aad_esperada, 'UTF8')
       OR vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
           acreditacion - 'recibo_ref'
       ) IS NOT NULL
       OR encode(sha256(
              vec_bolsa_convocatorias.acreditacion_kms_canonica_borrador_v1(
                  jsonb_set(acreditacion, '{revision_confirmada}', '3'::jsonb)
              )
          ), 'hex') =
          'ca54a859c80cf90b1dbd17214b797d34261f6d8c2427e588cd7d5f5688299b83'
       OR vec_bolsa_convocatorias.atestacion_kms_preimagen_borrador_v1(
              jsonb_set(
                  acreditacion, '{atestacion_ref}', '"atestacion:otra"'::jsonb
              )
          ) IS NOT DISTINCT FROM convert_to(preimagen_a_esperada, 'UTF8')
       OR vec_bolsa_convocatorias.revalidacion_kms_preimagen_borrador_v1(
              jsonb_set(
                  acreditacion, '{comprobacion_kms_ref}',
                  '"comprobacion:kms:otra"'::jsonb
              )
          ) IS NOT DISTINCT FROM convert_to(preimagen_b_esperada, 'UTF8') THEN
        RAISE EXCEPTION 'un negativo criptografico se acepto como vector dorado';
    END IF;
END
$vectores$;

ROLLBACK;

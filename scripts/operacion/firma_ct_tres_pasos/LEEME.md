# Firma con certificado en VEC

Dirección instala y prueba este corte. No modifica cidonia ni su arranque por sí solo.

1. Instalar, una vez, las dos migraciones nuevas de `deploy/principal/lista_sql_codexh_firma_tres_pasos_20261013.txt`. Conservar las migraciones y los recibos anteriores.
2. Compilar los ejecutables existentes `cmd/vec-publicar-cargos-firma` y `scripts/operacion/publicar_plantillas_v3`. Publicar el informe previo con `publicar_informe.sh --comprobar` y, después, `--aplicar PUBLICA-INFORME-PREVIO-V4`. Usa `VEC_PLANTILLAS_V3_BIN` y la conexión privada `VEC_PLANTILLAS_V3_DATABASE_URL`. La operación exige cabeza v3, conserva su historia y devuelve el recibo v4.
3. Revisar `cargos.v2.json` (organización, fechas y competencias) y copiarlo a una carpeta privada fuera de Git (`0700`; archivo `0600`). Preparar el plan con `vec-publicar-cargos-firma preparar --cargos <fuente> --plan <plan> --caduca 1h --idioma es --textos <ruta absoluta a textos/es/admin-cargos-firma.json>`. Aprobar su huella por el procedimiento AUT53 y ejecutar `aplicar` con conexión, aprobación y acuse privados. La estructura SQL no publica roles ni asigna perfiles al instalarse.
4. Con Administración y el circuito `vec-cargos-ct` existente, proveer persona, cuenta, vínculo de certificado CA25, cargo y asignación nominal de las cuatro personas. Carmen conserva su certificado; los otros tres se preparan con `certificados.sh --material <material existente> --salida <carpeta privada nueva> --comprobar` y `--aplicar PREPARA-CERTIFICADOS-CT`. Requiere la clave de la CA de desarrollo existente. El resultado contiene certificados, claves/P12 privados, CRL y fragmento Caddy; no asigna perfiles ni instala usuarios web. Revisar el fragmento y añadir las contraseñas web por el procedimiento habitual, sin guardarlas en Git.
5. Publicar el plan nominal de competencias por el gobierno de plan de firma ya existente. Su fichero, referencia, versión y huella deben coincidir con la publicación durable. El circuito v2 usa Técnico RRHH, Jefatura, Diputación delegada y Secretaría. No basta el catálogo de cargos del circuito.
6. Crear `identidad/firma-vec.json` privado con los valores reales resultantes de esas operaciones. Esquema: `version:1`, `autoridad:"no_autoritativo"`, política de autenticación `pga_…` y su SHA-256, `retirada_en` UTC futura, `crl_path` absoluto; `plan:{archivo,referencia,version,huella_sha256}`; cuatro `firmantes` y cuatro `admisiones`. Cada firmante lleva `certificado_sha256,persona_ref,cuenta_ref,cuenta_id,perfil_firma_ref,perfil_consulta_ct_ref,perfil_documentos_ref`. Cada admisión lleva `perfil_ref,referencia,version,huella_sha256,rol_version_ref,rol_sha256,control_revision,control_sha256`. Las huellas y revisiones proceden de los recibos y controles publicados: no se rellenan con valores de muestra.

Los perfiles de firma son `perfil:ct:tecnico_rrhh`, `perfil:ct:jefatura_servicio_rrhh`, `perfil:ct:diputacion_delegada_rrhh` y `perfil:ct:secretaria_general`. Cada persona usa otros dos perfiles distintos para consultar CT y operar Documentos. Sólo el técnico prepara originales; los otros firmantes descargan y custodian su revisión. No se suman permisos entre perfiles.

Añadir al arranque, después de la provisión:

```sh
export VEC_CT_FIRMA_VEC_R5_ENABLED=true
export VEC_CT_FIRMA_REGISTRO_ENABLED=true
export VEC_CT_ORIGINAL_FIRMABLE_ENABLED=true
export VEC_CT_FIRMAS_R5_V2_ENABLED=true
```

Conservar Documentos, GrxFirma, el circuito v2 y la doble llave de desarrollo ya activos. El nuevo selector sólo admite desarrollo. `VEC_CT_FIRMAS_R5_V2_ENABLED` conserva la recuperación histórica; el montaje nominal usa las rutas existentes de original, preflight, registro y consulta. El diagnóstico de un requisito ausente identifica selector y etapa; no anuncia una capacidad disponible.

LOGIN mínimos existentes en la copia recibida: registro de identidad `vec_ct_identidad_registro_desarrollo`; revalidación `vec_ct_identidad_revalidacion_desarrollo`; contexto `vec_ct_contexto_actor_desarrollo`; consulta CT `vec_ct_rrhh_consultas_desarrollo`; motivos RRHH `vec_ct_rrhh_motivos_desarrollo`; ejecución CT `vec_ct_o207_runtime`; registro V3 `vec_autorizacion_o207_registro`. Fuente y motivos generales pueden usar los LOGIN existentes `vec_ct_personal_b2_fuente` y `vec_ct_personal_b2_motivos` si dirección confirma su configuración exclusiva. Las conexiones privadas llevan sus credenciales actuales. No usar los grupos NOLOGIN como usuarios de conexión ni crear otro pool por cada firmante.

Comprobar en la web: informe antes de fiscalización, firmas del técnico y Jefatura, fiscalización con reparo/subsanación, resolución con las tres firmas del circuito y recuperación del mismo recibo después del reinicio. Cada firma requiere un PDF realmente firmado y verificado. La pantalla muestra garantía sustancial y el recibo. Portafirmas sólo puede mostrar enviado tras aceptación real.

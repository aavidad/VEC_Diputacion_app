# Firma con certificado en VEC

Dirección instala y prueba este corte. No modifica cidonia ni su arranque por sí solo.

1. Instalar, una vez, las migraciones nuevas de `deploy/principal/lista_sql_codexh_firma_tres_pasos_20261013.txt`. Conservar las migraciones y los recibos anteriores.
Antes de activar la firma, publicar la correspondencia aprobada de Organización con el LOGIN DBA/migrador existente. La fuente conserva versión, procedencia, vigencia y retirada; no concede ámbitos ni cambia expedientes. Ejecutar con la conexión privada y el archivo de datos:

```sh
psql --set=organizacion_operativa_ref=organizacion:desarrollo:dipgra \
  --set=organizacion_canonica_ref=org_0e4b1d56ba2e24fb8e4808cbc7abb660 \
  --file scripts/operacion/firma_ct_tres_pasos/publicar_correspondencia.sql
```

El LOGIN necesita poder asumir `vec_contexto_actor_v1_propietario` por la provisión DBA vigente; no usar el LOGIN de ejecución CT. La operación devuelve la prueba publicada y no modifica sus versiones anteriores. CA45 → AUT93 → CT214 consumen esa fuente en la transacción de firma; sin correspondencia vigente el circuito permanece cerrado.

2. Compilar los ejecutables existentes `cmd/vec-publicar-cargos-firma` y `scripts/operacion/publicar_plantillas_v3`. Publicar el informe previo con `publicar_informe.sh --comprobar` y, después, `--aplicar PUBLICA-INFORME-PREVIO-V4`. Usa `VEC_PLANTILLAS_V3_BIN` y la conexión privada `VEC_PLANTILLAS_V3_DATABASE_URL`. La operación exige cabeza v3, conserva su historia y devuelve el recibo v4. Copiar `data/catalogos/contratacion-temporal/plantillas_documentos.v4.json` a `/vec-incorporacion/ct_plantillas_documentos.v4.json`, mantener sus permisos privados y añadir `export VEC_CT_PLANTILLAS_SOURCE_PATH=/vec-incorporacion/ct_plantillas_documentos.v4.json` al arranque. Publicar el catálogo en PostgreSQL no cambia esa ruta: sin ella el renderizador puede seguir leyendo otra versión. Conservar los originales ya emitidos; la nueva fuente se usa al preparar un original nuevo.
3. Revisar `cargos.v2.json` (organización, fechas y competencias) y copiarlo a una carpeta privada fuera de Git (`0700`; archivo `0600`). Preparar el plan con `vec-publicar-cargos-firma preparar --cargos <fuente> --plan <plan> --caduca 1h --idioma es --textos <ruta absoluta a textos/es/admin-cargos-firma.json>`. Aprobar su huella por el procedimiento AUT53 y ejecutar `aplicar` con conexión, aprobación y acuse privados. La estructura SQL no publica roles ni asigna perfiles al instalarse.
4. Con Administración y el circuito `vec-cargos-ct` existente, proveer persona, cuenta, vínculo de certificado CA25, cargo y asignación nominal de las personas que firman en el circuito activo. Carmen conserva su certificado; los otros tres se preparan con `certificados.sh --material <material existente> --salida <carpeta privada nueva> --comprobar` y `--aplicar PREPARA-CERTIFICADOS-CT`. Requiere la clave de la CA de desarrollo existente. El resultado contiene certificados, claves/P12 privados, CRL y fragmento Caddy; no asigna perfiles ni instala usuarios web. Revisar el fragmento y añadir las contraseñas web por el procedimiento habitual, sin guardarlas en Git.
5. Publicar el plan nominal de competencias por el gobierno de plan de firma ya existente. Para el circuito RRHH v2, `circuito_ref` debe ser **`vec.contratacion_temporal.circuito_firma:2`**, `circuito_version` debe ser `2` y `circuito_sha256` debe coincidir con la consulta del circuito vigente. No usar el ID sin su versión. Conservar los perfiles lógicos de cada paso y sus referencias completas. Si ya existe una versión publicada del plan con otra referencia, crear y publicar la siguiente versión; no editar la anterior.

Usar `vec-plan-firma-validar preparar` para conservar el catálogo, la traza y el evento canónicos del kit, y su modo de validación para cotejar el SHA-256 del material exacto. El servicio `vec-admin` debe tener `VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE` apuntando a la configuración privada existente del gobierno del plan, con su LOGIN exclusivo de `vec_plan_firma_gobierno_ejecutor`. Enviar cada operación a `POST /api/admin/perfiles/v1/plan-firma/gobierno` como `{"material_base64":"<material validado>"}`: primero `crear` con Administración A, después `publicar` con Administración B, distinta de todas las personas que crearon o editaron esa versión. Conservar ambos recibos. Antes del acto, cada ADMIN selecciona su propio perfil de Aplicación en Administración: consulta `GET /api/admin/seleccion-perfil/v1/propios` y confirma `POST /api/admin/seleccion-perfil/v1/seleccion` con la revisión observada. La selección queda auditada; no se sustituye por una cabecera de perfil. Al activar la versión publicada, actualizar `plan.archivo`, `plan.referencia`, `plan.version` y `plan.huella_sha256` en `identidad/firma-vec.json` con los valores exactos resultantes y reiniciar. Un preflight HTTP 200 con `vias_disponibles: []` todavía no ofrece la firma; debe devolver `certificado_vec` a la persona del paso.

 El archivo indicado por `plan.archivo` contiene el paquete `{version_esquema, fuente, catalogo}` del publicador, no sólo el catálogo. Su referencia, versión y huella deben coincidir con la publicación durable. El caso de presentación de `docs/producto/FLUJO_APP.md` exige Técnico y Jefatura en el informe, y Jefatura, Diputación delegada y Secretaría General en la resolución. Conservar las dos firmas del informe y las tres de la resolución, en el orden del catálogo operativo. La capacidad del registro admite hasta tres pasos, pero no añade pasos al circuito publicado. No basta el catálogo de cargos del circuito.
6. Crear `identidad/firma-vec.json` privado con los valores reales resultantes de esas operaciones. Esquema: `version:1`, `autoridad:"no_autoritativo"`, política de autenticación `pga_…` y su SHA-256, `retirada_en` UTC futura, `crl_path` absoluto; `plan:{archivo,referencia,version,huella_sha256}`; `firmantes` y `admisiones` únicamente para las personas con pasos en ese circuito (Técnico, Jefatura, Diputación delegada y Secretaría General en el caso de presentación). Cada firmante lleva `certificado_sha256,persona_ref,cuenta_ref,cuenta_id,perfil_firma_ref,perfil_consulta_ct_ref,perfil_documentos_ref,perfil_consulta_circuito_ref`. Cada admisión lleva `perfil_ref,referencia,version,huella_sha256,rol_version_ref,rol_sha256,control_revision,control_sha256`. `referencia` identifica la política de admisión publicada (`politica:…:v1`), no la asignación del perfil. Las huellas y revisiones proceden de los recibos y controles publicados: no se rellenan con valores de muestra.

Las referencias lógicas del circuito son `perfil:ct:tecnico_rrhh`, `perfil:ct:jefatura_servicio_rrhh`, `perfil:ct:diputacion_delegada_rrhh` y `perfil:ct:secretaria_general`. El fichero privado lleva los perfiles físicos `prf_…` que devuelve Administración. El plan conserva la correspondencia entre perfil lógico, cargo, rol y selección física; AUT56 comprueba esa selección. Cada persona usa otros tres perfiles distintos para consultar CT, operar Documentos y leer el circuito. Sólo el técnico prepara originales; los otros firmantes descargan y custodian su revisión. No se suman permisos entre perfiles.

Añadir al arranque, después de la provisión:

```sh
export VEC_CT_FIRMA_VEC_R5_ENABLED=true
export VEC_CT_FIRMA_REGISTRO_ENABLED=true
export VEC_CT_ORIGINAL_FIRMABLE_ENABLED=true
export VEC_CT_FIRMAS_R5_V2_ENABLED=true
```

Conservar Documentos, GrxFirma, el circuito v2 y la doble llave de desarrollo ya activos. El cliente de verificación recibe `VEC_FIRMA_VERIFICACION_URL` con el origen HTTPS (sin `/v2/verify`), `VEC_FIRMA_VERIFICACION_CA_FILE` y `VEC_FIRMA_VERIFICACION_TOKEN_FILE` privados. Mantener `VEC_FIRMA_VERIFICACION_ENABLED=true`. AD207 necesita `VEC_AUDITORIA_SELLADO_DATABASE_URL` con el LOGIN existente de membresía exclusiva en `vec_auditoria_encadenador`; sin su latido se rechazan las operaciones auditadas. El nuevo selector sólo admite desarrollo. `VEC_CT_FIRMAS_R5_V2_ENABLED` conserva la recuperación histórica; el montaje nominal usa las rutas existentes de original, preflight, registro y consulta. El diagnóstico de un requisito ausente identifica selector y etapa; no anuncia una capacidad disponible.

LOGIN mínimos existentes en la copia recibida: registro de identidad `vec_ct_identidad_registro_desarrollo`; revalidación `vec_ct_identidad_revalidacion_desarrollo`; contexto `vec_ct_contexto_actor_desarrollo`; consulta CT `vec_ct_rrhh_consultas_desarrollo`; motivos RRHH `vec_ct_rrhh_motivos_desarrollo`; ejecución CT `vec_ct_o207_runtime`; registro V3 `vec_autorizacion_o207_registro`. Fuente y motivos generales pueden usar los LOGIN existentes `vec_ct_personal_b2_fuente` y `vec_ct_personal_b2_motivos` si dirección confirma su configuración exclusiva. Las conexiones privadas llevan sus credenciales actuales. No usar los grupos NOLOGIN como usuarios de conexión ni crear otro pool por cada firmante.

Pasos de pantalla del circuito RRHH v2:

1. El técnico abre Contratación temporal y el expediente; prepara el informe para firma. La pantalla muestra el documento preparado y su recibo.
2. Técnico y después Jefatura entran con sus certificados, abren ese expediente y pulsan «Firmar con certificado». Debe aprobar la operación en AutoFirma. Sólo una verificación aceptada permite mostrar la firma registrada.
3. Intervención fiscaliza el informe firmado. Si hay reparo, RRHH subsana y remite de nuevo por el circuito existente.
4. En nombramiento, el técnico pulsa «Preparar resolución para firma». Jefatura firma primero, Diputación delegada después y Secretaría General al final, cada persona con su certificado.
5. Después de reiniciar, abrir el expediente y comprobar el mismo documento y recibo. Portafirmas sólo muestra enviado tras aceptación real.

En la copia local se comprobó la preparación de la resolución tras subsanación y reemisión: expediente 2026/CT-000127, versión 9, HTTP 200 y recibo `VEC-2026-4`. El botón muestra «Documento preparado; pendiente de envío a firma». Los PDF nuevos llevan cabecera 1.4 y `/ID`; con GrxFirma `aeec0e5` superan la incompatibilidad anterior. Falta publicar el plan v2 con la referencia completa y ejecutar los pasos 2–4 con los certificados de dirección. El ensayo completo de la candidata se coordina con dirección y P; los actos de cidonia los ejecuta dirección con sus propias claves.

La configuración privada de Documentos necesita `custodia_firmado.documentos` con estos dos valores: `informe_definitivo: contratacion_temporal.informe_definitivo_firmado.v1` y `resolucion: contratacion_temporal.resolucion_firmada.v1`. D19 reserva el primer tipo; los catálogos de conservación mantienen sus políticas separadas. Conservar cuentas, motivos, registro externo, almacén y conexiones vigentes de Documentos. Su conexión de registro de autorización requiere un LOGIN separado con membresía exclusiva en `vec_autorizacion_registro`, `INHERIT TRUE` y `SET FALSE`; no reutilizar otro LOGIN cuya membresía permite `SET ROLE`. El informe no se identifica como resolución.

## Publicar el plan v2

Conservar en una carpeta privada `0700` los dos materiales canónicos y sus SHA-256: creación del borrador y publicación aprobada. Los prepara y comprueba `vec-plan-firma-validar`; los actores deben coincidir con los dos ADMIN del entorno. El material preparado en la copia está fuera de Git. Las claves ADMIN de cidonia permanecen allí.

Compilar el validador existente y revisar sin publicar:

```sh
export VEC_PLAN_FIRMA_VALIDAR_BIN=/ruta/privada/vec-plan-firma-validar
scripts/operacion/firma_ct_tres_pasos/publicar_plan_v2.sh --comprobar \
  /ruta/privada/crear-material.json <sha_crear> \
  /ruta/privada/publicar-material.json <sha_publicar> /ruta/privada/salida
```

El resultado `preparado_sin_publicar` conserva los dos cuerpos HTTP y las validaciones en una subcarpeta privada; no conecta con ADMIN. Las siguientes variables sólo se usan al aplicar:

```sh
export VEC_ADMIN_PLAN_ORIGEN=https://admin.cidonia.cloud
export VEC_ADMIN_PLAN_CA=/ruta/privada/ca.crt
export VEC_ADMIN_PLAN_CERT_A=/ruta/privada/admin-editor.crt
export VEC_ADMIN_PLAN_CLAVE_A=/ruta/privada/admin-editor.key
export VEC_ADMIN_PLAN_CERT_B=/ruta/privada/admin-publicador.crt
export VEC_ADMIN_PLAN_CLAVE_B=/ruta/privada/admin-publicador.key
```

Dirección ejecuta el mismo comando cambiando `--comprobar` por `--aplicar PUBLICA-PLAN-FIRMA-CT-V2`. Primero crea el borrador con A; sólo tras su recibo publica con B. El guion verifica TLS, rechaza redirecciones y certificados iguales, conserva las respuestas y exige recibos del mismo catálogo y estado. Si falla la publicación, conservar el borrador y repetir con los mismos materiales y claves de operación; no preparar otra versión para ocultar el fallo.

El LOGIN técnico del pool de plan se provisiona por el kit DBA, como infraestructura. Su única membresía es `vec_plan_firma_gobierno_ejecutor`, con `ADMIN FALSE`, `INHERIT TRUE`, `SET FALSE`; necesita `CONNECT` en la base del entorno y la provisión de origen AD172 del contrato `deploy/postgresql/autorizacion_atestada_v3/PLAN_NOMINAL_FIRMA_AD177.md`. No es el LOGIN de ejecución CT ni uno de los ADMIN. No instalar de nuevo AD200 para restaurar una ACL perdida al copiar la base.

Después del recibo publicado, actualizar la versión, archivo y SHA del plan privado y reiniciar. En el caso de presentación, Técnico y Jefatura deben ver `certificado_vec` en sus pasos del informe. En la resolución deben verlo Jefatura, Diputación delegada y Secretaría General, en ese orden. Dirección conserva los recibos del registro y comprueba la fiscalización y la recuperación final. Mantener `VEC_CT_REINCORPORACION_TITULAR_ENABLED=false` en esta entrega.

El guion envía `Origin` con la URL HTTPS configurada y las cabeceras
`Sec-Fetch` de una petición del mismo origen. Son obligatorias en la entrada
de Administración; sin ellas el POST se rechaza antes de crear o publicar.
La URL no admite espacios ni caracteres de control.

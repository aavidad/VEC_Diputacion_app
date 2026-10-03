# Fuentes iniciales de Administración — borrador AUT39

Contrato acordado antes de programar. Dirección ordenó detener el trabajo por
créditos. No hay migración AUT39 escrita ni ejecutada, ni cambio en el bootstrap.
AUT39 está reservada para las fuentes iniciales; AUT38 queda para el bootstrap
sucesor en otra minitarea y entrega.

El ensayo estructural de AUT36/37 está cerrado sobre
`2ae1403539d2949fcca1cffd65430a22c0162e7d`. La copia fría post36/37 tiene SHA256
`ab445cfc80a1f63affbee9f926c0c08f32f29361cc14c89702e814cee951d412`.
Esas migraciones tienen instalación e historia en esa postimagen: no reaplicar
UP/DOWN ni alterar sus archivos para corregirla. El cuerpo de PR del ensayo
estructural está preparado fuera de Git; no acredita operación favorable.

## Plan y autoridad

Entrada prevista:
`vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(plan_canonico text,huella_aprobada text)`.
El plan es `PlanFuentesInicialesAdminV1`, separado de `PlanBootstrapAdministracionV3`.
Los tipos de K2 están en `b940094e0`, pendientes de validación completa.

Orden del plan: `version`, `operacion_ref`, `preparado_en`, `caduca_en`, `entorno`,
`alcance_fuente`, `procedencia`, `organizacion`, `personas`, `fuente_hmac`,
`politica_admin`. Entorno `desarrollo`, alcance `sintetico_declarado`, dos personas,
instantes UTC con segundos y canon Go de hasta 64 KiB.

Organización: `organizacion_ref`, `version_esperada`, `vigente_hasta`.
Persona: `persona_ref`, `version_esperada`, `vigente_hasta`,
`operacion_cuenta_ordinaria_ref`, `operacion_cuenta_privilegiada_ref`,
`fuente_titularidad`. Las versiones esperadas iniciales son cero. Cada evidencia
contiene `referencia`, `version` y `huella_sha256`.

Política: `politica_ref`, `host_admin`, `ca_sha256`,
`huella_aprobacion_sha256`, `maxima_edad_revocacion_segundos`, `vigente_hasta`.
La aprobación de política es distinta del SHA del plan para evitar dependencia
circular. El plan no contiene nombres, claves, material HMAC, referencias de
cuenta forzadas, perfiles, roles ni estado de continuidad.

LOGIN, proceso, configuración, preimagen y aprobación quedan fuera del plan.
AUT39 deberá exigir un LOGIN exclusivo y configuración privada positiva del DBA
que apruebe el plan, preimagen completa, fuentes y vigencia sintéticas. El CLI
prepara y coteja; no concede esa autoridad. La aplicación sigue cerrada.

El material HMAC es canónico y privado, separado del plan y ligado a
`fuente_hmac.huella_sha256`. Su custodia en la configuración privada del owner
de AUT39 queda pendiente de implementación y revisión. No incluir claves HMAC,
datos civiles, nombres ni DSN en recibos, auditoría o logs.

## Fachadas propietarias acordadas con K4

CA33:

- `vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(jsonb)` → JSON.
- `vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(jsonb,jsonb,text,text,text,text)`
  con plan, recibo IS, SHA de preimagen CA, operación, SHA del plan y aprobación.

IS15:

- `vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(jsonb,text)`
  con plan y material HMAC canónico privado → JSON.
- `vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(jsonb,text,text,text,text,text)`
  con plan, material privado, SHA de preimagen IS, operación, SHA del plan y aprobación.
- `vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text)`
  con operación, SHA del plan y aprobación → evidencia real; acceso solo CA y AUT owners.

AUT deberá cotejar ambas preimágenes antes del primer efecto. Aplicará IS primero,
usando `provisionar_cuenta_v1` con los HMAC privados existentes. IS devolverá cuatro
referencias de cuenta reales, política y titularidad sin perfil. CA cotejará el
recibo por la fachada IS y creará Persona, organización, procedencia, proyecciones
y titularidad sin perfil. Cualquier divergencia revierte toda la transacción.

Recibos propios, diez claves acordadas: `esquema`, `version`, `recibo_ref`,
`operacion_ref`, `plan_sha256`, `aprobacion_ref`, `alcance_fuente`,
`registrada_en`, `datos`, `huella_sha256`. La huella compromete la representación
JSONB del documento sin `huella_sha256`. La fecha es UTC con microsegundos.
Los datos contienen referencias reales y sus fuentes, sin HMAC ni datos civiles.

## Auditoría AD174 acordada con K1

`vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)`
devuelve `auditoria_ref`, `secuencia`, `huella_sha256`, `correlacion_ref`,
`registrada_en`. Solo pueden invocarla los owners de AUT y AD3.

Orden exacto de sus diecinueve cadenas: `tipo_registro`, `evento_ref`,
`operador_login`, `plan_ref`, `plan_sha256`, `preimagen_sha256`,
`configuracion_sha256`, `aprobacion_ref`, `alcance_fuente`, `accion`, `recurso_ref`,
`resultado`, `motivo_ref`, `proceso`, `canal`, `finalidad_ref`, `correlacion_ref`,
`fuente_ref`, `fuente_sha256`.

Familia `provision_fuentes_iniciales_admin`; acción
`provisionar_fuentes_iniciales_admin_v1`; canal `operacion_tecnica_privada`;
finalidad `provision_fuentes_iniciales_admin`; resultado `permitido`; alcance
`sintetico_declarado`. El actor técnico es `session_user`, gobernado por la
configuración. Actor humano, perfil y decisión V3 permanecen nulos.

El material usa `vec.auditoria.fuentes-iniciales.v1` y el eslabón
`vec.auditoria.eslabon.fuentes-iniciales.v1`, con referencia `aud_v3_f_<32 hex>`.
Evento y correlación conservan `evento_<32 hex>` y `correlacion_<32 hex>`.
`fuente_sha256` será la huella del agregado canónico mínimo de los dos recibos
reales CA/IS. No se etiqueta como cifrado un material que no lo está.

El append ocurre dentro de la misma transacción SERIALIZABLE del consumidor.
AUT39 guardará recibo y operación única y solo confirmará después de COMMIT.
Un replay exacto conservará resultado; otro SHA deberá rechazarse. No inventar
denegaciones o errores después de un ROLLBACK como si esa provisión se hubiera
registrado. No cambia roles, asignaciones ni control de continuidad.

## Pendientes al detener el trabajo

Implementar AUT39 con los helpers propietarios finales CA33/IS15 y AD174, cerrar
el canon mínimo del agregado y sus vectores, preparar pruebas y lista causal,
obtener dos revisiones y ensayar en el único clon autorizado. No comenzar AUT38
en esta entrega. Su futura entrada deberá usar titularidad sin rol y conservar
la historia de AUT37 mediante una migración sucesora.

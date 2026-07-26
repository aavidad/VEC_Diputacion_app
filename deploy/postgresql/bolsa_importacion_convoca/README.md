# Persistencia PostgreSQL de importaciones Convoca

Este corte B1 materializa la persistencia T17 sin añadir API, web ni
composición. El XLS original no se almacena en PostgreSQL: el acta exige una
referencia opaca `fichero_custodiado_ref` a su custodia externa cifrada e
inmutable, y las filas aceptadas se guardan únicamente como sobres cifrados
opacos.

## Límites de autoridad

- La exportación conserva autoridad `no_autoritativa`.
- Ninguna fila habilita llamamientos, puntuación oficial o contratos.
- La conciliación usa solo una referencia opaca al registro corporativo.
- La autoridad del plazo de conservación es la política durable, versionada y
  encadenada en PostgreSQL. Solo gobernanza puede publicar la versión siguiente;
  el importador no inyecta duración, referencia ni versión.
- El adaptador exige un `ProtectorStagingConvoca`. La composición futura deberá
  proporcionar AEAD, derivación ciega y atestación de fila con tres referencias
  de clave versionadas y distintas, respaldadas por KMS/HSM. Este repositorio no
  incluye un protector productivo local.
- La recuperación queda separada del importador. Aunque el puerto y el rol
  técnico están probados, su habilitación en producto es **NO-GO** hasta que
  exista el wrapper VEC/T13 que autorice cada uso y aporte auditoría de negocio.
- El despliegue integrado también es **NO-GO** si la referencia de custodia no
  procede de una vertical externa verificada.
- No se han incluido datos, ficheros ni referencias productivas.

## Aislamiento

Las cuentas `LOGIN` se aprovisionan fuera de Git. Cada una recibe un único rol:

| Rol | Única capacidad |
| --- | --- |
| `vec_bolsa_importacion_convoca_ejecutor` | Guardar un lote; no puede consultar ni recuperar. |
| `vec_bolsa_importacion_convoca_recuperador` | Consultar estado y recuperar staging cifrado por páginas; reservado al futuro wrapper VEC/T13. |
| `vec_bolsa_importacion_convoca_conciliador` | Registrar una conciliación final e idempotente. |
| `vec_bolsa_importacion_convoca_retencion` | Bloquear/desbloquear y expurgar staging vencido. |
| `vec_bolsa_importacion_convoca_gobernanza` | Publicar la sucesión válida de políticas de retención. |
| `vec_bolsa_importacion_convoca_migrador` | Instalar o revertir mediante el propietario `NOLOGIN`. |

Los roles runtime no reciben privilegios sobre tablas ni secuencias. Solo
ejecutan funciones `SECURITY DEFINER` con `search_path` cerrado, RLS forzada,
transacciones serializables y límites locales de sesión.

## Modelo durable

- `lote`: acta, referencia opaca de custodia, estado de conciliación, estado del
  staging, política/version, plazo calculado por la base, bloqueo, versión
  optimista y huellas exacta y semántica del staging.
- `fila_staging`: nonce, ciphertext, referencias versionadas de cifrado,
  derivación y atestación, huella del ciphertext, HMAC ciego del documento
  enmascarado y HMAC de la fila canónica completa. No contiene nombres o
  documentos en claro.
- `conciliacion`: resultado final por lote, idempotencia y huella.
- `outbox`: evento mínimo de conciliación confirmada, insertado en la misma
  transacción que la conciliación.
- `decision_retencion`: historia append-only de bloqueos.
- `ejecucion_retencion`: recibo idempotente de cada lote de expurgo.
- `historia_estado`: transiciones append-only de importación, conciliación,
  bloqueo y expurgo, con secuencia, huella anterior y cabeza durable.
- `politica_retencion` y `politica_retencion_actual`: autoridad de conservación
  append-only, sucesión encadenada y puntero vigente.

El expurgo elimina solo `fila_staging`. Conserva acta, conciliación, historia,
política y recibos. Una reimportación del mismo SHA-256 devuelve el acta
original sin recrear filas.

## Prueba PostgreSQL 18

Requisitos: Docker, OpenSSL y Go.

```bash
deploy/postgresql/bolsa_importacion_convoca/probar_integracion.sh
```

El runner usa PostgreSQL 18 fijado por digest, TLS `verify-full`, contraseñas
efímeras generadas desde `/dev/urandom` y fixtures sintéticos. Prueba:

- reinicio y recuperación;
- idempotencia, carrera concurrente y conflicto por actor para el mismo SHA-256;
- validación cerrada de tipos JSON, valores `NULL`, actores, nombres y
  versiones compatibles con `bigint`;
- recuperación paginada y cancelable;
- integridad campo a campo de número, esquema, tres referencias de clave, nonce,
  cifrado, derivación y atestación;
- conflicto de conciliación, cadena de historia y outbox transaccional;
- rollback de lote/filas/historia;
- límites de tamaño y cardinalidad, incluida la inserción set-based de 100.001
  filas dentro del presupuesto declarado;
- RLS, privilegios mínimos y segregación;
- política gobernada encadenada, bloqueo, plazo, expurgo, replay y reimportación;
- rechazo del `down` sin confirmación y reversión limpia.

La reversión destructiva exige:

```text
vec.confirmar_destruccion_bolsa_importacion_convoca
= DESTRUIR_HISTORIA_IMPORTACION_CONVOCA_IRREVERSIBLE
```

Los errores del adaptador son códigos internos saneados. No añaden texto
visible; una futura superficie deberá traducirlos mediante claves i18n.

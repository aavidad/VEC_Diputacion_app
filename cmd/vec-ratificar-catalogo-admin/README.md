# Ratificación técnica del catálogo ADMIN v7

Esta CLI consume un plan aprobado externamente y llama a `vec_autorizacion.ratificar_catalogo_admin_v7(text,text)`. El DBA prepara antes una fila vigente en `vec_autorizacion.config_ratificacion_catalogo_admin_v1` para un LOGIN exclusivo del grupo `vec_admin_ratificacion_catalogo_admin_ejecutor`. La CLI no crea ese LOGIN, la configuración ni la aprobación.

```text
vec-ratificar-catalogo-admin \
  -plan /ruta-privada/plan.json \
  -conexion /ruta-privada/conexion.json \
  -aprobacion /ruta-privada/aprobacion.json \
  -acuse /ruta-privada/recibo.json \
  -idioma es
```

Las cuatro rutas deben ser absolutas y distintas. Los tres archivos de entrada son regulares, propiedad del UID ejecutor, `0600`, sin enlaces simbólicos y situados fuera de un repositorio Git en un directorio `0700` del mismo UID. El directorio del acuse cumple las mismas condiciones; el archivo de acuse debe ser nuevo. La CLI lo reserva antes de llamar a PostgreSQL y lo guarda como `0600` únicamente tras confirmar la transacción. No escriba planes, aprobaciones, DSN ni recibos en Git.

`conexion.json` contiene `dsn` y `permitir_socket_desarrollo` (booleano). Para TCP, la conexión debe usar TLS con verificación de servidor. El socket Unix requiere `permitir_socket_desarrollo:true` explícito. El DSN debe identificar la base y el LOGIN técnico. La CLI rechaza parámetros de sesión libres, incluido `role` y `options`; no hace `SET ROLE`.

`aprobacion.json` contiene `huella_plan_sha256`, `aprobacion_ref` y `aprobacion_sha256`. Deben proceder de la aprobación externa y coincidir con la configuración inmutable del DBA. `huella_plan_sha256` es el SHA256 hexadecimal de los bytes UTF-8 exactos de `plan.json`. El plan debe ser el texto canónico `jsonb::text` de PostgreSQL: tiene las 12 claves del esquema `vec.admin.ratificacion-catalogo.v1`, versión `0600`, una `operacion_ref` `rca_` y los siete descriptores ADMIN ordenados por `accion_ref`. La CLI comprueba los campos y PostgreSQL coteja el plan aprobado, la preimagen, el catálogo y la vigencia antes de escribir.

La llamada usa una transacción `SERIALIZABLE READ WRITE` con zona UTC, límite total de 15 segundos y espera de bloqueo de 2 segundos. La fachada SQL verifica de nuevo que el LOGIN tiene una sola pertenencia admisible al grupo técnico y que la configuración DBA es vigente; esa tabla no concede lectura al LOGIN. En un resultado permitido, la CLI coteja el recibo y el asiento de intento con el plan, la aprobación y el LOGIN antes de `COMMIT`. Guarda en el acuse la respuesta `{estado,codigo,recibo,replay,auditoria_intento}`. La salida estándar solo indica el estado y si fue una recuperación.

La fachada también puede devolver `denegado` o `error` con `recibo:null` y un asiento de intento. La CLI valida ese asiento, confirma la transacción para conservar la auditoría y comunica el rechazo por stderr. El acuse reservado se elimina sin crear un recibo. Un error SQL fuera de esa respuesta revierte la transacción y queda sin confirmar.

Un error de conexión, autorización, SQL o cotejo devuelve un código opaco en stderr y no confirma éxito. Si falla `COMMIT`, el resultado es `commit_indeterminado`: se debe consultar o recuperar con **la misma operación y los mismos bytes aprobados**, después de establecer qué ocurrió. La CLI no reintenta automáticamente. Si PostgreSQL confirma pero falla la escritura del acuse, `acuse_no_guardado` indica `confirmado:true`; hay que recuperar el recibo, sin crear otro plan.

La ratificación incorpora los siete descriptores prospectivos al catálogo ADMIN v7. No concede permisos RBAC a una persona ni acredita una aprobación que no se haya aportado por el circuito externo.

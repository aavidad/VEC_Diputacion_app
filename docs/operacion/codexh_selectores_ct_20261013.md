> Histórico: informe del 13/10/2026. Conserva evidencia de aquel corte; sus pasos de publicación en cidonia no son instrucciones vigentes. Desde el 15/10 se ensaya en copias locales; consulte la [operación actual](README.md).

# Arranque de los selectores de firmas y reincorporación CT

El parche identifica el selector, la etapa y el requisito que detienen la composición. Conserva la causa original para `errors.Is/As` y evita imprimir su mensaje libre. La línea estructurada `contratacion temporal: selector no compuesto` precede al fatal; este conserva un componente preciso y el mensaje catalogado `bootstrap: fuente configurada no disponible`.

R5 valida primero la configuración y el preflight del registrador dedicado de intentos. Así, un registrador ausente o no autorizado detiene estas rutas antes de publicar sus motivos y su perfil. Reincorporación consulta los seis objetos requeridos en una sola lectura y señala el primero que falta; distingue la ausencia de objeto de un fallo de la consulta o del permiso sobre CT153.

No se ha confirmado aún cuál es la dependencia que falla en el proceso de cidonia: faltan las líneas previas al fatal. La copia fría del 13/10 contiene las funciones de CT130, CT134, CT153 y AD98; también el preflight y el registro AD169. La función larga de AD98 figura truncada a 63 bytes en el dump, como corresponde a PostgreSQL. Esto no justifica reinstalar esas migraciones.

La consulta de solo lectura sobre el clon PostgreSQL 18 de dirección confirma ambas asignaciones activas, con el acto fijo esperado, un ámbito y dos/tres concesiones, vigentes de 2026 a 2036. CT153 tiene `EXECUTE` para el grupo ejecutor. Existe una configuración interna del registrador. Estos resultados no cotejan el archivo privado ni el LOGIN y proceso que usa el servicio que falló.

## Configuración común

Conservar las conexiones privadas existentes, sus LOGIN separados y la identidad nominal RRHH. Las variables siguientes son las llaves del perfil sintético; no conceden permisos de usuario por sí solas:

```dotenv
VEC_EXECUTION_PROFILE=desarrollo
VEC_AUTH_MODE=desarrollo
VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
VEC_CT_FIRMAS_R5_V2_ENABLED=true
VEC_CT_SEGUIMIENTO_CESE_ENABLED=true
VEC_CT_REINCORPORACION_TITULAR_ENABLED=true
VEC_BOLSA_BORRADORES_ENABLED=true
```

Debe existir el material privado indicado por `VEC_DEVELOPMENT_MATERIAL_DIR`. Conservar las conexiones de `VEC_CT_DATABASE_URL`, `VEC_CT_GOBIERNO_DATABASE_URL` y `VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL`, y las de consulta, registro/revalidación de identidad y contexto actor ya utilizadas por RRHH. No copiar DSN a este documento ni al log.

Para el seguimiento, declarar los tres catálogos en las variables existentes:

- `VEC_CT_REGLAS_SOURCE_PATH`.
- `VEC_CT_CAUSAS_CESE_SOURCE_PATH`.
- `VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH`.

Las rutas se mantienen cerradas sin una asignación publicada exacta y vigente. Se conserva la provisión administrativa existente. Si el aviso de arranque muestra `pendiente_provision`, dirección debe revisar la asignación de ese entorno antes de aprobar su huella mediante `VEC_CT_PROVISION_PERFILES_RRHH_APROBACION` y `VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES`. Nunca se aprueba automáticamente una asignación revocada o un ámbito de otro circuito. Retirar ambas variables después de la provisión CAS.

## Registrador de intentos de R5

Crear fuera de Git `auditoria-intentos.json` directamente bajo el directorio de material privado, con la protección y propietario que exige el lector seguro. Usar el proceso aprobado para ese entorno; `vec-rrhh` es solo un valor sintético de ejemplo:

```json
{
  "esquema": "vec.auditoria.intentos.servidor.v1",
  "dsn_file": "auditoria-intentos.dsn",
  "proceso": "vec-rrhh",
  "canal": "interna_corporativa",
  "limite_segundos": 10
}
```

`auditoria-intentos.dsn` contiene una única conexión privada con TLS válido y sin destinos alternativos. Debe apuntar a la misma base que CT. Su LOGIN es distinto de ejecución, gobierno y registro de autorización y tiene exclusivamente una membresía en `vec_autorizacion_atestada_v3_registrador_intentos`, con `ADMIN FALSE`, `INHERIT TRUE` y `SET FALSE`. El LOGIN necesita `LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`, sin `rolconfig` ni concesiones de negocio. El preflight común exige `TEMPORARY` denegado tanto a `PUBLIC` como al LOGIN de aplicación; no conceder ese privilegio.

La provisión DBA debe conservar una fila inmutable en `vec_autorizacion_atestada_v3.configuracion_runtime_intentos` que ligue ese LOGIN al mismo `proceso` y al canal `interna_corporativa`. Una fila existente se coteja; no se actualiza ni se borra para hacer pasar el arranque. El código no crea el LOGIN ni esta fila. `roles_intentos_up.sql`, CA26, IS13 y AD169 son prerrequisitos ya existentes; solo se instalan los que falten en un entorno nuevo.

Comprobación de solo lectura conectada con el LOGIN dedicado, sin `SET ROLE`:

```sql
BEGIN READ ONLY;
SELECT session_user = current_user,
       vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(
         'vec-rrhh', 'interna_corporativa');
ROLLBACK;
```

Ambos resultados deben ser `true`. La configuración del proceso en el ejemplo debe sustituirse por el valor aprobado antes de usar la consulta.

## Reincorporación

El ejecutor necesita CT130, CT134, CT153 y los consumidores AD92/AD98 con sus dependencias. CT153 debe devolver `organizacion_ref`; su `EXECUTE` está concedido a `vec_contratacion_temporal_ejecutor`. La consulta de presencia no acredita ese permiso: el arranque ejecuta además la fachada CT153 con el LOGIN real.

Se reutiliza el perfil fijo `reincorporacion_titular`, su contexto nominal registrado y su sesión propia derivada de RRHH. La provisión existente comprueba el acto de asignación, la versión de rol, las concesiones y la preimagen; el selector no aporta una identidad ni un permiso.

La ventana final de diagnóstico, con el mismo material y la aplicación anterior detenida, localizó el fallo restante en la política Bolsa: `bolsa_borrador_politica_desarrollo.go:249`, `version_rol esperado=7 / recibido=8`. La asignación preparada es v2 y está dentro del rango admitido; `prepararInstantanea` no devuelve error. El rol v7 ya está publicado y asignado a otro perfil, mientras el perfil de `identidad/bolsa-bback.json` sigue en rol v5. La preparación avanza a v8 y la guarda detiene el arranque antes de publicar por CAS. El salto previsto es v5→v7; activar ofertas para forzar v8 no es una corrección.

Dirección decidió entregar con `export VEC_CT_REINCORPORACION_TITULAR_ENABLED=false`. Así arranca la aplicación local y recupera el recibo tras reiniciar. No se cambió el código de Bolsa, no se reutilizó el perfil de otra persona y no se debilitó la guarda.

Para activarlo después, dirección debe:

1. Leer `identidad/bolsa-bback.json` del material activo y la asignación vigente de su `perfil_ref`; conservar su preimagen CAS, persona, unidad, ámbito, bolsas y procedencia.
2. Mediante el acto ADMIN existente, asignar a **ese mismo perfil** la versión v7 ya publicada del rol Bolsa, con su referencia y huella reales. El endpoint existente es `/api/admin/perfiles/v1/actos-ordinarios`; si el catálogo ADMIN exige propuesta sensible, usar `/api/admin/perfiles/v1/propuestas` y su cierre por otra persona. La autoridad decide el circuito, no el arranque. Conservar el recibo y comprobar el nuevo puntero de asignación; no crear una versión v8 ni transferir el perfil ajeno.
3. Cotejar y actualizar el archivo privado con los datos de esa provisión, manteniendo `0600` y el propietario del proceso. Su esquema sólo admite `version`, `autoridad`, `sujeto`, `certificado_sha256`, `perfil_ref`, `unidad_ref`, `ambito_ref` y `bolsas_ref`: **no lleva campos de versión de rol ni de asignación**. Si se conserva el mismo perfil y ámbito, el acto durable cambia la asignación y esos valores del archivo se conservan. No añadir campos inventados para indicar v7.
4. Con los catálogos de seguimiento declarados, añadir `export VEC_CT_REINCORPORACION_TITULAR_ENABLED=true`, reiniciar y comprobar arranque y consulta con el LOGIN ejecutor CT vigente. Si vuelve a rechazar una preimagen o procedencia, conservar el diagnóstico y el recibo; no sustituirlos por una asignación manual.

La provisión y el arranque con ese selector activo quedan a cargo de dirección; no se acreditan en esta entrega.

## Alcance y comprobación pendiente

Este corte modifica el diagnóstico y el orden del preflight de R5, sin añadir SQL ni nuevas autoridades. `VEC_CT_FIRMAS_R5_V2_ENABLED` compone consulta y recuperación nominales; encenderlo no acredita por sí solo las dos vías de registro, la firma de dos personas ni un PDF firmado.

La cabeza `1e36d923f` compila con CGO desactivado y conserva verdes las pruebas focales de originales y renderizado. Su revisión sensible dio GO. La CI anterior `38345199047` terminó con las cinco puertas verdes; la CI de la cabeza final se comprobará antes de entregar. No se ha accedido a cidonia.

## Preparación comprobada en la copia local

PostgreSQL 18.4 restaurado desde el dump del 13/10, certificados y material privados; aplicación local con R5 v2, Documentos, circuito RRHH v2 y fuente de plantillas v4. Chrome del sistema, sin respuestas HTTP simuladas.

La resolución de 2026/CT-000127, versión 9 después de reparo, subsanación y reemisión, se prepara por el botón de la ficha. La respuesta del original fue HTTP 200 en 126 ms, con `numero_vec: VEC-2026-4`, fecha `2026-10-13T05:53:08.11566Z` y SHA-256 `db878712c7e403a625621e9ef16a3b2ff11a0b26b6410d119146118d93c4cc11`. La pantalla muestra «Documento preparado; pendiente de envío a firma» y el recibo. A 1280 px no hay desbordamiento horizontal global. Este tiempo corresponde a recuperar el original custodiado; no acredita que todas las consultas comunes estén por debajo de 100 ms.

El informe nuevo v5 usa la fuente v4: 29.699 bytes, SHA-256 `b98e80ea37d32da25fbcfa46af3f5d231d517a3339ecef652bd47dd42f9e143c`; las tres frases retiradas no aparecen en su texto. La publicación de plantillas en PostgreSQL debe acompañarse de la ruta de archivo indicada en el LEEME. Los originales anteriores se conservan intactos.

Después de reiniciar el contenedor PostgreSQL propio y la aplicación, Chrome abre de nuevo la ficha y pulsa el botón. Recupera HTTP 200 en 116 ms, el mismo `VEC-2026-4`, la misma fecha y el mismo SHA-256. La pantalla conserva el estado preparado y no presenta desbordamiento horizontal. No se acredita una firma aceptada ni el recorrido posterior de fiscalización con este resultado.

## Diagnóstico anterior de GrxFirma

GrxFirma canónico permanece sin cambios, en `80e7dd5ceb4baab07e238f012ca8fa6eaeff8407`. Su verificador HTTPS real usa los certificados, CA y CRL privados. El registro nominal VEC llega al verificador y responde `422 firma_no_verificada`; no hay firma registrada ni recibo de firma acreditado.

Todas las variantes siguientes parten del mismo informe original de 29.699 bytes:

| Productor | Salida | SHA-256 | Dictamen real |
| --- | ---: | --- | --- |
| CLI GrxFirma actual | 68.978 bytes | `391ba0782a10fdebc438718ae6ae88f3a50fdc7f203977dbb6ec061c33a4ff5c` | `indeterminada`, `pdf_no_comprobado`, sin firmas reconocidas |
| AutoFirma oficial, `alwaysCreateRevision=true` | 85.593 bytes | `5671526dea8ca3553bffc03374e177e42038be2ac78a293a6005049ddedbb593` | `indeterminada`, `identificador_pdf_modificado`, una firma reconocida |
| GrxFirma actual más el arreglo existente `978ace06d6ba22da9a0fffac4be6758c51005eeb`, sólo en herramienta privada | 38.238 bytes | `b909ce725b693946bdb548745ae800b6d7bea56f263a403e18c490082783e1c2` | `no_valida`, `cambios_no_permitidos` / `catalogo_modificado`, una firma reconocida |

El CLI actual reintenta por tamaño CMS y acaba escribiendo dos mitades idénticas con referencias xref incoherentes. El original VEC no contiene campo de firma ni reserva `/Contents`: la reserva pertenece al productor GrxFirma. El arreglo ya existente evita ese reintento en este caso; no modifica VEC ni su verificador.

La salida incremental de AutoFirma conserva el prefijo original, pero añade `/ID` cuando el original no lo tenía. La salida con el arreglo GrxFirma conserva el prefijo y tiene integridad, cadena, certificado y CRL válidos; añade `/Version /1.5` al catálogo de un PDF con cabecera 1.3. Los campos originales Type, Pages, Lang y Names se conservan. El productor está en `third_party/pdfsign/sign/pdfcatalog.go:31`; el clasificador `internal/adapters/outbound/common/signer/pades_v2_changes.go:230` normaliza esa promoción sólo desde cabecera 1.4. Alberto debe decidir la corrección del producto GrxFirma. No se rebajan guardas, no se alteran originales emitidos y no se declara firmado un documento rechazado.

La verificación se solicitó a `/v2/verify` con contrato `autofirmav2.dictamen-verificacion.v2`, PDF firmado y PDF original. PDF, dictámenes íntegros y captura quedan en la carpeta privada de ensayo para dirección; no se incluyen certificados, claves ni tokens en Git.

## Continuación del 13/10: productor PDF y plan nominal

El commit `b8093c919` añade una cabecera PDF 1.4 y dos elementos `/ID` iguales y deterministas al productor común. La inserción queda después de `xref`: conserva sus desplazamientos. La prueba de compatibilidad reconstruye exactamente los bytes anteriores al retirar únicamente `/ID` y restaurar la cabecera 1.3; el texto, las fuentes y la estructura se conservan. Los originales ya emitidos se recuperan de custodia antes de volver a renderizar. Se actualizaron las 31 huellas de pruebas afectadas por los nuevos bytes. Los dos paquetes focales pasan; revisión sensible exacta GO, Semgrep local sin hallazgos y gosec sin hallazgos ni errores de carga en el paquete PDF.

Con la CLI y el verificador GrxFirma de `aeec0e5` suministrados por dirección, el informe nuevo de 29.314 bytes supera la verificación. El registro nominal llega al paso posterior y responde `403 acceso_denegado`: el plan privado conserva `circuito_ref` sin `:2`, mientras el circuito vigente devuelve `vec.contratacion_temporal.circuito_firma:2`. El preflight devuelve HTTP 200 y ninguna vía disponible. La comparación exacta del plan impide acreditar la competencia; no se modifica esa guarda ni se presenta el documento como firmado.

Dirección ordenó publicar una nueva versión del plan por la API ADMIN existente, con dos personas distintas, también en la copia local. El LEEME recoge el endpoint, el material validado, la referencia completa y los campos privados que deben actualizarse después de publicar. La copia tiene los dos perfiles ADMIN, pero sus certificados no están en el material local. Dirección mantiene las claves en cidonia y ejecutará allí el acto y la firma final. El LOGIN del gobierno del plan se prepara en la copia por el kit DBA autorizado, como infraestructura. No se han añadido filas del plan por SQL ni se ha accedido a cidonia.

El primer alta del original tardó 707 ms y acumuló 311 ms de base; el preflight tardó 406 ms. Estos resultados no cumplen los objetivos de 300 ms y 100 ms. Quedan pendientes el registro, la fiscalización posterior, las dos firmas de la resolución, la recuperación y la medida del recorrido final.

El guion `publicar_plan_v2.sh --comprobar` valida los dos materiales con la herramienta existente, coteja sus SHA-256 al preparar los cuerpos y comprueba que corresponden al mismo plan v2 y a personas distintas. Conserva los cuerpos HTTP exactos sin enviarlos. En la copia, la comprobación emitió `preparado_sin_publicar` para `ct.plan.firma.desarrollo:2`. Los materiales y sus fuentes canónicas permanecen en la carpeta privada de dirección; no se incluyen en Git. Por orden de dirección de las 12:30, la llamada con los certificados ADMIN y el recorrido final de firma se ejecutarán en cidonia después de HZ37. Esta entrega no afirma que ese acto ni las firmas hayan ocurrido.

La provisión técnica reutiliza una copia acotada del preparador DBA del kit ADMIN del 05/10. En la copia de 55449 creó únicamente `vec_adm_plan_firma_hfirma_20261013`, con una membresía exclusiva en `vec_plan_firma_gobierno_ejecutor` y `ADMIN FALSE`, `INHERIT TRUE`, `SET FALSE`. Restauró `CONNECT` en la base al grupo existente (`false` antes, `true` después) y los cuatro orígenes técnicos AD172. La conexión real del LOGIN pasó `verify-full` y confirmó TLS activo. Se conservan DSN, SQL operativo y comprobaciones en privado. No se añadieron identidades ADMIN ni filas de negocio. El acto requiere además la confianza V3 de gobierno del kit activo y los certificados nominales; ambos se conservan y configuran en cidonia por dirección.

## Corrección de CT214 tras la revisión de dirección

La revisión independiente del 13/10 a las 15:00 ensayó las nueve SQL después de la cadena HZ37 y CT215. Detectó que CT214 añadía la prueba a `registrar_firma_con_plan_v2`, retirada en CT181; la aplicación usa `registrar_firma_con_plan_v4` de CT185. La versión corregida conserva la tabla inmutable y sustituye esa entrada v4 por su definición final literal, con guarda SHA-256 de la definición instalada. El literal se obtuvo de ese cuerpo mediante cuatro sustituciones de marcas únicas, verificadas antes de prepararlo. Mantiene el consumo de plan, la llamada a `registrar_firma_verificada_v3`, las ACL y la vigencia; fija `search_path` a `pg_catalog, pg_temp`. Escribe la prueba con la firma nueva y coteja la histórica en los reintentos, sin crear evidencia para firmas anteriores.

CT214 no tiene historia de firmas en la principal, por lo que dirección autorizó corregir el mismo archivo antes de instalarlo. No se reaplicó en la copia de H ni se ejecutó DOWN. La lista conserva nueve archivos y actualiza sólo su huella. Los paros indican la clave, el valor recibido y el esperado. Dirección hará el ensayo y la revisión sobre esta versión final antes de incorporarla a la subida; la comprobación anterior no acredita el nuevo cuerpo.

La CI rechazó la primera corrección por reconstruir dinámicamente la función. La versión final usa `CREATE OR REPLACE FUNCTION` explícito, con `search_path` fijo y el mismo anclaje de preimagen; no se rebajó el verificador. La comparación del literal con el cuerpo instalado y las cuatro sustituciones coincide, normalizando sólo el salto final de línea. El verificador focal de `search_path` pasó para la única SQL cambiada; el ensayo de dirección se pedirá con la nueva huella de la lista.

La revisión posterior de dirección detectó dos fallos en CA45: faltaba `USAGE` del esquema al propietario CT y la rama canónica intentaba acceder al registro `v` sin asignar. La versión final concede sólo ese acceso al esquema (el `EXECUTE` ya existía) y separa el cotejo de la correspondencia en un `IF prueba IS NOT NULL`. Conserva el resto de guardas y no concede acceso directo a tablas.

La prueba focal `correspondencia_organizacion_canonica_000045.sql` ejecutó la definición corregida dentro de una transacción de la copia PG18.4, como propietario CT. El resolvedor devolvió la misma referencia canónica y `prueba: null`. Después del `ROLLBACK` se comprobaron la huella original y la ausencia de `USAGE`: no quedó ningún cambio de definición, ACL ni datos de negocio. No se reaplicó CA45. La lista actualiza su huella antes del ensayo final de las nueve SQL por dirección.

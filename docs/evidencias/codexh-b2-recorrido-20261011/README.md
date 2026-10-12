# Recorrido de Contratación temporal hacia Personal — 11 de octubre de 2026

El expediente `2026/94014` alcanzó la preparación del informe jurídico, versión 5. Intervención intentó registrar el resultado desde su propia pantalla y recibió `409 firma_remision_pendiente`. La fiscalización quedó pendiente. No se registró una firma ni una incorporación en Personal.

El corte posterior, sobre `e4346d1141c0717e4bd2d61854d46c5fe66996dc`, publicó ocho entradas de régimen y modalidad por HTTP y registró el vínculo de la categoría RPT del mismo expediente. Ambos resultados se conservaron tras reiniciar aplicación y PostgreSQL. El GET del plan B2 sigue en `409 preparacion_pendiente`.

## Árbol y entorno comprobados

El recorrido se ejecutó en una copia local aislada de HZ19 y HZ20, con PostgreSQL 18.4 y material de desarrollo conservado. No se actuó sobre cidonia ni sobre la principal. No se reaplicaron migraciones instaladas ni se ejecutó DOWN.

La fuente del servidor fue `126bdaca9acbf937265709ba313cd1ab7dabadb0`, compilada con Go 1.26.9. El binario instalado tiene SHA-256 `2efb47cfd4ead2ce39f9847f2965394cf12708ec7a71e72f69110d93b6f6065b`; `go version -m` acredita esa revisión y `vcs.modified=false`. La primera muestra de tiempos, anterior al recorrido de 94014, empleó la misma fuente con Go 1.26.6 y SHA-256 `e1967028b621691ca18d021eb6bbf1032fb69ad16b18ace0ed0590df41a03c88`.

El binario recibido inicialmente indicaba otra revisión y un árbol modificado. Se sustituyó antes del caso 94014. La compilación desde el worktree anidado también incorporaba metadatos Git de la raíz; una copia Git separada permitió verificar la revisión correcta.

Se usó Chrome del sistema mediante Playwright y certificados mTLS distintos para solicitante, ratificador, RRHH e Intervención. La aplicación mantuvo su guarda de escucha en loopback. Un túnel local del clon permitió acceder desde Chrome. El transporte de prueba envió cada solicitud original mediante curl con la CA y el certificado correspondientes: `route.fetch` añadía una cabecera `Connection` que la frontera HTTP rechaza. No se retiraron cabeceras de autoridad del navegador ni se desactivó TLS. Esta adaptación debe conservarse como limitación de la automatización.

La asignación de alta de RRHH se actualizó mediante la provisión nominal existente, con aprobación local y comparación de la preimagen vigente del clon. Las variables de provisión se retiraron después. La consulta de preferencias quedó en `503`: su cuenta configurada no está instalada en la autoridad de sesiones. Por ello se realizó el recorrido manual con los contratos de los guiones existentes, sin cambiar su preflight. Los actores de centro proceden de la consulta autorizada y el actor de alta, de la actuación persistida por CT; la sesión del shell no se utilizó como concesión.

## Caso 94014 y respuestas reales

La petición solicita el puesto 9 de Administrativo de Transformación Digital, centro 520, categoría `categoria:rpt:administrativo`, grupo C1, una persona, del 10 de noviembre al 10 de diciembre de 2026. La RPT pública del repositorio identifica ese puesto y categoría. La bolsa administrativa del clon tiene 41 participaciones, 26 de ellas disponibles. La propuesta de cobertura autorizada devolvió `viable` para `bolsa_vigente`.

La tabla recoge el primer corte del caso, antes de la configuración de 26 operaciones y la carga de categorías descritas en la extensión posterior.

| Operación | HTTP | Resultado |
| --- | ---: | --- |
| Presentar la petición del centro | 200 | Petición v1 |
| Ratificar con otra identidad | 200 | Petición v2 |
| Crear expediente en RRHH | 201 | 2026/94014, v1 |
| Registrar análisis | 201 | v2 |
| Confirmar cobertura por bolsa | 201 | v3 |
| Asignar a la unidad responsable | 201 | v4 |
| Preparar informe jurídico | 201 | v5, documento y recibo |
| Consultar candidatos de la bolsa | 503 | `servicio_no_disponible` |
| Registrar fiscalización con RRHH | 403 | Operación denegada |
| Registrar fiscalización desde Intervención | 409 | `firma_remision_pendiente` |
| Consultar vínculo de categoría RPT | 200 | Análisis legible; vínculo ausente |
| Listar categorías RPT del circuito B2 | 403 | Permiso pendiente |
| Consultar plan B2 | 409 | Preparación pendiente |

La referencia del expediente y los campos mínimos de cada recibo están en [recibos.json](recibos.json). Los cuerpos completos, comandos originales y capturas permanecen en la evidencia privada del clon. La entrada de Intervención usa la referencia y versión remitidas; su pantalla no depende de disponer del cuadro de RRHH.

El llamamiento, el vínculo CT–Bolsa, la aceptación, la formalización y la incorporación B2 no se completaron. El HTTP 409 de fiscalización comprueba la guarda de firma pendiente; no acredita una llamada al proveedor de firma.

## Recuperación

Se reiniciaron PostgreSQL y la aplicación del clon. Los siete comandos originales se recuperaron con las mismas claves y material. Sus respuestas fueron `200/200/200/201/201/201/201`. Las referencias, fechas y versiones de los recibos coinciden con las iniciales: [comparación](recuperacion.json).

Las seis tablas de historia comprobadas conservaron exactamente sus filas y SHA-256 antes y después del reinicio y los replays: [antes](historia-antes.json), [después](historia-despues.json). Permanecen 74 altas, 178 versiones, 104 actuaciones, 157 filas de outbox, 10 revisiones de peticiones y 4 entregas. Incluyen la historia anterior del clon y los casos de este recorrido.

Chrome volvió a consultar el detalle v5 con HTTP 200. Se observaron cero errores JavaScript, cookies, localStorage, sessionStorage e IndexedDB en esa comprobación. Las pantallas capturadas a 1440 y 390 px no presentan desbordamiento global salvo el informe jurídico: su JSON ocupa 8059 px a 390 px. Ese defecto ya figuraba en el acta del 10 de octubre y se ha reproducido en esta fuente. No se acredita una revisión completa de accesibilidad.

## Puertas que faltan

La lectura de candidatos falla tanto desde la pantalla como mediante curl con TLS y mTLS verificados. Las filas de staging identifican la clave histórica con sufijo `26818203`; la clave maestra conservada tiene sufijo `df9ff1d2`. Los archivos KMS e idempotencia del clon coinciden con el archivo original de ficheros HZ19. Ningún archivo de 32 bytes de ese archivo corresponde a la referencia histórica. La diferencia se registra en [coherencia-kms.json](coherencia-kms.json). No se regeneraron claves ni se modificaron los datos cifrados.

Personal conserva una publicación representativa de 32 puestos. Para el centro 520 solo incluye los puestos 21 y 22-520-001 de analista-programador; falta el puesto 9 de administrativo y el nodo publicado de `unidad:centro:520`. La incorporación requiere una publicación adicional de organización histórica por la autoridad existente. El script instalado de organización 2026 y sus filas se conservaron.

Antes del caso definitivo se conservaron otros dos expedientes: 94012, Programador, quedó en análisis v2 porque esa bolsa no está cargada; 94013, Auxiliar Administrativo, quedó en alta v1 al comprobar que esa categoría no tenía puesto en la RPT del centro 520. Un intento de rectificar 94012 recibió 403 y no cambió su análisis. No se asignaron esos expedientes ni se usaron como sustitutos del caso 94014.

## Tiempos y escala

Las lecturas directas se midieron secuencialmente, 20 veces por ruta, con TLS, mTLS y auditoría reales. p95 usa el rango 19 de las 20 muestras ordenadas. Los tiempos incluyen el servidor y el tránsito local; no incluyen el transporte de Chrome. [Muestras con Go 1.26.6](http-baseline-go1266.json), [lecturas adicionales con Go 1.26.9](http-adicional-go1269.json).

| Ruta y estado de partida | HTTP | p50 ms | p95 ms | Go |
| --- | ---: | ---: | ---: | --- |
| Cuadro, límite 100 | 200 | 245,44 | 253,43 | 1.26.6 |
| Detalle 2026/CT-000127, v9 | 200 | 180,83 | 195,37 | 1.26.6 |
| Plan B2 de ese expediente | 409 | 191,69 | 248,30 | 1.26.6 |
| Lista de bolsas | 200 | 43,91 | 44,72 | 1.26.6 |
| Borradores disponibles, expediente v9 | 200 | 235,72 | 281,45 | 1.26.9 |
| Propuesta de cobertura, expediente ya tramitado v9 | 503 | 45,72 | 54,92 | 1.26.9 |

Las mediciones del 409 y 503 describen esas puertas; no estiman un plan preparado ni una decisión aplicada. Tampoco son la comparación de rendimiento de las candidatas #1004 y #1012. El [inventario agregado](volumenes.json) conserva la escala observada por propietario: 12 bolsas y 390 participaciones, además de la historia CT indicada. No demuestra carga de producción.

### Comparación exacta #1004 → #1012

Después de recuperar el caso se instaló el binario de `bf2abb8ce76dbee567bed56c5c27dfb32ac09143` (#1004), compilado con Go 1.26.9 y SHA-256 `4a3caca7a0dba6a9f21d8e5eade0d6d924a6794339df6c6749a588c2c947b16f`. Las funciones de Personal41–44 ya coincidían con sus postimágenes, por lo que se conservaron. El kit AD172 ensayó cinco ternas nuevas en ROLLBACK y luego confirmó su alta por el mismo procedimiento. Los perfiles nominales de alta y Personal se actualizaron por CAS de sus preimágenes reales; la aprobación era local y las variables se retiraron.

La comparación usa `GET /api/vec/contratacion-temporal/incorporacion-personal-b2/plan/v1?expediente_ref=expediente:ct:1fe2c0eda847e5e11e4ab7b15953df0b7a19a525264f3cf2381890e3d0d2e85b`, certificado de RRHH y `Accept: application/json`. Ambos binarios usan Go 1.26.9, el mismo PostgreSQL, la configuración de 22 operaciones, el mismo actor y el expediente 94014 en v5. Cada tanda tiene 20 lecturas consecutivas, sin escrituras de negocio entre ellas. El servidor y PostgreSQL tienen cada uno un límite de 2 CPU y 2 GiB. Se sustituyó solamente el binario entre tandas; #1012 conserva los activos web de su base #1004.

| Fuente exacta | HTTP y código en las 20 muestras | p50 ms | p95 ms |
| --- | --- | ---: | ---: |
| [bf2abb8ce, baseline](baseline-bf2-b2.json) | 409 `preparacion_pendiente` | 181,865 | 226,48 |
| [e4346d114, candidata revisada](medicion-e434-b2.json) | 409 `preparacion_pendiente` | 162,135 | 200,65 |

En esta puerta el p95 bajó un 11,4 %. Las referencias de binario, revisiones y muestras están en los JSON enlazados. El resultado mide la lectura que termina en preparación pendiente; no acredita el tiempo del plan preparado, del alta en Personal ni del recorrido B2 completo. La candidata anterior `c4853c1a` fue rechazada y no se midió.

## Reproducción y siguiente corte

Los siguientes cuerpos conservan los datos del caso 94014 y sustituyen solo la clave de idempotencia por `CLAVE_ORIGINAL_EXTERNA`. Para un replay hay que aportar la clave privada original de **cada** operación; una clave nueva no recupera la misma actuación. El alta usa el vínculo único de petición y versión y no trae una clave libre. Las referencias de catálogo y sus huellas corresponden a esta instalación y deben cotejarse con las preparaciones autorizadas.

| Actor mTLS | Método y ruta | JSON exacto del caso | Versión esperada |
| --- | --- | --- | ---: |
| Solicitante | POST `/api/vec/contratacion-temporal/peticiones-centro/operaciones` | [Presentación](comandos/01-presentar.json) | — |
| Ratificador | POST `/api/vec/contratacion-temporal/peticiones-centro/operaciones` | [Ratificación](comandos/02-ratificar.json) | 1 |
| RRHH | POST `/api/vec/contratacion-temporal/peticiones-centro/rrhh` | [Alta](comandos/03-alta.json) | Petición 2 |
| RRHH | POST `/api/vec/contratacion-temporal/analisis/registros` | [Análisis](comandos/04-analisis.json) | 1 |
| RRHH | POST `/api/vec/contratacion-temporal/cobertura/decisiones` | [Cobertura](comandos/05-cobertura.json) | 2 |
| RRHH | POST `/api/vec/contratacion-temporal/asignaciones` | [Asignación](comandos/06-asignacion.json) | 3 |
| RRHH | POST `/api/vec/contratacion-temporal/informes-juridicos/preparaciones` | [Informe](comandos/07-informe.json) | 4 |
| Intervención; RRHH para la denegación | POST `/api/vec/contratacion-temporal/fiscalizaciones/resultados` | [Fiscalización](comandos/08-fiscalizacion.json) | 5 |

El siguiente comando ilustra la recuperación de análisis. `ORIGEN_CLON`, `CA_PRIVADA`, `CERTIFICADO_ACTOR`, `CLAVE_MTLS_ACTOR` y `CLAVE_ORIGINAL_ANALISIS` se aportan fuera de Git. El origen debe ser el loopback del clon reservado. La clave mTLS corresponde a RRHH; para cada otra operación se elige el certificado nominal de la tabla.

```sh
umask 077
cuerpo_privado=$(mktemp)
jq --arg clave "$CLAVE_ORIGINAL_ANALISIS" '.clave_idempotencia = $clave' \
  docs/evidencias/codexh-b2-recorrido-20261011/comandos/04-analisis.json > "$cuerpo_privado"
curl --silent --show-error --max-time 20 \
  --cacert "$CA_PRIVADA" --cert "$CERTIFICADO_ACTOR" --key "$CLAVE_MTLS_ACTOR" \
  --header 'Accept: application/json' --header 'Content-Type: application/json' \
  --data-binary "@$cuerpo_privado" \
  "$ORIGEN_CLON/api/vec/contratacion-temporal/analisis/registros"
rm "$cuerpo_privado"
```

Para la lectura posterior usar POST `/api/vec/contratacion-temporal/expedientes/consultas` con `{"expediente_ref":"expediente:ct:1fe2c0eda847e5e11e4ab7b15953df0b7a19a525264f3cf2381890e3d0d2e85b","version_observada":5}`. La propuesta de cobertura se consulta mediante POST `/api/vec/contratacion-temporal/cobertura/propuesta` con la misma referencia y `version_esperada:2`, en el estado de preparación previo a decidir. Ambas son lecturas y llevan `Accept: application/json` y `Content-Type: application/json`.

El bloqueo de candidatos se reproduce mediante GET `/api/vec/bolsa/bolsas/bolsa:administrativo:2026-09-17/candidatos?limite=50`, con mTLS de RRHH y `Accept: application/json`. El bloqueo de firma se reproduce con el cuerpo de fiscalización y mTLS de Intervención mientras el expediente siga en v5 y no conste firma de remisión. La misma ruta con RRHH devuelve la denegación 403. Estas dos solicitudes no completan la fiscalización.

Reutilizar los contratos de `scripts/recorridos/centro_rrhh/recorrer.py`, `scripts/recorridos/analisis_informe/recorrer.py` y el lector sin firma de `scripts/recorridos/propuesta_firma/recorrer.py`. Los dos primeros guiones necesitan adaptar su ejecución manual a los campos actuales de número de personas, puesto solicitado y número MOAD; sus guardas no se cambiaron.

Para recuperar este caso, aportar fuera de Git el origen del clon, su CA, certificados, comandos y recibos privados originales. Ejecutar únicamente los replays conservados, consultar el detalle y comparar los archivos de historia. Una ejecución nueva debe elegir juntas la categoría, bolsa, puesto, plaza, publicación, unidad y permisos antes de presentar otra petición.

## Extensión instalada de catálogos y vínculo RPT

Esta extensión emplea el servidor `e4346d1141c0717e4bd2d61854d46c5fe66996dc`, Go 1.26.9, SHA-256 `31596db07bfddd408d8a4f0cb161f7646b6d298f916562cf924e74dbfbd12c09`. La comparación de tiempos se realizó antes de esta configuración y carga de datos.

Dirección aprobó expresamente el mapa local de las cuatro operaciones opcionales `ct_vinculo_registrar`, `rpt_listar`, `personal_catalogo_publicar` y `personal_catalogo_retirar` al motivo V2 genérico publicado que ya compartían 21 operaciones. El motivo no concede permisos: actor, perfil, acción, finalidad, recurso y campos siguen sometidos al PDP y consumo V3 correspondientes.

Se activó el publicador existente con `VEC_PERSONAL_B2_GOBIERNO_ENABLED=true`. El preparador oficial de #1004 derivó y contrastó 26 capacidades en una salida nueva: [coordenadas públicas y huellas](material26-validado.json). El LOGIN heredado de gobierno tiene seis membresías y el preparador lo rechazó. Se creó un LOGIN técnico temporal con el contrato mínimo de la prueba oficial del preparador: una membresía en el migrador V3, sin ADMIN ni INHERIT de ese grupo y con SET, cuyo cierre es únicamente migrador y propietario AD3. La provisión tuvo dos revisiones focales, ensayo ROLLBACK con salida 0 y rol ausente, y COMMIT de la misma fuente SHA-256 `949da809dab28a571106507f59c27140e93337bc961f868c840f8b3df9dc003b`: [ensayo](provision-gobierno-ensayo.json), [confirmación](provision-gobierno-commit.json).

El preparador terminó con código 0. Después se deshabilitó ese LOGIN mediante NOLOGIN, se retiró su contraseña, se comprobaron cero sesiones y se borraron el DSN técnico y las entradas privadas de credencial. El LOGIN heredado y sus membresías se conservaron. Se activó la nueva configuración de 26 operaciones y se consumió por CAS la preimagen real del perfil de alta requerida por su plantilla; las variables de provisión se retiraron.

### Ocho publicaciones HTTP

Las ocho solicitudes a POST `/api/vec/contratacion-temporal/incorporacion-personal-b2/catalogos-registro-empleado/v1` devolvieron 201 con mTLS de RRHH. Cada una lleva `Accept: application/json`, `Content-Type: application/json` e `Idempotency-Key: UUID_ORIGINAL_EXTERNO` en la cabecera. El cuerpo no contiene actor, perfil ni organismo.

| Tipo | Entrada y cuerpo reproducible |
| --- | --- |
| Régimen | [Funcionario de carrera](comandos/catalogos-personal/01-funcionario_carrera.json) |
| Régimen | [Funcionario interino](comandos/catalogos-personal/02-funcionario_interino.json) |
| Régimen | [Personal laboral fijo](comandos/catalogos-personal/03-laboral_fijo.json) |
| Régimen | [Personal laboral temporal](comandos/catalogos-personal/04-laboral_temporal.json) |
| Modalidad | [Vacante](comandos/catalogos-personal/05-vacante.json) |
| Modalidad | [Sustitución](comandos/catalogos-personal/06-sustitucion.json) |
| Modalidad | [Programa](comandos/catalogos-personal/07-programa.json) |
| Modalidad | [Acumulación de tareas](comandos/catalogos-personal/08-acumulacion_tareas.json) |

Para reproducir una recuperación se usa el curl anterior, se añade `--header "Idempotency-Key: $CLAVE_ORIGINAL_CATALOGO"` y se envía directamente el JSON correspondiente sin añadir una clave al cuerpo. Cada entrada tiene su propia clave original conservada fuera de Git. La huella del cuerpo se calculó conforme a `HuellaPublicacionCatalogoEmpleadoB2`, que incluye el organismo resuelto por la configuración; las huellas publicadas corresponden a este clon.

Tras reiniciar aplicación y PostgreSQL, los ocho replays devolvieron 200. Todas las entradas y recibos originales, incluidas decisión, auditoría, huella de consumo y fecha, son idénticos: [comparación de ocho recibos](catalogos-personal-recuperacion.json). Personal conserva ocho filas actuales y ocho de historia. El acceso del replay mantiene su auditoría propia; no sustituye el recibo inicial. Estos datos locales no acreditan una aprobación jurídica ni un alta de empleado.

### Publicación de categorías y vínculo

La publicación RPT estaba vacía. Se cargó el archivo de datos existente de la misma fuente `deploy/postgresql/catalogos_configurables/datos/categorias_rpt_v1.sql`, SHA-256 `d4687997d4d71193b32f6692eb2d60cb0818e8e4f87c70faba4c80d0607e418b`, mediante la función CC1 y su propietario. Se conservaron íntegros su documento, referencias técnicas y precondiciones. No se creó SQL ni una versión distinta. Quedaron una publicación, 16 entradas y 16 controles, con huella canónica `ce7c6034ac465ed44e41d8de4347e641efe9e06818ce16c21d97308e3a527da6` y recibo `recibo:catalogos:categorias_rpt:v1`: [carga y procedencia](categorias-rpt-v1-carga.json). Es una carga técnica local basada en la fuente pública; no acredita aprobación jurídica de la RPT.

Después, GET `/api/vec/contratacion-temporal/incorporacion-personal-b2/categorias-rpt/v1` devolvió 200 con las 16 categorías. GET del vínculo del expediente devolvió el análisis v2, su recibo y huella, con el expediente aún en v5. POST `/api/vec/contratacion-temporal/incorporacion-personal-b2/vinculo-categoria-rpt/v1` con [el cuerpo del vínculo](comandos/09-vinculo-rpt.json) devolvió 200 y registró la revisión 1. Fuente y referencia de aprobación técnica proceden del GET de categorías; `motivo_ref` es el recibo real del análisis, igual que en el cliente web existente.

El vínculo conserva `prospectivo:true` y `acredita_procedencia_historica:false`. Tras otro reinicio, su replay devolvió 200 con el recibo íntegro idéntico, misma fecha y revisión; GET de vínculo y categorías también devolvieron 200: [recuperación del vínculo](vinculo26-recuperacion.json). Las seis tablas CT anteriores mantienen sus hashes exactos incluso después de esta extensión: [historia final](historia-tras-extension26.json).

Para repetir la carga de datos en otro clon autorizado, cotejar primero la publicación y ejecutar el archivo inmutable de la fuente exacta con `psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/catalogos_configurables/datos/categorias_rpt_v1.sql`, usando exclusivamente la conexión privada de ese clon. Para el vínculo, aportar fuera de Git su clave original, consultar los dos GET anteriores y enviar el JSON sanitizado con esa clave. No se cambian las referencias de una publicación instalada ni el material de un replay.

La firma de remisión, la clave histórica de Bolsa y la publicación adicional de puesto 9/nodo de Personal siguen pendientes. El último GET del plan B2 devuelve 409; no se preparó, ejecutó ni confirmó un plan, ni se registró alta en Personal. La banda de resultados de este acta separa el recorrido principal, las lecturas medidas y estos actos independientes.

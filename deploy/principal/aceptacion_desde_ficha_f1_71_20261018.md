# Confirmar una aceptación desde la ficha de Bolsa

Tarea F1-71. Estado: aceptación y recuperación tras reinicio comprobadas en copia local; verificaciones finales pendientes. CONFIG NUEVA: ninguna. La función reutiliza el montaje de Bolsa, la sesión certificada, el servicio de situación y el material nominal común.

## Preparación

1. Partir de una copia PostgreSQL 18 de la principal, con propietarios y ACL completos. Usar la lista única vigente `lista_sql_main_tras_hz37.txt` de la candidata conjunta, con AD288 de R antes de B128. Cotejar sus huellas y el registro de migraciones de la copia. No reaplicar SQL instalado ni omitir sus preimágenes. Dirección fija la integración de #1176 antes de #1164; S no modifica la SQL de R.
2. R aporta AD288 congelada en `a32a1d45ba80e5ff5b4ff7a8eaca6608fbaa743d` (#1176) con el consumidor `registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada`, diez argumentos de material V3, acción `bolsa.respuesta_revision.resolver`, audiencia `vec_bolsa_llamamientos.respuesta_revision.resolver.v1`, recurso de participación y tipo `respuesta_revision_bolsa`. Su SHA-256 es `88d014083dd8a7e692b40c7ec617c7b8ebca84c2824ec93d77ea3457670c52b2`. Registro, vigencia y consumo se cotejan en la transacción de B128. El contrato está en `docs/plan_modulos/bolsa_respuestas_en_revision_contrato.md`.
3. Instalar B128 una sola vez, después de AD288, desde la lista única de la candidata conjunta. Añade el acto inmutable y su evento pendiente; el cambio conserva la traza B34. Su SHA-256 es `d0f4ecb2f495c8306f197c1541b7fa1356ad9733825e132fbf3806753d596312`.
4. Admitir la fuente `fuentes/bolsa_confirmacion_aceptacion_rrhh_v1.json` dentro del catálogo completo de Administración. La fuente tiene garantía alta, una acción ordinaria y dimensiones `unidad_ref` y `ambito_ref`; su vigencia de datos es de 90 días. Preparar una propuesta y un cierre con dos administradores distintos mediante el [acto genérico de Bolsa](habilitacion_rol_bolsa_rrhh_generico_20261013.md). Conservar rol, asignación, ámbitos y concesiones anteriores. El archivo de fuente no concede permisos por sí solo.
5. Preparar el origen de consumo nominal para el LOGIN propio de Bolsa, audiencia y operación anteriores, proceso `vec-server` y canal `interna_corporativa`, por el procedimiento técnico vigente. No se crea ese origen al arrancar. Mantener LOGIN, credenciales y material fuera de Git.
6. Con los escritores detenidos, instalar la configuración común y el binario exactos y reiniciar con vigilancia. La adopción conserva la autoridad publicada y sus ámbitos; no publica otra semilla. El proveedor nuevo es opcional: su ausencia deja solo este acto no disponible. Una modificación del rol exige volver a cargar el montaje mediante reinicio; una instantánea revocada u obsoleta debe ser rechazada, nunca dar permiso.

## Un solo recorrido de aceptación

Usar datos ficticios y un llamamiento directo B17 con respuesta B30 `acepta` pendiente de reflejar. Reutilizar la semilla o preparar el dato en la copia, sin atribuirle un acto o una firma reales. Desde Bolsas, abrir la ficha, pulsar «Confirmar aceptación del llamamiento», escribir motivo, revisar y confirmar. Anotar HTTP201, recibo, resolución y paso a pendiente de incorporación. La respuesta original permanece guardada.

Si un envío devuelve 401 o 403, la ficha y el listado se retiran. Solo una nueva lectura autorizada permite recuperar la intención con su cuerpo y clave originales. Una caída de red conserva esa intención para consultar o reintentar.

Repetir la misma operación: HTTP200 y mismo recibo, motivo y fecha. Cambiar el motivo o la fecha con esa clave debe devolver 409 sin otro efecto. Reiniciar aplicación y PostgreSQL; comprobar situación, marca retirada y recuperación del recibo. Otra participación, llamamiento ajeno o permiso revocado deben quedar denegados y auditados. El guion `ensayo_confirmacion_directa_f1_06.sql` comprueba el resultado; no fabrica datos ni permisos.

Comprobar también arranque y consultas existentes cuando falta el consumidor, el origen o el material de este acto. Portal y Bolsa deben seguir disponibles; confirmar aceptación falla de forma aislada. Si falla la vuelta al montaje, restaurar binario y configuración anteriores y reiniciar; conservar las migraciones y la historia. No hay DOWN.

## Ensayo acreditado el 18 de octubre

La candidata conjunta ensayada es `0d41433df5999a054f63e6ee002f2a68dd021ad2`, con la rama S en `4cd8a3ba07065543b41739f28c652492cba5171d`, que incorpora main `4eafc9bc4c51af3a230a67d7d5ee2e6252f279f0`, y AD288 de R con la huella anterior. El binario y los archivos web de esa candidata se arrancaron sobre la copia existente. La base de la copia conserva su identificación `40cc1e991`; arrancar otra candidata no cambia la procedencia de la base.

La copia recibió la cadena base, AD288 y B128 por etapas. CT238 se instaló una sola vez después. Este ensayo acredita ese estado acumulado; no acredita una instalación desde cero de toda la lista vigente en su orden actual. Las conexiones, identidades de sesión y material de acceso permanecen fuera de Git.

La preparación utilizó únicamente datos ficticios: origen de consumo, contacto y datos normativos usados por la configuración de confianza. El llamamiento directo se registró desde RRHH con la operación B17/RRHH019, su correo por el servicio SMTP de la copia y el contacto general efectivo. Se habilitó la lectura de Calendarios que necesita Mi Bolsa para mostrar el plazo. La política de situación se publicó por la operación nativa de la pantalla de reglas: HTTP 201, versión 7. Dos administradores distintos publicaron por B103 el rol de RRHH, versión 11, y su asignación, versión 7; no se concedió ese permiso mediante SQL sobre la persona.

| Comprobación | Resultado acreditado | Evidencia privada |
| --- | --- | --- |
| Aceptar desde Mi Bolsa | Respuesta B30 registrada desde el navegador, HTTP 201. | `aceptacion-aspirante-201.json` |
| Confirmar desde la ficha de RRHH | HTTP 201 con recibo, motivo y fecha; la persona pasa a pendiente de incorporación. Se interrumpió la entrega de la respuesta en Chrome después del efecto para comprobar el resultado incierto. | `confirmacion-native-201-perdida.json`, `confirmacion-201-cuerpo.json` |
| Recuperar tras reiniciar aplicación y PostgreSQL | HTTP 200, misma clave y cuerpo, mismo recibo, fecha y motivo; `reutilizada: true`. | `recuperacion-tras-reinicio.json` |
| Cotejar la historia tras el reinicio | Una confirmación, un evento y un recibo; la marca de aceptación pendiente queda retirada. | `sql-tras-reinicio.log` |
| Cambiar el motivo con la misma clave | HTTP 409, `cambio_en_conflicto`; el recuento posterior conserva una confirmación, un evento y un recibo. | `conflicto-misma-clave.json`, `sql-tras-conflicto.json` |

La candidata posterior `6c134437b5fe67f3fbd3370ea1d0f474453324cd` incorpora el arreglo visual de `214e172f2`: sustituye la referencia larga por el botón común de copiar justificante, sin cambiar Go ni SQL. Se arrancó sobre la misma copia con las lecturas anteriores a 200. Para comprobar la presentación, se cargó el componente desde esa aplicación con el recibo real ya recuperado: español e inglés a 1440 y 390 píxeles, copia exacta con Intro y sin referencia visible ni desbordamiento horizontal. Esta comprobación del componente no repite el acto ni sustituye el recorrido nativo anterior. Evidencias: `componente-recibo-390-en.log`, `componente-recibo-1440-es.log` y capturas `componente-recibo-final-*`.

Antes del envío se comprobó la revisión del motivo y la cancelación desde la ficha: cero peticiones de confirmación. La confirmación final usa la misma clave y cuerpo conservados; no crea otra intención al perderse la respuesta. El acceso al recibo con teclado y la copia se comprobaron en el componente final; no se acredita accesibilidad completa del portal.

Administración siguió disponible durante el reinicio. Después respondieron 200 `/livez`, el portal y la consulta de Bolsa. El indicador global de salud del montaje reducido devolvió 503: este recorrido no acredita disponibilidad global de todos los módulos.

## Límites

Solo aceptación directa. Renuncia simple, oferta publicada y justificación admitida o rechazada esperan sus contratos o la integración de #1147. Una referencia y una huella documentales no acreditan custodia. El evento guardado queda pendiente; no se presenta como aviso enviado ni como incorporación realizada. No hay datos ni servicios corporativos en este ensayo.

Pendientes: arranque y aislamiento del acto sin consumidor, origen o material, y conciliación de la cabeza final con la principal. Las revisiones independientes del acto y la corrección visual son favorables; la integración y su puerta completa siguen pendientes. La instalación desde cero de la lista vigente y la integración siguen pendientes. Las focales Go y web no sustituyen esas comprobaciones. La activación descrita solo acredita la copia local; no acredita instalación en la principal ni en cidonia.

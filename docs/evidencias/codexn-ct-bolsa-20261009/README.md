# Resultado de Bolsa en el expediente de contratación

RRHH puede vincular una emisión existente de Bolsa al expediente mediante una confirmación expresa. La ficha muestra la respuesta de ese llamamiento, su fecha y recibo. Presenta por separado el contacto y el cambio de situación en Bolsa. Una aceptación pendiente de revisión no cuenta como firme.

Para repetir el recorrido: abrir un expediente con cobertura por bolsa vigente; emitir el llamamiento desde el asistente de Bolsa; volver a CT y vincular esa emisión; registrar el contacto efectivo; responder desde Mi Bolsa. Tras una renuncia, abrir desde CT el asistente de Bolsa para seleccionar y confirmar la siguiente persona. Vincular la nueva emisión conserva el llamamiento anterior. La respuesta no se reconstruye por el número visible ni por coincidencia de persona o fecha.

## Ensayo nominal local

PostgreSQL 18 desechable, límite de 2 GB, datos sintéticos y provisión administrativa nominal. Cuatro UP nuevos aplicados una vez sobre el clon causal postHZ15. Sin DOWN, cambios en cidonia ni correo externo. Los certificados y el material de provisión permanecen fuera de Git.

| Caso | Resultado antes y después de reiniciar aplicación y PostgreSQL |
| --- | --- |
| Una plaza y aceptación firme | Un vínculo, un evento de salida, tres versiones; misma respuesta, recibos, fechas y resultado completo. La ficha muestra una aceptación de una plaza; la fase conserva el estado del servidor y la asignación sigue en su trámite real. |
| Renuncia y siguiente persona | Dos emisiones reales B7 y dos vínculos CT, dos eventos de salida, tres versiones. Seis reintentos devuelven 200; recibos, fechas, respuesta y proyección completa idénticos. La nueva emisión queda pendiente y la renuncia anterior permanece visible. |
| Petición v3 anterior cuyo análisis falló | La misma petición, cuerpo y clave completan el análisis tras CT202; el reintento conserva recibo y versión. La huella de los bytes iniciales permanece intacta. |

Tiempos HTTP puntuales en el clon: vínculo 104 ms y reintento 80 ms; ficha con aceptación 163–227 ms; siguiente emisión Bolsa 88 ms, segundo vínculo 82 ms y ficha con ambos llamamientos 139 ms. Tras reiniciar, ficha 212–217 ms. La cobertura gobernada existente tardó 570 ms. Estos valores no son un p95 ni acreditan toda pantalla o todos los volúmenes.

## SQL y provisión

Lista y huellas: [lista SQL ordenada](../../../deploy/principal/lista_sql_codexn_ct_bolsa_20261009.txt). Orden B98, AD233, CT201, CT202. CT200 pertenece a la PR de campos del centro; si ya se integra, instalarla antes de CT201 según la lista.

El acto nuevo usa el perfil fijo `rrhh.contratacion_temporal.alta.v2`. La lectura y el alta anteriores siguen disponibles sin las nuevas migraciones o sin esa provisión. La administración debe aprobar la actualización por huella y CAS; no se conceden derechos por el nombre del puesto.

CONFIG NUEVA: para esa actualización administrativa se reutilizan `VEC_CT_PROVISION_PERFILES_RRHH_APROBACION` y `VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES`. Claude debe tomar el acto aprobado y la huella de la asignación vigente en la principal; los valores del clon no sirven para cidonia. Retirar la activación de la provisión después de comprobarla.

## Validación y límites

Dos revisiones sensibles independientes dieron GO sobre `f04cdcb7b386a6074c4e841c6f14cf7794823e65`; la revisión UI dio GO focal sobre `d2e79d9937e7228e42a4c42ea6380590ee676f06`. Las migraciones conservan las huellas de la lista. La revisión UI dio GO para `aec6ac94b5a181bc517b37f42cce99d0863a79cb`, incluidas navegación y distinción de la retirada posterior. Ambas revisiones sensibles dieron GO final para `c0f5fd5e56f5ec61008eb72e8b00052214b60397`: AD233 literal recupera exactamente la preimagen al quitar su única ampliación y produce la misma función instalada en los clones nominales.

Semgrep local, métricas desactivadas: sin hallazgos nuevos en el delta JavaScript. El análisis íntegro señaló una inserción HTML preexistente del circuito de firma, cuyo cambio en esta rama se limita a URL. Gosec se ejecutó sobre los paquetes afectados: no señaló líneas cambiadas, pero hubo errores de carga de metadatos de dependencias; no se presenta como cobertura completa. Las revisiones sensibles son de fuente, no reproducciones independientes del ensayo.

El primer clon montó material de desarrollo distinto del que protegía su staging histórico y la consulta de candidatos devolvía 503. Al restaurar el material original en el clon de Chrome, devolvió 200 con 41 candidatos en 167 ms, sin modificar SQL ni datos. Las capturas y trazas distinguen este fallo de restauración, las incidencias anteriores de preferencias y los errores JavaScript. No se acredita firma legal, envío externo ni despliegue.

## Chrome y capturas

Chrome del sistema con Playwright, PostgreSQL 18 y certificados de desarrollo. Sin excepciones JavaScript ni desbordamiento horizontal a 1440 y390; idioma ES/EN correcto y ningún catálogo del otro idioma. La emisión desde el asistente y su vínculo al expediente respondieron201/201. La ficha devolvió200 en168ms con la renuncia anterior y la siguiente persona pendiente.

Después de la aceptación se recorrió «Confirmar asignación»:201, versión4, ficha200 en153ms y siguiente paso «Preparar informe jurídico». Es asignación administrativa del expediente aRRHH, no nombramiento de la persona candidata. Las versiones1–3 y la participación aceptante permanecieron idénticas.

- [aceptacion-final-es-1440-resultado.png](aceptacion-final-es-1440-resultado.png)
- [aceptacion-final-es-390-resultado.png](aceptacion-final-es-390-resultado.png)
- [aceptacion-final-en-1440-resultado.png](aceptacion-final-en-1440-resultado.png)
- [renuncia-ui-final-es-1440-llamamiento-1.png](renuncia-ui-final-es-1440-llamamiento-1.png)
- [renuncia-ui-final-es-1440-llamamiento-2.png](renuncia-ui-final-es-1440-llamamiento-2.png)
- [renuncia-ui-final-es-390-llamamiento-1.png](renuncia-ui-final-es-390-llamamiento-1.png)
- [renuncia-ui-final-es-390-llamamiento-2.png](renuncia-ui-final-es-390-llamamiento-2.png)
- [asistente-final-es-1440.png](asistente-final-es-1440.png)
- [asistente-final-es-390.png](asistente-final-es-390.png)
- [asignacion-final-es-1440.png](asignacion-final-es-1440.png)

Rendimiento del navegador antes del ajuste de carga: desde la lista ya abierta,1067ms antes del POST,147ms de petición y1281ms hasta el panel. Desde la URL filtrada y la apertura del módulo,3,3–3,8s. Se cargaban21catálogos CT de varias fases al abrir la ficha. Es una cadena de carga ya existente enmain; T1 no añade otra petición de catálogo. El objetivo de300ms no queda acreditado por esas mediciones.

Persisten incidencias anteriores del clon: tres503 de preferencias, aviso de certificado al cargar un script y consulta de documentos no disponible. Las capturas de los paneles no sustituyen el registro del recorrido completo; este límite se conserva aquí.

## Rendimiento tras la ficha rápida (#964)

Al fusionar main con #964, la ficha ya no espera a descargar su vista para pedir el detalle. Se midió con Playwright y el Chrome del sistema contra un servidor local HTTP/1.1 con dobles sintéticos: 25 ms por estático, 140 ms por API, 25 expedientes, 1440 px y mediana de 7 vueltas, pulsando con la lista abierta y en reposo. El detalle de esta rama incluye una respuesta de Bolsa vinculada (renuncia con contacto y cambio de situación) y la ficha la muestra.

| Clic en la ficha → | esta rama antes de #964 | main con #964 | esta rama con #964 | esta rama con #964 y la precarga en reposo (#966, ensayo local) |
|---|---|---|---|---|
| POST de detalle, en frío | 1118 ms | 433 ms | 401 ms | 169 ms |
| ficha pintada, en frío | 1267 ms | 942 ms | 897 ms | 320 ms |
| panel sin cargas, en frío | 1410 ms | 1086 ms | 1040 ms | 462 ms |
| POST de detalle, con caché | 561 ms | 308 ms | 295 ms | 159 ms |
| ficha pintada, con caché | 706 ms | 451 ms | 439 ms | 305 ms |
| API tras el clic | 7 | 5 | 5 | 5 |
| JS / catálogos tras el clic, en frío | 118 / 21 | 116 / 22 | 118 / 22 | 21 / 2 |

La lectura de Bolsa viaja dentro de la respuesta del detalle, así que no añade peticiones. Añade dos módulos a la vista. La lista tarda lo mismo con y sin esta rama (1743 ms frente a 1749 ms de mediana). Sin errores JS en ninguna vuelta. Bajar de 300 ms en frío depende de la precarga en reposo de #966.

## Ensayo SQL con los límites de espera

B98 y CT201 fijan `lock_timeout` de 5 s y `statement_timeout` de 30 s tras `BEGIN`. `contar_aceptaciones_firmes_ct_v1` rechaza listas de más de 1000 vínculos. La lista completa se ensayó en PostgreSQL 18.4 desechable (`--rm --restart=no --network none`, 2 GB): copia de la base sintética simulada, las 19 SQL de `base_hz9.list` (de `9d835f99a`), las de HZ10 a HZ12, CT200 y después B98, AD233, CT201 y CT202, una vez cada una. Una segunda pasada de las cuatro para en su comprobación previa (`PARO clave=preimagen`) y deja intactas funciones, permisos y relaciones.

Se contrastó el patrón de carga por vista en documentación pública de [React](https://react.dev/reference/react/lazy), [Vue Router](https://router.vuejs.org/guide/advanced/lazy-loading) y [Next.js](https://nextjs.org/docs/app/guides/lazy-loading). No se incorporó ninguno de esos marcos ni se envió código o datos a ellos. La revisión independiente de arquitectura recomienda ese corte acotado aparte, en vez de introducir controles asíncronos incompletos en esta entrega.

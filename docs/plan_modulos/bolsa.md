# Bolsa de trabajo: inventario y continuación

Estado del 5 de octubre de 2026 sobre `origin/main@dfcbab6ce`. Este plan sirve para terminar Bolsa y dejarla lista para que RRHH la use de verdad. Se ha hecho leyendo el código, la composición de arranque, las pantallas y los documentos. No se ha ejecutado nada ni se ha consultado ninguna base de datos. Que un recorrido aparezca aquí como existente no significa que esté instalado en la principal de cidonia ni probado de nuevo en navegador.

Las respuestas de RRHH del 2 de octubre están en [decisiones_rrhh_2026-10-02.md](../estudio_requisitos/decisiones_rrhh_2026-10-02.md). Las preguntas que siguen abiertas están en [`dudas.md`](../../dudas.md) y aquí se citan por su número. El estado por punto de la petición de RRHH está en [estado_peticion_rrhh_2026-09-30.md](../portal_vec/estado_peticion_rrhh_2026-09-30.md). Los documentos de `docs/portal_vec/*bolsa*` son de julio y se han quedado atrás respecto al código.

## Cómo se monta hoy

Las pantallas internas de RRHH y «Mi Bolsa» se montan en la misma aplicación que sirve la principal. Se activan con dos selectores de despliegue, `VEC_BOLSA_BORRADORES_ENABLED` y `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED` ([config/selectores_despliegue_bolsa_ct.go](../../config/selectores_despliegue_bolsa_ct.go)), además de la conexión a PostgreSQL. El registro de rutas está en `internal/app/bootstrap/bolsa_*.go`, en `contratacion_temporal_desarrollo.go` y, para el proceso externo del aspirante, en `portal_externo_mi_bolsa*.go`. Aunque los ficheros se llamen «desarrollo», son la composición que sirve la aplicación.

## Qué existe y funciona

| Recorrido | Qué hay | Límite actual |
| --- | --- | --- |
| Carga inicial de bolsas desde CONVOCA | `vec-server importar-convoca` y `vec-server constituir-bolsa` (`cmd/vec-server/`, `bootstrap/bolsa_importacion_convoca_*.go`), lectura endurecida de XLS y XLSX en `adapters/xlsconvoca`. | Solo se usa desde la línea de órdenes. RRHH no puede hacerlo desde una pantalla (decisión 16 del 02/10). |
| Listas, posiciones y estados | `/api/vec/bolsa/{bolsas,estadisticas,avisos,panel}`; pantallas `portal-bolsas-api.js` y `portal-bolsas-avisos.js`. Estados disponible, trabajando, no disponible, pendiente de incorporación, renuncia, excluido y disponible desde una fecha. | Las reglas de cinco y nueve meses vienen del Reglamento (art. 9) como política configurable. Falta comprobarlas de punta a punta: cese, vuelta a la bolsa y reinicio. |
| Ficha del candidato para RRHH | Contacto, contratos anteriores, reincorporación del titular, sanciones con recurso, cambios de situación, renuncia y exclusión (`httpinterno/situacion_participacion.go`, `portal-bolsas-sanciones.js`, `portal-bolsas-operaciones.js`). | El catálogo de sanciones es de ejemplo (duda 62). Faltan las causas de baja (duda 18). |
| Llamamiento | Propuesta ordenada, borradores, emisión con vista previa y plantilla de correo, oferta con plazas, historial de ofrecimientos y resolución de aceptación o renuncia (`/propuestas-llamamiento`, `/llamamientos/*`, `/ofertas/*`). | Las reglas del 02/10 están en el catálogo `bolsa_reglas.rrhh-20261002.v4.json`: plazo de dos días, solo se pulsa «Aceptar», si no responde se entiende que no acepta y se adjudica entre quienes aceptan. Falta un recorrido completo con varias aceptaciones y el llamamiento directo posterior. |
| Avisos por correo | Emisión con recibo de intento y adaptador SMTP (`application/emision_correo_avisos.go`, `correo_llamamiento.go`). | Que conste «enviado» no prueba que haya llegado. Falta el SMTP corporativo configurable desde administración, que ya está decidido. |
| Reglas y baremo | Gobierno de reglas de baremo con alta, versiones y recuperación (`httpinterno/gobierno_reglas_baremo_v3.go`); reglas de situación, política de ofertas, política de cese y plazo de respuesta; pantallas `modulos/bolsa/baremo/*`, `rrhh-plazos-*.js`, `rrhh-politica-cese-*.js`. | La baremación avanzada con firma y custodia tiene dominio, puertos y SQL, pero no está conectada a ninguna ruta. |
| Documentos pendientes | `/api/vec/bolsa/solicitudes-documentales/pendientes` para RRHH y `/mi-bolsa/solicitudes-documentales` para el aspirante (Bolsa 000077). | Encaja con «en revisión» y con la renuncia justificada del 02/10. Falta probar el ciclo completo: el aspirante entrega el justificante, RRHH lo valida y el estado cambia. |
| Mi Bolsa (aspirante) | `/api/vec/bolsa/mi-bolsa`, `/historial`, `/solicitudes`, `/respuestas`, `/disposiciones` y `/contacto`, con la pantalla `web/static/area-personal/mi-bolsa-portal.js`. El titular sale del certificado, nunca del navegador. | El proceso externo separado y sus avisos siguen en PR antiguas sin cerrar (#179, #204, #205, #206, #209 y #221). |
| Consulta pública | `/api/publico/bolsa/{bolsas,categorias,convocatorias}` y `web/static/bolsa/{index,listas}.html`. | Qué campos se publican está pendiente del DPD (dudas 17 y 83). |
| Auditoría | Auditoría común de lecturas y consulta para RRHH. La de los contactos está en marcha (#719 y #727). | La exportación con valor de prueba es de Auditoría, no de Bolsa ([auditoria.md](auditoria.md)). |

Migraciones: la más alta es `bolsa_llamamientos/000077`. La 000078 está reservada por #727.

## Qué no existe o está a medias

| Hueco | Qué recorrido de RRHH bloquea | SQL | Depende de | Tamaño | En paralelo |
| --- | --- | --- | --- | --- | --- |
| **B1. Carga de bolsas desde Excel en una pantalla.** RRHH sube el Excel de CONVOCA, ve una vista previa con errores por fila y confirma. Reutiliza el importador y `constituir-bolsa`. | Poner en marcha VEC con las bolsas actuales (decisión 16). Mientras tanto Sistemas puede hacerlo por línea de órdenes. | Sí: consumidor de permiso propio y auditoría de la carga. | Usuarios y permisos (perfil de carga), Auditoría. | M | Sí: ficheros nuevos en `bolsa/adapters/httpinterno`, un `bootstrap/bolsa_carga_*.go` nuevo y una vista nueva. |
| **B2. Recorrido de oferta con varias plazas y llamamiento directo.** Comprobar en un clon, de punta a punta, el circuito del 02/10: correo, dos días, varias aceptaciones, adjudicación por posición, «sin respuesta» sin consecuencia, llamamiento directo y exclusión o renuncia justificada. Corregir solo lo que falle. | Cubrir un puesto desde la bolsa. | No, salvo que aparezca un fallo. | Ninguna. | M | Sí: es una prueba; los arreglos van en PR aparte. |
| **B3. Renuncia justificada y «en revisión».** El aspirante entrega el justificante desde Mi Bolsa y RRHH lo valida y cambia el estado. | Volver a estar disponible tras una renuncia justificada. | Probablemente no: ya existe Bolsa 000077. | Documentos (custodia del justificante). | S–M | Sí, aunque conviene hacerlo después de B2. |
| **B4. Retirar el asistente antiguo de solicitud del área personal.** `web/static/area-personal/cliente-http.js` y `flujo-solicitud.js` llaman a `/api/vec/bolsa/mis-solicitudes/{borrador,autobaremo,pago,firma,registro}`, `mis-llamamientos`, `mis-subsanaciones` y otras rutas que no tienen servidor. La solicitud de verdad va por Selección, el «Convoca integrado» de [selectivos.md](selectivos.md). | Evita que el aspirante vea botones que fallan. | No. | Ninguna. | S | Sí: solo web del área personal. |
| **B5. Correo saliente corporativo configurable desde administración.** Servidor, remitente y prueba de envío, gobernados y auditados. | Avisar de ofertas con el correo de la Diputación. | Probablemente sí (configuración gobernada). | Administración; Informática debe dar los datos del servidor. | M | Sí, pero antes hay que decidir qué se rescata de #179/#204/#206/#209. |
| **B6. Decidir sobre las PR antiguas del proceso externo** (#179, #204, #205, #206, #209 y #221: correos, imagen, avisos, cifrado y portal separado). Rescatar lo que sirva o cerrarlas. | Mi Bolsa y avisos en el portal externo separado. | Algunas traen SQL de autorización (000017, 000018, 000021) que hay que volver a medir. | Usuarios externos. | S para decidir; M–L para rescatar | La decisión es de dirección; lo rescatado va pieza a pieza. |
| **B7. Publicación, sustitución y retirada de convocatorias desde VEC.** Hoy solo hay borrador (`/convocatorias/borradores`). | Abrir una bolsa nueva sin pasar por CONVOCA. | Sí. | Selección (solicitud y listas), firma y publicación. | L | Va con el plan de Selección; no es necesario para empezar con las bolsas importadas. |
| **B8. Baremación con firma y custodia conectada a una ruta.** | Baremar una convocatoria nueva dentro de VEC. | Ya existe (`bolsa_baremacion` 000001–000005). Faltan el consumidor y la ruta. | Firma, Selección. | L | Después de B7. |
| **B9. Lista pública de cada bolsa.** | Publicar la lista como hace CONVOCA. | Probablemente no. | Dudas 17 y 83 (DPD). | S cuando haya respuesta | Sí. |
| **B10. Causas de baja y documentación tras aceptar.** | Dar de baja y pedir documentos al adjudicatario. | No: catálogo configurable. | Duda 18. | S | Sí. |
| **B11. Contacto de aspirantes importados que aún no han entrado en VEC.** | El primer llamamiento tras la carga inicial. | No. | Duda 45. | S | Sí. |

Fuera de este plan: el envío de SMS y Telegram (decisión del 02/10: solo correo por ahora) y el pago de tasas, que está cerrado a propósito.

## Por dónde seguir

1. Terminar las PR en marcha de Bolsa: #719 y #727, auditoría de contactos.
2. Lo que se puede hacer ya y RRHH notará antes: B2 (recorrido), B4 (limpieza del área personal) y B1 (carga desde pantalla). Las tres tocan ficheros distintos.
3. B3 después de B2. B5 después de la decisión B6.
4. B7 y B8 van con Selección cuando RRHH quiera abrir bolsas nuevas en VEC. Para empezar basta con las importadas.
5. B9, B10 y B11 cuando lleguen las respuestas. Son cambios de catálogo o de poco tamaño.

Cada pieza sigue las reglas de siempre. Si toca SQL: reserva previa en `RESERVAS_MIGRACIONES.md`, `revisar-sql-vec`, ensayo en el clon de la principal y revisión SQL independiente. Si toca pantallas: `usabilidad-vec`, `aspecto-vec` e `impeccable` antes de programar y revisión de usabilidad independiente. Textos en catálogos por idioma.

Ficheros grandes que conviene no engordar más: `bootstrap/bolsa_borrador_llamamiento_desarrollo.go` (936 líneas), `web/static/portal-empleado/portal-bolsas-api.js` (1152) y `bolsa/domain/llamamientos.go` (1208). Una pieza nueva va en un fichero nuevo.

## Recuento de llamamientos en el cuadro RRHH — 7 de octubre de 2026

El corte B85, preparado en `1b5b48e2117a45ee8fa22426b463062d2c14310a`, agrupa los llamamientos en curso de las bolsas constituidas. El cuadro y las estadísticas leen situaciones, políticas y recuentos en tres consultas dentro de una transacción `REPEATABLE READ`. La ruta de conjunto deja de consultar los llamamientos bolsa por bolsa. B85 conserva el criterio de la función B17 y devuelve cero para las bolsas sin llamamientos. Requiere B82 instalada y no añade configuración. Si falta B85, la aplicación arranca con el camino legado completo: continúa una consulta `ContarEnCurso` por bolsa y, por tanto, el N+1.

En un clon local PostgreSQL 18.4 con 13 bolsas y 2.390 candidaturas, B85 se instaló una vez. Sus 13 recuentos coincidieron con B17; la suma fue 1. El test del adaptador registró exactamente una consulta de situaciones, una de políticas y una de recuentos. En 100 lecturas del adaptador, el p95 fue de 12,99 ms; esa cifra no incluye HTTP.

El recorrido HTTP del mismo código fuente usó la autenticación mTLS y el perfil técnico vigente de la ruta existente. `/api/vec/bolsa/bolsas` y `/api/vec/bolsa/estadisticas` respondieron 200 en 11 peticiones cada una, con tres consultas SQL por petición. Entre las diez peticiones de medición, el p95 fue de 19,298 ms para bolsas y 18,039 ms para estadísticas. El conjunto conservó 13 bolsas y 2.390 participaciones; la respuesta de bolsas sumó un llamamiento en curso. La prueba no habilitó una acción V3 nueva ni cambió el alcance del permiso. La interfaz sigue indicando «No disponible» para el histórico de llamamientos, que este corte no acredita.

La evidencia HTTP se conserva fuera de Git con SHA256 `526249a66918018e2b4a7bc3638da964153e4244e117c1fba90315d3b2eb75b1`. La instalación y el recorrido fueron locales; no acreditan instalación en la principal ni producción.

### Optimización posterior B86 — 8 de octubre de 2026

B85 evita el N+1, pero su consulta todavía descompone las participaciones de cada llamamiento histórico de una bolsa vigente. B86 prepara una marca privada de completitud de correo por llamamiento y la rellena una vez con el predicado exacto de B17. En adelante, los INSERT de emisiones y contactos actualizan la proyección en la misma transacción; el cuadro suma las marcas por bolsa, sin volver a recorrer los contactos históricos. Una fila privada de coordinación por llamamiento impide que dos contactos simultáneos confirmen una marca incompleta bajo `READ COMMITTED`, `REPEATABLE READ` o `SERIALIZABLE`. La proyección conserva referencias opacas y no guarda datos de contacto ni modifica la historia B13/B17.

B86 requiere B17, B82 y B85 instaladas. B85 ya tiene historia: no debe repetirse su UP ni ejecutarse un DOWN. El candidato `c5ed5154946568b319aef04b432569b53cbdbabd` recibió dos revisiones estáticas favorables. Se instaló una vez en un clon aislado PostgreSQL 18.4, después de restaurar HX, aplicar las 14 SQL de HZ y B85 en orden. No se ha instalado en una base compartida ni añade una decisión de autorización.

En ese clon, el backfill de B86 sobre 39.000 llamamientos y 100.002 contactos quedó incluido en una instalación de 0,87 s. Los 30.001 completos coincidieron con B17. Tras ampliar el historial a 100.012 llamamientos, la prueba volvió a confirmar paridad; la lectura agrupada tuvo p95 de 12,184 ms en 100 consultas SQL, frente a 13,103 ms con 39.000. Son tiempos de SQL local, sin HTTP. Se probaron seis carreras: dos últimos contactos y contacto anterior a la emisión bajo `READ COMMITTED`, `REPEATABLE READ` y `SERIALIZABLE`; en las dos últimas, la escritura concurrente obtuvo `40001` y el reintento completo del ensayo dejó una sola marca. El backend no hace ese reintento automáticamente, por lo que esta prueba no acredita recuperación transparente de una finalización concurrente. También pasaron el lote de 100 contactos, una entrada inválida, un replay sin incremento, la reversión y la lectura tras reiniciar PostgreSQL.

Antes de actualizar estadísticas, una lectura B85 sobre el historial nuevo seguía activa a los 185 s y se canceló; después de `ANALYZE` tardó 0,59 s. Estas cifras miden planes distintos y no se usan como una comparación directa de p95. Una base vacía copiada solo a nivel de esquema devolvió cero bolsas y pasó las pruebas tras restituir los permisos de sus tipos de fila, que la copia de esquema no había conservado. La evidencia completa queda fuera de Git, en el entorno privado de ensayo. Faltan la revisión SQL final del resultado físico, la integración y el despliegue controlado.


## Inicio y filtros al volver — 8 de octubre de 2026

La cifra de disponibles de cada bolsa en Inicio abre su lista paginada con el estado «Disponible». También permite consultar una lista vacía cuando la cifra es cero. Un dato desconocido o una referencia inválida se muestra sin enlace. El nombre de la bolsa sigue abriendo la lista completa; la portada no pide candidaturas hasta que se pulsa el enlace.

Al salir de una lista de Bolsa hacia Inicio se retiran `bolsa_ref`, `estado` y `cursor` de la URL; se conservan los parámetros del portal. Atrás y Adelante recuperan la entrada anterior con su filtro. Las URL antiguas que ya apuntan a Inicio se corrigen sin crear otra entrada de historial.

Los tres totales globales siguen pendientes de una lectura global autorizada y paginada. Hoy suman participaciones en bolsas, por lo que una persona incluida en varias bolsas puede contar varias veces. No se debe sustituir ese conjunto por una lista de una sola bolsa ni consultar cada bolsa desde el navegador para reconstruirlo.


## Ceses sin candidato — revisión de #774, 8 de octubre de 2026

B81 permite continuar el relevo cuando el llamamiento del puente CT eligió
una participación que no pertenece a ninguna bolsa constituida. Conserva el
cese y su registro con actor, fecha y huella; no asigna candidato ni cambia
disponibilidad. Si la bolsa está constituida y falta el vínculo, el cese sigue
pendiente y el relevo se detiene. El camino de regreso a la bolsa requiere
que CT utilice participaciones reales de la bolsa; queda fuera de esta PR.

La rama se actualiza con main y el cursor se toma de la definición instalada
en postHX + HZ + B85 + B86 + CT193 + B87. B81 sigue reservada para #774,
sin colisión con esas migraciones. Antes de integrar, Claude revisa el SQL
exacto y comprueba el ensayo privado y la CI. No se ha instalado B81 en
ninguna base compartida. La lista de instalación contiene solo B81; no se
reaplican las migraciones anteriores. CONFIG NUEVA: ninguna.

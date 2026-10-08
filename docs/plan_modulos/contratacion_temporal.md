# Contratación temporal: inventario y continuación

Estado del 5 de octubre de 2026 sobre `origin/main@dfcbab6ce`. El objetivo es que RRHH pueda tramitar de verdad un expediente de personal temporal, desde la petición del centro hasta el cese, con Bolsa y Personal. Se ha hecho leyendo el código, la composición de arranque, las pantallas y los documentos. No se ha ejecutado nada. Que algo aparezca como existente no significa que esté instalado en la principal ni probado otra vez en navegador.

El circuito que RRHH describió el 2 de octubre (petición, autorización, crédito, oferta, adjudicación, informe, fiscalización, resolución, GINPIX y toma de posesión) está en [decisiones_rrhh_2026-10-02.md](../estudio_requisitos/decisiones_rrhh_2026-10-02.md). Las preguntas abiertas se citan por su número en [`dudas.md`](../../dudas.md). La firma tiene su propio plan en [firmas.md](firmas.md) y aquí no se repite.

## Cómo se monta hoy

Hay una sola composición: la que sirve `vec-server`. Está en `internal/app/bootstrap/contratacion_temporal_*.go` y es la que corre en la principal. Cada fase se activa con un selector `VEC_CT_*` (seguimiento y cese, cancelación, registro de firma, firmas R5, plantillas, reincorporación del titular, incorporación acreditada). Si falta lo que una fase necesita, su ruta no se registra o responde «no disponible». Las rutas cuelgan de `/api/vec/contratacion-temporal`. Las del paso a Personal cuelgan de `/api/interno/`. Las pantallas están en `web/static/portal-empleado/modulos/contratacion-temporal/`.

## Qué existe y funciona

| Paso | Qué hay | Límite actual |
| --- | --- | --- |
| Petición del centro y ratificación | `peticiones-centro/*`, con bandeja, operaciones, entrega a RRHH, cancelación y confirmación de incorporación. | Quién puede tener el perfil de entrega a RRHH está pendiente (duda 141). |
| Alta del expediente | `/solicitudes` y `/catalogos-alta`. El número de expediente lo guarda VEC y lo genera MOAD (`domain/politica_numero_expediente.go`). | — |
| Análisis RRHH | `/configuracion-analisis` y `/analisis/{registros,rectificaciones}`, con urgencia y motivo, jornada y entradas de retención de crédito. | La urgencia se declara al analizar como valor provisional (duda 63). La jornada no puede superar la completa (duda 38). La retención se pide como dato y documento porque no hay conexión con SICAL (duda 125). |
| Cuadro, lista y detalle | `/cuadro/consultas`, `/expedientes/consultas`, `/estadisticas`, `/expedientes/{comunicaciones,borradores}`. | Necesitan el PostgreSQL de consultas; sin él responden «no disponible». |
| Vía de cobertura | `/cobertura/{propuesta,decisiones,rectificaciones,resultados}`. | La oferta al SAE (Servicio Andaluz de Empleo) es solo un aviso, sin envío (dudas 72, 73 y 126). |
| Informe, fiscalización y subsanación | `/informes-juridicos/preparaciones`, `/fiscalizaciones/resultados` y `/subsanacion-reparos`, con informe nuevo tras subsanar (CT123). | Si el plazo se reanuda o empieza de nuevo tras subsanar está pendiente (duda 95), igual que el justificante de cada subsanación (duda 96). |
| Llamamiento a Bolsa | `/llamamientos/{seleccion,comunicaciones,resoluciones,respuestas/*,plazos/eventos,siguientes}`: oferta, respuesta, aceptación, renuncia y siguiente candidato. | Solo funciona si Bolsa tiene PostgreSQL. Las reglas son las de Bolsa ([bolsa.md](bolsa.md)). |
| Propuesta y resolución | `/formalizacion/{propuestas,documentacion}` y `/resoluciones-formalizacion`. | — |
| Documentos | Borradores PDF y DOCX de diez tipos: informe, resolución, diligencia, toma de posesión, notificación, comunicación al centro, contrato, nombramiento, cese y modificación. También `/plantillas`. | Las plantillas son de ejemplo y no tienen validez. Faltan las de RRHH en Word (duda 124). La descarga auditada está en marcha (#734). |
| Firma | GrxFirma por `afirma://`, circuito de firma, registro y verificación, y recuperación con el plan de firmantes. | Lo que falta está en [firmas.md](firmas.md). Está en marcha (#736–#742). El portafirmas de Diputación tiene un conector apagado (`adapters/portafirmasapagado`) y depende de las dudas 74 y 128. |
| GINPIX | Ficha resumen para grabar a mano, seguimiento y `/confirmaciones-ginpix`, que deja constancia de cuándo se grabó. | No hay importación automática (duda 127). Así lo decidió RRHH para empezar. |
| Cese, cierre y cambios | `/ceses`, `/cierres-expediente`, `/modificaciones-nombramiento`, `/no-incorporaciones`, `/seguimiento-cese`, cerrar sin cese, reapertura excepcional, reincorporación del titular y cancelación. | El paso del cese a Personal está en marcha (#715). |
| Paso a Personal | Plan y confirmación de la incorporación (`/api/interno/contratacion-temporal/incorporacion-personal-b2/*`). | Qué datos se dan por buenos y cuándo cuenta la toma de posesión: duda 140. |
| Auditoría | Auditoría común de las operaciones y de las denegaciones de frontera. Las lecturas fallidas de recibos y comunicaciones están en marcha (#713). | — |

Migraciones: la más alta en main es CT176. CT177 y CT178 están reservadas por #734 y #742.

## Qué no existe o está a medias

| Hueco | Qué recorrido de RRHH bloquea | SQL | Depende de | Tamaño | En paralelo |
| --- | --- | --- | --- | --- | --- |
| **C1. Recorrido completo en el clon de la principal.** Petición, ratificación, análisis, llamamiento, aceptación, informe, fiscalización con un reparo, resolución, GINPIX, Personal y cese con vuelta a Bolsa. Después, reinicio de la aplicación y de PostgreSQL. Cada fallo se anota como una PR aparte. | Saber qué falla antes de que RRHH lo use. | No. | Firma como sustituto marcado mientras no esté el recorrido de dos firmas. | M | Sí: es una prueba. |
| **C2. Selectores de la principal.** Lista de los `VEC_CT_*` y `VEC_BOLSA_*` que deben estar encendidos para el uso real, con lo que necesita cada uno y una comprobación al arrancar que avise si falta una dependencia. | Que ninguna fase aparezca como «no disponible» sin motivo. | No. | Ninguna. | S | Sí: fichero nuevo junto a `config/selectores_despliegue_bolsa_ct.go` y su prueba. |
| **C3. Cierre administrativo sin componer.** `AutoridadCierreAdministrativo` y `EjecutorCierreAdministrativo` siguen como «no compuesta» (`contratacion_temporal_desarrollo.go`, hacia la línea 857). El cierre real va por el seguimiento del cese. Hay que retirar la ruta muerta o componerla. | Evita un botón o una ruta que siempre responde «no disponible». | No. | Ninguna. | S | Después de #713, #715 y #734, que tocan el mismo fichero. |
| **C4. Retención de crédito como condición.** El 02/10 RRHH dijo que sin crédito no se tramita nada. Hay que comprobar que el expediente no avanza a la oferta sin número y documento de la retención, o sin la constancia del estado de las partidas, y mostrar por qué. | Respetar el orden de RRHH: crédito antes de ofrecer. | Puede necesitarlo, si la guarda tiene que estar en la transición. | Duda 125 (solo para conectar con SICAL; la guarda no espera). | S–M | Sí: análisis y cobertura. |
| **C5. Plantillas oficiales.** Cargar las plantillas Word de RRHH con sus campos variables por el catálogo de `/plantillas`, sin tocar código. | Generar documentos válidos. | No. | Duda 124 (RRHH envía las plantillas). | S por plantilla | Sí, cuando lleguen. |
| **C6. Plazo de fiscalización tras subsanar.** Configurable: reanudar o empezar de nuevo. | El plazo que se ve en el cuadro. | Posiblemente, si va en la instantánea de reglas. | Duda 95. Conviene antes decidir sobre las PR #233, #237 y #240 (plazos), que siguen en borrador. | S | Sí. |
| **C7. Oferta al SAE.** Datos, canal y selección de los candidatos que remite el SAE. | Cubrir cuando la bolsa se agota. | Sí. | Dudas 72, 73 y 126 (reunión con RRHH). | L | Después de la reunión. |
| **C8. Delegaciones de firma y suplencias.** Recibirlas y aplicarlas como perfiles con referencia del acto. | Saber quién firma cada día. | Sí (Autorización). | Dudas 122 y 128. Dueño: Administración. | M | Sí, en Administración. |
| **C9. Portafirmas de Diputación.** Sustituir el conector apagado por el real. | Firma por Firmadoc en lugar de GrxFirma, si RRHH lo pide. | Puede. | Duda 74 y el contrato técnico con Informática. | L | Después de las firmas en curso. |
| **C10. Paso a Personal sin volver a teclear.** Rellenar el alta en Personal con los datos del expediente y fijar cuándo cuenta la toma de posesión. | Alta de la persona contratada en Personal. | Puede. | Duda 140, y #715 fusionada. Dueño: Personal ([personal.md](personal.md)). | M | Después de #715. |
| **C11. Revisión de usabilidad del recorrido de RRHH.** Una persona de RRHH sin formación previa recorre bandeja, detalle, análisis y llamamiento. Se apuntan los cambios por pantalla. | Que RRHH lo use sin manual. | No. | Ninguna. | S para revisar; los arreglos por separado | Sí: es una revisión. |

## Por dónde seguir

1. Dejar que terminen las PR en marcha: #713, #715, #734 y la cadena de firmas #736–#742.
2. Se puede empezar ya y en paralelo: C1 (recorrido), C2 (selectores), C4 (crédito) y C11 (usabilidad). No tocan los ficheros de esas PR.
3. C3 cuando estén fusionadas #713, #715 y #734.
4. C5, C6, C7, C9 y C10 cuando llegue la respuesta de RRHH que les corresponde. C5 y C6 son cambios de catálogo.
5. La firma sigue el orden de [firmas.md](firmas.md). El recorrido de dos firmas (tarea 6) es lo que falta para decir que un documento sale firmado.

Ficheros que no conviene engordar más: `bootstrap/contratacion_temporal_desarrollo.go` (1510 líneas), `contratacion_temporal_postgresql_desarrollo.go` (1047) y `contratacion_temporal_seguimiento_cese_desarrollo.go` (964). Lo nuevo va en ficheros nuevos.

## Encargos para empezar ya (Bolsa y Contratación temporal)

Ninguno toca las PR en marcha (#713, #715, #719, #727, #733, #734, #736–#742). Cada uno va en su rama y su PR, con las skills que le correspondan.

1. **C1 · Recorrido completo de Contratación temporal y Bolsa en el clon de la principal.** Skill `probar-recorridos-vec` y Playwright con el Chrome del sistema. Entrega un acta con cada paso, la respuesta HTTP y una captura, antes y después del reinicio. Cada fallo queda como una tarea aparte; no se arregla dentro de este encargo. Tamaño M.
2. **B2 · Oferta con varias plazas y llamamiento directo según las reglas del 02/10.** Recorrido en el clon: varias aceptaciones, adjudicación por posición, «sin respuesta» sin consecuencia, llamamiento directo con exclusión o renuncia justificada. Se corrigen en la misma rama solo los fallos que estén en ficheros nuevos o en la aplicación de Bolsa. Tamaño M.
3. **B4 · Retirar del área personal el asistente antiguo de solicitud,** que llama a rutas sin servidor (`mis-solicitudes/*`, `mis-llamamientos`, `mis-subsanaciones` y las demás). Solo `web/static/area-personal/`, con `usabilidad-vec` y una revisión de usabilidad independiente. Tamaño S.
4. **C2 · Selectores de la principal.** Documento y comprobación al arrancar de los `VEC_CT_*` y `VEC_BOLSA_*` necesarios para el uso real, con un aviso claro si falta una dependencia. Fichero nuevo en `config/` y su prueba. Tamaño S.
5. **C4 · Sin crédito no se ofrece.** Guarda en el expediente que impide pasar a la oferta sin la retención o la constancia de las partidas, con el motivo en pantalla. Si necesita SQL: reserva, `revisar-sql-vec`, ensayo en el clon y revisión SQL independiente. Tamaño S–M.
6. **B1 · Carga de bolsas desde el Excel de CONVOCA en una pantalla,** con vista previa por fila y confirmación. Reutiliza el importador. Permiso propio y auditoría común. Lleva SQL y pantalla, así que necesita todas las revisiones. Tamaño M.
7. **C11 · Revisión de usabilidad del recorrido de RRHH** (bandeja, detalle, análisis, llamamiento, ficha del candidato de Bolsa) con `revisor-usabilidad-vec`. Entrega la lista de cambios por pantalla, ordenada por gravedad. Tamaño S.
8. **B6 · Decidir sobre las PR antiguas** del proceso externo (#179, #204, #205, #206, #209, #221) y de plazos (#233, #237, #240): qué se rescata, sobre qué main y qué se cierra. Lo decide dirección. Desbloquea B5 (correo corporativo) y C6 (plazo tras subsanar). Tamaño S.
9. **B3 · Renuncia justificada y «en revisión» de punta a punta:** el aspirante entrega el justificante desde Mi Bolsa, RRHH lo valida y el estado cambia, con auditoría. Va después de B2. Tamaño S–M.


## Correcciones del recorrido del 07/10: auditoría del expediente

La consulta web distingue ahora una ruta de auditoría no disponible (404) de un acceso denegado (401/403). Ante un fallo temporal permite reintentar y mantiene los filtros cerrados hasta recuperar las opciones del servidor. El número del expediente se recibe como dato de presentación desde la ficha; no se deduce de su referencia opaca. El montaje que lo transmite se entrega con la corrección de la ficha de CT.

La activación sigue en manos de dirección: el despliegue requiere `VEC_RRHH_AUDITORIA_ENABLED` y sus dependencias nominales. Esta corrección de interfaz no activa el servicio ni cambia permisos, consultas o registros de auditoría. El contrato Go y SQL de Documentos D14 del equipo V está integrado en main (#845). La ficha de #850 transmite la referencia `expediente:ct:<64 hex>` y rechaza la referencia vacía de desarrollo. La instalación de D14 y la consulta nominal en el entorno de destino siguen requiriendo su comprobación; esta entrega no las acredita.

Siguiente corte: recuperación de carga de CT, ficha y navegación de Bolsa; después, las lecturas de Bolsa con una decisión V3, auditoría y consulta en la misma transacción, paginación SQL y listas filtradas para las cifras del resumen. CT187 está integrado desde la PR #840; su medición y sus límites constan allí. No se da por terminado el recorrido completo ni la firma.


## Inicio sin acceso SAE pendiente — 8 de octubre de 2026

Se retiran la tarjeta y la entrada de ofertas al SAE de Inicio: abrían una pantalla sin gestión disponible. Inicio conserva los indicadores de peticiones y Bolsa, en la rejilla compacta común. La elección de cobertura en el expediente y los datos de las ofertas de Bolsa conservan su recorrido. La gestión del SAE se ofrecerá cuando tenga un consumidor real.

## Alta según la circular del 19/02/2026 — preparación del 08/10

El catálogo v2 distingue vacante, sustitución, acumulación de tareas y programa. Recoge el número de personas y exige los códigos de plaza y puesto para vacante. La jornada se muestra en horas y minutos y se conserva en minutos enteros. La modalidad jurídica se decide durante el análisis. El catálogo de ejemplo es configurable y cita la circular oficial comprobada.

Los textos de necesidades se cargan en el idioma activo al abrir Alta, con reintento. La interfaz valida las referencias alternativas de financiación y los periodos de cada causa. El coordinador entrega el getter v2 sin pedirlo en otras vistas. El recibo HTTP conserva sus cinco campos. La consulta RPT por código exacto, cedida por Personal, filtra antes del recuento y la paginación; la recuperación de la ficha distingue incidencia, ausencia y denegación.

CT193 y Go conservan la necesidad y el catálogo en el efecto sellado. Dos revisiones estáticas y el ensayo PostgreSQL 18 desechable verificaron catálogo v2 y número de personas igual a2, incluidos reinicio, colisión y concurrencia; las instantáneas anteriores conservaron sus reglas. El formulario pasó Chrome con APIs de prueba. Falta el recorrido HTTP nominal completo. La puerta de calidad rechazó los bloques de reconstrucción dinámica. El primer cambio a definiciones explícitas (`83a14d71…e798`) omitía CT164; dirección lo rechazó y su ensayo no reproducía la principal. La corrección `eefd7bc6…6bb3` parte de postHX más los 14 cambios HZ: conserva CT164 y reconstruye las 71 altas con los mismos bytes. El ensayo PostgreSQL 18 terminó con código 0 y mantuvo las cinco funciones y el trigger de CT164. La prueba del circuito inicial usa una transacción revertida; no acredita un alta V3 nominal. La prueba de programa temporal que fallaba en CI ya incluye el número de personas obligatorio y pasa. Las revisiones finales y la CI del correctivo siguen pendientes. CONFIG NUEVA: `VEC_CT_NECESIDADES_ALTA_SOURCE_PATH=/app/data/catalogos/contratacion-temporal/necesidades_v1.ejemplo.json`. Docker copia la fuente única a esa ruta. En el kit nativo se copia la misma fuente a `data/catalogos/contratacion-temporal/necesidades_v1.ejemplo.json`; dirección monta el directorio data ya usado por la aplicación y activa el selector después de instalar CT193. La imagen no activa el selector por defecto. No se instaló SQL en una base compartida.


## Disponibilidad de documentos en la ficha — 8 de octubre de 2026

El detalle leído y auditado informa de si la consulta de borradores está montada. La ficha conserva el expediente y evita pedir los borradores cuando esa consulta falta. Los metadatos corresponden al mismo expediente y versión; se retiran al empezar otra lectura. Si la consulta está montada y todavía falta comprobar su permiso nominal, se mantiene cerrada y se ofrece reintentar la comprobación. No se deduce el permiso del perfil visible.

Este corte reutiliza la fuente de interfaz de S (`6455e0ead`), añade el consumidor del detalle y no lleva SQL ni configuración nueva. Sigue pendiente conectar la proyección nominal por recurso de V para los documentos montados; el estado de montaje por sí solo no concede acceso.

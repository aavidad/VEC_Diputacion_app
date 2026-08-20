# CT-CUM-08 — Expediente candidato de evaluación y declaración de accesibilidad

Fecha de corte: 20 de agosto de 2026.

Base documental: `0fdc31d19865ddb2183eeda958aff5b2d3c826dd`.

Estado: **NO_EVALUADO; no aprobado; ninguna declaración publicada**.

## Autoridad, capability e invariante

La
[matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-08` como la evaluación de accesibilidad y su declaración. Esta
puerta bloquea la publicación productiva.

La capability de este expediente se limita a inventariar familias de
superficies, fijar criterios y evidencia de evaluación y preparar la estructura
de una declaración futura. No constituye:

- una auditoría de una interfaz desplegada;
- una afirmación de conformidad total o parcial;
- una declaración publicada ni un mecanismo de reclamación activo;
- una evaluación visual o con tecnologías de asistencia completada;
- la aprobación de excepciones, cargas desproporcionadas o contenidos
  excluidos;
- una activación de web, documentos, datos, servicios o producción.

La invariante es cerrada: la ausencia de evaluación o evidencia conserva cada
superficie en `NO_EVALUADO`. Un test automático, captura, componente aislado o
texto alternativo puntual nunca acredita accesibilidad integral.

## Fuentes y límites

El expediente usa únicamente:

- el
  [inventario candidato CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- el [dossier RAT/EIPD candidato CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- el
  [dossier ENS/SoA candidato CT-CUM-04](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- el
  [plan de riesgos candidato CT-CUM-05](ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md);
- la
  [política ENI candidata CT-CUM-06](ct_cum_06_politica_eni_documento_expediente_firma_conservacion_2026-08-20.md);
- la
  [matriz de competencia candidata CT-CUM-07](ct_cum_07_matriz_competencia_actuaciones_humanas_automatizadas_2026-08-20.md);
- el [expediente remitido por RRHH](expediente_contratacion_temporal_rrhh.md) y
  los [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md).

No se inspecciona ni describe una URL, entorno, cuenta, persona, expediente,
documento o dato real. Las capturas históricas del manual no se toman como
evidencia vigente. CT-CUM-02 a CT-CUM-07 siguen siendo candidatos locales y
este documento no altera su estado.

## Vocabulario de estado

| Estado | Significado cerrado |
| --- | --- |
| `NO_EVALUADO` | No existe evaluación suficiente del alcance. |
| `CRITERIO_PENDIENTE` | Falta concretar perfil, método o criterio de aceptación. |
| `EVIDENCIA_NO_APORTADA` | No hay prueba aprobada y ligada a una versión. |
| `HALLAZGO_ABIERTO` | Defecto o incertidumbre pendiente de remediación y nueva prueba. |
| `DECLARACION_NO_PUBLICADA` | El borrador carece de aprobación, fecha, URL y mecanismo activo. |
| `EXCEPCION_NO_APROBADA` | Una exclusión o carga desproporcionada no ha sido decidida. |
| `BLOQUEANTE` | Impide publicación productiva. |

No se usan `CONFORME`, `PARCIALMENTE_CONFORME` o `NO_CONFORME` como resultado
real porque no se ha ejecutado la evaluación requerida.

# Parte A — Alcance candidato

## Familias de superficies y contenidos

| ID | Familia | Muestra futura mínima | Estado |
| --- | --- | --- | --- |
| AX-01 | Web interna de RRHH | Navegación, bandejas, detalle, filtros, tablas, diálogos y acciones por fase. | `NO_EVALUADO` |
| AX-02 | Web pública | Información, búsqueda, ayuda, cotejo y contenidos públicos que lleguen a autorizarse. | `NO_EVALUADO` |
| AX-03 | Formularios | Alta, análisis, subsanación, revisión, carga de documentos y confirmaciones. | `NO_EVALUADO` |
| AX-04 | Autenticación y sesión | Identificación, errores, caducidad, garantía adicional y recuperación autorizadas. | `NO_EVALUADO` |
| AX-05 | Flujos complejos | Pasos, fases, comparaciones, validaciones, revisión humana y vuelta atrás. | `NO_EVALUADO` |
| AX-06 | Tablas y visualizaciones | Orden, paginación, filtros, estados, leyendas y alternativas textuales. | `NO_EVALUADO` |
| AX-07 | Firma, registro y notificación | Encargos, revisión de documento, estado y recibos, todos todavía sin efecto real. | `NO_EVALUADO` |
| AX-08 | Documentos descargables | PDF y formatos ofimáticos, índices, copias y representaciones accesibles. | `NO_EVALUADO` |
| AX-09 | Ayuda y errores | Instrucciones, validación, incidencias, soporte y mecanismos de comunicación. | `NO_EVALUADO` |
| AX-10 | Audio, vídeo o multimedia | Reproductores, subtítulos, audiodescripción, transcripción y controles si existen. | `NO_EVALUADO` |
| AX-11 | Correos o comunicaciones | Contenido, estructura, enlaces, adjuntos y alternativas cuando se autoricen. | `NO_EVALUADO` |
| AX-12 | Clientes alternativos | API, escritorio, CLI o MCP: mismos permisos y alternativas, sin presentar salida inaccesible como equivalente. | `NO_EVALUADO` |

Una familia inexistente en la versión evaluada se registra como no presente,
con evidencia de alcance; no se elimina silenciosamente del inventario.

## Versionado del alcance

Cada evaluación aprobable deberá fijar:

- versión de la aplicación y commit o artefacto evaluado;
- entornos y configuraciones autorizados sin incluir secretos;
- rutas, componentes, documentos y estados muestreados;
- idiomas, temas, tamaños de pantalla y modos de entrada;
- navegadores y tecnologías de asistencia aprobados;
- contenidos externos, heredados o de terceros y su autoridad;
- fecha, equipo evaluador, método y limitaciones;
- defectos conocidos, remediaciones y siguiente revisión.

# Parte B — Criterios de evaluación

## Perceptible

| Área | Evidencia futura requerida | Estado |
| --- | --- | --- |
| Alternativas textuales | Imágenes, iconos, gráficos y controles con propósito equivalente. | `EVIDENCIA_NO_APORTADA` |
| Estructura | Encabezados, regiones, listas, tablas y relaciones programáticas. | `EVIDENCIA_NO_APORTADA` |
| Color y contraste | Texto, componentes, foco, gráficos y estados sin depender solo del color. | `EVIDENCIA_NO_APORTADA` |
| Zoom y reflujo | Contenido y acciones utilizables con ampliación y anchuras reducidas. | `EVIDENCIA_NO_APORTADA` |
| Multimedia | Subtítulos, transcripción, audiodescripción y controles accesibles. | `NO_EVALUADO` |
| Documentos | Etiquetado, orden, idioma, tablas, formularios y alternativas en formatos descargables. | `EVIDENCIA_NO_APORTADA` |

## Operable

| Área | Evidencia futura requerida | Estado |
| --- | --- | --- |
| Teclado | Recorrido completo sin ratón, sin trampas y con orden coherente. | `EVIDENCIA_NO_APORTADA` |
| Foco | Indicador visible, movimiento predecible, retorno correcto y contenido no oculto. | `EVIDENCIA_NO_APORTADA` |
| Tiempo | Aviso, ampliación o alternativa cuando proceda; errores temporales recuperables. | `CRITERIO_PENDIENTE` |
| Navegación | Saltos, títulos, encabezados, propósito de enlaces y múltiples vías. | `EVIDENCIA_NO_APORTADA` |
| Gestos y puntero | Alternativas simples, cancelación y tamaño/espaciado evaluados. | `EVIDENCIA_NO_APORTADA` |
| Movimiento y parpadeo | Sin contenido peligroso; posibilidad de detener animaciones cuando proceda. | `NO_EVALUADO` |

## Comprensible

| Área | Evidencia futura requerida | Estado |
| --- | --- | --- |
| Idioma | Idioma principal y cambios identificados programáticamente. | `EVIDENCIA_NO_APORTADA` |
| Consistencia | Navegación, nombres, ayuda y comportamiento previsibles. | `EVIDENCIA_NO_APORTADA` |
| Etiquetas e instrucciones | Requisitos, formato, obligatoriedad y ejemplos sin depender de posición/color. | `EVIDENCIA_NO_APORTADA` |
| Errores | Identificación, asociación al campo, resumen, sugerencia y conservación segura de entrada. | `EVIDENCIA_NO_APORTADA` |
| Prevención | Revisión, corrección y confirmación antes de operaciones relevantes. | `EVIDENCIA_NO_APORTADA` |
| Ayuda | Canales consistentes, accesibles y con alcance claro. | `CRITERIO_PENDIENTE` |

## Robusto

| Área | Evidencia futura requerida | Estado |
| --- | --- | --- |
| Nombre, función y valor | Controles nativos o semántica equivalente, estados y cambios anunciados. | `EVIDENCIA_NO_APORTADA` |
| Mensajes de estado | Éxito, error, espera y progreso disponibles sin mover foco indebidamente. | `EVIDENCIA_NO_APORTADA` |
| Compatibilidad | Navegadores y tecnologías de asistencia definidos en alcance. | `NO_EVALUADO` |
| Contenido dinámico | Diálogos, menús, tablas, autocompletado y notificaciones con patrón verificable. | `EVIDENCIA_NO_APORTADA` |
| Documentos y descargas | Tipo, tamaño, idioma y alternativa accesible informados antes de descargar. | `EVIDENCIA_NO_APORTADA` |

# Parte C — Método y evidencias

## Capas de evaluación

| Capa | Propósito | Limitación obligatoria |
| --- | --- | --- |
| Revisión de alcance | Confirmar superficies, estados, idiomas y contenidos incluidos. | No determina conformidad. |
| Análisis automático | Detectar un subconjunto de errores repetibles. | Un resultado verde no cubre criterios manuales. |
| Inspección de código/semántica | Revisar estructura, nombres, relaciones y estados. | No sustituye uso real. |
| Teclado y foco | Recorrer tareas completas y estados de error/espera. | Requiere muestra y versión cerradas. |
| Tecnologías de asistencia | Evaluar lectura, navegación, formularios, tablas y cambios dinámicos. | Requiere combinaciones aprobadas y evaluadores competentes. |
| Evaluación visual | Contraste, reflujo, zoom, foco, espaciado y temas. | Capturas aisladas no prueban interacción. |
| Documentos | Evaluar estructura, lectura, formularios, firma visible y alternativas. | Cada tipo/plantilla/versión requiere muestra. |
| Pruebas con personas | Detectar barreras de uso y comprensión. | No reemplaza la evaluación técnica ni usa datos reales. |

## Ficha de evidencia

Cada resultado futuro deberá conservar:

1. criterio y perfil aplicable;
2. superficie, ruta, componente, contenido y estado;
3. versión exacta del artefacto;
4. método, herramienta y configuración;
5. pasos reproducibles y resultado observado;
6. impacto, frecuencia en la muestra y personas afectadas;
7. evidencia minimizada sin datos ni credenciales;
8. responsable de remediación y estado real;
9. repetición sobre el SHA o artefacto corregido;
10. revisión independiente y fecha de vigencia.

Los hallazgos no se cierran por cambio de texto, imposibilidad de reproducir o
ausencia de quejas. Deben verificarse contra el criterio y alcance originales.

## Matriz de muestreo pendiente

| Muestra | Estados obligatorios | Modos obligatorios | Estado |
| --- | --- | --- | --- |
| Navegación principal | Inicial, foco, sección activa y error de carga. | Teclado, lector, zoom/reflujo y contraste. | `NO_EVALUADO` |
| Formulario | Vacío, válido, errores múltiples, espera, cancelación y recibo. | Teclado, lector, voz cuando se apruebe y ampliación. | `NO_EVALUADO` |
| Tabla/listado | Vacío, una fila, muchas filas, filtros, orden y paginación. | Teclado, lector, reflujo y alternativa lineal. | `NO_EVALUADO` |
| Flujo por pasos | Inicio, avance, vuelta, conflicto, caducidad y recuperación. | Teclado, foco, anuncios y revisión antes de efecto. | `NO_EVALUADO` |
| Diálogo | Apertura, contenido, acción, error y cierre. | Foco contenido, escape y retorno al origen. | `NO_EVALUADO` |
| Documento | Original, representación accesible, firma visible y descarga. | Lector, teclado, zoom y validación estructural. | `NO_EVALUADO` |
| Ayuda/reclamación | Entrada, envío, error, recibo y seguimiento. | Teclado, lector, lenguaje claro y canal alternativo. | `NO_EVALUADO` |

# Parte D — Gestión de hallazgos

## Registro candidato

| Campo | Requisito |
| --- | --- |
| Identificador | Opaco, estable y no derivado de persona o expediente. |
| Criterio | Referencia y versión del requisito evaluado. |
| Alcance | Superficie, componente, estado, idioma y artefacto exactos. |
| Descripción | Barrera observable sin datos reales. |
| Impacto | Personas y tareas afectadas, pendiente de validación competente. |
| Evidencia | Pasos, resultado, captura minimizada o registro permitido. |
| Remediación | Cambio propuesto, propietario y dependencias, sin declararlo implantado. |
| Verificación | Nueva prueba independiente sobre versión corregida. |
| Estado | Abierto, en tratamiento o verificado; nunca cerrado por presunción. |

La priorización no permite publicar una barrera bloqueante por conveniencia.
Excepciones y carga desproporcionada requieren expediente, alternativas,
vigencia y decisión formal; todas están `EXCEPCION_NO_APROBADA` en este corte.

## Revisión periódica

La futura política deberá exigir revisión tras cambios sustanciales de
interfaz, componentes, navegación, identidad, firma, documentos, temas,
idiomas, contenido o tecnologías de asistencia, además de la periodicidad que
apruebe la autoridad. Este dossier no fija fechas.

# Parte E — Borrador de declaración

## Campos pendientes

| Campo de la declaración | Contenido en este corte |
| --- | --- |
| Responsable y ámbito | `PENDIENTE_VALIDACION`; no se designan órganos o sitios reales. |
| Sitio/aplicación incluida | `NO_EVALUADO`; sin URL ni entorno. |
| Estado de conformidad | No determinado. |
| Contenido no accesible | No inventariado; requiere evaluación y evidencia. |
| Motivos | No determinados; ninguna excepción aprobada. |
| Preparación y método | `PENDIENTE_VALIDACION`; sin fecha o evaluador oficial. |
| Observaciones y contacto | Canal corporativo pendiente; no se incluye contacto real. |
| Procedimiento de aplicación | `PENDIENTE_VALIDACION`. |
| Revisión | Sin calendario aprobado. |

Estado de la declaración: `DECLARACION_NO_PUBLICADA`.

## Comunicación y reclamación

El mecanismo futuro deberá ser accesible, permitir describir barreras y pedir
información accesible, emitir recibo, informar de plazos/seguimiento y enlazar
el procedimiento competente. No se presume correo, teléfono, formulario,
órgano o plazo. CT-CUM-07 deberá acreditar competencia para responder y
resolver; ningún canal está activo por este dossier.

# Parte F — Gobierno y cierre

## Decisiones reservadas

| Decisión | Autoridad requerida | Estado |
| --- | --- | --- |
| Perfil y alcance de evaluación | Responsables competentes de accesibilidad y servicio | `PENDIENTE_VALIDACION` |
| Muestra y combinaciones técnicas | Equipo evaluador competente | `CRITERIO_PENDIENTE` |
| Resultado de conformidad | Autoridad tras evaluación completa | `NO_EVALUADO` |
| Excepciones/carga desproporcionada | Autoridad jurídica y funcional competente | `EXCEPCION_NO_APROBADA` |
| Declaración y mecanismo | Autoridad responsable y canal aprobado | `DECLARACION_NO_PUBLICADA` |
| Publicación productiva | Autoridad formal tras cerrar dependencias | `BLOQUEANTE` |

## Bloqueos conservados

- CT-CUM-08 permanece abierto y no aprobado.
- CT-CUM-09 y CT-CUM-10 conservan sus puertas propias.
- CT-CUM-02 a CT-CUM-07 siguen como candidatos locales hasta integración y
  aprobación competentes.
- O4 y los demás carriles técnicos conservan sus NO-GO y dependencias.
- Datos reales, efectos, automatización, declaraciones, web productiva,
  documentos reales, publicación y producción siguen prohibidos.
- El tablero, métricas y documentos transversales no cambian.

## Criterio técnico de revisión

La revisión independiente solo puede comprobar que el candidato:

1. inventaría todas las familias exigidas sin afirmar que existen o cumplen;
2. cubre perceptible, operable, comprensible y robusto con evidencia pendiente;
3. separa análisis automático, inspección manual, tecnología de asistencia,
   evaluación visual, documentos y pruebas con personas;
4. mantiene hallazgos, excepciones y declaración sin aprobar;
5. conserva CT-CUM-08 y publicación productiva como bloqueos;
6. no contiene URL, contacto, persona, cuenta, documento, dato, proveedor,
   credencial o infraestructura reales;
7. resuelve enlaces locales, cita una base Git existente y supera
   `git diff --check`.

Un `GO` documental no cierra CT-CUM-08, no declara conformidad y no autoriza
publicación.

# CT-CUM-02 — Inventario de tratamientos de contratación temporal

Fecha de corte: 20 de agosto de 2026.

Base técnica inventariada: `781bb5891ba3304bfed9d104e71d48846a5b6679`.

Estado: **candidato documental; pendiente de revisión y validación de los
responsables competentes**.

## Autoridad y efecto de este documento

La
[matriz normativa](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-02` como el inventario verificable de tratamientos, campos,
fuentes, finalidades, bases y destinatarios. La tarea está lista y bloquea
cualquier API con datos reales.

Este inventario describe categorías y referencias técnicas, nunca valores de
personas. No es el Registro de Actividades de Tratamiento, una EIPD, una tabla
de valoración documental, una categorización ENS ni una autorización para
usar datos reales. El
[RAT histórico de Bolsas](../cumplimiento/rat_registro_actividades_tratamiento.md)
es una entrada no aprobada y no sustituye las validaciones de contratación
temporal.

Las propuestas de base jurídica, destinatario o conservación de las tablas
siguientes significan siempre **pendiente de validación** por el responsable
del tratamiento, DPD, Jurídico, Archivo, Seguridad, Sistemas y RRHH según sus
competencias. Una ausencia o duda mantiene el tratamiento cerrado.

## Alcance y estados de activación

| Estado | Significado |
| --- | --- |
| `ARBOL_ACTUAL` | Contrato o modelo presente en la base técnica inventariada; no implica composición ni producción. |
| `CANDIDATO_LOCAL` | Material revisado en otra rama, todavía no integrado, publicado ni activo. |
| `FUTURO` | Capacidad prevista por el expediente o tablero, sin tratamiento ejecutable en este corte. |
| `PENDIENTE_VALIDACION` | Decisión organizativa, jurídica o documental no adoptada por este documento. |

Los candidatos locales `a312375` (`O7-01`, contrato Personal/RPT) y
`2b15fe8` (`O7-03`, modelo GINPIX) se registran solo para que una integración
posterior obligue a reconciliar el inventario. No forman parte de
`ARBOL_ACTUAL` y no habilitan altas, incorporaciones ni envíos.

## Invariantes comunes

1. Bolsa conserva convocatorias, integrantes, posiciones, disponibilidad,
   reglas y llamamientos; Personal conserva relación jurídica, ocupación,
   puesto, incorporación y cese.
2. Contratación temporal conserva expediente, fases, tareas, decisiones y
   coordinación. Los intercambios usan referencias opacas, comandos, eventos
   y recibos; no leen ni escriben tablas ajenas.
3. Identidad, perfil, organización, capacidad, finalidad y correlación solo
   proceden de fronteras confiables. Un identificador recibido como dato no
   concede autoridad.
4. Documentos, declaraciones, contactos y expedientes de autoridades externas
   cruzan únicamente por referencia; su contenido no se duplica en eventos,
   trazas ni cuadros de mando.
5. La indisponibilidad, la falta de evidencia o un esquema desconocido nunca
   se interpretan como autorización, validación, alta, entrega o éxito.
6. No se incorporan valores reales, secretos, credenciales, contenido
   documental ni categorías especiales. Cualquier necesidad futura de estas
   últimas exige una minitarea y base específicas.
7. Versiones, huellas, procedencia, idempotencia, correlación y recibos se
   conservan para exactitud y replay; esto no fija por sí solo un plazo legal
   de conservación.

## Matriz de tratamientos

### CT-T01 — Identidad, autorización y trazabilidad técnica

| Dimensión | Inventario |
| --- | --- |
| Estado | `ARBOL_ACTUAL` |
| Campos o grupos | Referencias opacas de actor, perfil, organización, operación, acción, recurso, finalidad, capacidad, decisión, concesión, correlación, auditoría y versiones/huellas asociadas. |
| Fuente y autoridad | VEC común para identidad, autorización y auditoría transversal; configuración confiable del servidor para emisores y claves. |
| Finalidad cerrada | Denegar o permitir una operación exacta y dejar evidencia técnica de acceso o efecto solicitado. |
| Base propuesta | Cumplimiento de obligación legal y misión en interés público (`RGPD 6.1.c/e`), **pendiente de validación** para cada operación. |
| Interesados/categorías | Personal usuario del sistema; identificadores técnicos seudonimizados y metadatos de actuación. |
| Destinatarios | Unidades competentes y control interno estrictamente por capacidad; detalle y habilitación **pendientes de validación**. |
| Conservación | Plazo y serie **pendientes de Archivo/DPD**; prohibido expurgo automático. |
| Evidencia técnica | Capacidades opacas, consumo único, CAS, recibos, auditoría y outbox en los cortes que los implementan. |

### CT-T02 — Solicitud del centro y expediente administrativo

| Dimensión | Inventario |
| --- | --- |
| Estado | `ARBOL_ACTUAL`; composición real incompleta. |
| Campos o grupos | `CentroRef`, `ContactoRef`, `CategoriaRef`, `GrupoSubgrupo`, `MotivoClave`, `Detalle`, periodo, declaración de RC, referencias documentales y observaciones; referencia, número visible, fase, versión y actuaciones del expediente. |
| Fuente y autoridad | Centro solicitante para la petición; catálogos publicados para motivo/flujo; Documentos para contenido; contratación temporal para expediente y actuaciones. |
| Finalidad cerrada | Iniciar y tramitar una necesidad temporal, conservando qué se pidió y su cronología. |
| Base propuesta | Obligación legal y misión pública en gestión de personal (`RGPD 6.1.c/e`), **pendiente de validación** y de matriz campo–finalidad definitiva. |
| Interesados/categorías | Personal de contacto por referencia; futura persona vinculada al expediente solo cuando proceda. Datos administrativos y económicos de la necesidad, sin contenido documental duplicado. |
| Destinatarios | Centro, RRHH y unidades competentes según fase; relación exacta **pendiente de validación**. |
| Conservación | Serie y plazo del expediente **pendientes de política ENI, Archivo y DPD**. |
| Minimización | Contacto y documentos por referencia; listas y proyecciones separadas; no se inventarian aquí valores de detalle u observaciones. |

### CT-T03 — Análisis RRHH, coste y retención de crédito

| Dimensión | Inventario |
| --- | --- |
| Estado | `ARBOL_ACTUAL` |
| Campos o grupos | Modalidad, categoría, grupo/subgrupo, causa, periodo, jornada, entrada RC ligada por referencia/huella, resultado RC, fuente/recibo/fecha/documento, importe total, fuente de coste, actuación y observaciones. |
| Fuente y autoridad | RRHH para análisis; Intervención o unidad presupuestaria para RC; conector autorizado para coste; Documentos para declaraciones. |
| Finalidad cerrada | Analizar modalidad, necesidad, jornada y periodo; acreditar suficiencia o no exigencia de RC; obtener coste reproducible sin desglose de nómina. |
| Base propuesta | Obligación legal, misión pública y control presupuestario, **pendientes de validación** por Jurídico, RRHH e Intervención. |
| Interesados/categorías | Expediente administrativo y referencias de fuentes; importe agregado interno, sin conceptos retributivos personales. |
| Destinatarios | RRHH, unidad presupuestaria e Intervención por operación; acceso exacto **pendiente de validación**. |
| Conservación | Véase la [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md); plazo todavía no aprobado. |
| Evidencia técnica | Fuente, entrada, huella, recibo, instante, CAS, rectificación append-only y rechazo de ausencia/discordancia. |

### CT-T04 — Consulta y decisión de vía de cobertura

| Dimensión | Inventario |
| --- | --- |
| Estado | `ARBOL_ACTUAL`; O4 material posterior mantiene bloqueos propios. |
| Campos o grupos | Catálogo/política/definición por referencia, versión y huella; comprobaciones, fuente, resultado, vigencia, procedencia, propuesta, motivo gobernado, decisión, rectificación y recibo. |
| Fuente y autoridad | Catálogo de contratación temporal; Bolsa, SAE y convocatorias mediante puertos; RRHH para decisión motivada. |
| Finalidad cerrada | Comparar vías posibles y registrar una decisión explicable sin copiar agregados externos. |
| Base propuesta | Misión pública y obligaciones de gestión de empleo (`RGPD 6.1.c/e`), **pendientes de validación** por RRHH/Jurídico. |
| Interesados/categorías | Necesidad y resultados minimizados; sin identidades de aspirantes en la comparación. |
| Destinatarios | RRHH y órganos competentes para la decisión; proyección minimizada para otras unidades. |
| Conservación | Decisión, fuentes y justificación ligadas al expediente; serie/plazo **pendientes de validación**. |
| Evidencia técnica | Procedencia, tiempos, huellas, motivos versionados, CAS, autorización y recibos donde están implementados. |

### CT-T05 — Asignación organizativa y bandejas internas

| Dimensión | Inventario |
| --- | --- |
| Estado | `ARBOL_ACTUAL` para dominio; persistencia/composición del hito continúa incompleta. |
| Campos o grupos | Unidad, responsable y ámbito por referencia; motivo, versión, actuación, correlación, capacidad y recibo. |
| Fuente y autoridad | Historia organizativa e identidad corporativa para unidad/vínculo; contratación temporal para asignación del expediente. |
| Finalidad cerrada | Encaminar el expediente a una unidad competente y registrar reasignaciones motivadas. |
| Base propuesta | Organización administrativa y misión pública, **pendientes de validación** de competencia y acceso. |
| Interesados/categorías | Metadatos organizativos y referencias de personal interno; ninguna identidad se deriva de formulario o navegador. |
| Destinatarios | Unidad asignada, RRHH y supervisión autorizada; relación exacta **pendiente de validación**. |
| Conservación | Historial append-only previsto; plazo **pendiente de Archivo/DPD**. |

### CT-T06 — Coordinación con Bolsa y llamamiento

| Dimensión | Inventario |
| --- | --- |
| Estado | Contrato de integración `ARBOL_ACTUAL`; selección, comunicaciones y formalización permanecen futuras. |
| Campos o grupos | Necesidad, categoría, bolsa, política, orden, acción, resultado, totales, seudónimo, evento/acuse, procedencia, evidencias, idempotencia, correlación y recibos, todos por referencia o valor agregado mínimo. |
| Fuente y autoridad | Bolsa para convocatoria, orden, disponibilidad, exclusiones y llamamiento; contratación temporal solo coordina. |
| Finalidad cerrada | Consultar disponibilidad y coordinar una propuesta/llamamiento sin acceder a tablas ni copiar participantes. |
| Base propuesta | Obligación legal y misión pública en selección/cobertura, **pendientes de validación**. La fase contractual no se presume activa. |
| Interesados/categorías | Integrantes de Bolsa representados mediante referencias o seudónimos; no se inventarían identidad, contacto, puntuación ni causa detallada. |
| Destinatarios | Bolsa y RRHH por contratos de módulo; comunicaciones, Seguridad Social u otros destinatarios son `FUTURO` y requieren validación propia. |
| Conservación | Eventos/recibos según expediente y autoridad Bolsa; plazos **pendientes de validación**. |
| Bloqueo | Cero comunicación, aceptación, renuncia, nombramiento o firma en este corte. |

### CT-T07 — Seguimiento, prórroga, incidencia y cese

| Dimensión | Inventario |
| --- | --- |
| Estado | Modelo de dominio `ARBOL_ACTUAL`; caso de uso, persistencia y efectos futuros. |
| Campos o grupos | Definición/version/huella/vigencia; estados, motivos, transiciones, requisitos documentales y de calendario; periodos, documentos por referencia, evidencia de calendario, actor, unidad, instante, correlación, recibo y rectificaciones. |
| Fuente y autoridad | Catálogos gobernados para flujo; calendario mediante autoridad propia; Documentos por referencia; Personal conserva incorporación y cese reales. |
| Finalidad cerrada | Modelar la cronología administrativa y sus rectificaciones sin ejecutar incorporación, prórroga o cese real. |
| Base propuesta | Gestión de la relación de empleo y misión pública, **pendientes de validación** y de competencia para cada transición. |
| Interesados/categorías | Referencias administrativas de la relación y actores internos; sin contenido documental ni calendario copiado. |
| Destinatarios | RRHH, unidad y Personal únicamente cuando una minitarea autorizada componga el efecto. |
| Conservación | Historia append-only técnica; plazo/serie **pendientes de política ENI, Archivo y DPD**. |

### CT-T08 — Proyecciones RRHH, acceso y auditoría

| Dimensión | Inventario |
| --- | --- |
| Estado | Contratos/proyecciones internas `ARBOL_ACTUAL`; superficie productiva no acreditada. |
| Campos o grupos | Contexto interno, filtros, cursores, orden, límites, referencias de expediente/resultado, versiones, huellas, concesión, guardianes, motivos de acceso, recibos y sellos de consulta. |
| Fuente y autoridad | Contratación temporal para proyección; VEC común para identidad/capacidad; autoridades del expediente para contenido enlazado. |
| Finalidad cerrada | Consulta interna minimizada y trazabilidad de lectura por operación, expediente, unidad y finalidad. |
| Base propuesta | Obligación legal/misión pública y seguridad (`RGPD 6.1.c/e`), **pendientes de validación** por perfil y finalidad. |
| Interesados/categorías | Metadatos del expediente y referencias de actores; vistas públicas o de otros perfiles deben ser distintas. |
| Destinatarios | RRHH y control interno según capacidad; exportación o descarga real requieren tarea y autorización específicas. |
| Conservación | Accesos y recibos sujetos a plazos de responsabilidad **pendientes de validación**. |

### CT-T09 — Alta con Personal/RPT

| Dimensión | Inventario |
| --- | --- |
| Estado | `CANDIDATO_LOCAL` `a312375`; no integrado ni activo. |
| Campos o grupos | Esquema/versión, solicitud, expediente/versión, capacidad, correlación, idempotencia, fuente RPT por referencia/versión/huella, puesto, plaza, resultado/recibo y, de forma excluyente, relación/ocupación o motivo de rechazo gobernado. |
| Fuente y autoridad | Contratación temporal solicita; Personal/RPT confirma o rechaza y conserva relación, ocupación, puesto e incorporación. |
| Finalidad cerrada | Preparar un contrato neutral de alta y verificar coherencia estructural, sin realizar el alta. |
| Base propuesta | Gestión de personal y misión pública, **pendientes de validación**; la referencia de capacidad no concede autoridad por sí sola. |
| Interesados/categorías | Solo referencias opacas y metadatos técnicos; ningún dato personal real. |
| Destinatarios | Personal cuando exista adaptador/composición autorizados; hoy ninguno. |
| Conservación | No definida; recibo y resultado futuros quedan bloqueados hasta política aprobada. |

### CT-T10 — Modelo y mapeo GINPIX

| Dimensión | Inventario |
| --- | --- |
| Estado | `CANDIDATO_LOCAL` `2b15fe8`; no integrado, enviado ni activo. |
| Campos o grupos | Esquema/modelo/mapeo por versión, procedencia y huella; expediente/incorporación por referencia; correlación/idempotencia; campos canónicos sintéticos con estados distintos de ausente, nulo y valor. |
| Fuente y autoridad | Especificación gobernada RRHH/GINPIX; contratación temporal prepara; GINPIX conserva el sistema externo. |
| Finalidad cerrada | Validar un modelo canónico y compatibilidad de mapeo sin activar API, fichero ni entrega. |
| Base propuesta | Gestión de personal, **pendiente de validación** campo a campo antes de incorporar datos personales. |
| Interesados/categorías | En el candidato solo fixtures sintéticos y referencias opacas. Categorías reales todavía no autorizadas. |
| Destinatarios | Ninguno en este corte; GINPIX solo tras adaptador, seguridad, recibo y aprobación formal. |
| Conservación/transferencia | No definida. No se acredita transferencia ni exportación; cualquier salida permanece prohibida. |

## Destinatarios, transferencias y encargados pendientes

- No existe en este corte una lista corporativa aprobada de destinatarios,
  encargados, subencargados o accesos extraordinarios para contratación
  temporal.
- No se acredita ninguna transferencia internacional. La ausencia de una
  transferencia prevista no equivale a una decisión formal sobre proveedores
  o servicios futuros.
- Comunicaciones a otras administraciones, órganos de control, órganos
  judiciales, Seguridad Social, administración tributaria o personas
  interesadas requieren finalidad, base, minimización y canal aprobados en la
  minitarea que materialice cada entrega.
- Los adaptadores no pueden ampliar campos ni reutilizar datos por conveniencia;
  traducen el contrato propietario y fallan cerrados ante diferencias.

## Conservación, bloqueo y derechos

No hay un calendario aprobado para estas familias. Hasta cerrar `CT-CUM-06`
y obtener las validaciones de Archivo y DPD:

1. no se programa expurgo ni borrado silencioso;
2. el contenido documental permanece en su autoridad y este módulo conserva
   referencias;
3. una obligación de conservación, investigación o litigio debe poder bloquear
   cualquier eliminación futura;
4. rectificaciones y reaperturas se expresan como historia añadida, no como
   reescritura probatoria;
5. acceso, rectificación, oposición o limitación no se declaran operativos sin
   procedimiento, autoridad y E2E propios.

## Bloqueos que permanecen vigentes

Este candidato no cierra ni sustituye:

- `CT-CUM-03`: RAT corporativo y EIPD;
- `CT-CUM-04`: categorización ENS y declaración de aplicabilidad;
- `CT-CUM-05`: análisis y tratamiento de riesgos;
- `CT-CUM-06`: política ENI y conservación;
- `CT-CUM-07`: competencia y actuaciones humanas/automatizadas;
- `CT-CUM-08`: evaluación y declaración de accesibilidad;
- `CT-CUM-09`: gobierno de casos de IA;
- `CT-CUM-10`: auditoría y actas conjuntas.

Por ello siguen prohibidos datos reales, efectos jurídicos, altas en Personal,
envíos a GINPIX, comunicaciones, exportaciones, preproducción y producción.
El estado y los porcentajes del tablero no se modifican con este documento.

## Trazabilidad y criterio de revisión

Fuentes principales del inventario:

- [expediente RRHH](expediente_contratacion_temporal_rrhh.md);
- [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md);
- [mapa de tareas](mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md);
- [tablero](tablero_tareas_contratacion_temporal_2026-07-23.md);
- [matriz normativa](matriz_normativa_contratacion_temporal_2026-07-23.md);
- [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md);
- contratos del árbol bajo `internal/modules/contrataciontemporal/domain` y
  `internal/modules/contrataciontemporal/ports`.

La revisión independiente debe comprobar, como mínimo:

1. que cada familia declare campos, fuente/autoridad, finalidad, base propuesta,
   interesados, destinatarios y conservación;
2. que no aparezcan valores reales, secretos o contenido documental;
3. que candidatos locales y estado vivo no se confundan;
4. que ninguna incertidumbre se transforme en aprobación;
5. que los enlaces locales existan y el write-set sea únicamente este fichero;
6. que `git diff --check` termine sin hallazgos.

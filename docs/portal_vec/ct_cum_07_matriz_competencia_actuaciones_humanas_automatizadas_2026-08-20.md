# CT-CUM-07 — Matriz candidata de competencia y actuaciones humanas o automatizadas

Fecha de corte: 20 de agosto de 2026.

Base documental: `74d20d309d4cf4030f8fd15e45caac2c9ab155cd`.

Estado: **PENDIENTE_VALIDACION; no aprobada; cero resoluciones o efectos autorizados**.

## Autoridad, capability e invariante

La
[matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-07` como la matriz de competencia y actuaciones humanas y
automatizadas. Esta puerta bloquea toda resolución o efecto jurídico.

La capability de este dossier se limita a preparar una estructura candidata
que obligue a identificar competencia, intervención humana, automatización,
firma y evidencia para cada actuación. No constituye:

- una atribución, delegación, suplencia o resolución de competencia;
- una autorización de actuación administrativa automatizada;
- una clasificación o habilitación de IA;
- una matriz nominativa de personas, puestos o unidades reales;
- una decisión, informe, fiscalización, resolución, firma o notificación;
- una activación de datos, integraciones, preproducción o producción.

La invariante es estricta: un rol, permiso técnico, identificador, pantalla,
regla, algoritmo, modelo o resultado externo nunca concede competencia ni
produce un acto. Sin autoridad acreditada y revisión humana exigible, el
resultado máximo es una propuesta no efectiva y el flujo se detiene.

## Fuentes y límites

El dossier usa únicamente:

- el
  [inventario candidato CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- el [dossier RAT/EIPD candidato CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- el
  [dossier ENS/SoA candidato CT-CUM-04](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- el
  [plan de riesgos candidato CT-CUM-05](ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md);
- la
  [política ENI candidata CT-CUM-06](ct_cum_06_politica_eni_documento_expediente_firma_conservacion_2026-08-20.md);
- el [expediente remitido por RRHH](expediente_contratacion_temporal_rrhh.md);
- los [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md) y
  la
  [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md).

No se incorporan nombres, identidades, puestos, unidades, delegaciones,
resoluciones, expedientes, datos, documentos, sistemas, proveedores o
credenciales reales. CT-CUM-02 a CT-CUM-06 siguen siendo candidatos locales;
este dossier no cambia su estado transversal.

## Vocabulario de estado

| Estado | Significado cerrado |
| --- | --- |
| `PENDIENTE_VALIDACION` | Requiere decisión de la autoridad competente. |
| `COMPETENCIA_NO_ACREDITADA` | No existe evidencia suficiente para atribuir la actuación. |
| `SOLO_PROPUESTA` | Salida informativa o preparatoria sin efecto. |
| `REVISION_HUMANA_OBLIGATORIA` | Una persona competente debe evaluar y decidir; no es un gesto formal. |
| `AUTOMATIZACION_PROHIBIDA` | No puede configurarse actuación automatizada. |
| `IA_CERRADA` | El caso queda bloqueado hasta CT-CUM-09 y sus demás obligaciones. |
| `EFECTO_PROHIBIDO` | No se ejecuta ni confirma una actuación real. |
| `BLOQUEANTE` | Su ausencia impide resolución o efecto jurídico. |

No se usa `COMPETENTE`, `APROBADO`, `RESUELTO`, `FIRMADO`, `AUTOMATIZADO` o
`NOTIFICADO` como estado real del sistema.

# Parte A — Modelo candidato de competencia

## Separación de autoridades

| Concepto | Qué acredita | Qué no acredita |
| --- | --- | --- |
| Identidad | Quién actúa desde una frontera confiable. | Puesto, competencia o permiso para el acto. |
| Perfil o rol técnico | Acceso potencial a una superficie. | Competencia organizativa o jurídica. |
| Capacidad opaca | Autorización técnica exacta, limitada y vigente. | Título de atribución, delegación o suplencia. |
| Puesto o función | Posición organizativa gobernada. | Ocupación vigente o competencia para todo acto. |
| Competencia | Título, ámbito, acto y vigencia acreditados. | Acceso técnico automático. |
| Delegación o suplencia | Ejercicio derivado dentro de alcance y vigencia exactos. | Transferencia general o permanente de autoridad. |
| Firma o sello | Vinculación criptográfica y evidencia según política. | Competencia si esta no estaba acreditada. |
| Revisión humana | Evaluación significativa por persona competente. | Validación automática por mera pulsación. |

Un efecto futuro exigirá simultáneamente identidad, acceso técnico,
competencia, finalidad, ámbito, vigencia y ausencia de incompatibilidad. La
falta de cualquiera provoca `EFECTO_PROHIBIDO`.

## Ficha obligatoria por actuación

| Campo | Requisito candidato |
| --- | --- |
| Actuación | Tipo publicado y versión, distinto de la acción de interfaz. |
| Procedimiento | Referencia gobernada, fase y expediente. |
| Resultado | Borrador, propuesta, informe, decisión u otro estado cerrado. |
| Autoridad | Órgano o función competente, pendiente de designación formal. |
| Título | Norma, resolución, delegación o suplencia mediante referencia. |
| Ámbito | Materia, territorio, cuantía, fase y otras condiciones aplicables. |
| Vigencia | Inicio, fin, suspensión y versión del título. |
| Actor | Identidad atestada y vínculo vigente con la función. |
| Acceso | Capacidad exacta, finalidad, recurso, acción y caducidad. |
| Segregación | Incompatibilidades, abstención, doble control y separación de funciones. |
| Documentos | Versiones y huellas por referencia, sin contenido duplicado. |
| Automatización | Clasificación, resolución habilitante, órganos y supervisión. |
| Revisión humana | Alcance, información disponible, alternativas, tiempo y facultad real de cambiar el resultado. |
| Firma/sello | Política y evidencia separadas, sujetas a CT-CUM-06. |
| Recibo | Correlación, idempotencia, instante, versión y resultado exactos. |

# Parte B — Matriz funcional candidata

La matriz identifica clases de actuación, no personas ni competencias reales.
Todas las autoridades permanecen `PENDIENTE_VALIDACION`.

| ID | Actuación | Naturaleza máxima permitida en el corte | Autoridad pendiente | Revisión humana | Automatización/IA | Estado |
| --- | --- | --- | --- | --- | --- | --- |
| AC-01 | Guardar borrador | Preparación reversible sin presentación ni efecto. | Titularidad y ámbito del borrador. | Responsable de su contenido antes de presentar. | Sin IA; reglas técnicas no deciden. | `SOLO_PROPUESTA` |
| AC-02 | Presentar o registrar | Ningún envío real; solo preparación y validación estructural. | Órgano, canal y persona legitimada. | Obligatoria antes del efecto. | `AUTOMATIZACION_PROHIBIDA` sin resolución. | `EFECTO_PROHIBIDO` |
| AC-03 | Requerir o subsanar | Borrador de requerimiento o respuesta. | Competencia y representación pendientes. | Obligatoria y significativa. | IA cerrada; plazos no se deciden automáticamente. | `EFECTO_PROHIBIDO` |
| AC-04 | Informar técnicamente | Proyecto de informe reproducible y trazable. | Autoría, función y alcance del informe. | Firma/revisión por persona competente. | Herramientas solo asisten; no sustituyen juicio. | `SOLO_PROPUESTA` |
| AC-05 | Informar jurídicamente | Estructura o borrador sin conclusión jurídica. | Función jurídica competente. | Obligatoria; no delegable en algoritmo. | `IA_CERRADA` | `EFECTO_PROHIBIDO` |
| AC-06 | Fiscalizar | Datos y comprobaciones preparatorias sin resultado fiscal. | Intervención o autoridad competente. | Obligatoria según procedimiento aprobado. | `AUTOMATIZACION_PROHIBIDA` sin resolución específica. | `EFECTO_PROHIBIDO` |
| AC-07 | Proponer cobertura o nombramiento | Propuesta motivada sin aceptación ni efecto. | Función proponente y límites pendientes. | Obligatoria antes de elevar. | IA cerrada en empleo/selección. | `SOLO_PROPUESTA` |
| AC-08 | Resolver | Ninguna resolución real. | Órgano resolutor y título pendientes. | Decisión por autoridad competente. | `AUTOMATIZACION_PROHIBIDA` salvo resolución formal futura. | `BLOQUEANTE` |
| AC-09 | Firmar o sellar | Encargo candidato, sin firma/sello real. | Firmante, suplencia/delegación y política. | Verificación de competencia en el instante. | Sello automatizado solo con habilitación formal. | `EFECTO_PROHIBIDO` |
| AC-10 | Notificar o publicar | Preparación de contenido y destinatario, sin comunicación. | Órgano, canal, destinatario y finalidad. | Revisión previa de contenido, acceso y datos. | `AUTOMATIZACION_PROHIBIDA` sin autoridad. | `EFECTO_PROHIBIDO` |
| AC-11 | Incorporar en Personal/RPT | Solicitud neutral por referencias, sin alta. | Personal conserva relación, puesto y ocupación. | Confirmación/rechazo por su autoridad. | Sin IA; esquema desconocido se deniega. | `EFECTO_PROHIBIDO` |
| AC-12 | Preparar entrega GINPIX | Modelo/mapeo sintético sin envío. | RRHH/GINPIX y campos pendientes. | Revisión de esquema, finalidad y contenido. | Sin decisión sobre personas; CT-CUM-09 si se añade IA. | `EFECTO_PROHIBIDO` |
| AC-13 | Rectificar, revocar o reabrir | Propuesta de nueva actuación, sin reescribir historia. | Misma autoridad o la que corresponda formalmente. | Obligatoria con motivo y alcance. | `AUTOMATIZACION_PROHIBIDA` sin resolución. | `EFECTO_PROHIBIDO` |
| AC-14 | Cerrar o archivar expediente | Preparación de índice/estado, sin cierre real. | Órgano y Archivo según política. | Revisión de integridad, bloqueos y completitud. | Sin automatización autorizada. | `EFECTO_PROHIBIDO` |

La matriz no es exhaustiva. Una actuación no catalogada se deniega y exige
nueva entrada versionada y revisión independiente.

# Parte C — Delegación, suplencia, abstención y segregación

## Evidencia mínima pendiente

| Situación | Evidencia requerida | Resultado sin evidencia |
| --- | --- | --- |
| Titularidad | Puesto/función, persona por referencia, vigencia y fuente atestada. | `COMPETENCIA_NO_ACREDITADA` |
| Delegación | Resolución, acto, ámbito, condiciones, publicación y vigencia. | `EFECTO_PROHIBIDO` |
| Suplencia | Causa, orden, periodo, función sustituida y límites. | `EFECTO_PROHIBIDO` |
| Abstención/recusación | Incidencia, alcance y decisión de sustitución competente. | Actuación detenida. |
| Órgano colegiado | Composición, convocatoria, quorum, votación, acta y firma. | `EFECTO_PROHIBIDO` |
| Doble control | Operaciones, participantes incompatibles y evidencia de ambos pasos. | Actuación detenida. |
| Caducidad o revocación | Instante fiable y propagación a capacidades pendientes. | Acceso y efecto denegados. |

Las cuentas compartidas o genéricas no acreditan actor, competencia ni
responsabilidad. Una sustitución no prevista no se resuelve reutilizando el
permiso de la persona sustituida.

## Segregaciones candidatas

- solicitud frente a análisis o aprobación;
- preparación frente a firma o sello;
- informe frente a resolución cuando la autoridad lo exija;
- fiscalización frente al acto fiscalizado;
- operación técnica frente a decisión funcional;
- desarrollo frente a administración y auditoría;
- gestión de Bolsa frente a relación/ocupación de Personal;
- preparación GINPIX frente a recepción/conciliación del sistema externo.

La autoridad competente debe validar cada incompatibilidad. Esta lista no
atribuye funciones ni afirma que una separación esté implantada.

# Parte D — Actuación administrativa automatizada

## Regla de cierre

No existe actuación administrativa automatizada autorizada en este corte.
Antes de configurarla deben constar resolución habilitante y, al menos:

| Dimensión | Evidencia pendiente |
| --- | --- |
| Especificaciones | Órgano responsable, alcance y versión de reglas. |
| Programación | Órgano responsable, repositorio, cambios y separación de funciones. |
| Mantenimiento | Responsable, vigencia, incidencias y reversión. |
| Supervisión | Autoridad humana, señales, intervención y suspensión. |
| Control de calidad | Casos, límites, errores, sesgo y criterios de aceptación. |
| Auditoría | Independencia, registros, acceso, hallazgos y remediación. |
| Impugnación | Canal, información, revisión humana y rectificación. |
| Firma/sello | Sistema legalmente previsto y política CT-CUM-06 aprobada. |
| Seguridad | Riesgos CT-CUM-05, identidad, integridad, trazabilidad y continuidad. |
| Datos | Finalidad, base, minimización, exactitud, conservación y derechos. |

Sin esa resolución, cualquier cálculo produce `SOLO_PROPUESTA` y exige una
decisión humana competente. La indisponibilidad o confianza estadística nunca
se convierten en aprobación.

## Registro candidato por automatización

Cada caso futuro deberá conservar referencia, versión, finalidad, entradas,
fuentes, reglas, salida, incertidumbres, motivo, responsable, revisión humana,
firma/sello, recibo, incidentes, auditoría y vigencia. Los datos y decisiones
reales permanecen fuera de este dossier.

# Parte E — IA y revisión humana

## Frontera con CT-CUM-09

CT-CUM-07 no clasifica ni autoriza sistemas de IA. Cualquier funcionalidad que
use IA en contratación, selección, evaluación, promoción, asignación de tareas
o supervisión laboral queda `IA_CERRADA` hasta CT-CUM-09, EIPD y demás
obligaciones aplicables.

Un asistente meramente informativo tampoco recibe datos internos, amplía
permisos o produce propuestas individuales sin caso de uso y autoridad
separados.

## Revisión humana significativa

La revisión futura no puede reducirse a confirmar por defecto una salida. Debe
garantizar que la persona competente:

1. conoce finalidad, alcance, datos, fuentes, versión y limitaciones;
2. dispone de tiempo e información comprensible para evaluar;
3. puede solicitar evidencia, corregir, rechazar, detener o escalar;
4. no sufre incentivos o interfaz que induzcan aceptación automática;
5. registra motivo propio y no copia sin examen la explicación del sistema;
6. entiende consecuencias para personas y vías de impugnación;
7. conserva competencia y ausencia de incompatibilidad en ese instante;
8. deja trazabilidad de entrada, salida, cambios y decisión final.

La evidencia de una revisión humana debe demostrar intervención material, no
solo una marca, un tiempo mínimo o una sesión abierta.

## Atributos y reglas prohibidos

No se autoriza utilizar atributos protegidos, categorías especiales, proxies
no aprobados, inferencias no gobernadas o datos obtenidos para otra finalidad.
Tampoco se autoriza entrenar, ajustar o evaluar IA con datos reales del módulo.

# Parte F — Evidencia y gobierno

## Cadena de evidencia por actuación

Una futura actuación efectiva deberá enlazar:

1. tipo y versión de actuación;
2. procedimiento, fase, expediente y finalidad;
3. identidad, puesto o función y competencia acreditados;
4. título, ámbito, vigencia y posibles sustituciones;
5. capacidad técnica exacta y consumida con el efecto cuando corresponda;
6. documentos, fuentes, reglas y versiones por referencia y huella;
7. intervención humana o resolución de automatización aplicable;
8. firma/sello, registro y recibos cuando procedan;
9. auditoría, rectificación, impugnación y conservación;
10. revisión independiente y defectos abiertos.

Un test local o una revisión de código no acredita competencia ni efecto.

## Decisiones reservadas

| Decisión | Autoridad requerida | Estado |
| --- | --- | --- |
| Catálogo de actuaciones | Secretaría, RRHH y autoridades funcionales competentes | `PENDIENTE_VALIDACION` |
| Competencia por actuación | Órgano competente y título formal | `COMPETENCIA_NO_ACREDITADA` |
| Delegación/suplencia | Resolución y fuente organizativa autorizada | `PENDIENTE_VALIDACION` |
| Segregación e incompatibilidades | Autoridades organizativas, Seguridad y control | `PENDIENTE_VALIDACION` |
| Actuación automatizada | Resolución formal y órganos responsables | `AUTOMATIZACION_PROHIBIDA` |
| Caso de uso de IA | CT-CUM-09 y autoridades aplicables | `IA_CERRADA` |
| Resolución o efecto jurídico | Autoridad formal tras cerrar dependencias | `BLOQUEANTE` |

## Bloqueos conservados

- CT-CUM-07 permanece abierto y no aprobado.
- CT-CUM-08 a CT-CUM-10 conservan sus puertas propias.
- CT-CUM-02 a CT-CUM-06 siguen siendo candidatos locales hasta integración y
  aprobación competentes.
- O4 y los demás carriles técnicos conservan NO-GO y dependencias.
- Datos reales, automatización, IA, informes efectivos, fiscalización,
  resolución, firma, registro, notificación, altas, envíos, preproducción y
  producción siguen prohibidos.
- El tablero, métricas y documentos transversales no cambian.

## Criterio técnico de revisión

La revisión independiente solo puede comprobar que el candidato:

1. separa identidad, acceso, puesto, competencia, firma y revisión humana;
2. cubre las clases de actuación sin atribuir autoridades reales;
3. bloquea delegación, suplencia, automatización e IA sin evidencia formal;
4. exige revisión humana significativa y capacidad real de cambiar la salida;
5. conserva CT-CUM-07 y resolución/efecto jurídico como bloqueos;
6. no contiene personas, decisiones, expedientes, datos, proveedores,
   credenciales o infraestructura reales;
7. resuelve enlaces locales, cita una base Git existente y supera
   `git diff --check`.

Un `GO` documental no cierra CT-CUM-07, no atribuye competencia y no autoriza
automatización, IA, resolución o efecto.

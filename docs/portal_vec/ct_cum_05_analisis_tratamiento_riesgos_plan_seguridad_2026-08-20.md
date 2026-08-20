# CT-CUM-05 — Análisis y tratamiento de riesgos y plan de seguridad candidatos

Fecha de corte: 20 de agosto de 2026.

Base documental: `946256e1682c33327918f2be15df1e9899b331d5`.

Estado: **PENDIENTE_VALIDACION; no aprobado; ningún riesgo aceptado**.

## Autoridad, capability y efecto

La
[matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-05` como el análisis y tratamiento de riesgos y el plan de
seguridad. Esta puerta bloquea el paso a preproducción.

La capability de este dossier se limita a ordenar escenarios candidatos,
consecuencias posibles, medidas propuestas, evidencia exigida y decisiones
pendientes. No constituye:

- un inventario de activos o infraestructura reales;
- una valoración aprobada de probabilidad, impacto o riesgo residual;
- la implantación o eficacia demostrada de una medida;
- una aceptación, transferencia o cierre de riesgo;
- un plan operativo aprobado, con responsables o fechas comprometidos;
- una autorización de datos reales, servicios, preproducción o producción.

La ausencia de información, evidencia, responsable o aprobación conserva el
riesgo abierto y la puerta cerrada. Ningún estado de este documento significa
que el sistema sea seguro o conforme.

## Fuentes y alcance

El dossier usa únicamente:

- el
  [inventario candidato CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- el [dossier RAT/EIPD candidato CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- el
  [dossier ENS/SoA candidato CT-CUM-04](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- el [expediente remitido por RRHH](expediente_contratacion_temporal_rrhh.md);
- los [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md);
- la
  [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md);
- el [tablero](tablero_tareas_contratacion_temporal_2026-07-23.md) y el
  [mapa de paralelización](mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md).

CT-CUM-02, CT-CUM-03 y CT-CUM-04 siguen siendo candidatos locales. Este
dossier no integra sus revisiones ni altera el estado transversal. Tampoco
incorpora nombres de activos, ubicaciones, dominios, direcciones, redes,
cuentas, proveedores, configuraciones, secretos o valores de personas.

## Invariante de valoración

La valoración formal requiere delimitar sistema, activos, dependencias,
amenazas, vulnerabilidades, salvaguardas, propietarios y contexto operativo.
Esos elementos no están aprobados. Por ello:

1. probabilidad e impacto quedan `PENDIENTE_VALORACION`;
2. no se calcula ni etiqueta riesgo inherente o residual;
3. las medidas son `TRATAMIENTO_PROPUESTO`, no controles implantados;
4. cada medida requiere responsable, plazo y evidencia aprobados;
5. cualquier excepción o no aplicabilidad permanece bloqueada;
6. la única decisión admisible en este corte es mantener el riesgo abierto.

## Vocabulario de estado

| Estado | Significado cerrado |
| --- | --- |
| `RIESGO_IDENTIFICADO` | Escenario candidato que debe valorar la autoridad competente. |
| `PENDIENTE_VALORACION` | Probabilidad, impacto o alcance no han sido aprobados. |
| `TRATAMIENTO_PROPUESTO` | Medida candidata sin implantación ni eficacia acreditadas. |
| `EVIDENCIA_NO_APORTADA` | No existe evidencia aprobada para el alcance requerido. |
| `NO_EVALUADO` | El escenario o dependencia carece todavía de análisis suficiente. |
| `RESPONSABLE_PENDIENTE` | La autoridad no ha asignado propietario formal. |
| `RESIDUAL_NO_CALCULADO` | No puede derivarse riesgo residual con los datos disponibles. |
| `NO_ACEPTADO` | El riesgo permanece abierto y no autoriza avance. |
| `BLOQUEANTE` | Impide el efecto asociado mientras no se cierre formalmente. |

# Parte A — Método candidato

## Unidad de análisis futura

Cada ficha aprobable deberá enlazar, como mínimo:

| Elemento | Información requerida |
| --- | --- |
| Contexto | Finalidad, proceso, tratamiento, fase y dependencia afectados. |
| Bien protegido | Información, servicio o función mediante referencia gobernada. |
| Dimensiones | Disponibilidad, autenticidad, integridad, confidencialidad y trazabilidad. |
| Origen | Amenaza o condición de fallo, sin atribuir intención cuando no conste. |
| Evento | Suceso observable y delimitado. |
| Consecuencia | Daño posible a personas, procedimiento, organización o servicio. |
| Salvaguardas existentes | Solo las demostradas para el alcance exacto, con evidencia. |
| Probabilidad e impacto | Escalas, criterios, justificación y autoridad aprobadora. |
| Tratamiento | Evitar o mitigar; otras opciones requieren decisión formal específica. |
| Evidencia | Prueba, registro, procedimiento, revisión y vigencia. |
| Residual | Recalculado después de demostrar eficacia, nunca por intención. |
| Decisión | Aprobación, rechazo o remediación por la autoridad competente. |

## Reglas de evaluación

- Una prueba unitaria no acredita una salvaguarda organizativa u operativa.
- Una medida de diseño no reduce riesgo hasta demostrar implantación y
  eficacia en el entorno autorizado.
- Un resultado verde aislado no sustituye una evidencia repetible y ligada a
  la versión evaluada.
- La externalización no elimina responsabilidad ni riesgo.
- La indisponibilidad, error, esquema desconocido o evidencia incompleta se
  interpretan de forma cerrada.
- Un riesgo que afecte a varios tratamientos conserva trazabilidad por cada
  finalidad y autoridad; no se legitima acceso transversal por conveniencia.
- Riesgo residual alto o incierto no se acepta desde este repositorio y debe
  escalarse según las autoridades competentes.

# Parte B — Registro candidato de riesgos

Las fichas siguientes desarrollan los doce escenarios `E01`–`E12` de
CT-CUM-03. No asignan frecuencia, severidad ni nivel.

## R-01 — Identidad, perfil o capacidad no acreditados

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una solicitud usa identidad, perfil, organización o capacidad que no procede de una frontera confiable o no está ligada a la operación. |
| Consecuencia | Acceso, consulta o actuación sin competencia; atribución incorrecta y posible exposición o efecto indebido. |
| Dimensiones | Autenticidad, confidencialidad, integridad y trazabilidad. |
| Tratamiento propuesto | Frontera atestada, garantía adecuada, privilegio mínimo, capacidad ligada a acción/recurso/finalidad, consumo único y segregación administrativa. |
| Evidencia requerida | Modelo de confianza aprobado, pruebas de negativas, alta/baja/revisión de acceso, trazas minimizadas y revisión independiente. |
| Estado | `PENDIENTE_VALORACION`; `EVIDENCIA_NO_APORTADA`; `NO_ACEPTADO`. |

## R-02 — Dato inexacto o procedencia discordante

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Un dato está desactualizado, incompleto, atribuido a otra fuente o no coincide con versión y huella esperadas. |
| Consecuencia | Análisis, decisión, cálculo o expediente incorrectos; dificultad para rectificar y rendir cuentas. |
| Dimensiones | Integridad, autenticidad y trazabilidad. |
| Tratamiento propuesto | Fuente gobernada, versión, huella, recibo, vigencia, CAS, reconciliación y rectificación de solo adición. |
| Evidencia requerida | Catálogo de fuentes y propietarios, contrato de rectificación, casos de discordancia, reintento seguro y revisión de conflictos. |
| Estado | `PENDIENTE_VALORACION`; `RESIDUAL_NO_CALCULADO`; `NO_ACEPTADO`. |

## R-03 — Duplicación de autoridad entre módulos

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Contratación temporal copia o modifica agregados de Bolsa, Personal, Documentos u otra autoridad. |
| Consecuencia | Divergencia, acceso excesivo, decisiones inconsistentes y pérdida de propiedad del dato. |
| Dimensiones | Integridad, confidencialidad, autenticidad y trazabilidad. |
| Tratamiento propuesto | Puertos mínimos, referencias opacas, comandos/eventos/recibos y prohibición de tablas ajenas. |
| Evidencia requerida | Diagrama de propiedad aprobado, inventario de intercambios, pruebas de contrato y auditoría de composición. |
| Estado | `RIESGO_IDENTIFICADO`; `TRATAMIENTO_PROPUESTO`; `NO_ACEPTADO`. |

## R-04 — Exceso de datos en vistas, eventos, registros o exportaciones

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una proyección, log, evento, error o salida contiene campos ajenos a la finalidad. |
| Consecuencia | Revelación, reutilización incompatible, ampliación de destinatarios o persistencia no gobernada. |
| Dimensiones | Confidencialidad y trazabilidad. |
| Tratamiento propuesto | DTO cerrados, referencias, redacción, límites, vistas separadas, listas blancas y revisión de finalidad. |
| Evidencia requerida | Inventario campo–finalidad, pruebas de minimización, inspección de errores/logs y autorización específica de cualquier exportación. |
| Estado | `PENDIENTE_VALORACION`; `EVIDENCIA_NO_APORTADA`; `NO_ACEPTADO`. |

## R-05 — Replay, colisión o falso éxito

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una repetición, colisión, respuesta perdida o confirmación ambigua se interpreta como un efecto nuevo o exitoso. |
| Consecuencia | Doble actuación, estados incompatibles, pérdida de recibo o avance administrativo falso. |
| Dimensiones | Integridad, autenticidad, disponibilidad y trazabilidad. |
| Tratamiento propuesto | Idempotencia semántica, correlación, huella de entrada, versión, recibo, CAS y reconciliación explícita. |
| Evidencia requerida | Concurrencia, reinicio, respuesta perdida, replay y reconciliación en el entorno autorizado, sin datos reales. |
| Estado | `PENDIENTE_VALORACION`; `TRATAMIENTO_PROPUESTO`; `NO_ACEPTADO`. |

## R-06 — Dependencia indisponible tratada como validación

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Un fallo, timeout, respuesta incompleta o esquema desconocido se convierte en autorización, ausencia o éxito. |
| Consecuencia | Avance indebido, omisión de comprobación o pérdida de coherencia entre autoridades. |
| Dimensiones | Disponibilidad, integridad y autenticidad. |
| Tratamiento propuesto | Denegación predeterminada, estados recuperables explícitos, límites temporales y ninguna equivalencia entre error y resultado válido. |
| Evidencia requerida | Matriz de fallos por dependencia, pruebas de timeout/cancelación, telemetría gobernada y procedimiento de recuperación. |
| Estado | `RIESGO_IDENTIFICADO`; `EVIDENCIA_NO_APORTADA`; `NO_ACEPTADO`. |

## R-07 — Conservación o eliminación indebidas

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Información se conserva sin límite aprobado o se elimina antes de finalizar obligación, bloqueo o necesidad probatoria. |
| Consecuencia | Exposición prolongada, incumplimiento, pérdida de derechos o de evidencia administrativa. |
| Dimensiones | Confidencialidad, integridad, disponibilidad y trazabilidad. |
| Tratamiento propuesto | Política por serie, bloqueo, historia de solo adición, eliminación autorizada y prueba de borrado cuando proceda. |
| Evidencia requerida | CT-CUM-06, tabla de valoración, procedimiento de bloqueo/expurgo, restauración y decisión de Archivo/DPD. |
| Estado | `BLOQUEANTE`; `RESPONSABLE_PENDIENTE`; `NO_ACEPTADO`. |

## R-08 — Automatización opaca, incorrecta o discriminatoria

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una regla, modelo o automatización afecta empleo sin versión, explicación, competencia o revisión humana suficientes. |
| Consecuencia | Trato desigual, perjuicio, decisión no impugnable o efecto jurídico sin autoridad. |
| Dimensiones | Integridad, autenticidad y trazabilidad. |
| Tratamiento propuesto | Catálogos versionados, motivos gobernados, pruebas de reglas, supervisión humana y cierre predeterminado de IA/automatización. |
| Evidencia requerida | CT-CUM-07/09, base y competencia, análisis de sesgo, explicación, revisión humana y procedimiento de reclamación. |
| Estado | `BLOQUEANTE`; `RESIDUAL_NO_CALCULADO`; `NO_ACEPTADO`. |

## R-09 — Acceso transversal o administración no segregada

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una cuenta, perfil o superficie acumula privilegios incompatibles o permite administración sin control independiente. |
| Consecuencia | Uso interno indebido, manipulación no detectada o ampliación de impacto. |
| Dimensiones | Confidencialidad, integridad, autenticidad y trazabilidad. |
| Tratamiento propuesto | Segregación de funciones y superficies, alta garantía, privilegio temporal mínimo, doble control cuando se apruebe y registro reforzado. |
| Evidencia requerida | Matriz CT-CUM-07, inventario de perfiles, revisiones de acceso, sesiones administrativas y pruebas operativas. |
| Estado | `PENDIENTE_VALORACION`; `RESPONSABLE_PENDIENTE`; `NO_ACEPTADO`. |

## R-10 — Derechos o rectificación sin autoridad suficiente

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una solicitud de acceso o rectificación se atiende a una identidad no acreditada o altera historia probatoria. |
| Consecuencia | Divulgación a tercero, pérdida de exactitud o imposibilidad de reconstruir el expediente. |
| Dimensiones | Confidencialidad, integridad, autenticidad y trazabilidad. |
| Tratamiento propuesto | Identidad confiable, ámbito exacto, rectificación de solo adición, recibos y separación entre dato fuente y proyección. |
| Evidencia requerida | Procedimiento aprobado, autoridades, plazos, negativas y pruebas integrales de derechos. |
| Estado | `PENDIENTE_VALORACION`; `EVIDENCIA_NO_APORTADA`; `NO_ACEPTADO`. |

## R-11 — Entrega externa incompatible o no conciliada

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una futura entrega usa esquema, mapeo, procedencia o versión incompatibles, o carece de confirmación conciliable. |
| Consecuencia | Datos incorrectos, destino erróneo, duplicidad o falso éxito de entrega. |
| Dimensiones | Integridad, autenticidad, confidencialidad y trazabilidad. |
| Tratamiento propuesto | Esquema y mapeo gobernados, cobertura exacta, ausente/nulo/vacío distintos, huella, idempotencia, recibo y conciliación. |
| Evidencia requerida | Contrato institucional aprobado, inventario campo–finalidad, adaptador revisado, seguridad del canal y pruebas sin valores reales. |
| Estado | `BLOQUEANTE`; cero envíos autorizados; `NO_ACEPTADO`. |

## R-12 — Incidente no detectado o evidencia insuficiente

| Campo | Contenido candidato |
| --- | --- |
| Origen/evento | Una anomalía no genera señal útil o los registros no permiten clasificar, contener y reconstruir el incidente. |
| Consecuencia | Daño prolongado, notificación tardía, pérdida de custodia o rendición de cuentas incompleta. |
| Dimensiones | Disponibilidad, integridad, confidencialidad y trazabilidad. |
| Tratamiento propuesto | Registro gobernado, sincronización, alertas, preservación, clasificación, escalado y procedimientos de respuesta. |
| Evidencia requerida | Casos de detección, cobertura de eventos, custodia, simulacro, tiempos y revisión posterior. |
| Estado | `PENDIENTE_VALORACION`; `EVIDENCIA_NO_APORTADA`; `NO_ACEPTADO`. |

## Escenarios adicionales que debe valorar la autoridad

| ID | Escenario candidato | Tratamiento propuesto | Estado |
| --- | --- | --- | --- |
| R-13 | Cambio o dependencia de software sin procedencia, revisión o reversión suficientes. | Gobierno de cambios, dependencias, construcción reproducible, análisis y rollback probado. | `PENDIENTE_VALORACION`; `NO_ACEPTADO` |
| R-14 | Copia o restauración incompleta, no auténtica o no disponible dentro del objetivo aprobado. | Política, cifrado, separación, restauraciones verificadas y continuidad. | `PENDIENTE_VALORACION`; `NO_ACEPTADO` |
| R-15 | Configuración, clave o secreto expuestos o sin rotación/custodia adecuada. | Secretos externos, privilegio mínimo, rotación, revocación y evidencia sin material sensible. | `PENDIENTE_VALORACION`; `NO_ACEPTADO` |
| R-16 | Capacidad insuficiente o agotamiento de recursos degrada controles o servicio. | Límites previos, cuotas, cancelación, capacidad, observabilidad y degradación cerrada. | `PENDIENTE_VALORACION`; `NO_ACEPTADO` |
| R-17 | Tercero o dependencia contractual no ofrece garantías, supervisión o salida adecuadas. | Diligencia, cláusulas, subencargos, localización, supervisión y reversibilidad. | `NO_EVALUADO`; no se presume tercero |
| R-18 | Procedimiento o formación insuficientes provocan operación incorrecta. | Procedimientos versionados, formación por rol, ejercicios y revisión. | `PENDIENTE_VALORACION`; `NO_ACEPTADO` |

# Parte C — Plan de tratamiento candidato

No se asignan fechas, personas ni entornos reales. Cada línea requiere una
minitarea, propietario competente y evidencia independiente antes de cambiar
su estado.

| Línea | Objetivo verificable | Entregables requeridos | Dependencias | Estado |
| --- | --- | --- | --- | --- |
| PT-01 Gobierno | Aprobar alcance, roles, políticas y excepciones. | Delimitación, responsables, ciclo de revisión y registro de decisiones. | CT-CUM-04/07/10 | `TRATAMIENTO_PROPUESTO` |
| PT-02 Inventario | Completar información, servicios, dependencias y activos mediante referencias gobernadas. | Inventario aprobado, propietarios, clasificación y relaciones. | CT-CUM-04 | `TRATAMIENTO_PROPUESTO` |
| PT-03 Identidad y acceso | Acreditar identidad, garantía, altas/bajas, privilegio mínimo y segregación. | Matriz de acceso, negativas, revisiones y procedimientos administrativos. | CT-CUM-07 | `TRATAMIENTO_PROPUESTO` |
| PT-04 Datos y privacidad | Demostrar minimización, exactitud, derechos, conservación y protección. | RAT/EIPD aprobados, matriz campo–finalidad, derechos y política de conservación. | CT-CUM-02/03/06 | `TRATAMIENTO_PROPUESTO` |
| PT-05 Arquitectura y comunicaciones | Aprobar fronteras, flujos, cifrado y dependencias. | Arquitectura, amenazas, canales, claves y pruebas sin secretos en Git. | CT-CUM-04 | `TRATAMIENTO_PROPUESTO` |
| PT-06 Explotación segura | Gobernar configuración, cambios, vulnerabilidades y administración. | Baselines, cambios, parches, escaneo, excepciones y reversión. | Infraestructura autorizada | `TRATAMIENTO_PROPUESTO` |
| PT-07 Registro e incidentes | Detectar, clasificar, preservar y responder. | Catálogo de eventos, alertas, playbooks, custodia y simulacro. | CT-CUM-03/10 | `TRATAMIENTO_PROPUESTO` |
| PT-08 Continuidad | Aprobar objetivos, copias, restauración, capacidad y recuperación. | Plan, copias, pruebas de restauración y ejercicios. | Inventario formal | `TRATAMIENTO_PROPUESTO` |
| PT-09 Desarrollo y suministro | Demostrar procedencia, revisión, pruebas y segregación de cambios. | Inventario de dependencias, builds, revisiones, análisis y trazabilidad. | Política aprobada | `TRATAMIENTO_PROPUESTO` |
| PT-10 Operación humana | Preparar procedimientos, formación y supervisión por rol. | Manuales versionados, formación, ejercicios y evidencias. | CT-CUM-07/08 | `TRATAMIENTO_PROPUESTO` |
| PT-11 Auditoría | Verificar alcance, eficacia y remediación con independencia. | Plan, pruebas, hallazgos, responsables y seguimiento. | CT-CUM-10 | `TRATAMIENTO_PROPUESTO` |

## Ficha de ejecución obligatoria

Antes de declarar una línea implantada deberá constar:

1. riesgo y requisito que trata;
2. alcance y versión exactos;
3. propietario y autoridad aprobadora;
4. resultado observable y criterio de aceptación;
5. procedimiento y configuración versionados;
6. pruebas normales, negativas, de fallo y recuperación;
7. evidencia operativa sin datos ni secretos expuestos;
8. revisión independiente y defectos abiertos;
9. fecha de vigencia y próxima revisión;
10. riesgo residual recalculado y decisión formal separada.

# Parte D — Gobierno del riesgo y plan de seguridad

## Decisiones reservadas

| Decisión | Autoridad necesaria | Estado |
| --- | --- | --- |
| Método y escalas | Seguridad y responsables definidos por la política | `PENDIENTE_VALIDACION` |
| Inventario y propietarios | Responsables de información, servicio y sistema | `PENDIENTE_VALIDACION` |
| Probabilidad e impacto | Autoridades competentes con evidencia del contexto | `PENDIENTE_VALORACION` |
| Prioridad y tratamiento | Propietario del riesgo y órganos competentes | `PENDIENTE_VALIDACION` |
| Riesgo residual | Recalculado tras evidencia de eficacia | `RESIDUAL_NO_CALCULADO` |
| Aceptación o excepción | Autoridad formal competente | `NO_ACEPTADO` |
| Paso a preproducción | Autoridad formal tras cerrar CT-CUM-05 y dependencias | `BLOQUEANTE` |

Ningún mantenedor, agente, revisor técnico o commit sustituye estas decisiones.

## Registro de avance futuro

Cada actualización deberá conservar historia de solo adición o una secuencia
versionada que permita reconstruir:

- cambio de contexto o alcance;
- evidencia incorporada o retirada;
- valoración anterior y nueva;
- tratamiento acordado y estado real;
- hallazgo, desviación, excepción y caducidad;
- responsable de proponer, revisar y decidir;
- relación con incidentes, cambios y auditorías.

La eliminación de una incertidumbre exige evidencia positiva. No se cierra por
silencio, antigüedad, imposibilidad de reproducir o ausencia de incidentes.

## Condiciones mínimas antes de preproducción

Todas permanecen bloqueadas en este corte:

1. CT-CUM-02/03/04 integrados y aprobaciones competentes incorporadas.
2. Sistema delimitado, categoría y aplicabilidad formalmente decididas.
3. Inventario de activos y dependencias aprobado fuera de este dossier.
4. Método, escalas, propietarios y apetito de riesgo aprobados.
5. Riesgos valorados, tratamientos implantados y eficacia demostrada.
6. Riesgo residual calculado y decisión competente documentada.
7. Conservación, competencia, accesibilidad e IA resueltas según proceda.
8. Incidentes, continuidad, restauración y operación ensayados.
9. Auditoría independiente y remediación de hallazgos bloqueantes.
10. Autorización formal de preproducción documentada por la autoridad
    competente.

No se habilitan excepciones técnicas a estas condiciones.

## Bloqueos conservados

- CT-CUM-05 permanece abierto y no aprobado.
- CT-CUM-06 a CT-CUM-10 mantienen sus puertas propias.
- O4 y los demás carriles técnicos conservan sus NO-GO y dependencias.
- Datos personales reales, efectos, altas, envíos, comunicaciones, red,
  servicios, preproducción y producción siguen prohibidos.
- El tablero, porcentajes y documentos transversales no cambian por este
  candidato.

## Criterio técnico de revisión del candidato

La revisión independiente solo puede comprobar que el dossier:

1. enlaza los doce escenarios EIPD y añade riesgos transversales sin afirmar
   exhaustividad;
2. deja probabilidad, impacto, inherente y residual sin valorar;
3. marca toda medida como propuesta y toda aceptación como ausente;
4. preserva CT-CUM-05 y preproducción como bloqueos;
5. no incorpora activos, infraestructura, proveedores, secretos o datos
   reales;
6. mantiene separadas evidencia técnica, eficacia y decisión competente;
7. resuelve sus enlaces locales, cita una base Git existente y supera
   `git diff --check`.

Un `GO` de revisión acredita coherencia del candidato, no cierra CT-CUM-05,
no reduce riesgo y no autoriza preproducción.

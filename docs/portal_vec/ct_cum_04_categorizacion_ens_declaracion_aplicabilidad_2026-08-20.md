# CT-CUM-04 — Dossier candidato de categorización ENS y declaración de aplicabilidad

Fecha de corte: 20 de agosto de 2026.

Base documental: `b2f7a7c0d889c110a765aa9d2d70bab7beeeeabb`.

Estado: **PENDIENTE_VALIDACION; no aprobado; sin categoría ENS asignada**.

## Autoridad y efecto del dossier

La
[matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-04` como la categorización ENS y la declaración de
aplicabilidad. Esta puerta bloquea la autorización de infraestructura
productiva.

La misma matriz reserva la categorización a los responsables competentes. El
presente dossier solo prepara una hoja de decisión verificable. No constituye:

- la categorización formal del sistema;
- una declaración de aplicabilidad aprobada;
- un inventario de activos o una topología de infraestructura real;
- un análisis o aceptación de riesgos;
- acreditación de que una medida ENS está implantada;
- autorización para preproducción, producción o datos personales reales.

Mientras falte una decisión formal, el diseño conserva como suelo prudente la
hipótesis de categoría alta indicada por la matriz. Esta hipótesis no equivale
a asignar una categoría y nunca permite declarar una medida como cumplida.

## Fuentes y límites

Este dossier se basa exclusivamente en:

- el
  [inventario candidato CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- el [dossier candidato CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- el [expediente remitido por RRHH](expediente_contratacion_temporal_rrhh.md);
- los [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md);
- la
  [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md);
- el [tablero](tablero_tareas_contratacion_temporal_2026-07-23.md) y el
  [mapa de paralelización](mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md).

No se incorporan nombres de sistemas reales, direcciones, dominios, redes,
cuentas, proveedores, ubicaciones, configuraciones, credenciales ni valores de
personas. Los candidatos técnicos locales citados por CT-CUM-02 continúan sin
integrar y no se consideran capacidades activas.

## Vocabulario de estado

| Estado | Significado en este documento |
| --- | --- |
| `PENDIENTE_VALIDACION` | Requiere decisión o evidencia de la autoridad competente. |
| `DISEÑO_EXIGIDO` | Invariante o control requerido por la arquitectura; no acredita implantación. |
| `EVIDENCIA_PARCIAL` | Existe alguna prueba técnica local, insuficiente para acreditar la medida completa. |
| `NO_EVALUADO` | No existe aún evaluación suficiente para proponer aplicabilidad o cumplimiento. |
| `NO_APLICA_PENDIENTE` | Exención propuesta que carece de motivación y aprobación; se trata como aplicable hasta resolverla. |
| `BLOQUEANTE` | Su ausencia impide avanzar al efecto indicado, sin excepción implícita. |

No se usa el estado `CUMPLIDO` porque este corte no contiene la evidencia
organizativa, operativa y técnica necesaria para sostenerlo.

## Delimitación candidata del sistema

| Elemento de categorización | Descripción mínima candidata | Evidencia o decisión faltante |
| --- | --- | --- |
| Misión | Tramitar necesidades de contratación temporal y coordinar autoridades conservando separación de módulos. | Validación de RRHH, Sistemas, Seguridad y órgano responsable. |
| Información | Expedientes, decisiones, referencias opacas, documentos por referencia, trazabilidad y datos de empleo descritos en CT-CUM-02/03. | Inventario formal de información y propietarios. |
| Servicios | Capacidades de dominio, aplicación y puertos; adaptadores y composición real solo cuando cada corte esté autorizado. | Catálogo formal de servicios y dependencias en explotación. |
| Usuarios | Personal actuante y unidades competentes, sin identidades reales en este dossier. | Censo de perfiles, responsabilidades y segregación aprobado. |
| Dependencias | Identidad, autorización, auditoría, documentos, firma, Bolsa, Personal, presupuesto y sistemas externos mediante contratos. | Inventario de dependencias, criticidad y acuerdos vigentes. |
| Activos | No se enumeran activos reales. | Inventario corporativo con propietario, ubicación, clasificación y ciclo de vida. |
| Fronteras | Interfaces hexagonales y referencias opacas; ninguna topología real se presume. | Diagrama aprobado de arquitectura, zonas, flujos y fronteras de confianza. |
| Ciclo de vida | Diseño y candidatos locales; sin preproducción ni producción autorizadas. | Procedimientos de cambio, operación, retirada y conservación. |

La delimitación permanece `PENDIENTE_VALIDACION`. Una omisión en el inventario
no reduce el alcance ni la categoría: obliga a completar la entrada.

# Parte A — Hoja de categorización

## Reglas de valoración pendientes

Para cada dimensión, la autoridad competente debe documentar al menos:

1. funciones y tratamientos afectados;
2. consecuencia posible de una pérdida o degradación;
3. sujetos, derechos, obligaciones y servicios afectados;
4. dependencia con otras dimensiones;
5. nivel propuesto y justificación trazable;
6. responsable de la valoración, fecha, versión y aprobación;
7. evidencia utilizada y discrepancias abiertas.

La severidad no se deduce de una prueba unitaria, del uso de cifrado, de una
etiqueta del repositorio ni de la ausencia de incidentes observados.

## Disponibilidad

| Aspecto | Estado candidato |
| --- | --- |
| Necesidad | Los plazos administrativos, continuidad del expediente y coordinación con autoridades hacen material la dimensión. |
| Impacto | `PENDIENTE_VALIDACION`; faltan objetivos de recuperación, periodos críticos y consecuencias formalizadas. |
| Evidencia requerida | Catálogo de servicios, dependencias, capacidad, copias, restauración, continuidad y pruebas operativas. |
| Decisión | Sin nivel asignado. Se mantiene la hipótesis prudente sin afirmar disponibilidad efectiva. |

## Autenticidad

| Aspecto | Estado candidato |
| --- | --- |
| Necesidad | Actores, perfiles, fuentes, documentos, decisiones y sistemas externos deben ser auténticos y estar atestados. |
| Impacto | `PENDIENTE_VALIDACION`; una identidad o procedencia falsa puede alterar el expediente o atribuir actuaciones indebidamente. |
| Evidencia requerida | Fronteras de identidad, garantía de autenticación, firma/sello cuando proceda, custodia de claves y validación de procedencia. |
| Decisión | Sin nivel asignado; los identificadores recibidos como datos nunca conceden autoridad. |

## Integridad

| Aspecto | Estado candidato |
| --- | --- |
| Necesidad | Fases, versiones, actuaciones, cálculos, documentos, decisiones y recibos deben permanecer coherentes y trazables. |
| Impacto | `PENDIENTE_VALIDACION`; una alteración puede afectar derechos, presupuesto, motivación o resultado administrativo. |
| Evidencia requerida | Control de versión, historia de solo adición, huellas, firma, transacciones, reconciliación, copias y pruebas de restauración. |
| Decisión | Sin nivel asignado; la existencia de CAS o hashes locales es solo `EVIDENCIA_PARCIAL`. |

## Confidencialidad

| Aspecto | Estado candidato |
| --- | --- |
| Necesidad | El contexto de empleo puede reunir datos personales y administrativos cuya exposición debe minimizarse. |
| Impacto | `PENDIENTE_VALIDACION`; faltan clasificación formal de información, perfiles, destinatarios y escenarios completos. |
| Evidencia requerida | Inventario de datos, capacidades, segregación, cifrado, gestión de claves, exportaciones, borrado y control de soportes. |
| Decisión | Sin nivel asignado; datos reales siguen prohibidos y las categorías especiales requieren tarea propia. |

## Trazabilidad

| Aspecto | Estado candidato |
| --- | --- |
| Necesidad | Debe poder reconstruirse quién hizo qué, con qué autoridad, finalidad, motivo, versión y resultado. |
| Impacto | `PENDIENTE_VALIDACION`; una pérdida de trazabilidad puede impedir rendición de cuentas, revisión o respuesta a incidentes. |
| Evidencia requerida | Auditoría de solo adición, sellado temporal, sincronización horaria, monitorización, custodia y preservación de evidencia. |
| Decisión | Sin nivel asignado; los recibos técnicos locales no acreditan el sistema operativo completo. |

## Resultado de categorización

| Dimensión | Nivel formal | Responsable | Evidencia aprobada | Estado |
| --- | --- | --- | --- | --- |
| Disponibilidad | Sin asignar | `PENDIENTE_VALIDACION` | No aportada | `BLOQUEANTE` |
| Autenticidad | Sin asignar | `PENDIENTE_VALIDACION` | No aportada | `BLOQUEANTE` |
| Integridad | Sin asignar | `PENDIENTE_VALIDACION` | No aportada | `BLOQUEANTE` |
| Confidencialidad | Sin asignar | `PENDIENTE_VALIDACION` | No aportada | `BLOQUEANTE` |
| Trazabilidad | Sin asignar | `PENDIENTE_VALIDACION` | No aportada | `BLOQUEANTE` |

Categoría resultante: **NO DETERMINADA**.

# Parte B — Declaración candidata de aplicabilidad

## Criterio de inclusión y exclusión

Cada familia se presume aplicable a efectos de diseño mientras no exista una
exclusión motivada y aprobada. Una exclusión debe indicar requisito, activo,
riesgo, medida compensatoria, responsable, vigencia y evidencia. El silencio,
la falta de componente o la externalización futura no significan `NO APLICA`.

| Familia de medidas | Aplicabilidad candidata | Evidencia mínima exigida | Situación del corte |
| --- | --- | --- | --- |
| Política y gobierno de seguridad | Aplicable | Política aprobada, roles, responsables, revisión y excepciones. | `NO_EVALUADO` |
| Inventario y clasificación | Aplicable | Activos, servicios, información, propietarios, dependencias y clasificación. | `EVIDENCIA_PARCIAL`: CT-CUM-02/03 solo describen tratamientos candidatos. |
| Gestión de riesgos | Aplicable | Método, amenazas, probabilidad, impacto, riesgo residual, tratamiento y aceptación. | `BLOQUEANTE`: corresponde a CT-CUM-05. |
| Identidad y control de acceso | Aplicable | Altas/bajas, garantía, MFA cuando proceda, privilegio mínimo, segregación y revisiones. | `DISEÑO_EXIGIDO`; implantación no acreditada. |
| Protección de la información | Aplicable | Minimización, clasificación, cifrado, claves, exportación, soportes y borrado. | `DISEÑO_EXIGIDO`; operación no acreditada. |
| Protección de comunicaciones | Aplicable | Flujos autorizados, autenticación mutua cuando proceda, cifrado y gestión de certificados. | `NO_EVALUADO`; no se inventaría red real. |
| Seguridad de explotación | Aplicable | Configuración, endurecimiento, parches, vulnerabilidades, cambios y segregación administrativa. | `NO_EVALUADO` |
| Registro, monitorización e incidentes | Aplicable | Eventos gobernados, sincronización, alertas, custodia, respuesta y simulacros. | `DISEÑO_EXIGIDO`; evidencia operativa ausente. |
| Continuidad, copias y restauración | Aplicable | Objetivos aprobados, copias, restauraciones, capacidad y pruebas de continuidad. | `NO_EVALUADO` |
| Desarrollo y cadena de suministro | Aplicable | Revisión, dependencias, procedencia, construcción, secretos, segregación y trazabilidad de cambios. | `EVIDENCIA_PARCIAL`; repositorio local no acredita operación completa. |
| Servicios y terceros | Aplicable si existen | Contrato, responsabilidades, garantías, ubicación, subencargos, supervisión y salida. | `NO_EVALUADO`; no se presume proveedor. |
| Protección de instalaciones y equipos | Aplicable a los activos que se autoricen | Inventario, control físico, mantenimiento, retirada y soportes. | `NO_EVALUADO`; activos reales fuera de alcance. |
| Formación y procedimientos | Aplicable | Plan por rol, procedimientos vigentes, ejercicios y evidencia de capacitación. | `NO_EVALUADO` |
| Auditoría y mejora | Aplicable | Plan, alcance, independencia, hallazgos, remediación y seguimiento. | `NO_EVALUADO` |

## Ficha obligatoria por medida

La declaración aprobable deberá sustituir la tabla de familias por fichas
versionadas que contengan:

| Campo | Requisito |
| --- | --- |
| Identificador y versión | Referencia inequívoca a la medida y al marco aplicable. |
| Aplicabilidad | Aplicable o exclusión motivada; nunca inferida por ausencia. |
| Alcance | Información, servicio, activo, entorno y fase cubiertos. |
| Responsable | Propietario de implantar, operar, revisar y aceptar evidencia. |
| Implementación | Configuración y procedimiento aprobados, sin secretos en el expediente. |
| Evidencia | Prueba técnica, registro operativo, revisión y fecha de vigencia. |
| Dependencias | Medidas, servicios o decisiones necesarias para que sea efectiva. |
| Excepciones | Riesgo, compensación, caducidad y autoridad aprobadora. |
| Resultado | Conforme, no conforme o no evaluado, con hallazgos trazables. |

# Parte C — Gobierno, evidencia y cierre

## Responsabilidades pendientes

| Decisión | Autoridad requerida | Estado |
| --- | --- | --- |
| Delimitación del sistema | Responsables de información, servicio y sistema | `PENDIENTE_VALIDACION` |
| Nivel de cada dimensión | Responsables competentes con Seguridad y negocio | `PENDIENTE_VALIDACION` |
| Categoría resultante | Órgano competente | `PENDIENTE_VALIDACION` |
| Declaración de aplicabilidad | Responsable del sistema y autoridades definidas por la política | `PENDIENTE_VALIDACION` |
| Riesgos y tratamiento | CT-CUM-05 y autoridad aceptante | `PENDIENTE_VALIDACION` |
| Auditoría y remediación | Función independiente competente | `PENDIENTE_VALIDACION` |
| Autorización de infraestructura productiva | Autoridad formal, solo tras cerrar dependencias | `BLOQUEANTE` |

El repositorio no sustituye firmas, resoluciones, designaciones ni registros
corporativos. Las personas y unidades concretas no se inventan en este corte.

## Cadena de evidencia exigida

Una afirmación futura de cumplimiento debe enlazar, como mínimo:

1. requisito y versión normativa;
2. medida aplicable y alcance exacto;
3. responsable y aprobación;
4. activo o servicio inventariado mediante referencia gobernada;
5. configuración o procedimiento versionado;
6. prueba técnica y registro operativo;
7. hallazgos, excepciones y remediación;
8. vigencia y próxima revisión.

Una prueba local solo acredita el predicado que observa. No acredita por sí
misma una familia, una categoría, la declaración completa ni la autorización
de un entorno.

## Dependencias y bloqueos conservados

1. CT-CUM-02 y CT-CUM-03 siguen siendo candidatos locales hasta que dirección
   los integre y actualice el estado transversal.
2. CT-CUM-04 permanece abierto hasta recibir delimitación, niveles, categoría,
   aplicabilidad, responsables, evidencia y aprobación formales.
3. CT-CUM-05 debe completar análisis, tratamiento y aceptación de riesgos.
4. CT-CUM-06 a CT-CUM-10 conservan sus propias puertas de documento,
   competencia, accesibilidad, IA y operación.
5. No se autorizan datos personales reales, red, servicios, preproducción,
   producción ni comunicaciones por la existencia de este dossier.
6. Los bloqueos técnicos de O4 y de otros carriles no se modifican.

## Criterio de cierre documental de este candidato

El candidato es revisable cuando:

1. mantiene el estado `PENDIENTE_VALIDACION` y la categoría sin asignar;
2. contiene las cinco dimensiones ENS sin nivel inventado;
3. diferencia diseño, evidencia parcial, no evaluado y decisión formal;
4. trata las exclusiones no aprobadas como aplicables o bloqueantes;
5. conserva CT-CUM-05 y la autorización de infraestructura como bloqueos;
6. no contiene activos, ubicaciones, proveedores, credenciales o datos reales;
7. resuelve sus enlaces locales, cita una base Git existente y supera
   `git diff --check`.

Superar estos puntos permite revisar el dossier, no cerrar CT-CUM-04 ni
autorizar infraestructura.

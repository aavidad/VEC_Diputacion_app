# CT-CUM-09 — Clasificación y gobierno candidatos de casos de uso de IA

Fecha del corte: 2026-08-20

Base exacta: `e5477b25461747bf5758f6c543d4e1cf0e279754`

Estado del dossier: `CANDIDATO_LOCAL_NO_APROBADO`

Estado de la capacidad: `IA_CERRADA`

## 1. Objeto y límite

Este dossier prepara la clasificación y el gobierno que exigiría cualquier
caso de uso futuro de inteligencia artificial relacionado con contratación
temporal. No selecciona tecnología, proveedor, modelo, datos, infraestructura
ni responsable. Tampoco autoriza una prueba, tratamiento, recomendación,
decisión, comunicación o efecto.

La capability de este corte es exclusivamente documental: inventariar clases
de uso candidatas, separar las meramente informativas de las que podrían
afectar a personas y definir la evidencia mínima que tendría que evaluar la
autoridad competente.

La invariante es:

> Todo caso permanece `IA_CERRADA` mientras no consten clasificación formal,
> finalidad y autoridad, obligaciones aplicables, evaluaciones requeridas,
> supervisión humana efectiva y aprobación expresa del órgano competente.

La ausencia, indisponibilidad, incertidumbre o confianza estadística nunca se
interpreta como autorización, idoneidad, puntuación, prioridad o éxito.

## 2. Autoridades y antecedentes

Este candidato se apoya en las siguientes fuentes locales, sin sustituirlas:

- [matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md), en especial su bloque sobre Reglamento de IA y la tarea CT-CUM-09;
- [inventario candidato de tratamientos](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- [RAT y EIPD candidatos](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- [categorización ENS y declaración de aplicabilidad candidatas](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- [análisis y tratamiento de riesgos candidato](ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md);
- [política ENI candidata](ct_cum_06_politica_eni_documento_expediente_firma_conservacion_2026-08-20.md);
- [matriz candidata de competencia y actuaciones](ct_cum_07_matriz_competencia_actuaciones_humanas_automatizadas_2026-08-20.md);
- [evaluación de accesibilidad candidata](ct_cum_08_evaluacion_accesibilidad_declaracion_candidata_2026-08-20.md).

Todos esos documentos conservan su estado candidato. Sus referencias sirven
para trazabilidad, no acreditan implantación, conformidad ni autorización.

## 3. Vocabulario de estado

| Estado | Significado cerrado |
| --- | --- |
| `SIN_CASO_ACTIVO` | No existe capacidad de IA habilitada en este alcance. |
| `PENDIENTE_CLASIFICACION` | Faltan hechos y resolución competentes para clasificar el caso. |
| `INFORMATIVO_PUBLICO_CANDIDATO` | Solo describe un posible uso sobre corpus público gobernado; no lo habilita. |
| `POTENCIAL_ALTO_RIESGO` | Puede incidir en empleo, selección, evaluación, promoción, asignación o supervisión; queda cerrado. |
| `USO_NO_ADMITIDO` | El propósito o funcionamiento propuesto no puede continuar en este alcance. |
| `EVIDENCIA_NO_APORTADA` | No existe evidencia suficiente para afirmar el requisito. |
| `IA_CERRADA` | No se ejecuta modelo, inferencia, entrenamiento, evaluación ni integración. |
| `BLOQUEANTE` | Impide cualquier avance técnico o tratamiento hasta resolución expresa. |

Ninguna etiqueta de esta tabla constituye una clasificación jurídica final.
La clasificación definitiva y sus consecuencias corresponden a los órganos y
perfiles competentes identificados fuera de este dossier.

## 4. Frontera: reglas deterministas frente a IA

Una regla versionada, una plantilla, una búsqueda literal o una transformación
determinista no se considerarán IA solo por automatizar una operación. Esa
conclusión deberá demostrarse con arquitectura, dependencias, entradas,
salidas y versión; el nombre comercial o la etiqueta del componente no bastan.

Si una pieza aprende, infiere, perfila, clasifica, genera o recomienda mediante
un modelo o técnica que requiera evaluación bajo la normativa aplicable, pasa
al registro CT-CUM-09 y queda `IA_CERRADA` hasta clasificación competente.

La capa de presentación, API, CLI, MCP o integración externa no altera esta
frontera ni amplía permisos. Cambiar el canal tampoco convierte una propuesta
en decisión ni una salida probabilística en hecho acreditado.

## 5. Inventario de clases candidatas

El inventario no describe sistemas existentes. Solo enumera clases previsibles
para impedir que aparezcan sin gobierno.

| ID | Clase candidata | Entradas admisibles en este dossier | Salida o finalidad hipotética | Clasificación provisional | Estado |
| --- | --- | --- | --- | --- | --- |
| IA-01 | Consulta informativa sobre corpus público gobernado | Ninguna entrada personal o interna; solo referencia abstracta a corpus público | Explicación general con procedencia | `INFORMATIVO_PUBLICO_CANDIDATO` | `IA_CERRADA` |
| IA-02 | Búsqueda o resumen de información interna | Ningún documento ni dato se incorpora aquí | Ayuda interna no decisoria | `PENDIENTE_CLASIFICACION` | `IA_CERRADA` |
| IA-03 | Redacción asistida de informes o propuestas | Ningún expediente ni antecedente real | Borrador sujeto a autoría y revisión | `PENDIENTE_CLASIFICACION` | `IA_CERRADA` |
| IA-04 | Ordenación, priorización o recomendación de candidaturas | Prohibida cualquier candidatura real | Recomendación que podría incidir en selección | `POTENCIAL_ALTO_RIESGO` | `IA_CERRADA` |
| IA-05 | Baremación, evaluación o predicción de idoneidad | Prohibidos méritos, perfiles y atributos reales | Puntuación o evaluación individual | `POTENCIAL_ALTO_RIESGO` | `IA_CERRADA` |
| IA-06 | Propuesta de cobertura, puesto, tarea o llamamiento | Prohibidos expedientes, puestos y personas reales | Asignación o recomendación laboral | `POTENCIAL_ALTO_RIESGO` | `IA_CERRADA` |
| IA-07 | Seguimiento, perfilado o supervisión laboral | Prohibidos actividad, rendimiento y conducta reales | Alerta, perfil o medida sobre una persona | `POTENCIAL_ALTO_RIESGO` | `IA_CERRADA` |
| IA-08 | Detección de anomalías o riesgo con efecto individual | Ninguna señal, documento o evento real | Alerta que podría afectar a un procedimiento | `PENDIENTE_CLASIFICACION` | `IA_CERRADA` |
| IA-09 | Enriquecimiento o correspondencia de datos GINPIX | Solo se reconoce el modelo determinista sintético ya separado; no se incorpora ningún dato | Correspondencia o propuesta de campo | `PENDIENTE_CLASIFICACION` | `IA_CERRADA` |
| IA-10 | Personalización de ayuda o accesibilidad | Ninguna identidad, preferencia ni interacción real | Adaptación de ayuda no decisoria | `PENDIENTE_CLASIFICACION` | `IA_CERRADA` |

No se aceptan casos implícitos. Una finalidad, entrada, población afectada,
salida o integración distinta exige un identificador y una evaluación nuevos.

## 6. Ficha mínima de clasificación por caso

Antes de proponer una implementación, cada caso deberá documentar, como
`EVIDENCIA_NO_APORTADA` hasta su validación:

1. identificador, versión y propietario funcional;
2. finalidad prevista y usos expresamente excluidos;
3. contexto de uso, personas y derechos potencialmente afectados;
4. entradas, fuentes, procedencia, calidad y categorías de datos;
5. salidas, destinatarios y decisiones o efectos que podrían seguirlas;
6. técnica, componentes, dependencias y cambios de versión relevantes;
7. papeles de proveedor, responsable del despliegue y demás intervinientes;
8. clasificación legal propuesta y fundamento verificable;
9. encaje con protección de datos, igualdad y procedimiento administrativo;
10. riesgos, límites, incertidumbres y condiciones de fallo;
11. supervisión humana, autoridad, competencia y capacidad real de revocación;
12. transparencia, explicabilidad e instrucciones para cada perfil;
13. exactitud, robustez, ciberseguridad y accesibilidad verificables;
14. registro técnico, auditoría, incidentes, cambios y retirada;
15. evaluaciones, consultas, registro y aprobaciones requeridas.

Una ficha incompleta no avanza por defecto. Tampoco se reutiliza la aprobación
de un caso para otra versión, finalidad, población, dato, modelo o canal.

## 7. Puertas candidatas de gobierno

| ID | Puerta | Evidencia requerida | Estado de este dossier |
| --- | --- | --- | --- |
| GOV-01 | Clasificación y roles | Resolución del caso y de los papeles aplicables | `EVIDENCIA_NO_APORTADA` |
| GOV-02 | Sistema de riesgos y calidad | Procedimiento, responsables, criterios y registros | `EVIDENCIA_NO_APORTADA` |
| GOV-03 | Gobierno de datos | Finalidad, minimización, calidad, representatividad, sesgos y trazabilidad | `EVIDENCIA_NO_APORTADA` |
| GOV-04 | Documentación y registro técnico | Versiones, configuración, límites, entradas, salidas y logs gobernados | `EVIDENCIA_NO_APORTADA` |
| GOV-05 | Transparencia e instrucciones | Información comprensible, límites y canal de impugnación o ayuda | `EVIDENCIA_NO_APORTADA` |
| GOV-06 | Supervisión humana efectiva | Competencia, tiempo, información, independencia, parada y reversión | `EVIDENCIA_NO_APORTADA` |
| GOV-07 | Exactitud, robustez y ciberseguridad | Umbrales aprobados, pruebas, abuso, deriva, indisponibilidad y recuperación | `EVIDENCIA_NO_APORTADA` |
| GOV-08 | Evaluaciones y derechos fundamentales | EIPD y evaluación de impacto cuando correspondan, con consultas y firmas | `EVIDENCIA_NO_APORTADA` |
| GOV-09 | Registro, seguimiento e incidentes | Inventario, métricas aprobadas, alertas, notificación, retirada y conservación | `EVIDENCIA_NO_APORTADA` |
| GOV-10 | Contratación y ciclo de vida | Condiciones del proveedor, portabilidad, cambios, auditoría y cese | `EVIDENCIA_NO_APORTADA` |

Todas las puertas son acumulativas cuando resulten aplicables. La aprobación de
una no compensa la ausencia o el rechazo de otra.

## 8. Supervisión humana no nominal

La revisión humana solo será efectiva si la persona revisora:

- tiene competencia y autoridad para rechazar o detener la actuación;
- dispone de tiempo, contexto, evidencia y formación suficientes;
- conoce límites, incertidumbre, procedencia y versión de la salida;
- no recibe una interfaz que induzca confirmación automática;
- puede pedir rectificación, explicación o revisión por otro cauce;
- deja motivación propia y trazabilidad de la decisión;
- no convierte el resultado del sistema en presunción favorable o adversa.

Una mera confirmación, firma mecánica o posibilidad teórica de intervenir no
cumple esta invariante. CT-CUM-07 mantiene la autoridad sobre competencias y
actuaciones; CT-CUM-09 no las redefine.

## 9. Caso informativo público candidato

IA-01 tampoco queda habilitado. Para considerarlo en un corte posterior deberá
probar, además de las puertas aplicables, que:

- usa exclusivamente corpus público, versionado, aprobado y con procedencia;
- no accede a expedientes, candidaturas, perfiles, puestos ni datos internos;
- no identifica, perfila ni personaliza respuestas sobre personas;
- no decide, recomienda, barema, prioriza ni interpreta un caso particular;
- distingue información, cita y límite, y falla cerrado ante incertidumbre;
- no usa la sesión o el navegador como autoridad de identidad o permiso;
- ofrece alternativa accesible y un canal humano gobernado;
- registra cambios e incidentes sin conservar consultas personales indebidas.

La etiqueta “bot informativo” no basta: cualquier desviación remite el caso a
una nueva clasificación y mantiene `IA_CERRADA`.

## 10. Datos, pruebas y proveedores

Este dossier contiene cero datos personales, expedientes, candidaturas,
puestos, historiales, prompts, respuestas de modelos o corpus reales. Tampoco
incluye claves, tokens, endpoints, contratos, marcas, proveedores, modelos,
datasets, métricas o infraestructura concreta.

Una futura prueba, si llegara a autorizarse, deberá usar un corte separado,
datos sintéticos aprobados, referencias opacas y criterios previos. No podrá
usar reintentos, selección de resultados favorables ni tolerancias que oculten
un fallo. Ninguna salida de prueba podrá producir un efecto funcional.

## 11. Cambio, incidente y retirada

El gobierno futuro deberá tratar como cambio material, al menos, cualquier
modificación de finalidad, datos, población, salida, integración, modelo,
proveedor, versión, configuración, umbral, supervisión o canal. El cambio
reabre la clasificación y las evaluaciones correspondientes antes de uso.

La indisponibilidad o anomalía obliga a detener el uso; nunca permite una ruta
menos controlada. Deben existir criterios de retirada, conservación gobernada
de evidencia, investigación, notificación y retorno a un proceso humano sin
IA. Este dossier no declara implantado ninguno de esos controles.

## 12. Bloqueos y decisiones pendientes

Permanecen `BLOQUEANTE`:

- clasificación competente de cada caso y de los roles aplicables;
- determinación de usos no admitidos y de potencial alto riesgo;
- sistema de riesgos y calidad, gobierno de datos y documentación técnica;
- supervisión humana efectiva y competencia de quienes intervengan;
- EIPD, evaluación de derechos fundamentales y consultas cuando procedan;
- exactitud, robustez, ciberseguridad, accesibilidad y gestión de incidentes;
- contratación, registro, seguimiento, cambios y retirada;
- aprobaciones de Protección de Datos, Seguridad, Sistemas, Jurídico, RRHH y
  demás órganos que correspondan.

No se acepta riesgo, excepción ni medida compensatoria. No se afirma que un
control esté implantado. CT-CUM-09 permanece abierto hasta resolución formal y
CT-CUM-10 conserva su auditoría conjunta. Producción y cualquier IA real
continúan bloqueadas.

## 13. Criterios de revisión de este candidato

La revisión independiente deberá comprobar:

1. base exacta y alta de un único Markdown;
2. trazabilidad de autoridades y enlaces locales;
3. diez clases IA identificadas, ninguna habilitada;
4. diez puertas de gobierno en `EVIDENCIA_NO_APORTADA`;
5. separación explícita entre automatización determinista e IA;
6. cierre de selección, evaluación, asignación y supervisión laboral;
7. supervisión humana efectiva sin confirmación nominal;
8. ausencia de datos, proveedor, modelo, infraestructura y decisión reales;
9. ausencia de aprobación, aceptación de riesgo o control implantado;
10. mantenimiento de CT-CUM-09, CT-CUM-10 y producción como bloqueos.

Este dossier solo puede recibir un GO documental acotado a su coherencia. Ese
GO no cambia el tablero, no habilita una implementación y no acredita
cumplimiento, conformidad ni autorización de uso.

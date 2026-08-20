# CT-CUM-10 — Expediente candidato para auditoría conjunta y actas pendientes

Fecha del corte: 2026-08-20

Base exacta: `ef80ddce760d0712f5be5a2af97ae553ec43ef74`

Estado del expediente: `CANDIDATO_LOCAL_NO_APROBADO`

Estado de la auditoría: `AUDITORIA_NO_REALIZADA`

Estado de producción: `PRODUCCION_BLOQUEADA`

## 1. Objeto y límite de autoridad

La matriz normativa define CT-CUM-10 como la auditoría conjunta y las actas de
Protección de Datos, Seguridad, Sistemas, Archivo, Jurídico y RRHH. Este
expediente no realiza esa auditoría ni emite esas actas. Prepara un paquete
común para que las seis autoridades puedan evaluar el mismo alcance, registrar
hallazgos propios y resolver sin que una revisión técnica suplante su criterio.

La capability de este corte es únicamente documental: ordenar entradas,
preguntas, evidencias esperadas, decisiones pendientes y un formato de acta.
No convoca personas, asigna responsabilidades, acepta riesgos, aprueba medidas,
autoriza datos, declara cumplimiento ni habilita un entorno.

La invariante es:

> Mientras falte una sola acta competente requerida, exista un hallazgo
> bloqueante o no haya evidencia positiva de cierre, CT-CUM-10 permanece
> `AUDITORIA_NO_REALIZADA` y producción permanece `PRODUCCION_BLOQUEADA`.

Silencio, ausencia de respuesta, indisponibilidad, revisión de código o GO
documental no equivalen a conformidad ni aprobación.

## 2. Autoridades y fuentes candidatas

El expediente se traza a las siguientes fuentes locales:

- [matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md);
- [inventario candidato de tratamientos CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- [RAT y EIPD candidatos CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- [categorización ENS y aplicabilidad candidatas CT-CUM-04](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- [análisis y tratamiento de riesgos candidato CT-CUM-05](ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md);
- [política ENI candidata CT-CUM-06](ct_cum_06_politica_eni_documento_expediente_firma_conservacion_2026-08-20.md);
- [matriz de competencia candidata CT-CUM-07](ct_cum_07_matriz_competencia_actuaciones_humanas_automatizadas_2026-08-20.md);
- [evaluación de accesibilidad candidata CT-CUM-08](ct_cum_08_evaluacion_accesibilidad_declaracion_candidata_2026-08-20.md);
- [clasificación y gobierno candidatos de IA CT-CUM-09](ct_cum_09_clasificacion_gobierno_casos_uso_ia_2026-08-20.md).

Todos esos documentos conservan carácter candidato. Sus revisiones técnicas
independientes acreditan coherencia de cada corte, no aprobación material por
las seis autoridades de CT-CUM-10.

## 3. Recibo técnico de antecedentes locales

| Nodo | Candidato | Revisión independiente local | Uso permitido en CT-CUM-10 |
| --- | --- | --- | --- |
| CT-CUM-02 | `475042ba57d63b6abf1810fd26cc65c975496029` | `03f36f22bd45bd8b075650ee21cd4b54fba06fbc` | Entrada candidata; no RAT aprobado. |
| CT-CUM-03 | `b2f7a7c0d889c110a765aa9d2d70bab7beeeeabb` | `80dff859674fcac02ee82d131fa19b0344df8ce7` | Entrada candidata; no dictamen del DPD. |
| CT-CUM-04 | `946256e1682c33327918f2be15df1e9899b331d5` | `32eca7b4b4cfd4014c9732fc01bd23e7f1a99e40` | Entrada candidata; sin categoría ENS aprobada. |
| CT-CUM-05 | `002805e3b7c01fd13c85fafd14574144b115ea48` | `d8ffeaa7f98629b9b44067cfc2385204bbaaa203` | Entrada candidata; riesgos no valorados ni aceptados. |
| CT-CUM-06 | `74d20d309d4cf4030f8fd15e45caac2c9ab155cd` | `14a2c3f4bd8000d7247af2ce5397864abd5c0f9a` | Entrada candidata; política ENI no aprobada. |
| CT-CUM-07 | `0fdc31d19865ddb2183eeda958aff5b2d3c826dd` | `57f583c14e2a4e1b1b10de5b4c2bfaea463d46a5` | Entrada candidata; competencia no validada. |
| CT-CUM-08 | `e5477b25461747bf5758f6c543d4e1cf0e279754` | `a298c76b03b124bf7f237cc8179ae8b7c8389629` | Entrada candidata; evaluación no realizada. |
| CT-CUM-09 | `ef80ddce760d0712f5be5a2af97ae553ec43ef74` | `50685261c567d3cd51b093af537150b9f0897c61` | Entrada candidata; toda IA sigue cerrada. |

Que un objeto Git exista no demuestra integración, publicación, vigencia,
firma ni decisión de un órgano. La auditoría deberá fijar su propia línea base
autorizada y comprobar que el material evaluado coincide con ella.

## 4. Vocabulario común de auditoría

| Estado | Significado |
| --- | --- |
| `NO_EVALUADO` | La autoridad competente no ha emitido evaluación. |
| `EVIDENCIA_NO_APORTADA` | Falta evidencia positiva y verificable. |
| `HALLAZGO_ABIERTO` | Existe una desviación o incertidumbre pendiente. |
| `BLOQUEANTE` | Impide continuar a producción. |
| `REMEDIACION_PROPUESTA` | Hay una posible acción, aún no aprobada ni ejecutada. |
| `CIERRE_PENDIENTE_VERIFICACION` | Se afirma una corrección, pero falta verificación independiente. |
| `ACTA_PENDIENTE` | La autoridad no ha emitido y firmado su acta. |
| `NO_APROBADO` | No existe decisión positiva válida. |

No se usará `CONFORME`, `ACEPTADO`, `IMPLANTADO`, `EFICAZ` o `APROBADO` sin
autoridad identificada, alcance exacto, evidencia, fecha, firma y condiciones.

## 5. Alcance común que debe fijar la auditoría

Antes de evaluar, las seis autoridades deberán acordar y sellar:

1. sistema, módulos, capacidades y exclusiones exactos;
2. commit, árbol, artefactos y configuración que componen la línea base;
3. entornos incluidos y confirmación de que no se usaron datos reales antes de
   las puertas correspondientes;
4. tratamientos, categorías de datos, finalidades, fuentes y destinatarios;
5. responsables funcionales, técnicos, de información, servicio y sistema;
6. integraciones, proveedores, encargados y transferencias, si existieran;
7. actos, documentos, decisiones y efectos que se pretenden habilitar;
8. amenazas, riesgos, medidas, pruebas y excepciones que se someten a examen;
9. periodo de evidencia y reglas para cambios durante la auditoría;
10. criterio común de bloqueo, remediación, reauditoría y conservación.

En este expediente todos esos elementos permanecen `NO_EVALUADO` o
`EVIDENCIA_NO_APORTADA`. No se incorporan inventarios, personas, activos,
credenciales, datos, servicios ni rutas reales.

## 6. Cuaderno de Protección de Datos y DPD

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| DPD-01 | ¿Responsable, DPD, finalidades y bases están formalmente validados? | RAT aprobado, designaciones y dictamen | `ACTA_PENDIENTE` |
| DPD-02 | ¿Campos, fuentes, interesados, destinatarios y conservación son necesarios y proporcionales? | Matriz campo–finalidad–base y decisión | `ACTA_PENDIENTE` |
| DPD-03 | ¿La EIPD está concluida y las consultas requeridas están resueltas? | EIPD firmada, riesgo residual y consultas | `ACTA_PENDIENTE` |
| DPD-04 | ¿Derechos, transparencia, exactitud, rectificación y oposición son ejercitables? | Procedimientos y pruebas autorizadas | `ACTA_PENDIENTE` |
| DPD-05 | ¿Encargados, transferencias, incidentes y notificaciones están gobernados? | Acuerdos, registros y simulacros | `ACTA_PENDIENTE` |

No se aporta dictamen del DPD ni se presume resultado favorable.

## 7. Cuaderno de Seguridad

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| SEG-01 | ¿Categoría, aplicabilidad y medidas ENS están aprobadas? | Resolución, SoA y justificaciones | `ACTA_PENDIENTE` |
| SEG-02 | ¿Amenazas y riesgos están valorados con contexto real autorizado? | Método, activos, probabilidad, impacto y propietarios | `ACTA_PENDIENTE` |
| SEG-03 | ¿Tratamientos están implantados y su eficacia demostrada? | Pruebas, hallazgos, remediaciones y revalidación | `ACTA_PENDIENTE` |
| SEG-04 | ¿Identidad, permisos, secretos, cifrado y auditoría fallan cerrado? | Diseño, configuración y pruebas independientes | `ACTA_PENDIENTE` |
| SEG-05 | ¿Incidentes, continuidad, restauración y cadena de custodia están ensayados? | Procedimientos, ejercicios y recibos | `ACTA_PENDIENTE` |

No se declara control implantado, riesgo residual calculado ni riesgo aceptado.

## 8. Cuaderno de Sistemas

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| SIS-01 | ¿Arquitectura, dependencias, configuración y suministro están inventariados? | Línea base reproducible y procedencia | `ACTA_PENDIENTE` |
| SIS-02 | ¿Capacidad, disponibilidad, copias, restauración y reversión son suficientes? | Objetivos aprobados y pruebas | `ACTA_PENDIENTE` |
| SIS-03 | ¿Segregación de entornos, cambios y despliegues impide efectos no autorizados? | Flujo, permisos, evidencias y rollback | `ACTA_PENDIENTE` |
| SIS-04 | ¿Observabilidad, soporte, vulnerabilidades y obsolescencia están gobernados? | Inventarios, alertas y procedimientos | `ACTA_PENDIENTE` |
| SIS-05 | ¿Integraciones externas preservan contratos, minimización y fallo cerrado? | Contratos, pruebas y responsables | `ACTA_PENDIENTE` |

No se identifica ni autoriza infraestructura concreta en este documento.

## 9. Cuaderno de Archivo

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| ARC-01 | ¿Series y expedientes están identificados por la autoridad archivística? | Cuadro, series y resolución | `ACTA_PENDIENTE` |
| ARC-02 | ¿Metadatos, integridad, firma, sello, CSV y copias auténticas son suficientes? | Política aprobada y pruebas | `ACTA_PENDIENTE` |
| ARC-03 | ¿Conservación, bloqueo, transferencia y expurgo tienen plazos y autoridad? | Tabla de valoración y procedimientos | `ACTA_PENDIENTE` |
| ARC-04 | ¿Formatos, migración, accesibilidad y preservación mantienen evidencia? | Plan y ensayos de preservación | `ACTA_PENDIENTE` |
| ARC-05 | ¿La historia evita reescritura y mantiene trazabilidad probatoria? | Modelo, auditoría y pruebas | `ACTA_PENDIENTE` |

No se fija serie, plazo, formato definitivo ni decisión de expurgo.

## 10. Cuaderno Jurídico

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| JUR-01 | ¿Competencia, delegaciones, bases y procedimiento están acreditados? | Normas, resoluciones y matriz de competencia | `ACTA_PENDIENTE` |
| JUR-02 | ¿Actuaciones automatizadas y humanas conservan motivación y revisión? | Procedimiento aprobado y pruebas | `ACTA_PENDIENTE` |
| JUR-03 | ¿Notificación, audiencia, recursos, firma y evidencia tienen validez? | Modelos aprobados y circuito probatorio | `ACTA_PENDIENTE` |
| JUR-04 | ¿Contratación, propiedad, licencias y terceros están regularizados? | Expediente contractual y dictámenes | `ACTA_PENDIENTE` |
| JUR-05 | ¿Accesibilidad e IA cumplen sus obligaciones antes de habilitarse? | Resoluciones CT-CUM-08/09 y evidencias | `ACTA_PENDIENTE` |

Este expediente no emite interpretación jurídica ni resolución administrativa.

## 11. Cuaderno de RRHH

| ID | Pregunta de auditoría | Evidencia esperada | Estado |
| --- | --- | --- | --- |
| RRH-01 | ¿El procedimiento y cada finalidad reflejan la práctica autorizada? | Procedimiento aprobado y responsables | `ACTA_PENDIENTE` |
| RRH-02 | ¿Campos, fuentes, catálogos y reglas son exactos y suficientes? | Catálogos versionados y validación funcional | `ACTA_PENDIENTE` |
| RRH-03 | ¿Selección, cobertura, Bolsa y Personal preservan igualdad y competencia? | Reglas publicadas y pruebas de límites | `ACTA_PENDIENTE` |
| RRH-04 | ¿Revisión humana, rectificación y atención de incidencias son operables? | Procedimientos, formación y ejercicios | `ACTA_PENDIENTE` |
| RRH-05 | ¿GINPIX y otras entregas tienen esquema, autoridad y finalidad aprobados? | Contratos versionados y validación del receptor | `ACTA_PENDIENTE` |

No se valida dato, catálogo, regla, puesto, candidatura o envío real.

## 12. Registro conjunto de hallazgos

Cada hallazgo futuro deberá conservar:

- identificador único y autoridad que lo emite;
- línea base, evidencia, requisito y alcance afectados;
- clasificación y fundamento;
- impacto, riesgo y dependencias sin datos personales innecesarios;
- responsable y fecha solo cuando hayan sido formalmente asignados;
- remediación propuesta, criterio de aceptación y pruebas requeridas;
- evidencia de corrección y revisión independiente;
- decisión de cierre, condiciones, firma y trazabilidad.

Un hallazgo no se cierra por antigüedad, silencio, imposibilidad de reproducir,
resultado verde posterior o cambio de redacción. Si falta evidencia de cierre,
continúa `HALLAZGO_ABIERTO`.

## 13. Formato mínimo de cada acta competente

Cada una de las seis actas deberá contener, al menos:

1. autoridad, competencia, participantes y posibles incompatibilidades;
2. fecha, línea base exacta, alcance, exclusiones y evidencia examinada;
3. requisitos y criterios de muestreo o prueba;
4. hallazgos, severidad, fundamento y dependencias;
5. limitaciones, evidencia no disponible y asuntos no evaluados;
6. remediaciones, responsables y plazos formalmente acordados;
7. decisión `NO_APROBADO`, `APROBADO_CON_CONDICIONES` o `APROBADO`, cuando la
   autoridad esté habilitada para emitirla;
8. condiciones de vigencia, cambio material y reauditoría;
9. firma, sello o mecanismo corporativo válido y referencia de custodia.

Este documento no contiene campos de firma cumplimentados. Una plantilla o un
commit no son un acta emitida.

## 14. Regla de decisión conjunta

Para un eventual avance serían necesarias todas estas condiciones, sin que
una compense otra:

- seis actas emitidas por autoridades competentes sobre la misma línea base;
- cero hallazgos bloqueantes abiertos;
- remediaciones verificadas de forma independiente;
- riesgos residuales valorados y aceptados solo por quien tenga competencia;
- EIPD, ENS, ENI, accesibilidad e IA resueltos según alcance;
- evidencia de pruebas funcionales, seguridad, privacidad, recuperación,
  accesibilidad y operación autorizadas;
- autorización formal de producción y condiciones de vigilancia.

En este corte las seis actas están `ACTA_PENDIENTE`; por tanto el resultado es
`NO_APROBADO` y `PRODUCCION_BLOQUEADA`.

## 15. Privacidad y material excluido

El expediente contiene cero datos personales, expedientes, candidaturas,
puestos, contactos, firmas, credenciales, secretos, endpoints, inventarios de
activos, proveedores o infraestructura reales. No incorpora resultados de una
auditoría real ni evidencia operativa de producción.

La futura custodia deberá minimizar datos, usar referencias opacas y separar
el acta compartible de anexos restringidos. Esa regla no autoriza crear ahora
ningún anexo ni acceder a información externa.

## 16. Bloqueos preservados

- CT-CUM-02 a CT-CUM-09 siguen pendientes de sus decisiones materiales.
- CT-CUM-10 permanece abierto y `AUDITORIA_NO_REALIZADA`.
- Las seis actas competentes permanecen `ACTA_PENDIENTE`.
- No se aceptan riesgos, excepciones, categorías, políticas o declaraciones.
- Datos reales, efectos jurídicos, IA, integraciones, envíos, red, servicios,
  preproducción y producción permanecen bloqueados.
- O4A y sus NO-GO no cambian por este expediente.
- Tablero, métricas y estado transversal permanecen intactos.

## 17. Criterios de revisión del candidato

La revisión técnica independiente solo puede comprobar:

1. base exacta y alta de un único Markdown;
2. trazabilidad a CT-CUM-02..09 y existencia de SHAs citados;
3. seis cuadernos, cada uno con cinco preguntas y acta pendiente;
4. formato común de hallazgos y actas sin firmas ni decisiones fabricadas;
5. regla conjunta que exige seis actas y cero bloqueantes;
6. ausencia de datos, autoridades nominativas, secretos, activos e
   infraestructura reales;
7. conservación explícita de `NO_APROBADO` y `PRODUCCION_BLOQUEADA`;
8. enlaces locales válidos y `git diff --check` verde.

Un GO de revisión técnica acredita solo coherencia del expediente candidato.
No es una de las seis actas, no cierra CT-CUM-10 y no autoriza producción.

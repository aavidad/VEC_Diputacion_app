# Matriz de campos del análisis RRHH y la retención de crédito

Fecha: 23 de julio de 2026.

Alcance: tarea O3-01 del módulo `contrataciontemporal`. Este anexo concreta el
modelo de dominio; no acredita por sí solo conformidad jurídica, suficiencia
presupuestaria ni competencia para validar un expediente.

Fuentes funcionales:

- documento de RRHH «Pantalla de procedimiento de gestión de contratación y
  gestión de bolsas»;
- [especificación normalizada](expediente_contratacion_temporal_rrhh.md);
- [matriz normativa del módulo](matriz_normativa_contratacion_temporal_2026-07-23.md).

## Análisis de RRHH

| Campo de dominio | Obligación | Autoridad | Invariante técnica | Finalidad |
| --- | --- | --- | --- | --- |
| `ModalidadClave` | Sí | Catálogo publicado aplicable al expediente | Clave bien formada; el código no compila una lista de modalidades | Identificar la modalidad que RRHH ha analizado |
| `CategoriaRef` | Sí | Fuente maestra de categorías | Referencia opaca válida | Ligar el análisis a la categoría comprobada sin copiar datos personales |
| `GrupoSubgrupo` | Sí | Fuente maestra aplicable | Código canónico acotado | Conservar el grupo o subgrupo validado |
| `CausaClave` | Sí | Catálogo publicado aplicable al expediente | Clave bien formada; la legalidad de la causa no se infiere de su texto | Identificar la causa analizada |
| `Periodo` | Sí | Análisis de RRHH y fuentes del expediente | Fechas civiles UTC, inicio no posterior a fin y horizonte técnico máximo de cien años | Delimitar la necesidad; las duraciones legales siguen gobernadas fuera del binario |
| `PorcentajeJornada` | Sí | Análisis de RRHH y fuente de jornada autorizada | Entero entre 1 y 10.000 diezmilésimas | Representar exactamente jornada completa, parcial o reducida sin coma flotante |
| `EntradaRCEsperada` | Sí | Comando interno construido desde la declaración del expediente | Referencia y huella válidas; deben coincidir exactamente con las devueltas en `ValidacionRC` | Evitar aplicar al análisis un resultado preparado para otra entrada |
| `ValidacionRC` | Sí para registrar el análisis | Fuente presupuestaria autorizada | Contrato descrito en la sección siguiente | Conservar el resultado presupuestario y su evidencia |
| `CostePrevisto` | No, mientras la fuente no lo proporcione | Conector de cálculo autorizado | Céntimos positivos en EUR, con límite técnico calculable | Conservar una estimación reproducible, no un valor de nómina |
| `FuenteCosteRef` | Sí cuando existe coste | Versión de tabla o cálculo autorizado | Referencia opaca; no puede quedar sin coste ni faltar cuando hay coste | Acreditar procedencia y versión de la estimación |
| `Observaciones` | No | RRHH | UTF-8, NFC, sin controles y máximo de 4.000 caracteres | Motivación complementaria minimizada |

El límite de cien años y el máximo monetario son defensas técnicas. No
establecen la duración permitida de una modalidad, un importe presupuestario
máximo ni sustituyen las reglas publicadas, convenios, bases o acuerdos.

## Resultado de la RC

No existe un estado compilado «pendiente». Pendiente significa que todavía no
hay una `ValidacionRC` registrable.

| Resultado | Registrable | Habilita avance | Evidencia financiera copiada al resultado |
| --- | --- | --- | --- |
| `validada` | Sí | Sí | Fecha autoritativa, número, importe y documento |
| `no_requerida` | Sí, con motivo | Sí | Ninguna; la entrada queda ligada por referencia y huella |
| `rechazada` | Sí, con motivo | No | Ninguna; la entrada rechazada no se borra ni se transforma |
| Ausencia | No es una decisión | No | Ninguna |

`Expediente.RegistrarViaCobertura` exige
`AnalisisRRHH.HabilitaAvance()`. Que un rechazo sea estructuralmente válido
solo permite conservarlo con trazabilidad; el dominio impide continuar.

## Evidencia de validación de la RC

| Campo de dominio | Resultados | Invariante |
| --- | --- | --- |
| `EntradaRCEsperada.Referencia` | Todos | Referencia que el análisis esperaba validar |
| `EntradaRCEsperada.HuellaSHA256` | Todos | Huella que el análisis esperaba validar |
| `ValidacionRC.EntradaRef` | Todos | Referencia declarada por el resultado; debe coincidir con la esperada |
| `ValidacionRC.HuellaEntradaSHA256` | Todos | Huella declarada por el resultado; debe coincidir con la esperada y no ser centinela |
| `FuenteRef` | Todos | Identificador de la fuente que el resultado afirma haber consultado |
| `ReciboRef` | Todos | Referencia al recibo que O3-03 deberá atestar |
| `ValidadaEn` | Todos | Instante UTC canónico, con precisión máxima de microsegundo |
| `FechaRC` | Solo `validada` | Fecha civil declarada por la fuente, UTC a medianoche y no posterior a `ValidadaEn` |
| `Numero` | Solo `validada` | Número de RC validado |
| `Importe` | Solo `validada` | Importe exacto positivo en EUR y dentro del límite técnico |
| `DocumentoRef` | Solo `validada` | Referencia opaca al documento custodiado |
| `Motivo` | Obligatorio en `rechazada` y `no_requerida` | Texto normalizado, acotado y sin datos residuales innecesarios |

El dominio coteja en tiempo constante la huella esperada con la declarada y
rechaza cualquier diferencia de referencia o contenido. Esto aporta ligadura
semántica interna, pero **no atesta** que la fuente, el recibo, la fecha o el
resultado sean auténticos. O3-03 debe verificar esa autoridad mediante un
puerto y un recibo durable antes de que O3-02 confirme el efecto.

`ValidadaEn` no puede ser posterior a la actuación que registra el análisis.
La fuente, el recibo, el vínculo esperado y los instantes proceden de la
composición interna; ningún JSON, cookie o cabecera libre concede autoridad.

En resultados distintos de `validada`, `FechaRC`, `Numero`, `Importe` y
`DocumentoRef` deben estar vacíos. La declaración original continúa disponible
por su referencia y huella, evitando duplicar en la decisión datos que no han
sido confirmados por la fuente autoritativa.

## Invariantes cruzadas

1. Un coste no puede existir sin fuente ni una fuente sin coste.
2. El coste y la RC usan céntimos exactos y EUR; nunca `float64`.
3. Cuando la RC está validada, el coste puede ser exactamente igual a la RC,
   pero no superior.
4. Una RC rechazada puede guardarse, rectificarse mediante una nueva versión y
   auditarse, pero no permite pasar a cobertura.
5. Una RC no requerida solo permite avanzar con fuente, recibo, entrada ligada
   y motivo.
6. El objeto se valida antes de clonarse y la copia no comparte punteros de
   coste, fecha o importe de RC.
7. Modalidades, causas y jornadas legalmente permitidas se resuelven con la
   versión de catálogo aplicable; las invariantes técnicas no las inventan.

## Correspondencia de cumplimiento

| Exigencia | Evidencia aportada por O3-01 | Evidencia que sigue pendiente |
| --- | --- | --- |
| RGPD: minimización, exactitud e integridad | Referencias opacas, entrada ligada, ausencia de PII y resultados sin residuos | Inventario campo–finalidad–base–retención, RAT y EIPD aprobados |
| ENS: autenticidad, integridad y trazabilidad | Fuente, huella, recibo e instantes canónicos; rechazo cerrado | Categorización, declaración de aplicabilidad, riesgos, operación y auditoría |
| Procedimiento administrativo | Diferencia entre decisión pendiente, validada, no requerida y rechazada | Competencia, firma, registro, rectificación y recibo durables |
| ENI y archivo | Documento por referencia, no incrustado en el agregado | Metadatos ENI, índice, firma, conservación y política de acceso |
| Control presupuestario | Coste exacto, RC suficiente y fuente explícita | Contrato del conector, reglas aprobadas por Intervención y aceptación funcional |

## Clasificación, minimización y acceso

Clasificación provisional para diseño, pendiente del inventario de
tratamientos, EIPD y categorización ENS:

| Grupo de campos | Clasificación provisional | Minimización |
| --- | --- | --- |
| Modalidad, categoría, grupo, causa, periodo y jornada | Uso interno RRHH; pueden revelar información de organización o empleo al relacionarse con el expediente | El agregado usa claves y referencias; no copia identidad, DNI, contacto ni datos de salud |
| Entrada, huella, fuente y recibo de RC | Evidencia interna restringida | Referencias opacas y huella; el contenido de la declaración permanece en su autoridad |
| Fecha, número, importe y documento de RC | Información presupuestaria restringida | Solo aparecen en un resultado `validada`; el documento permanece en almacén por referencia |
| Coste previsto | Información económica interna | Importe total exacto y fuente; no incorpora desglose de nómina ni conceptos personales |
| Observaciones y motivo | Uso interno restringido | Límites de longitud y prohibición organizativa de incluir datos no necesarios |

El acceso será por operación, expediente, unidad y finalidad, con denegación
predeterminada:

| Operación | Capacidad técnica prevista | Datos mínimos |
| --- | --- | --- |
| Registrar o rectificar análisis | `contratacion_temporal.analisis.validar` | Entrada esperada, resultado, fuente, recibo y campos funcionales |
| Consultar resumen | `contratacion_temporal.expediente.consultar` | Estado y referencias; sin documento ni observaciones si no son necesarios |
| Abrir documento de RC | Capacidad documental específica aún por definir | `DocumentoRef`, expediente y finalidad exacta |
| Consultar auditoría | `contratacion_temporal.auditoria.consultar` | Actuación y recibos minimizados; no contenido documental |
| Avanzar a cobertura | `contratacion_temporal.cobertura.decidir` | Predicado de avance y versión; no reinterpreta la RC |

La matriz anterior no asigna capacidades a perfiles ni aprueba accesos. El PDP
común deberá decidir cada operación con garantía alta y ámbito interno.

## Conservación y responsables pendientes

No existe todavía un plazo de conservación aprobado para estos campos. Como
medida provisional de bloqueo:

1. no se usarán datos reales hasta cerrar CT-CUM-02, CT-CUM-03, CT-CUM-04,
   CT-CUM-06, CT-CUM-07 y CT-CUM-10;
2. no se programará expurgo ni borrado silencioso;
3. el documento y la declaración se conservan en sus autoridades y este
   agregado solo mantiene referencias;
4. Archivo y DPD deben aprobar serie, plazo, bloqueo, acceso, transferencia y
   eliminación;
5. una obligación de conservación o litigio debe suspender cualquier expurgo.

Propuesta de propiedad pendiente de designación formal:

| Materia | Propietario funcional propuesto | Aprobación o intervención obligatoria pendiente |
| --- | --- | --- |
| Análisis de modalidad, causa, periodo y jornada | RRHH | Jefatura/órgano competente, Asesoría Jurídica cuando proceda |
| Fuente, validación y suficiencia de RC | Intervención o unidad presupuestaria competente | Intervención y responsable económico formalmente designados |
| Tratamiento y acceso | Responsable del tratamiento | DPD, responsable de seguridad y responsable del sistema |
| Documento y conservación | Unidad productora del expediente | Archivo, Secretaría y DPD |
| Integración técnica y recibo | Sistemas | Responsable del sistema, seguridad e Intervención/RRHH según autoridad |

Esta tabla no atribuye competencias. Las designaciones y aprobaciones deben
constar en el expediente de cumplimiento antes de datos reales o efectos
jurídicos.

## Pruebas de cierre del dominio

`analisis_test.go` cubre:

- jornada mínima, completa, nula y superior al 100 %;
- periodo exacto en el límite técnico y exceso de un día;
- fechas civiles, UTC y precisión canónica;
- los tres resultados cerrados y la ausencia pendiente;
- cotejo de referencia y huella esperadas, huella centinela, fuente y recibo;
- orden `FechaRC <= ValidadaEn <= actuación`;
- JSON sin fecha, número, importe ni documento residuales;
- coste igual a la RC, superior por un céntimo y máximo técnico exacto;
- transición a cobertura permitida con RC validada/no requerida y prohibida
  con RC rechazada;
- texto normalizado y copia defensiva.

O3-01 no incluye atestación de fuente/recibo, autorización, persistencia,
rectificación, auditoría durable, outbox, API ni pantalla. Esas capacidades
corresponden a O3-02 a O3-05. Web, API, CLI y una futura aplicación de
escritorio deben invocar los mismos casos de uso; ninguna depende de cookies
para construir autoridad.

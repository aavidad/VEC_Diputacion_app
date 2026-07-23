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

La aplicación que registre el siguiente hito debe exigir el predicado
`AnalisisRRHH.HabilitaAvance()`. Que un rechazo sea estructuralmente válido
solo permite conservarlo con trazabilidad; no autoriza a continuar.

## Evidencia de validación de la RC

| Campo de dominio | Resultados | Invariante |
| --- | --- | --- |
| `EntradaRef` | Todos | Referencia opaca a la declaración o entrada exacta que se validó |
| `HuellaEntradaSHA256` | Todos | SHA-256 hexadecimal de 64 caracteres; se rechaza la huella centinela de ceros |
| `FuenteRef` | Todos | Fuente presupuestaria autoritativa consultada |
| `ReciboRef` | Todos | Recibo opaco de la consulta o decisión |
| `ValidadaEn` | Todos | Instante UTC canónico, con precisión máxima de microsegundo |
| `FechaRC` | Solo `validada` | Fecha civil autoritativa de la RC, UTC a medianoche |
| `Numero` | Solo `validada` | Número de RC validado |
| `Importe` | Solo `validada` | Importe exacto positivo en EUR y dentro del límite técnico |
| `DocumentoRef` | Solo `validada` | Referencia opaca al documento custodiado |
| `Motivo` | Obligatorio en `rechazada` y `no_requerida` | Texto normalizado, acotado y sin datos residuales innecesarios |

La pareja `EntradaRef`–`HuellaEntradaSHA256` impide que un recibo válido para
una declaración se reutilice sobre otra. La fuente, el recibo y el instante no
se reconstruyen desde el navegador.

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
6. El objeto se valida antes de clonarse y la copia no comparte el puntero del
   coste.
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

## Pruebas de cierre del dominio

`analisis_test.go` cubre:

- jornada mínima, completa, nula y superior al 100 %;
- periodo exacto en el límite técnico y exceso de un día;
- fechas civiles, UTC y precisión canónica;
- los tres resultados cerrados y la ausencia pendiente;
- ligadura de entrada, huella centinela, fuente, recibo e instante;
- fecha autoritativa obligatoria y residuos prohibidos;
- coste igual a la RC, superior por un céntimo y máximo técnico exacto;
- rechazo registrable que no habilita avance;
- texto normalizado y copia defensiva.

O3-01 no incluye autorización, persistencia, rectificación, auditoría durable,
outbox, API ni pantalla. Esas capacidades corresponden a O3-02 a O3-05.

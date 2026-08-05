# Enmienda O1b: corrección final bajo parada 800

Fecha: 5 de agosto de 2026.

Estado: **propuesta para revisión independiente; NO-GO para editar código**.

Base candidata: `56c0ac079419b187f851de8183d5f87b5b367b71`.

Acta del NO-GO:
[revisión funcional del candidato 56c0ac0](revisiones/revision_f0_h0b_c4b2_g2o_o1b_codigo_56c0ac0_nogo_2026-08-05.md).

## Motivo único

El candidato implementa un lector correcto según la inspección disponible,
pero su autoprueba no mata mutantes del constructor, de la carga íntegra de la
trama y del estado L1. También faltan dos vectores explícitos de L2 y
precedencia. G2 ocupa 788 líneas: la parada 790 aceptada deja dos líneas y no
permite una corrección legible.

No se comprime, retira o fusiona evidencia para obtener un verde. Esta
enmienda autoriza, solo tras su GO, una deduplicación semánticamente neutra y
la evidencia ausente hasta el tope absoluto DEC-051 de 800 líneas.

## Correcciones obligatorias

### Constructor y fixtures

Los recorridos pueden conservar una fábrica controlada de fixtures para no
propagar artificialmente errores imposibles después de la prueba del
constructor. La API real `nuevoLectorTramaM38` debe probarse primero y exigir,
para cada clase válida:

- lector no nulo y error nulo;
- clase exacta, límite derivado de `limiteTramaM38` y estado L0;
- longitud cero, buffer completamente cero y error interno nulo.

La clase inválida mantiene lector nulo e identidad
`errClaseLectorM38`. Ningún error se ignora y un fallo no se sustituye por un
lector sintético, pánico o resultado válido.

### Contenido íntegro

Un auxiliar de prueba recodifica cada `tramaM38` entregada y exige igualdad
byte a byte con su entrada canónica, además de clase exacta. Se aplica a:

- todos los puntos de corte de las cuatro clases;
- fragmentación byte a byte;
- confirmación monoframa desde L2 y en la misma llamada;
- cada uno de los controles coalescidos;
- controles reensamblados desde parcial y entregas directas.

Así se comprueban clase, orden y totalidad de campos y ticket; no basta con la
clase o el primer campo.

### Estados y precedencias

La matriz exige explícitamente:

- L0 después de CONTROL sin EOF y tras limpiar una entrega;
- L1 después de una parcial, incluida entrada vacía posterior sin EOF;
- L2 después de una monoframa completa sin EOF;
- L3 antes de devolver CONTROL coincidente con EOF y tras monoframa final;
- L4 después de cada error, junto con tupla cero, limpieza y pegajosidad.

Se añaden tres secuencias diferenciadas:

1. monoframa gramaticalmente inválida completa sin EOF: L2 sin resultado;
   EOF vacío posterior: `errTramaFlujoM38`, L4;
2. copia de ese L2 más dato y `fin=true`: `errDatosPosterioresM38`, L4;
3. monoframa gramaticalmente inválida, LF y cola en un único fragmento:
   `errDatosPosterioresM38` antes de O1a, tanto con `fin=false` como con
   `fin=true`.

La terminalidad conserva casos y diagnósticos separados para EOF limpio
inicial, repetición, limpieza, dato con EOF desde L3, CONTROL+EOF con L3
enclavado, dato sin EOF desde ese L3 y llamada vacía sin EOF. Se permiten
clones de un estado material ya obtenido para evitar repetir preparación; no
se combinan aserciones que impidan identificar la transición fallida.

## Deduplicación permitida

Solo se permite:

- reutilizar una copia del snapshot material de frontera para probar exceso y
  byte inválido como ramas todavía independientes;
- usar clones de L2/L3 ya acreditados para bifurcar entradas;
- recorrer EOF repetido mediante un bucle que conserve diagnóstico de índice;
- integrar un auxiliar privado en la función principal si cada caso conserva
  su diagnóstico y aserción exactos.

No se elimina ningún vector de `56c0ac0`, no se comprimen varias expectativas
en una comprobación vacua y no se acortan nombres o mensajes para caber.

## Ledger

Dos estimaciones independientes discrepan: una separación sin deduplicar
añadiría 65..95 líneas; una corrección con los solapamientos anteriores puede
cerrar en 798..800. Se autoriza únicamente el segundo diseño como experimento
acotado; la parada decide, no la estimación.

Desde su GO documental, esta enmienda sustituye exclusivamente la previsión
713..783, la parada 790 y el delta máximo +390 del ledger correctivo anterior
por la previsión 798..800, la parada 800 y el delta máximo +400. El resto del
contrato, matriz, write-set, invariantes, puertas y prohibiciones permanece
vigente sin modificación.

| Unidad | Base O1b | Candidato | Total previsto | Parada |
| --- | ---: | ---: | ---: | ---: |
| Runner | 800 | 800 | 800 | 800 |
| G1 | 686 | 686 | 686 | 686 |
| G2 | 400 | 788 | 798..800 | 800 |
| Capturador | 799 | 799 | 799 | 799 |
| Adaptador | 527 | 527 | 527 | 527 |
| D2d | 145 | 145 | 145 | 145 |
| D2c | 588 | 588 | 588 | 588 |
| H0b | 580 | 580 | 580 | 580 |

El delta G2 parte de `+388`, prevé `+398..+400` y se detiene al alcanzar
800. Si falta un vector, un diagnóstico o una aserción a esa altura, no se
edita G1 ni se excede DEC-051: se vuelve a revisión para diseñar una fuente G3
y su manifiesto.

## Write-set e invariantes

Tras el GO documental solo pueden cambiar:

1. G2, para esta corrección exacta;
2. runner, al final, únicamente los literales SHA-256 de G2 y del binario
   compuesto.

G1, su literal SHA, capturador, adaptador, D2d, D2c y H0b son invariantes. El
manifiesto conserva seis fuentes. No se abre un séptimo fichero, O2, FD,
procesos, señales, pidfd operativo, Bash de caso, Docker, PostgreSQL, SQL,
red, reloj o dependencia.

## Paradas y puertas

Se detiene sin commit si:

- G2 supera 800 o falta una corrección al llegar a 800;
- se reduce cobertura o diagnóstico para caber;
- cambia G1 u otra unidad invariante;
- el runner cambia fuera de los dos literales;
- `--supervisar-m38` deja de devolver 64;
- cualquier mutante documentado continúa verde.

Después de estabilizar G2 se repiten dos builds privados disjuntos, huellas,
autoprueba y mutantes focales, modos 64, FD/hijos, Bash, ShellCheck, Gitleaks y
doble revisión independiente. Con doble GO focal se ejecutan las puertas
globales sobre el candidato antes de integrar. Tras la integración se repiten
proporcionalmente sobre el árbol conjunto antes de publicar. Docker,
PostgreSQL, red y E2E siguen fuera de O1b.

Esta enmienda no cambia métricas ni acepta `56c0ac0`.

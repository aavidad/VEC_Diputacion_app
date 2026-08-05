# Revisión F0-H0b/C4b-2/G2-O/O2a-P0

Fecha: 5 de agosto de 2026.

Documento revisado:
[decisión O2a-P0](../decision_f0_h0b_c4b2_g2o_o2a_preparacion_g3_2026-08-05.md).

## Primera revisión

El primer dictamen funcional emitió `NO-GO`, `P0=0`, `P1=2`, `P2=1`:

1. faltaba fijar la base exacta `ec53009`;
2. una comparación «normalizada» no garantizaba el traslado literal del Shell;
3. la propia decisión no declaraba la doble revisión documental previa.

La corrección fijó la base completa, la huella SHA-256 del bloque de 17 líneas,
las cardinalidades de definición/llamada/carga y dos GO documentales antes de
programar, distintos de las dos revisiones posteriores del código.

## Dictámenes finales

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Funcional, autoridad y fallo cerrado | 0 | 0 | 0 | GO |
| Ledger, captura e invariantes | 0 | 0 | 0 | GO |

Ambas revisiones reprodujeron:

- base `ec530091e6f157baa54ff50e9c70f21c7a014e94`;
- bloque runner 301–317, 17 líneas y SHA-256
  `aae98945ae26e7b4f2637e662157bdaf26a414d3100b046d2c91c4cf1fa59d74`;
- runner `800−17=783`;
- D2d `145+17+1=163`;
- manifiesto exacto de seis fuentes;
- D2d capturado y cargado antes de la llamada única;
- G1, G2, capturador, adaptador, D2c, H0b y binario Go invariantes;
- ejecución directa de D2d cerrada en 64;
- ausencia de ampliación de autoridad, cambio de orden o debilitamiento del
  fallo cerrado;
- prevalencia O1a: `SOBRE` antes de `ARMAR` y ausencia de ACK vivos.

## Decisión

O2a-P0 queda autorizada exclusivamente sobre la base fijada y dentro de su
write-set. No autoriza crear G3, implementar S0, O2a u O2b ni cambiar métricas.
El candidato necesita dos revisiones independientes nuevas y todas las puertas
definidas antes de integrarse.

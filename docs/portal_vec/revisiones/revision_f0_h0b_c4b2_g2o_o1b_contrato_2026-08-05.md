# Revisión independiente F0-H0b/C4b-2 G2-O/O1b

Fecha: 5 de agosto de 2026.

Estado: **GO documental para implementar exclusivamente O1b**.

Base técnica: `67331c695d217adeca9efd7142c612c3bc6652e6`.

Contrato final: `4b765eb8bb4a2977c924e6077c1d03f43231744d`.

## Trazabilidad de revisión

| Corte | Seguridad | Máquina/API | Ledger | Resultado |
| --- | --- | --- | --- | --- |
| `6f4a118` | 3 P1 | 4 P1 | 2 P1 | NO-GO |
| `086a122` | 2 P1, 1 P2 | GO | GO | NO-GO |
| `4b765eb` | GO | GO | GO | GO final |

Ningún NO-GO autorizó código. Dirección corrigió el contrato y volvió a
someter el documento completo a los tres revisores.

## Hallazgos cerrados

- EOF inicial en una clase monoframa es error tipado y pegajoso;
- un `CONTROL` que termina junto a EOF enclava el fin antes de devolver la
  trama;
- el mismo EOF se propaga al sobrante de un fragmento coalescido;
- firma, valores cero y contador consumido quedan definidos por resultado;
- L1 vacío sin EOF conserva estado y parcial;
- L2 conserva una copia cruda completa, no un alias del fragmento;
- la monoframa no se decodifica ni entrega antes del EOF limpio;
- buffer mutable y cadenas inmutables tienen garantías de borrado distintas y
  explícitas;
- máximos canónicos y límites físicos se prueban por separado;
- la matriz no exige tramas gramaticalmente imposibles.

Máximos canónicos confirmados, incluido LF:

| Clase | Máximo canónico | Máximo físico |
| --- | ---: | ---: |
| `SOBRE` | 2212 | 4096 |
| `CONTROL` | 100 | 1024 |
| `TERMINAL` | 179 | 1024 |
| `TICKET` | 2060 | 2060 |

## Ledger aceptado

El write-set de código queda limitado a:

1. G2, para lector y autoprueba O1b;
2. runner, únicamente los literales SHA-256 de G2 y del binario compuesto.

G1, su literal, capturador, adaptador M38, D2d, D2c y H0b son invariantes. El
runner permanece en 800 líneas, el manifiesto en seis fuentes y G2 parte de 400
líneas con previsión 585..665 y parada dura 680. Superar el delta +265 o la
parada exige detenerse; no permite minificar, retirar pruebas o crear un
séptimo fichero.

## Alcance autorizado

O1b continúa siendo código puro: no abre FD, procesos, señales, pidfd, Bash,
Docker, PostgreSQL, SQL, red o reloj. `--supervisar-m38` debe seguir devolviendo
64. O2 y fases posteriores permanecen cerradas.

El candidato necesita al menos dos revisiones independientes posteriores al
productor y todas las puertas fijadas en el contrato. Este GO no modifica F0
`10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14`
ni el `NO-GO` de producción.

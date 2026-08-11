# Checkpoint O4A-P1D — contrato del deadline

Fecha: 11 de agosto de 2026.

Estado: candidato documental. No abre implementación.

## Base verificada

- SHA: `50f2a81302dc202bca3cf4d9986b7351b10fa9ae`.
- Rama: `trabajo/o4a-p2-semilla-v2-20260811`.
- CI: [31529462600](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31529462600), `Success`, cinco puertas.
- Contrato O4a: SHA-256 `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.
- P1B: SHA-256 `e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`.

## Hallazgo y decisión

P1/P1C no copia `ahoraCaso` ni `finCaso`. Un reemplazo
coordinado puede conservar monotonicidad, 180 segundos y genealogía aparente,
por lo que P3 no puede distinguirlo de la pareja heredada.

La decisión
[`decision_f0_h0b_c4b2_g2o_o4a_p1d_contrato_deadline_2026-08-11.md`](../decision_f0_h0b_c4b2_g2o_o4a_p1d_contrato_deadline_2026-08-11.md)
autoriza únicamente dos copias privadas por valor en P1, capturadas una vez
después de acreditar C5 y antes del CAS `2→3`. No autoriza reloj ni efecto.
`finBootstrap` se coteja en ese snapshot pre-CAS, pero no se copia porque su
límite queda consumido en el handoff y no gobierna P2/P3.

## Write-set P1D

1. decisión P1D;
2. este checkpoint;
3. acta funcional independiente;
4. acta de seguridad independiente.

Cero código, pruebas, herramientas o migraciones. O3, P0, P1, P1B, P1C y P2
permanecen byte-inmutables.

## Dependencias

`P1D → P1E-SELLOS-DEADLINE → P3-ARBITRAJE`.

P1E editará exclusivamente el Go P1 existente y su prueba. P2 no se modifica.
P3 solo se reabre tras doble GO y CI 5/5 de P1E.

## Puertas pendientes

- relectura completa y hashes finales;
- enlaces, formato y `git diff --check`;
- escaneo de secretos proporcional y puerta CI `secretos`;
- doble GO `P0=P1=P2=0`;
- commit/push normal, árbol limpio y CI 5/5 del SHA exacto.

No hay merge, integración, métricas, master, producción ni despliegue.

# Checkpoint O4A-P1B-CONTRATO-SELLOS

Fecha: 11 de agosto de 2026.

Estado: **GO DOCUMENTAL PARA PUBLICACIÓN, P0=P1=P2=0**. No autoriza
implementación; queda condicionado a commit, push normal y CI 5/5.

## Base y hallazgo

- Base exacta P1: `0e750d41cbf72b0f0952341fa18e25474c3269fb`.
- Rama P1: `trabajo/o4a-p1-autoridad-20260811`.
- CI P1: `31516149966`, `completed/success`, cinco jobs.
- Contrato O4a SHA-256:
  `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.
- P2 quedó limpio, sin commit ni push, al acreditarse que P1 no transporta el
  payload inmutable requerido por `senal_raw` y `control_raw`.

## Decisión mínima

La decisión propia autoriza para una minitarea posterior únicamente:

- sellar la palabra completa del observador bajo `senal_raw`;
- sellar un enum CONTROL cerrado de cuatro causas bajo `control_raw`;
- mantener VACIO para el resto;
- preservar CAS 2→3, owners, recursos, primera observación y raw CONT.

No autoriza P2/P3 ni efectos. La implementación se limita al Go P1 y su prueba
existentes.

## DAG condicionado

`P1 publicado → P1B documental → P1C-SELLOS-RAW → P2-SEMILLA → P3-ARBITRAJE`.

P1C y P2 solo se abren por asignación de dirección tras doble GO y CI 5/5 del
nodo anterior. O4b/O4c, integración, producción y métricas permanecen cerrados.

## Snapshot y revisiones

La decisión material tiene SHA-256
`e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`.
Las revisiones [funcional](revision_funcional_f0_h0b_c4b2_g2o_o4a_p1b_contrato_sellos_2026-08-11.md)
y de [seguridad](revision_seguridad_f0_h0b_c4b2_g2o_o4a_p1b_contrato_sellos_2026-08-11.md)
reproducen el defecto y exigen `P0=P1=P2=0` sobre los bytes finales.

## Cierre pendiente

Quedan commit documental pequeño, push normal y CI 5/5 del SHA exacto. No hay
merge ni apertura automática de P1C.

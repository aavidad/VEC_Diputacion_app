# Revisión funcional O4A-P1B-CONTRATO-SELLOS

Fecha: 11 de agosto de 2026.

Resultado: **GO**, `P0=P1=P2=0`.

## Alcance y base

Revisión independiente, sin editar el worktree productor, sobre la base exacta
P1 `0e750d41cbf72b0f0952341fa18e25474c3269fb`. La rama remota
`trabajo/o4a-p1-autoridad-20260811` resuelve a ese SHA. El contrato O4a
publicado conserva SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.

Bytes documentales revisados:

- decisión P1B: SHA-256
  `e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`;
- checkpoint P1B: SHA-256
  `d7491d9cc58a52ce87ecabcf676a79f477c67add14bb997431e23718f8db763b`.

## Reproducción causal

El hallazgo queda confirmado contra las fuentes reales:

1. O3c P4 carga localmente palabra y signo del observador, pero su agregado
   conserva solo el discriminante `senal_raw`.
2. O3c P5 admite el handoff mientras el estado enmascarado continúe en 2.
3. P1 exige actualmente `palabra==baseline` y signo cero, por lo que rechaza
   una `senal_raw` legítima y tampoco sella su payload.
4. O3c conserva solo `control_raw`; P1 sella el puntero CONTROL, no el canon
   inmutable necesario para distinguir las cuatro causas.

La decisión corregida resuelve el defecto con dos únicos valores privados:
la palabra completa y un enum CONTROL cerrado. No duplica baseline o signo,
ni transporta errores, texto libre, buffers, nonce, ticket o recursos.

## Comprobaciones funcionales

- La carga única pre-CAS es el punto de linealización; P2 no relee estado
  mutable.
- Para `senal_raw`, estado distinto de 2 o contador regresivo es AF. Delta
  uno con INT/TERM es traducible; delta cero, mayor que uno o signo ambiguo
  queda sellado para `INCIDENTE` en P2.
- Para los otros discriminantes se conserva la exigencia exacta
  palabra-baseline y signo cero.
- CONTROL terminal funcional exige coincidencia exacta entre
  `control.causa` y `custodia.primeraCausa`.
- CONTROL activo solo es admisible íntegro en S3, con causa, primera causa y
  fallo vacíos; representa el fallo de ronda ya acreditado y normaliza a
  `PROTOCOLO_65`. S1/S2, terminal interno o combinación imposible son AF.
- `VACIO` solo corresponde a un discriminante distinto de `control_raw`.
- Owners O4A, CAS 2→3, recursos, primera observación, raw CONT y máquina A0--AF
  permanecen invariantes.

La matriz S01--S10 y los mutantes cubren ramas canónicas, ambigüedad,
corrupción, alias/replay, inmutabilidad y ausencia estructural de efectos.

## Hallazgos resueltos

La primera versión convertía CONTROL terminal interno en causa funcional y
rechazaba CONTROL activo. La segunda versión no exigía vacío el latch
`primeraCausa` en la rama activa y situaba de forma imprecisa la captura. La
versión final corrige todos esos puntos. No quedan hallazgos P0, P1 o P2.

## Límites y cierre

El write-set futuro queda mínimo: editar solo el Go P1 y su prueba existentes,
sin tercer fichero, O3, P0 ni contrato transversal. `O4A-P1C-SELLOS-RAW` no
abre P2: requiere doble GO propio y CI 5/5 antes de reabrir
`O4A-P2-SEMILLA`. P3 y fronteras posteriores permanecen bloqueadas.

Puertas documentales reproducidas: lectura completa, hashes, rama/base remota,
enlaces locales y `git diff --check`. `gh` y Gitleaks no están instalados en
el entorno revisor; la CI y la puerta de secretos corresponden al cierre del
productor.

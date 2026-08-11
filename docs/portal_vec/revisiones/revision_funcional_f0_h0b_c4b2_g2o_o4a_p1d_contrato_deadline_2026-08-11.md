# Revisión funcional O4A-P1D — contrato del deadline

Fecha: 11 de agosto de 2026.

Revisor: agente funcional independiente, rama
`revision/o4a-p1d-funcional-20260811`, sin edición del worktree productor.

## Material revisado

- base exacta `50f2a81302dc202bca3cf4d9986b7351b10fa9ae`;
- contrato O4a SHA-256
  `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
- decisión P1B SHA-256
  `e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`;
- decisión P1D SHA-256
  `41253389abb537755f3320dcd84696e7acafb3fba59b41a5bed7b1a6cb53931e`;
- checkpoint P1D SHA-256
  `0ef2a738ba25de70ecb8eed55761683acb5ee413464faede99f2460e87e9bafa`.

También contrasté los bytes productivos congelados de P1/P1C, su prueba,
P2 y su prueba, cuyos SHA-256 son respectivamente `d2cd635c0bc6e06c...`,
`bad464cf6b92e3b8...`, `493c99cc2dcdeb62...` y `c7357d6a856b803e...`.

## Reproducción del hallazgo

El agregado C5 contiene `ahoraCaso` y `finCaso` como campos privados mutables.
P1 conserva el puntero al agregado, pero su `sellosO4aM38` no copia ninguno de
los dos. `entradaBaseExactaO4aM38` tampoco valida la pareja. Por ello, otro par
monotónico, no cero y separado 180 segundos puede sustituirlos coordinadamente
después de P1 sin que P3 distinga el reinicio. El NO-GO de P3 queda confirmado
como una carencia de autoridad temporal.

## Auditoría funcional de la decisión

La corrección es mínima y suficiente:

- sella por valor solo `ahoraCaso` y `finCaso` durante P1;
- toma la pareja y `finBootstrap` una vez, tras acreditar C5 y antes del CAS
  de custodia `2→3`;
- valida marca cero, componente monotónico, orden, duración exacta de 180 s,
  resultado exacto del helper O3c, overflow y `ahoraCaso < finBootstrap`;
- usa `finBootstrap` solo para la genealogía de entrada: no copiarlo es
  coherente porque ese límite queda consumido y no gobierna P2 ni P3;
- prohíbe segundas cargas, reloj nuevo, reconstrucción civil y recálculo;
- obliga a P3 a calcular exclusivamente desde los sellos y a comparar el
  origen mutable con `==` antes de cualquier evento, syscall o lectura de
  reloj; una divergencia persistente converge a AF y un ABA no adquiere
  autoridad;
- mantiene P2 y todos los sellos raw, owners, recursos y estados
  byte-inmutables y sin efectos.

El write-set P1E queda limitado al Go P1 existente y su prueba. D01–D10 y los
mutantes cubren omisión/intercambio de sellos, cero/civil, límites 180±1 ns,
bootstrap, overflow, captura tardía, segunda carga, comparación insuficiente,
sustitución y alteración de autoridad. El DAG es acíclico y exacto:
`P1D → P1E-SELLOS-DEADLINE → P3-ARBITRAJE`; P2 no se repite.

## Dictamen

**GO funcional. P0=0, P1=0, P2=0.**

El documento no abre implementación. P3 solo podrá reabrirse después del
doble GO y CI 5/5 de P1E sobre su SHA exacto.

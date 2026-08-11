# Revisión de seguridad O4A-P1D — contrato del deadline

Fecha: 11 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4a-p1d-seguridad-20260811`, sin edición del worktree productor.

Dictamen: **GO**, `P0=0`, `P1=0`, `P2=0`.

## Material revisado

- base exacta `50f2a81302dc202bca3cf4d9986b7351b10fa9ae`;
- contrato O4a SHA-256
  `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
- decisión P1B SHA-256
  `e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`;
- decisión P1D V3 SHA-256
  `41253389abb537755f3320dcd84696e7acafb3fba59b41a5bed7b1a6cb53931e`;
- checkpoint P1D SHA-256
  `0ef2a738ba25de70ecb8eed55761683acb5ee413464faede99f2460e87e9bafa`;
- P1/P1C, P2 y los tipos y operaciones O3c que crean y entregan las marcas.

## Reproducción del riesgo

Los bytes de la base confirman que `agregadoO4aM38` contiene `ahoraCaso` y
`finCaso` como campos mutables. O3c los crea antes del CONT y valida su
monotonicidad y separación de 180 segundos, pero `sellosO4aM38` no conserva
copias. P1 retiene el puntero al agregado y tampoco coteja la pareja con el
`finBootstrap` heredado.

En consecuencia, reemplazar antes de P3 ambos campos por otra pareja
monotónica, no cero y separada exactamente 180 segundos supera las
comprobaciones existentes. Una primera captura en P3 convertiría esa pareja
reemplazada en autoridad y permitiría reiniciar el plazo. El NO-GO previo de
P3 queda reproducido desde código, no inferido del relevo.

## Controles acreditados

La decisión V3 autoriza el cambio mínimo: dos copias privadas por valor de
`time.Time` en los sellos P1. La captura ocurre una vez después de validar C5
y antes del CAS `2→3`; la validación y preasignación trabajan solo con los
locales capturados y no releen el origen. `finBootstrap` se usa en ese mismo
snapshot para acreditar genealogía y no se convierte en un plazo activo de
P2/P3.

El canon temporal falla cerrado ante cero, pérdida monotónica, relación
invertida, duración distinta, overflow o `ahoraCaso >= finBootstrap`. El
helper existente se usa como cálculo puro con sus dos retornos; el fin
calculado solo se coteja mediante `ok && finCaso == finCalculado`, no sustituye
ni reinicia la marca recibida.

P3 deberá calcular exclusivamente desde las copias selladas y, antes de
evento, syscall o lectura de reloj, cotejar cada original con el operador Go
`==`. Una divergencia persistente es AF. Una escritura ABA restaurada no
cambia la autoridad porque el origen solo participa en ese cotejo y no en el
cálculo. No se introduce `Now`, timer, efecto, parser, getter, API, log,
serialización, goroutine, canal ni autoridad exterior.

El write-set P1E queda limitado al Go P1 existente y su prueba. P2 permanece
byte-inmutable. La matriz D01–D10 y los mutantes cubren omisión/intercambio de
sellos, reconstrucción civil, bordes 180 s, bootstrap, captura tardía,
segunda carga, sustitución, carrera y preservación de raw, owners y recursos.

## Hallazgos y cierre

La primera versión dejaba ambiguo un segundo cotejo del origen y el tipo de
igualdad temporal. La segunda conservaba una expresión imposible para el
helper de dos retornos, una contradicción sobre `Add` y una afirmación excesiva
sobre ABA. La V3 corrige los hallazgos: locales de carga única, igualdad
estructural, retorno `(finCalculado, ok)`, cotejo sin sustitución y autoridad
sellada inmune a una restauración sin efecto.

No quedan hallazgos P0, P1 ni P2. El GO solo acredita el contrato documental.
No autoriza implementar P3: primero debe cerrarse
`O4A-P1E-SELLOS-DEADLINE` con doble GO y CI 5/5 sobre su SHA exacto.

# Revisión independiente O1b: ledger correctivo 790

Fecha: 5 de agosto de 2026.

Estado: **GO documental para corregir exclusivamente O1b**.

Base material: `fac31c939c30715d30bd5273a9ab578c27fb4e16`.

Documentos revisados: `e5e69e8` y corrección `fb9e966`.

## Trazabilidad

| Corte | Funcional/seguridad | Ledger/reproducibilidad | Resultado |
| --- | --- | --- | --- |
| `e5e69e8` | NO-GO: P0=0, P1=1, P2=1 | GO: P0=P1=P2=0 | NO-GO |
| `fb9e966` | GO: P0=P1=P2=0 | GO: P0=P1=P2=0 | GO final |

El primer corte conservaba un total global correcto, pero presentaba dos
subrangos como si fueran sumables y no declaraba con precisión qué parte del
ledger original sustituiría. La corrección deja un único presupuesto global
vinculante y limita la sustitución a las tablas de delta/totales y a los
umbrales `+265/680`.

## Evidencia física congelada

El candidato continúa sin commit en
`.worktrees/f0-h0b-c4b2-o1b-20260805`, con HEAD `fac31c9`. Solo G2 está
modificado: 678 líneas, delta `+278` y SHA-256
`d44f0cc3c011d09d95dc1f1b56f69382c61bfac45de12376feb0dee0788568f5`.
Runner, G1, capturador, adaptador, D2d, D2c y H0b permanecen byte a byte
invariantes. El runner conserva correctamente las huellas del último estado
aceptado y `--supervisar-m38` continúa cerrado en 64.

## Presupuesto aceptado

```text
candidato:                400 + 278 = 678
corrección restante:      +35..+105
delta final previsto:     +313..+383
G2 final previsto:        713..783
parada de revisión:       +390 = 790
tope absoluto DEC-051:    800
```

La parada 790 no es un objetivo de consumo. Si a esa altura falta un vector,
se detiene de nuevo. No se minifican o retiran pruebas y el margen no se
transfiere a O2.

## Alcance del GO

Se autoriza corregir el mismo árbol material:

1. restaurar la prevalidación O1a y enganchar la autoprueba O1b una sola vez;
2. completar la matriz contractual indicada por el ledger;
3. mantener G2 en 790 líneas o menos;
4. estabilizar G2 y solo entonces sustituir juntos los dos literales de huella
   autorizados del runner;
5. ejecutar las puertas fijadas y obtener doble revisión independiente del
   código.

No se acepta todavía el candidato, no se autoriza O2 y no cambian F0 `10/23`,
O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14` ni el
`NO-GO` de producción.

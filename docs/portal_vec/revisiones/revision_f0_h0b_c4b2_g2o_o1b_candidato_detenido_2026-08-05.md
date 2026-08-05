# Revisión O1b: primer candidato detenido por ledger

Fecha: 5 de agosto de 2026.

Estado: **NO-GO; árbol sin commit y sin integración**.

Base: `fac31c939c30715d30bd5273a9ab578c27fb4e16`.

## Evidencia física

| Unidad | Líneas | SHA-256 abreviado | Resultado |
| --- | ---: | --- | --- |
| Runner | 800 | `5ce51623…f1153` | invariante; huellas anteriores |
| G1 | 686 | `9fab2cae…e1afe` | invariante |
| G2 base | 400 | `20980b27…ab04f5` | base aceptada |
| G2 detenido | 678 | `d44f0cc3…568f5` | +278, fuera del ledger |
| Capturador | 799 | `4a967fd1…78902` | invariante |
| Adaptador | 527 | `98d22a30…a8cb7` | invariante |
| D2d | 145 | `9b137f13…2c5e81` | invariante |
| D2c | 588 | `a07057fb…badde5` | invariante |
| H0b | 580 | `02a00f2f…bafded` | invariante |

`git diff --numstat` mostró únicamente `278 0` en G2. No hay séptima fuente,
operación real, FD, proceso, señal, red, SQL o dependencia nueva. El modo
`--supervisar-m38` permanece cerrado en 64 por inspección.

## Veredictos

| Revisión | P0 | P1 | P2 | Resultado |
| --- | ---: | ---: | ---: | --- |
| Productor | 0 | parada física | matriz incompleta | NO-GO |
| Funcional y seguridad | 0 | 3 grupos | 1 grupo | NO-GO |
| Ledger y cobertura | 0 | 4 | 2 | NO-GO |

Los hallazgos materiales son:

- delta +278 frente al máximo +265;
- autoprueba conectada indebidamente a la prevalidación del encoder;
- matriz aún incompleta y algunos controles vacuos;
- construcción ignorada y orden de muestras no determinista;
- runner deliberadamente sin las huellas de un candidato no aceptado.

`gofmt` y `git diff --check` fueron verdes. No existe evidencia válida de
`go vet`, build privado, reproducibilidad, autoprueba mediante runner, Docker,
PostgreSQL, red, E2E o puertas globales para este estado.

## Continuación autorizada, sin aceptación del candidato

La
[enmienda de ledger 790](../enmienda_f0_h0b_c4b2_g2o_o1b_ledger_correctivo_790_2026-08-05.md)
obtuvo doble GO final en `fb9e966`. Debe mantenerse el mismo worktree sin
commit, corregir primero el enganche, completar la matriz, actualizar al final
solo dos huellas del runner y obtener doble revisión independiente. El estado
material descrito por esta acta continúa en NO-GO hasta superar esas puertas.

El NO-GO no modifica métricas ni abre O2.

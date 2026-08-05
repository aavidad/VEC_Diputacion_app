# Revisión O1b: candidato 56c0ac0

Fecha: 5 de agosto de 2026.

Estado: **NO-GO para integrar**.

Commit: `56c0ac079419b187f851de8183d5f87b5b367b71`.

## Veredictos

| Revisión | P0 | P1 | P2 | Resultado |
| --- | ---: | ---: | ---: | --- |
| Ledger y reproducibilidad | 0 | 0 | 0 | GO |
| Funcional y seguridad | 0 | 3 grupos | 1 grupo | NO-GO |

Un solo NO-GO impide integrar. El commit permanece limpio en la rama
`candidate/f0-h0b-c4b2-o1b-20260805` y no forma parte de la integradora.

## Evidencia positiva

- G2 788 líneas, delta `+388`, SHA-256
  `669ca019cbdd259a21b1f7c38794df983e510863bbe9ddbdebccd8a9f4a33406`;
- runner 800 líneas y cambio exclusivo de dos literales;
- G1 686 e invariantes byte a byte;
- seis fuentes, dos builds privados reproducibles y SHA binaria
  `99eeeb61349e7971d79755e7198a8aa147ddce90df8db0e3055ca000ce04d61d`;
- `gofmt`, `go vet`, autoprueba, modos 64, Bash, ShellCheck, tamaños,
  `git diff --check` y Gitleaks focal verdes;
- no aparecen superficies, efectos, dependencias u O2.

La inspección no encontró un defecto funcional directo en el lector. El
bloqueo es probatorio: tres mutantes incorrectos continúan verdes.

## Hallazgos bloqueantes

1. La matriz funcional construye fixtures directamente y la prueba de la API
   real no verifica clase, L0, limpieza o error interno. Un constructor que
   devuelve siempre CONTROL queda verde.
2. Las entregas comparan principalmente clase. Vaciar `campos` y `ticket` en
   monoframas queda verde.
3. L1, varios éxitos L0 y los errores L4 no se afirman de forma exacta. Una
   transición L1→L0 incorrecta queda verde.
4. Faltan monoframa inválida retenida en L2 y rechazada al EOF, dato posterior
   L2 con `fin=true` y precedencia de cola sobre O1a en la misma llamada.

## Continuación

No se corrige código hasta revisar la
[enmienda de parada 800](../enmienda_f0_h0b_c4b2_g2o_o1b_correccion_final_800_2026-08-05.md).
Si esa corrección no cabe con legibilidad y diagnósticos completos, se diseña
G3; no se contamina G1 ni se excede DEC-051.

No se ejecutaron puertas globales, Docker, PostgreSQL, red o E2E porque el
NO-GO focal detuvo la integración.

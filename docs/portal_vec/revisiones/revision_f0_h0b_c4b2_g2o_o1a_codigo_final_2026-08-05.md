# Revisión final F0-H0b/C4b-2 G2-O/O1a

Fecha: 5 de agosto de 2026.

Estado: **GO e integración local completada**.

## Trazabilidad

| Etapa | Commit candidato | Resultado |
| --- | --- | --- |
| Codec inicial | `b250d38` | NO-GO funcional; ledger físico verde |
| Cobertura contractual | `dcf4383` | NO-GO por fronteras vacuas |
| Centinelas de límite | `ce549a6` | NO-GO por mutante hex incompleto |
| Mutante hexadecimal | `66ebd411` | triple GO final |
| Integración | `dd029ad`–`52c8852` | verde |

Los NO-GO no se ocultaron ni se integraron de forma aislada. La secuencia
completa fue revisada y después aplicada sobre la rama integradora.

## Resultado final

- runner: 800 líneas;
- G1: 686 líneas, fuente y literal estables durante las correcciones;
- G2: 400 líneas, delta final +101 sobre la primera candidata;
- capturador, adaptador M38, D2d, D2c y H0b: byte a byte invariantes;
- modo `--supervisar-m38`: 64 sin efectos;
- dos builds privados aislados: SHA idéntico al literal del runner;
- gramáticas `SOBRE`, `CONTROL`, `TERMINAL` y ticket completas;
- causas, estados, fases S1..S4 y bloques con/sin Bash coherentes;
- límites físicos con centinelas distintos y fronteras -1/exacto/+1;
- máximos, overflow, cardinalidad, CR, TAB, NUL, no ASCII, mayúscula,
  longitud y alfabeto hexadecimal adverso cubiertos;
- prevalidación del encoder anterior a cualquier concatenación;
- control vacío rechazado sin pánico.

## Revisiones finales

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Seguridad | 0 | 0 | 0 | GO |
| Máquina y cobertura | 0 | 0 | 0 | GO |
| Ledger y reproducibilidad | 0 | 0 | 0 | GO |

## Puertas reproducidas en la integradora

- `bash -n`, ShellCheck, `gofmt` y `git diff --check`;
- `go test ./...`;
- `go test -race ./...`;
- `go vet ./...`;
- `scripts/verificar_calidad.sh`, incluida carrera repetida, grafos de
  dependencias, manifiestos aislados, vulnerabilidades y tamaños.

Todas terminaron verdes. La carrera de `internal/vec/ports` fue costosa pero
terminó correctamente; no quedó `ports.test` en ejecución.

No se ejecutaron Docker, PostgreSQL, red o E2E del supervisor porque O1a es un
codec puro y mantiene el modo operativo cerrado. Esas pruebas corresponden a
las fases que materialicen proceso y puente.

## Alcance que permanece abierto

O1a no cierra C4b-2, C4b, H0b, C2, F0, O4-05 o producción y no modifica
métricas. La siguiente minitarea es O1b: lector incremental acotado,
fragmentación, coalescencia y sobrante. Requiere contrato, ledger y doble GO
antes de código. O2 y fases posteriores permanecen sin autorizar.

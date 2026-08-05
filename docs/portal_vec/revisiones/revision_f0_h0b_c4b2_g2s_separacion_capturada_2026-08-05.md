# Revisión F0-H0b/C4b-2 G2-S: separación capturada

Fecha: 5 de agosto de 2026.

## Dictamen

Estado: **GO documental**.

```text
P0=0
P1=0
P2=0
```

Dos revisores independientes contrastaron la segunda propuesta G2-S de
`f732451` con el código real, Q5a, G1 y la decisión C4b-2. No modificaron
archivos ni ejecutaron Docker, PostgreSQL o puertas globales.

## Evidencia contrastada

- runner 789, supervisor G1 754, capturador 799 y adaptador 527 líneas;
- ledger máximo del runner `789 + 8 = 797`, sin minificar controles;
- capturador invariante `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902`;
- lista lexicográfica exacta de seis y rechazo de duplicados;
- build explícito y cerrado de dos Go privados;
- paquete ordinario con capturador en `GoFiles` y dos supervisores en
  `IgnoredGoFiles`;
- una sola `main` por composición admitida y prohibición de `-tags=ignore`;
- cierre `--supervisar-m38 → 64` sin FD, procesos o señales;
- traslado posible de primitivas Linux compartidas bajo el máximo G2 de 160;
- adaptador, D2c, D2d y H0b byte a byte invariantes;
- Q5a y G1 conservan su alcance histórico de cinco fuentes.

## Alcance autorizado

Se autoriza únicamente G2-S con el write-set fijado en la enmienda: runner,
fuente G1 y nueva fuente G2, más pruebas y documentación propias. La
implementación debe demostrar físicamente manifiesto, huellas, dos builds,
presupuesto y regresión G1 100/100 antes de revisión independiente.

No se autorizan todavía canales, FD 3..9 de caso, Bash operativo, plazos,
recibos, Docker, PostgreSQL, red o G2-O. G2-S no cierra C4b-2, C4b, H0b, C2,
F0, O4-05 ni producción.

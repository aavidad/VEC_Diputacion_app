# Revisión F0-H0b/C4b-2 G2-S: implementación

Fecha: 5 de agosto de 2026.

## Dictamen final

Estado: **GO**.

```text
P0=0
P1=0
P2=0
```

Código exacto revisado:

```text
ccd72365a86b9afe623806e3f4bebd511ec13830
dd01b44449ec3953e9706991ecfd59751337a78e
b3c32ba1785ab49182f6926661ef54a9925fac6f
```

Autoridad documental: `2409620`, con el ledger físico de 800 líneas. Dos
revisores distintos del productor reprodujeron el corte y emitieron `GO` final
sin hallazgos abiertos.

## NO-GO corregidos

El primer candidato recibió doble `NO-GO` porque los dos builds compartían
`HOME`, `TMPDIR` y `GOCACHE`, el runner excedía el ledger previsto y dos
controles se habían condensado. `dd01b444` introdujo raíces y cachés privadas,
restauró el formato legible y dejó el runner en 800 líneas.

La primera revisión de esa corrección detectó que la reacreditación post-build
había perdido la ruta física `/usr/bin/sha256sum`. `b3c32ba` la restableció en
el mismo bucle `antes/despues`, cotejando la salida completa `SHA  ruta` sin
aumentar el fichero.

## Evidencia reproducida

- runner 800, G1 683, G2 91, capturador 799 y adaptador M38 527 líneas;
- manifiesto privado exacto de seis fuentes y clasificación ordinaria de un
  `GoFiles` y dos `IgnoredGoFiles`;
- dos raíces e inodos distintos, no simbólicos y `0700`, con `HOME`, `TMPDIR`
  y `GOCACHE` disjuntos;
- cachés nuevas y compilación forzada en ambos builds;
- binarios regulares `0700`, de un enlace y SHA-256 idéntica
  `8b265176f7acb8f1e3210776a81e541070613505ce12ba957d6fc4279b6e4740`;
- G1 100/100 y `--supervisar-m38` 100/100 con estado 64;
- desconocido y ayudante inválido 64, FD invariantes, hijos y procesos
  residuales cero;
- rechazo de 5/7 entradas, ausencia, duplicado, ruta, SHA, seis copias y ambos
  binarios mutados;
- entorno y `PATH` hostiles sin capacidad para sesgar SHA o toolchain;
- capturador, adaptador, D2c, D2d y H0b byte a byte invariantes;
- `gofmt`, `go vet`, builds explícitos, Bash, ShellCheck y
  `git diff --check` verdes.

La revisión no ejecutó Docker, PostgreSQL, red, SQL ni E2E porque G2-S solo
prepara la separación capturada. Dirección ejecutará las puertas globales
sobre el árbol integrado antes de publicar.

## Alcance

G2-S conserva un cierre seguro: `supervisarM38()` solo devuelve 64. No
implementa G2-O, no cierra C4b-2 ni cambia las métricas. El runner queda sin
reserva de líneas; cualquier trabajo posterior que necesite modificarlo exige
una separación o topología aprobada previamente.

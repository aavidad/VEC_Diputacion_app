# Revisión F0-H0b Q5a: enmienda de arranque auditable

Fecha: 4 de agosto de 2026.

## Resultado

**GO documental** sobre `4c0f3b1`, confirmado por dos contrarrevisiones
independientes con `P0=0`, `P1=0` y `P2=0`.

El documento aceptado es
[`enmienda_f0_h0b_q5a_arranque_auditable_2026-08-04.md`](../enmienda_f0_h0b_q5a_arranque_auditable_2026-08-04.md).
Autoriza únicamente la refactorización runner/D2d y la corrección posterior de
arranque, adaptador, formato y pruebas Q5a. No acredita todavía ese código.

## Historial de parada

La primera propuesta, `935bb28`, obtuvo `GO` de capacidad y dos `NO-GO P1`.
Las sondas con `bash -p -x` todavía podían ejecutar un `PS4` heredado antes de
la puerta. Además, el estado global de `env | grep` confundía el productor
fallido `(1,1)` con la ausencia válida `(0,1)`. También faltaban la frontera
del cargador, redirección de la salida cruda y el mutante `ENV`.

Dirección no autorizó código en ese estado. `4c0f3b1` añadió el entorno vacío
de las sondas, la pareja completa `PIPESTATUS`, salida nula, rechazo de
`BASH_FUNC_*` y `LD_*`, marcadores `BASH_ENV`/`ENV` y el límite explícito del
launcher, EUID y cargador.

## Contrarrevisión de seguridad

Resultado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Reprodujo:

- entorno limpio: `PIPESTATUS=(0 1)` y único avance permitido;
- función exportada: `(0 0)`, estado 65 y cuerpo no ejecutado;
- productor o consumidor fallido: estado 65, sin falso limpio;
- `LD_*`: estado 65;
- `grep` sin `-q`, consumo completo y salida dirigida a `/dev/null`;
- `PS4`, `SHELLOPTS` y `BASHOPTS` adversos bajo el launcher de las sondas:
  prefijo literal `+ ` y marcador ausente;
- `BASH_ENV` y `ENV` ignorados por Bash `-p`.

Confirmó que el documento no atribuye al runner la capacidad imposible de
revertir una biblioteca cargada antes de Bash. La confianza en launcher,
EUID, kernel, cargador y binarios queda explícita.

## Contrarrevisión ABI y Bash

Resultado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Reprodujo el shebang protegido, la invocación por descriptor retirado, FD 6
como único destino de traza y el orden futuro:

```text
[/usr/bin/bash, -p, /proc/self/fd/8, --caso-inyeccion-h0b, selector]
```

Confirmó que el entorno vacío elimina `PS4`, `SHELLOPTS`, `BASHOPTS`,
`BASH_ENV` y funciones; que `p` permanece activa; y que la fuente Go, sus
syscalls 424/434 y el flag 4 no cambian.

## Capacidad y decisión

La revisión de capacidad previa dio `GO`, `P0=0`, `P1=0`, `P2=0`:

- el traslado literal retira 47 líneas del runner;
- D2d queda aproximadamente en 143..146 líneas;
- el reflujo proyecta 776..784 líneas de runner, bajo el stop local 790;
- siguen existiendo exactamente cinco fuentes;
- D2d conserva la autoridad de operaciones y pruebas de snapshot/contenedor;
- C4b-2, C4b-3 y C4c mantienen reserva hasta DEC-051.

Dirección acepta `4c0f3b1`. El productor puede ejecutar, en secuencia:

1. traslado sin cambio semántico de las tres funciones de snapshot a D2d;
2. arranque protegido, huellas nuevas, reflujo y mutantes Q5a.

El mismo candidato final requiere nuevas revisiones independientes de código.
No se incrementan contadores de capacidad ni cambia el `NO-GO` de producción.

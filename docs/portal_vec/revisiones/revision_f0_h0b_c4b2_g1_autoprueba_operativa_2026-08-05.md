# Revisión F0-H0b/C4b-2 G1: autoprueba operativa

Fecha: 5 de agosto de 2026.

## Dictamen

Estado final: **GO**.

```text
P0=0
P1=0
P2=0
```

Árbol revisado: `233e9bd`, compuesto por el candidato `4fdeddb` y su
corrección `233e9bd`.

El primer pase independiente sobre `4fdeddb` emitió doble `NO-GO`, con
`P0=0`, `P1=1` y `P2=0`. Si la recepción `SCM_RIGHTS` fallaba antes de
registrar el pidfd del descendiente, la limpieza podía matar solo al líder y
dejar vivo al adoptado desconocido.

La corrección:

1. marca cada líder únicamente después de un `Start` exitoso;
2. extingue el grupo mediante el pidfd del líder y
   `PIDFD_SIGNAL_PROCESS_GROUP`;
3. conserva esa autoridad hasta drenar adoptados a `ECHILD` y acreditar
   `ESRCH`;
4. cierra los pidfds después de esas postcondiciones;
5. añade un mutante `SOCK_SEQPACKET` truncado que recibe y cierra el
   `SCM_RIGHTS`, pero no registra el pidfd del descendiente.

Dos revisores distintos reprodujeron la corrección y emitieron `GO` final sin
hallazgos abiertos.

## Evidencia reproducida

| Evidencia | Resultado |
| --- | --- |
| Fuente Go | 754 líneas; `c024ab13362bc6953028e185ab25a5c50f3a158efa444e9f407727e57d720b2f` |
| Binario Go 1.26.5 | dos builds iguales; `2b28a395956775aeebcbb5807cb89f00d3c9451db533304216077308a6e2c99b` |
| Autoprueba de dirección | `100/100` verde |
| Autopruebas de revisores | `50/50` por revisor, incluida variante `GOMAXPROCS=1` |
| CLI desconocida/ayudante sin capacidad | estado `64` |
| Residuos | cero procesos, hijos, grupos y FD adicionales |
| APIs prohibidas | cero coincidencias; un solo `cmd.Wait()` |
| Runner | 789 líneas, Bash y ShellCheck verdes |
| Globales | `go test ./...`, `go test -race ./...`, `go vet ./...` verdes |
| Puerta completa | `scripts/verificar_calidad.sh` verde |

La puerta completa acreditó además grafos de dependencias, manifiestos web,
TLS, pruebas documentales, ausencia de vulnerabilidades conocidas y límites
de tamaño.

No se ejecutaron Docker, PostgreSQL, red ni E2E en G1, porque no existen aún
recursos de caso ni SQL en este corte.

## Alcance no cerrado

G1 acredita solo las primitivas y su rollback sintético. No cierra C4b-2,
C4b, H0b, C2, F0, O4-05 ni producción. G2 debe incorporar la máquina
operativa y el protocolo con runner/adaptador. Como el supervisor ocupa 754
líneas, G2 necesita antes una decisión explícita de separación de
responsabilidades compatible con la captura privada; no se autoriza superar
800 líneas ni minificar controles.

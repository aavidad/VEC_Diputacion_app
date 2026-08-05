# Revisión independiente del candidato G2-O/O1a `b250d38`

Fecha: 5 de agosto de 2026.

Estado: **NO-GO de código**. El candidato no se integra.

Commit candidato: `b250d38aadd165aa80471dbbf1200d56bd4165bb`.

## Veredictos

| Revisión | P0 | P1 | P2 | Resultado |
| --- | ---: | ---: | ---: | --- |
| Ledger y puertas | 0 | 0 | 0 | GO físico |
| Máquina y cobertura | 0 | 2 | 0 | NO-GO |
| Seguridad | 0 | 2 | 1 | NO-GO |

## Hechos conformes

- write-set exacto R/G1/G2;
- R 800, G1 686 y G2 299 líneas;
- R cambia solo los tres SHA; capturador, adaptador, D2d, D2c y H0b son
  invariantes;
- dos builds aislados producen el mismo binario;
- `gofmt`, `go vet`, autoprueba, Bash, ShellCheck y `git diff --check` verdes;
- `--supervisar-m38` y modo desconocido devuelven 64;
- no hay FD, proceso operativo, Docker, PostgreSQL, SQL o red nuevos.

## Hallazgos bloqueantes

1. `terminalM38` acepta un bloque sin Bash con `FASE_ORIGEN=S4`, combinación
   prohibida porque S4 implica ejecución;
2. la autoprueba contiene solo seis tramas válidas y diez inválidas y no cubre
   máximos, overflow, cardinalidades, versión/tipo, selectores, bytes adversos,
   todas las parejas causa/estado ni cruces completos;
3. las supuestas fronteras construyen entradas siempre inválidas y, por ello,
   no demuestran el límite que pretenden probar;
4. el encoder concatena antes de validar cardinalidad/tamaño y puede reservar
   memoria no acotada desde una estructura interna sobredimensionada.

## Bloqueo físico de la corrección

G2 ocupa 299 de las 320 líneas autorizadas. La corrección productiva legible
requiere unas 15..20 líneas y la matriz tabular unas 45..60 adicionales. No
cabe en 21 sin minificar, omitir pruebas o superar el ledger.

El productor dejó el worktree limpio en `b250d38` y no creó corrección parcial.
Antes de editar se necesita una enmienda de presupuesto revisada. El candidato
continúa NO-GO aunque esa enmienda sea aceptada: después deberá corregirse y
recibir nueva revisión independiente.

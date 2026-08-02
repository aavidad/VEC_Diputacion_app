# Revisión F0-H0b C4b-1: señales y régimen shell

Fecha: 2 de agosto de 2026.

## Resultado

El checkpoint C4b-1 finaliza en `ffce19c` con dos revisiones independientes
finales `GO` y `P0=0`, `P1=0`, `P2=0` dentro de su alcance.

C4b-1 acredita conservación y restauración del régimen shell, señales
`INT`/`TERM` diferidas, causal única, espera resistente a interrupciones, H0
nominal PostgreSQL 18.4 y residuos cero. No acredita C4b-2, C4b-3, C4b
completo, C4c, C4d, H0b global, C2, F0 ni producción.

## Cadena revisada

La [decisión de semántica de señales](../decision_f0_h0b_c4b1_semantica_senales_2026-08-02.md)
queda integrada en `fb45b93`. Sobre ella se aplican tres commits acotados:

1. `84de42f`: régimen shell, señales diferidas y espera común;
2. `ee769c3`: conservación de la primera causal observada;
3. `ffce19c`: bloqueo de reentrada mediante el marco completo del manejador.

Los commits `db240a5`, `075610f` y `1524feb` son candidatos `NO-GO` de ramas
previas, no antepasados que se oculten como aceptados. Sus parches se
reaplicaron sobre la decisión como `84de42f`, `ee769c3` y `ffce19c`; los
`patch-id` estables coinciden por parejas. La revisión final evalúa el árbol
combinado actual, no hereda el resultado de las ramas rechazadas. `fb45b93` es
a su vez equivalente por `patch-id` a la decisión documental original
`cd193ef`.

El write-set funcional acumulado contiene exactamente:

- runner PostgreSQL ContextoActor V3;
- adaptador privado de recursos M38/H0b.

H0b, D2c, D2d y el capturador Go permanecen byte a byte inmutables.

## Hallazgos y correcciones

El primer candidato difería `INT` y `TERM`, pero una señal posterior podía
sustituir la causal pendiente. La primera corrección enclavó esa causal; la
revisión de seguridad reprodujo todavía una reentrada antes de la primera
orden del manejador y obtuvo 143 para `INT→TERM` y 130 para `TERM→INT`.

La corrección final coloca como primer constructo una guardia que inspecciona
toda `FUNCNAME`, no solo `FUNCNAME[1]`. Un manejador anidado retorna sin tocar
causal, generación, efectos ni trabajos, incluso con marcos intermedios
`DEBUG`, función auxiliar, `ERR` o `RETURN`. Cada `trap` conserva una sola
llamada directa al manejador.

La decisión `fb45b93` elimina además una promesa que las señales estándar no
pueden sostener: no se afirma que gane el primer `kill`. Se enclava la primera
señal entregada y observada al iniciarse un manejador. Una ráfaga anterior a
ese marco acepta exactamente uno de `{130, 143}`.

## Garantías funcionales y de seguridad

Las dos revisiones finales aceptaron:

- tabla de trabajos inicialmente vacía y rechazo de régimen ya ocupado;
- conservación de `monitor` y del atributo de exportación de `SHELLOPTS`;
- restauración exacta de sus cuatro combinaciones de entrada;
- `INT` única = 130 y `TERM` única = 143;
- primera causal observada inmutable en reentradas directas y mediadas;
- cuatro fronteras de drenaje del finalizador con causal estable;
- una generación que despierta esperas sin abrir efectos ni trabajos;
- `wait -n -f` y `wait -f` resistentes a interrupciones;
- plazo del cliente, terminación del reloj y bloqueo de un cliente nuevo tras
  quedar una señal pendiente;
- limpieza que libera el enclavamiento, restaura régimen y converge una sola
  vez;
- cero autoridad causal derivada del orden de `kill` o del planificador.

## Campaña de 600 señales y ráfagas

La revisión de seguridad repitió cien veces cada vector:

| Vector | Estado 130 | Estado 143 | Total |
| --- | ---: | ---: | ---: |
| `INT` única | 100 | 0 | 100 |
| `TERM` única | 0 | 100 | 100 |
| `INT→TERM` | 82 | 18 | 100 |
| `TERM→INT` | 81 | 19 | 100 |
| `INT→TERM→INT` | 80 | 20 | 100 |
| `TERM→INT→TERM` | 83 | 17 | 100 |
| **Total** | **426** | **174** | **600** |

La distribución 426/174 no es oráculo, proporción esperada ni prioridad. Solo
evidencia que todas las ejecuciones pertenecen a `{130, 143}`, producen una
única causal y convergen sin efectos o trabajos nuevos. Los dos vectores de señal
única sí conservan obligatoriamente sus estados exactos.

## Presupuestos y huellas

| Fichero | Líneas | Límite aplicable |
| --- | ---: | ---: |
| Runner | 769 | 775 |
| Auxiliar H0b | 580 | 580 |
| Adaptador privado | 527 | 540 al cerrar C4b |

Huellas verificadas:

| Artefacto | SHA-256 |
| --- | --- |
| D2c | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| Adaptador | `d9b61a183e5a32c321a3eeb48483ce40c83551bc7a700354ccc88e8206d9ee1f` |
| D2d | `8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075` |
| Capturador | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |

## Pruebas finales

Sobre el HEAD exacto `ffce19cde39e8b04752529c280ba6f0669390783`
quedaron verdes:

- matriz funcional de régimen, señales, cuatro drenajes, cliente bloqueado,
  esperas interrumpidas, timeout, limpieza y restauración;
- matriz de seguridad directa y mediada, ambos órdenes y tres señales;
- 600 ejecuciones de señales y ráfagas con la distribución anterior;
- `bash -n` y ShellCheck `-x` sobre los cinco scripts acreditados;
- ejecución directa de los cuatro auxiliares: estado 64;
- captura canónica exacta de cuatro y SHA-256 literal del adaptador;
- invariancia byte a byte de H0b, D2c, D2d y capturador;
- `git diff --check`, `git show --check` y Gitleaks focal/acumulado;
- H0 nominal mediante `--etapa H0`, imagen PostgreSQL 18.4 fijada por digest:
  estado 0 en 21,362 segundos.

Después de H0 se acreditaron cero contenedores por nombre y etiqueta propios,
cero rutas `/tmp/vec-f0-h0-*` y cero procesos del runner o con prefijo propio.
Los worktrees de productor y revisores permanecieron limpios.

## Pendientes expresos

C4b-1 no corrige ni oculta:

- C4b-2: trabajo provisional, cardinalidad del fork, PID/PGID/PPID, tiempo de
  inicio, plazo absoluto y extinción completa del grupo;
- C4b-3: Docker, temporal, reconciliación del daemon y epílogo único;
- C4c/C4d, H0b global, C2 y el resto de F0.

## Estado

C4b-1 puede conservarse como checkpoint no productivo integrado localmente.
El siguiente corte exacto es C4b-2. Las métricas no aumentan: F0 `10/23`,
O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14`;
producción continúa en `NO-GO`.

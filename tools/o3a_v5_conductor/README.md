# Diseño del conductor durable externo O3a V5 (P1-F01)

Estado: **CND-V5-11 implementado, reproducido y propuesto como autoridad CI
de O3a V5 mediante la enmienda sin R; pendiente doble revisión de esa
composición**.

## Ejecución durable única

`conductor.sh` es la única entrada del conjunto. Falla cerrado salvo con Go
1.26.5 y las diez fuentes exactas declaradas en `fuentes_v5.tsv`. Copia esas
fuentes a un directorio privado, comprueba `gofmt` en modo solo lectura y
ejecuta los siete bloques primero en modo normal y después con un build real
`-race` (`CND_RACE=1`, `CGO_ENABLED=1`). El target nunca se formatea ni se
modifica.

```text
./conductor.sh RUTA_TARGET DIRECTORIO_EVIDENCIA_VACIO
```

Cada bloque tiene watchdog exterior. La salida durable contiene el ledger de
fuentes, manifiesto de catorce ejecuciones con SHA de cada script, duración,
estado y número de filas, un registro NDJSON derivado de cada TSV, logs,
residuos observables, resumen derivado y `SHA256SUMS`. Un `SKIP`, una fila
`NO-GO`, una salida vacía, un timeout, un residuo o una divergencia de huella
hace fallar toda la ejecución. Para congelar otra autoridad se debe revisar y
reemplazar explícitamente `fuentes_v5.tsv`; el conductor nunca aprende hashes
del target que está juzgando.

Este directorio contiene la herramienta de verificación integrable, reproducible
y fail-closed de O3a V5. En desarrollo puede juzgar una proyección externa; en
CI recibe la raíz del checkout como target. No modifica las fuentes, no toca R
y no arranca el modo operativo de Orquesta.

La reproducción congelada vigente es
`evidencia-cnd-v5-11-final-r3`: 14/14 bloques, 74 registros NDJSON, modos
normal y `-race` real, FD del conductor 5→5 y residuos cero. C20 distingue en
una columna durable las variantes `canonica` y `m18_sin_pdeathsig`. El resumen
y `SHA256SUMS` se derivan dentro de ese directorio; una corrida nueva debe usar
otro directorio inicialmente inexistente o vacío.

La composición propuesta en `.github/workflows/ci.yml` fija este directorio
como `working-directory` y ejecuta
`./conductor.sh "$GITHUB_WORKSPACE" "$RUNNER_TEMP/evidencia-o3a-v5"` después
de la puerta de calidad. Así, un cambio en cualquiera de las diez fuentes, un
caso omitido, un build race falso o un residuo hace fallar la misma puerta CI.
El workflow publica en su propio log el resumen, manifiesto, TSV y logs de cada
bloque, conservando sin traducción el estado del conductor. El plazo test-only
para observar la salida nominal es diez segundos y no altera los tres segundos
productivos de retirada. C21 compara descriptores todavía vivos mediante
`F_GETFD`: una entrada obsoleta de `/proc/self/fd` solo se omite si devuelve
`EBADF`; cualquier otro error o descriptor vivo adicional mantiene el NO-GO.
Los estados test-only 77--98 distinguen preparación —incluidas sus causas
entrada, subreaper, `Pdeathsig`, inventario y forma FD—, mapa FD, avance,
identidad, limpieza, retirada y residuos; nunca son estados GO ni alteran las
salidas contractuales 65 y 72--76. `C08_BARRIDO` espera 79 porque su mutación
de barrido debe ser rechazada durante la preparación de la fixture.
R conserva su autoridad histórica y sus nueve entradas para cortes anteriores;
no es autoridad de O3a V5.

## Genealogía de diseño ya superada

Las secciones siguientes conservan el razonamiento incremental que llevó al
conductor único. Describen carencias y propuestas de cortes anteriores; no son
el estado vigente ni sustituyen `conductor.sh`, `fuentes_v5.tsv` y la evidencia
final indicada arriba.

## Hecho observado

El ejecutable proyectado solo admite `--autoprueba`. La raíz llama a
`autoprobarArranqueO3aM38`, que ejecuta una matriz agregada, pero no ofrece una
frontera exterior para:

- escoger por separado las tuplas A, B y C devueltas por `Start`;
- provocar de modo determinista el error de `F_DUPFD_CLOEXEC` inmediatamente
  posterior a `Start`;
- fijar el mapa low-hole de 0..9 y el bundle alto antes de cada caso;
- ejecutar los dos procesos del oráculo `Pdeathsig` (muere el hilo creador / no
  muere otro hilo) y observarlos desde un abuelo;
- ejecutar AF de forma aislada y acreditar simultáneamente estado 65, EOF y
  stdout/stderr vacíos;
- imponer eventos simultáneos y acreditar la precedencia CONTROL > terminalidad;
- seleccionar y registrar individualmente los casos contractuales 1..21.

Sin una frontera completa, un script exterior podría acreditar únicamente el resultado
agregado `autoprueba=ok`; atribuirle los oráculos anteriores sería evidencia
inventada. Forzar fallos con agotamiento global de procesos/FD o temporización
del planificador tampoco es reproducible.

## Instrumentación mínima CND-V5-01

La proyección externa incorpora ahora una entrada cerrada
`--autoprueba-o3a-caso <ID>` en G1 y el despachador en G7b. Admite únicamente:

```text
DUPFD_FALLO
AF_DIRECTO
AF_PIDFD_PRIMARIO_CERRADO
AF_PIDFD_RESERVA_CERRADA
AF_PIDFD_AMBOS_CERRADOS
```

El primer caso acredita que el syscall real `F_DUPFD_CLOEXEC` falla con
`EBADF`; los restantes se ejecutan en subprocesos y acreditan AF mediante
estado 65, EOF y stdout/stderr vacíos. Los tres casos pidfd arrancan una fixture
real, degradan una o ambas referencias y recorren barrera/retirada hasta AF.
Un ID ajeno termina en 64 sin E/S. Esto hace observables las primitivas, pero no
equivale aún al oráculo completo de C10 ni sustituye la futura inyección causal
inmediatamente posterior a `Start`.

Ejecutar `./conductor_minimo.sh RUTA_BINARIO` para producir TSV durable.

## Cambio restante de fuente requerido

La instrumentación debe vivir solo en G7b
`supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go`,
dominada por `--autoprueba`, sin ruta operativa, `init`, global mutable,
callback, hook o mock. G6a-c y G1 no necesitan cambiar de contrato.

G7b necesita un despachador test-only, consumido desde
`autoprobarArranqueO3aM38`, con entrada cerrada y validada, por ejemplo:

```text
VEC_O3A_CASO_EXTERNO=<id>
VEC_O3A_RESULTADO_FD=<fd >= 10, O_CLOEXEC>
```

El descriptor de resultado debe recibirse ya abierto por el conductor, no
abrirse por ruta, y emitir una sola línea ASCII acotada (máximo 1024 bytes) tras
cerrar/recoger todos los recursos del caso. La entrada debe aceptar únicamente
IDs literales registrados; presencia de cualquier variable desconocida,
descriptor inválido o ejecución fuera de `--autoprueba` termina en 64 antes de
crear procesos.

IDs mínimos:

```text
C01..C21
TUPLA_A TUPLA_B TUPLA_C
DUPFD_FALLO LOW_HOLE
PDEATH_CREADOR PDEATH_OTRO
AF_AMBOS_PIDFD_CERRADOS
PREC_CANCELAR_TERMINAL PREC_EOF_TERMINAL
```

Para tuplas no producibles por `os/exec` real, G7b no debe alterar G6c ni
inyectar retornos. Debe lanzar como subproceso una **variante compilada
test-only** generada por transformación literal de cardinalidad uno sobre una
copia externa de G6c; el conductor registra patrón anterior/posterior, diff y
SHA. La variante conserva una única llamada a `Start` y modifica solo la
clasificación/resultado inmediatamente posterior. Este mecanismo coincide con
la disciplina M001..MN y evita hooks en el grafo productivo.

G7b tiene margen contractual hasta 700 líneas (539 en el checkpoint); aun así,
el despachador y sus oráculos deben refactorizarse si la proyección corregida
supera ese tope. G6c no puede recibir líneas: el checkpoint estaba en 550 y la
corrección externa informada queda en 548; cualquier cambio allí exige refactor
y nuevas huellas.

## Arquitectura del conductor una vez exista la frontera

`conductor.sh` creará un directorio con `mktemp -d`, copiará solo las diez
fuentes declaradas, verificará SHA/Go 1.26.5 y construirá dos veces con
`CGO_ENABLED=0`. Cada caso se ejecutará en un nuevo grupo/sesión de prueba con:

1. watchdog exterior y límites finitos;
2. 0/1/2 sellados a ficheros privados; huecos 3..9 y capacidades por encima de
   10 para `LOW_HOLE`;
3. FD de resultado heredado y `CLOEXEC`;
4. inventarios `/proc/<pid>/fd`, hijos y zombis tomados únicamente por el
   conductor exterior;
5. salida NDJSON por caso con SHA del binario, ID, estado, EOF, bytes de
   stdout/stderr, `Wait`, señales, inventarios y duración monotónica;
6. cien iteraciones para C21 y los casos nominales, con orden fijo y semilla
   registrada;
7. limpieza acotada del grupo de prueba y fallo si queda cualquier residuo.

`PDEATH_CREADOR` usa un abuelo que observa al Bash y mata el proceso/hilo que
lo creó; `PDEATH_OTRO` termina otro hilo y exige que Bash continúe hasta el
cierre controlado del ticket. Ninguno ejecuta el modo operativo. Los escenarios
AF exigen `exit=65`, EOF del FD de resultado y stdout/stderr de longitud cero.

## Oráculos por bloque

- C01..C07: preparación, frontera, mapa, lease, TERMINAL y ticket.
- C08: low-hole, tres referencias pidfd exactas y rechazo de cuarta/flags.
- C09: tuplas A/B/C; solo A puede ser `sin_hijo`.
- C10: fallo de duplicación, retirada, terminalidad, `Wait=1`, residuo cero.
- C11..C14: terminalidad inmediata, precedencias, referencia fiable y EOF/KILL.
- C15: AF 65/EOF sin E/S previa.
- C16..C19: consumo lineal, testigos, límites y barrera posterior.
- C20: ambos oráculos `Pdeathsig`.
- C21: cien inventarios inicial/final idénticos y cero residuo.

El cierre requiere que todos los IDs aparezcan exactamente una vez en el
manifiesto, que ningún caso quede como `SKIP`, y que el resumen sea derivado de
los registros, nunca escrito a mano.

## Frontera C18

El conductor O3a acredita fragmentación, aplazamiento, consumo en una vuelta
posterior y retirada por el límite de bootstrap. La revalidación inmediatamente
posterior al handoff pertenece a O3b y el plazo funcional iniciado antes de
`CONT` pertenece a O3c/O6; este corte no abre ni simula esas fases. En O3a, la
unicidad de `finRetirada` y su cota de tres segundos se prueba combinando el
oráculo causal C10 con el analizador AST: existe una sola asignación después
del intento de duplicación y ninguna ruta la reinicia.

## Comprobación histórica del primer corte

`./verificar_disponibilidad.sh` pertenece al primer corte y se conserva solo
como genealogía; no es una puerta vigente. La entrada durable actual es
`conductor.sh`, que compila únicamente copias privadas y nunca inicia el modo
operativo de Orquesta.

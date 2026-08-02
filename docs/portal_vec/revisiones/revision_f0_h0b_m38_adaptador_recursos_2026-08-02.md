# Revisión F0-H0b M38: adaptador privado de recursos

Fecha: 2 de agosto de 2026.

## Resultado

La
[enmienda del adaptador privado de recursos](../enmienda_f0_h0b_m38_ciclo_privado_2026-08-02.md),
confirmada en el candidato final `a6881f7`, obtuvo dos revisiones documentales
independientes `GO`, ambas con `P0=0`, `P1=0` y `P2=0`.

El dictamen autoriza exclusivamente a implementar el diseño y las puertas que
describe la enmienda. No acredita todavía el código, Bash, ShellCheck,
PostgreSQL 18.4, la matriz de 39 procesos, H0b, C2, F0 ni producción.

## Motivo de la enmienda

El candidato funcional preservado alcanzó 772 líneas en el runner y 580 en el
auxiliar H0b. Una preprueba nominal H0 resultó verde, pero las revisiones
detectaron que N01/E01, N07/E07 y F02/F03/F04 no conservaban todavía toda la
identidad física necesaria para una retirada segura. También faltaba hacer
incondicional la recuperación exterior ante resultados ausentes o inválidos.

Añadir esas garantías al runner excedía el máximo global de 800 líneas de
[DEC-051](../registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
La enmienda adopta por ello un cuarto auxiliar privado, dedicado únicamente al
mecanismo del ciclo de recursos, dentro de la misma captura coherente y sin
crear un supervisor adicional.

## Cadena de revisión

Los objetos intermedios fueron candidatos documentales observados durante
sucesivos `commit --amend`. No se consideran aprobados ni se presupone que
conserven una referencia Git.

| Candidato | Dictamen observado | Corrección exigida |
| --- | --- | --- |
| `c53e91d` | `NO-GO`, con `P1=3` y `P2=1` | Mantener el bootstrap temprano en el runner; cerrar la ventana fork -> `$!`; acotar Docker; desglosar el presupuesto del adaptador. |
| `f4df50f` | `NO-GO` | Impedir que los clientes Docker del hijo escapasen del PGID del caso. |
| `2e6203b` | dos `NO-GO`, cada uno con un único `P1` | Añadir un plazo absoluto para `docker exec/cp` y neutralizar la herencia de `monitor` mediante `SHELLOPTS`. |
| `a6881f7` | dos `GO`, ambos `P0=P1=P2=0` | Candidato documental final. |

El candidato final impone 180 segundos desde `CONT` para el caso completo,
retira el atributo de exportación de `SHELLOPTS`, ejecuta `set +m` en el hijo y
comprueba que `m` no figure en `$-` antes del auto-`STOP`. Los clientes
`docker exec/cp` permanecen en primer plano y dentro del PGID del caso.

## Propiedad y presupuesto aprobados

El runner conserva parser, selector, causal, oráculo, resultado, decisión,
trampas y conducción de los 39 procesos. El auxiliar H0b conserva sus
plantillas, R0 sintético, catálogo y oráculo independiente. El nuevo adaptador
solo implementa mecanismos de temporal, contenedor, transporte, grupo de
procesos, identidades interiores, retirada y mutantes del ciclo.

El *write-set* de implementación queda limitado a:

1. el runner existente;
2. el auxiliar H0b existente;
3. el nuevo adaptador privado de recursos.

D2c, D2d y el capturador Go deben permanecer byte a byte inmutables. La captura
privada pasa de tres a cuatro auxiliares con rutas y SHA-256 literales.

| Fichero | Estimación aprobada | Límite local | DEC-051 |
| --- | ---: | ---: | ---: |
| Runner | 734..760 | 775 | 800 |
| Auxiliar H0b existente | 580 exactas | 580 | 800 |
| Adaptador privado nuevo | 265..349 | 350 | 800 |

Superar un límite es un alto obligatorio. No se autoriza minificar, agrupar
sentencias, retirar mensajes o mutantes ni trasladar al adaptador decisiones
del runner para hacer caber el código.

## Garantías contrastadas

Las dos revisiones finales aceptaron conjuntamente:

- captura privada coherente de cuatro auxiliares, sin reapertura de rutas vivas;
- contenedor único precreado, acreditado y listo antes de lanzar al hijo;
- adopción por CID, nombre y etiqueta exactos, sin segundo `docker run`;
- hijo Bash directo con `PID=PGID`, PPID acreditado y barrera `set +m`/`STOP`;
- recuperación de la ventana provisional entre fork y asignación de `$!`;
- plazos de 30 segundos para creación y borrado, 60 para readiness y 180 para
  el caso completo;
- extinción del PGID, espera del hijo directo y retirada exacta e idempotente;
- identidades físicas y topología de padres, hojas y wrappers interiores;
- validación completa antes de F02/F03/F04 y revalidación inmediata antes de
  cada retirada;
- ausencia total idempotente, rechazo cerrado de ausencias parciales y
  protección de padres basales;
- recuperación anterior a interpretar `RESULTADO` y estado 65 ante cualquier
  fallo de limpieza;
- rechazos 64 con traza no vacía, un único `+ exit 64` final y cero recursos;
- mutantes de señales, `SHELLOPTS`, clientes Docker bloqueados, identidad,
  sustitución, topología, parcialidad e idempotencia.

## Puertas documentales ejecutadas

- objeto exacto revisado: `a6881f70a48a6718d84ea81eee730080d0dc9de5`;
- worktree del candidato: limpio;
- `git show --check a6881f7`: verde;
- `git diff --check 381d04b..a6881f7`: verde;
- Gitleaks limitado a `381d04b..a6881f7`: un commit y cero fugas;
- enlaces locales y presupuestos: coherentes;
- revisiones solo lectura, sin editar código ni ejecutar PostgreSQL.

## Estado

La decisión permite reanudar la implementación funcional desde el candidato
preservado. No aumenta métricas ni hereda la preprueba H0 como prueba final. F0
permanece en `10/23`, O4-05 en `3/5`, Contratación temporal en `24/46`, Bolsa
productiva en `1/14` y producción en `NO-GO` hasta superar todas las puertas
funcionales y dos revisiones independientes del commit de código.

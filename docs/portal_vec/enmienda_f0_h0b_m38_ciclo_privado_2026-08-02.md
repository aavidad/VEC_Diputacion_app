# Enmienda F0-H0b M38: adaptador privado de recursos y recuperación total

Fecha: 2 de agosto de 2026

Estado: candidata; no autoriza implementación hasta obtener dos revisiones
independientes con `P0=0`, `P1=0` y `P2=0`.

Prevalencia propuesta: sustituye el límite y el reparto de responsabilidades de
[la enmienda M38 de 775 líneas](enmienda_f0_h0b_matriz_limite_runner_775_2026-08-02.md)
solo en lo que esta decisión contradiga expresamente.

## Motivo del nuevo alto

El candidato funcional preservado en `agent/f0-h0b-funcional-20260802` alcanzó
772 líneas en el runner y 580 en el auxiliar H0b. Una preprueba H0 real sobre
PostgreSQL 18.4 terminó con estado cero y acreditó el rechazo sin R0, el
recorrido nominal, el error posterior y la restauración de las líneas base. Ese
resultado no autoriza la matriz M38 ni el cierre funcional.

El productor detuvo el trabajo antes de confirmar o ejecutar los 39 procesos
porque F02 y F04 todavía retiraban recursos por ruta sin conservar su identidad
física. Dos revisiones independientes ampliaron el hallazgo:

- N01/E01 no registraban dispositivo, inodo, UID, tipo, modo y enlaces de las
  hojas creadas;
- N07/E07 acreditaban el contenido, pero no la identidad física de los
  wrappers que F02 retiraba después;
- F03 y F04 podían retirar componentes, hojas o padres sin una propiedad física
  suficiente;
- un `RESULTADO` ausente o malformado podía devolver 65 antes de ejecutar la
  recuperación exterior del hijo;
- los fallos entre la preasignación y la espera tampoco convergían siempre en
  una única recuperación;
- la autoprueba de rechazo 64 aceptaba una traza vacía y no exigía exactamente
  el `exit 64` observado.

El incremento mínimo legible dentro del runner se ha estimado en 45..61 líneas
netas, antes de reordenar las líneas excesivamente largas del candidato. El
resultado 817..833 superaría el tope global de 800 fijado por
[DEC-051](registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
No se autoriza comprimir sentencias, reducir pruebas ni convertir el tope de
800 en una excepción.

## Alternativas examinadas

### A. Mantener runner y auxiliar congelados

Rechazada. Las 28 líneas disponibles hasta DEC-051 no cubren identidad,
validación agrupada, recuperación incondicional, mutantes y legibilidad. Un
objetivo de 795 o un límite de 800 producirían un falso margen.

### B. Trasladar el ciclo al auxiliar H0b de 580 líneas

Rechazada. Ese fichero conserva plantillas, fixtures, R0 sintético, catálogo y
oráculo literal independiente. Añadirle materialización, identidades y retirada
mezclaría la autoridad que ejecuta los efectos con la que determina la
observación esperada y lo acercaría innecesariamente a DEC-051.

### C. Adaptador privado dedicado al ciclo de recursos M38/H0b

Adoptada, condicionada a doble revisión. Se incorpora un único fuente Shell
nuevo en la ruta literal
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh`.
No es un segundo conductor y no puede ejecutarse de forma autónoma.

La conclusión anterior de que todo cuarto artefacto abría necesariamente otra
ventana TOCTOU deja de ser válida para este diseño concreto. El nuevo fuente
entra en la misma captura privada coherente que los tres auxiliares actuales:
se copia una sola vez desde el árbol acreditado, se valida en el manifiesto por
ruta y SHA-256 literal, se rechaza su ejecución directa con 64 y se carga solo
desde esa copia. Tras la captura no se vuelve a abrir su ruta viva.

La nueva superficie aumenta una ruta y una huella en el manifiesto, pero no
crea otra clase de proceso de caso, descriptor, temporal exterior, contenedor
ni autoridad de limpieza exterior.

## Propiedad exacta

### Runner

El runner sigue siendo el único propietario de:

- parser y rechazos tempranos;
- normalización literal `set +m` y comprobación de que `m` no figura en `$-`;
- bootstrap literal `builtin kill -STOP "$BASHPID"` para hijos válidos;
- validación temprana de FD y temporal preasignado antes de cargar auxiliares;
- conductor de 39 procesos, FD del runner y FD de la raíz;
- contrato del ticket, PPID e identidad que corresponde a cada caso;
- decisión de crear, lanzar, señalizar, esperar y retirar cada recurso;
- verificación final de que descubrimiento, acreditación y retirada coinciden
  con la identidad ordenada;
- causal 0/64/65/79, `RESULTADO` y estado real posterior a `EXIT`;
- activación de la trampa y del epílogo desde la primera identidad;
- orden F01..F15 y comparación con el oráculo independiente.

El runner delega el mecanismo en funciones del adaptador cargadas dentro de su
propio proceso. La delegación no crea un supervisor intermedio: el runner sigue
siendo el padre directo del Bash de cada caso y quien interpreta el resultado.

### Auxiliar H0b existente

Permanece congelado en 580 líneas. Conserva plantillas, validadores, R0
sintético, catálogo de 38 casos, oráculo literal de 39 observaciones y mutantes
estructurales del oráculo. No recibe funciones del ciclo interior.

### Nuevo adaptador privado de recursos

Es propietario exclusivamente de:

- reserva paterna, acreditación y retirada del temporal exterior de cada caso;
- creación síncrona, acreditación y adopción del contenedor ya identificado;
- transporte por FD, mitad paterna de la barrera y grupo de procesos del hijo;
- señalización del PGID, espera del hijo directo y comprobación de extinción;
- retirada exterior idempotente invocada por el epílogo del runner;
- materialización H0b dentro del contenedor ya acreditado;
- preausencia, creación y registro físico de componentes, padres, hojas y
  wrappers exactos;
- cotejo de contenido e identidad en N01/E01 y N07/E07;
- validación agrupada anterior a toda retirada de F02, F03 y F04;
- retirada en orden seguro de los recursos que el propio ciclo registró;
- mutantes de sustitución, ausencia, nodo no vacío e idempotencia de esos
  recursos interiores.

No contiene el catálogo, selector, observación esperada, oráculo, causal ni
decisión de aceptación. Recibe del runner el caso y la identidad, construye el
transporte conforme al contrato literal y devuelve una observación mecánica
sin interpretarla. Como se carga con `source` dentro del runner, sus funciones
pueden lanzar directamente `/usr/bin/bash`, Docker y las operaciones interiores
sin alterar el PPID. Todas las invocaciones se esperan de forma síncrona o se
extinguen mediante el epílogo; no queda un proceso supervisor propio.

El auxiliar SQL D2c, el auxiliar operativo D2d y el capturador Go permanecen
byte a byte inmutables.

## Captura privada de cuatro auxiliares

El runner debe ampliar la captura actual de manera cerrada:

1. declarar la cuarta ruta y su SHA-256 literal;
2. pasar exactamente las cuatro rutas al capturador en un orden literal;
3. exigir exactamente cuatro entradas, sin ausencias, duplicados ni extras;
4. comparar ruta y SHA-256 de cada entrada antes de cargar cualquiera;
5. exigir que la ejecución directa de los cuatro auxiliares devuelva 64;
6. ejecutar Bash y ShellCheck sobre el runner y las cuatro copias privadas;
7. cargar el nuevo adaptador desde el snapshot solo con la marca efímera de
   carga privada, que se consume al entrar.

Un mutante debe sustituir la cuarta ruta o su huella y demostrar fallo cerrado
antes de Docker. La copia privada y el manifiesto siguen siendo recursos del
temporal exterior acreditado y se retiran mediante la trampa del runner.

## Identidades y topología interior

El ciclo parte del inventario físico del snapshot. Para cada padre canónico
distingue entre:

- padre basal presente: registra y valida su identidad, pero nunca lo adopta ni
  lo retira;
- padre ausente: acredita la preausencia, lo crea con modo privado, registra su
  identidad y solo entonces lo considera propio.

Ningún nombre como `pruebas_sql` o `000007_componentes` se retira por el mero
hecho de coincidir con una ruta esperada. F03 solo puede retirar un padre de
componentes si acreditó su preausencia, creación e identidad; un padre basal se
limita a validarse. F04 queda restringida al árbol dedicado de wrappers y no
retira padres de componentes. Las rutas que deban volver a estar ausentes se
asignan a F03 o F04 de forma única, nunca a ambas.

Para cada directorio propio se registra al menos dispositivo, inodo, UID, tipo
y modo. El número de enlaces se acredita en cada transición que añada o retire
una hoja. Para cada fichero se añaden tamaño, número de enlaces y SHA-256.

N01 y E01 realizan `mkdir --mode=0700` sin `--parents` sobre hojas distintas y
ausentes después de acreditar R0. La identidad queda registrada antes del seam
inyectable. N07 y E07 copian cada wrapper a su hoja ausente, registran su
identidad física y N08/E08 acreditan su contenido. Un fallo entre copia y
registro no autoriza a borrar por ruta un objeto no identificado: la
finalización interior devuelve 65 y la trampa exterior retira el contenedor
completo por ID, nombre y etiqueta.

## F02, F03 y F04: validación total y retirada acotada

Cada acción opera sobre el grupo de recursos registrados para esa fase. Un
grupo se considera idempotentemente ausente solo si todos sus miembros están
ausentes. Una mezcla de miembros presentes y ausentes devuelve 65 y no retira
nada. Las rutas nunca registradas no pertenecen al grupo.

F02, F03 y F04 ejecutan un único `docker exec` por acción con dos pasadas:

1. validar todos los recursos presentes contra sus identidades y, para los
   directorios, demostrar la topología y el vacío esperados;
2. si la primera pasada es completamente verde, revalidar la identidad de cada
   miembro inmediatamente antes de retirarlo, hacerlo en orden de hojas a
   padres y acreditar la postausencia del grupo.

La primera discrepancia detiene la acción antes de la segunda pasada y no
retira nada. Durante la segunda pasada, una discrepancia en la revalidación
inmediata detiene las retiradas restantes y fuerza 65. Puede quedar una retirada
parcial controlada de recursos originales ya revalidados; el auxiliar registra
qué miembros retiró y cuáles permanecen, no borra el sustituto y la recuperación
exterior destruye después el contenedor completo. No se promete atomicidad
criptográfica o de filesystem que Shell no ofrece.

La garantía se acota al contenedor privado, sin red, sin sesiones de cliente ni
otro `docker exec` lanzado por el runner mientras actúa el finalizador. Un
proceso del mismo UID con acceso concurrente al filesystem del contenedor o un
administrador de Docker queda fuera de esta frontera y se controla
operativamente. No se usan glob, prefijos ni limpieza global.

Los mutantes deben cubrir como mínimo:

- wrapper con igual contenido y distinto inodo;
- hoja vacía sustituida por otro directorio en la misma ruta;
- padre propio sustituido y padre propio no vacío;
- intento de adoptar o retirar un padre basal;
- ausencia parcial antes de F02, F03 y F04;
- segunda ejecución idempotente cuando todo el grupo ya está ausente;
- fallo en la primera y en la última validación;
- sustitución entre la primera pasada y la revalidación inmediata.

## Epílogo exterior único

La identidad del caso se marca activa inmediatamente después de generarla y
antes de comprobar preausencia o crear recursos. Desde ese punto no existe un
`return`, `exit` ni fallo de oráculo que evite el epílogo único. Todas estas
transiciones convergen en él:

```text
identidad -> preausencia -> mkdir -> stat -> docker run -> CID
          -> ticket -> STOP -> acreditación PGID -> CONT -> espera
          -> RESULTADO -> oráculo
```

El adaptador guarda el estado previo de control de trabajos, exige inicialmente
cero trabajos activos, activa `set -m` solo durante la matriz y lo restaura
tanto en la salida normal como en el epílogo. Antes del fork arma una identidad
de trabajo provisional, retira el atributo de exportación de `SHELLOPTS` y
difiere `INT`/`TERM`. Lanza `/usr/bin/bash` como orden simple en segundo plano,
sin subshell, `env`, `setsid` ni supervisor intermedio. Bash crea antes de
`exec` un grupo de procesos nuevo encabezado por el hijo.

El adaptador copia `$!` inmediatamente. Si una señal se entrega en la ventana
entre fork y asignación, el epílogo solo puede consultar la tabla de trabajos
porque se acreditó vacía antes y debe encontrar cero o un único trabajo
provisional; cualquier cardinalidad distinta fuerza incidente. Antes de `CONT`
registra PID, PGID, PPID y tiempo de inicio, exige `PID=PGID` y liga esos datos a
la identidad activa. No enumera ni mata procesos por prefijo o coincidencia
ambigua.

Después de validar internamente el ticket, el bootstrap temprano del runner —no
el adaptador, que todavía no está cargado en el nuevo Bash— ejecuta
literalmente `set +m`, exige que `m` no figure en `$-` y después ejecuta
`builtin kill -STOP "$BASHPID"` antes de cualquier orden externa. El conductor
no envía `CONT` hasta acreditar estado detenido, PID, PGID, PPID y tiempo de
inicio. Una señal o fallo anterior encuentra un trabajo provisional detenido
que el epílogo puede extinguir. El ticket conserva la ligadura directa
`PPID|caso|identidad` y ninguna orden externa precede a la barrera.

Antes de ese lanzamiento, el adaptador crea de forma síncrona el único
contenedor del caso con nombre y etiqueta derivados de la identidad activa. El
comando de creación debe terminar, el CID completo, nombre, etiqueta, imagen y
estado deben quedar acreditados y PostgreSQL debe estar listo antes de entregar
el ticket. Durante esta sección crítica `INT` y `TERM` se difieren; la trampa no
puede iniciar limpieza mientras el cliente Docker siga en curso. Al salir de la
sección, una señal pendiente converge en el epílogo con la identidad todavía
activa.

La implementación concreta de esa espera no ignora la señal: la trampa solo
registra el código pendiente. Únicamente los clientes Docker paternos de
precreación y retirada se lanzan en grupos de trabajo separados; el adaptador
conserva sus PID directos y los espera explícitamente. `wait -f` solo se da por
verde al obtener el estado terminal. La creación y el borrado tienen 30
segundos y la readiness dispone de 60 segundos totales, con sondeos individuales
también acotados.

Dentro del hijo, el control de trabajos permanece desactivado. Sus clientes
`docker exec` y `docker cp` se ejecutan en primer plano y conservan el PGID del
caso; nunca se separan en otro grupo ni se pretende obtener `$!` de una orden en
primer plano. El conductor aplica un plazo absoluto literal de 180 segundos al
caso completo desde `CONT`. Si el Bash no termina, el epílogo extingue el PGID,
espera al hijo directo, fuerza 65 y retira los recursos exactos. Así el plazo y
la extinción alcanzan también cualquier `docker exec/cp` bloqueado aunque muera
el Bash que lo esperaba.

Si vence un plazo, se extingue y espera el grupo del cliente, la identidad sigue
armada, la matriz se detiene y no se lanza otro caso. Cuando el daemon responde,
se reconcilian CID, nombre y etiqueta exactos. Si el daemon continúa
indisponible, se registra un incidente 65 y no se afirma postausencia ni cierre
verde. `SIGKILL`, caída del host y actuación de un administrador Docker quedan
fuera de la garantía en proceso y corresponden al procedimiento operativo de
recuperación.

El ticket incorpora el CID y las formas esperadas del cidfile y del contenedor.
El hijo no ejecuta `docker run`: solo adopta el recurso ya existente si CID,
nombre y etiqueta coinciden exactamente. Un selector de hijo que intente crear
otro contenedor o adoptar uno distinto devuelve 65. Así no queda ninguna
petición de creación del daemon iniciada por el hijo que pueda completarse
después de su muerte.

El epílogo y la trampa `EXIT` invocan una única primitiva idempotente del
adaptador. Si hay PGID activo, envían `CONT` si estaba detenido, `TERM`, esperan
como máximo dos segundos de gracia, envían `KILL` solo si el grupo persiste y
hacen `wait -f` únicamente sobre el Bash hijo directo. Después comprueban por
PGID que no queda ningún miembro. No se afirma que Bash pueda recolectar nietos.
Esto cubre los clientes `docker exec` y `docker cp` descendientes del Bash del
caso. Los procesos ya iniciados dentro del contenedor se extinguen al retirar el
contenedor completo.

Después reacreditan y retiran contenedor y temporal de esa identidad. La
retirada espera la respuesta del daemon y acredita la postausencia por CID,
nombre y etiqueta antes de desarmar la identidad activa. Si el hijo no llegó a
arrancar, omite la señalización pero ejecuta la misma retirada exacta. No se
acepta una ventana temporal de sondeos como sustituto de la barrera causal:
todo `docker run` posible terminó y fue acreditado antes del lanzamiento.

La forma física del temporal se registra en cuanto está disponible. Si un fallo
ocurre antes de obtenerla, la retirada por ruta solo se permite dentro de la
frontera operativa sin escritor concurrente y tras acreditar que la ruta
secreta, preausente y recién creada conserva UID, tipo y modo esperados. Una
identidad dudosa no se borra a ciegas; fuerza 65 y queda a cargo de la evidencia
operativa de incidente.

Los mutantes cubren, como mínimo, fallos después de generar identidad, después
de `mkdir`, después de `stat`, al construir el ticket, en las redirecciones, en
el lanzamiento, antes y durante la espera, al analizar `RESULTADO`, al consultar
el oráculo y al reacreditar runner y raíz. Se inyectan además:

- en la ventana exacta fork -> `$!` -> PID/PGID provisional;
- con `SHELLOPTS` exportado y `monitor` activo en el padre;
- antes, durante y después de `docker run`;
- con cidfile ausente, malformado, sustituido o discrepante;
- entre CID, nombre, etiqueta, imagen, readiness y ticket;
- durante adopción, `docker exec` y `docker cp`;
- matando el Bash mientras espera cada cliente Docker del hijo;
- manteniendo `docker exec` y `docker cp` sin terminar hasta vencer los 180
  segundos del caso;
- con `wait -f` interrumpido y con vencimiento del plazo Docker;
- intentando un segundo `docker run` desde el hijo.

Cada mutante debe demostrar grupo extinguido, hijo directo recolectado y cero
temporal antes del siguiente caso. El mutante de daemon no terminante acredita
que no existe siguiente caso y que la identidad permanece armada hasta la
reconciliación; no exige una postausencia que el sistema no puede observar.

## Resultado y rechazos 64

El conductor separa `observación_válida` de `recuperación_completa`. Conserva el
estado declarado, espera al proceso, observa el estado real posterior a `EXIT`
y ejecuta el epílogo antes de decidir el caso. Un `RESULTADO` ausente, duplicado,
malformado o discrepante no puede saltarse la recuperación.

El estado del caso solo es aceptable cuando coinciden la observación literal,
el estado declarado, el estado real, la postausencia de recursos y las
identidades del runner y la raíz. Un fallo de recuperación fuerza 65.

Cada rechazo temprano debe acreditar conjuntamente:

- estado 64;
- traza no vacía;
- exactamente una línea `+ exit 64`, situada al final;
- ninguna línea fuera de la lista positiva de operaciones internas del parser;
- cero identidad, temporal, nombre, etiqueta, Docker y `psql` del hijo.

Se añaden mutantes de traza vacía, `exit` ausente, duplicado, estado distinto y
orden externa no permitida.

## Presupuesto reproducible

El ledger parte del candidato observado de 772 líneas:

| Movimiento del runner | Mínimo | Máximo |
| --- | ---: | ---: |
| Mover seis constantes de rutas interiores | -6 | -6 |
| Mover `preparar_integracion_h0b_f0` | -19 | -19 |
| Mover `ejecutar_wrapper_h0b_f0` | -11 | -11 |
| Extraer el ciclo interior puro de `probar_h0b_funcional_f0` | -23 | -23 |
| Mover `retirar_recursos_m38_f0` | -11 | -11 |
| Mover preparación/lanzamiento/retirada mecánica del bucle | -5 | -5 |
| Mantener en runner selector, causal, finalizador y `RESULTADO` | 0 | 0 |
| Base resultante | 697 | 697 |
| Manifiesto/carga de cuatro y puentes | +10 | +14 |
| Activación de trampa y decisión sobre el adaptador | +8 | +14 |
| Bootstrap `STOP` y trabajo provisional temprano | +2 | +4 |
| Normalización `set +m` y comprobación de flags | +1 | +2 |
| Traza y resultado exactos | +3 | +5 |
| Descompresión de líneas opacas | +8 | +16 |
| Coordinación y mutantes exteriores | +5 | +8 |
| **Estimación final del runner** | **734** | **760** |

El ledger independiente del adaptador es:

| Bloque del adaptador | Mínimo | Máximo |
| --- | ---: | ---: |
| Código trasladado desde el runner | 75 | 75 |
| Estado temporal exterior y retirada exacta | +20 | +30 |
| Precreación, CID, adopción y readiness Docker | +32 | +45 |
| Trabajo provisional, PGID, plazos y epílogo | +38 | +52 |
| Identidades y F02/F03/F04 interiores | +45 | +61 |
| Mutantes interiores y exteriores | +45 | +70 |
| Guardia de carga y contratos privados | +10 | +16 |
| **Estimación final del adaptador** | **265** | **349** |

La parte de `probar_h0b_funcional_f0` que se extrae solo ejecuta el ciclo y
solicita seams mediante una primitiva cerrada del runner; no lee selector, modo
M38, observación esperada ni oráculo, y no emite `RESULTADO`.

| Fichero | Objetivo | Límite local | DEC-051 |
| --- | ---: | ---: | ---: |
| Runner | 760 o menos | 775 | 800 |
| Auxiliar H0b existente | 580 exactas | 580 | 800 |
| Nuevo adaptador privado de recursos | 350 o menos | 350 | 800 |

No se cuenta la diferencia hasta 800 como reserva de I0. I0 permanece bloqueado
y requerirá su propia decisión. No se autoriza obtener estos límites agrupando
sentencias de control, eliminando mensajes o mutantes, ni conservando líneas
opacas introducidas solo para hacer caber el candidato de 772 líneas.

## Puertas de implementación

La implementación solo puede confirmarse tras superar:

1. write-set exacto: runner, auxiliar H0b existente y nuevo adaptador de recursos;
2. hashes invariantes de D2c, D2d y capturador;
3. SHA literal de ambos auxiliares H0b y manifiesto privado exacto de cuatro;
4. Bash, ShellCheck, `git diff --check`, límites y Gitleaks;
5. rechazos 64 y mutantes antes de Docker;
6. mutantes de identidad, sustitución, topología, no vacío, grupos parciales e
   idempotencia;
7. mutantes del epílogo desde la primera identidad hasta oráculo y
   reacreditación;
8. precreación acreditada, cidfile/identidad/readiness adversos, barrera
   `set +m`/`STOP`, `SHELLOPTS` exportado, trabajo provisional, plazos Docker,
   adopción exacta por el hijo y rechazo de todo segundo `docker run`;
9. matriz literal de 39 procesos PostgreSQL 18.4, con estados 0/79/65, una sola
   línea `RESULTADO`, traza completa y estado real posterior a `EXIT`;
10. H0 tres veces, A1, C1 y mutante A1;
11. cero temporales, contenedores, sesiones, roles, preparados u objetos
    residuales después de cada caso;
12. dos revisiones independientes del commit funcional con
    `P0=0`, `P1=0`, `P2=0`.

La preprueba H0 ya observada es evidencia de diagnóstico, no sustituye ninguna
de estas puertas. Producción y despliegue permanecen en `NO-GO`.

# Decisión F0-H0b/C4b-2/G2-O/O3a-P1: barrera temprana del ticket

Fecha: 9 de agosto de 2026.

Estado: **DECISIÓN APROBADA PARA PUBLICACIÓN DOCUMENTAL**. El doble `GO` final
autoriza únicamente confirmar y publicar esta decisión junto con su acta. La
edición material R-only continúa bloqueada hasta que ese commit documental
obtenga CI `5/5`; no autoriza O3a, `Start`, mapa FD ni producción.

## Resultado único

O3a-P1 corrige una precondición física de O3a: en el modo hijo H0b, la lectura,
cardinalidad, cierre y validación de FD 9, seguidas del auto-`STOP`, deben ocurrir
antes de ejecutar cualquier comando externo o crear cualquier descendiente.

El framing y las validaciones del ticket no cambian. Sí cambia deliberadamente
la precedencia defensiva ante dos entradas simultáneamente adversas: un ticket
ausente o inválido termina 64 antes de inspeccionar el entorno; con ticket
válido, el hijo se auto-detiene y, tras `CONT`, un entorno hostil termina 65.

La intervención es un traslado reversible de un bloque existente del runner.
No crea una capacidad funcional, no arranca el futuro supervisor Go, no cambia
el protocolo ni autoriza O3a, `Start`, el mapa FD, pidfd, ticket escrito,
`/proc`, `CONT`, Docker, PostgreSQL real o producción.

## Hallazgo que obliga al corte

En la base publicada, el runner ejecuta actualmente:

1. `/usr/bin/env -0 | /usr/bin/grep ...` en R9--R10;
2. la lectura de FD 9 en R29;
3. el auto-`STOP` en R45.

Por tanto el Bash provisional puede crear dos descendientes antes de recibir el
ticket y antes del `STOP`. Cerrar el escritor desde O3a no acredita por sí solo
que nunca existieron esos descendientes. Compensarlo dentro de O3a obligaría a
introducir señalización de grupo y drenaje de adoptados, responsabilidades que
pertenecen a O4 y ampliarían innecesariamente la superficie crítica.

La corrección elegida hace que, en modo hijo, antes de la lectura de FD 9 y del
auto-`STOP` solo se ejecuten builtins de Bash. El control de entorno conserva su
contenido exacto y se ejecuta inmediatamente después de cerrar la rama de
ticket/auto-`STOP`, antes de seleccionar raíz y runner.

## Base y autoridad

La base exacta es el commit publicado
`623a3739169421d723ea3458c8acb5c12b157a2b`, de la rama
`integracion/ct-o4-04e-20260726`. La CI `31291669273` terminó
`completed/success` con cinco de cinco puertas verdes.

El único material autorizado en este corte es:

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

Ledger previo de R:

```text
líneas: 702
SHA-256: e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c
modo Git: 100755
```

El resto del repositorio, incluido D2d, G1--G5, capturador, adaptador, SQL,
migraciones y documentos de autoridad previos, permanece byte a byte.

## Transformación proyectada

La proyección externa debe extraer de R base exactamente el bloque continuo
R8--R20, incluidos sus trece saltos de línea finales:

```bash
estados_entorno=()
if /usr/bin/env -0 |
    /usr/bin/grep -zE '^(BASH_FUNC_|LD_)' >/dev/null
then
    estados_entorno=("${PIPESTATUS[@]}")
    exit 65
else
    estados_entorno=("${PIPESTATUS[@]}")
fi
((${#estados_entorno[@]} == 2)) || exit 65
[[ "${estados_entorno[0]}" == 0 &&
   "${estados_entorno[1]}" == 1 ]] || exit 65
unset estados_entorno
```

Debe insertarlo una sola vez inmediatamente después del `fi` que cierra la
rama `--caso-inyeccion-h0b` --R47 en la base-- y antes de:

```bash
if [[ "${modo_m38}" == hijo ]]; then
```

No se cambia ningún byte interior del bloque. No se mueven el shebang, las
guardas de ejecutables, `export LC_ALL=C`, `umask 077`, los `unset`, la
inicialización de estado, la lectura/validación de FD 9 ni el auto-`STOP`.

Consecuencias deliberadas y acotadas:

- en modo hijo, ticket inválido, cardinalidad inválida o EOF termina 64 antes
  de crear descendientes;
- con ticket válido, el Bash cierra FD 9 y se auto-detiene antes del control de
  entorno;
- tras el futuro `CONT`, el bloque de entorno se ejecuta una vez y sigue
  rechazando `BASH_FUNC_*` y `LD_*`;
- en modo directo, el mismo bloque se ejecuta una vez antes de resolver raíz y
  runner;
- el bloque recibe ya `LC_ALL=C`; esta normalización determinista no amplía la
  lista de entorno ni cambia sus dos patrones prohibidos.

La proyección conserva 702 líneas. Debe calcular y fijar en este documento la
huella posterior exacta antes de cualquier autorización material. No se
autoriza ajustar texto, compactar controles ni aprovechar el traslado para
otra edición.

## Ledger posterior proyectado

Dos proyecciones independientes han reproducido exactamente el mismo candidato
fuera del repositorio:

| Unidad | Líneas | Bytes | Modo Git | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| R base | 702 | 46.123 | `100755` | `e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c` |
| bloque trasladado | 13 | — | — | `831b07df84fd6fa05d3f060e1c275de730b66eff54ac68caf720cc932eb2d00e` |
| R proyectado | 702 | 46.123 | `100755` | `7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487` |

En R proyectado, el auto-`STOP` queda en R32, el bloque trasladado en R35--R47,
la pipeline en R36 y la selección `modo_m38 == hijo` en R48. El diff tiene dos
hunks: elimina el bloque base R8--R20 y añade esos mismos bytes tras el `fi` base
R47. La transformación directa y la inversa reproducen ambos extremos byte a
byte; `bash -n`, ShellCheck 0.11.0 y diff-check quedan verdes.

No se modifica el tipo de fin de línea: todo R conserva LF. El ledger posterior
material debe repetir líneas, bytes, modo Git y las dos huellas exactas; una
diferencia detiene el corte.

## Invariantes estructurales

En el candidato proyectado deben cumplirse simultáneamente:

1. existe una sola pipeline `/usr/bin/env -0 | /usr/bin/grep ...`;
2. el bloque trasladado coincide byte a byte con R8--R20 de la base;
3. la rama de modo hijo contiene, en este orden, lectura única de FD 9,
   comprobación de cardinalidad, cierre único de FD 9, validación completa del
   ticket, `set +m`, comprobación de monitor y auto-`STOP`;
4. ningún comando externo, subshell, `pipeline`, sustitución `$(...)`,
   redirección a proceso, `source`, función externa ni ejecutable se alcanza
   entre la entrada del script y el auto-`STOP` en modo hijo;
5. los únicos pasos previos son asignaciones, `[[ ]]`, `(( ))`, `read`, `exec`
   de cierre, `set`, `unset`, `umask`, `export`, `builtin kill` y `exit` solo
   en ramas de rechazo; las construcciones `if`/`then`/`else`/`fi` y
   `for`/`in`/`do`/`done` son las únicas palabras de control permitidas;
6. el bloque de entorno queda después del cierre de la rama y antes de la
   selección `modo_m38 == hijo`;
7. el manifiesto de nueve artefactos, sus huellas, el build de cinco fuentes Go
   y la huella binaria `6153f03a...` permanecen byte a byte;
8. en R, `--supervisar-m38` y un argumento desconocido conservan estado 1,
   stdout vacío y stderr exacto
   `[F0 H0] ERROR: uso: runner [--etapa ETAPA|--matriz-inyeccion-h0b]` más LF;
   el binario Go B conserva estado 64 para su modo supervisor todavía cerrado y
   para argumento desconocido. Los modos directos admitidos de R conservan sus
   estados, salidas y efectos externos funcionales posteriores. La única
   diferencia de orden interno autorizada es que la inicialización local,
   `LC_ALL=C`, `umask` y los `unset` preceden ahora a la pipeline; no se promete
   identidad de la vista completa de `env` fuera de la detección exacta
   `BASH_FUNC_|LD_`.

La expresión «antes de cualquier comando externo» no prohíbe el propio
intérprete `/usr/bin/bash` ya arrancado ni los builtins enumerados. Sí prohíbe
cualquier descendiente adicional antes de FD 9 y `STOP`.

## Pruebas y oráculos

La proyección debe fijar comandos y salidas exactos para, como mínimo:

1. reconstrucción directa desde R base y reversión byte a byte a R base;
2. `bash -n`, ShellCheck 0.11.0 y `git diff --check` verdes;
3. analizador Shell portable que acredite el orden, resuelva builtins frente a
   ejecutables y fije ruta, SHA-256 y salida exacta; la huella global de R no
   cuenta por sí sola como oráculo;
4. hijo real con pipe FD 9 vacío: queda bloqueado, sin descendiente adicional,
   sin `STOP`, salida ni efecto;
5. cierre del escritor sin bytes: salida 64, sin descendiente ni residuo;
6. ticket adverso y segunda línea: salida 64 antes de descendientes;
7. ticket válido: auto-`STOP` previo a la pipeline; tras `CONT`, entorno limpio
   continúa y entorno `LD_*`/`BASH_FUNC_*` termina cerrado;
8. cruce adverso: ticket inválido más entorno hostil termina 64 sin ejecutar la
   pipeline; ticket válido más el entorno inerte `LD_PRUEBA=1` produce `STOP`,
   después `CONT`, y termina 65 sin efectos H0 ni otros descendientes que `env`
   y `grep`; no se usa `LD_PRELOAD` ni otra variable con efecto del cargador;
9. 100 iteraciones con inventarios inicial/final de hijos y FD del conductor
   iguales, sin depender de que Bash no abra sus propios FD;
10. H0 PostgreSQL 18.4, calidad global, carrera, `govulncheck`, Gitleaks y
   residuos cero;
11. R con `--supervisar-m38` y argumento desconocido termina 1, stdout vacío y
    la línea de uso exacta; el binario Go con ambos argumentos termina 64;
12. dos builds Go 1.26.5 reproducibles con la huella binaria invariante.

Las pruebas de proceso se ejecutan en un conductor subreaper aislado, con grupo
propio, pidfd y plazo total. El conductor cierra el ticket y acredita
terminalidad antes de un único `Wait`; recolecta también cualquier `env` o
`grep` adoptado y prueba tabla final vacía. Su salida de emergencia usa el
pidfd para el líder y el aislamiento del grupo para los descendientes, nunca un
PID reciclable no acreditado. `CONT` solo se admite dentro de esta evidencia
aislada y no autoriza la conducta productiva O3c. Nunca deja un Bash detenido o
un descendiente si falla el oráculo.

## Mutantes exactos

La proyección ha fijado once transformaciones de patrón único. Todas conservan
sintaxis Shell válida y superan `bash -n` y ShellCheck 0.11.0 antes de ejecutar
su oráculo:

| ID | Transformación exacta sobre R proyectado | Líneas | SHA-256 | Rechazo estructural |
| --- | --- | ---: | --- | --- |
| M01 | mover el bloque inmediatamente antes de `export LC_ALL=C` | 702 | `e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c` | `entorno_antes_stop` |
| M02 | mover el bloque inmediatamente antes de `[[ -n "${zh}"` | 702 | `d95cbd7d2a401239742f3d1b67d46b9dcc12869e945aeedc1ad5f17bc450f276` | `entorno_antes_stop` |
| M03 | insertar `/usr/bin/true` tras la inicialización `forma_*_m38` | 703 | `b6527b16546ee3117a3a446886ded76c824b1dc391bc8560c10bb9b0090ff661` | `prefijo_no_permitido` |
| M04 | insertar `/usr/bin/true` inmediatamente antes del auto-`STOP` | 703 | `f976c5be2d348318838935fd704ba16e9afa9e360eaa003113ec0a94fcb3ae93` | `prefijo_no_permitido` |
| M05 | eliminar la única línea `exec 9<&-` | 701 | `739eb921f06f55c289015c116953a5676be458eccb5c28c7dd83fedc45f91e27` | `cierre_fd9` |
| M06 | eliminar la lectura que rechaza una segunda línea | 701 | `b59fef757c28b08cbfba445fc0101f00574e94c220d2e15e69a1bd9eb70d146d` | `cardinalidad_fd9` |
| M07 | quitar el auto-`STOP` de su sitio e insertarlo justo después de `exec 9<&-` | 702 | `d39bb8a5f98a219cab32f74c85882094310e8e69f6de46f8d9122baec616e766` | `orden_auto_stop` |
| M08 | eliminar exactamente el bloque trasladado | 689 | `d7bebb53a104dd5f4de5d48f318993f978ad2b37bf64767b566d471365f8154d` | `pipeline_cardinalidad` |
| M09 | duplicar consecutivamente el bloque trasladado | 715 | `b28379924d40c230ca122147b5d3c19de9b7391ae42ed7dbe8ec939ed46ae0af` | `pipeline_cardinalidad` |
| M10 | sustituir una vez `^(BASH_FUNC_|LD_)` por `^(BASH_FUNC_)` | 702 | `bcffdc15cfd4777cd9cc84e9454cabdd5c580be1d64918391ef894e201dd5db3` | `bloque_entorno_exacto` |
| M11 | mover el bloque inmediatamente antes de `cd -- "${raiz}"` | 702 | `a19e13b8d500cee925b2f92bc9ff08f883f77f49265c1897591a29e1dc1569a8` | `entorno_despues_modo` |

El analizador preparatorio tiene 131 líneas y SHA-256
`936a1a242ca2c9120f87c412a7660ab357acf3e1d9c04184fc6890856abd7c1a`;
el generador preparatorio tiene 65 líneas y SHA-256
`ce00e0284a9349ece653b195afff7c935f75cf39ee4441c9078c2ff0bfa94977`.
Son plantillas externas, no evidencia durable ni permiso para copiarlas sin
revisión. La evidencia final debe retirar rutas físicas de sus diagnósticos,
ligar genealogía y SHA material y conservar la lógica exacta acreditada.

El analizador específico de este contrato no se presenta como parser Shell
general: fija SHA de R base, reconstruye la transformación autorizada, compara
bytes, cardinalidades, anclas, orden y prefijo permitido, pero no usa la huella
del candidato como atajo. Su salida positiva exacta debe ser:

```text
ok=1 lineas=702 sha=7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487
```

Además del rechazo estructural, la matriz conductual debe acreditar causa real:

- M01/M03 crean un proceso antes de FD 9 con pipe vacío;
- M02/M04 crean un proceso antes de `STOP` con ticket válido;
- M05 conserva FD 9 al observar `STOP`;
- M06 acepta indebidamente la segunda línea;
- M07 alcanza `STOP` con ticket inválido;
- M08 y M10 no rechazan `LD_PRUEBA=1` con estado 65;
- M11 alcanza `dirname` antes del rechazo del entorno;
- M09 puede morir solo por cardinalidad estructural.

Cada mutante se aplica exactamente una vez. La evidencia separa muertes
estructurales y conductuales; para M10 el rechazo estructural no sustituye su
oráculo `LD_PRUEBA=1`. Un fallo de parseo, una huella distinta, una salida por
otra causa o la imposibilidad de arrancar no cuentan como muerte funcional.

## Secuencia

1. doble revisión completa de este borrador;
2. corrección de todo P0/P1/P2 y nueva doble revisión;
3. solo con doble `GO`, proyección externa sin editar el repositorio;
4. incorporación al documento del ledger posterior y oráculos exactos;
5. doble revisión final, cambio limitado de estado y acta;
6. commit documental, publicación y CI `5/5`;
7. aplicación material R-only en worktree exclusivo;
8. puertas, commit material autónomo y doble revisión independiente;
9. integración, actualización documental, publicación y CI `5/5`;
10. rebasar el borrador O3a sobre esa nueva base y reanudar su corrección.

## Paradas duras

Se detiene ante una segunda ruta material; cambio interior del bloque; otra
edición de R; línea/hash posterior no fijado; comando externo antes de FD 9 o
`STOP`; modo hijo que no bloquea con ticket vacío; entorno hostil aceptado;
descendiente, Bash detenido, FD o temporal residual; cambio de manifiesto,
fuente Go o binario; modo que deja de cerrar; mutante superviviente o falso
muerto; puerta, revisión o CI no verde.

## Métricas

O3a-P1 es una precondición defensiva y no suma capacidad funcional. Mantiene
F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
`1/14` y producción `NO-GO`. No autoriza datos reales.

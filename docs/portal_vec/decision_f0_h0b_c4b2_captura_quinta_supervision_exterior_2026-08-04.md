# Decisión F0-H0b C4b-2: quinta captura para la supervisión exterior

Fecha: 4 de agosto de 2026.

Estado: corrección de dirección posterior al `NO-GO` documental de `f7ca3a9`,
pendiente de dos contrarrevisiones independientes con `P0=0`, `P1=0` y
`P2=0`. Mientras no obtenga ambos `GO`, no autoriza cambios de código ni
modifica C4b-2, H0b, C2, F0 o producción.

## Prevalencia exacta

Esta decisión desarrolla la cláusula de parada de la
[enmienda de presupuesto y topología](enmienda_f0_h0b_m38_presupuesto_real_topologia_2026-08-02.md)
y sustituye únicamente estos cuatro extremos de las enmiendas M38 anteriores:

1. «captura exacta de cuatro» pasa a ser captura exacta de cinco fuentes;
2. la prohibición del quinto auxiliar se sustituye por un único supervisor Go
   capturado, sin configuración o fuente viva posterior;
3. el write-set común de runner y adaptador se sustituye por los write-sets
   separados de Q5a, C4b-2, C4b-3 y C4c definidos aquí;
4. la prohibición de un supervisor intermedio se conserva para la relación
   paterna del caso, pero se permite un supervisor lateral Go: es hijo directo
   adicional del runner, nunca padre, abuelo o ejecutor del Bash del caso.

Permanecen vigentes la semántica de señales de C4b-1, la topología H0b, el
oráculo, las causales, D2c, D2d, el capturador y todas las demás puertas.

## Motivo del alto

El corte estable `91cb804` conserva esta medida reproducible:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 769 | se acreditará de nuevo sobre cada candidato |
| Adaptador privado de ciclo | 527 | `d9b61a183e5a32c321a3eeb48483ce40c83551bc7a700354ccc88e8206d9ee1f` |

El checkpoint 540 deja trece líneas, pero C4b-2 necesita al menos 48 netas
después de sustituir la lógica incompleta: estado provisional, identidad
física, recuperación cardinal, plazo, extinción y mutantes. Comprimir
sentencias, retirar controles o consumir la reserva de C4c queda prohibido.

El primer documento `f7ca3a9` propuso otro auxiliar Shell y obtuvo tres
revisiones `NO-GO`. Sus hallazgos son vinculantes para esta corrección:

- no distinguía el temporal global de captura del temporal de cada caso;
- permitía más de un iniciador de limpieza y write-sets solapados;
- no contenía un ledger reproducible de líneas;
- una relectura `/proc → kill` no cierra la carrera entre comprobación y
  señalización si el PID o PGID se recicla.

No se escribió código en aquel intento. `bash -n`, ShellCheck,
`git diff --check` y las huellas de H0b, D2c, D2d y capturador siguieron
verdes.

## Hecho técnico que cambia el diseño

`/proc/PID/stat` aporta PID, estado, PPID, grupo e instante de inicio, pero una
lectura por ruta no fija el proceso durante el syscall posterior. La
[documentación de procfs](https://docs.kernel.org/filesystems/proc.html)
advierte expresamente que mantener abierto `/proc/PID` no impide reutilizar el
PID. Por ello `starttime` sirve para rechazar una sustitución, no para cerrar
por sí solo la ventana hasta `kill`.

Linux proporciona `pidfd_open` y `pidfd_send_signal`: el descriptor es una
referencia estable y la señal falla si el proceso original ya terminó, en vez
de alcanzar otro PID. Son la frontera adoptada. Las referencias técnicas son
[pidfd_open(2)](https://man7.org/linux/man-pages/man2/pidfd_open.2.html),
[pidfd_send_signal(2)](https://man7.org/linux/man-pages/man2/pidfd_send_signal.2.html)
y [proc_pid_stat(5)](https://man7.org/linux/man-pages/man5/proc_pid_stat.5.html).
`pidfd_send_signal` actúa sobre un proceso, no sobre un grupo; el supervisor
mantiene abierto el pidfd del líder y detenido al líder mientras señaliza el
grupo. Nunca usa un PGID después de cerrar esa referencia.

La espera mantiene el contrato documentado por
[Bash](https://www.gnu.org/software/bash/manual/html_node/Job-Control-Builtins.html):
`wait -f` espera terminación y una interrupción deja sin asignar la variable
de `wait -n -p`. La implementación debe distinguir interrupción de estado
terminal y no reiniciar el plazo.

## Alternativa adoptada

Se incorpora una sola fuente Go nueva:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
```

Se captura en la misma operación que las cuatro fuentes actuales, se valida
por ruta y SHA-256 literal, se compila desde la copia privada con el toolchain
Go ya acreditado y se ejecuta únicamente como binario privado del temporal de
snapshot. No se vuelve a abrir la ruta viva.

El supervisor es un proceso lateral. El runner continúa siendo padre directo
del Bash del caso y del supervisor. El supervisor no interpreta caso, oráculo,
causal, `RESULTADO`, SQL, Docker o RRHH; solo mantiene la referencia pidfd,
aplica el plazo y extingue el grupo.

Se rechazan:

- otro auxiliar Shell que mantenga `leer /proc; kill`;
- `setsid`, `nohup`, `disown`, `pkill`, `killall` o búsqueda por nombre;
- abrir pidfd mediante una dependencia dinámica no fijada;
- degradar a PID/PGID tradicional si pidfd no está disponible;
- un sexto fuente para pruebas o para el protocolo.

El perfil ya exige Linux/amd64. Q5a debe comprobar en una autoprueba local que
el kernel admite ambos syscalls; ausencia o denegación devuelve 65 antes de
`CONT`, Docker del caso o SQL.

## Cinco fuentes y orden de composición

La captura única contiene exactamente:

| Orden de carga/compilación | Fuente | Tratamiento |
| ---: | --- | --- |
| 1 | D2c `arnes_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 2 | D2d `operaciones_runner_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 3 | H0b `arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 4 | `ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 5 | `supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go` | `vet`, compilación y autoprueba |

El manifiesto conserva su orden canónico por ruta y se coteja contra los cinco
pares literales antes de cargar o compilar cualquiera. El orden de manifiesto
no se usa como orden implícito de dependencias.

Cada `source` recibe `VEC_F0_CARGA_PRIVADA=1` inmediatamente antes, consume la
marca y usa solo `${snapshot}/${ruta}`. La fuente Go se copia desde ese mismo
snapshot a una ruta privada de compilación, se vuelve a cotejar por forma y
huella, se compila con red y módulos externos deshabilitados y el binario queda
modo 0700 dentro del temporal acreditado.

Las cuatro ejecuciones Shell directas devuelven 64. La fuente Go no es un
script: `go vet`, `go build`, su parser cerrado y `--autoprueba` deben ser
verdes; cualquier otro modo o argumento devuelve 64 sin efectos.

La validación del runner puede usar arrays literales y bucles para recuperar
presupuesto, pero no glob, descubrimiento, configuración ni cardinalidad
abierta.

## Dos temporales y handoff de autoridad

Se distinguen sin ambigüedad:

1. `temporales`: raíz global del arnés, creada antes de compilar el capturador
   y las cinco fuentes; contiene el snapshot y vive toda la matriz;
2. `ruta_caso_m38`: temporal exclusivo de un caso, creado solo después de
   cargar y autoprobar las cinco fuentes.

Antes del handoff no puede existir identidad de caso, contenedor, trabajo,
supervisor ni SQL. La trampa del runner solo puede retirar la raíz global
acreditada mediante su primitiva bootstrap actual. Tras cargar las cinco
fuentes se arma una marca irreversible `bootstrap → ciclo`; desde entonces la
trampa no invoca directamente Docker, D2d ni borrado de caso: llama una sola
vez a `retirar_recursos_m38_f0` del adaptador.

Esta autoridad es secuencial, no concurrente. El bootstrap nunca conoce un
recurso de caso; el adaptador nunca adopta una raíz global no acreditada. Un
fallo durante el handoff devuelve 65 y usa todavía la autoridad bootstrap.

## Grafo único de responsabilidades

### Runner

Conserva:

- parser, selector, catálogo, oráculo y causal `0/64/65/79`;
- ticket, conductor, `RESULTADO` y validación final;
- relación paterna directa con Bash y supervisor;
- trampa de señales C4b-1 y estado de handoff;
- decisión e invocación única de `finalizar_h0b_f0` para F01..F15;
- una invocación posterior del epílogo exterior del adaptador;
- captura, validación, compilación y carga de las cinco fuentes.

El runner no vuelve a invocar `finalizar_h0b_f0` desde el epílogo, no ejecuta
`docker run`, `docker rm` o retirada de caso después del handoff y no
desarma identidades de proceso.

### Adaptador privado de ciclo

`ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh` conserva:

- régimen Shell y protocolo de control con el supervisor;
- preparación y retirada exactas de temporal de caso y contenedor;
- transporte por FD y lanzamiento directo del Bash;
- materialización, identidades y acciones interiores H0b;
- una sola primitiva pública e idempotente `retirar_recursos_m38_f0`.

Esa primitiva reconcilia y espera primero Bash y supervisor, después invoca el
primitivo mecánico D2d `retirar_contenedor_propio_f0`, acredita la postausencia
y finalmente retira el temporal de caso. D2d permanece byte a byte inmutable y
subordinado: no es otra autoridad. El supervisor nunca llama al finalizador
interior, Docker o el adaptador.

### Supervisor Go

Posee exclusivamente:

- parser de protocolo y límites del canal local;
- `pidfd_open` del Bash detenido y revalidación posterior de
  `PID|estado|PPID|PGID|SID|starttime`;
- señal `CONT` al líder exacto mediante `pidfd_send_signal`;
- reloj monotónico absoluto de 180 segundos iniciado antes de `CONT`;
- recepción de una única orden de cancelación por canal local, no por PID;
- escalado seguro y prueba de extinción del grupo;
- recibo mecánico final sin datos del caso ni del resultado SQL.

No es padre del Bash, no ejecuta `wait`, Docker, Shell, SQL ni otro proceso. El
runner hace `wait -f` únicamente sobre sus dos hijos directos, Bash y
supervisor; nunca intenta recolectar descendientes.

## Protocolo de procesos C4b-2

1. El adaptador exige tabla de trabajos vacía y arma `provisional` antes del
   fork del Bash.
2. La ventana `fork → $!` se recupera por cardinalidad exacta cero/uno; más de
   uno es incidente 65 y no se escoge PID.
3. El Bash válido instala el bootstrap revisado, desactiva `monitor` y se
   detiene antes de cualquier orden externa.
4. El padre captura una tupla estable
   `PID|T|PPID|PGID=PID|SID|starttime`; el parser de `stat` trata `comm` entre
   paréntesis y rechaza lectura truncada, desaparecida o cambiante.
5. Solo entonces se crean dos canales locales privados, se lanza el supervisor
   lateral y se registra su PID como segundo trabajo esperado.
6. El supervisor abre pidfd, vuelve a leer y cotejar la tupla y confirma
   disponibilidad. Una sustitución o falta de syscall devuelve 65 sin `CONT`.
7. Arma el plazo monotónico y envía `CONT` mediante pidfd. No existe un
   `sleep 180` Shell ni un segundo plazo reiniciable.
8. El runner espera sus dos hijos. Una señal solo enclava C4b-1; al despertar,
   escribe una única orden autenticada por posesión del descriptor/canal. No
   señaliza el PID del supervisor.
9. En terminación normal, el supervisor acredita que el grupo no conserva
   miembros y emite recibo. Si quedan miembros, inicia la misma extinción.
10. En cancelación o plazo, mantiene abierto el pidfd, detiene al líder exacto
    por pidfd, detiene el grupo mientras el líder fija el PGID, vuelve a
    inventariar sus miembros, envía `TERM`, concede como máximo dos segundos y
    aplica `KILL` antes de cerrar el pidfd. No señaliza un PGID tras cerrar la
    referencia ni acepta un miembro nuevo sin repetir la acreditación.
11. El runner ejecuta `wait -f` sobre supervisor y Bash, conserva ambos estados
    reales y exige recibo exacto, grupo sin miembros, canales cerrados y tabla
    de trabajos vacía antes de pasar a `terminal`.
12. Un fallo de pidfd, canal, inventario, señal, espera o postausencia fuerza 65
    y detiene la matriz; nunca degrada a un `kill` tradicional no ligado.

El supervisor aplica `PR_SET_PDEATHSIG` y comprueba de nuevo el PPID del runner
para que una muerte paterna active la extinción. El protocolo no contiene
secreto ni dato funcional, pero sus nodos FIFO o socket se crean con modo
privado, identidad física y pre/postausencia exactas.

## Presupuesto reproducible

El nuevo reparto evita mover 151 líneas Shell y evita agrandar el runner con
mecanismo de procesos:

| Corte y movimiento | Runner | Adaptador | Supervisor Go |
| --- | ---: | ---: | ---: |
| Base `91cb804` | 769 | 527 | 0 |
| Q5a: arrays de cinco, `vet/build/autoprueba` | 763..775 | 527 exactas | 220..320 |
| C4b-2: retirar 37 líneas de espera/señal incompleta y añadir puente cerrado | ≤775 | 508..522 | 300..420 |
| Reserva C4b-3: Docker, canales y epílogo | ≤775 | +12..18 | +0..30 |
| Cierre C4b | ≤770 objetivo, 775 máximo | 520..540 | ≤450 objetivo, 500 máximo |
| Reserva C4c | sin crecimiento no decidido | hasta 40 | inmutable |
| Cierre C4c | nueva revisión | ≤580 | misma huella de C4b |

Checkpoints obligatorios:

- Q5a se detiene si el runner supera 775, el adaptador cambia o el supervisor
  supera 320;
- C4b-2 se detiene si el adaptador supera 522 o el supervisor supera 420;
- C4b-3 se detiene antes de superar adaptador 540 o supervisor 500;
- C4c no toca el supervisor y se detiene antes de adaptador 580.

No se agrupan sentencias, pruebas o mutantes para caber. Superar un checkpoint
exige nueva decisión antes de continuar.

## Write-set por minitarea

| Minitarea | Write-set productivo/probatorio |
| --- | --- |
| Q5a | runner + nuevo supervisor Go; adaptador, H0b, D2c, D2d y capturador invariantes |
| C4b-2 | runner + adaptador + supervisor Go |
| C4b-3 | runner + adaptador + supervisor Go |
| C4c | runner + adaptador; supervisor Go byte a byte inmutable |

La documentación y actas pertenecen a dirección. Productor y revisor son
distintos. Ningún candidato se integra con un `NO-GO`.

## Mutantes obligatorios

### Captura y composición

- cardinalidad 4/5/6, duplicado, quinta ausente y huella/ruta quinta mutada;
- modificación de la ruta viva después de capturar: la copia no cambia;
- sustitución de la copia entre cotejo y `vet/build`: fallo cerrado;
- doble captura, binario sustituido, modo no privado y autoprueba omitida;
- cuatro marcas Shell ausentes, reutilizadas o consumidas dos veces;
- modo Go desconocido: 64 y cero temporal de caso, Docker, procesos o SQL.

El mutante de quinta fuente puede crear la raíz global de snapshot porque la
captura vive en ella; debe fallar antes de `ruta_caso_m38`, Docker, procesos o
SQL y retirar después la raíz global.

### Proceso y reciclado

- señal/fallo antes del fork, `fork→$!` y después de captura;
- cardinalidad de trabajos 0/1/>1 en provisional;
- `comm` con espacios/paréntesis, `stat` truncado, estado, PPID, PGID, SID e
  inicio adversos;
- PID reciclado antes y después de `pidfd_open`;
- señuelo que recibe el mismo número tras la lectura: ninguna señal al señuelo;
- pidfd ausente, cerrado, sustituido o referido a otro proceso;
- caída del líder con descendiente durante la gracia: el pidfd abierto conserva
  la ligadura; ninguna señal alcanza un grupo reciclado;
- miembro nuevo entre inventario y señal: nueva acreditación o incidente 65;
- `wait -f` interrumpido repetidamente y estado terminal real 0/65/79/130/143;
- fin en el borde de 180 segundos, timeout, cien interrupciones y salto del
  reloj civil sin ampliar el plazo monotónico;
- Bash y descendiente resistentes a TERM, grupo ya vacío y fallo de KILL;
- supervisor termina antes, canal roto, orden duplicada y epílogo reentrante;
- muerte del runner: `PDEATHSIG`, extinción o incidente operable, nunca falso
  verde.

Cada mutante prueba ambos hijos directos recolectados, pidfd cerrado después de
la última actuación, PGID sin miembros, canales y trabajos cero, temporal de
caso retirado y ausencia de siguiente caso si hay incidente.

## Puertas estáticas y dinámicas

Además de Bash, ShellCheck, Go, `git diff --check`, Gitleaks, tamaños, H0
PostgreSQL 18.4 y C4b-1, se exige:

- manifiesto exacto de cinco y ninguna reapertura de ruta viva;
- `go vet`, compilación sin red y `--autoprueba` del supervisor;
- búsqueda que rechace `kill -PGID` o `kill PID` en runner/adaptador para el
  hijo del caso; la señalización vive en el supervisor y usa pidfd;
- búsqueda que demuestre un único call graph
  `trampa → finalizar interior una vez → retirar_recursos_m38_f0 una vez →`
  `supervisor/D2d/temporal`;
- D2d contiene el mecanismo `retirar_contenedor_propio_f0`, pero solo el
  adaptador lo invoca tras el handoff; runner y supervisor no ejecutan
  `docker rm`;
- hashes invariantes de H0b, D2c, D2d y capturador;
- SHA del supervisor fijado tras C4b como invariante de C4c;
- matriz de mutantes con prueba positiva de ejecución y residuos cero.

## Secuencia

1. Dos contrarrevisiones documentales de esta corrección.
2. Q5a: captura, compilación y autoprueba inertes; H0 nominal sin cambios.
3. Revisión independiente de Q5a.
4. C4b-2: protocolo pidfd, plazo, extinción y mutantes sin Docker.
5. Doble revisión independiente de C4b-2.
6. C4b-3: Docker, temporal y epílogo único.
7. C4c y C4d según el DAG vigente.

Q5a no cierra C4b-2. C4b-2 no cierra C4b ni H0b. Ningún checkpoint modifica
F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva
`1/14` ni el `NO-GO` de producción.

## Criterio de aprobación

Dos revisores distintos deben confirmar:

1. snapshot único de cinco sin ruta viva posterior;
2. relación paterna directa y único supervisor lateral limitado a procesos;
3. pidfd obligatorio sin caída a PID/PGID no ligado;
4. call graph único de finalizador interior y epílogo exterior;
5. dos temporales y handoff sin autoridad simultánea;
6. ledger con reserva real de C4b-3 y C4c;
7. write-sets disjuntos e invariantes por fase;
8. conservación íntegra de C4b-1, H0b, D2c, D2d y capturador.

Un solo `NO-GO` mantiene detenido el código y obliga a corregir otra vez la
decisión.

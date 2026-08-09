# Decisión F0-H0b/C4b-2/G2-O/O3a: arranque y mapa FD

Fecha: 9 de agosto de 2026.
Estado: **CANDIDATO V13, doble `GO` técnico (`P0=P1=P2=0`), sin autorización de integración**; R sigue intacto por orden de dirección y no captura las cinco fuentes nuevas, por lo que el acta detiene el corte antes de CI e integración.

## Cierre de la primera falsación

La primera ronda recibió `NO-GO` porque el runner creaba la pipeline externa `/usr/bin/env -0 | /usr/bin/grep ...` antes de leer FD 9 y del auto-`STOP`.
El corte independiente
[`O3a-P1: barrera temprana del ticket`](decision_f0_h0b_c4b2_g2o_o3a_p1_barrera_ticket_temprana_2026-08-09.md)
movió byte a byte esa comprobación detrás de la barrera. Quedó publicado en `ce0848ed4332c746d6a908673f3b3cad9cd90c1b`; la CI `31298943127` terminó con cinco de cinco puertas verdes.

Este documento sustituye íntegramente el diseño detenido. Incorpora los hallazgos funcionales y de seguridad: dos pidfd explícitos y el handle opaco de Go dentro de O3a, terminalidad en la segunda barrera, máquina total, primera observación inmutable, propiedad exhaustiva de FD, entorno cerrado, retirada total, fatalidad no retornable y prohibición de `close_range` en el proceso Go.

## Resultado único

O3a transforma un control O2b válido en S3, en una vuelta acreditada posterior a `INICIAR` y sobre la misma goroutine fijada a un hilo del sistema, en un único Bash provisional arrancado mediante `exec.Cmd`. El hijo recibe un mapa FD cerrado y queda bloqueado en un pipe de ticket vacío antes de ejecutar comandos externos, auto-detenerse o producir efectos.

El resultado positivo es un agregado opaco, no exportado y de propiedad única que conserva juntos controlador S3, `exec.Cmd`, proceso con handle pidfd opaco, pidfd primario/reserva, escritor ticket y capacidades O3b. El negativo anterior a `Start` deja cero Bash. Un fallo posterior ejecuta retirada acotada y no devuelve hijo sin propietario; si no puede acreditarse, la rama es fatal y no retorna.

O3a no escribe el ticket, no acredita `STOP` ni identidad `/proc`, no envía
`CONT`, no fija el plazo de caso, no escribe TERMINAL ni activa el modo operativo.

## Base publicada y autoridad

La base técnica exacta es
`ce0848ed4332c746d6a908673f3b3cad9cd90c1b`, publicada en la rama
`integracion/ct-o4-04e-20260726`. Su CI `31298943127` terminó con cinco de
cinco puertas verdes. Contiene sin reescritura el material O3a-P1
`a1aeab7ed2a884d0fb10c265bd11982d3519ce49` y su evidencia
`f3a1e961849018e8cf8bb0aa38d3d43006b1fd44`.

El relevo factual posterior está en
`de9b98e9dae14c6a9812adda35694bac5838b191`; no cambia bytes técnicos. Su CI
`31299739676` terminó `success` con cinco de cinco trabajos verdes. Esta puerta
factual queda satisfecha, pero no sustituye ninguna revisión semántica ni
autoriza por sí sola la proyección.

Prevalecen, en este orden para sus responsabilidades respectivas:

1. el contrato canónico O1a de framing y dominios;
2. O1b, lector y codec;
3. O2a, recepción y retención única del sobre;
4. O2b, control previo hasta S3 o S5;
5. O3a-P0, margen estructural del runner;
6. O3a-P1, barrera de ticket y auto-`STOP` anterior a comandos externos;
7. la decisión C4b-2 para reserva pidfd inmediata y propiedad terminal;
8. esta decisión, para el conteo físico real de pidfd de Go 1.26.5, arranque
   provisional y mapa FD.

La especificación conjunta G2-O de 5 de agosto permanece en `NO-GO` y es
únicamente antecedente. Esta decisión no la reactiva en bloque.

## Cuatro precisiones de prevalencia

### Comando privilegiado

La decisión C4b-2 aceptada expresa en prosa `/usr/bin/bash -p`, el runner real
exige que la opción `p` esté activa, pero su antiguo literal de `Cmd.Args`
omitió `-p`. Esta decisión sustituye exclusivamente ese literal:

```text
Path = /usr/bin/bash
Args = [
  /usr/bin/bash,
  -p,
  /proc/self/fd/8,
  --caso-inyeccion-h0b,
  SELECTOR_VALIDADO
]
```

No se usa `LookPath`, `exec.Command`, `-c`, una ruta recibida ni una ruta viva
del repositorio.

### Ticket antes de STOP

El runner lee primero FD 9, valida el sobre y solo después ejecuta
`kill -STOP` sobre sí mismo. Por tanto el reparto anterior «O3b acredita STOP»
y «O3c escribe ticket» es físicamente circular. Esta decisión lo corrige:

- `Start` entrega el pidfd primario y Go conserva además un handle pidfd opaco
  dentro de `os.Process`; el primer syscall posterior del llamador crea la
  reserva explícita con `F_DUPFD_CLOEXEC`;
- O3b recibe `os.Process` y las dos referencias explícitas, revalida la
  barrera, escribe y cierra el ticket, acredita el auto-STOP y la identidad
  `/proc`; no vuelve a duplicar;
- O3c fija el plazo monotónico, envía el primer `CONT` y observa salida natural.

O3a conserva el escritor sin escribir un byte. Cerrar ese escritor antes de
la entrega provoca EOF y evita `STOP`, descendientes y efectos.

### Modelo pidfd real de Go 1.26.5

Con `SysProcAttr.PidFD` no existen solo dos referencias físicas. Antes de que
`cmd.Start()` retorne, `os.startProcess` duplica internamente el pidfd para el
handle privado de `os.Process`. El conjunto verde es exactamente: primario
explícito, reserva explícita y handle opaco de `Process`, los tres de la misma
identidad y `CLOEXEC`. El handle nunca se extrae, cierra ni señaliza por
separado; `cmd.Wait` lo libera. El inventario lo cuenta como la única tercera
referencia pidfd aparecida durante `Start`.

Antes de recibir el bundle, un preflight ejecuta una vez `os.FindProcess(os.Getpid())`, un `WithHandle` literal que valida `CLOEXEC` y
señal cero sobre sí mismo, y `Release` para acreditar/calentar la sonda
pidfd de la biblioteca estándar. Solo ese preflight anterior a O3a ejecuta dos
`pidfd_open` sobre el supervisor —sonda y handle de `FindProcess`— y un clon
interno efímero; los tres quedan cerrados/recogidos antes del baseline, sin
recursos del caso. En O3a no se admite `pidfd_open`, `WithHandle` ni callback.

### TERMINAL regular y ausencia de ACK

La enmienda O1a posterior ya sustituyó expresamente dos elementos de C4b-2:
FD 8 dejó de ser pipe de recibo y pasó a ser el fichero TERMINAL regular 0600;
`ACK_LISTO` y `ACK_CASO` desaparecieron del protocolo. Esta decisión mantiene
esa prevalencia. O5 deberá crear el fichero y su lector exterior; O3a conserva
solo el escritor. Ninguna fase reintroduce un ACK vivo.

## Ledger inicial obligatorio

| Alias | Unidad | Líneas | SHA-256 |
| --- | --- | ---: | --- |
| R | Runner H0 | 702 | `7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487` |
| D | D2d | 264 | `681efbbd7f856eb539d1656cffed87c26f48609e65d6d6adf8265c350ae69442` |
| G1 | Supervisor privado | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 | Control previo O2b | 404 | `2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744` |
| G5 | Pruebas O2b | 507 | `10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867` |
| G2 | Lector y operativo | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | Sobre S0 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| C | Capturador | 799 | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |
| A | Adaptador M38 | 527 | `98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7` |
| B | Binario Go | — | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` |

## Write-set proyectado y presupuesto

La futura proyección puede tocar solo siete rutas materiales:

1. R, para capturar cinco fuentes nuevas y ampliar manifiesto, `vet`, build y huellas;
2. G1, únicamente para llamar a la nueva autoprueba después de activar el subreaper;
3. G6a nueva, única definidora de tipos/métodos/registro de autoridad, estado, reloj, preflight, lease y observador; O5 será el único dueño productivo de su instancia y G7a solo ejercita una fixture por los mismos métodos; G6b prepara y G6c avanza/`Start`/retira;
4. G7a solo fixtures/positivos; G7b, única agregadora, llama G7a, ejecuta negativos/estrés/residuos y limpieza test-only cerrada; G1 llama solo G7b tras subreaper.

Aristas permitidas: G1→G7b; G7b→G7a/G6a-b-c; G7a→G6a-b-c; G6a→G4; G6b→G6a; G6c→G6a-b. No hay inversas, ciclos ni G6→G7. Solo métodos G6a escriben autoridad/estado; G6b-c/G7 no definen sus literales ni campos. G7 no define `init`, global mutable, función variable, hook/callback/mock ni llama `Start`. Solo G7b consume linealmente el agregado test-only: cierra ticket sin escribir, acredita terminalidad, hace un `Wait`, cierra pidfd explícitos/CONTROL/TERMINAL y fixtures, libera `leaseGuardia`/observador y bloquea segundo consumo, sin señal; es inalcanzable operativo.

Nombres previstos:

```text
supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go
supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go
supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go
supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go
supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go
```

D, G2, G3, G4, G5, capturador, adaptador, Shell auxiliares, SQL, migraciones y
resto del repositorio permanecen byte a byte. El manifiesto pasa de nueve a
catorce entradas y el build de cinco a diez fuentes Go.

| Unidad | Base | Objetivo | Parada |
| --- | ---: | ---: | ---: |
| R | 702 | 728–738 | 740 |
| G1 | 692 | 695–698 | 700 |
| G6a | 0 | 470–530 | 560 |
| G6b | 0 | 350–470 | 500 |
| G6c | 0 | 400–520 | 550 |
| G7a | 0 | 450–600 | 650 |
| G7b | 0 | 500–650 | 700 |

Estas cifras son presupuesto, no validación de un prototipo ni ledger posterior. Antes de autorizar código
se debe proyectar fuera del repositorio, fijar líneas y SHA-256 exactos de R,
G1, G6a--c, G7a--b y binario, reproducir dos builds Go 1.26.5 y volver a revisar el
documento completo. No se actualiza una huella para acomodar una deriva.

## Máquina y resultados totales

G6a define estados privados, no serializables y sin valores aceptados desde el
exterior:

```text
A0 observando
  -> A1 preparado | A7 retirado_sin_hijo
A1 preparado
  -> A2 aplazado | A3 iniciando | A7 retirado_sin_hijo
A2 aplazado
  -> A2 aplazado | A3 iniciando | A7 retirado_sin_hijo
A3 iniciando
  -> A4 provisional | A7 retirado_sin_hijo
A4 provisional
  -> A5 pidfd_tres | A8 retirando_con_hijo | AF fatal
A5 pidfd_tres
  -> A6 entregado | A8 retirando_con_hijo | AF fatal
A8 retirando_con_hijo
  -> A9 retirado_con_hijo | AF fatal
```

No hay transición desde `A6`, `A7`, `A9` o `AF` a un estado iniciable. `Start`
total es cero o uno. `AF` es un estado interno no retornable, no una variante
de resultado. Un alias consumido solo puede devolver el mismo error estable de
uso sin tocar recursos ya transferidos ni ejecutar otro `Start`.

Los resultados forman una unión discriminada cerrada:

- `preparado`: custodia viva, cero `Start`;
- `aplazado`: custodia viva por fragmento CONTROL, cero `Start`;
- `entregado`: agregado opaco completo, exactamente un `Start`;
- `retirado`: origen `sin_hijo`, `con_hijo` o `uso_consumido`, primera
  observación inmutable, vector de propiedad propio del origen y cero
  hijo/zombi.

Una combinación de discriminante, estado, recursos o error distinta de esas
cuatro es un fallo de invariante. No existe una tupla `nil, nil` ambigua. La
rama `AF` llama directamente a `fatalO3aM38`, cuya única terminación es
`os.Exit(65)`; no escribe log ni hace E/S antes de salir. Un subproceso exterior
acredita el estado 65 y el EOF, y nunca recibe un valor `fatal`.

## Reloj y testigo de vuelta

G4 permanece byte a byte. G6a envuelve su llamada sin duplicar parser ni
estado. Define un `relojVueltaM38` de propiedad única y un
`*testigoVueltaM38` que apunta a una celda única registrada por ese reloj. La
celda contiene autoidentidad, identidad del reloj, contador `uint64`, TID y un
bit de consumo atómico:

1. el único emisor incrementa una vez al comienzo de cada vuelta exterior;
2. la envoltura G6a llama a `controladorPreinicioM38.consumir` y, si observa S3,
   consume mediante CAS y conserva el token de esa vuelta dentro de
   `preparado`;
3. `avanzar` exige otro testigo del mismo reloj y TID, no consumido y con
   contador estrictamente mayor, registrado y consumible una sola vez por CAS;
4. un alias de puntero es el mismo token: el primer uso puede ser válido y los
   siguientes fallan por el bit compartido; un clon de valor, literal, token
   sin autoidentidad, no registrado, ajeno, agotado o del mismo turno retira
   sin `Start`.

El reloj no es global, no se serializa ni se acepta como entero crudo. La
composición O5 futura deberá usar esta misma autoridad. G7b agrega G7a y la
ejercita sin habilitar el modo operativo.

## Frontera de entrada

La goroutine propietaria ya está fijada mediante `runtime.LockOSThread` y no
puede liberar el hilo hasta O4c. O3a acredita el mismo TID en observación,
preparación, ambas barreras, `Start`, duplicación y entrega.

La entrada exige:

- controlador O2b no nulo, exactamente S3, sin causa ni fallo;
- receptor y sobre retenidos por O2b, sin copia adicional del ticket, nonce,
  identidad o sobre completo;
- lector CONTROL exacto, limpio en L0, sin fragmento pendiente;
- token consumido de la vuelta que produjo S3, almacenado en `preparado`; el
  testigo posterior se exige únicamente al entrar en `avanzar`;
- `PR_SET_CHILD_SUBREAPER` ya activado y comprobado;
- observador de señal real registrado por O5, ligado a TID/generación y con
  signo/contador atómicos; `PR_SET_PDEATHSIG(SIGTERM)` y PPID ya comprobados;
- ningún provisional, hijo o agregado anterior;
- siete capacidades exclusivas del supervisor: roles raíz 3, runner privado 4,
  salida 5, error 6, CONTROL lector 7, TERMINAL escritor 8 y SOBRE lector 9 ya
  consumido hasta EOF limpio.

Esos números son roles lógicos del protocolo Shell, no números exigidos en la
tabla FD del proceso Go. Los siete `*os.File` de entrada tienen números físicos
arbitrarios, distintos y acreditados. Solo los índices de `Stdin/Stdout/Stderr`
y `ExtraFiles` fijan los números 0..9 del Bash hijo; O3a nunca usa `dup2` para
renumerar la tabla del padre.

O5 garantiza que no conserva otro escritor ni un alias de la misma descripción
abierta. Solo retiene lectores separados, abiertos independientemente y
`CLOEXEC`, para salida, error y TERMINAL; nunca se heredan al Bash. La llamada
recibe un `*bundleEntradaO3aM38`; su primera operación copia el bundle a
custodia privada y pone a cero todos los campos del llamador. Desde ese punto
O3a posee el conjunto completo incluso si falla una validación. O3a solo
acredita ausencia de alias entre las siete capacidades recibidas: números de
FD distintos y tuplas físicas `(tipo, dispositivo, inodo)` no repetidas.

O5 crea y acredita el origen; O3a vuelve a ejecutar `F_GETFD`, `F_GETFL` y
`fstat`, compara con el sobre/baseline retenido y acepta solo esta tabla
cerrada:

`formaRunnerM38` contiene la identidad, SHA y tamaño del runner proyectado;
O5 la sella y G6b la recibe, nunca compila ese SHA. La huella posterior vive
solo en manifiesto, ledger y evidencia, evitando el ciclo R↔G6b.

| Rol supervisor | Clase física exacta | Acceso/flags | Propietario, modo, enlaces, tamaño y offset |
| ---: | --- | --- | --- |
| 3 raíz | directorio acreditado, identidad igual a `formaRaizM38` retenida | `O_RDONLY`, `FD_CLOEXEC` | EUID, modo y enlaces iguales al baseline sellado; tamaño/offset no autorizan |
| 4 runner | regular privado, identidad, SHA y tamaño iguales a `formaRunnerM38` sellada | `O_RDONLY`, `FD_CLOEXEC`, sin `O_APPEND/O_NONBLOCK` | EUID, `0600`, un enlace, offset 0 |
| 5 salida | regular privado exclusivo | `O_WRONLY`, `FD_CLOEXEC`, sin `O_APPEND/O_NONBLOCK` | EUID, `0600`, un enlace, tamaño 0, offset 0 |
| 6 error | regular privado exclusivo y distinto de salida | `O_WRONLY`, `FD_CLOEXEC`, sin `O_APPEND/O_NONBLOCK` | EUID, `0600`, un enlace, tamaño 0, offset 0 |
| 7 CONTROL | pipe anónimo, extremo lector | `O_RDONLY|O_NONBLOCK`, `FD_CLOEXEC` | EUID, `0600`, un enlace; tamaño/offset no autorizan |
| 8 TERMINAL | regular privado exclusivo | `O_WRONLY`, `FD_CLOEXEC`, sin `O_APPEND/O_NONBLOCK` | EUID, `0600`, un enlace, tamaño 0, offset 0 |
| 9 SOBRE | pipe anónimo, extremo lector en EOF limpio | `O_RDONLY|O_NONBLOCK`, `FD_CLOEXEC` | EUID, `0600`, un enlace; tamaño/offset no autorizan |

Cualquier socket, dispositivo, directorio fuera del rol raíz, fichero compartido,
modo, propietario, enlace, tamaño, offset, acceso o flag distinto se rechaza.
Para stdin y los cuatro fillers, O3a abre por separado la ruta literal
`/dev/null` con `O_RDONLY|O_CLOEXEC|O_NOFOLLOW`, y cada FD debe ser dispositivo
de caracteres con `st_rdev` Linux `1:3`; de otro modo retira. No abre raíz,
runner, salida, error, terminal ni repositorio por ruta.

## Propiedad y cierre de FD

| Recurso | Entrada | Preparado/aplazado | Tras `Start` en padre | Entregado | Retirada/fatal |
| --- | --- | --- | --- | --- | --- |
| raíz original, rol 3 | bundle | cerrado tras duplicar | cerrado | no | cerrado |
| runner original, rol 4 | bundle | cerrado tras duplicar | cerrado | no | cerrado |
| salida original, rol 5 | bundle | cerrado tras duplicar | cerrado | no | cerrado |
| error original, rol 6 | bundle | cerrado tras duplicar | cerrado | no | cerrado |
| CONTROL supervisor, rol 7 | bundle | O3a | O3a | agregado | custodia negativa o cierre de proceso fatal |
| TERMINAL escritor, rol 8 | bundle | O3a | O3a | agregado | custodia negativa o cierre de proceso fatal |
| SOBRE supervisor, rol 9 | bundle, EOF | cierre antes de barrera | cerrado | cerrado | cerrado |
| duplicado raíz para hijo FD 7 | no existe | O3a/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrado |
| duplicado runner para hijo FD 8 | no existe | O3a/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrado |
| duplicado salida para stdout | no existe | O3a/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrado |
| duplicado error para stderr | no existe | O3a/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrado |
| stdin `/dev/null` | no existe | O3a/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrado |
| fillers `/dev/null` 3--6 | no existen | cuatro FD distintos/CLOEXEC | O3a hasta revalidación de barrera dos | no | cerrados |
| lector ticket Bash FD 9 | no existe | O3a/CLOEXEC | cerrado tras las tres referencias pidfd | no | cerrado |
| escritor ticket Bash | no existe | O3a/CLOEXEC | O3a | agregado | cerrado primero o por salida fatal |
| pidfd primario/reserva | no existen | no existen | O3a/CLOEXEC | ambos juntos | ambos cerrados o por salida fatal |
| handle pidfd opaco de `Process` | no existe | no existe | Go/O3a, CLOEXEC | dentro de `Process` | liberado por `Wait` o salida fatal |
| `Cmd`/`Process` | no existen | `Cmd`, proceso nulo | O3a | agregado opaco | recogido o salida fatal |
| acreditación preflight registrada | O5; G7a solo fixture | consumida/eliminada antes de validar | no existe | no | retirada o salida fatal |
| `leaseGuardiaO3aM38` registrada | O5; G7a solo fixture | O3a | O3a | agregado, sin alias | custodia negativa o salida fatal |
| observador de señal + baseline | O5; G7a solo fixture | O3a, contador monótono | O3a | agregado, sin alias | custodia negativa o salida fatal |

El rol 8 del supervisor es TERMINAL regular 0600, enlace uno y escritor único
en Go. No se confunde con el FD 8 del Bash, que es la copia privada del runner.
En toda rama que retorna, TERMINAL nunca se hereda, escribe ni cierra en O3a y
viaja en el agregado positivo o la custodia negativa para la fase terminal. En
`AF` lo cierra la terminación del proceso, sin promesa de custodia retornada.

La transferencia del bundle es incondicional al entrar; el llamador invalida
sus campos antes de que empiece cualquier validación y no conserva aliases.
Cada cierre ocurre una vez; un error de cierre se agrega como secundario sin
sustituir la primera observación. En `AF`, `os.Exit(65)` deja al kernel cerrar
los FD del supervisor; el contrato no presenta esa salida como custodia
negativa ni limpieza local acreditada.

## Preparación antes de Start

La API es bifásica:

1. `preparar` valida, toma propiedad y construye todos los recursos, pero no
   llama `Start`;
2. `avanzar` exige un testigo posterior, ejecuta la barrera y puede llamar
   `Start` una sola vez.

Una instancia de la autoridad concreta definida en G6a registra y emite cada
`*acreditacionPidfdGoM38` opaco, autoidentificado, ligado a TID/generación y consumible por CAS. O5 posee la instancia productiva y G7a solo una fixture. `preparar`
verifica pertenencia, lo elimina atómicamente del registro o retira; no se admite
forjarlo, construirlo ni calentar el preflight dentro de O3a.

Antes de la barrera se completan todas las operaciones potencialmente
falibles: duplicados privados con `F_DUPFD_CLOEXEC`, cinco aperturas distintas
de `/dev/null` con `O_CLOEXEC`, pipe-ticket con `pipe2(O_CLOEXEC)`,
`fstat`/`fcntl`, entorno, `exec.Cmd` y estructuras de propiedad. No se crea aún
un deadline de retirada. No se usa `dup` seguido de `fcntl`, `pipe` seguido de
`fcntl`, ni `close_range`.

Todo FD que solo pertenece al supervisor conserva `FD_CLOEXEC`. O3a no cambia
en bloque la tabla de descriptores del proceso Go, no crea goroutines y no
abre `/proc`. Una sonda hija acredita herencia exacta; los FD del runtime no se
mutan para hacer pasar el oráculo.

La clausura usa `maxFDInspeccionM38=1_048_576`, `minFDDuplicadoM38=10` y `RLIMIT_NOFILE` finito.
Fotografía `0..Cur-1` con `F_GETFD`, `F_GETFL`, `fstat` y `lseek` raw —offset o `ESPIPE`—.
Los FD 0/1/2 deben existir, quedan sellados por identidad/clase/acceso/flags y se revalidan; pueden carecer de `CLOEXEC` porque `Cmd` los sustituye.
Todo FD físico ajeno >=3 es basal y `CLOEXEC`; `EBADF` significa ausente y el snapshot se congela justo antes de `Start`.
El máximo es tope de rechazo, no promesa temporal: O5 fija el menor `Cur` suficiente y el borde que consume el bootstrap falla cerrado.
Tras reserva, el delta contiene primario/reserva conocidos y una alta opaca de igual identidad, los tres `CLOEXEC`; el mapa vive hasta handoff.
O5 garantiza cobertura hasta `Cur`, cero FD superior, cero concurrencia cooperativa ajena y cero cambio de `RLIMIT`.
G6a implementa la autoridad; O5 posee la instancia productiva y G7a una fixture dominada por `--autoprueba`, sin emisor alterno.
La lease liga TID, generación, tabla, límite, descripciones, metadatos y flags.
G6a `comenzar(op,pre,post)` hace CAS estable→pending y devuelve permiso opaco ligado a TID/generación; G6b/c ejecuta exactamente el syscall autorizado, sin callback.
G6a `consolidar(permiso,resultado)` compara/aplica post o revalida/restaura pre y retira/entra en `AF`; prohíbe permiso ajeno/reusado, segundo begin, retorno, observación o handoff pending.
Solo G6c ejecuta `Start`: (A) `err!=nil && Process=nil && pidfd=-1` restaura pre y cierra sin reserva; (B) cualquier otra tupla en la que falte `Process`, el PID no sea positivo o el pidfd no sea fiable consume pending→`AF` y llama fatal directo, sin reserva/E/S; (C) solo con esas tres autoridades consolida lógicamente sin syscall, hace begin de reserva y `F_DUPFD_CLOEXEC` como primer syscall, consolida su resultado y acredita cardinalidad tres.
O3a detecta deriva observable y retira; O5/O6 acreditan aislamiento frente a elusiones no observables.

O3a recibe sin recrear el `finBootstrapGoM38` monotónico fijado al entrar en el
supervisor: entrada más seis segundos, absoluto y no reiniciable. Constantes
exactas: `duracionRetiradaO3aM38 = 3 s` y `reservaO3bO3cM38 = 1 s`. Solo se
permite `Start` si la última lectura monotónica observa al menos cuatro
segundos hasta `finBootstrapGoM38`; así la retirada dispone de hasta tres
segundos. El handoff exige observar al menos un segundo y O3b vuelve a medir
antes de cualquier efecto. `preparar`, cada
`avanzar` y ambas barreras observan el límite. Un aplazamiento puede continuar
solo dentro de esa ventana; nunca crea ni reinicia otro deadline.

## Comando, entorno y mapa del Bash

`exec.Cmd` se construye mediante literal, con `Dir="/"`, `Stdin` en
`/dev/null`, `Stdout` en la salida acreditada y `Stderr` en el error
acreditado.

El entorno no es nulo y contiene exactamente dos entradas en orden canónico,
sin `os.Environ`:

```text
LC_ALL=C
PATH=/usr/local/go/bin:/usr/bin:/bin
```

No hay tercera entrada ni excepción por imagen. Quedan prohibidos `BASH_ENV`,
`ENV`, `SHELLOPTS`, `BASH_FUNC_*`, `LD_*`, HOME, GOPATH, GOMODCACHE y
cualquier ruta del repositorio.

`SysProcAttr` exacto:

```text
Setpgid:   true
Pgid:      0
Pdeathsig: SIGKILL
PidFD:     &pidfdPrimario
```

No se habilitan `Setsid`, `Ptrace`, credenciales, capacidades, namespaces,
cgroup, chroot, terminal de control ni flags adicionales.

`ExtraFiles` contiene exactamente siete entradas no nulas:

```text
[null3, null4, null5, null6, raiz7, runner8, ticket9]
```

El Bash recibe:

| FD hijo | Contenido |
| ---: | --- |
| 0 | `/dev/null`, lectura |
| 1 | salida privada del caso |
| 2 | error privado del caso |
| 3..6 | cuatro `/dev/null` privados |
| 7 | raíz acreditada |
| 8 | copia privada del runner 0600 |
| 9 | lector del pipe-ticket vacío |
| >=10 | ningún descriptor del supervisor heredado |

CONTROL, TERMINAL, SOBRE inicial ya consumido, escritor del ticket, ambos
pidfd y auxiliares del supervisor nunca llegan al Bash. La sonda distingue los
FD heredados de los que el propio Bash abre después, incluido su FD interno
255; no promete una tabla absoluta vacía durante la ejecución del intérprete.

## Barrera inmediatamente anterior a Start

La barrera se ejecuta en el mismo hilo, con el testigo posterior ya consumido,
y no asigna memoria sensible. Lee CONTROL en modo no bloqueante mediante un
buffer fijo de 1024 bytes, como máximo cuatro lecturas, 4096 bytes totales y
ocho reintentos por `EINTR`, hasta `EAGAIN`, terminal o EOF. Entrega
literalmente cada fragmento a G4; no implementa un segundo parser ni
reconstruye tramas.

Precedencia en esta frontera:

1. una trama completa, causa terminal o error de framing acreditado por G4 en
   el orden del stream;
2. señal encolada, `PDEATHSIG`, cambio de PPID o TID;
3. `finBootstrapGoM38` vencido o menos de cuatro segundos restantes: resultado
   `retirado`, cero `Start`;
4. lector parcial, presupuesto agotado sin `EAGAIN` o `EINTR` persistente:
   resultado `aplazado`, con custodia viva y cero `Start` si aún queda ventana;
5. inventario FD final completo y una relectura no bloqueante de
   CONTROL/señal/PPID/TID; cualquier novedad vuelve a aplicar los pasos 1--4;
6. una nueva lectura monotónica, posterior al inventario y a las relecturas,
   exige todavía cuatro segundos completos;
7. solo `EAGAIN` con G4 en S3, lector CONTROL L0 limpio y todos los FD ajenos
   `CLOEXEC` permite llamar inmediatamente a `Start`.

Un fragmento parcial, incluido un prefijo de `CANCELAR`, aplaza y no inicia.
`INICIAR\nCANCELAR\n` se consume antes de `Start`. EOF se entrega a G4 y su
resultado terminal conserva la causa O2b. `EINTR` obliga a observar señal y
reintentar de forma acotada; cualquier otro error retira cerrado. El
inventario y la relectura pertenecen a la barrera; después de su verde no se
abre, duplica ni consulta otro recurso.

Entre el final verde de esta barrera y `Start` no se permite ninguna otra
operación falible, lenta o de E/S.

## Start, segunda barrera y entrega

`pidfdPrimario` empieza en `-1`. Existe exactamente una llamada productiva a
`cmd.Start()` y ningún `Run`, `Output` o `CombinedOutput`. La duplicación
interna del handle de `Process` ocurre dentro de esa llamada; no es un syscall
del grafo O3a posterior al retorno.

La única tupla de fallo que acredita ausencia de hijo es
`errStart != nil && Process == nil && pidfdPrimario == -1`: no existe `Wait` ni
señalización, se cierran una vez los transitorios y se devuelve
`retirado/sin_hijo`.

Toda otra tupla se considera potencialmente post-`Start`:

- si falta `Process`, su PID no es positivo o `pidfdPrimario < 0`, O3a no puede
  acreditar el `Wait` y entra inmediatamente en `AF`;
- con `Process` y pidfd fiables, el primer syscall del llamador posterior al
  retorno, aunque `errStart != nil`, es exactamente
  `fcntl(pidfdPrimario, F_DUPFD_CLOEXEC, 10)`;
- la siguiente lectura monotónica fija una vez
  `finRetirada = min(ahora+3 s, finBootstrapGoM38-1 s)`; no se reinicia ni si
  la duplicación falló;
- si `errStart != nil`, ese error queda como primera observación y se ejecuta
  retirada; nunca se intenta entregar;
- si `errStart == nil`, `pidfdReserva` debe ser distinto, no negativo y
  `FD_CLOEXEC`; el inventario exige además una única tercera referencia de la
  misma identidad, `CLOEXEC` y poseída opacamente por `Process`; cualquier
  otra cardinalidad entra en retirada/`AF`;
- solo tras duplicación verde se cierra inmediatamente en el padre el lector
  del ticket; las demás copias destinadas al hijo permanecen bajo `leaseGuardia`
  hasta su revalidación y cierre en la segunda barrera;
- se ejecuta la segunda barrera; cualquier evento activa retirada y el Bash
  sigue bloqueado porque el ticket continúa vacío;
- solo la segunda barrera verde entrega juntos controlador S3, `Cmd`,
  `Process` con handle opaco, ticket, CONTROL, TERMINAL, dos pidfd explícitos,
  lease y observador con baseline a O3b.

La primera observación nunca se sustituye y no se abre otro pidfd por PID.

La segunda barrera usa el mismo buffer y presupuestos, pero nunca aplaza con un
Bash vivo. Su precedencia total es:

1. CONTROL en orden de stream: trama, framing y EOF se entregan a G4; un
   resultado terminal retira;
2. fragmento parcial, presupuesto agotado, `EINTR` persistente u otro error de
   lectura: retirada;
3. señal/PDEATHSIG o PPID/TID incoherente: retirada;
4. bootstrap vencido o, como comprobación preliminar, menos de un segundo
   restante para O3b/O3c: retirada;
5. sondeo no recolector de ambos pidfd explícitos: solo `POLLIN` acredita
   terminalidad; `POLLNVAL`, `POLLERR`, `POLLHUP`, bit extra, cierre/alteración
   o identidad no coincidente retira; sin pidfd fiable, `AF`;
6. revalidación de exactamente tres referencias pidfd de la misma identidad y
   final de `leaseGuardia`, tabla, flags y metadatos sellados, incluidos
   los duplicados raíz/runner/salida/error y los cinco `/dev/null` aún poseídos;
7. cierre exacto en el padre de esos nueve FD destinados al hijo; cualquier
   error de cierre activa retirada y nunca se declara verde;
8. una lectura monotónica final, posterior a todo lo anterior, debe observar
   todavía un segundo completo;
9. solo `EAGAIN`, G4 S3/L0, ambos pidfd explícitos y el handle opaco presentes
   permiten handoff inmediato.

Una cancelación o EOF ya disponible gana a la terminalidad observada después.
Un primario inválido con reserva viva, o la situación inversa, retira usando la
referencia fiable; ambos inválidos entran en `AF`. O3a no envía señal cero ni
acredita identidad de grupo; esas garantías son O3b. Entre barrera verde y
handoff no hay operación falible o E/S. El agregado nunca expone por separado
PID, pidfd, `*exec.Cmd`, CONTROL, TERMINAL o escritor.

## Retirada anterior a la entrega

Después de `Start` y antes de O3b, O3a es el único propietario. `finRetirada`
se fija solo tras el intento inmediato de duplicación, está limitado por el
bootstrap y no se reinicia. La retirada:

1. congela la primera observación; los errores posteriores son secundarios;
2. cierra primero el escritor del ticket para provocar EOF antes de
   auto-`STOP`, comandos externos o efectos gracias a O3a-P1;
3. cierra las copias del padre destinadas al hijo y todo transitorio pendiente;
4. observa terminalidad mediante el primario o la reserva, sin recoger, dentro
   de `finRetirada`;
5. si EOF no basta, envía una sola vez
   `pidfd_send_signal(pidfdFiable, SIGKILL, 0)` al proceso individual;
6. acredita terminalidad y solo entonces ejecuta exactamente un `cmd.Wait`;
7. cierra ambos pidfd y recolecta únicamente hijos inesperados ya terminales
   con espera no bloqueante hasta `ECHILD`, siempre dentro de `finRetirada`;
   cualquier adoptado vivo o inventario no basal entra en `AF`, sin señal de
   grupo;
8. devuelve `retirado` con CONTROL, TERMINAL, lease y observador en custodia
   negativa; no resetea ni desregistra el observador.

La reserva puede sustituir al primario solo para terminar la retirada; no se
promueve como nueva identidad ni se abre `pidfd_open(PID)`. No se usa PID/PGID
numérico, `Process.Kill`, `Process.Signal` ni señal de grupo. Un `ExitError`
de `Wait` conserva el estado real del hijo y no equivale por sí solo a fallo de
recolección.

Si una tupla potencialmente post-`Start` no tiene pidfd fiable, vence
`finRetirada` sin terminalidad, no puede ejecutarse el único `Wait` o queda un
hijo/zombi/FD, la rama entra en `AF` y llama inmediatamente a
`fatalO3aM38`, sin cierre previo, log ni otra E/S. El cierre del proceso y
`Pdeathsig` reducen el daño, pero no se presentan como retirada completa. El
exterior acredita estado 65 y EOF; nunca se devuelve limpieza ficticia ni se
permite el siguiente caso.

## Fronteras posteriores

- O3b recibe `Process`, pidfd explícitos, lease y observador, relee el contador
  monótono antes de efectos, ejecuta última barrera, revalida el bootstrap y
  construye/escribe/cierra una sola vez
  `PID_SUPERVISOR|TICKET\n`, acredita auto-STOP e identidad `/proc` y transfiere
  el capturado; no duplica pidfd;
- O3c crea una sola marca monotónica de 180 segundos inmediatamente antes del
  primer `CONT`, envía `CONT` por pidfd de grupo antes de
  `finBootstrapGoM38` y no reinicia esa marca;
- O4a posee causa primaria y precedencias, conserva y evalúa el vencimiento de
  la marca creada por O3c, y recibe sembrada la primera observación prehandoff
  sin permitir que una limpieza la sustituya;
- O4b posee STOP/TERM/CONT/KILL funcional de grupo;
- O4c posee `waitid`, único `cmd.Wait` funcional que libera el handle opaco,
  drenaje, `ECHILD`, `ESRCH`, TERMINAL y liberación final de lease/observador;
- O5 posee preflight pidfd anterior al baseline, registros de acreditación,
  lease y observador de señal, aislamiento cooperativo, puente Shell, FD del
  supervisor, TERMINAL, validador y activación; O3a construye `ExtraFiles`;
- O6 posee matriz integradora y fronteras externas.

La segunda barrera de O3a evita entregar un proceso cuyo evento ya era
observable. Conserva la primera observación para O4a, pero no implementa el
latch funcional posterior.

## Pruebas conductuales mínimas

1. Preflight deja token en registro privado y cero hijo/FD; `preparar` lo elimina y construye sin `Start`; token nulo/fallido/forjado/clonado/ajeno/reusado o sin autoidentidad/TID/generación/CAS rechaza.
2. S3 íntegro, testigo posterior, CONTROL en `EAGAIN`, PPID/TID y baseline de señal estables producen un `Start`, tres referencias pidfd y agregado S3 con lease/observador; señal antes/después de cada última lectura o handoff retira y el contador nunca se resetea.
3. Los 39 selectores válidos producen los mismos cinco argumentos literales.
4. Una sonda acredita mapa hijo 0..9; padre 0/1/2 sellado puede no tener `CLOEXEC` pero se sustituye, y ningún FD supervisor >=10, CONTROL, TERMINAL, SOBRE, pidfd o ticket-writer se hereda; ausencia/cambio de 0/1/2 rechaza.
5. FD ajeno >=3 sin `CLOEXEC` rechaza; límite >máximo o escaneo que consume bootstrap falla cerrado. `leaseGuardia` excluye concurrencia ajena; syscall sin pending, consolidación anticipada, fallo no restaurado o pending al handoff rechaza. Alterar las nueve copias tras `Start` retira; cero quedan al handoff.
6. TERMINAL es regular 0600, enlace uno, escritor único, CLOEXEC y permanece abierto en agregado o custodia negativa.
7. Bash bloquea en FD 9 vacío sin STOP, comando, descendiente o efecto; cerrar escritor produce EOF/salida 64.
8. El conductor externo, con 0/1/2 sellados, huecos solo 3..9 y bundle alto, acredita tres altas exactas del mismo objeto y `CLOEXEC`; cuarta referencia, identidad/flag adverso o barrido desde 10 rechaza.
9. Error `Start` con `Process=nil`/pidfd `-1`: cero hijo/Wait/señal/FD; toda tupla adversa se retira o entra en `AF`, nunca «sin hijo».
10. Fallo `F_DUPFD_CLOEXEC`: retirada con primario, un `Wait` posterior a terminalidad y cero residuo.
11. Hijo terminal justo tras `Start`: la segunda barrera no entrega.
12. `CANCELAR`/EOF disponible gana a terminalidad simultánea y conserva la observación CONTROL.
13. Primario cerrado/reserva viva y viceversa retiran; ambos cerrados ejecutan `AF` bajo conductor externo.
14. EOF y KILL individual cubren retirada; `Wait` total uno, solo tras terminalidad.
15. Conductor durable externo acredita que fatal no retorna: 65/EOF, stdout/stderr vacíos y dominancia sin cierre/log/E/S previa.
16. Alias consumido tras entrega/retirada no repite `Start` ni toca recursos; no se repite tras `AF`.
17. Alias de testigo comparte CAS: primer uso posible, segundo falla; clon/autoidentidad rota/no registrado falla pre-`Start`.
18. Fragmentación/vuelta tardía/borde temporal dan cero `Start`; pausas prueban revalidación O3b y retirada hasta 3 s sin reinicio. O3c/O6 prueban `CONT` previo al límite.
19. Parcial, presupuesto, `EINTR` persistente o `poll` erróneo en barrera dos retira, nunca entrega/aplaza.
20. Conductor externo Linux prueba `Pdeathsig`: muerte del hilo creador mata Bash; otro hilo no; ambos sin proceso/grupo/FD. M18 muere conductualmente.
21. Cien iteraciones reproducen inventarios inicial/final de FD, hijos, zombis y temporales cero.

Negativos obligatorios: nulo, S1/S2/S5, causa/fallo previo; testigo del mismo
turno, literal, ajeno, clon de valor, reutilizado, agotado o de otro TID; `CANCELAR`,
parcial, EOF, framing, `EINTR` persistente, señal, PPID/TID distinto, subreaper
o `prctl` ausente; cada FD ausente, cerrado, alias, reordenado, de tipo/acceso,
modo, enlace, tamaño, identidad o flags adversos; TERMINAL heredado, cerrado o
aliased con salida/error; SOBRE sin EOF; entorno hostil; pipe-ticket no vacío;
`ExtraFiles` de seis u ocho; fallo de `Start`, pidfd `-1`, duplicación fallida,
pidfd terminal, cuarta referencia, handle de identidad/flag adverso, FD alto
heredable, preflight adverso, bootstrap vencido y retirada
cuyo deadline vence.

## Mutantes compilables mínimos

M01--M66 son familias obligatorias, no sesenta y seis transformaciones. La
proyección debe expandir cada alternativa separada por «o», cada tipo de FD y
cada rama agrupada a identificadores atómicos `M001..MN`. Cada ID atómico
modifica exactamente un patrón y tiene un solo oráculo causal:

- M01 ruta Bash distinta; M02 quitar `-p`; M03 reordenar Args; M04 usar `-c`;
- M05 heredar entorno; M06 admitir `BASH_ENV`; M07 usar CWD vivo;
- M08 reordenar `ExtraFiles`; M09 omitir filler; M10 intercambiar raíz/runner;
- M11 mover ticket de FD9; M12 intercambiar stdout/error;
- M13 heredar escritor ticket; M14 omitir `CLOEXEC`;
- M15 aceptar, para cada FD/`devnull` y columna, propietario, tipo, acceso,
  modo, enlace, tamaño, offset, flag, identidad, baseline o `st_rdev` adverso;
  M16 quitar `Setpgid`;
- M17 cambiar `Pgid`; M18 quitar `Pdeathsig`; M19 quitar `PidFD`;
- M20 añadir `Setsid` o una capacidad; M21 quitar barrera anterior;
- M22 quitar barrera posterior; M23 aceptar parcial; M24 ignorar `CANCELAR`;
- M25 ignorar EOF; M26 omitir/alias/resetear observador o ignorar TID/generación/signo/contador/cambio tardío; M27 ignorar PPID/TID;
- M28 permitir `Start` fuera de S3; M29 doble `Start`;
- M30 entregar pidfd negativo; M31 escribir ticket en O3a;
- M32 leer `/proc/PID`; M33 omitir cierre de ticket en retirada;
- M34 usar kill numérico; M35 omitir o duplicar `Wait` de retirada;
- M36 activar `--supervisar-m38`.

- M37 aceptar la misma vuelta de `INICIAR`; M38 aceptar testigo literal,
  ajeno, clon de valor, autoidentidad rota o reutilizado; M39 implementar un
  segundo parser o cambiar G4;
- M40 omitir/añadir reserva o handle opaco, cambiar su identidad/`CLOEXEC`; M41 duplicar después de cierre, log, barrera o
  E/S; M42 entregar una sola referencia o dos aliases sin duplicación real;
- M43 ignorar fallo de `F_DUPFD_CLOEXEC`; M44 omitir terminalidad o aceptar
  `POLLNVAL`/`POLLERR`/`POLLHUP`/bit extra como terminal; M45 dar terminalidad prioridad sobre `CANCELAR` ya disponible;
- M46 omitir TERMINAL FD 8; M47 heredarlo al Bash; M48 cerrarlo en retirada;
  M49 alias de TERMINAL con salida o error;
- M50 usar `os.Environ` o añadir entorno opcional; M51 usar `close_range` en el
  padre; M52, para cada clase, cerrarla antes de la revalidación viva, sustituir
  esta por metadatos cacheados u omitir su cierre posterior;
- M53 cerrar CONTROL o TERMINAL en entrega; M54 ejecutar cero o dos `Wait` en
  retirada; M55 ejecutar `Wait` antes de terminalidad;
- M56 sustituir la primera observación por un error de limpieza; M57 permitir
  retorno, cierre, log o cualquier E/S antes de `fatalO3aM38`; M58 usar
  `pidfd_open`, `os.NewFile` o `*os.File` sobre pidfd prestado;
- M59 usar señal de grupo en retirada o exponer por separado el agregado;
- M60 permitir `Start` con parcial, EOF, cancelación o FD supervisor >=10
  heredado;
- M61 omitir autoridad/registro/pertenencia/autoidentidad/TID/generación/CAS/orden del preflight o recrear `finBootstrapGoM38`; M62 aceptar en la última
  lectura menos de cuatro segundos pre-`Start`, menos de uno prehandoff u
  omitir la revalidación O3b;
- M63 omitir/acortar inventario, iniciar en 10, aceptar 0/1/2 ausente/cambiado, FD ajeno >=3 heredable o límite mayor; forjar/liberar lease/observador, omitir/reordenar comenzar-syscall-consolidar, aceptar permiso ajeno/reusado/resultado forjado, clasificar B como C, reservar en B, ejecutar syscall al consolidar Start, no restaurar fallo o entregar pending;
- M64 clasificar como «sin hijo» una tupla no canónica, omitir su duplicación o
  `finRetirada`, o entregarla pese a `errStart`;
- M65 alterar buffer 1024, cuatro lecturas, 4096 bytes, ocho `EINTR`, mínimo FD
  10, retirada 3 s o reserva O3b/O3c 1 s.
- M66 añadir arista/ciclo no permitido, segundo owner/constructor/escritura de autoridad/estado, `Start`/`Wait` productivo fuera de G6c, `Start` directo, cleanup G7b omitido/doble/reordenado/fuera, o `init`/global/hook/callback/mock en G7.

Para cada ID atómico la proyección fija ruta, patrón anterior y posterior
literales de cardinalidad uno, diff, líneas y SHA mutado, y oráculo causal. La
tabla de expansión demuestra que ninguna alternativa de M01--M66 quedó
implícita. Cada mutante debe pasar `gofmt`, `go vet` y build antes del oráculo;
un fallo de compilación no cuenta como muerto. La huella global por sí sola no
es oráculo y el modo mutante solo relaja la fuente declarada.

## Evidencia y puertas

La evidencia durable se confirma después del futuro commit material y queda
ligada a su SHA. Debe incluir:

- analizador AST/tipado portable con grafo de llamadas, literales, orden, propiedad y APIs prohibidas;
- mutador separado que aplique una vez cada patrón y distinga muerte AST de muerte conductual;
- dos builds Go 1.26.5 aislados y byte a byte iguales;
- autoprueba Linux real y conductor durable externo ligado al SHA material, 100 iteraciones y residuos cero;
- `bash -n`, ShellCheck, H0 PostgreSQL 18.4, calidad global, carrera, `govulncheck`, Gitleaks y `git diff --check`;
- modo `--supervisar-m38` y argumento desconocido todavía en 64.

El AST/tipado enumera definiciones, raíces, aristas y call-sites por fichero; demuestra una sola llamada productiva a `Start`, dos fases,
autoridad concreta G6a, única fixture G7a sin ruta operativa, registro/pertenencia/borrado/autoidentidad/TID/generación/CAS de preflight, lease con cardinalidad/orden exactos `comenzar→syscall→consolidar`, permiso opaco y tuplas Start A/B/C, observador y testigos, barreras anterior/posterior,
terminalidad en la segunda, duplicación pidfd inmediatamente posterior,
comando/mapa/entorno exactos, agregado opaco con `Process` y dos pidfd
explícitos más una única tercera identidad opaca, barrido `0..Cur-1`, delta de
tres y mapa vivo, propiedad lineal de TERMINAL, `leaseGuardia` transferida y dominancia directa de `fatalO3aM38` sin cierre/log/E/S previa. También acredita que solo el lector del ticket se cierra
tras la duplicación: las otras nueve copias siguen abiertas e idénticas hasta
`fstat`/`fcntl` vivos en la barrera dos, se cierran exactamente una vez después
y antes del reloj final/handoff; además, cero ticket, `/proc`, señal funcional
o `Wait` productivo fuera de retirada. La única excepción AST es G7b dominada por `--autoprueba`, con la secuencia test-only completa de la línea 143.

## APIs y conductas prohibidas

En el grafo productivo O3a quedan prohibidos:

- `exec.Command`, `exec.CommandContext`, `LookPath`, `Run`, `Output`,
  `CombinedOutput`, `StartProcess` y alternativas directas de fork/clone;
- `pidfd_open`, reapertura por PID, señales por PID/PGID —fuera del preflight
  cerrado sobre el propio supervisor—,
  `Process.Kill` y `Process.Signal`;
- `os.NewFile` o cualquier `*os.File` sobre un pidfd prestado; identidad, flags
  y estado se consultan solo mediante `fstat`/`fcntl`/`poll` raw sin propiedad;
- `dup`, `pipe`, `close_range` o cambios masivos de flags en la tabla FD del
  padre; solo `F_DUPFD_CLOEXEC` y `pipe2(O_CLOEXEC)`;
- entorno heredado o nulo, `-c`, ruta Bash distinta, CWD vivo y cualquier
  opción `SysProcAttr` fuera de la lista exacta;
- `/proc/<pid>`; la única cadena `/proc` admitida en O3a es el argumento
  literal `/proc/self/fd/8`;
- escribir/cerrar el ticket en la rama de éxito, `STOP`, `CONT`, señal cero o
  funcional, plazo de caso, escritura TERMINAL y `Wait` fuera de retirada;
- goroutines nuevas, callbacks de `Start`, funciones variables, mocks en el
  grafo productivo y transferencia separada de PID, pidfd, `Cmd`, CONTROL,
  TERMINAL o escritor del ticket;
- copia, conversión, log o exposición del ticket, nonce, identidad o sobre
  retenido.

## Secuencia de autorización

1. este borrador recibe revisión funcional y de seguridad completas;
2. se corrigen todos los P0/P1/P2 y se vuelve a revisar;
3. con doble GO semántico se autoriza solo una proyección externa, sin editar
   el repositorio;
4. la proyección fija líneas y SHA exactos posteriores, matriz, transformaciones
   mutantes y oráculos;
5. el documento revisado recibe doble GO final, cambia solo entonces de estado
   y se crea acta;
6. contrato y acta se publican y esperan CI 5/5;
7. el productor implementa en worktree exclusivo y confirma material autónomo;
8. evidencia portable se confirma por separado y dos revisores independientes
   reproducen material y evidencia;
9. solo tras doble GO se integra, publica y espera CI 5/5.

El productor no revisa ni integra su propio código.

El ledger posterior V13, el doble GO y la parada por R se fijan en el
[acta de revisión V13](revisiones/revision_f0_h0b_c4b2_g2o_o3a_v5_candidato_2026-08-09.md).
R sigue en 702 líneas y `7ad65a66…`; mientras rija «No uses R», no se declara
material autónomo, CI autorizante, integración ni producción.

## Paradas duras

Se detiene si cambia una octava ruta material; se toca D, G2, G3, G4, G5,
capturador o adaptador; se supera un presupuesto; el manifiesto no contiene
catorce entradas exactas; falta `-p`; aparece entorno o FD extra; se inicia en la
misma vuelta que `INICIAR`; una causa pendiente permite `Start`; se entrega sin
primario, reserva y handle opaco únicos o sin segunda barrera; la reserva no es
el primer syscall del llamador tras `Start`; se exige al padre un FD físico
3..9; falta preflight; O3a escribe ticket, consulta `/proc`, usa
`close_range` o envía señal funcional; TERMINAL se pierde, hereda, copia o
cierra; un fallo posterior a `Start` puede retornar sin retirada acreditada;
una rama fatal retorna; se sustituye la primera observación; se usa PID/PGID
numérico; `RLIMIT_NOFILE` no se inspecciona íntegramente, 0/1/2 no quedan sellados o cambian, o aparece un FD ajeno >=3 heredable; lease/observador no registrado, transacción pending inválida o tabla/flags no congelados; un FD incumple su dominio
físico cerrado; se recrea/reinicia el bootstrap o una última lectura acepta
menos de cuatro segundos pre-`Start`/uno prehandoff; cambian 1024/4/4096/8,
3 s/1 s o el mínimo FD 10; aparece `Wait` productivo fuera de retirada o Wait
test-only fuera de G7b/no único/no posterior a terminalidad; M18 carece de muerte
conductual; una familia mutante no se expande a cambios atómicos; el modo deja
64; sobrevive un mutante; una prueba deja hijo, zombi, FD o temporal; falta una
huella posterior exacta; o cualquier puerta/revisión termina no verde.

## Métricas

O3a no cierra todavía O3, C4b-2, C4b, H0b, C2, F0 u O4-05. Este borrador no cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` ni producción `NO-GO`. No autoriza datos reales.

# Decisión F0-H0b/C4b-2/G2-O/O4b: señales funcionales de grupo

Fecha: 12 de agosto de 2026.

Identificador: `O4B-P0-CONTRATO`.

Estado: **CANDIDATA DOCUMENTAL**. No autoriza código, integración, O4A-P4,
O4b, O4c, producción ni despliegue. Requiere doble revisión independiente,
publicación y CI 5/5 sobre el SHA exacto.

## Resultado observable

O4b es el único propietario de ejecutar las señales funcionales de grupo que
O4a autorice. Consume una autorización opaca, cardinal y one-shot; valida la
custodia prestada; intenta exactamente la secuencia de la etapa; consolida
cada syscall bajo un permiso lease distinto; y devuelve a O4a un resultado
opaco sellado. Las cuatro etapas cerradas son `PARADA_INICIAL`,
`TERMINAR_Y_REANUDAR`, `PARADA_FINAL` y `MATAR_GRUPO`.

O4b no elige causa, incidente, etapa ni plazo. Tampoco crea tiempo, permisos,
identidad o recursos. O4a conserva causa, precedencias, deadlines y la máquina
de extinción; O4c conserva terminalidad final, Wait, drenaje, TERMINAL y
liberación. Un resultado O4b nunca constituye recogida ni estado final.

## Base, huellas y prevalencia

La base exacta es `1946b671afbcc47c681d6d6bb22e3c1122935247`, cierre de
`O4A-P3-ARBITRAJE v2` en `trabajo/o4a-p3-arbitraje-v2-20260811`. La CI
[31543335809](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31543335809)
terminó `Success`, intento 1, con cinco de cinco puertas verdes.

Decisiones autoritativas congeladas:

| Decisión | SHA-256 | Responsabilidad que prevalece |
| --- | --- | --- |
| [O3a](decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md) | `39514c827486f385db89e2117ab4e8a2f43e0be7ade98158a1ab0c7a49685a90` | mapa FD, lease, observador, TID, identidad y retirada pre-CONT |
| [O3b](decision_f0_h0b_c4b2_g2o_o3b_ticket_stop_identidad_2026-08-10.md) | `d9aa33eddf90da2fb0e7f1aac239a18797e70b8afe7f9fe3024f1a9e5f401ada` | ticket, parser `/proc/<PID>/stat`, auto-STOP e identidad sellada |
| [O3c](decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md) | `f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6` | CONT inicial, marca de 180 s, primera observación y custodia C5 |
| [O4a](decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md) | `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc` | causa, precedencia, etapas, subplazos y autorizaciones one-shot |

La especificación conjunta G2-O permanece genealogía `NO-GO`. Ante una
aparente contradicción prevalece el propietario de la fila anterior: O3a para
capacidad física, O3b para la lectura probatoria ya publicada, O3c para el
handoff post-CONT, O4a para toda decisión y este documento únicamente para el
efecto señalado. O4b no recupera ACK, ticket, selector ni autoridad temporal.

## Alcance cerrado

O4b posee solamente:

- consumo lineal de una autorización O4a emitida en A4;
- validación privada de autorización, custodia e identidad prestadas;
- una señal de grupo STOP, TERM, CONT o KILL por cardinal autorizado;
- evidencia mínima no recolectora posterior al intento;
- resultado opaco A5 con retornos raw y cardinalidad exacta.

O4b no posee:

- causa primaria, incidente de cierre, selección de etapa o transición O4a;
- creación, extensión, pausa, reinicio o interpretación de deadlines;
- `Wait`, `waitid`, `wait4`, recogida, drenaje o código de salida;
- escritura/cierre de TERMINAL, cierre de CONTROL/pidfd o liberación;
- señal por PID/PGID, fallback degradado o recurso alternativo;
- parser, `/proc` o primitiva de identidad nuevos;
- API pública, getter, ticket, nonce, log, serialización o dato humano;
- O4c, O5, O6, HTTP, SQL, red, Docker, producción o despliegue.

## Entrada opaca, propiedad y selección de pidfd

La entrada futura es `**autorizacionEtapaO4aM38` junto a la autoridad O4a
privada que conserva la custodia. El puntero del llamador se pone a nil antes
de leer la autorización. No se acepta una lista de argumentos ni se
reconstruye una autorización.

La autorización contiene de forma privada y sellada: autoidentidad,
generación, TID, etapa, cardinalidad, operación, límite absoluto, rol pidfd y
estado `EMITIDO`. No contiene un FD exportable. O4a fija siempre el rol
`PRIMARIO`: es la referencia explícita usada por el CONT O3c, acreditada junto
con reserva y handle opaco. La reserva y el handle son testigos de identidad;
nunca se promueven, señalizan ni sirven de fallback. Una pérdida del primario
es AF, aunque la reserva siga abierta.

Antes de todo syscall O4b acredita conjuntamente:

1. autorización no nula, autoidéntica, estado `EMITIDO` y perteneciente al
   único slot O4a pendiente;
2. autoridad O4a autoidéntica en A4, causa válida e inmutable, etapa, operación,
   cardinal y deadline iguales al slot;
3. owners subyacentes `ownerLease=O4A` y `ownerObservador=O4A`, lease 3,
   observador 2, `pending` nil, registro, generaciones y TID exactos;
4. `syscall.Gettid()` igual al TID sellado, ejecutado bajo su propio permiso
   lease y consolidado antes de comparar;
5. `Cmd`, `Process`, handle opaco, pidfd primario/reserva, CONTROL y TERMINAL
   iguales a los sellos O4a; ticket ausente;
6. primario, reserva y handle como exactamente tres referencias de la misma
   identidad física; primario/reserva distintos, abiertos y `CLOEXEC`;
7. identidad `/proc` ya sellada, registro, generación, PPID, SID, PGID y
   `starttime` intactos, sin exponer sus números;
8. límite monotónico no cero y perteneciente exactamente a la etapa/rama.

La validación física usa únicamente `identidadPidfdBarreraO3bM38` sobre las
referencias ya prestadas; cada `fstat`, `F_GETFD` y `F_GETFL` interno conserva
su permiso lease separado. No abre, duplica ni envuelve pidfd. Una divergencia
de identidad, flags, owner, TID, registro, generación, recurso o cardinal es
AF, no resultado raw.

Nulo retorna `uso_consumido` sin efecto. Alias, clon, forja, replay, reuso o
carrera permiten exactamente un ganador del CAS de autorización
`EMITIDO→CONSUMIENDO`; los demás observan `CONSUMIENDO/CONSUMIDO` y retornan
`uso_consumido` sin syscall ni mutación de recursos. Un estado imposible o
autoidentidad rota es AF. No hay rollback a `EMITIDO`.

## Máquina privada total

Estados no exportables:

```text
OB0 recibida
  -> OB1 validada | OBF fatal
OB1 validada
  -> OB2 consumiendo | OBF fatal
OB2 consumiendo
  -> OB3 permiso_preparado | OBF fatal
OB3 permiso_preparado
  -> OB4 syscall_intentado | OBF fatal
OB4 syscall_intentado
  -> OB5 consolidado | OBF fatal
OB5 consolidado
  -> OB3 permiso_preparado | OB6 evidencia | OBF fatal
OB6 evidencia
  -> OB7 resultado_sellado | OBF fatal
OB7 resultado_sellado
  -> OB8 consumido | OBF fatal
```

`OB5→OB3` existe solo para el segundo syscall CONT de la etapa compuesta y
solo si TERM consolidó con raw cero. No hay transición desde OB8 u OBF. OB4
significa intento aunque el raw sea `EINTR` u otro error. Un raw solo es
interpretable en OB5. El CAS `EMITIDO→CONSUMIENDO` y la transición final
`CONSUMIENDO→CONSUMIDO` ocurren una vez; no hay `Store`, `Swap`, segundo CAS,
rollback ni resultado parcial.

## Disciplina lease por syscall

Cada syscall, incluidos los de validación y evidencia, usa un permiso opaco
nuevo mediante:

```text
lease.comenzar -> syscall único -> lease.consolidarCritico
```

Se consolida antes de leer o clasificar el retorno. Nunca se usa
`comenzarCritico`, se comparte o reutiliza permiso, se agrupan dos syscalls en
una closure ni queda `pending`. La consolidación no ejecuta otro syscall.

Fallo de `comenzar`, duda o fallo de `consolidarCritico` es OBF directo. El
raw del syscall ya intentado queda como evidencia no interpretable dentro de
la autoridad fatal; no se devuelve, no fija incidente y no autoriza syscall
posterior. Un raw `EINTR` correctamente consolidado sí cuenta como intento y
se devuelve sin retry.

## Única primitiva de señalización

Todo efecto funcional es exactamente:

```text
pidfd_send_signal(pidfdPrimario, SIGNAL, NULL,
                  PIDFD_SIGNAL_PROCESS_GROUP)
```

El flag es literalmente `1<<2`. `SIGNAL` es la constante cerrada de la etapa:
STOP, TERM, CONT o KILL. No existe señal cero funcional ni de evidencia en
O4b. Quedan prohibidos `kill`, PID/PGID numérico, `Process.Signal`,
`Process.Kill`, `pidfd_open`, `dup*`, `F_DUPFD*`, `os.NewFile` y fallback.

Cada llamada ocurre una vez. `EINTR`, `ESRCH`, `EPERM`, `EINVAL`, `ENOSYS`,
`EBADF` o cualquier raw no cero se conservan literalmente; no se traducen a
éxito ni se reintentan. O4a interpreta el resultado consolidado.

## Deadlines prestados

O4a crea y sella cada límite antes de emitir el permiso. O4b no recibe una
duración y no calcula otro límite. La autorización porta una copia privada del
`time.Time` monotónico absoluto y su clase; O4b coteja igualdad estructural Go
`==` contra el slot O4a.

Después de validar propiedad e identidad, pero antes de preparar el permiso
del efecto, O4b ejecuta una única lectura monotónica de comprobación. No es una
decisión temporal transferida: solo prueba `ahora.Before(limite)`. Igualdad
vence. Marca cero, componente civil reconstruido, `ahora` anterior a la
genealogía sellada, divergencia o límite vencido son OBF y no ejecutan señal.
No usa `Add`, `Sub` para recrear autoridad, `Round`, timer, `Sleep`, ticker ni
contexto con deadline.

Para `MATAR_GRUPO`, KILL es el siguiente syscall después de esa comprobación;
no se intercala identidad, evidencia, reloj, log ni asignación falible. Para
las otras etapas, la misma comprobación precede inmediatamente al permiso de
su primer efecto. TERM y CONT comparten el límite `finGracia`, pero cada
syscall tiene permiso propio. Tras consolidar TERM con raw cero, O4b ejecuta
una segunda y última lectura monotónica de comprobación, sin otro syscall ni
efecto: exige de nuevo `ahora.Before(finGracia)`. Si vence, entra en OBF y no
autoriza CONT ni devuelve un resultado parcial. Si sigue verde, prepara un
permiso nuevo y CONT es el siguiente syscall literal. No se recrea ni extiende
el límite.

## Tabla total de etapas y resultados

| Etapa | Límite prestado | Efecto exacto | Evidencia posterior mínima | Resultado cerrado |
| --- | --- | --- | --- | --- |
| `PARADA_INICIAL` | `finParadaInicial` | un STOP | dos muestras T estables antes del límite o no-estabilidad cerrada | cardinal 1, raw STOP, `ESTABLE`/`NO_ESTABLE` y último reloj observado |
| `TERMINAR_Y_REANUDAR` | `finGracia` | TERM; solo si raw TERM=0 consolidado, CONT inmediatamente | terminalidad o grupo presente no recolectores después del último efecto intentado | cardinal 1 si TERM falla; 2 si TERM=0 y CONT se intenta; raws separados |
| `PARADA_FINAL` | `finParadaFinal` | un STOP | dos muestras T estables o no-estabilidad cerrada antes del límite | cardinal 1, raw STOP y `ESTABLE`/`NO_ESTABLE` |
| `MATAR_GRUPO` | `finDrenajeRapido` o `finDrenajeCooperativo` | un KILL como siguiente syscall | ninguna sonda posterior en O4b | cardinal 1 y raw KILL |

Reglas exhaustivas:

1. STOP raw no cero, incluido EINTR, termina la etapa sin observación de
   estabilidad; se devuelve intento consolidado y O4a decide la rama.
2. STOP raw cero habilita únicamente la evidencia no recolectora. Estable T
   antes del borde se devuelve `ESTABLE`. Cualquier forma controlable distinta
   —incluida terminalidad del líder o líder vivo sin dos T idénticos— se
   devuelve `NO_ESTABLE`, que O4a consume por su única fila STOP no estable y
   converge a KILL. Corrupción/lease incierta es OBF. O4b no devuelve
   `TERMINAL` desde una etapa STOP porque O4a no define esa arista.
3. TERM raw no cero omite CONT y toda evidencia posterior. Cardinal=1.
4. TERM raw cero consolidado obliga a CONT como siguiente efecto. CONT raw,
   incluso EINTR/error, termina la parte funcional. La única operación
   intermedia es el cotejo monotónico descrito; su fallo es OBF. Cardinal=2.
5. Tras CONT consolidado, una observación no recolectora devuelve
   `TERMINAL` o `GRUPO_PRESENTE`. Una duda física, identidad/flags discordantes
   o retorno no clasificable es OBF, porque O4a no tiene una arista A5 para un
   resultado infraestructura de esta etapa. Nunca se fabrica una tercera
   señal ni un resultado que O4a deba inferir.
6. KILL no observa después, no reintenta y siempre termina OB6 con el raw
   consolidado. O4c decidirá terminalidad.
7. Un límite alcanzado durante evidencia de STOP produce `NO_ESTABLE`, no un
   efecto tardío. Un límite vencido antes del efecto es OBF.
8. Ningún resultado cambia causa, incidente, etapa o deadline. O4a aplica su
   tabla A5 y prepara, o no, la autorización siguiente.

## Evidencia no recolectora exacta

O4b no inventa un observador. Reutiliza solo primitivas O3 publicadas:

- `identidadPidfdBarreraO3bM38` acredita las referencias y flags;
- `pidfdVivoBarreraO3bM38` sondea terminalidad del líder con poll timeout 0;
- `leerStatStopO3bM38` abre/lee/cierra exclusivamente
  `/proc/<PID>/stat` bajo permisos separados;
- `parsearStatO3bM38` interpreta ese buffer acotado sin parser nuevo.

Cada invocación conserva sus permisos internos por syscall y debe terminar
sin `pending` antes de interpretar. O4b no llama `muestraTStopO3bM38`, porque
la evidencia contractual exige la identidad completa ya sellada.

`ESTABLE` requiere dos muestras consecutivas, separadas solo por
`runtime.Gosched`, con estado T y PID/PPID/PGID/SID/starttime exactamente
iguales al sello O3b y entre sí. Cada muestra va precedida por identidad pidfd
y seguida por el cotejo del límite prestado; no se duerme. Una sola T no basta.
La etapa ejecuta exactamente una pareja de muestras: no gira hasta el límite,
no crea un bucle de espera y no repite una lectura adversa. Si la pareja no es
estable y acreditable, devuelve `NO_ESTABLE` de inmediato.

`TERMINAL` significa exclusivamente, y solo después de TERM→CONT, que primario
y reserva acreditados presentan readiness natural exacta mediante dos sondeos
separados. No es Wait ni prueba de estado final. En una etapa STOP la misma
observación se normaliza a `NO_ESTABLE` para ajustarse a la tabla total O4a.
Divergencia entre referencias es OBF.

`GRUPO_PRESENTE` es evidencia mínima: ambos pidfd acreditados siguen no
terminales y una muestra `/proc` conserva identidad completa y estado no Z/X.
No afirma número de miembros ni estabilidad. O4b jamás usa señal cero para
probar presencia. Ausencia, identidad/flags adversos, parser adverso o
referencias discordantes producen OBF, no presencia.

La lectura `/proc` es reutilización de autoridad O3b, no una ruta ni parser
nuevos. El PID/PGID interno permanece en la custodia y nunca entra en permiso,
resultado, error, log o serialización.

## Resultado opaco hacia O4a

El resultado privado contiene solo:

- autoidentidad, generación y vínculo al slot consumido;
- etapa y límite exactos;
- cardinalidad 1 o 2;
- `rawPrimero` y, solo si cardinal=2, `rawSegundo`, separados;
- clase cerrada `SIN_EVIDENCIA`, `ESTABLE`, `TERMINAL`,
  `GRUPO_PRESENTE` o `NO_ESTABLE`;
- como máximo una marca monotónica observada copiada por valor, solo para
  acreditar el borde de evidencia.

No contiene causa, incidente, próxima etapa, PID, PGID, pidfd, `Cmd`,
`Process`, lease, observador, ticket, nonce, actor, tenant, texto libre, error
libre, PII ni getter. No es serializable. Los raws son los retornos ya
consolidados, no errno reinterpretados. El resultado se preasigna antes del
primer efecto y se sella una vez en OB7.

O4a acepta exactamente el resultado de su slot, consume el vínculo una vez y
aplica su autómata. Resultado nulo, clonado, forjado, de otra etapa, con
cardinal/limite/raw/clase incompatibles o replay es AF.

## Fatalidad y fronteras

Propiedad, identidad, permiso, consolidación o deadline no acreditables llaman
la fatalidad heredada: estado 65, EOF/no retorno, stdout=0 y stderr=0. Desde
OBF no hay señal posterior, limpieza, cierre, log, rollback ni E/S.

O4b nunca:

- llama `Wait`, `waitid`, `wait4`, recoge o drena;
- escribe/cierra TERMINAL, CONTROL, pidfd o streams;
- libera lease/observador, altera owners o recrea custodia;
- fija causa/incidente, crea etapa/subplazo o autoriza otro efecto;
- añade global mutable, `init`, hook, callback, goroutine o canal;
- registra, imprime o serializa autoridad;
- usa red, HTTP, SQL, Docker, Orquesta, Firecracker, Jailer o producción.

## DAG de implementación O4b

Cada nodo depende de doble GO, publicación y CI 5/5 del anterior:

| ID | Responsabilidad única | Write-set máximo | Cierre |
| --- | --- | --- | --- |
| `O4B-P1-AUTORIDAD` | Tipo privado OB0--OBF, consumo y validación; cero syscall. | Un Go productivo nuevo + prueba focal. | Anti-nulo/alias/clon/forja/replay/carrera y owners exactos. |
| `O4B-P2-EFECTO` | Primitiva única pidfd de grupo, permiso lease y raw; una etapa simple sintética. | Un Go productivo nuevo + prueba; P1 inmóvil. | Un syscall por permiso, flag/señal/raw exactos, cero retry. |
| `O4B-P3-STOP` | PARADA_INICIAL/FINAL y evidencia T estable reutilizada. | Un Go productivo nuevo + prueba; previos inmóviles. | STOP único, dos muestras exactas, bordes y no-estabilidad. |
| `O4B-P4-TERM-CONT` | Etapa compuesta TERM→CONT. | Un Go productivo nuevo + prueba; previos inmóviles. | Permisos separados, omisión/orden/cardinal/raw cerrados. |
| `O4B-P5-KILL-RESULTADO` | KILL inmediato y resultado opaco total. | Un Go productivo nuevo + prueba; previos inmóviles. | Límite estricto, cero sonda posterior, resultado sellado. |
| `O4B-P6-CONDUCTOR` | Black-box, AST/tipos/DAG y catálogo mutante. | Solo herramientas/pruebas propias. | Normal/race, mutantes muertos, residuos cero. |
| `O4B-P7-EVIDENCIA` | Ledger, checkpoint, revisiones y CI. | Solo evidencia/documentos propios. | Doble GO, hashes, push normal y CI 5/5. |

Aristas internas exactas:

```text
O4B-P0 -> P1 -> P2 -> P3 -> P4 -> P5 -> P6 -> P7
```

Aristas entre fronteras, sin ciclo:

```text
O4A-P3 -> O4B-P0 -> O4A-P4
O4A-P4 -> O4B-P1
O4A-P0 + O4B-P0 -> O4C-P0
O4A-P4 + O4B-P5 + O4C-P0 -> O4A-P5
```

O4A-P4 material permanece bloqueado hasta cerrar este P0. O4b necesita los
tipos opacos materializados por O4A-P4 antes de P1. O4C-P0 necesita ambos
contratos O4a/O4b publicados, no O4A-P4 material. O4A-P5 necesita el contrato O4c y el resultado O4b
material. Ningún nodo edita fuentes de otra frontera ni crea arista O4→O3.

Cada implementación añade un productivo y una focal; objetivo 300--500 líneas,
parada 650 y tope duro 800 por fichero. Superar 650, necesitar un getter o
compartir write-set exige división/decisión separada.

## Matriz mínima de pruebas

| ID | Oráculo causal |
| --- | --- |
| B01 | Autorización nominal se consume una vez; nulo/alias/clon/forja/replay/carrera dejan un ganador. |
| B02 | Owners O4A, lease3/observador2, registro/generación/TID/pending y recursos exactos. |
| B03 | Primario exclusivo, reserva/handle testigos; identidad/flags/custodia adversa AF. |
| B04 | STOP inicial/final: una llamada, raw incluido EINTR, sin retry. |
| B05 | Dos T completas estables acreditan ESTABLE; una T, cambio o borde produce NO_ESTABLE. |
| B06 | TERM raw no cero omite CONT; TERM cero hace la segunda lectura de `finGracia`; vencimiento post-TERM produce OBF sin CONT/resultado y verde hace CONT siguiente con permiso distinto. |
| B07 | CONT raw error/EINTR se conserva; no hay tercer efecto ni retry. |
| B08 | KILL es siguiente syscall tras comprobar límite; igualdad/vencido AF; cero sonda posterior. |
| B09 | Terminalidad, presencia e infraestructura se distinguen sin Wait ni señal cero. |
| B10 | Cada fstat/fcntl/poll/Gettid/open/read/close/señal tiene permiso distinto y consolidación previa. |
| B11 | Fallo comenzar/consolidar en cada posición es AF; raw consolidado sí retorna. |
| B12 | Resultado etapa/cardinal/limite/raw/clase exactos; forja o replay AF. |
| B13 | O4b no cambia causa/incidente/etapa/deadline ni owners. |
| B14 | BF directo acredita 65, EOF/no retorno, stdout/stderr cero y cero E/S posterior. |
| B15 | 100 normal + 100 race dejan delta cero FD/hijo/zombi/grupo/temporal. |
| B16 | AST/tipos/DAG prueba una única familia de señal, cero API prohibida y fronteras acíclicas. |

Fixtures usan procesos sintéticos en PGID aislado y limpieza exterior segura;
nunca procesos ajenos. Se prueban simultaneidades: terminal durante STOP,
TERM/CONT, presencia al borde y KILL con límite exacto. No hay SKIP ni retry
del caso fallido.

## Familias mutantes mínimas

Mutantes atómicos compilables y muerte causal:

- B01 aceptar nulo/clon/forja/replay o dos ganadores;
- B02 omitir autoidentidad, owner, registro, generación, TID o pending;
- B03 promover reserva, omitir una referencia, aceptar flags/identidad adversa;
- B04 usar PID/PGID, kill, Process.Signal/Kill, flag cero u otra señal;
- B05 omitir/reusar/compartir permiso o interpretar antes de consolidar;
- B06 convertir fallo lease en raw/resultado o ejecutar syscall posterior;
- B07 reintentar EINTR/error o tratarlo como no intento;
- B08 invertir TERM/CONT, ejecutar CONT tras TERM error, omitir/aceptar vencida la segunda lectura de `finGracia` o compartir permiso;
- B09 cambiar cardinal 1/2, mezclar raws o inventar segundo raw;
- B10 aceptar una T, identidad parcial, muestras distintas o parser nuevo;
- B11 usar señal cero como evidencia, Wait/waitid o terminalidad recolectora;
- B12 confundir presencia/terminal/infraestructura o afirmar miembros de grupo;
- B13 recrear/redondear/extender límite, aceptar igualdad o intercalar antes de KILL;
- B14 emitir dos STOP/KILL, sondear tras KILL o usar fallback;
- B15 añadir PID/pidfd/ticket/nonce/texto/getter/serialización al resultado;
- B16 fijar causa/incidente/etapa/plazo u ownership desde O4b;
- B17 cerrar/escribir/liberar/drenar/Wait o limpiar antes de fatal;
- B18 añadir parser/ruta `/proc`, pidfd_open/dup, global/init/hook/goroutine/canal;
- B19 falsear residuos, aceptar SKIP o evidencia de otro caso;
- B20 añadir arista inversa, compartir fichero o abrir O4c/O5/O6.

Cada mutante pasa gofmt, compilación y vet antes del oráculo. Timeout, SHA
global o no compilación no cuentan como muerte. AST/tipos/DAG cuenta permisos,
syscalls, flags, señales, CAS, estados, raws, parser reutilizado, deadlines,
prohibiciones y aristas.

## Puertas y evidencia

Cada corte futuro exige focal normal/race/repetida, black-box seguro, gofmt,
vet, AST/tipos/DAG, todos los mutantes, `go test ./...`,
`go test -race ./...`, `go vet ./...`, calidad, Gitleaks, hashes, enlaces,
`git diff --check`, doble revisión y CI 5/5. El conductor durable liga base,
toolchain, fuentes, catálogo y resultados, y acredita inventarios/residuos cero.

Este P0 modifica solo cuatro documentos nuevos propios. Código, pruebas,
runner, workflows, SQL, O3, O4a y documentos transversales permanecen
byte-inmutables.

## Seguridad, datos y métricas

Solo se tratan capacidades técnicas opacas. No aparece identidad humana,
tenant, RRHH, PII, ticket o secreto. La indisponibilidad nunca equivale a
autorización o éxito. No hay interfaz visible, i18n o accesibilidad nueva.

Contratación temporal permanece `24/46`, Bolsa productiva `1/14` y producción
`NO-GO`. Este contrato no integra, fusiona master ni modifica porcentajes.

## Condición de cierre y siguiente paso

GO documental requiere dos revisiones completas `P0=P1=P2=0` sobre los mismos
bytes, cuatro documentos coherentes, enlaces/hashes/secretos/diff-check verdes,
commit pequeño, push normal y CI 5/5. Cualquier discrepancia, autoridad
inventada, efecto adicional, parser nuevo, resultado ambiguo, ciclo DAG o
puerta no verde es NO-GO.

Tras el cierre, la siguiente dependencia exacta es `O4A-P4-ETAPAS`. No queda
autoasignada. Toda implementación O4b, `O4C-P0` material, `O4A-P5`, O5 y O6
siguen bloqueados según el DAG.

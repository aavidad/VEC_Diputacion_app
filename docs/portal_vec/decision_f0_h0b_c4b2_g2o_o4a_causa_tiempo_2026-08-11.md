# Decisión F0-H0b/C4b-2/G2-O/O4a: causa primaria y tiempo

Fecha: 11 de agosto de 2026.

Identificador: `O4A-P0-CONTRATO`.

Estado: **CANDIDATA DOCUMENTAL**. No autoriza código, integración, O4b, O4c,
producción ni despliegue. Requiere doble revisión independiente, publicación y
CI 5/5 antes de la primera minitarea de implementación.

## Resultado observable

O4a consume exactamente una vez el agregado opaco C5 entregado por O3c,
preserva literalmente la primera observación raw y el retorno raw del único
`CONT`, y conserva conjuntamente toda la custodia. Es la única autoridad que
ordena acontecimientos, evalúa relojes monotónicos y enclava una sola causa
primaria. A partir de esa causa emite únicamente autorizaciones opacas,
cardinales y consumibles una vez para las etapas futuras de O4b.

O4a no envía señales, no recoge procesos y no limpia. O4b será el único dueño
de ejecutar STOP/TERM/CONT/KILL funcional de grupo conforme a una autorización
O4a. O4c será el único dueño de observar terminalidad para cierre, ejecutar el
único `cmd.Wait` funcional, drenar adoptados, acreditar `ECHILD`/`ESRCH`,
escribir/cerrar TERMINAL y liberar recursos.

## Base exacta, prevalencia y verificación

La base es `0029cfe03b2f2f637169be8340985e38b1fa6557`, publicada en
`trabajo/o3c-p7-evidencia-20260811`. Su padre es
`981ed3d413a3bc0c061c74dd17939dbd55eaf759`. La CI final
[31509858146](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31509858146)
alcanzó `Success` con los cinco jobs concluidos sobre esa rama y SHA exactos.

La decisión O3c tiene SHA-256
`f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6`;
su ledger final tiene SHA-256
`a1de39d1c80492cae8b6b858a096d8f3051b346913064fe51a83dd6573dbb3b1`.
Prevalecen, por responsabilidad:

1. la decisión O3a para `Cmd`, `Process`, mapa FD, lease, observador y TID;
2. la decisión O3b para ticket cerrado, auto-STOP e identidad `/proc`;
3. la decisión O3c para el agregado C5, marca de 180 s, CONT y primera
   observación;
4. esta decisión exclusivamente para causa, precedencias, tiempo y permisos
   de etapa O4a.

La especificación conjunta G2-O del 5 de agosto permanece `NO-GO` genealógico.
Solo se conservan sus causas canónicas y la secuencia temporal donde no fue
sustituida por O3a/O3b/O3c o por esta separación O4a/O4b/O4c.

## Alcance cerrado

O4a posee exclusivamente:

- consumo lineal del agregado O4a C5;
- custodia conjunta y autoridad lógica hasta la entrega terminal a O4c;
- preservación raw de la primera observación y del retorno CONT;
- lectura no recolectora de CONTROL, observador, pidfd y reloj;
- causa primaria, precedencias y simultaneidad;
- `finCaso` heredado de 180 s y subplazos monotónicos de etapa;
- autorizaciones privadas one-shot para O4b y orden de convergencia a O4c;
- fallo cerrado cuando la propiedad, el tiempo o la causalidad no son
  acreditables.

O4a no posee:

- STOP, TERM, CONT o KILL, ni siquiera señal cero;
- `Wait`, `waitid`, `wait4`, recogida, drenaje o código de salida;
- escritura o cierre de TERMINAL, cierre de CONTROL/pidfd o liberación de
  lease/observador;
- un parser nuevo de CONTROL, `/proc`, ticket, salida, error o TERMINAL;
- getters o serialización de PID, PGID, pidfd, ticket, nonce, `Cmd`, `Process`,
  lease, observador, identidad o autoridad;
- HTTP, SQL, red, Docker, Orquesta, Firecracker, Jailer, O5, O6, producción,
  despliegue o métricas.

## Entrada única y consumo lineal

La única entrada futura es `**agregadoO4aM38`. La operación copia el puntero a
una autoridad privada O4a y pone a cero el puntero del llamador antes de leer
campos. No acepta una lista de recursos ni reconstruye el agregado.

Antes de consumir exige conjuntamente:

- autoidentidad exacta del agregado y de la autoridad conjunta;
- owners `ownerObservador=O4A` y `ownerLease=O4A`;
- lease en 3, observador en 2, mismo registro, generaciones y TID sellados;
- baseline exacto, signo cero, `pending` ausente y pertenencia vigente;
- `Cmd`, `Process`, handle opaco, primario/reserva, CONTROL y TERMINAL iguales
  a los sellos O3c, con identidad y snapshot completos;
- `ahoraCaso` y `finCaso` monotónicos, no cero, relación exacta de 180 s y
  `ahoraCaso < finBootstrap` heredado; una divergencia es AF, no causa;
- `retornoCont >= 0` como constancia raw, no como errno interpretado aún;
- exactamente uno de los cinco discriminantes raw de primera observación.

El consumo usa una autoidentidad O4a privada preasignada y un CAS único de
estado `ENTREGADO_O3C→RECIBIDO_O4A`; no cambia los owners subyacentes 3/2. El
origen queda nulo e inutilizable. Nulo, alias, clon, reuso, estado imposible o
autoidentidad rota no toca recursos. Corrupción de propiedad tras el consumo
es fatal 65/EOF/stdout=0/stderr=0, sin rollback, cierre, log ni E/S.

Los owners subyacentes permanecen O4A durante O4a y O4b. O4b recibe permisos
prestados, nunca ownership. O4c recibe al final el mismo agregado conjunto y
será el único que pueda liberar observador y lease.

## Valores cerrados

### Primera observación raw heredada

Es exactamente una e inmutable:

1. `control_raw`;
2. `senal_raw`;
3. `pidfd_vacio`;
4. `pidfd_terminal_natural`;
5. `pidfd_infraestructura`.

O4a no vuelve a escribir ese CAS. Los eventos posteriores ocupan un latch
separado y nunca sustituyen, mezclan o “mejoran” la observación heredada.
`retornoCont` permanece en otro campo raw separado.

### Causas primarias

La unión privada y no serializable es:

- `CANCELADO/65`;
- `PROTOCOLO/65`;
- `SENAL_INT/130`;
- `SENAL_TERM/143`;
- `PLAZO/65`;
- `SALIDA` con estado todavía desconocido para O4a;
- `INCIDENTE/65`.

O4a solo fija causa, no el estado real de `SALIDA`. O4c medirá el estado tras
el Wait único y validará que sea uno de 0, 64, 65 o 79; cualquier otro valor
convierte el resultado exterior en incidente de cierre sin cambiar la causa
primaria histórica. Una causa distinta de `SALIDA` lleva su estado canónico ya
sellado, pero O4c conserva la autoridad de medir la postcondición.

El latch de causa es CAS único `VACIA→CAUSA`. No existe setter público,
rollback, sustitución por limpieza ni segundo owner.

## Relojes y deadlines

Todos los tiempos son `time.Time` con componente monotónico. No se acepta reloj
civil reconstruido, duración relativa exterior ni marca cero.

### Bootstrap heredado

`finBootstrap` es read-only. O4a acredita al entrar la genealogía
`ahoraCaso < finBootstrap` y que ninguna fase lo recreó. Su vencimiento después
del intento CONT no se convierte en `PLAZO`: el bootstrap terminó al efectuar
el handoff. Relación, componente monotónico o sello inválido al entrar es AF
directo: no se fabrica una causa funcional a partir de corrupción temporal.

### Caso de 180 segundos

O4a conserva, nunca crea, `ahoraCaso` y `finCaso=ahoraCaso+180s`. La decisión
de borde usa una lectura monotónica por ronda:

- evento acreditado antes de la lectura de reloj gana;
- si no hay evento anterior y `ahora >= finCaso`, se enclava `PLAZO/65`;
- `ahora == finCaso` está vencido;
- el deadline no se pausa, extiende, copia, recalcula ni reinicia.

La primera observación de O3c se considera ocurrida antes de cualquier lectura
O4a, incluso si O4a recibe el agregado después de `finCaso`.

### Subplazos de extinción

Solo se crean después de una causa y cada uno una vez:

- al enclavar extinción, `finParadaInicial = ahora + 1s` y
  `finDrenajeRapido = finParadaInicial + 5s`;
- tras STOP estable se fijan juntos `finGracia = ahora + 2s`,
  `finParadaFinal = finGracia + 1s` y
  `finDrenajeCooperativo = finParadaFinal + 5s`;
- para SALIDA natural, `finDrenajeNatural = ahora + 5s`;
- si STOP inicial no se acredita se usa `finDrenajeRapido` y se omiten TERM,
  CONT, gracia y parada final.

La aritmética rechaza overflow y exige marcas estrictamente posteriores. O4a
crea el subplazo antes de emitir la autorización correspondiente. O4b no puede
crear ni modificar tiempo; O4c solo consume `finDrenaje` para su observación y
limpieza. Ningún error permite reiniciar un subplazo.

## Precedencia total y arbitraje

Cada ronda es inmediata y no recolectora. Cada syscall de CONTROL, observador
—incluido su `Gettid` interno— o pidfd usa un permiso lease distinto mediante
`lease.comenzar→syscall→lease.consolidarCritico`. No usa `comenzarCritico`, no
comparte/reutiliza permisos ni deja `pending`; consolida antes de interpretar
el retorno. Fallo lease o de consolidación es AF, nunca causa. La ronda se
prepara en memoria y después lee en orden fijo:

1. causa primaria ya enclavada;
2. retorno CONT raw distinto de cero;
3. primera observación O3c, una sola vez;
4. CONTROL mediante el parser existente hasta EAGAIN;
5. observador/señal y autoridad de hilo;
6. pidfd no recolector;
7. reloj monotónico del caso o de la etapa activa;
8. ausencia de evento: exactamente un subpaso mecánico.

La traducción causal cerrada es:

- retorno CONT raw distinto de cero → `INCIDENTE/65`;
- `control_raw` o CONTROL posterior canónico conserva exactamente
  `CANCELADO`, `PROTOCOLO`, `SENAL_INT` o `SENAL_TERM`; framing, parcial+EOF,
  presupuesto, secuencia o dominio inválidos → `PROTOCOLO/65`; EOF limpio →
  `CANCELADO/65`;
- `senal_raw`, cambio de contador, TID/PPID/PDEATHSIG o autoridad de señal →
  `SENAL_INT/130` o `SENAL_TERM/143` solo si el observador sellado identifica
  exactamente ese signo; cualquier ambigüedad → `INCIDENTE/65`;
- `pidfd_terminal_natural` o terminalidad posterior exacta → `SALIDA`;
- `pidfd_infraestructura`, ambas referencias no fiables, identidad o flags
  divergentes → `INCIDENTE/65`;
- `pidfd_vacio` continúa a eventos posteriores y al deadline;
- sin evento anterior y caso vencido → `PLAZO/65`.

CONTROL gana a señal; señal gana a pidfd; pidfd gana al deadline. La primera
observación heredada se procesa antes de lecturas nuevas porque ya ocurrió. Si
la observación heredada es `pidfd_vacio`, no bloquea los eventos posteriores.
Una cancelación completa ya disponible gana a terminalidad simultánea. Un
fallo de limpieza futuro nunca cambia la causa primaria.

## Máquina total O4a

Estados privados, no exportables:

```text
A0 recibido
  -> A1 autoridad_acreditada | AF fatal
A1 autoridad_acreditada
  -> A2 arbitrando | AF fatal
A2 arbitrando
  -> A2 arbitrando | A3 causa_enclavada | AF fatal
A3 causa_enclavada
  -> A4 permiso_etapa_preparado | A7 entrega_o4c_preparada | AF fatal
A4 permiso_etapa_preparado
  -> A5 esperando_resultado_o4b | AF fatal
A5 esperando_resultado_o4b
  -> A3 causa_enclavada | A4 permiso_etapa_preparado |
     A7 entrega_o4c_preparada | AF fatal
A7 entrega_o4c_preparada
  -> A8 entregado_o4c | AF fatal
```

No existe transición desde A8 o AF. A2 puede repetir solo una ronda acotada y
sin efecto; la propia máquina productiva exige el deadline absoluto activo y
una única ronda pendiente, y sale a A3/AF al vencer o divergir. El conductor
solo acredita esa propiedad, nunca la impone. A4 no es retornable. A5 acepta
exactamente un resultado opaco de la autorización que emitió; alias, replay,
etapa distinta o resultado forjado es fatal.

`SALIDA` o terminalidad ya acreditada va de A3 a A7 sin permiso de señal. Las
causas de extinción van A3→A4. Un error de CONT o infraestructura también
converge por autorizaciones; nunca ejecuta retirada local O3c.

## Autorizaciones opacas para O4b

La autoridad O4a preasigna un único slot privado con autoidentidad, generación,
TID, etapa, cardinalidad, deadline, operación y estado atómico. No contiene
getters ni expone el pidfd; O4b lo consume junto con el agregado privado dentro
del mismo paquete y bajo lease. Estados: `VACIO→EMITIDO→CONSUMIDO`; no hay
reuso, clon, rollback o serialización.

Etapas autorizables:

1. `PARADA_INICIAL`: exactamente un STOP de grupo antes de
   `finParadaInicial`;
2. `TERMINAR_Y_REANUDAR`: exactamente TERM y luego CONT de grupo, en ese orden,
   dentro de `finGracia`; es una etapa compuesta indivisible para causalidad,
   aunque O4b consolida un permiso lease por syscall;
3. `PARADA_FINAL`: exactamente un STOP de grupo antes de `finParadaFinal`;
4. `MATAR_GRUPO`: exactamente un KILL de grupo cuando la rama autorizada
   acredita grupo todavía vivo. El permiso porta como límite absoluto el
   `finDrenajeRapido` o `finDrenajeCooperativo` de su rama; O4a lo emite
   inmediatamente tras el resultado consolidado y una lectura final
   `ahora < finDrenaje`. O4b deberá hacer de KILL su siguiente syscall tras
   validar el permiso. En el borde `ahora >= finDrenaje` domina AF y no se
   autoriza una señal tardía.

O4a no autoriza segundo efecto de la misma etapa. O4b devuelve solo un resultado
opaco sellado: etapa, intento cardinal, retornos raw, evidencia de estabilidad
no recolectora y reloj observado; no devuelve PID/pidfd. O4a interpreta el
resultado antes de preparar la siguiente etapa. Error o EINTR nunca autoriza
reintento. Antes de A3 puede originar `INCIDENTE/65`; después fija solo
`INCIDENTE_DE_CIERRE`, conserva la causa primaria y sigue la tabla inferior.
Consolidación lease incierta es AF y no autoriza KILL.

O4b-P0 deberá definir la materialización exacta de estas cuatro etapas. Este
contrato no autoriza ninguna syscall.

### Autómata total de etapas

| Estado/resultado | Única transición autorizada |
| --- | --- |
| A3 con SALIDA o terminalidad exacta | Crear `finDrenajeNatural`; A7, cero señal. |
| A3 con causa de extinción | Crear `finParadaInicial` y `finDrenajeRapido`; autorizar PARADA_INICIAL; A5. |
| STOP estable antes del límite | Crear juntos gracia/parada final/drenaje cooperativo; autorizar TERMINAR_Y_REANUDAR; A5. |
| STOP no estable, error/EINTR o límite vencido | Fijar incidente de cierre; omitir TERM/CONT/parada final; autorizar MATAR_GRUPO con drenaje rápido; A5. |
| TERM devuelve error raw con consolidación acreditada | Omitir CONT; incidente de cierre; autorizar MATAR_GRUPO con drenaje cooperativo. |
| TERM o CONT no acredita `consolidarCritico` | AF directo; cero autorización posterior. |
| TERM consolida y CONT falla/EINTR | Conservar ambos raw, no reintentar; incidente de cierre; autorizar MATAR_GRUPO con drenaje cooperativo. |
| TERM y CONT consolidan; terminalidad antes de gracia | A7 con drenaje cooperativo, sin otra señal. |
| TERM y CONT consolidan; grupo vivo al vencer gracia | Autorizar PARADA_FINAL; A5. |
| PARADA_FINAL observa terminalidad | A7 con drenaje cooperativo, cero KILL. |
| PARADA_FINAL estable y grupo presente | Autorizar inmediatamente MATAR_GRUPO con límite `finDrenajeCooperativo`; A5. |
| PARADA_FINAL no estable/error/EINTR/límite | Incidente de cierre; autorizar MATAR_GRUPO; A5. |
| KILL intentado, incluso error/EINTR | No reintentar ni cambiar causa; A7 con el drenaje de la rama; O4c decide terminalidad. |
| Resultado forjado, etapa/deadline/cardinalidad ajeno u ownership incierto | AF directo, sin nueva autorización. |

Solo SALIDA evita siempre señales. Todo camino legítimo llega una vez a A7 o
AF; no existe busy-loop. `INCIDENTE_DE_CIERRE` podrá forzar estado exterior 65
y cuarentena en O4c, pero nunca sustituye la causa primaria.
En todas las etapas, «error/EINTR» significa retorno raw con permiso lease
correctamente consolidado. Fallo o duda de `consolidarCritico` domina AF
directo: conserva el intento/raw solo como evidencia no interpretable, no
autoriza ni ejecuta ningún syscall posterior y no fija latch alguno.

## Entrega a O4c

O4a entrega a O4c un agregado privado y conjunto que contiene:

- la custodia original completa y owners todavía O4A;
- causa primaria inmutable y, separadamente, latch de incidente de cierre en
  su valor exacto —vacío o enclavado una vez—;
- primera observación y retorno CONT raw originales;
- todos los deadlines originales y de etapa, sin reinicio;
- historial privado de autorizaciones/resultados O4b de cardinalidad cerrada;
- indicación opaca de terminalidad ya observada o extinción intentada;
- TERMINAL todavía abierto y nunca escrito.

La transferencia futura será conjunta, CAS observador-owner antes de
lease-owner, sin cambiar estados 3/2, y deberá añadir owners privados O4C sin
editar los tipos O3c congelados o exponerlos. La forma exacta pertenece a
O4c-P0; hasta entonces A7/A8 son estados de contrato y no código.

Si una transferencia se parte después del primer CAS, domina AF directo sin
rollback, cierre, log o E/S. O4c no puede cambiar causa ni deadlines.

## Cancelación y fallo cerrado

Cancelación llega solo por CONTROL canónico o por observador sellado. No existe
`context.Context`, `CommandContext`, `Cmd.Cancel`, canal, callback o hook que
fabrique autoridad. EOF limpio se clasifica `CANCELADO`; framing o dominio
adverso, `PROTOCOLO`; señal ambigua o propiedad dudosa, `INCIDENTE`.

Antes de A3, un fallo controlable con custodia completa puede fijar
`INCIDENTE/65`. Después de A3 fija como máximo `INCIDENTE_DE_CIERRE`, conserva
la causa primaria y converge por las etapas exactas O4b/O4c.
Si no puede acreditarse ownership, lease, TID, identidad, deadline o entrega
conjunta, AF llama directamente a la fatalidad heredada: estado 65, EOF/no
retorno, stdout/stderr cero, sin señal, cierre, log ni otra E/S posterior.

No hay retirada local post-CONT, postausencia ficticia ni éxito degradado.

## Custodia por frontera

| Recurso | O4a | O4b | O4c |
| --- | --- | --- | --- |
| `Cmd`/`Process`/handle | custodia opaca | préstamo sin getter | Wait único y liberación |
| pidfd primario/reserva | sondeo no recolector | señales autorizadas bajo lease | ESRCH y cierre final |
| CONTROL | lectura parser existente | inmóvil | cierre final |
| TERMINAL | inmóvil, nunca escribe/cierra | inmóvil | única escritura y cierre |
| lease/observador | owners O4A, estados 3/2 | permisos prestados, no owner | liberación final desde 3/2 |
| `finBootstrap` | verifica genealogía | no usa | conserva evidencia |
| `finCaso` 180 s | evalúa, no reinicia | no modifica | conserva evidencia |
| subplazos | crea una vez | consume etapa | consume drenaje |
| causa/primera/raw CONT | fija/preserva | no cambia | conserva en terminal |

## APIs y conductas prohibidas en O4a

- `pidfd_send_signal`, `kill`, `Process.Signal/Kill` o cualquier señal;
- `cmd.Wait`, `waitid`, `wait4`, `WaitDelay`, goroutine de espera o recogida;
- `Close`, escritura de TERMINAL, CONTROL, stdout o stderr;
- `pidfd_open`, `dup*`, `F_DUPFD*`, `os.NewFile` o cuarta referencia;
- PID/PGID numérico, `/proc` nuevo o parser nuevo;
- `time.NewTimer`, `After`, `Sleep`, ticker, contexto con deadline o reloj civil;
- reinicio de bootstrap/caso/subplazo, pausa o duración exterior;
- getter, interfaz pública, ticket, nonce, log o serialización de autoridad;
- goroutine, canal, global mutable, `init`, hook, callback, mock o función
  variable en el grafo productivo;
- O4b/O4c efectivo, O5/O6, HTTP, SQL, red, producción o despliegue.

Las esperas futuras O4a serán un único ciclo propietario basado en poll
existente y deadline absoluto; P0 no lo implementa.

## DAG obligatorio de minitareas O4a

Cada fila depende de la anterior publicada con doble GO y CI 5/5:

| ID | Responsabilidad única | Write-set máximo | Cierre observable |
| --- | --- | --- | --- |
| O4A-P1-AUTORIDAD | Consumir C5, máquina A0--AF, sellos y causa vacía. | Un Go productivo nuevo + prueba focal. | Anti-alias/replay; cero reloj/evento/efecto. |
| O4A-P2-SEMILLA | Traducir raw CONT y primera observación a causa o continuidad. | Un Go productivo nuevo + prueba focal; P1 inmóvil salvo sello mínimo revisado. | Unión cerrada y precedencia heredada; cero poll/señal. |
| O4A-P3-ARBITRAJE | Ronda CONTROL→observador→pidfd→reloj y latch causal. | Un Go productivo nuevo + prueba focal. | Deadline 180 exacto, simultaneidad y causa CAS única. |
| O4A-P4-ETAPAS | Subplazos y permisos opacos one-shot para O4b, sin syscall. | Un Go productivo nuevo + prueba focal. | Secuencia/ramas exactas, cero señal/Wait. |
| O4A-P5-HANDOFF | Agregado conjunto contractual hacia O4c, sin limpiarlo. | Un Go productivo nuevo + prueba focal. | A8 completo o AF; TERMINAL y recursos intactos. |
| O4A-P6-CONDUCTOR | Conductor durable, black-box, AST/tipos/DAG y mutantes. | Solo herramientas/pruebas externas. | Normal/race, mutantes muertos y residuos cero. |
| O4A-P7-EVIDENCIA | Ledger, checkpoint, revisiones y CI. | Solo documentación/evidencia propia. | Doble GO, push normal y CI 5/5. |

Aristas exactas O4a: `P0→P1→P2→P3→P4→P5→P6→P7`.

- `O4B-P0-CONTRATO` depende de este contrato O4a publicado y CI 5/5, y deberá
  cerrarse antes de cualquier implementación O4b o de O4A-P4 material.
- `O4C-P0-CONTRATO` depende de los contratos O4a y O4b publicados y CI 5/5,
  y deberá cerrarse antes de O4A-P5 material o cualquier implementación O4c.
- No hay arista de retorno O4→O3 ni edición paralela del mismo fichero.

P0 no abre P1, O4b, O4c, O5 ni O6 por sí solo; dirección asigna cada paso.

## Presupuestos y archivos futuros

- máximo cinco fuentes productivas nuevas O4A-P1--P5, una por minitarea;
- una prueba focal nueva por fuente;
- objetivo 300--500 líneas; parada 650; tope duro 800 por fichero;
- P1--P5 previos quedan byte-inmutables salvo sello mínimo demostrado en una
  decisión separada y doblemente revisada;
- P6 puede añadir como máximo una unidad de conductor, una AST/tipos/DAG y una
  de mutantes, sin producción;
- P7 solo evidencia y documentación propias;
- runner, workflow, SQL, adaptadores, O3a/O3b/O3c y documentos transversales
  de dirección permanecen byte a byte.

Superar 650, necesitar un getter, tocar una fuente O3 o mezclar dos
responsabilidades obliga a detenerse y dividir documentalmente.

## Matriz mínima de pruebas

| ID | Oráculo causal |
| --- | --- |
| A01 | Entrada C5 se consume una vez; nulo/alias/clon/reuso no toca recursos. |
| A02 | Owners O4A, lease3/observador2, registro/generación/TID/baseline/pending exactos. |
| A03 | Custodia, identidad, tres pidfd, CONTROL/TERMINAL y autoidentidades sellados. |
| A04 | Primera observación conserva exactamente uno de cinco y raw CONT separado. |
| A05 | Error raw CONT enclava INCIDENTE antes de cualquier observación nueva. |
| A06 | CONTROL canónico/framing/EOF traduce causa exacta y gana simultaneidad. |
| A07 | Señal exacta traduce INT/TERM; ambigüedad es INCIDENTE y gana a pidfd. |
| A08 | Pidfd natural produce SALIDA; infraestructura INCIDENTE; vacío continúa. |
| A09 | `finCaso-ahoraCaso=180s`, borde igual vence, marca nunca se reinicia. |
| A10 | Primera observación prehandoff gana aunque O4a reciba tras el deadline. |
| A11 | Causa CAS única e incidente de cierre separado no la sustituye. |
| A12 | Subplazos 1/2/1/5 exactos, monotónicos, únicos y sin overflow/reinicio. |
| A13 | STOP estable autoriza TERM→CONT; STOP fallido salta a KILL sin gracia. |
| A14 | Cada permiso O4b es opaco, one-shot, etapa/cardinalidad/deadline exactos. |
| A15 | Error/EINTR/replay de O4b no reintenta y no cambia causa. |
| A16 | SALIDA/terminalidad entrega a O4c sin señal; extinción entrega historial completo. |
| A17 | O4a nunca señala, Wait, drena, cierra/escribe TERMINAL ni libera. |
| A18 | O4b no decide causa/Wait; O4c no cambia causa/plazo ni emite señales funcionales. |
| A19 | Partición de autoridad o ownership incierto produce 65/EOF/0/0 sin E/S. |
| A20 | 100 normal +100 race: deltas FD/hijo/zombi/grupo/temporal cero. |
| A21 | Cada lectura CONTROL/observador/pidfd usa permiso lease separado y consolida antes de interpretar. |
| A22 | Error raw consolidado converge; fallo de consolidación en cualquier etapa es AF sin CONT/KILL posterior. |

Black-box ejecuta fixtures sintéticos aislados, PGID propio y limpieza exterior
segura. Ningún negativo usa procesos ajenos. Los BF son procesos directos y
acreditan 65, EOF/no retorno y stdout/stderr cero. No hay SKIP ni retry.

## Plan de mutantes

Cada alternativa se expande a mutante atómico compilable con patrón de
cardinalidad uno y oráculo causal; una familia no acredita sus alternativas:

- A01 omitir consumo, no anular origen, aceptar nulo/alias/clon/reuso;
- A02 aceptar owner/estado/registro/generación/TID/baseline/pending adverso;
- A03 omitir autoidentidad, recurso, sello, identidad o admitir cuarta referencia;
- A04 cambiar/combinar/borrar primera observación o mezclar raw CONT;
- A05 ignorar, reinterpretar o retrasar error CONT;
- A06 invertir CONTROL, aceptar framing/EOF/dominio o causa transportada adversa;
- A07 omitir signo/contador/autoridad o invertir señal/pidfd;
- A08 confundir natural/vacío/infraestructura o usar evento recolector;
- A09 recrear, extender, truncar o comparar mal 180 s/borde;
- A10 evaluar deadline antes de la primera observación heredada;
- A11 segundo CAS, Store/Swap, sustituir causa por limpieza o estado SALIDA prematuro;
- A12 cambiar 1/2/1/5, reiniciar, usar reloj civil/timer o aceptar overflow;
- A13 reordenar/omitir etapa, autorizar gracia tras STOP fallido o KILL temprano;
- A14 compartir, clonar, forjar, reutilizar o serializar permiso O4b;
- A15 reintentar efecto, tratar EINTR como no intento o perder retorno raw;
- A16 entregar parcial, señalizar SALIDA o perder historial/deadline;
- A17 añadir señal, Wait/waitid/wait4, close/write TERMINAL o liberación;
- A18 mover causa a O4b, Wait a O4b, señal a O4c o tiempo fuera de O4a;
- A19 rollback, log/cierre/E/S antes de fatal o retorno tras fatal;
- A20 falsear inventario/residuos, aceptar SKIP/retry o reutilizar evidencia;
- A21 omitir/reordenar/compartir/reusar permiso de lectura, interpretar antes
  de consolidar o convertir fallo lease en causa;
- A22 confundir retorno raw con fallo de consolidación, autorizar efecto tras
  consolidación incierta o fijar latch antes de acreditarla;
- A23 añadir getter/PID/pidfd/ticket/nonce, global/init/hook/callback/goroutine/canal;
- A24 cambiar límites 1024/4/4096/8, 180/1/2/1/5 s, parada 650 o tope 800.

Todos pasan primero gofmt, build y vet. AST/tipos/DAG acredita máquina total,
ownership acíclico, causa CAS única, orden raw→CONTROL→señal→pidfd→reloj,
deadlines, permisos, fronteras y ausencia de APIs prohibidas. Todo mutante debe
morir; SHA global, timeout o no compilación no cuentan como muerte.

## Conductor, residuos y evidencia

O4A-P6 deberá ser ejecutable desde checkout limpio y ligar por SHA-256 base,
toolchain, fuentes, conductor, AST, catálogo y resultados. Ejecutará normal y
race, 100+100 capturas, BF directos, permisos simulados sin señal real y
black-box solo sobre procesos sintéticos aislados. Acreditará delta cero de FD,
hijos, zombis, grupos y temporales, y PGID ausente por ESRCH.

Puertas: focal normal/race, gofmt, vet, AST/tipos/DAG, todos los mutantes,
`go test ./...`, `go test -race ./...`, `go vet ./...`, calidad, Gitleaks,
enlaces, hashes, `git diff --check`, doble revisión, push normal y CI 5/5.

## Identidad, tenant y datos

O4a no crea identidad funcional ni tenant. Conserva solo referencias técnicas
opacas ya selladas para custodiar el proceso. No registra PID/pidfd, ticket,
nonce, selector, identidad `/proc`, datos RRHH ni PII. No hay decisión sobre
personas, HTTP, SQL o datos reales. La indisponibilidad nunca equivale a
autorización o éxito.

## Paradas duras

Se detiene si O4a envía una señal; llama Wait/waitid/wait4; recoge/drena;
escribe/cierra TERMINAL; libera recursos; recrea 180 s/bootstrap; cambia una
causa; evalúa deadline antes de la primera observación; usa reloj civil/timer;
reintenta un permiso; expone autoridad; abre O4b/O4c sin sus contratos; toca
O3; supera write-set/límites; sobrevive un mutante; queda residuo; falta doble
GO o CI 5/5.

## Métricas y autorización

Este contrato no cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` ni producción `NO-GO`.

Secuencia: doble revisión documental; corrección y relectura completa; commit
pequeño; push normal; CI 5/5. Solo dirección puede después asignar
`O4A-P1-AUTORIDAD` y los contratos O4b/O4c en sus dependencias exactas. El
productor no aprueba, integra ni fusiona su propio trabajo.

# Decisión F0-H0b/C4b-2/G2-O — O4c terminalidad y limpieza

Fecha: 13 de agosto de 2026.

Identificador: `O4C-P0-CONTRATO`.

Estado: **CANDIDATO RECONCILIADO A REVISIÓN**. Este documento no se
autoaprueba, no abre
código O4c y no acredita `O4A-P4-ETAPAS`, `O4A-P5-HANDOFF`, O5, O6,
integración, producción, despliegue ni cambio de métricas.

## Resultado contractual

O4c queda delimitado como único propietario funcional de la terminalidad y
limpieza posteriores a O4a/O4b. Consume una custodia conjunta ya transferida,
observa terminalidad antes del límite absoluto prestado, ejecuta exactamente
un `cmd.Wait`, drena adoptados terminales hasta `ECHILD`, exige ausencia de
grupo por `ESRCH`, emite y cierra un único TERMINAL canónico cuando existe una
postcondición normal completa, cierra los recursos poseídos y libera primero
el observador y por último la lease.

O4c nunca decide ni sustituye la causa primaria, nunca crea o amplía plazos,
nunca envía señales funcionales y nunca inventa postausencia. Incertidumbre de
propiedad, terminalidad, Wait, drenaje, grupo, escritura, cierre o liberación
es fallo cerrado: estado exterior 65, cuarentena y cero caso siguiente.

## Autoridad y genealogía exactas

La reconciliación parte de la base de producto exacta
`de2c9be8ea25c25a4e4173d1fdf6f5dcdfb769c8`, en la rama productora
`trabajo/o4c-p0-reconciliacion-20260831`. Reutiliza sin reprogramar ni cambiar
semántica los dos blobs de la candidata fuente
`de4ee5a11c3611e56ab09899da860ef88647ea0c`, cuyo padre es
`64351a469df4e7bba4850ef1500d9e2c4bf378de`; esa genealogía fuente no se
convierte en genealogía de producto ni traslada sus revisiones al nuevo hash.

Autoridades normativas directas:

- [contrato O4a](decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md),
  535 líneas, SHA-256
  `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`,
  publicado en `7a28d99b380c621de3c85964ad770224497d8b90` con doble GO;
- [checkpoint O4a](revisiones/checkpoint_f0_h0b_c4b2_g2o_o4a_p0_contrato_2026-08-11.md),
  SHA-256
  `0ba877cd831f7957d36cab885644dd6f8e5069d80e1629669b92d282e99d1513`;
- [contrato O4b](decision_f0_h0b_c4b2_g2o_o4b_senales_grupo_2026-08-12.md),
  443 líneas, SHA-256
  `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`,
  y sus dos revisiones independientes `GO`, `P0=P1=P2=0`;
- [checkpoint O4b](revisiones/checkpoint_f0_h0b_c4b2_g2o_o4b_p0_contrato_2026-08-12.md),
  SHA-256
  `1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45`;
- publicación O4b `5345d5d097b51ab3567983f048feabeceaf2957b` y CI
  `31546649383`, cinco de cinco puertas verdes, según el relevo de dirección;
- [contrato O3c](decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md),
  SHA-256
  `f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6`;
- [ledger final O3c](revisiones/revision_f0_h0b_c4b2_g2o_o3c_codigo_final_2026-08-11.md),
  SHA-256
  `a1de39d1c80492cae8b6b858a096d8f3051b346913064fe51a83dd6573dbb3b1`;
- [corrección canónica O1a](enmienda_f0_h0b_c4b2_g2o0_correccion_o1a_2026-08-05.md),
  SHA-256
  `29c1520f6aab91d832ec6ddb41efd75df587d240b8fb1e7dc51f107df831a659`.

La propuesta conjunta G2-O del 5 de agosto conserva estado histórico NO-GO y
no es autoridad. O1a solo aporta la gramática canónica ya aceptada; O3c aporta
la custodia y sus primitivas; O4a aporta causa, precedencias y tiempo; O4b
aporta únicamente los efectos de grupo ya autorizados.

La cadena O4A-P4
`1e75c829215c43b4472908e9e00acc255aa016d9` →
`4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9` →
`1f8186cf0705043fea638db4ba1ab4ff086455a3` →
`6a7a83b252a24971d1255f19c0e30ae7f4e4eb90` →
`2b7eaf498f8f68a90b66c004f166a62f070b5064` es histórica y fue sustituida.
La cadena vigente es `99992cf44afb362a34f346ee2a5a5a8408ea7afd` →
`6c56135b0eee69cff460e3df6562e0b557e0c49d`, integrada en producto mediante
`01c5c1fb19458dff3794cae69c4777a5399dd60f`, ancestro de la base de
reconciliación. Este P0 no reabre ni revisa O4A-P4 y tampoco hereda sus GO.

## Alcance exclusivo

O4c posee únicamente:

- consumo lineal del agregado conjunto producido por O4A-P5;
- owners privados O4C sobre observador y lease, sin alterar sus estados
  físicos heredados 2/3;
- observación final no recolectora de terminalidad por pidfd fiable;
- el único `cmd.Wait` funcional después de `Start`;
- estado real del Bash obtenido del `ProcessState` del Wait;
- drenaje no bloqueante de adoptados ya terminales hasta `ECHILD`;
- prueba de grupo ausente exclusivamente por `ESRCH`;
- codificación, escritura lógica única y cierre de TERMINAL;
- cierre final de pidfd primario/reserva y CONTROL;
- inventario cero de los recursos poseídos;
- liberación ordenada observador→lease y resultado exterior cerrado.

Quedan fuera:

- causa, precedencia, deadlines, subplazos y selección de etapa O4a;
- STOP, TERM, CONT, KILL o cualquier señal O4b;
- lectura adicional de CONTROL, observador o `/proc` para decidir causa;
- `Start`, ticket, sobre, ACK, recibo Shell, coproc o activación del modo;
- O5a--O5d, O6a--O6c, runner, Bash, workflow, SQL, HTTP, red o despliegue;
- API pública, getter, serialización, log o exposición de PID/pidfd/nonce;
- una retirada alternativa, un segundo Wait o limpieza concurrente.

## Entrada conjunta exigida a O4A-P5

O4A-P5 materializará un único agregado privado O4c, preasignado antes del
último verde y no serializable. No duplica recursos: conserva referencias a
la misma autoridad O4a y la misma custodia física. El agregado contiene o
permite cotejar internamente:

1. autoidentidad y estado O4a exactamente A7 antes de transferir y A8 al
   entregar;
2. autoridad conjunta O3c con ambos owners O4A;
3. custodia, `Cmd`, `Process`, handle opaco, pidfd primario/reserva, CONTROL y
   TERMINAL originales;
4. lease, observador, registro, generaciones, TID, PPID y baseline exactos;
5. identidad sellada Bash completa PID/PPID/PGID/SID/starttime;
6. nonce de 64 hexadecimales minúsculos conservado dentro del controlador;
7. causa primaria O4a inmutable y latch separado de incidente de cierre;
8. primera observación, raw CONT e historial O4b completos, sin reinterpretar;
9. `finBootstrap`, `ahoraCaso`, `finCaso` y todos los subplazos originales;
10. una única clase de drenaje elegida por O4a —natural, rápida o
    cooperativa— y su `time.Time` monotónico absoluto exacto;
11. indicación opaca de terminalidad ya observada o extinción intentada;
12. TERMINAL regular privado, abierto, tamaño y offset cero, nunca escrito.

El límite elegido coincide por igualdad estructural Go `==` con exactamente
uno de `finDrenajeNatural`, `finDrenajeRapido` o
`finDrenajeCooperativo`. Marca cero, tiempo civil reconstruido, combinación de
ramas o más de un límite activo son fatales. O4c recibe un instante absoluto,
nunca una duración.

O4A-P5 consume su autoridad con un CAS de custodia
`RECIBIDA_O4A→ENTREGADA_O4A`, transfiere primero
`ownerObservador O4A→O4C` y después `ownerLease O4A→O4C`, conserva estados
subyacentes observador 2 y lease 3, anula el origen y consolida A8. La
constante privada O4C usa un valor nuevo, distinto de INACTIVO, O3C, O4A y
LIBERADO, sin editar el tipo ni las constantes O3c congeladas.

Si el primer owner cambia y el segundo no, o si falla cualquier CAS posterior
al último verde, domina AF directo: no rollback, cierre, señal, log, escritura
ni retorno. Entre el último verde y A8 solo hay CAS y asignaciones infalibles
sobre memoria preasignada.

## Consumo y autoridad O4c

O4c recibe un doble puntero, anula el puntero del llamador antes de observar
el contenido y acepta un único ganador. Nulo, origen ya consumido, alias,
clon, autoidentidad falsa, replay o carrera devuelven consumo inválido sin
tocar recursos cuando todavía no existe propiedad; divergencia dentro de una
custodia que afirma owners O4C es fatal.

La validación previa exige simultáneamente:

- A8 terminal, causa primaria válida y latch 0/1;
- estado de custodia `ENTREGADA_O4A`, que O4c consume una vez;
- ambos owners O4C y estados físicos observador 2/lease 3;
- mismo registro, generaciones, TID, mapa, `pending` vacío y baseline sellado;
- `Cmd.Process` vivo como handle Go, handle opaco y tres pidfd distintos;
- primario/reserva/opaco con la misma identidad física y `FD_CLOEXEC`;
- CONTROL O_RDONLY|O_NONBLOCK y TERMINAL O_WRONLY, regular privado 0600,
  enlace único, CLOEXEC, tamaño/offset cero;
- identidad Bash igual al sello O3b y controlador S3/L0 o causa terminal
  transportada coherente;
- deadlines monotónicos exactos, sin cero, extensión o recomputación.

Solo después de todos esos cotejos se consolida
`ENTREGADA_O4A→RECIBIDA_O4C`. La autoridad O4c no vuelve a validar mediante
PID numérico, nueva ruta `/proc`, ticket o datos del exterior.

## Máquina total O4c

```text
OC0 recibido
  -> OC1 autoridad_acreditada | OCF fatal
OC1 autoridad_acreditada
  -> OC2 esperando_terminalidad | OCF fatal
OC2 esperando_terminalidad
  -> OC2 esperando_terminalidad | OC3 wait_consolidado | OCF fatal
OC3 wait_consolidado
  -> OC4 drenaje_acreditado | OCF fatal
OC4 drenaje_acreditado
  -> OC5 terminal_resuelto | OCF fatal
OC5 terminal_resuelto
  -> OC6 recursos_cerrados | OCF fatal
OC6 recursos_cerrados
  -> OC7 liberado | OCF fatal
```

OC2 puede repetir únicamente un sondeo acotado mientras
`ahora.Before(finDrenaje)`. No existe busy-loop sin reloj ni transición desde
OC7/OCF. Cada syscall cruza la lease mediante un permiso propio y su raw solo
se interpreta después de consolidarlo. Duda en `comenzar` o
`consolidarCritico` es OCF, no un incidente funcional recuperable.

La misma goroutine fijada al TID heredado recorre OC0--OC7. No hay goroutine de
Wait, canal, callback, hook, timer ni actor de limpieza auxiliar.

## Terminalidad antes de Wait

O4c selecciona primario como única autoridad funcional. Reserva solo coteja la
misma terminalidad; nunca sustituye al primario, nunca recibe señales y no es
fallback. Cada pareja de sondeos reutiliza `pidfdVivoBarreraO3bM38` bajo
permisos separados y exige resultados naturales concordantes.

- Si O4a entregó terminalidad ya observada, O4c la confirma con una pareja
  inmediata primario/reserva antes del borde.
- Si hubo extinción intentada, O4c repite parejas acotadas mientras el reloj
  monotónico permanece estrictamente anterior al límite prestado.
- Ambos terminales permiten OC3.
- Ambos vivos permiten otra vuelta solo si `ahora < finDrenaje`.
- Discordancia, identidad/flags adversos, error de poll, igualdad o vencimiento
  son OCF: cero Wait y frontera externa.

O4c no usa `waitid`, señal cero, `/proc` ni `ProcessState` para anticipar
terminalidad. No duerme hasta una duración recreada. El límite no se pausa,
redondea, amplía ni reinicia.

## Wait único y estado real

Solo desde terminalidad acreditada antes del límite se llama exactamente una
vez a `esperarConLeaseO3aM38`. Esa primitiva ejecuta el único `cmd.Wait`,
consume el handle pidfd opaco y deja `pidfdOpaco=-1` mediante consolidación
física de la lease.

El retorno se acepta únicamente como:

- `nil` y `ProcessState.ExitCode()==0`; o
- `*exec.ExitError` perteneciente al mismo `Cmd`, con `ProcessState` no nulo.

Cualquier error ajeno a esos dos casos o ProcessState ausente enclava
incidente de cierre. Para causa `SALIDA`, el estado terminal normal es
exactamente el código real y debe ser uno de `0,64,65,79`; señal u otro código
enclavan incidente. Para las demás causas, el Wait acredita recolección, pero
el estado de la trama permanece el canónico ya ligado a la causa: 65, 130 o
143. El estado real —incluida muerte por una señal de extinción previamente
autorizada— se conserva como evidencia privada y no sustituye causa ni estado.

No existe segundo `Wait`, retry de EINTR, `Process.Wait`, Wait4 del líder ni
recogida por otra frontera.

## Drenaje, ECHILD y ESRCH

Tras OC3, O4c drena exclusivamente hijos adoptados ya terminales. Cada
`syscall.Wait4(-1, ..., WNOHANG, nil)` tiene permiso lease propio y se
consolida antes de interpretar PID, estado o error.

- PID positivo acredita un adoptado recogido y permite otra vuelta antes del
  mismo límite absoluto.
- `ECHILD` exacto cierra el drenaje con `ADOPTADOS_PENDIENTES=0`.
- PID cero, otro error, límite igual/vencido o estado no acreditable son OCF;
  nunca se espera bloqueando a un adoptado vivo.

Después de `ECHILD`, y mientras primario/reserva explícitos siguen abiertos,
O4c reutiliza `sondaGrupoCeroO3bM38` sobre el primario. Solo
`errors.Is(raw, syscall.ESRCH)` acredita `GRUPO_AUSENTE=1`. Retorno cero,
EPERM, EINTR u otro error no prueban ausencia y son OCF. No existe fallback a
PID/PGID, `/proc`, reserva o señal repetida.

`ECHILD` precede estrictamente a `ESRCH`; ambos preceden a cualquier cierre o
escritura normal.

## TERMINAL canónico

O4c reutiliza `tramaM38`, `codificarTramaM38` y
`decodificarTramaM38("TERMINAL", ...)`; no crea otra gramática ni otro
parser. La trama lógica única, con barras y LF literales, es:

```text
V1|TERMINAL|NONCE|PID_SUPERVISOR|FASE_ORIGEN|ESTADO|CAUSA|PID_BASH|PPID|PGID|SID|INICIO|BASH_CREADO|BASH_ESPERADO|ADOPTADOS_PENDIENTES|GRUPO_AUSENTE\n
```

Fuentes exactas:

| Campo | Autoridad O4c |
| --- | --- |
| `NONCE` | los 64 bytes sellados de `control.nonce`; nunca getter ni log |
| `PID_SUPERVISOR` | `os.Getpid()` medido una vez bajo permiso lease y decimal canónico |
| `FASE_ORIGEN` | literal `S3`; O3c entregó controlador S3/L0 y ninguna autoridad publicada materializa S4 |
| `ESTADO` | estado canónico ligado a la causa; para SALIDA, estado real permitido del Wait |
| `CAUSA` | causa primaria O4a inmutable |
| proceso | identidad O3b sellada PID/PPID/PGID/SID/starttime |
| indicadores | literales `1|1|0|1`, solo tras Wait/ECHILD/ESRCH |

La fase no se “mejora” a S4 por haber ejecutado CONT: ninguna transición
publicada muta el controlador S3. S5/S6 tampoco son fase de origen.

Relación cerrada:

| Causa primaria | Estado de terminal normal |
| --- | --- |
| `SALIDA` | real `0`, `64`, `65` o `79` |
| `CANCELADO`, `PLAZO`, `PROTOCOLO`, `INCIDENTE` | `65` |
| `SENAL_INT` | `130` |
| `SENAL_TERM` | `143` |

La codificación completa debe medir menos o igual a 1024 bytes incluido LF,
validarse con el codec existente y conservar tamaño/offset inicial cero. No se
admiten guiones, ceros inventados, causa exterior, texto libre ni mezcla de
identidades.

## Incidente de cierre y cuarentena

El latch `INCIDENTE_DE_CIERRE` es independiente de la causa primaria. Puede
llegar ya enclavado desde O4a o fijarse una sola vez en O4c por estado Wait no
admisible, postcondición incompleta o fallo controlable de emisión/cierre. No
se limpia, sustituye ni usa para codificar otra causa.

Un TERMINAL normal solo se emite si el latch era y sigue vacío y todas las
mediciones `1|1|0|1` están acreditadas. Si el latch está enclavado antes de
iniciar la emisión, TERMINAL permanece sin trama normal; se conservan causa y
raws en el resultado privado, el estado exterior es 65 y O5 deberá
cuarentenarlo. O4c nunca escribe `INCIDENTE|65` para ocultar o reemplazar una
causa histórica distinta.

Una emisión iniciada es una sola trama lógica prevalidada. Escrituras
parciales o EINTR solo pueden continuar el sufijo pendiente mediante permisos
lease nuevos, con máximo ocho EINTR y sin recomenzar desde offset cero. Otro
error deja el artefacto no normal, fija incidente y obliga cuarentena; jamás
trunca, sobreescribe, añade una segunda trama o declara éxito degradado.

## Orden único de cierre y liberación

La secuencia positiva completa es:

1. terminalidad primario/reserva antes de `finDrenaje`;
2. único `cmd.Wait`, handle opaco consumido y estado real medido;
3. Wait4 no bloqueante hasta `ECHILD` dentro del límite;
4. sonda cero del grupo por primario igual a `ESRCH`;
5. cierre del pidfd primario bajo permiso lease; campo `-1`;
6. cierre del pidfd reserva bajo otro permiso; campo `-1`;
7. cierre de CONTROL bajo permiso separado; campo nulo;
8. emisión lógica única y cierre de TERMINAL bajo permisos acotados; campo
   nulo, haya trama normal o artefacto de cuarentena;
9. inventario físico igual al snapshot sellado menos exactamente los cinco FD
   poseídos `{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}`;
   ningún FD nuevo o ajeno cambia;
10. liberación única del observador desde estado 2 y owner O4C→LIBERADO;
11. liberación única de la lease desde estado 3 como última capacidad y owner
    O4C→LIBERADO;
12. `Cmd`, `Process`, autoridad, sellos y agregado se ponen a cero; OC7.

Cada cierre se intenta una sola vez. EINTR o resultado incierto no autoriza
repetir `Close`. El inventario no “repara” discrepancias. Antes de iniciar la
trama se cotejan todos los estados lógicos finales cuya comprobación no exige
cerrar TERMINAL; así no se publica un recibo normal tras una corrupción ya
observable.

La lease es la última capacidad. No hay syscall, lectura, escritura, snapshot,
log o asignación falible después de liberarla. Fallo de liberación o owner
partido es OCF directo; no rollback ni reapertura.

## Resultado privado hacia O5

OC7 produce un resultado privado, autoidentificado y consumible una vez con:

- estado exterior canónico;
- causa primaria histórica en su enum opaco;
- bit terminal normal emitido;
- bit cuarentena, equivalente a incidente de cierre;
- confirmación cerrada de Bash esperado, adoptados cero, grupo ausente y
  custodia liberada.

No contiene PID, PGID, pidfd, nonce, ticket, paths, errores libres, tiempos,
actor, tenant o datos personales; no tiene getter ni serialización. Estado
exterior es el estado de la trama solo en terminal normal sin incidente. En
cualquier incidente es 65 y `cuarentena=true`.

O5a--O5d definirán por contrato separado el coproc, lectura del fichero,
validación del recibo y relación con `wait -f`. Este P0 no crea esa interfaz ni
declara que un resultado privado sustituya la validación Shell.

## Custodia por frontera

| Recurso o decisión | O4a | O4b | O4c |
| --- | --- | --- | --- |
| causa/precedencia | fija una vez | inmóvil | preserva |
| fin de caso/subplazos | crea una vez | consume prestado | consume drenaje prestado |
| STOP/TERM/CONT/KILL | autoriza | ejecuta | prohibido |
| terminalidad final | indicación opaca | evidencia no recolectora limitada | decide para cierre |
| `cmd.Wait` | prohibido | prohibido | exactamente uno |
| adoptados/ECHILD | prohibido | prohibido | drenaje único |
| grupo ESRCH | no cierre | no post-KILL | postcondición obligatoria |
| pidfd explícitos | custodia | préstamo funcional | cierre final |
| CONTROL | lectura causal | inmóvil | cierre final |
| TERMINAL | inmóvil | inmóvil | emisión/cierre únicos |
| observador/lease | owners O4A | préstamo, no owner | owners O4C y liberación |

## APIs y conductas prohibidas

O4c nunca:

- llama `pidfd_send_signal`, `kill`, `Process.Signal/Kill` ni señal funcional;
- crea pidfd, duplica FD, promueve reserva o usa fallback PID/PGID;
- llama `Start`, `CommandContext`, `Cmd.Cancel`, `waitid` o segundo Wait;
- decide CONTROL, observador, causa, precedencia, etapa o raw O4b;
- recrea deadlines con `Add`, duración exterior, timer, ticker o contexto;
- bloquea en Wait4 de un adoptado vivo o acepta PID cero como drenaje;
- interpreta retorno cero/EPERM como grupo ausente;
- escribe más de una trama TERMINAL, una causa sustitutiva o un desconocido;
- libera lease antes del observador o ejecuta efecto después de liberarla;
- añade global mutable, `init`, hook, callback, goroutine o canal;
- expone autoridad mediante getter, interfaz, log, error libre o serialización;
- usa SQL, HTTP, red, Docker, Orquesta, Firecracker, Jailer o producción.

## DAG obligatorio de minitareas O4c

Cada nodo depende del anterior con doble GO, publicación y CI 5/5:

| ID | Responsabilidad única | Write-set máximo | Cierre observable |
| --- | --- | --- | --- |
| `O4C-P1-AUTORIDAD` | OC0--OC1, consumo, owners y sellos; cero syscall. | Un Go productivo nuevo + prueba focal. | Anti-nulo/alias/clon/replay/carrera y autoridad exacta. |
| `O4C-P2-TERMINAL-WAIT` | Terminalidad por pidfd y Wait único; captura de estado real. | Un Go productivo nuevo + prueba; P1 inmóvil. | Borde estricto, primario/reserva concordantes, Wait cardinal 1. |
| `O4C-P3-DRENAJE` | Wait4 WNOHANG hasta ECHILD y sonda ESRCH. | Un Go productivo nuevo + prueba; previos inmóviles. | Adoptados cero y grupo ausente antes de cierres. |
| `O4C-P4-CIERRES-PREVIOS` | Cierre pidfd primario/reserva y CONTROL, cada uno bajo permiso propio. | Un Go productivo nuevo + prueba; previos inmóviles. | Tres cierres únicos tras ESRCH; TERMINAL intacto. |
| `O4C-P5-TERMINAL` | Codec, emisión lógica única y cierre de TERMINAL. | Un Go productivo nuevo + prueba; previos inmóviles. | Trama exacta o cuarentena, sin causa inventada. |
| `O4C-P6-LIBERACION` | Inventario, observador, lease y resultado OC7. | Un Go productivo nuevo + prueba; previos inmóviles. | Recursos cero, lease última, resultado one-shot. |
| `O4C-P7-CONDUCTOR` | Black-box, AST/tipos/DAG y catálogo mutante. | Solo herramientas/pruebas propias. | Normal/race, mutantes muertos, residuos cero. |
| `O4C-P8-EVIDENCIA` | Ledger, checkpoint, revisiones y CI. | Solo evidencia/documentos propios. | Doble GO, hashes, publicación y CI 5/5. |

Aristas internas:

```text
O4C-P0 -> P1 -> P2 -> P3 -> P4 -> P5 -> P6 -> P7 -> P8
```

Aristas entre fronteras:

```text
O4A-P0 + O4B-P0 -> O4C-P0
O4A-P4 -> O4B-P1 -> O4B-P2 -> O4B-P3 -> O4B-P4 -> O4B-P5
O4A-P4 + O4B-P5 + O4C-P0 -> O4A-P5
O4A-P5 -> O4C-P1
O4C-P8 -> O5a
```

No existe arista O4c→O4a/O4b ni edición compartida. O4C-P1 continúa bloqueado
hasta recibir O4A-P5 material acreditado. O4A-P4 vigente ya está integrado;
O4A-P5 sigue bloqueado por O4B-P5 material y por el cierre revisado de este
O4C-P0 reconciliado.

Cada P1--P6 añade como máximo un productivo y una focal, objetivo 300--500
líneas, parada 650 y tope duro 800. Necesitar getter, tocar una fuente O3/O4a/
O4b o mezclar dos responsabilidades exige detenerse y dividir.

## Matriz mínima de pruebas

| ID | Oráculo causal |
| --- | --- |
| OC01 | Agregado A8 se consume una vez; nulo/alias/clon/replay/carrera dejan un ganador. |
| OC02 | Owners O4C, estados 3/2, registro/generación/TID/pending y los cinco FD exactos. |
| OC03 | Causa, incidente, historial y deadlines se preservan byte/valor a valor. |
| OC04 | Límite elegido coincide con una sola rama; cero/civil/recreado/divergente es fatal. |
| OC05 | Primario/reserva concuerdan terminalidad; vivo repite acotado, borde igual no Wait. |
| OC06 | Terminalidad acreditada ejecuta un solo cmd.Wait y consume solo handle opaco. |
| OC07 | nil/ExitError/ProcessState acreditan recolección; solo SALIDA exige estado real 0/64/65/79. |
| OC08 | Wait4 WNOHANG recoge solo PID positivo y termina únicamente en ECHILD. |
| OC09 | PID cero, hijo vivo, error o borde de drenaje son fallo; nunca Wait4 bloqueante. |
| OC10 | ECHILD precede a sonda grupo; solo ESRCH acredita ausencia. |
| OC11 | pidfd primario/reserva cierran una vez y antes de afirmar `1|1|0|1`. |
| OC12 | Nonce, PID supervisor, S3, causa/estado e identidad forman la trama O1a exacta. |
| OC13 | SALIDA usa 0/64/65/79 real; demás causas conservan su estado canónico. |
| OC14 | Incidente previo/nuevo no sustituye causa ni emite terminal normal; exterior 65/cuarentena. |
| OC15 | Escritura parcial/EINTR continúa solo sufijo, máximo ocho; nunca segunda trama. |
| OC16 | Los cinco FD cierran o se consumen una vez; snapshot final elimina exactamente esos cinco. |
| OC17 | Observador se libera antes y lease es la última capacidad; owners acaban LIBERADOS. |
| OC18 | Resultado OC7 one-shot no expone PID/pidfd/nonce/ticket/error libre. |
| OC19 | O4c no señala, no cambia causa/plazo/etapa y no usa API O5/O6. |
| OC20 | Fallo de permiso/consolidación/ownership entra OCF sin efecto posterior. |
| OC21 | 100 normal + 100 race dejan delta cero FD/hijo/zombi/grupo/temporal. |
| OC22 | BF directos acreditan 65, EOF/no retorno, stdout/stderr cero y cuarentena. |

Fixtures crean procesos sintéticos en PGID aislado y limpieza exterior segura;
nunca usan procesos ajenos. Casos de borde incluyen terminalidad al límite,
KILL raw fallido, hijo adoptado vivo, ExitError no canónico, ESRCH/EPERM,
escritura parcial y liberación partida. No hay SKIP ni retry del caso fallido.

## Familias mutantes mínimas

Mutantes atómicos compilables, cada uno con oráculo causal propio:

- OC01 aceptar nulo/clon/replay o dos consumidores;
- OC02 omitir autoidentidad, owner, estado, registro, generación, TID, pending
  o uno de los cinco FD `{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}`;
- OC03 cambiar causa, incidente, historial, raw o deadline heredado;
- OC04 escoger dos límites, reconstruir, extender o aceptar marca cero/civil;
- OC05 promover reserva, aceptar discordancia o invertir borde temporal;
- OC06 omitir/duplicar Wait, usar waitid/Process.Wait o Wait antes de terminalidad;
- OC07 inventar estado SALIDA, rechazar señal con causa de extinción, aceptar error ajeno o ProcessState nulo;
- OC08 usar Wait4 bloqueante, aceptar PID cero o omitir ECHILD;
- OC09 drenar después del borde, reintentar error o recoger líder dos veces;
- OC10 sondear antes de ECHILD, aceptar cero/EPERM o usar fallback PID/PGID;
- OC11 escribir antes de cerrar pidfd o dejar una referencia explícita;
- OC12 cambiar nonce/PID/fase/identidad/indicador/límite 1024 o parser;
- OC13 sustituir causa, mapear estado real adverso o fabricar SALIDA;
- OC14 emitir terminal normal con incidente o no forzar cuarentena/65;
- OC15 reiniciar offset, truncar, emitir dos tramas o reintentar sin límite;
- OC16 omitir, duplicar o reordenar uno de los cinco consumos/cierres, contar
  seis recursos o modificar un FD ajeno;
- OC17 liberar lease primero, omitir owner o usar capacidad después;
- OC18 añadir getter, PID/pidfd/nonce/ticket/error libre/serialización;
- OC19 añadir señal, CONTROL causal, etapa, timer, goroutine, canal o API O5;
- OC20 rollback, log, cierre o E/S después de autoridad incierta/fatal;
- OC21 falsear inventarios/residuos, aceptar SKIP/retry o evidencia reciclada;
- OC22 añadir arista inversa, editar O3/O4a/O4b o compartir write-set;
- OC23 cambiar límites 1024/8/180/1/2/1/5, parada 650 o tope 800.

Cada mutante pasa gofmt, compilación y vet antes del oráculo. Timeout, SHA
global o no compilación no cuentan como muerte. AST/tipos/DAG prueba estados,
owners, orden Wait/ECHILD/ESRCH/cierres/liberación, cardinalidad de syscalls,
codec, causa inmutable, deadlines y ausencia de APIs prohibidas.

## Conductor, puertas y evidencia

O4C-P7 será ejecutable desde checkout limpio y ligará por SHA-256 base,
toolchain, seis fuentes, focales, conductor, AST, catálogo y resultados.
Ejecutará normal/race, 100+100 capturas y BF directos. Acreditará delta cero de
FD, hijos, zombis, grupos y temporales, `ECHILD`, PGID ausente por ESRCH y
ningún proceso ajeno.

Puertas por corte material: focal normal/race/repetida, gofmt, vet,
AST/tipos/DAG, todos los mutantes, `go test ./...`, `go test -race ./...`,
`go vet ./...`, calidad, Gitleaks, enlaces, hashes, `git diff --check`, doble
revisión, publicación normal y CI 5/5.

Este P0 solo documental exige hashes de autoridades, enlaces locales,
write-set, formato, secretos, Gitleaks y `git diff --check`. No existe paquete
Go afectado sobre el que una focal normal/race aporte evidencia material.

## Seguridad, datos y presentación

O4c trata solo capacidades técnicas privadas. PID, pidfd, nonce e identidad
de proceso no salen de la trama técnica ya gobernada y jamás entran en logs,
errores libres o APIs. No hay identidad humana, tenant, dato RRHH, PII, SQL,
HTTP, interfaz visible, texto de producto, i18n o accesibilidad nuevos.

La indisponibilidad no es terminalidad, ausencia, Wait, limpieza ni éxito. Una
duda nunca habilita señal, fallback o caso siguiente.

## Paradas duras

Se detiene si O4c señala; cambia causa/plazo/etapa; llama Wait sin terminalidad
o más de una vez; omite ECHILD/ESRCH; acepta hijo vivo o retorno incierto;
escribe un terminal normal con incidente; fabrica S4/estado/identidad; libera
lease antes del observador; usa capacidad después; crea API pública; toca O3,
O4a u O4b; supera write-set/límites; sobrevive un mutante; queda residuo; falta
doble GO o CI 5/5.

## Métricas y condición de cierre

Este corte documental no modifica métricas ni estados transversales. Los
valores numéricos consignados en la candidata fuente eran un corte histórico:
no se trasladan como valores actuales ni se recalculan aquí; esa autoridad
permanece en dirección.

El candidato reconciliado necesita dos revisiones nuevas e independientes,
funcional y de seguridad, completas sobre los mismos bytes y el nuevo hash,
ambas `GO` con `P0=P1=P2=0`, además de corrección de cualquier NO-GO, commit
documental pequeño, publicación normal y CI 5/5. Los GO funcional
`bebeff77d92ddfd8bec926c7a473557039b70ce2` y de seguridad
`3839c8e9d330ca4584f91a399478cb15834e22c9` acreditaron únicamente
`de4ee5a11c3611e56ab09899da860ef88647ea0c` y no se trasladan. El productor
no revisa, aprueba, integra ni cambia estado transversal.

Después del cierre independiente, este contrato solo satisface una de las
dependencias de O4A-P5. `O4C-P1-AUTORIDAD` continúa bloqueado hasta O4A-P5
material acreditado. La siguiente acción de este candidato es revisión
funcional y de seguridad independiente; no queda autoasignada.

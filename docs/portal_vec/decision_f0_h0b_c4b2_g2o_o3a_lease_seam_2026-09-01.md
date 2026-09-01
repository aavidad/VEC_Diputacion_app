# Decisión O3A-LEASE-CLOSED-OPS-R2: operaciones físicas cerradas

Fecha: 1 de septiembre de 2026.

Estado: **CANDIDATA DOCUMENTAL R2 A DOBLE REVISIÓN**. Este documento no se
autoaprueba. Sustituye por completo R1 en
`6c22222796bb2d11d3dba86f8e8dbcfa3492de7c` y la propuesta transferible de
`5a4fad03035f67b46c248cbbed64cfeda5ec71c7`. No autoriza código, pruebas
dinámicas, integración, publicación, CI, O4B-P2, producción ni despliegue.

## Capability, invariante y alcance

Capability: cada operación física bajo la lease O3a se ejecuta mediante una
función privada, síncrona, concreta y cerrada. La misma invocación reclama
primero el slot; solo después lee el estado compartido; emite `Gettid`,
consolida su raw, compara el TID y ejecuta el efecto literal. No existe una
acreditación TID que pueda sobrevivir, salir o separarse de esa invocación.

Invariante central:

```text
CAS slot nil→celda
→ lecturas y mutaciones compartidas
→ estado autorizado→2
→ GETTID_RAW
→ consolidación raw sin interpretar
→ comparación TID
→ efecto físico literal cerrado
→ consolidación de su raw
→ restauración con slot ocupado
→ cierre verde
→ retirada exacta del slot.
```

Ningún token, puntero, permiso, callback, closure, interfaz, hook o variable de
tipo función puede transportar acreditación TID. Un argumento concreto como
un FD, un buffer de bytes o `*exec.Cmd` es dato de la operación y no autoridad
TID. Tampoco puede adquirir esa autoridad por quedar guardado en una celda,
estructura, mapa, resultado o testigo.

El alcance comprende los consumidores vivos O3a, O3b, O3c y O4a. R2 fija
además la forma obligatoria de un futuro consumo O4b, pero **no autoriza
O4B-P2** ni anticipa su decisión.

## Base, genealogía y preflight reproducido

La base exacta de este corte es
`6c22222796bb2d11d3dba86f8e8dbcfa3492de7c`, con padre
`5a4fad03035f67b46c248cbbed64cfeda5ec71c7`, en la rama existente
`trabajo/o3a-lease-seam-p0-contrato-20260901`. Al abrir R2, `HEAD` coincidía
con esa base, el árbol estaba limpio y el único write-set autorizado era:

`docs/portal_vec/decision_f0_h0b_c4b2_g2o_o3a_lease_seam_2026-09-01.md`.

El fichero R1 tenía modo `0644`, 450 líneas, 21.253 bytes y SHA-256
`1000222fdcd6f563414ee96b9ae779c9b539befc2bcf72807150eab569d54fce`.
R2 conserva `0644` y debe quedar por debajo de 800 líneas. No se ha cambiado
ninguna fuente, test, herramienta, manifiesto, evidencia ni workflow.

El producto local exacto observado antes de editar fue la rama
`integracion/ct-producto-ligero-20260821` en
`640610a4f806b0848682bbe844ff9d672c2777a6`, limpia. Su base común con R1 fue
`58800913b32e22b0f77eb8d62900d95c452e98fa`. El `merge-tree` de esos tres
hashes no presentó marcador ni conflicto y su salida tuvo SHA-256
`05f3e95573c8c2617bb90c8c509dd3fe568d80f9e1f95f9cda18141026351332`.
La puerta final de R2 se repite contra el mismo producto exacto y el commit R2;
si la rama de producto deja de apuntar a `640610a…`, el resultado caduca y se
repite contra el nuevo hash antes de revisar o integrar.

Las fuentes afectables observadas son `0644`. Sus máximos actuales son 732
líneas en O3a, 428 de producción y 364 de test en O3b, 293 de producción y
310 de test en O3c, y 563 de producción y 572 de test en O4a. G6a tiene 559
líneas y SHA-256
`9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8`.
Toda fuente o test futuro conserva `0644`, todo script ejecutable `0755` y
cada fichero el tope duro de 800 líneas. Alcanzarlo obliga a extraer una
responsabilidad coherente, no a comprimir u ocultar código.

## Autoridades leídas y prevalencia expresa

R2 se coordinó tras leer completas las autoridades documentales de O3a,
incluidas P0, P1 y la enmienda CI; O3b; O3c; O4a, incluidas P1B y P1D; O4b;
y la enmienda O4ab de terminalidad y STOP. También se leyeron completas las
dos revisiones independientes `NO-GO` de R1, funcional y de seguridad.

Las autoridades nominales, todas bajo `docs/portal_vec/`, son:

- `decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md`;
- `decision_f0_h0b_c4b2_g2o_o3a_p0_margen_runner_2026-08-09.md`;
- `decision_f0_h0b_c4b2_g2o_o3a_p1_barrera_ticket_temprana_2026-08-09.md`;
- `enmienda_f0_h0b_c4b2_g2o_o3a_v5_autoridad_ci_sin_r_2026-08-09.md`;
- `decision_f0_h0b_c4b2_g2o_o3b_ticket_stop_identidad_2026-08-10.md`;
- `decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md`;
- `decision_f0_h0b_c4b2_g2o_o4a_causa_tiempo_2026-08-11.md`;
- `decision_f0_h0b_c4b2_g2o_o4a_p1b_sellos_raw_2026-08-11.md`;
- `decision_f0_h0b_c4b2_g2o_o4a_p1d_contrato_deadline_2026-08-11.md`;
- `decision_f0_h0b_c4b2_g2o_o4b_senales_grupo_2026-08-12.md`;
- `enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md`.

Esta R2 enmienda expresamente, solo ante contradicción, las autoridades:

1. **O3a:** sustituye la API permiso→efecto→consolidación por funciones
   físicas cerradas y adelanta el CAS del slot a toda lectura compartida;
2. **O3b:** elimina su broker con callback, cierra cada observación y hace del
   primer ticket una operación compuesta indivisible;
3. **O3c:** elimina su broker con callback y hace de CONT una operación
   compuesta indivisible;
4. **O4a:** sus observaciones vivas consumen operaciones concretas cerradas y
   no un callback acreditado;
5. **O4b:** una señal futura deberá usar una función literal por efecto y
   situar la acreditación TID antes de la última marca o deadline cuya regla
   exige que la señal sea el siguiente syscall.

La enmienda O4ab se conserva con esa misma corrección causal. El resto de las
reglas de identidad, roles de pidfd, causa, precedencia, tiempo, cardinalidad,
terminalidad, no recolección, fallos, salidas y códigos sigue vigente.

R2 no promete compatibilidad byte-inmutable. Las firmas, tipos y llamadores
que permitan transferir la acreditación deben cambiar o desaparecer. La
compatibilidad funcional admisible se limita a los resultados y garantías
que no contradigan esta decisión.

## Hallazgos R1 que R2 cierra contractualmente

Las revisiones R1 identificaron un P0 común: la API heredada todavía devolvía
un permiso y una celda transferibles. Otra goroutine podía ejecutar el efecto
antes de que una consolidación posterior detectara un TID distinto. Ocultar
el efecto en callback o closure no cerraba la separación.

La revisión funcional añadió un P0 de orden: insertar `Gettid` entre la última
marca/deadline O4a/O4b/O4ab y la señal contradecía la exigencia de que la señal
fuera el siguiente syscall. R2 mueve la acreditación dentro de la misma
función cerrada, pero antes de esa última marca.

Los hallazgos secundarios fueron:

- el manifiesto C01-C22 propuesto era decorativo porque el conductor seguía
  seleccionando runners mediante una lista codificada;
- la evidencia pretendía contener la huella del commit que la contenía, una
  autorreferencia imposible;
- `merge-tree` no fijaba rama y SHA de producto reproducibles.

R2 convierte esos tres puntos en puertas explícitas más adelante.

## P0 obligatorio: retirada total de superficies transferibles

Al finalizar C6 deben haber desaparecido del código de producción, tests,
analizadores, mutantes, runners y manifiestos vivos estos símbolos:

- `permisoGuardiaO3aM38`;
- `comenzar` y `comenzarCritico` como API de la lease;
- `permisoValido`;
- `consolidarFisico` y `consolidarCritico`;
- `fatalPendiente`;
- `operarConLeaseBarreraO3bM38`;
- `syscallLeaseO3cM38`.

No se renombran, aliasan ni envuelven. Tampoco se reemplazan por otra API que
devuelva una celda, un ticket técnico, una función, una interfaz o un objeto
capaz de habilitar un efecto tras salir de la función acreditadora. Se permite
que una prueba histórica contenga el texto dentro de una evidencia histórica
inmutable; ningún artefacto vivo puede compilar, buscar o aceptar ese patrón.

La celda interna del slot no es una acreditación: solo registra el único ciclo
en curso. No se devuelve, no se pasa entre funciones, no se captura y ninguna
decisión de ejecutar se basa en que alguien posea su puntero. Su identidad
sirve exclusivamente para el CAS de retirada dentro de la misma función
privada que la creó.

## Mapa de consumidores vivos que deben migrar

La migración no puede declararse completa atendiendo solo a O4b. El inventario
vivo es:

### O3a

- cierres de descriptor y `os.File.Close` protegidos por snapshot físico;
- duplicaciones `F_DUPFD_CLOEXEC` de archivos y pidfd;
- apertura literal de `/dev/null` y creación `Pipe2` con flags cerrados;
- `(*exec.Cmd).Start` y obtención/reserva del pidfd resultante;
- `(*exec.Cmd).Wait`, cierre del pidfd y observaciones pidfd asociadas;
- pruebas normales y adversas que hoy crean o consolidan permisos.

Los consumidores están en G6a, `..._arranque_preparacion.go`,
`..._arranque_inicio.go`, `..._arranque_pruebas.go` y
`..._arranque_pruebas_adversas.go`.

### O3b

- el broker `operarConLeaseBarreraO3bM38` y sus usos para `fcntl`, `fstat`,
  `poll`, `read`, `open`, `close`, identidad, CONTROL y observaciones;
- la reserva y primera escritura del ticket, y las escrituras posteriores;
- auto-STOP, sondas pidfd y retirada/handoff;
- las pruebas de autoridad, barrera, ticket, STOP, identidad y handoff.

El primer ticket merece una operación compuesta cerrada propia; no puede ser
una reserva de permiso seguida por `Write` en otra función.

### O3c

- el broker `syscallLeaseO3cM38`, hoy delegado además en el broker O3b;
- revalidaciones de TID, PPID, pdeathsig, inventario e identidad;
- CONT inicial, primera observación, retirada y handoff con `Wait4`/pidfd;
- sus cinco fuentes productivas y cinco pruebas focales.

CONT merece una operación compuesta cerrada propia; la revalidación no puede
devolver un permiso consumible por `intentarContO3cM38`.

### O4a y frontera O4b

- `observarSenalArbitrajeO4aM38`, `pidfdFiableArbitrajeO4aM38` y
  `pollPidfdArbitrajeO4aM38` consumen hoy el broker O3b;
- las pruebas de autoridad, semilla, arbitraje y etapas deben probar el nuevo
  orden sin crear una segunda autoridad física;
- las señales futuras O4b —STOP, TERM, CONT y KILL de grupo— quedan sujetas a
  esta forma cerrada, pero su implementación sigue bloqueada por O4B-P2.

## Modelo cerrado de una operación simple

No habrá un ejecutor genérico que reciba operación, syscall, señal o flags.
Cada efecto tiene una función privada distinta y una llamada literal. Un enum
puede clasificar evidencia ya consolidada, pero no puede seleccionar un
syscall, señal, flags, siginfo, ruta, cardinalidad ni objetivo libres.

Ejemplos de familias cerradas, no firmas públicas, son cerrar un FD exacto,
duplicar con `F_DUPFD_CLOEXEC`, abrir `/dev/null` con flags literales, crear un
pipe con flags literales, iniciar un `*exec.Cmd`, esperar ese `*exec.Cmd`,
observar un pidfd y emitir cada una de las cuatro señales mediante una función
distinta. Si dos efectos tienen distinta semántica o flags, son dos funciones.

La secuencia obligatoria de cada función es:

1. construir solo locales no compartidos y una celda privada;
2. ejecutar como primera operación compartida
   `slot.CompareAndSwap(nil, celda)`; no se permite un `Load` previo;
3. si pierde el CAS, devolver denegación sin leer ni mutar registro,
   generación, secuencia, estado, snapshot o mapas;
4. si gana, leer y validar, en ese orden cerrado, lease, registro,
   generación, secuencia, estado, snapshot, mapas y argumentos concretos;
5. reservar la secuencia y transicionar el estado admitido a 2 con el slot
   todavía ocupado;
6. marcar `GETTID_RAW`, ejecutar un único `syscall.Gettid`, copiar el raw y
   marcar su consolidación sin compararlo ni interpretarlo;
7. comparar después el raw positivo con el TID sellado de lease y registro;
8. emitir el efecto literal exacto en esa misma invocación y goroutine;
9. copiar y consolidar su resultado raw antes de clasificarlo; cualquier
   resultado que la operación cerrada clasifique como fallo deja fatal+5;
10. consolidar snapshot/mapa/resultado y restaurar el estado de origen con el
    slot todavía ocupado;
11. declarar cierre verde y solo entonces retirar por CAS la misma celda;
12. devolver únicamente datos funcionales, nunca autoridad TID.

El proceso llamador ya debe estar en la goroutine bloqueada al OS thread que
gobierna O3a. La función no crea goroutines, no desbloquea el thread y no
difiere el efecto. El bloqueo por sí solo no acredita nada: la comparación
raw anterior sigue siendo obligatoria.

## Slot, estado 2 y fallo fatal

El CAS del slot precede toda lectura o mutación compartida, incluidas las que
parezcan de prevalidación. La única preparación anterior es local y no puede
consultar la lease ni sus punteros. Después de ganar el slot, cualquier fallo
—estado inesperado, nil interno, registro/generación divergentes, overflow,
secuencia, snapshot/mapa, argumento, TID, raw no consolidable, transición o
retirada— fija la celda fatal y el estado 5.

No existe rollback ordinario después del CAS ganado. Una celda fatal no se
retira, no se recicla y no habilita un segundo intento. Error de sistema del
efecto, incluido `EINTR`, se consolida como raw de un único intento y termina
en fatal+5; no sale como error ordinario ni restaura el estado. Una observación
solo puede cerrar verde con los discriminantes raw que su contrato concreto
declare válidos; indisponibilidad o duda nunca se reclasifican como éxito. No
hay retry, fallback, segundo pidfd ni segundo efecto.

La relación es obligatoria en toda observación concurrente:

```text
estado == 2  ⇒  slot != nil
```

El slot puede estar ocupado brevemente antes de leer el estado y después de
restaurarlo; nunca al revés. La restauración de estado, snapshot y mapa ocurre
con slot ocupado. La retirada solo ocurre tras cierre verde. Una operación
que requiera más de un efecto literal debe ser una composición cerrada
expresamente autorizada, no una lista, callback o enum libre.

## Operación compuesta del primer ticket O3b

La operación privada del primer ticket integra, sin devolver permiso:

1. CAS del slot y validaciones compartidas;
2. `Gettid` raw, consolidación y comparación;
3. la ronda final cerrada O3b de identidad, inventario y CONTROL, con orden y
   syscalls literales fijados en la función;
4. transición a barrera verde;
5. `syscall.Write` del primer fragmento del ticket como **primer syscall
   posterior** a esa barrera verde;
6. consolidación del raw, restauración con slot ocupado, cierre y retirada.

La acreditación TID ocurre antes de la ronda final, pero permanece válida por
causalidad local: la función es síncrona, no cambia de goroutine ni libera el
OS thread. Entre barrera verde y el primer `Write` no puede aparecer `Gettid`,
reloj, sonda, log, cierre ni otro syscall. El buffer y FD concretos no portan
autoridad TID.

Una escritura parcial queda consolidada como primer intento. Los fragmentos
restantes usan otra función cerrada concreta, sin reetiquetarse como primer
ticket. No hay replay automático ni reanudación tras `EINTR`.

## Operación compuesta CONT O3c

La revalidación ya no entrega permiso a otra función. Una única función
privada integra:

1. CAS del slot y validaciones O3c/O4a compartidas;
2. `Gettid` raw, consolidación y comparación;
3. las rondas cerradas de identidad, inventario y señales;
4. la lectura monotónica final `ahoraCaso` y el cálculo puro de
   `finCaso=ahoraCaso+180s`;
5. el `pidfd_send_signal(SIGCONT)` literal como **primer syscall posterior** a
   la marca monotónica;
6. consolidación raw, restauración con slot ocupado, cierre y retirada.

No hay `Gettid`, `time.Now`, sonda, log ni otra llamada al kernel entre la
marca final y CONT. El pidfd primario es argumento concreto ya gobernado por
O3c; no acredita TID ni permite elegir señal o flags.

La misma regla causal se aplica al futuro O4b: la función cerrada acredita TID
antes de la última comprobación de deadline o marca que su contrato considere
verde; desde esa comprobación hasta STOP, TERM, CONT o KILL, la señal literal
es el siguiente syscall. Hay cuatro funciones, no una señal libre.

## Observaciones O3b, O3c y O4a

Cada observación física vive en una función concreta: `fcntl`, `fstat`,
`poll`, lectura CONTROL, pdeathsig, TID/PPID, identidad pidfd, presencia,
`Wait4`, apertura/lectura/cierre de proc y observador de señal. La función
reclama su slot, acredita TID y consolida su raw antes de devolver la
observación minimizada.

O4a sigue decidiendo causa, precedencia, etapa y tiempo. Puede consumir los
datos devueltos, pero no un permiso físico, callback o puntero. O3a/O3b/O3c
no interpretan esos datos como autorización funcional. Las observaciones que
formen una unidad atómica se expresan como una secuencia cerrada concreta;
nunca como `func() error`, interfaz ejecutora o slice de operaciones.

## DAG obligatorio y granularidad

El DAG de autoridad es exactamente:

```text
documento R2 con doble GO
→ integración documental
→ C1 núcleo slot/operaciones O3a
→ C2 migración O3a
→ C3 O3b/primer ticket
→ C4 O3c/CONT
→ C5 O4a
→ C6 eliminación de API transferible
→ analizadores/mutantes/ledgers V26
→ conductor C22
→ doble revisión de código
→ integración
→ CI 5/5
→ decisión separada O4B-P2.
```

Ninguna flecha se adelanta. Los nodos C2-C5 pueden usar subcortes internos
secuenciales para respetar tamaño y compilación, pero el nodo siguiente solo
empieza cuando todos los subcortes del anterior están verdes e integrados en
la rama candidata. Cada commit tiene una responsabilidad observable y
compila con sus pruebas focales; no se produce un commit monolítico.

## Prefijo y write-sets máximos de código

En los listados siguientes, `S/` significa
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/`. Los globs son
fronteras máximas declaradas, no permiso para tocar cada fichero sin necesidad.
Toda ruta cambiada debe justificarse en el commit del subcorte y no se mezclan
dos nodos en un mismo commit.

### C1 — núcleo slot y operaciones O3a

Write-set máximo:

- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`;
- nueva `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas.go`;
- nueva `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_test.go`.

C1 publica el slot, las fases y operaciones simples cerradas sin retirar aún
la API anterior. La prueba focal demuestra CAS primero, raw→consolidación→
comparación→efecto y fatal+5. Si G6a no cabe, la nueva fuente posee la mecánica
cerrada y G6a conserva solo el campo mínimo; no se exceden 800 líneas.

### C2 — migración O3a

C2a migra close/dup/open/pipe. C2b migra Start/pidfd/Wait. Write-set agregado
máximo, repartido en commits distintos:

- las tres rutas C1;
- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`;
- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`;
- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go`;
- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go`.

Cada subcorte mantiene compilable el conjunto. C2 no borra aún la API mientras
quede un consumidor posterior.

### C3 — O3b y primer ticket

C3a migra observaciones/barrera; C3b implementa el primer ticket compuesto;
C3c migra STOP y handoff. Write-set agregado máximo:

- nueva `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- los stems exactos `autoridad`, `barrera`, `ticket`, `stop`, `identidad` y
  `handoff` de `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_`,
  cada uno en su `.go` y `_test.go` correspondiente;
- G6a y `..._arranque_operaciones_cerradas.go`, solo si el contrato interno
  mínimo de slot exige un cambio secuencial.

Compartir `barrera.go` entre C3a y C3b obliga a secuencia, no a fusionar
responsabilidades.

### C4 — O3c y CONT

C4a migra revalidaciones; C4b implementa CONT compuesto; C4c migra observación
y handoff. Write-set agregado máximo:

- nueva `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- los stems exactos `autoridad`, `revalidacion`, `cont`, `observacion` y
  `handoff` de `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_`,
  cada uno en su `.go` y `_test.go` correspondiente;
- G6a y la fuente cerrada O3a, solo por el contrato interno compartido.

### C5 — observaciones O4a

C5a migra arbitraje; C5b ajusta regresiones de autoridad/semilla/etapas.
Write-set agregado máximo:

- nueva `S/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observaciones_cerradas.go` y su `_test.go`;
- los stems exactos `autoridad`, `semilla`, `arbitraje` y `etapas` de
  `S/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_`, cada uno
  en su `.go` y `_test.go` correspondiente;
- las fuentes cerradas O3a/O3b estrictamente necesarias para componer la
  observación, en commit secuencial y sin añadir política O4a allí.

C5 no toca fuentes productivas O4b ni implementa señales de O4B-P2.

### C6 — eliminación completa de la API

Write-set máximo:

- G6a y las dos pruebas O3a de arranque;
- las fuentes y tests O3a/O3b/O3c/O4a anteriores únicamente si conservan una
  referencia residual que debía haberse migrado en su nodo.

C6 retira tipos, métodos, brokers y compatibilidad. Su criterio es cero
definiciones y cero referencias vivas de los nueve símbolos prohibidos, más
compilación y pruebas focales. Una referencia productiva descubierta aquí
obliga a corregir el nodo propietario; no se improvisa un adaptador.

## Write-set de analizadores, mutantes y V26

Este nodo se divide en material de herramientas y actas V26. Su frontera
máxima contabilizada comprende:

- `tools/o3a_v5_ast/**` completo;
- `tools/o3b_p7_ast/{README.md,main.go,main_test.go}`;
- `tools/o3b_p7_mutantes/**`;
- `tools/o3b_p7_mutantes_v3a/**` y `tools/o3b_p7_mutantes_v3b/**`;
- `tools/o3c_p6_ast/{README.md,invariantes.go,main.go,main_test.go,retirada.go,seguridad.go}`;
- `tools/o3c_p6_mutantes/{README.md,fusion.go,main.go,main_test.go}`;
- los ficheros vivos `README.md`, `conductor.sh`, `casos.tsv` y `fuentes.tsv`
  de `tools/o3b_p7_conductor/` y `tools/o3c_p6_conductor/`;
- nuevos `tools/o3b_p7_conductor/manifiesto_v26.tsv` y
  `tools/o3c_p6_conductor/manifiesto_v26.tsv`;
- nuevas evidencias V26 bajo directorios nuevos, nunca sobre actas existentes.

La cobertura `tools/o3a_v5_ast/**` es completa: README, aplicador, validador,
main, catálogos, manifests, receta, scripts y evidencia se inventarían y
hashearían. Los ledgers V21 y V25 son historia inmutable; deben entrar como
entradas verificadas y no tener diff. La nueva autoridad es V26, con nuevos
ledgers, manifests, fuentes, GOROOT, Go tool y herramientas. Igual regla rige
para las evidencias históricas O3b/O3c: se verifican, no se reescriben.

Los AST deben probar, por cada operación, dominancia del CAS del slot sobre
toda lectura compartida; `Gettid` raw antes de comparación; efecto literal en
la misma función; ausencia de callback/closure/interfaz/func variable; y
restauración antes de retirada. Los mutantes deben matar, como mínimo, CAS
tardío, lectura previa, comparación previa a consolidación, efecto movido a
otra función, señal/flags libres, retorno de celda, restauración sin slot,
retirada temprana, retry y supervivencia de cualquiera de los nueve símbolos.

Los `fuentes.tsv` de O3a/O3b/O3c y toda huella embebida en AST o mutantes se
actualizan al material real. Ningún artefacto vivo puede conservar
`9015dff…` como huella vigente de G6a; las evidencias V21/V25 sí la conservan
como genealogía y no acreditan V26.

## C22 como autoridad viva

El write-set máximo del nodo C22 es:

- `tools/o3a_v5_conductor/README.md`;
- `tools/o3a_v5_conductor/conductor.sh`;
- los runners existentes `conductor_c*.sh` cuando deban publicar su SHA real;
- nuevo `tools/o3a_v5_conductor/conductor_c22_lease_closed_ops.sh`;
- `tools/o3a_v5_conductor/fuentes_v5.tsv`;
- baja de `tools/o3a_v5_conductor/manifiesto_c01_c21.tsv`;
- alta de `tools/o3a_v5_conductor/manifiesto_c01_c22.tsv`;
- `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_test.go`;
- nuevo `tools/o3a_v5_conductor/evidencia-cnd-c22-lease-closed-ops-r1/`,
  exclusivamente en el commit de acta posterior al material C22.

`manifiesto_c01_c22.tsv` es autoridad viva, no inventario decorativo. Contiene
exactamente C01-C22 y, por caso, ruta relativa normalizada del runner, SHA-256
del runner, ruta del manifiesto de fuentes y su SHA-256. El mismo runner puede
agrupar varios casos solo con ruta y SHA idénticos; cada ID de caso aparece
exactamente una vez.

`conductor.sh` no conserva una lista paralela de runners. Lee el manifiesto,
rechaza cardinalidad distinta de 22, IDs fuera de C01-C22, huecos, IDs
duplicados, rutas absolutas o escapadas, SHA inválido, rutas inexistentes,
modos incorrectos y una misma ruta con huellas contradictorias. De ahí deriva
la lista única de runners, verifica sus SHA y ejecuta cada uno en normal y
`-race`. Un runner omitido, hardcoded fuera del manifiesto o marcado `SKIP`
es `NO-GO`.

La evidencia liga manifiesto, fuentes, runners, modos, resultados, residuos y
commit material. El conductor falla si no puede enlazarlos. C22 prueba además
los bordes concurrentes `estado==2 ⇒ slot!=nil`, transferencia adversa entre
goroutines con cero efecto, ticket primero, marca→CONT, las operaciones
O3a/O3b/O3c/O4a y ausencia total de la API retirada.

## Separación material/acta y ausencia de autorreferencia

Los commits de C1-C6, herramientas V26 y C22 son commits materiales pequeños.
Después de fijar el último hash material se ejecutan las puertas autorizadas
en el futuro corte y se crea un commit de acta probatoria separado. El acta
registra el hash de su padre material, rutas y SHA de entradas/salidas y
resultados; nunca promete contener su propio hash ni la huella de un árbol que
la incluye.

La doble revisión de código recibe el hash exacto del commit de acta y comprueba
su padre material. El hash del acta se conserva en las revisiones externas o
en la entrega de Dirección, no dentro del acta misma. Una corrección material
invalida el acta y exige regenerarla. V21 y V25 solo son evidencia histórica;
la evidencia nueva se denomina V26.

## Matriz y puertas del futuro código

Cada precorte ejecutará `gofmt`, prueba focal normal y `-race`, `go vet` y
`git diff --check`. Antes de revisión se añaden los conductores O3a/O3b/O3c,
AST, mutantes, calidad global y las puertas proporcionadas por el contrato
publicado, sin sustituirlas por mocks de syscall.

La matriz mínima incluye:

- CAS perdido sin ninguna lectura compartida ni efecto;
- cada fallo posterior al CAS como celda fatal+estado 5;
- observación concurrente de todos los bordes sin estado 2/slot nulo;
- TID de goroutine correcta e incorrecta, siempre raw→consolidar→comparar;
- FD, buffer y `*exec.Cmd` adversos sin convertirlos en autoridad;
- close, dup, open, pipe, Start, Wait y pidfd con una función literal cada uno;
- primer `Write` como siguiente syscall tras barrera verde;
- CONT como siguiente syscall tras la marca monotónica;
- broker, callback, closure, interfaz, hook y función variable ausentes;
- enum incapaz de inyectar syscall, señal o flags;
- raw de éxito con cierre verde; error/`EINTR` con un intento, fatal+5 y cero
  retry/fallback;
- restauración completa con slot ocupado y retirada solo tras verde;
- cero referencias vivas a la API retirada;
- C01-C22 cardinales, sin huecos ni duplicados, normal y race derivados del
  manifiesto y ligados a V26.

Esta R2 no ejecuta esas dinámicas ni acredita su resultado.

## Seguridad, privacidad, i18n y accesibilidad

La decisión reduce autoridad transferible y aplica denegación predeterminada.
No añade identidad humana, dato personal, secreto, credencial, token externo,
red, SQL, Docker, HTTP, persistencia ni log. Los raws y TID se mantienen
locales y minimizados; no se imprimen ni entran en interfaces.

No hay texto visible ni superficie de usuario, por lo que i18n y accesibilidad
no cambian. No se autoriza producción, datos reales, señal a procesos ajenos,
prueba dinámica en este corte ni despliegue.

## Condiciones de NO-GO y cierre documental

Es `NO-GO` cualquiera de estas condiciones:

- sobrevive o se sustituye nominalmente alguna API transferible;
- un efecto queda fuera de la función que acredita TID;
- slot CAS posterior a una lectura compartida;
- comparación TID previa a consolidar raw;
- estado 2 con slot nulo, restauración sin slot o retirada antes de verde;
- fallo posterior al CAS que no termina en fatal+5;
- callback, closure, interfaz, hook, func variable o enum libre;
- `Gettid` entre la última barrera/marca/deadline y primer ticket, CONT o
  señal futura;
- write-set no declarado, fichero mayor de 800 líneas o commit monolítico;
- V21/V25 reescritos o presentados como V26;
- acta autorreferencial;
- C22 no derivado exclusivamente del manifiesto o sin normal/race;
- producto distinto de `640610a…` sin repetir `merge-tree`;
- intento de usar R2 para autorizar O4B-P2.

El siguiente y único paso es la doble revisión documental independiente,
funcional y de seguridad, de los bytes exactos del commit R2. Debe emitir dos
`GO` con `P0=P1=P2=0` antes de que Dirección integre el documento y abra C1.
Este productor no revisa, integra, publica, actualiza métricas ni declara GO.

Las puertas permitidas de este corte documental son `git diff --check`,
Gitleaks focal exacto con `/tmp/vec-gitleaks-20260831`, modo `0700` y
SHA-256 `c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`,
y `merge-tree` contra
`integracion/ct-producto-ligero-20260821=640610a4f806b0848682bbe844ff9d672c2777a6`.
Siguen prohibidos `fetch`, `pull`, `reset`, `push`, deploy, dinámicas,
producción, credenciales y gates pesados.

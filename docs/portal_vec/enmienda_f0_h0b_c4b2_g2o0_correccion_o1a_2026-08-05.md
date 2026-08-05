# Corrección F0-H0b/C4b-2 G2-O0: contrato canónico y corte O1a

Fecha: 5 de agosto de 2026.

Estado: **quinta propuesta de dirección pendiente de doble revisión
independiente**. Sobre `6a27aab`, dos revisiones obtuvieron GO, pero una tercera
detectó un P1 de señal funcional en S1; por tanto no se autorizó código. Corrige
ese P1 y los anteriores documentados en la
[revisión consolidada](revisiones/revision_f0_h0b_c4b2_g2o0_correccion_o1a_2026-08-05.md).
No autoriza O1a, O1b ni ninguna fase posterior hasta nueva doble revisión.

Base de código exacta: `3e36ecae23e0608bc1e7b9ce374e8fb35d13b4a2`.
Antecedente documental: `bdfec1f`, que conserva el primer G2-O0 como NO-GO y
su triple revisión consolidada.

## Alcance reducido

Esta corrección fija el lenguaje de cable, la topología sin lector binario
concurrente y las reglas transversales. Su único corte programable tras el
doble GO es **O1a: codec de una trama completa y su autoprueba pura**.

O1a no abre FDs, no crea procesos, no instala señales, no llama a `prctl`, no
usa pidfd, no ejecuta Bash, Docker, PostgreSQL, SQL o red y no habilita
`--supervisar-m38`: el modo continúa devolviendo 64 sin efectos. El lector
incremental O1b y todas las fases posteriores necesitan corte, ledger y
revisión propios.

## Prevalencia exacta

Sobre la
[decisión C4b-2 aceptada](decision_f0_h0b_c4b2_captura_quinta_supervision_exterior_2026-08-04.md),
esta enmienda sustituye exclusivamente:

1. Go consume y valida primero la única trama `SOBRE` del FD 9, pero no usa el
   ticket ni crea Bash antes de un `ARMAR` e `INICIAR` válidos;
2. después de `INICIAR` el control permanece abierto y admite como máximo un
   `CANCELAR` terminal;
3. desaparecen los ACK vivos: `ACK_LISTO` y `ACK_CASO` no forman parte del
   nuevo protocolo ni de O1a;
4. FD 8 deja de ser pipe de recibo: es un fichero regular privado preabierto
   donde Go escribe un único terminal antes de salir;
5. después de recoger al supervisor, el runner puede crear y esperar, sin
   solapamiento, un segundo proceso del mismo binario privado en modo
   `--validar-recibo-m38`;
6. EOF limpio sin causa previa es `CANCELADO|65`; PDEATHSIG, señal directa al
   supervisor o cambio de PPID son `INCIDENTE|65`;
7. las precedencias pasan a ser las fijadas aquí.

La separación G2-S ya aceptada sustituyó el manifiesto de cinco por seis
fuentes capturadas y separó G1/G2; esta enmienda parte de ese hecho.

Permanecen vigentes las demás garantías de C4b-2, C4b-1 y G2-S:

- durante el caso la topología es runner → supervisor Go → Bash;
- D2d, D2c, H0b, capturador y adaptador M38 son invariantes en O1a;
- no hay fallback PID/PGID ni ejecutable resuelto por `PATH`;
- el runner decide en exclusiva cuarentena, reutilización y caso siguiente;
- el adaptador solo materializa preservación y devuelve evidencia;
- no se hereda gramática de G2-O ni del primer G2-O0, ambos rechazados.

## Topología cerrada del puente

Bash puro queda descartado para validar una entrada binaria adversa: sus
variables no conservan NUL y `read`, `read -N` o `mapfile` pueden aceptar por
alias una trama adulterada. Tampoco se admiten `od`, `dd`, Python o Perl como
autoridad adicional.

La topología exacta es secuencial:

```text
durante el caso:
runner Bash
└── supervisor Go privado --supervisar-m38
    └── Bash de caso

después de recoger al supervisor:
runner Bash
└── mismo Go privado --validar-recibo-m38
```

Supervisor y validador nunca coexisten. El segundo modo no crea Bash, no
interpreta política ni recibe por argumento nonce, identidad, selector o valor
esperado: solo valida sintaxis y reserializa una trama canónica.

El futuro puente operativo usará:

- control Shell → Go mediante pipe anónimo;
- antes de `INICIAR`, captura Shell acreditada por la conjunción: hijo directo
  único, ejecutable privado acreditado, duplicados estables de control y vida,
  FD 8/9 ligados a sus objetos acreditados y ausencia de otro trabajo. El nonce
  se valida dentro de Go antes de aceptar `INICIAR` y se reconcilia en el
  terminal final. Este predicado sustituye expresamente al anterior basado en
  `ACK_LISTO`;
- stdout del `coproc` solo como canal de vida, sin datos y sin herencia al Bash.
  Go no escribe, duplica ni cierra explícitamente su escritor; solo el cierre
  automático del kernel al terminar el supervisor puede producir EOF;
- FD 8 como fichero regular `0600`, un enlace, dentro del temporal `0700`,
  creado con exclusión. El runner abre por separado un descriptor lector con
  offset cero y otro escritor: nunca son `dup` de la misma descripción abierta;
  solo Go hereda el escritor y el runner lo cierra inmediatamente tras el fork.
  El Bash no hereda ninguno: Go marca su escritor `CLOEXEC` antes de `Start`.
  El validador recibe únicamente el lector retenido y no reabre la ruta;
- FD 9 como sobre monoframa preabierto;
- al recibir EOF del canal de vida se exige además que la tabla de trabajos
  marque al supervisor terminal e inmediatamente recolectable. Solo entonces
  se ejecuta `wait -f` una vez. Cierre anticipado, proceso aún vivo al vencer
  el subplazo o canal abierto al límite absoluto son frontera externa: nunca
  llevan a espera bloqueante y exigen evidencia, cuarentena y reprovisión;
- solo tras recoger al supervisor, invocación por ruta privada exacta de
  `--validar-recibo-m38`, con entorno `LC_ALL=C`, FD mínimos, sin red ni
  secretos. El runner mide el plazo contra `/proc/uptime`, observa salida y
  terminalidad sin un proceso reloj, y solo espera de forma síncrona cuando el
  trabajo es inmediatamente recolectable;
- salida del validador limitada a una reserialización ASCII canónica; el runner
  compara después nonce, PID, causa/estado y política. El modo devuelve 0 solo
  por trama válida, 64 por uso y 65 por entrada inválida o incidente;
- antes de ambas ejecuciones se reacreditan el mismo inode, SHA-256, EUID, modo
  `0700`, tipo regular y enlace único del binario privado; tras el validador se
  repite la acreditación. Cualquier cambio invalida su salida, fija incidente
  65, cuarentena y ningún caso siguiente.

Propietarios de espera:

- Go supervisor: único `cmd.Wait` del Bash y drenaje de adoptados;
- adaptador: único `wait -f` del supervisor, solo después del EOF de vida;
- Shell: espera síncrona del validador, solo después del supervisor.

O1a únicamente codifica la gramática reutilizable; no materializa esta
topología.

## Clases físicas

| Clase | Dirección/soporte | Máximo incluido LF | Cardinalidad |
| --- | --- | ---: | --- |
| Sobre | Shell → supervisor por FD 9 | 4096 bytes | una y después EOF |
| Control | Shell → supervisor por pipe | 1024 bytes por trama | según estado |
| Terminal | supervisor → fichero FD 8 | 1024 bytes | exactamente una en cierre controlado |
| Salida validada | validador → runner | 1024 bytes | una y después EOF |
| Ticket Bash | supervisor → Bash por FD 9 | 2060 bytes | una y después EOF |

Todas terminan en un único LF. Se rechazan NUL, CR, TAB y bytes fuera de ASCII
de siete bits en todas las clases. El ticket funcional admite solo bytes
imprimibles `0x20..0x7e`; puede contener `|`, pero no LF, CR, TAB o NUL.

El lector incremental O1b, aún no autorizado, deberá:

- comprobar el límite antes de ampliar el buffer;
- admitir fragmentación byte a byte y coalescencia;
- entregar una trama por transición conservando exactamente el sobrante;
- distinguir EOF en frontera, EOF parcial y exceso sin LF;
- clasificar trama parcial más EOF como `PROTOCOLO`, nunca `CANCELADO`;
- validar alfabeto y desbordamiento antes de convertir;
- no usar `bufio.Scanner` con límite implícito ni una lectura como mensaje.

O1a recibe una sola trama completa `[]byte`, incluido LF, y rechaza longitud,
terminador, byte o cardinalidad adversos sin conservar estado.

## Gramática canónica

Las barras y el LF son literales:

```text
V1|SOBRE|NONCE|PID_RUNNER|SELECTOR|IDENTIDAD|LONGITUD_TICKET|TICKET\n

V1|CONTROL|ARMAR|NONCE|PID_RUNNER\n
V1|CONTROL|INICIAR|NONCE\n
V1|CONTROL|CANCELAR|NONCE|CAUSA|ESTADO\n

V1|TERMINAL|NONCE|PID_SUPERVISOR|FASE_ORIGEN|ESTADO|CAUSA|PID_BASH|PPID|PGID|SID|INICIO|BASH_CREADO|BASH_ESPERADO|ADOPTADOS_PENDIENTES|GRUPO_AUSENTE\n

PID_SUPERVISOR|TICKET\n
```

`SOBRE` se divide por los siete primeros separadores; todo lo restante es el
ticket. El ticket Bash se divide solo por el primero. Las demás tramas exigen
cardinalidad exacta y ningún campo contiene separadores.

## Dominios y coherencia

| Campo | Dominio canónico |
| --- | --- |
| `NONCE`, `IDENTIDAD` | 64 hexadecimales minúsculos exactos |
| PID, PPID, PGID, SID | decimal mínimo `1..2147483647` |
| `INICIO` | decimal mínimo `1..18446744073709551615` |
| `LONGITUD_TICKET` | decimal mínimo `1..2048`, igual a los bytes del ticket |
| `SELECTOR` | `NOMINAL` o una mayúscula ASCII seguida de dos dígitos |
| `FASE_ORIGEN` | uno de `S1`, `S2`, `S3`, `S4` |
| Indicadores | un byte `0` o `1` |
| `ADOPTADOS_PENDIENTES` | decimal mínimo `0..2147483647` |
| Estado | `0`, `64`, `65`, `79`, `130` o `143` |

El selector es autoridad exclusiva del runner. Go solo valida su sintaxis y
lo transporta como argumento separado; no replica el catálogo ni autoriza un
caso.

Los decimales no admiten signo, espacio o cero inicial. Cero solo se escribe
como `0` y solo es válido en estado, indicadores y adoptados pendientes.

`CANCELAR` acepta exclusivamente:

| Causa | Estado |
| --- | ---: |
| `CANCELADO` | 65 |
| `PROTOCOLO` | 65 |
| `SENAL_INT` | 130 |
| `SENAL_TERM` | 143 |

El terminal acepta exclusivamente:

| Causa | Estado |
| --- | --- |
| `SALIDA` | estado Bash real `0`, `64`, `65` o `79` |
| `CANCELADO`, `PLAZO`, `PROTOCOLO`, `INCIDENTE` | `65` |
| `SENAL_INT` | `130` |
| `SENAL_TERM` | `143` |

Una terminación Bash distinta de `0/64/65/79`, incluida muerte por señal sin
causa enclavada, es `INCIDENTE|65`, nunca `SALIDA`.

`FASE_ORIGEN` es el estado activo cuando se enclavó `CAUSA_PRIMARIA`, justo
antes de entrar en S5. Nunca representa S5/S6. S0 tampoco aparece: sin sobre y
nonce confiables no se emite terminal normal.

Los cinco campos de proceso son un bloque:

- sin Bash: `PID_BASH|PPID|PGID|SID|INICIO` es `-|-|-|-|-` y la postcondición
  es `BASH_CREADO=0`, `BASH_ESPERADO=0`, `ADOPTADOS_PENDIENTES=0`,
  `GRUPO_AUSENTE=1`;
- con Bash: los cinco son decimales canónicos medidos y el único terminal
  normal exige `1|1|0|1`;
- no se admite mezcla, campo desconocido ni cero fabricado.

Coherencia cruzada obligatoria:

| Bloque | Fase de origen | Causa admisible |
| --- | --- | --- |
| Sin Bash `-|-|-|-|-` y `0|0|0|1` | S1 | `CANCELADO`, `PLAZO`, `PROTOCOLO` o `INCIDENTE` |
| Sin Bash `-|-|-|-|-` y `0|0|0|1` | S2 o S3 | cualquiera salvo `SALIDA` |
| Con Bash medido y `1|1|0|1` | S3 o S4 | cualquiera de la tabla terminal |

`SALIDA` exige siempre Bash medido, `1|1|0|1` y fase S3/S4. Cualquier bloque
con Bash exige S3/S4. Un bloque sin Bash nunca admite `SALIDA`. El codec O1a
rechaza las combinaciones cruzadas, no las pospone a la máquina operativa.

## Máquina y precedencia para fases futuras

Estados:

```text
S0 ESPERAR_SOBRE
S1 ESPERAR_ARMAR
S2 LISTO
S3 INICIANDO
S4 EJECUTANDO
S5 TERMINANDO
S6 TERMINAL
```

Solo la goroutine propietaria transforma acontecimientos. Los manejadores de
señal solo fijan un indicador y despiertan el ciclo. Hay dos enclavamientos:

- `CAUSA_PRIMARIA`: primera causa válida, inmutable;
- `INCIDENTE_DE_CIERRE`: fallo posterior de limpieza. Fuerza salida externa
  65 y cuarentena, conservando la causa primaria como evidencia.

Antes de cada acción irreversible se hace una vuelta no bloqueante. Orden:

1. causa primaria existente → S5, sin interpretar más control;
2. una trama completa ya disponible o error de framing; parcial más EOF,
   exceso, dominio o secuencia inválidos → `PROTOCOLO`;
3. señal directa/PDEATHSIG o PPID incoherente → `INCIDENTE`;
4. terminalidad pidfd → `SALIDA`, o `INCIDENTE` si el estado no es admisible;
5. deadline absoluto → `PLAZO`;
6. EOF limpio en frontera → `CANCELADO`;
7. sin evento, un solo subpaso mecánico acotado.

Desempates:

- `INICIAR` solo cambia a S3; `Start` no ocurre en esa vuelta;
- señal directa, PDEATHSIG o PPID incoherente en la vuelta siguiente impiden
  `Start`, aunque `INICIAR` esté almacenado;
- `INICIAR\nCANCELAR\n` conserva el sobrante y cancela antes de `Start`;
- después de `Start`, `CANCELAR` completo ya disponible gana a terminalidad;
- señal directa/PDEATHSIG gana a pidfd; pidfd gana al deadline; deadline gana
  a EOF limpio; parcial más EOF gana como `PROTOCOLO`;
- INT/TERM funcionales del runner solo llegan mediante `CANCELAR` canónico.

| Estado | Única entrada admisible y transición |
| --- | --- |
| S0 | `SOBRE` válido seguido de EOF → S1. Sin nonce confiable, fallo 65 sin terminal normal. |
| S1 | `ARMAR` coherente → S2. |
| S2 | `CANCELAR` → S5 sin Bash; `INICIAR` almacenado → S3. |
| S3 antes de `Start` | Solo `CANCELAR`; cualquier evento anterior impide crear Bash. |
| S3 tras `Start` | Solo `CANCELAR`; error lleva a S5 sin inventar fase superior. |
| S4 | Solo `CANCELAR`; salida, plazo, incidente o EOF llevan a S5. |
| S5 | No interpreta tramas; extingue y mide. Fallo fija incidente de cierre. |
| S6 | No admite entrada ni cambio de causa, estado o recibo. |

## Terminalidad y cuarentena

Un terminal normal solo se escribe cuando todos sus campos han sido medidos.
Un desconocido nunca se codifica como cero, guion parcial o postausencia.

| Camino | Número de `cmd.Wait` |
| --- | ---: |
| Antes de `Start` o `Start` fallido | 0 |
| `Start` exitoso y terminalidad pidfd acreditada antes de `fin_drenaje` | exactamente 1 |
| pidfd perdido o `fin_drenaje` agotado antes de terminalidad | 0; frontera externa |

Tras el único `cmd.Wait`, la misma goroutine drena hasta `ECHILD`, exige señal
cero de grupo igual a `ESRCH`, cierra ambas referencias pidfd y solo entonces
puede escribir `1|1|0|1`. Si algo no se puede medir, no hay terminal normal:
salida 65 si es posible, evidencia, preservación, cuarentena y ningún caso
siguiente.

## Ledger exclusivo de O1a

Medición exacta:

| Alias | Ruta | Líneas | SHA-256 abreviado |
| --- | --- | ---: | --- |
| R | `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` | 800 | `63491daa…249` |
| G1 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go` | 683 | `0a14142d…e5e` |
| G2 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go` | 91 | `b9890888…21c` |
| C | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/capturar_snapshot_fuente_corporativa_contexto_actor_v1.go` | 799 | `4a967fd1…902` |
| A | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh` | 527 | `98d22a30…8cb7` |
| D2d | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh` | 145 | `9b137f13…5e81` |
| D2c | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_fuente_corporativa_contexto_actor_v1.sh` | 588 | `a07057fb…dde5` |
| H0b | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh` | 580 | `02a00f2f…ded` |

Write-set O1a:

1. G2: tipos, encoder/decoder de trama completa, dominios y autopruebas;
2. G1: una llamada agregadora desde `autoprobar` y su error;
3. R: solo reemplazar los tres literales SHA de G1, G2 y binario supervisor,
   sin cambiar sus 800 líneas.

C, A, D2d, D2c y H0b quedan byte a byte invariantes. No se crea fichero.

| Unidad | Base | Delta conservador | Total previsto | Parada dura |
| --- | ---: | ---: | ---: | ---: |
| R | 800 | 0 líneas | 800 | 800 |
| G1 | 683 | +2..+6 | 685..689 | 690 |
| G2 | 91 | +140..+220 | 231..311 | 320 |
| C | 799 | 0 | 799 | 799 |
| A | 527 | 0 | 527 | 527 |
| D2d | 145 | 0 | 145 | 145 |
| D2c | 588 | 0 | 588 | 588 |
| H0b | 580 | 0 | 580 | 580 |

El delta G2 se descompone en `+80..+120` para codec productivo y
`+60..+100` para su autoprueba capturada. Permanecen en O1a porque comparten la
misma gramática/tipos y ningún codec queda admitido sin sus mutantes en el mismo
candidato; la suma conservadora continúa siendo `+140..+220` y la parada 320.

Antes de editar se registra `wc -l`, SHA y `git diff --numstat`. Se detiene si
hace falta otra ruta, minificar, encadenar controles, retirar pruebas o superar
un tope. El espacio no usado no se rellena ni se transfiere.

## Autoprueba O1a

Sin FD ni hijos, cubre:

- todas las tramas y ticket en ida/vuelta canónica;
- longitudes exactas de cada clase y un byte por debajo/encima;
- cardinalidad menor/mayor, campo vacío y versión/tipo/dirección adversos;
- cero, máximos, ceros iniciales y desbordamientos antes de convertir;
- hex minúsculo válido y mutantes de longitud, mayúscula y alfabeto;
- selector sintáctico válido/adverso sin replicar catálogo;
- ticket con `|`, longitud falsa, vacío, LF, CR, TAB, NUL y no ASCII;
- causas, estados, parejas, fase, indicadores y bloques de proceso adversos;
- `SALIDA` sin Bash o en S1/S2, Bash en S1/S2, bloque sin Bash con `SALIDA` y
  cualquier mezcla entre identificadores y postcondición;
- `S1|SENAL_INT|130` y `S1|SENAL_TERM|143`, porque antes de `ARMAR` una trama
  `CANCELAR` es secuencia inválida y solo puede terminar como `PROTOCOLO|65`;
- bytes posteriores al LF y ausencia de LF;
- modo `--supervisar-m38` todavía 64 sin variar FD o hijos.

Fragmentación, coalescencia, sobrante y EOF incremental pertenecen a O1b y no
se simulan en O1a.

## Roadmap no autorizado

```text
O1b  lector incremental, fragmentación, coalescencia y sobrante
O2a  sobre S0
O2b  ARMAR y cancelación sin Bash
O3a  Start y mapa FD
O3b  pidfd doble, barrera, STOP y /proc
O3c  ticket, plazo, CONT y salida natural
O4a  latch, precedencias y plazos
O4b  STOP, TERM, CONT y KILL
O4c  terminalidad, Wait, drenaje, ECHILD, ESRCH y terminal
O5a  coproc, canal de vida y fichero FD 8
O5b  modo secuencial --validar-recibo-m38
O5c  consumo Shell, evidencia y cuarentena
O5d  activar --supervisar-m38
O6a  mutantes controlados
O6b  fronteras externas
O6c  matriz integradora
```

Cada corte requiere rutas exactas, delta conservador, total, criterio único,
modo 64 cerrado, parada y revisión distinta del productor. Un NO-GO crea otra
minitarea; O6 nunca corrige producción mientras prueba.

## Puertas y aprobación

O1a exige `gofmt`, dos builds privados aislados con SHA idéntico, `go vet`,
autoprueba positiva, mutantes negativos, Bash/ShellCheck del runner,
invariantes SHA, `git diff --check`, Gitleaks y puertas globales.

Dos revisores distintos deben aprobar primero este documento. Tras el código,
otros dos revisan implementación y ledger. O1a no cambia F0 `10/23`, O4-05
`3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14` ni el NO-GO de
producción.

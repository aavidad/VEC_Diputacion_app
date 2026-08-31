# Enmienda F0-H0b/C4b-2/G2-O/O4a-O4b: terminalidad y STOP final

Fecha: 13 de agosto de 2026.

Identificador: `O4AB-P0-ENMIENDA-TERMINALIDAD-STOP`.

Estado: **CANDIDATA DOCUMENTAL V3 A REVISIÓN**. No corrige ni autoriza código,
no acredita `O4A-P4-ETAPAS`, no abre O4B-P1, O4A-P5, O4c, O5/O6,
integración, producción ni despliegue. Requiere doble revisión independiente
sobre los mismos bytes, publicación y CI 5/5 antes de un nuevo candidato
material O4A-P4.

## Motivo y hallazgos independientes

Las revisiones funcional y de seguridad del candidato exacto O4A-P4
`2b7eaf498f8f68a90b66c004f166a62f070b5064`, sobre la base
`5345d5d097b51ab3567983f048feabeceaf2957b`, emitieron `NO-GO`.

La revisión funcional, commit
`e20a5fe597d9c172d59b99e75878f8e99e196c76`, clasificó
`P0=1, P1=2, P2=0`. La revisión de seguridad, commit
`a62ee60f70315a1fc0d290290b681c9c3f35227b`, clasificó
`P0=0, P1=1, P2=0`. Coinciden en que la decisión O4a exige terminalidad
durante `PARADA_FINAL` hacia A7 sin KILL, mientras O4b la normaliza a
`NO_ESTABLE` y el candidato siempre autoriza KILL. También quedó acreditado
que una presencia observada exactamente en `finGracia` se rechaza aunque esa
igualdad es el borde que abre la parada final.

La revisión de seguridad añadió una ventana causal: una presencia posterior
a CONT puede dejar de ser cierta antes de `finGracia`. Comparar solamente el
reloj al vencer gracia no vuelve a acreditar que el grupo siga vivo. Conceder
STOP y KILL desde esa evidencia anterior viola privilegio mínimo cuando ya
existe terminalidad.

No hay fuga de autoridad, dato o credencial. El defecto es de causalidad y
permisos de señal. El candidato `2b7eaf4` permanece histórico y no se modifica.

La primera versión de esta enmienda, commit
`a89a3228554f53b32f5d81fc8b0438835f35b0f6`, recibió también doble
`NO-GO`, `P0=0, P1=1, P2=0`: revisión funcional
`cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` y revisión de seguridad
`e7e06423941807a41c40946e5a4af12e1b30bb11`. Ambas probaron que el preflight
podía cruzar `finParadaFinal` después de su única comprobación y, al prohibir
un reloj posterior, permitir un STOP tardío. V2 corrige solo ese borde; V1 y
sus actas permanecen históricas.

V2, commit `9de3ba320dfa4beede58e5a1d34aa02a3459f073`, recibió `GO` de
seguridad `569cc9ccfb94d3f195a6c08340b0e0e10429da96`, pero `NO-GO`
funcional `876b7c8e330dc28003dab1d0057ca4bc9d83a727`,
`P0=0, P1=1, P2=0`. El orden físico era seguro, pero su texto atribuía
conjuntamente presencia y vigencia a la lectura `ahoraFinal`, posterior al
último sondeo. V3 corrige solo esa atribución causal; V2 y sus actas permanecen
históricas y no acreditan estos bytes.

## Base, autoridades y prevalencia

La base de esta enmienda es
`5345d5d097b51ab3567983f048feabeceaf2957b`, cierre publicado de
`O4B-P0-CONTRATO`. O4B-P0 obtuvo doble GO y la CI `31546649383` terminó con
cinco de cinco puertas verdes.

Prevalecen sin cambio:

1. O3a para mapa FD, lease, observador, TID e identidad;
2. O3b para identidad `/proc`, auto-STOP y primitivas probatorias;
3. O3c para CONT inicial, primera observación y custodia C5;
4. O4a para causa, precedencia, reloj, etapas y autorización;
5. O4b solo para ejecutar el efecto autorizado y producir evidencia física
   no recolectora.

Esta enmienda sustituye únicamente las frases O4b que niegan la arista
terminal de `PARADA_FINAL`, la cardinalidad incondicional de esa etapa y la
clasificación del borde de gracia. Todo lo demás de O4a y O4b conserva su
autoridad, incluidos los límites absolutos, la disciplina lease, raws,
one-shot, fatalidad, custodia y prohibiciones.

## Decisión cerrada

### 1. Resultado posterior a TERM y CONT

TERM y CONT siguen siendo dos efectos como máximo, en ese orden y bajo
permisos lease separados. Los errores raw y la consolidación incierta
conservan exactamente las ramas publicadas.

Si TERM=0 y CONT=0 están consolidados, la evidencia posterior opaca puede ser
solo:

- `TERMINAL`, con terminalidad natural acreditada por ambas referencias
  pidfd; o
- `GRUPO_PRESENTE`, con ambas referencias no terminales y la identidad
  `/proc` completa vigente.

La marca `observado` debe ser monotónica, no anterior a `ahoraCaso` y no
posterior a `finGracia`. El arbitraje del resultado precede a la decisión del
deadline: por ello `TERMINAL` observado en igualdad exacta con `finGracia`
también llega A7 con drenaje cooperativo y cero señal posterior.

`GRUPO_PRESENTE` observado antes de `finGracia` devuelve O4a a A3 y permite
esperar sin emitir efecto. La igualdad exacta acredita el borde y permite
preparar inmediatamente `PARADA_FINAL`. Una marca posterior al límite, cero,
reconstruida, futura respecto de la lectura consumidora o con otra clase es
AF. Nunca se cambia `TERMINAL` por presencia ni a la inversa.

La presencia anterior al límite no prueba por sí sola presencia al vencerlo.
Al llegar `ahora >= finGracia`, pero siempre con
`ahora < finParadaFinal`, O4a puede emitir únicamente la autorización
condicional `PARADA_FINAL` definida abajo. Esa autorización todavía no prueba
que STOP sea necesario.

### 2. Preflight obligatorio de PARADA_FINAL

`PARADA_FINAL` autoriza **como máximo un STOP**, no obliga a intentar uno
contra un proceso ya terminal. Después de consumir el permiso one-shot,
validar custodia y acreditar `ahora < finParadaFinal`, O4b ejecuta una única
observación no recolectora de terminalidad antes del efecto. Reutiliza las
primitivas O3b publicadas; primario y reserva se acreditan por separado y cada
syscall conserva su permiso lease propio.

La autorización conserva `cardinalidadMaxima=1`; el resultado conserva la
cardinalidad realmente intentada, únicamente 0 o 1. No se reutiliza el mismo
campo para dos significados ni se fabrica un raw de éxito cuando no hubo
syscall.

El resultado del preflight es exhaustivo:

| Evidencia previa | Efecto y resultado |
| --- | --- |
| Ambas referencias acreditan terminalidad natural | Cero STOP; resultado `TERMINAL`, cardinalidad 0, raws cero y marca observada. |
| Ambas referencias acreditan no terminalidad e identidad vigente y la lectura final queda antes del límite | STOP es el siguiente syscall funcional. |
| La lectura final es igual o posterior a `finParadaFinal` | OBF directo; cero STOP y ningún resultado. |
| Referencias discordantes, identidad/flags/lease dudosos o retorno no clasificable | OBF directo; cero STOP, resultado o efecto posterior. |

Si acredita no terminalidad e identidad vigente, después de consolidar toda
esa evidencia física el preflight hace exactamente una lectura monotónica
final. Para presencia exige
`ahoraFinal.Before(finParadaFinal)`. Igualdad o vencimiento son OBF directo:
la autorización queda consumida, no se sella ni devuelve resultado, no existen
cardinalidad, raw, marca o incidente ordinarios y O4a no toma otra transición
A5; rige la fatalidad heredada 65/EOF/stdout=0/stderr=0, sin señal, cierre,
log, limpieza ni efecto posterior.

La evidencia física de presencia se linealiza exclusivamente en el último
sondeo consolidado que acredita ambas referencias no terminales e identidad
vigente. La lectura posterior `ahoraFinal` no vuelve a observar presencia ni
traslada ese instante: acredita solo que el permiso sigue vigente. Esta
decisión combina dos hechos ordenados —presencia en el último sondeo y
vigencia en `ahoraFinal`— sin afirmar simultaneidad entre ellos.

Si la lectura queda verde, el sobre ya estaba preasignado; solo se prepara el
permiso lease de señal y STOP es la siguiente syscall literal. No se intercala
otra lectura de reloj, sonda, validación, asignación falible, espera o log. La
terminalidad que ocurra después del último sondeo de presencia, incluso entre
ese sondeo y `ahoraFinal`, es posterior a la decisión física autorizante;
STOP continúa autorizado siempre que `ahoraFinal` quede verde. No se abre un
bucle. La evidencia posterior al STOP detecta esa terminalidad y evita KILL.

### 3. Resultado de PARADA_FINAL

Si el preflight acreditó presencia, STOP se intenta una vez. Raw no cero,
incluido EINTR, conserva cardinalidad 1, `SIN_EVIDENCIA` y cero reintento.
Raw cero habilita la evidencia no recolectora ya contratada, ampliada de forma
cerrada a:

- `TERMINAL`: ambas referencias acreditan terminalidad; cardinalidad 1;
- `ESTABLE`: dos muestras T completas e idénticas; cardinalidad 1;
- `NO_ESTABLE`: cualquier otra forma controlable ya publicada;
- duda física o lease incierto: OBF, no resultado.

O4a acepta `TERMINAL` en `PARADA_FINAL` con cardinalidad 0 o 1, exige raws
cero y una marca monotónica dentro de `finParadaFinal`, consume el resultado,
fija la indicación privada de terminalidad y transita A5→A7 con drenaje
cooperativo. No enclava incidente y no autoriza KILL.

`ESTABLE` conserva la autorización inmediata de KILL cooperativo sin
incidente. `NO_ESTABLE` o raw STOP no cero enclavan incidente de cierre y
autorizan KILL cooperativo si `ahora < finDrenajeCooperativo`. La duda de
consolidación sigue siendo AF y nunca autoriza otra syscall.

### 4. PARADA_INICIAL permanece distinta

`PARADA_INICIAL` no adquiere cardinalidad cero ni resultado `TERMINAL`. Sigue
intentando un STOP y exige las dos muestras T para `ESTABLE`. La terminalidad
del líder u otra no-estabilidad controlable tras STOP se devuelve
`NO_ESTABLE`: todavía puede existir grupo y no se ha abierto la rama
cooperativa, por lo que O4a enclava incidente y converge a KILL con drenaje
rápido. Un `TERMINAL` forjado para esta etapa es AF.

Esta asimetría es deliberada: preserva la fila STOP inicial publicada y
corrige solo la arista terminal explícita de `PARADA_FINAL`.

## Tabla sustitutiva de las aristas afectadas

| Estado/observación | Única transición |
| --- | --- |
| TERM y CONT consolidados; `TERMINAL` con `observado <= finGracia` | A7, drenaje cooperativo y cero otra señal. |
| TERM y CONT consolidados; `GRUPO_PRESENTE` con `observado < finGracia` | A3; esperar hasta el borde sin señal. |
| TERM y CONT consolidados; `GRUPO_PRESENTE` con `observado == finGracia` | Autorizar `PARADA_FINAL` condicional; A5. |
| A3 alcanza `finGracia` desde presencia anterior y aún está antes de `finParadaFinal` | Autorizar `PARADA_FINAL` condicional; A5. |
| Preflight final observa terminalidad | Resultado `TERMINAL`, cardinal 0; A7, cero STOP/KILL/incidente. |
| Preflight final observa presencia y la lectura final es `>= finParadaFinal` | OBF, autorización consumida, cero resultado/STOP/efecto posterior. |
| Preflight final observa presencia y la lectura final es `< finParadaFinal` | Preparar permiso; STOP es la siguiente syscall literal. |
| Preflight final observa presencia y STOP=0; evidencia posterior terminal | Resultado `TERMINAL`, cardinal 1; A7, cero KILL/incidente. |
| STOP final=0 y evidencia `ESTABLE` | Autorizar KILL cooperativo sin incidente. |
| STOP final=0 y `NO_ESTABLE`, o raw STOP no cero | Incidente de cierre y KILL cooperativo dentro del drenaje. |
| Marca posterior al límite, clase/cardinal/raw forjados o propiedad incierta | AF/OBF según propietario; cero autorización posterior. |

Las demás filas de las tablas O4a/O4b permanecen literales.

## Materialización futura acotada

Esta enmienda no contiene código. Solo después de doble GO, publicación y CI
5/5 dirección puede asignar un nuevo candidato corrector O4A-P4 sobre el
padre histórico exacto que determine. Ese corte material deberá:

1. aceptar por separado `TERMINAL` final cardinal 0/1 y presencia al borde;
2. llegar A7 sin incidente ni KILL para ambos terminales finales;
3. conservar `PARADA_INICIAL` y todas las ramas no afectadas;
4. añadir focales para `NO_ESTABLE` inicial/final, terminalidad final antes y
   después de STOP, presencia en igualdad, cruce de `finParadaFinal` dentro
   del preflight y forjas de etapa/cardinalidad;
5. no añadir syscall, señal, Wait, FD, parser, goroutine, API o getter a O4a.

La futura implementación O4b-P3 deberá materializar el preflight y su
cardinalidad 0/1; O4B-P1 sigue bloqueado por O4A-P4 y no se abre mediante esta
enmienda. O4A-P4 no ejecutará el preflight: solo define y consume el resultado
opaco que O4b materialice después.

## Matriz mínima y mutantes

| ID | Oráculo nuevo o corregido |
| --- | --- |
| E01 | `TERMINAL` post-CONT antes e igual a gracia produce A7, cero otra señal. |
| E02 | Presencia antes de gracia espera; presencia en igualdad prepara una sola parada final. |
| E03 | Presencia o terminalidad posterior a gracia falla cerrada sin permiso nuevo. |
| E04 | Preflight final terminal produce cardinal 0, raws cero, A7 y cero STOP/KILL/incidente. |
| E05 | Preflight presente cuya lectura final iguala o vence `finParadaFinal` produce OBF, cero resultado y cero STOP. |
| E06 | Preflight presente con lectura final verde hace de STOP la siguiente syscall literal. |
| E07 | Presencia se linealiza en el último sondeo; `ahoraFinal` acredita solo vigencia. |
| E08 | Terminalidad entre el último sondeo y el reloj es posterior a la decisión física; con reloj verde STOP sigue autorizado. |
| E09 | Preflight presente, STOP y terminalidad posterior produce cardinal 1, A7 y cero KILL/incidente. |
| E10 | STOP final estable/no estable/raw error conserva sus tres ramas exactas. |
| E11 | Terminalidad en STOP inicial sigue `NO_ESTABLE`; `TERMINAL` forjado es AF. |
| E12 | Cardinal 0 fuera de `PARADA_FINAL`, cardinal 2, raw no cero con terminal o marca forjada son AF. |
| E13 | Alias/replay/carrera dejan un ganador y ninguna señal excede el cardinal máximo. |
| E14 | AST/tipos prueba cero syscall en O4a y preflight inmediato sin Wait/señal cero en O4b. |

Mutantes mínimos: volver a normalizar terminal final; emitir STOP tras
preflight terminal; emitir KILL tras terminal cardinal 0 o 1; enclavar
incidente terminal; aceptar cardinal 0 inicial; convertir terminal inicial en
A7; rechazar igualdad de gracia; aceptar marca posterior; omitir una
referencia pidfd; omitir, invertir, adelantar o falsear la lectura final;
aceptar igualdad de `finParadaFinal`; sellar resultado al vencer; intercalar
una segunda lectura/sonda entre la lectura final verde y STOP; reintentar
sondeo/STOP; compartir permiso; tratar duda física como presencia; atribuir
presencia a `ahoraFinal`; exigir que la no-terminalidad permanezca monotónica
hasta el reloj; clasificar como previa una terminalidad posterior al sondeo.
Cada mutante debe compilar y morir por su oráculo causal; timeout, SHA global
o no compilación no cuentan como muerte.

## Seguridad, datos y límites

La corrección reduce permisos: terminalidad acreditada impide STOP/KILL. No
añade identidad, dato personal, ticket, nonce, PID/pidfd exportable, secreto,
log, HTTP, SQL, red, Docker ni producción. No cambia causa, estado exterior,
deadline, ownership, lease, observador, TERMINAL, Wait o limpieza.

No se cambian F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` ni producción `NO-GO`.

## Paradas y cierre

Se detiene ante cualquier señal después de terminalidad acreditada; STOP sin
preflight final; cardinalidad ambigua; terminal inicial aceptado como A7;
marca posterior al límite; ausencia/inversión de la lectura monotónica final;
resultado o STOP al vencer; repetición del preflight, espera u otra lectura,
sonda o efecto entre la lectura final verde y STOP; raw sin consolidar; parser/primitiva nuevos; Wait,
señal cero, fallback o recurso alternativo; cambio de causa, métricas, estado
transversal o candidatos históricos.

El cierre documental exige dos revisiones independientes `GO`,
`P0=P1=P2=0`, hashes, enlaces, Gitleaks y diff-check verdes, publicación normal
y CI 5/5. El productor no se autoaprueba, integra, fusiona ni despliega.

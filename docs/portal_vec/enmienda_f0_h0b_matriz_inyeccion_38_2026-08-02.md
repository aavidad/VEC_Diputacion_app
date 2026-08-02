# Enmienda F0-H0b: matriz de inyección de 38 casos

Fecha: 2 de agosto de 2026.

Estado: **propuesta documental; exige doble `GO` documental independiente
antes de reanudar el checkpoint 4**.

## Motivo medido

El checkpoint 3 deja el runner funcional en 630 líneas y el auxiliar H0b en
460. La lectura del flujo real posterior al `COMMIT` R0 identifica 23
fronteras y el finalizador contiene 15 llamadas de limpieza o acreditación.
El checkpoint 4 debe probar cada una por separado, además del recorrido
nominal, sin convertir varias fronteras en un único caso aparente.

El trabajo se detuvo antes de exceder el límite duro de 650 líneas del runner.
La segunda contrarrevisión confirma los 38 casos, pero emite `NO-GO` al
presupuesto anterior porque no estaba desglosado ni era reproducible. La nueva
medición conservadora no descuenta ahorros:

| Bloque del runner | Base o incremento mínimo | Máximo conservador |
| --- | ---: | ---: |
| Runner del checkpoint 3 | 630 | 630 |
| Selector, *seam* y traza | +8 | +12 |
| Cruce de las 23 fronteras | +10 | +14 |
| Finalizador, idempotencia y centinelas | +6 | +10 |
| Driver de 39 procesos y recuperación exacta | +14 | +20 |
| Propagación, nominal y rechazos | +5 | +8 |
| **Delta checkpoint 4** | **+43** | **+64** |
| **Runner completo** | **673** | **694** |

Para el auxiliar H0b la medición, también sin ahorros, es:

| Bloque del auxiliar H0b | Base o incremento mínimo | Máximo conservador |
| --- | ---: | ---: |
| Auxiliar del checkpoint 3 | 460 | 460 |
| Catálogo positivo | +43 | +43 |
| Oráculo independiente completo | +43 | +43 |
| Estructura y mutantes | +10 | +14 |
| **Auxiliar completo** | **556** | **560** |

Esta medición real sustituye la previsión de capacidad anterior para la
matriz, no el objetivo general de 500 líneas ni el límite global de 800 de
[DEC-051](registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
No se autoriza a compactar artificialmente, eliminar controles o trasladar
fronteras para hacer caber la prueba.

Conservar simultáneamente 150 líneas de reserva para I0 no está demostrado con
la matriz real. Prima implementar las 38 inyecciones completas con código
legible; quedan 100 líneas hasta el límite global. Esa reserva no acredita ni
desbloquea I0. Antes de iniciar I0 se exige una replanificación y decisión
separadas; no se minifica ni se supera el límite global de 800.

## Límites y propiedad

El runner
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` conserva:

- objetivo o umbral local de revisión de **690 líneas o menos**;
- límite duro de **700 líneas o menos**;
- **100 líneas** hasta el límite global de 800, sin habilitar I0.

El mismo auxiliar privado
`arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh` pasa a:

- objetivo de revisión de **560 líneas o menos**;
- límite duro de **580 líneas o menos**, como margen para conservar legibilidad.

La ampliación del auxiliar autoriza exclusivamente el catálogo positivo de
casos y un oráculo puro que almacena resultados esperados literales. El
auxiliar no calcula ni propaga la causal real. Tampoco autoriza procesos,
Docker, `psql`, sesiones, estados causales, activación del *seam*, orden de
ejecución, finalización ni recuperación. No se crea un cuarto auxiliar.

El auxiliar SQL D2c, el auxiliar operativo D2d y el capturador permanecen byte
a byte inmutables. El runner sigue siendo el único propietario del driver, los
procesos hijos, contenedores, sesiones, propagación de estados, *seam*,
finalizador y trampa exterior.

## Las 23 fronteras posteriores al COMMIT R0

La matriz usa identificadores literales y estables. El orden completo es:

| Caso | Frontera tras la que se inyecta el fallo |
| --- | --- |
| `A01` | Primera acreditación del R0 recién confirmado. |
| `N01` | Creación del directorio para el modo nominal. |
| `N02` | Materialización nominal. |
| `N03` | Copia nominal de M080. |
| `N04` | Acreditación de la huella nominal de M080. |
| `N05` | Copia nominal de T080. |
| `N06` | Acreditación de la huella nominal de T080. |
| `N07` | Copia nominal del wrapper. |
| `N08` | Acreditación de la huella nominal del wrapper. |
| `N09` | Cierre de la sesión `psql` nominal. |
| `N10` | Comprobación exacta del resultado nominal. |
| `A02` | Segunda acreditación R0, posterior al modo nominal. |
| `E01` | Creación del directorio para el modo de error. |
| `E02` | Materialización del modo de error. |
| `E03` | Copia de M080 para el modo de error. |
| `E04` | Acreditación de la huella de M080 para el modo de error. |
| `E05` | Copia de T080 para el modo de error. |
| `E06` | Acreditación de la huella de T080 para el modo de error. |
| `E07` | Copia del wrapper para el modo de error. |
| `E08` | Acreditación de la huella del wrapper para el modo de error. |
| `E09` | Cierre de la sesión `psql` del modo de error. |
| `E10` | Comprobación exacta del resultado de error. |
| `A03` | Tercera acreditación R0, posterior al modo de error. |

Son exactamente tres acreditaciones R0 y diez fronteras para cada uno de los
modos nominal y de error: `3 + 10 × 2 = 23`. La creación de directorio, cada
copia y cada huella son fronteras distintas. Un punto común situado únicamente
al final de la preparación no satisface este contrato.

Cada inyección devuelve una causal distintiva `79`. Todos los retornos desde
que R0 puede haberse confirmado atraviesan el finalizador único. Si sus 15
acciones terminan verdes, el proceso debe conservar y devolver exactamente
`79`; no puede sustituirlo por `1`, `0` ni `65`.

## Las 15 acciones reales del finalizador

La matriz inyecta un fallo independiente antes de cada llamada real y conserva
su orden actual:

| Caso | Acción que falla de forma aislada |
| --- | --- |
| `F01` | Retirar R0. |
| `F02` | Retirar el wrapper. |
| `F03` | Retirar M080 y T080 en una única acción agrupada. |
| `F04` | Retirar sus dos directorios en una única acción agrupada. |
| `F05` | Acreditar ausencia de R0. |
| `F06` | Acreditar el retorno exacto de `/repo_h0b`. |
| `F07` | Acreditar el retorno exacto de `/repo`. |
| `F08` | Acreditar la audiencia base. |
| `F09` | Acreditar el checkpoint base. |
| `F10` | Acreditar el catálogo base. |
| `F11` | Acreditar roles y membresías base. |
| `F12` | Acreditar ausencia de objetos F0. |
| `F13` | Acreditar ausencia de transacciones preparadas. |
| `F14` | Acreditar ausencia de objetos temporales PostgreSQL. |
| `F15` | Acreditar ausencia de sesiones cliente. |

El oráculo fija literalmente la secuencia ordenada completa desde la acción
seleccionada hasta el centinela terminal; no la calcula desde el bucle ni desde
el array que ejecuta el finalizador:

| Caso | Secuencia completa esperada desde la inyección |
| --- | --- |
| `F01` | `F01 → F02 → F03 → F04 → F05 → F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F02` | `F02 → F03 → F04 → F05 → F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F03` | `F03 → F04 → F05 → F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F04` | `F04 → F05 → F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F05` | `F05 → F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F06` | `F06 → F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F07` | `F07 → F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F08` | `F08 → F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F09` | `F09 → F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F10` | `F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F11` | `F11 → F12 → F13 → F14 → F15 → TERMINAL` |
| `F12` | `F12 → F13 → F14 → F15 → TERMINAL` |
| `F13` | `F13 → F14 → F15 → TERMINAL` |
| `F14` | `F14 → F15 → TERMINAL` |
| `F15` | `F15 → TERMINAL` |

El fallo seleccionado no cortocircuita el finalizador. Un centinela por acción
demuestra cada intento posterior y un centinela terminal demuestra que se
alcanzó el final de la secuencia, incluso cuando falla `F15`. Cualquier fallo
de `F01` a `F15` sustituye la causal por `65`; ninguna combinación devuelve el
estado previo ni éxito.

## Aislamiento y recuperación exterior

La matriz contiene **38 casos de fallo aislados más un caso nominal**. Cada
uno de los 39 casos nace en un proceso nuevo y usa un contenedor PostgreSQL
nuevo; no se reutilizan base, roles, filesystem, temporales ni estado de
shell entre casos.

El proceso nominal mantiene el *seam* cerrado, exige estado `0`, acredita la
secuencia completa `F01 → F02 → F03 → F04 → F05 → F06 → F07 → F08 → F09 →
F10 → F11 → F12 → F13 → F14 → F15 → TERMINAL` y demuestra recuperación
exacta de base de datos, ambas raíces, contenedor y temporales.

El proceso conductor ejecuta los 23 casos de frontera y exige `79`, ejecuta
los 15 casos del finalizador y exige `65`, y contrasta en todos ellos los
centinelas esperados. Después de terminar cada hijo, la trampa exterior retira
el contenedor y el directorio temporal propios. El conductor acredita por la
identidad exacta de ambos recursos que no queda ningún residuo antes de iniciar
el caso siguiente. Una retirada global por prefijo o una inspección realizada
solo al final de los 38 casos no es equivalente.

El *seam* permanece cerrado en el recorrido nominal. En modo inyección, un
selector vacío, desconocido, repetido o no emitido por el conductor se rechaza
con estado exacto `64` antes de crear un temporal, reservar o arrancar un
contenedor y ejecutar cualquier llamada a Docker o `psql`. El conductor
acredita en cada una de esas autopruebas cero temporales y contenedores propios
y cero invocaciones a Docker o `psql`. Los identificadores `A`, `N` y `E`
reservan exclusivamente el estado `79`; los identificadores `F`, el estado
`65`. El *seam* no transporta secretos ni habilita una ruta productiva.

## Catálogo y oráculo independientes

El auxiliar H0b contiene dos autoridades probatorias separadas:

1. un catálogo positivo literal con los 38 identificadores ejecutables;
2. un oráculo puro y literal que fija, de forma independiente, estado esperado,
   secuencia ordenada completa de centinelas y condición de recuperación para
   cada identificador.

El oráculo no se deriva, transforma ni completa desde el catálogo usado por el
driver. El runner compara ambos conjuntos y cardinalidades, pero obtiene el
resultado esperado solo del oráculo independiente. Las autopruebas mutan por
separado un identificador y un estado. Sobre secuencias `F` con centinela
intermedio eliminan, reordenan, duplican y sustituyen ese elemento en casos
separados; además mutan el par mínimo `F15 → TERMINAL`. Cada divergencia debe
ser rechazada. Otra autoprueba muta de forma independiente la condición de
recuperación esperada y exige el mismo rechazo. Contar el mismo array que
gobierna la ejecución, derivar del bucle la secuencia esperada o calcular su
estado por el prefijo `A`, `N`, `E` o `F` sería un oráculo tautológico y no
cierra esta matriz.

## Prohibiciones y cierre

Para alcanzar los límites no se permite:

- minificar o agrupar sentencias sin una razón de diseño;
- retirar mensajes, huellas, acreditaciones, autopruebas o regresiones;
- convertir dos fronteras en un solo punto de inyección;
- mover Docker, `psql`, procesos, estados, *seam* o finalizador al auxiliar;
- modificar D2c, D2d o el capturador;
- añadir un cuarto auxiliar.

El checkpoint 4 requiere Bash, ShellCheck, PostgreSQL 18.4, los 39 procesos
aislados, límites, `git diff --check`, Gitleaks, cero residuos caso a caso y
revisión independiente. Esta enmienda no acredita su implementación, no
cierra H0b, C2 ni F0 y no cambia métricas. Producción continúa en `NO-GO`.

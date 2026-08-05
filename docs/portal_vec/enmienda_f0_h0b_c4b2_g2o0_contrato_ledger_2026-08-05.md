# Enmienda F0-H0b/C4b-2 G2-O0: contrato y ledger operativo

Fecha: 5 de agosto de 2026.

Estado: **NO-GO documental**. Tres revisiones independientes sobre `c71cab9`
obtuvieron `P0=0`, pero detectaron contratos incompletos de framing,
precedencia, topología binaria y ledger. No autoriza código. Su evidencia y
corrección exigida están en la
[revisión consolidada](revisiones/revision_f0_h0b_c4b2_g2o0_contrato_ledger_2026-08-05.md).

## Alcance único

G2-O0 sustituye la propuesta G2-O que recibió `NO-GO` documental. Fija la
máquina, precedencias, dominios, topología, write-set y condiciones de parada
para O1–O6. No modifica código, no abre el modo operativo y no anticipa
Docker, PostgreSQL, SQL, red o producción.

Base exacta: `3e36ecae23e0608bc1e7b9ce374e8fb35d13b4a2`.

## Autoridad y topología

- El runner conserva selector, nonce, identidad, ticket, causa funcional,
  señal observada, plazos exteriores, oráculo, cuarentena y decisión sobre el
  caso siguiente.
- El supervisor Go solo posee mecánica Linux: framing, proceso Bash, pidfd de
  grupo, señales, terminalidad, un único `cmd.Wait`, drenaje y recibo.
- Bash ejecuta únicamente el caso capturado mediante `/usr/bin/bash -p`; no
  decide autoridad ni interpreta identidad.
- El adaptador M38 conserva el ciclo de FD, `coproc`, espera exterior y
  cuarentena. D2d conserva operaciones auxiliares acotadas; ninguno fabrica
  éxito ni postausencia.
- Si Bash necesita validar un stream binario, no usa `read` ordinario. O5 debe
  elegir y acreditar un lector binario acotado; puede reutilizar un modo
  puramente sintáctico del mismo binario Go medido, sin aceptar decisiones ni
  campos de identidad. Cualquier proceso auxiliar queda inventariado,
  esperado y extinguido. La topología concreta necesita `GO` antes de O5.

## Estados y secuencia

Estados privados del supervisor:

```text
S0 ESPERAR_SOBRE
S1 ESPERAR_ARMAR
S2 LISTO
S3 INICIANDO
S4 EJECUTANDO
S5 TERMINANDO
S6 TERMINAL
```

1. Go lee y valida primero el sobre FD 9, de forma acotada. Retiene sus datos
   sin crear Bash ni usar el ticket.
2. En S1 acepta exactamente un `ARMAR` coherente con nonce y PID del runner y
   emite `ACK_LISTO`.
3. En S2 acepta una única decisión inicial: `CANCELAR` o `INICIAR`.
4. Tras aceptar `INICIAR`, el control permanece abierto. En S3 o S4 admite
   cero o un `CANCELAR` terminal. La primera cancelación válida queda
   enclavada; duplicados, EOF y señales posteriores no la sustituyen.
5. Mientras espera `ACK_CASO`, el adaptador acepta exactamente `ACK_CASO` o
   un `TERMINAL` válido. Un fallo de `Start`, pidfd, STOP o handoff puede ir
   directamente a `TERMINAL` sin fabricar `ACK_CASO`.
6. S6 no admite otra trama. El estado de salida del supervisor coincide con
   el estado terminal.

## Precedencia canónica

El propietario único del ciclo serializa acontecimientos. En cada iteración:

1. conserva cualquier terminal o cancelación ya enclavados;
2. procesa como máximo una trama de control completa ya disponible;
3. observa terminalidad pidfd;
4. evalúa el plazo absoluto;
5. trata EOF solo si no existe causa previa.

La primera cancelación válida procesada antes de observar terminalidad gana.
Si la terminalidad se observa primero, gana `SALIDA` con el estado real. Una
trama inválida anterior a ambas fija `PROTOCOLO`. Ningún acontecimiento
posterior cambia causa o estado ni provoca otra señal, espera o terminación.

Correspondencia sin causa previa:

| Acontecimiento | Causa | Estado |
| --- | --- | ---: |
| Cancelación funcional o EOF de control | `CANCELADO` | 65 |
| Plazo de bootstrap o de caso | `PLAZO` | 65 |
| Trama, dominio o secuencia inválidos | `PROTOCOLO` | 65 |
| Fallo de `Start`, pidfd, `prctl`, `poll`, handoff o recurso interno | `INCIDENTE` | 65 |
| Muerte paterna/PDEATHSIG | `INCIDENTE` | 65 |
| Primera señal INT observada por el runner | `SENAL_INT` | 130 |
| Primera señal TERM observada por el runner | `SENAL_TERM` | 143 |
| Salida Bash observada primero | `SALIDA` | estado real `0`, `64`, `65` o `79` |

## Framing y dominios

- Control y recibo son streams ASCII de siete bits. Cada trama termina en un
  único `LF` y ocupa como máximo 1024 bytes, incluido `LF`.
- El sobre FD 9 es una clase distinta, monoframa, con máximo 4096 bytes
  incluido `LF`. Tras la trama solo admite EOF.
- El ticket ocupa 1..2048 bytes imprimibles `0x20..0x7e`, puede contener `|` y
  debe caber además en el sobre completo. Su longitud se comprueba antes de
  copiarlo.
- Nonce e identidad son 64 hexadecimales minúsculos. PID, PPID y PGID son
  decimales mínimos `1..2147483647`; inicio `/proc`, `1..uint64_max`.
- `ADOPTADOS_PENDIENTES` es decimal mínimo canónico `0..2147483647`; solo el
  cero admite esa única representación.
- Se rechazan NUL, CR, TAB fuera del ticket, byte no ASCII, línea vacía, campo
  vacío no admitido, signo, espacio exterior, ceros iniciales y desbordamiento
  antes de ampliar o convertir.
- El lector incremental conserva sobrante, admite fragmentación y coalescencia
  y limita el buffer antes de cada ampliación. No confunde `read` con mensaje.

## Bootstrap y terminal exterior

Un sobre válido permite ligar cualquier terminal posterior al nonce. Si el
sobre falta, está truncado o es inválido, no existe nonce confiable: Go sale
65 sin emitir un `TERMINAL` normal y el adaptador registra una frontera externa
de bootstrap, cuarentena el conjunto y prohíbe el caso siguiente.

Después de validar el sobre:

- EOF o ausencia de `ARMAR` dentro del plazo produce `CANCELADO|65` o
  `PLAZO|65`, respectivamente;
- `ARMAR` malformado o incoherente produce `PROTOCOLO|65`;
- el ticket no se entrega ni se crea Bash antes de `INICIAR` válido.

## Postcondición terminal

El recibo conserva:

```text
BASH_CREADO|BASH_ESPERADO|ADOPTADOS_PENDIENTES|GRUPO_AUSENTE
```

| Fase alcanzada | Cierre controlado exigido |
| --- | --- |
| Antes de `Start` | `0|0|0|1` |
| Después de `Start`, antes de `ACK_CASO` | `1|1|0|1` |
| Después de `ACK_CASO` | `1|1|0|1` |

Solo una frontera externa que impida observar o comunicar la postcondición
puede dejar valores incompletos. Se clasifica siempre `INCIDENTE|65`, conserva
evidencia disponible, activa cuarentena y prohíbe un caso siguiente; nunca se
presenta como limpieza acreditada.

## Ledger físico y write-set

Estado integrado:

| Unidad | Líneas | Regla O1–O6 |
| --- | ---: | --- |
| Runner | 800 | Sin altas ni compresión. Solo sustitución de hashes literales ya existentes. |
| G1 | 683 | Dispatcher/autoprueba; sin mecánica operativa. Tope de parada 790. |
| G2 | 91 | Propietario de la mecánica Go. Tope de parada 790. |
| Capturador | 799 | Invariante byte a byte. |
| Adaptador M38 | 527 | Puente Shell y cuarentena. Detenerse antes de superar 600. |
| D2d | 145 | Primitivas Shell acotadas; no absorbe decisiones. Tope de parada 300. |

Estos topes son barreras de admisión, no previsiones de tamaño ni permiso para
rellenar capacidad. Antes de cada O1–O6 se adjuntan `wc -l`, `numstat`, rangos
reemplazados, SHA y una estimación conservadora de su único cambio. Si no cabe
sin minificar, si el runner necesita código o si el adaptador superaría 600,
la tarea se detiene y se aprueba primero una separación capturada nueva.

Write-set permitido por fase:

| Fase | Producción permitida |
| --- | --- |
| O1 | G2 y llamada de autoprueba en G1; hashes literales del runner. |
| O2 | G2 y autopruebas; hashes literales. Sin Bash creado. |
| O3 | G2 y autopruebas; hashes literales. |
| O4 | G2 y autopruebas; hashes literales. |
| O5 | Adaptador M38, D2d solo si es imprescindible, dispatcher/codec G1-G2 y hashes. |
| O6 | Solo pruebas, oráculos y documentación; producción solo para corregir un NO-GO reproducido. |

G1 y G2 siguen siendo las dos únicas fuentes Go privadas capturadas. O1–O4 no
añaden fichero, helper, proceso, FD o ruta al manifiesto. O5 debe aprobar antes
su topología exacta si necesita un lector auxiliar.

## Minitareas y cierre

| Fase | Resultado observable |
| --- | --- |
| O1 | Codec puro con límites, canonicidad, fragmentación y coalescencia. |
| O2 | Bootstrap `S0..S2`, cancelación/EOF sin crear Bash y recibo terminal. |
| O3 | `Start`, FD, pidfd, STOP, `/proc`, `ACK_CASO` y salida natural. |
| O4 | Plazo, cancelación en S3/S4, señales, `Wait`, drenaje, `ESRCH` y terminal. |
| O5 | Puente Shell, `coproc`, FD estables, lectura binaria acotada y cuarentena. |
| O6 | Matriz integradora, mutantes y fronteras externas. |

Cada fase compila y autoprueba de forma independiente. `--supervisar-m38`
permanece cerrado en 64 hasta que O4 complete la máquina Go y O5 complete el
puente; ningún corte intermedio ejecuta un caso real.

La matriz O6 contiene por fila: fase, estímulo, ACK esperado, causa/estado,
postcondición, número de `Wait`, `ECHILD`, `ESRCH`, FD, cuarentena y permiso de
caso siguiente. Los mutantes controlados exigen un solo terminal y residuos
cero. Las fronteras externas permiten ausencia de terminal únicamente cuando
este no puede ligarse o entregarse; siempre exigen 65, evidencia, cuarentena y
ningún caso siguiente.

O0 se cierra solo con doble `GO` documental y `P0=P1=P2=0`. No modifica
métricas ni cierra G2-O, C4b-2, C4b, H0b, C2, F0, O4-05 o producción.

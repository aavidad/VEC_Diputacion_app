# Enmienda F0-H0b/C4b-2 G2-O/O2b: control previo sin Bash

Fecha: 5 de agosto de 2026.

Estado: **doble `GO` documental final, `P0=P1=P2=0`**. La autorización de
código queda condicionada a confirmar y publicar este contrato con su acta, y
a comprobar verde la CI de ese padre exacto.

Base material exacta:

```text
0caa140409db5ac0a2f1312f13e002b702691e1b
```

El código de esa base quedó publicado dentro de `ef1f08b`; la ejecución
GitHub `31021785711` terminó con cinco de cinco puertas verdes. El corte
documental vigente es `fd8c2d8e1c39a315136dff5f49cc5904002887e9`. El futuro
commit que integre este contrato y sus dos dictámenes será el padre exacto del
candidato de código; dirección comunicará su SHA completo antes de editar.

## Prevalencia y frontera

Esta enmienda sustituye para O2b cualquier indicación incompatible de:

- la especificación G2-O rechazada
  `especificacion_f0_h0b_c4b2_g2o_protocolo_operativo_2026-08-05.md`;
- la enmienda O0 rechazada
  `enmienda_f0_h0b_c4b2_g2o0_contrato_ledger_2026-08-05.md`;
- cualquier protocolo con `ACK_LISTO`, `ACK_CASO` o `RECIBO` vivo;
- cualquier flujo que reciba `SOBRE` después de `ARMAR`;
- cualquier corte antiguo que agrupe S0--S2 o cree Bash dentro de O2.

Conserva como autoridades:

1. la corrección canónica O1a y su gramática;
2. el codec O1a y el lector incremental O1b de G2;
3. la recepción O2a/S0 de G3 integrada en `0caa140`;
4. la decisión C4b-2 en todo lo no sustituido expresamente.

Para O2b sustituye expresamente las cláusulas C4b-2 que fijaban el hilo,
instalaban el receptor de señales y ejecutaban `prctl(PR_SET_PDEATHSIG)`
inmediatamente después de `ARMAR`. Esas acciones Linux no pertenecen a una
máquina pura de memoria. El futuro O3 deberá, antes de construir `exec.Cmd` o
realizar cualquier acción Linux irreversible, fijar el hilo, instalar la
señalización y `prctl`, volver a comprobar PPID, EOF/control pendiente y todo
acontecimiento ya encolado. Ninguna muerte del runner ni control pendiente
podrá convertirse en `Start`. Las únicas transiciones no terminales que O2b
registra son S1 -> S2 y S2 -> S3; las entradas S5 permanecen obligatorias.

La división vigente queda:

```text
O2a  recibir y retener SOBRE: S0 -> S1                 completado
O2b  ARMAR, reconocer INICIAR y cancelar antes de Bash  este corte
O3   crear, acreditar y arrancar Bash                   no autorizado
```

«Reconocer `INICIAR`» significa únicamente registrar S2 -> S3. O2b no llama
`Start`, no crea proceso y no entrega el ticket. Esta separación es necesaria
para que una entrada ya disponible:

```text
ARMAR\nINICIAR\nCANCELAR\n
```

termine en cancelación antes de que una fase posterior pueda crear Bash.

## Responsabilidad única

O2b añadirá una fuente G4 pura y confinada a un propietario que:

1. toma propiedad exclusiva de un receptor O2a confirmado en S1;
2. crea un lector O1b configurado exactamente para `CONTROL`;
3. recibe fragmentos de memoria y un indicador de EOF;
4. consume las tramas completas ya suministradas hasta agotar la entrada o
   entrar en S5, incluidas las coalescidas después de `INICIAR`;
5. acepta una única secuencia `ARMAR` -> `INICIAR` coherente con el sobre;
6. enclava antes de Bash una cancelación canónica o un error de protocolo;
7. conserva el sobre únicamente mientras S2/S3 necesiten una fase posterior;
8. retira las referencias sensibles al entrar en S5 o ante fallo interno.

El controlador no es concurrente. El llamador transfiere la propiedad del
receptor al constructor y no vuelve a utilizar su alias. El contrato no añade
mutex, goroutine, canal, callback o interfaz para simular concurrencia.

O2b solo conoce los fragmentos entregados en cada llamada. La futura fase O3
será responsable de comprobar la frontera real del pipe antes de ejecutar
`Start`. Ningún resultado S2/S3 puede ocultar una trama completa o parcial ya
suministrada. S5, en cambio, termina deliberadamente el análisis: el sufijo
posterior a la causa primaria queda sin consumir y su propietario debe
descartarlo sin interpretarlo.

## Gramática heredada

O2b no vuelve a codificar la gramática. Consume exclusivamente las
`tramaM38` ya validadas por O1a/O1b:

```text
V1|CONTROL|ARMAR|NONCE|PID_RUNNER\n
V1|CONTROL|INICIAR|NONCE\n
V1|CONTROL|CANCELAR|NONCE|CAUSA|ESTADO\n
```

`CANCELAR` admite solo estas parejas canónicas, ya validadas por el codec:

| Causa | Estado |
| --- | ---: |
| `CANCELADO` | 65 |
| `PROTOCOLO` | 65 |
| `SENAL_INT` | 130 |
| `SENAL_TERM` | 143 |

`SENAL_INT` y `SENAL_TERM` son etiquetas transportadas por una trama de
control confiable. O2b no instala, observa, reenvía ni genera señales.

## Ruta y API privadas

Ruta nueva exacta G4:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go
```

La fuente empieza exactamente con:

```go
//go:build ignore && linux && amd64

package main
```

`go list` debe conservar el capturador como único `GoFiles` y clasificar G1,
G2, G3 y G4 como `IgnoredGoFiles`. Se prohíbe usar `-tags=ignore`.

API definitiva:

```go
type resultadoControlPreinicioM38 uint8

const (
	controlPreinicioNecesitaDatosM38 resultadoControlPreinicioM38 = iota
	controlPreinicioArmadoM38
	controlPreinicioInicioPendienteM38
	controlPreinicioCausaEnclavadaM38
)

type faseControlPreinicioM38 uint8

const (
	controlPreinicioS1M38 faseControlPreinicioM38 = iota
	controlPreinicioS2M38
	controlPreinicioS3M38
	controlPreinicioS5M38
)

type causaPreinicioM38 struct {
	faseOrigen faseControlPreinicioM38
	causa      string
	estado     string
}

type controladorPreinicioM38 struct {
	recepcion *receptorSobreS0M38
	lector    *lectorTramaM38
	nonce     [64]byte
	fase      faseControlPreinicioM38
	causa     causaPreinicioM38
	fallo     error
}

func nuevoControladorPreinicioM38(
	recepcion *receptorSobreS0M38,
) (*controladorPreinicioM38, error)

func (c *controladorPreinicioM38) consumir(
	fragmento []byte,
	fin bool,
) (consumidos int, resultado resultadoControlPreinicioM38, err error)
```

No se autoriza otro constructor, método público, getter, `String`,
serializador, canal, callback o interfaz. Las fases posteriores compartirán el
mismo paquete privado y recibirán el controlador completo mediante otro
contrato; O2b no expone el sobre, el ticket o la causa.

G4 puede definir únicamente estos centinelas nuevos:

```go
errControlPreinicioNuloM38
errSobreNoConfirmadoPreinicioM38
errInvarianteControlPreinicioM38
errUsoPosteriorControlPreinicioM38
```

El constructor nulo o con O2a aún en S0 devuelve
`errSobreNoConfirmadoPreinicioM38`. Un receptor que afirma S1 pero contradice
sus invariantes devuelve `errInvarianteControlPreinicioM38`. Un método sobre
controlador nulo devuelve `errControlPreinicioNuloM38`; S5 devuelve siempre
`errUsoPosteriorControlPreinicioM38` cuando es funcional; S5 con fallo interno
devuelve el primer error exacto. Si la construcción exacta del lector CONTROL
devolviera un error O1b, se conserva ese objeto después de retirar el sobre.
No se decide por el texto del error.

## Constructor, propiedad y retención

El constructor:

1. toma propiedad del receptor no nulo desde el comienzo de la llamada;
2. exige `fallo == nil`, fase O2a S1, lector O2a L3 limpio y un sobre completo;
3. copia exactamente los 64 bytes del nonce validado a `nonce [64]byte`;
4. invoca exactamente `nuevoLectorTramaM38("CONTROL")`;
5. retorna un controlador en S1, sin causa ni fallo.

La copia fija del nonce es la única copia material nueva autorizada. El nonce
es un identificador técnico aleatorio y minimizado, pero puede correlacionar
acontecimientos del mismo caso. Nunca se registra, expone ni persiste y se
conserva solo hasta el terminal posterior. La comparación con campos de
control se hace byte a byte contra ese array, sin convertirlo a `string` ni
crear otra `[]byte`.

No se copia, interpreta ni revalida PID, selector, identidad, longitud o
ticket. El constructor solo comprueba estructuralmente que las seis cabeceras
retenidas estén presentes y que el nonce mida 64 bytes antes de copiarlo; la
validez sintáctica continúa siendo autoridad de O2a/O1a. Mientras el
controlador esté en S1, S2 o S3 conserva una sola referencia al receptor O2a
y, por ella, el único respaldo inmutable del sobre. El PID se compara una sola
vez al aceptar `ARMAR`; el contenido de selector, identidad, longitud y ticket
no se consulta.

Si el constructor recibe un receptor no nulo pero inválido, la transferencia
de propiedad ya se ha producido: limpia el buffer O2a, retira todas las
referencias del sobre y deja el receptor inutilizable antes de devolver error.
No devuelve un controlador parcial. Los estados exactos son:

| Entrada al constructor | Estado del receptor después | Error devuelto |
| --- | --- | --- |
| receptor nulo | no existe | `errSobreNoConfirmadoPreinicioM38` |
| receptor con `fallo != nil` | S0, limpio, mismo `fallo` | el mismo objeto |
| receptor limpio todavía en S0 | S0, limpio, `fallo=errSobreNoConfirmadoPreinicioM38` | ese centinela |
| receptor que afirma S1 pero contradice invariantes | S0, limpio, `fallo=errInvarianteControlPreinicioM38` | ese centinela |
| fallo al crear el lector CONTROL | S0, limpio, `fallo` igual al error O1b | el mismo objeto |

Al enclavar una causa S5 o un fallo interno, G4:

- limpia el buffer CONTROL;
- limpia el buffer O2a si aún existe;
- pone a cero los seis campos de `sobreRetenidoM38`;
- deja el receptor O2a inutilizable con el error de retirada fijado para la
  vía funcional o interna;
- elimina del controlador las referencias al receptor y a ambos lectores;
- conserva solo el nonce fijo y la causa canónica no sensible si existe.

Después de una causa funcional, el controlador queda exactamente en S5, con
causa completa, `fallo=nil` y referencias nulas; el receptor retirado queda en
S0 con `fallo=errUsoPosteriorSobreM38`. Después de un fallo interno, el
controlador queda en S5, causa cero, el primer error exacto y pegajoso en
`fallo`, y referencias nulas; el receptor retirado queda en S0 con ese mismo
error. Una llamada posterior a S5 funcional devuelve
`errUsoPosteriorControlPreinicioM38`; una posterior al fallo interno devuelve
el mismo objeto interno. Las dos entradas en S5 pasan por ayudantes separados
y únicos: causa funcional e invalidación interna.

Poner a cero cabeceras de cadena retira referencias; no promete borrado físico
de cadenas inmutables en el runtime Go. La revisión estructural debe demostrar
que G4 no crea otra copia que prolongue la vida del ticket o de la identidad.

En S3 sin causa el receptor se conserva intacto para O3. O2b no lo entrega ni
lo consume todavía.

## Máquina de estados

| Fase activa | Evento | Transición y resultado |
| --- | --- | --- |
| S1 | `ARMAR` con nonce y PID exactos | S2; armado |
| S1 | cualquier otra trama completa | S5; `PROTOCOLO|65`, origen S1 |
| S1 | EOF limpio | S5; `CANCELADO|65`, origen S1 |
| S2 | `INICIAR` con nonce exacto | S3; inicio pendiente, sin `Start` |
| S2 | `CANCELAR` con nonce y pareja exactos | S5; causa transportada, origen S2 |
| S2 | cualquier otra trama completa | S5; `PROTOCOLO|65`, origen S2 |
| S2 | EOF limpio | S5; `CANCELADO|65`, origen S2 |
| S3 | `CANCELAR` con nonce y pareja exactos | S5; causa transportada, origen S3 |
| S3 | cualquier otra trama completa | S5; `PROTOCOLO|65`, origen S3 |
| S3 | EOF limpio | S5; `CANCELADO|65`, origen S3 |
| S5 | cualquier llamada | error estable de uso posterior, sin inspección |

Reglas cruzadas:

- `CANCELAR`, incluido uno con señal canónica, en S1 es secuencia inválida y
  se normaliza a `PROTOCOLO|65`;
- `INICIAR` en S1, `ARMAR` repetido e `INICIAR` repetido son protocolo;
- nonce o PID distintos son protocolo, no error interno;
- la fase de origen se captura justo antes de S5 y nunca puede ser S5;
- una causa primaria es inmutable;
- las cadenas almacenadas en `causaPreinicioM38` son constantes canónicas de
  G4, nunca cabeceras procedentes de la trama recibida.

## Drenaje, EOF y contadores

Una llamada a `consumir` recorre el fragmento hasta que ocurra una de estas
condiciones:

1. necesita más bytes para completar la siguiente trama;
2. ha drenado todos los bytes suministrados y queda en S2 o S3;
3. observa EOF limpio;
4. enclava una causa S5;
5. detecta un fallo interno.

Una llamada interpreta como máximo tres tramas completas —`ARMAR`, `INICIAR`
y una tercera trama terminal o inválida—, como máximo 3072 bytes de prefijo
físico en total y una única trama parcial, incluida dentro de ese límite y
acotada por O1b. No recorre, copia ni reserva un sufijo posterior a S5.

Después de una trama completa continúa con el sobrante de la misma llamada.
Por ello:

- `ARMAR+INICIAR` coalescidos terminan en S3;
- `ARMAR+CANCELAR` coalescidos terminan en S5;
- `ARMAR+INICIAR+CANCELAR` coalescidos terminan en S5 antes de Bash;
- `ARMAR+INICIAR+` una parcial de `CANCELAR` devuelve necesita datos, no
  inicio pendiente;
- ningún resultado armado o inicio pendiente oculta bytes suministrados;
- después de `ARMAR+CANCELAR`, cualquier `RESTO` queda deliberadamente fuera
  del contador y se descarta sin interpretar por el propietario.

Los consumidos son la suma exacta de los contadores O1b. Si O1b falla al
procesar una trama, su contador es cero y `consumidos` conserva únicamente los
bytes de tramas completas anteriores de esa llamada. Un fallo interno devuelve
siempre tupla cero, aunque hubiera trabajo previo; el estado sensible queda
retirado.

Una entrada vacía sin EOF devuelve el estado estable si el lector está L0:
armado en S2 e inicio pendiente en S3. Si existe una parcial L1, devuelve
necesita datos. En S1 vacío devuelve necesita datos.

La precedencia de EOF es:

- trama completa antes que el EOF indicado en la misma entrega;
- después de aplicar `ARMAR` o `INICIAR`, ese EOF limpio termina como
  `CANCELADO|65` desde la nueva fase S2 o S3;
- EOF limpio inicial o posterior en S1/S2/S3 es cancelación;
- EOF con parcial es `PROTOCOLO|65`, nunca cancelación;
- si una cancelación completa precede al EOF, la causa de la cancelación es
  primaria y el EOF no la sustituye.

### Matriz física exigida a O1b

Al comenzar una llamada pública, un controlador activo solo admite L0 o L1.
L2 o L4, y cualquier L3 no producido en esa misma vuelta para drenar un EOF,
son fallo interno. Cada devolución O1b debe cumplir exactamente:

| Resultado O1b | Trama/contador | Estado físico |
| --- | --- | --- |
| `lecturaNecesitaDatosM38` | trama cero; contador igual a todo el sufijo entregado; `fin=false` | L0 solo sin parcial previa o nueva; L1 en otro caso |
| `lecturaTramaM38` | contador `1..len(sufijo)`; clase `CONTROL`; cardinalidad ya validada | L0, salvo L3 si consume el último byte con `fin=true` |
| `lecturaEOFLimpioM38` | trama cero, contador cero, sufijo vacío y `fin=true` | L3 |
| `lecturaTramaFinalM38` | imposible para CONTROL | fallo interno |
| error físico reconocido | trama cero, contador y resultado cero | L4 |

El contador nunca puede ser negativo, superar el sufijo entregado o hacer que
el total supere el fragmento original. Una trama con contador cero es fallo
interno y nunca vuelve al bucle. El L3 transitorio producido por una trama que
consume exactamente el último byte con `fin=true` se drena de inmediato
mediante una única llamada EOF vacía; no se devuelve al propietario como fase
activa.

### Tuplas externas exactas

| Situación al retornar | `consumidos` | `resultado` | `error` |
| --- | ---: | --- | --- |
| S1 vacío/parcial sin EOF | bytes aceptados en esta llamada | necesita datos | `nil` |
| `ARMAR` válido, entrada agotada sin EOF | total hasta ARMAR | armado | `nil` |
| S2 L0, llamada vacía sin EOF | 0 | armado | `nil` |
| S2 con parcial | bytes aceptados | necesita datos | `nil` |
| `INICIAR` válido, entrada agotada sin EOF | total hasta INICIAR | inicio pendiente | `nil` |
| S3 L0, llamada vacía sin EOF | 0 | inicio pendiente | `nil` |
| S3 con parcial | bytes aceptados | necesita datos | `nil` |
| `CANCELAR` válido o trama completa semánticamente inválida | total incluido el LF de esa trama | causa enclavada | `nil` |
| error físico/codec O1b reconocido | solo tramas completas anteriores | causa enclavada `PROTOCOLO|65` | `nil` |
| EOF limpio | total anterior; EOF aporta 0 | causa enclavada `CANCELADO|65` | `nil` |
| `fin=true` tras ARMAR/INICIAR válidos | total incluido su LF | causa enclavada `CANCELADO|65` desde S2/S3 | `nil` |
| fallo interno nuevo o pegajoso | 0 | valor cero | primer error exacto |
| controlador nulo | 0 | valor cero | `errControlPreinicioNuloM38` |
| llamada posterior a S5 funcional | 0 | valor cero | `errUsoPosteriorControlPreinicioM38` |

El valor cero del resultado coincide físicamente con «necesita datos», pero
carece de significado cuando `error != nil`; nunca se interpreta como progreso.

## Precedencia de errores

1. controlador nulo: centinela y tupla cero;
2. fallo interno ya enclavado: mismo objeto y tupla cero;
3. S5 funcional: error estable de uso posterior, sin inspeccionar entrada;
4. invariantes propias, de O2a y del lector CONTROL;
5. error físico o de codec O1b reconocido: S5 `PROTOCOLO|65`;
6. trama completa: secuencia, nonce, PID y causa;
7. EOF limpio: S5 `CANCELADO|65`;
8. parcial o ausencia de datos: resultado no terminal correspondiente.

Los únicos errores O1b atribuibles a entrada CONTROL y normalizables a
`PROTOCOLO|65` son `errByteFlujoM38`, `errExcesoFlujoM38`,
`errTramaFlujoM38` y `errEOFParcialM38`, también envueltos cuando `errors.Is`
los identifique. `errEOFSinMonotramaM38`, `errDatosPosterioresM38` y
`errUsoPosteriorEOFM38`, cualquier error desconocido, una combinación
imposible de resultado/estado, un receptor alterado o una causa escrita fuera
de S5 son fallos internos pegajosos: no se traducen a terminal funcional.

Los errores y mensajes nunca contienen nonce, PID, selector, identidad,
longitud, ticket, fragmento ni causa recibida.

## Prohibiciones comprobables

G4 y su grafo productivo no pueden:

- abrir, leer, duplicar o cerrar descriptores;
- crear procesos, ejecutar Bash o invocar `Start`, `Wait` o `Run`;
- usar `os`, `os/exec`, `syscall`, `unix`, `unsafe`, pidfd, `prctl` o `/proc`;
- instalar, observar, reenviar o generar señales;
- usar goroutines, canales, sincronización, contexto, reloj o plazos;
- escribir terminales, ficheros, logs, métricas o auditoría;
- llamar Docker, PostgreSQL, SQL, red, HTTP o DNS;
- validar el catálogo de selectores o la identidad del proceso real;
- consumir, entregar, comparar funcionalmente, serializar o registrar el
  ticket; solo se permite comprobar `ticket != ""` en la invariante
  estructural del constructor y retirar su cabecera al limpiar;
- codificar terminal, ACK, recibo o activar `--supervisar-m38`;
- crear otra autoridad de cuarentena, reutilización o caso siguiente;
- llamar primitivas Linux ya existentes en G2.

El bloque completo de imports de G4 admite exactamente y en este orden de
`gofmt`:

```go
import (
	"errors"
	"fmt"
	"strings"
)
```

`errors` sirve a centinelas y `errors.Is`. `fmt` y `strings` solo pueden ser
alcanzables desde la autoprueba. Las raíces productivas y sus ayudantes no
pueden llamar `fmt` o `strings`.

## Matriz mínima de autoprueba G4

### Positivos

- constructor desde una recepción O2a real, confirmada con EOF;
- nonce fijo exacto y ausencia de otra copia;
- `ARMAR` directo, todos los puntos de corte y byte a byte;
- `ARMAR+INICIAR`, `ARMAR+CANCELAR` y
  `ARMAR+INICIAR+CANCELAR` coalescidos;
- parcial de `CANCELAR` después de `INICIAR` que impide inicio pendiente;
- las cuatro causas válidas en S2 y en S3;
- EOF limpio en S1, S2 y S3;
- trama exacta más EOF en la misma llamada;
- contadores exactos con una, dos y tres tramas y con sobrante parcial;
- contador detenido tras `ARMAR+CANCELAR`, sin recorrer un sufijo enorme;
- entrada vacía sin EOF en S1, S2, S3 y con parcial L1;
- conservación íntegra del único sobre en S2/S3;
- retirada del sobre, lectores y buffers en S5;
- causa, estado y fase de origen exactos e inmutables;
- mutación posterior de cada fragmento del llamador sin efecto;
- llamada posterior a S5 con fragmento enorme, sin inspección ni reserva.

### Negativos e invariantes

- constructor nulo;
- receptor O2a nuevo, parcial, fallido, ya limpiado o con fase, lector, buffer
  o campos estructurales incoherentes;
- lector O2a no L3, buffer no cero, sobre vacío o campo ausente;
- `ARMAR` con nonce o PID distintos;
- `INICIAR` o `CANCELAR` antes de `ARMAR`;
- `ARMAR` repetido e `INICIAR` repetido;
- `CANCELAR` con nonce distinto;
- versión, clase, cardinalidad, causa o estado inválidos;
- EOF parcial de uno y varios fragmentos;
- NUL, CR, TAB, no ASCII y fronteras físicas 1024/1025;
- trama inválida seguida de otra válida;
- combinación física O1b imposible y error O1b desconocido;
- causa presente fuera de S5 o fase fuera de S1/S2/S3/S5;
- fallo interno después de haber consumido una trama, con tupla cero;
- primer fallo interno pegajoso y retirada sensible completa;
- cero cambio en modo `--supervisar-m38`, FD, procesos y residuos.

La matriz O1a, O1b y O2a sigue siendo obligatoria; G4 no duplica sus fixtures
para aparentar cobertura.

## Mutantes obligatorios

El mutador debe demostrar primero que la copia **sin modificar** de
G1+G2+G3+G4 compila, ejecuta la autoprueba y supera el control AST. Solo
después aplicará de uno en uno, identificará el cambio exacto y matará como
mínimo estos mutantes:

1. constructor de lector distinto de `CONTROL`, incluida la rama de error y
   retirada sensible que la línea base no puede inyectar;
2. receptor O2a no confirmado aceptado;
3. segunda copia completa del sobre;
4. copia o retención adicional de ticket o identidad;
5. nonce fijo omitido o duplicado;
6. `ARMAR` sin comparar nonce;
7. `ARMAR` sin comparar PID;
8. `CANCELAR` de S1 transportado en vez de protocolo;
9. `INICIAR` aceptado en S1;
10. `ARMAR` duplicado aceptado;
11. `INICIAR` que llama `Start` o salta a S4;
12. segundo `INICIAR` aceptado en S3;
13. nonce de `INICIAR` o `CANCELAR` omitido;
14. causa/estado normalizados a una pareja distinta;
15. señal interpretada mediante API del sistema;
16. EOF ignorado o convertido siempre en protocolo;
17. EOF parcial convertido en cancelación;
18. EOF de la misma entrega ocultado después de una trama;
19. causa primaria sobrescrita por EOF o segunda trama;
20. sobrante coalescido ignorado;
21. inicio pendiente devuelto con `CANCELAR` completa o parcial disponible;
22. error interno recuperable o con contador/resultado útil;
23. referencias sensibles conservadas después de S5;
24. getter, log o serializador del sobre incorporado;
25. modo operativo activado o proceso/FD incorporado.

No se considera muerto un mutante solo porque el árbol dejó de compilar por
omitir G4, por una sustitución no aplicada o por otro error anterior. La
evidencia registra, para cada mutante, patrón único aplicado, diff focal,
build completo G1--G4 y fallo esperado de autoprueba o AST.

## Control estructural reproducible

La evidencia incluirá un programa AST autocontenido y un mutador autocontenido
con estas comprobaciones:

1. imports positivos exactos de G4;
2. ausencia de las familias prohibidas en todo G4;
3. raíces productivas limitadas al constructor, `consumir` y sus ayudantes;
4. ninguna función de prueba alcanzable desde producción;
5. exactamente una construcción del lector `CONTROL`;
6. exactamente una copia fija de 64 bytes de nonce;
7. cero copia, conversión o exposición de ticket, identidad o sobre completo;
   la única comparación de ticket admisible es `ticket != ""` dentro de la
   validación estructural del constructor;
8. G2 y G3 invariantes byte a byte;
9. transiciones no terminales exactamente S1 -> S2 y S2 -> S3;
10. transiciones terminales exactamente S1 -> S5, S2 -> S5 y S3 -> S5,
    mediante los ayudantes únicos de causa o fallo;
11. fallo interno representado por S5, causa cero y `fallo != nil`;
12. ausencia de declaraciones o transiciones propias S0/S4/S6, permitiendo
    las referencias tipadas y la única asignación de retirada a O2a/S0;
13. ausencia de ACK, terminal, modo, FD, proceso o Bash en G4;
14. causas almacenadas solo desde literales canónicos;
15. limpieza obligatoria de receptor, sobre y lectores en toda entrada S5 o
    fallo interno.

Los artefactos portables quedarán en:

```text
docs/portal_vec/revisiones/evidencias/f0_h0b_c4b2_g2o_o2b_ast_<sha7>.go.txt
docs/portal_vec/revisiones/evidencias/f0_h0b_c4b2_g2o_o2b_mutantes_<sha7>.sh.txt
```

`<sha7>` es una metavariable documental y se sustituye, solo cuando exista el
candidato material, por sus siete primeros caracteres hexadecimales. Los
nombres definitivos no conservan los signos `<`/`>`. Los artefactos no
contienen rutas privadas. La revisión registra sus SHA-256 y los ejecuta desde
un worktree limpio del candidato.

## Write-set exacto

### Runner

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

Solo puede añadir G4 mediante cinco líneas legibles:

1. ruta G4;
2. SHA-256 G4;
3. comprobación nueva de G4 en el índice ordenado 5 del manifiesto;
4. asignación de la fuente privada G4;
5. par G4 en el bucle de acreditación.

La captura, cardinalidad, `go vet` y ambos builds incorporan G4 modificando
líneas existentes sin encadenar órdenes, minificar controles o superar una
longitud ilegible. También cambian las huellas esperadas de G1 y del binario.

El capturador ordena las rutas. El manifiesto de ocho debe quedar exactamente:

| Índice | Fuente |
| ---: | --- |
| 0 | `ruta_helper_sql`, D2c |
| 1 | `ruta_helper_h0b`, H0b |
| 2 | `ruta_adaptador_m38` |
| 3 | `ruta_helper_operativo`, D2d |
| 4 | `ruta_supervisor_m38`, G1 |
| 5 | `ruta_supervisor_m38_control_preinicio`, G4 |
| 6 | `ruta_supervisor_m38_operativo`, G2 |
| 7 | `ruta_supervisor_m38_sobre`, G3 |

### G1

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
```

Solo tres líneas legibles para llamar `autoprobarControlPreinicioM38()` desde
la autoprueba existente.

### G4 nueva

La ruta fijada arriba contiene producción O2b y su autoprueba focal.

### Invariantes byte a byte

G2, G3, capturador, adaptador M38, D2d, D2c, H0b, migraciones, SQL y
aplicación. El productor no modifica documentos transversales ni actas.

## Ledger físico

| Unidad | Base | Delta autorizado | Resultado/parada |
| --- | ---: | ---: | ---: |
| Runner | 794 | exactamente +5 | exactamente 799 |
| G1 | 689 | exactamente +3 | exactamente 692 |
| G2 | 798 | 0 | exactamente 798 |
| G3 | 431 | 0 | exactamente 431 |
| G4 | 0 | previsión 380--540 | revisar >540; parar >600 |
| D2d | 164 | 0 | exactamente 164 |
| Capturador | 799 | 0 | exactamente 799 |
| Adaptador M38 | 527 | 0 | exactamente 527 |
| D2c | 588 | 0 | exactamente 588 |
| H0b | 580 | 0 | exactamente 580 |
| Manifiesto | 7 | +1 | exactamente 8 |

G4 puede ocupar menos de 380 líneas si cumple todo; no se rellena. Superar 540
exige revalidar el contrato y el ledger antes de continuar. Superar 600,
necesitar una sexta línea del runner o tocar G2/G3 detiene sin commit y obliga
a una nueva decisión estructural.

Huellas materiales de la base:

| Unidad | SHA-256 |
| --- | --- |
| Runner | `fc8c27a3b6ef1651a9ef97676a4ca9d34924fa5dbb026921a7a1e529cda81176` |
| G1 | `f9ab7b20accac9af56cfcb5e42c25c62b087d7e0ee81a2fea09250a35fc0c58f` |
| G2 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| D2d | `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e` |
| Binario compuesto | `46d247156316b56ca5f30082c4964aaaefdc2442eb8fb8685f595aeb230dde30` |

El candidato registra antes de editar las huellas de todos los invariantes,
no solo las de esta tabla.

## Evidencia histórica O2a

Los programas AST y mutadores publicados de O2a son evidencia ligada al árbol
material `0caa140`, que aún solo contiene G1--G3. No se modifican ni se
recalculan después de añadir G4.

Al integrar O2b, dirección actualizará el acta O2a para indicar que su
reproducción histórica exige un checkout o worktree separado en `0caa140`.
Ejecutar el mutador O2a desde el futuro HEAD sin G4 en su lista de build no es
una reproducción válida y podría producir falsos mutantes muertos. El mutador
O2b evita ese defecto con una línea base positiva obligatoria G1--G4.

## Puertas previas a integración

- padre documental exacto publicado y CI verde;
- huellas y líneas base exactas antes de editar;
- `gofmt` y `go vet` sobre G1+G2+G3+G4;
- dos builds privados aislados, `-a -trimpath`, con SHA binaria idéntica;
- fuentes estables antes/después y manifiesto exacto de ocho;
- inventario `go list` correcto sin `-tags=ignore`;
- autoprueba completa y matriz negativa G4;
- línea base positiva, 25 mutantes muertos y AST verde;
- modo `--supervisar-m38` y modo desconocido todavía en 64;
- cero variación de FD, hijos, procesos y residuos;
- Bash y ShellCheck del runner;
- H0 PostgreSQL 18.4 completo sobre imagen fijada por digest;
- `git diff --check`, Gitleaks, pruebas globales, carrera y `go vet` global;
- `scripts/verificar_calidad.sh` verde;
- dos revisiones de código independientes con `P0=P1=P2=0`.

H0 solo acredita que el cableado privado de G4 no rompe la puerta existente;
O2b no incorpora Docker, PostgreSQL o SQL a su lógica.

## Paradas obligatorias

Detener sin confirmar si:

- HEAD no coincide con el padre completo comunicado por dirección;
- la base o su CI no están verdes;
- una huella material difiere antes de editar;
- cambia G2, G3 o cualquier invariante;
- runner no queda en 799, G1 en 692 o manifiesto en ocho;
- G4 supera 540 sin revisión o 600 en todo caso;
- se necesita más de cinco líneas nuevas en el runner;
- aparece una copia adicional del sobre, ticket o identidad;
- una causa conserva una cabecera recibida en vez de un literal canónico;
- un resultado de inicio oculta una trama completa o parcial suministrada;
- un error devuelve contador o resultado útil;
- sobrevive un mutante o la línea base positiva no está acreditada;
- aparece FD, proceso, Bash, señal, terminal, ACK, reloj, red o log;
- cambia el estado 64 del modo operativo;
- falla una puerta o una revisión emite `NO-GO`.

## Cierre limitado

O2b solo podrá acreditar:

```text
receptor O2a confirmado en S1
-> CONTROL incremental drenado hasta agotar la entrada o entrar en S5
-> ARMAR coherente en S2
-> INICIAR reconocido en S3, todavía sin Start
o
-> causa previa a Bash enclavada en S5 y referencias sensibles retiradas
```

No cierra creación de Bash, terminal, G2-O, C4b-2, H0b, F0, O4-05 ni
producción. No modifica F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` ni el `NO-GO` de producción.

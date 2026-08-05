# Enmienda F0-H0b/C4b-2 G2-O/O2a: recepción S0 y G3

Fecha: 5 de agosto de 2026.

Estado: **doble GO documental; autorización condicionada a publicación y CI
verde del commit que integre este contrato y su acta**.

Base material exacta del código:

```text
ef027cf2f94955cdf6de9c091c531a85d05c2e04
```

Esa base está publicada y su ejecución `31007941124` terminó con cinco de
cinco puertas verdes. O2a-P0 está integrado. El commit documental posterior
`c09b1533bafdcc38d22ec73f274800a4f0f685a4` no modifica los blobs materiales y
su ejecución `31009787813` también terminó con cinco puertas verdes.

El futuro commit que integre este contrato y sus dos GO será el **padre
obligatorio del candidato**. Su SHA completo solo existe tras confirmar:
dirección lo fijará en el mensaje de entrega al productor y la evidencia del
candidato lo registrará antes de editar. El productor debe partir de ese
commit; además, las huellas de todos los blobs materiales deben coincidir con
la base `ef027cf`. Las dos revisiones documentales independientes han concluido
con `P0=P1=P2=0`; no se autoriza código mientras el commit documental no esté
publicado y su CI no termine verde.

## Prevalencia

Esta enmienda sustituye para O2a, solo si obtiene el doble GO exigido:

- la especificación G2-O rechazada
  `especificacion_f0_h0b_c4b2_g2o_protocolo_operativo_2026-08-05.md`;
- la enmienda O0 rechazada
  `enmienda_f0_h0b_c4b2_g2o0_contrato_ledger_2026-08-05.md`;
- cualquier indicación que retrase el consumo de `SOBRE` hasta `ARMAR`;
- cualquier `ACK_LISTO` o `ACK_CASO`;
- cualquier agrupación antigua de S0--S2 en una sola minitarea O2.

Conserva como autoridades O1a corregida, el codec O1a, el lector O1b y la
preparación O2a-P0 integrados. La división vigente es:

```text
O2a  recepción y retención del SOBRE: S0 -> S1
O2b  ARMAR y cancelación sin Bash
```

No autoriza O2b, FD 9, procesos, Bash, ACK, `ARMAR`, terminales, señales ni la
activación de `--supervisar-m38`.

## Responsabilidad única

O2a añadirá una fuente G3 pura que:

1. recibe fragmentos de memoria y un indicador de EOF;
2. compone el lector O1b configurado exclusivamente para `SOBRE`;
3. espera una sola monoframa completa seguida de EOF limpio;
4. acepta únicamente `lecturaTramaFinalM38`;
5. retiene una sola representación privada del sobre validado;
6. cambia atómicamente de S0 a S1;
7. queda cerrada a cualquier segunda recepción.

O2a no lee el FD 9. Su futuro propietario entregará fragmentos a esta API en
otra minitarea.

El receptor no es concurrente: queda confinado a un único propietario que
serializa todas sus llamadas. «Atómicamente» significa que ningún estado
intermedio puede observarse al retornar de `consumir`; no implica primitivas
de sincronización ni autoriza acceso concurrente.

## Gramática heredada

```text
V1|SOBRE|NONCE|PID_RUNNER|SELECTOR|IDENTIDAD|LONGITUD_TICKET|TICKET\n
```

El codec O1a divide solo por los siete primeros separadores; el ticket puede
contener `|`. G3 no replica gramática, límites, conversión numérica ni catálogo.
Hereda:

- `NONCE` e `IDENTIDAD`: 64 hexadecimales minúsculos;
- `PID_RUNNER`: decimal canónico `1..2147483647`;
- `SELECTOR`: `NOMINAL` o una mayúscula ASCII y dos dígitos;
- `LONGITUD_TICKET`: decimal canónico `1..2048`, coincidente con el ticket;
- ticket limitado a bytes imprimibles `0x20..0x7e`;
- máximo físico: 4096 bytes incluido LF;
- máximo canónico real: 2212 bytes incluido LF.

El catálogo de selectores continúa siendo autoridad exclusiva del runner.

## Ruta y API privadas

Ruta nueva exacta:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go
```

G3 empieza exactamente con:

```go
//go:build ignore && linux && amd64

package main
```

`go list` debe mantener el capturador como único `GoFiles` y clasificar G1,
G2 y G3 dentro de `IgnoredGoFiles`. Se prohíbe compilar el paquete ordinario
con `-tags=ignore`.

API definitiva:

```go
type resultadoRecepcionSobreM38 uint8

const (
	recepcionSobreNecesitaDatosM38 resultadoRecepcionSobreM38 = iota
	recepcionSobreConfirmadaM38
)

type faseRecepcionSobreM38 uint8

const (
	recepcionSobreS0M38 faseRecepcionSobreM38 = iota
	recepcionSobreS1M38
)

type sobreRetenidoM38 struct {
	nonce          string
	pidRunner      string
	selector       string
	identidad      string
	longitudTicket string
	ticket         string
}

type receptorSobreS0M38 struct {
	lector *lectorTramaM38
	fase   faseRecepcionSobreM38
	sobre  sobreRetenidoM38
	fallo  error
}

func nuevoReceptorSobreS0M38() (*receptorSobreS0M38, error)

func (r *receptorSobreS0M38) consumir(
	fragmento []byte,
	fin bool,
) (consumidos int, resultado resultadoRecepcionSobreM38, err error)
```

El constructor invoca exactamente `nuevoLectorTramaM38("SOBRE")`. Un fallo
devuelve receptor nulo y conserva la identidad del error O1b. No se autoriza
otro constructor, interfaz, getter, callback, canal, serializador o método de
exposición.

## Separación de estados

O1b conserva sus estados físicos:

```text
L0 ABIERTO_VACIO
L1 ABIERTO_PARCIAL
L2 MONOTRAMA_ESPERANDO_EOF
L3 EOF_LIMPIO
L4 ERROR_TERMINAL
```

O2a solo posee:

```text
S0 ESPERAR_SOBRE
S1 ESPERAR_ARMAR
```

| Fase O2a | Estado O1b compatible |
| --- | --- |
| S0 | L0, L1 o L2 |
| S1 | L3 después de `TRAMA_FINAL` |

Invariantes:

- L0, L1 o L2 nunca implican S1;
- LF sin EOF conserva S0 y deja O1b en L2;
- solo `lecturaTramaFinalM38` permite S0 -> S1;
- retención completa y S1 forman una transición atómica;
- L4 no se representa como S6;
- `fallo != nil` es un enclavamiento interno, no una fase operativa;
- sin nonce confiable no hay terminal normal ni `FASE_ORIGEN`.

## Resultados y transiciones

| Entrada | O1b | O2a | Consumidos |
| --- | --- | --- | ---: |
| Vacía sin EOF | necesita datos, L0 | necesita datos, S0 | 0 |
| Parcial válida | necesita datos, L1 | necesita datos, S0 | longitud del fragmento |
| Completa sin EOF | necesita datos, L2 | necesita datos, S0 | longitud del fragmento |
| Completa con EOF | trama final, L3 | confirmada, S1 | longitud del fragmento |
| L2 más EOF vacío | trama final, L3 | confirmada, S1 | 0 |
| Error O1b durante S0 | L4 | fallo enclavado, sobre cero | 0 |
| Receptor nulo | no existe lector | error tipado, tupla cero | 0 |
| Invariante O2a incumplida | estado físico sin uso posterior | fallo enclavado, sobre cero | 0 |
| Uso posterior en S1 | L3 invariable | S1 y sobre invariables | 0 |

`lecturaTramaM38` y `lecturaEOFLimpioM38` son imposibles en una recepción
válida de `SOBRE`; observarlos produce fallo interno cerrado.

## Precedencia O2a

1. receptor nulo: error tipado y tupla cero;
2. fallo previo en S0: mismo error, sin inspeccionar la entrada;
3. fase S1: error estable de uso posterior, sin inspeccionar ni modificar el
   sobre;
4. delegación íntegra a O1b;
5. error O1b: enclavar exactamente ese error, limpiar retención y devolver
   tupla cero;
6. `NECESITA_DATOS`: exigir trama cero y permanecer en S0;
7. `TRAMA_FINAL`: comprobar estructura, retener y pasar a S1;
8. cualquier combinación imposible: error interno, limpieza y fallo cerrado.

La precedencia física continúa siendo la fijada por O1b: L4 devuelve su error;
L3 solo admite EOF vacío repetido; L2 rechaza cualquier byte; byte inválido
precede a exceso; capacidad precede a copia; el codec precede al EOF; y EOF
parcial no puede convertirse en cancelación. O2a es más estricta que O1b tras
aceptar: una llamada posterior a S1 no se delega a L3.

## Errores tipados y pegajosos

G3 puede definir solo:

```go
errReceptorSobreNuloM38
errInvarianteRecepcionSobreM38
errUsoPosteriorSobreM38
```

Reglas:

- los errores O1b se conservan sin decidir por su texto y `errors.Is` sigue
  funcionando;
- el primer error en S0 se conserva en `fallo` y las llamadas posteriores
  devuelven el mismo objeto;
- cualquier error devuelve contador y resultado cero;
- ningún error incluye nonce, identidad, selector, PID, longitud o ticket;
- un error S0 deja `sobreRetenidoM38{}` y no admite recuperación;
- usar S1 devuelve siempre `errUsoPosteriorSobreM38`, conserva S1 y el sobre;
- no se traduce a `CANCELADO`, `PROTOCOLO`, `INCIDENTE` o terminal normal.

La futura frontera operativa podrá terminar externamente en 65 si falla S0;
esa traducción no pertenece a G3.

## Retención opaca y privacidad

Tras `TRAMA_FINAL`, G3 exige clase `SOBRE`, cinco campos, ticket presente,
ninguna retención previa y fase S0. El único mapeo es:

```text
campos[0] -> nonce
campos[1] -> pidRunner
campos[2] -> selector
campos[3] -> identidad
campos[4] -> longitudTicket
ticket    -> ticket
```

O1a crea una cadena inmutable de la trama y `SplitN` produce subcadenas que
comparten ese único respaldo. Ese respaldo completo compartido cuenta como la
única representación material: puede quedar fijado por los seis campos, pero
G3 no lo almacena ni expone como trama completa. Se prohíbe conservar además
fragmentos del llamador, otra `tramaM38`, una reserialización, una segunda
copia material del ticket o datos para logs/errores. La asignación copia
cabeceras de cadenas, no sus bytes. La `tramaM38` temporal deja de conservarse
tras el mapeo.

No habrá getter, `String`, `MarshalJSON`, métrica con campos del sobre, log,
stdout, stderr, comparación ni interpretación del ticket. Mutar después el
`[]byte` original no altera lo retenido. El buffer fijo O1b queda limpio tras
éxito o error. No se promete borrado físico de cadenas inmutables porque Go no
permite acreditarlo.

## Límites y coste

G3 reutiliza el almacenamiento fijo O1b de 4096 bytes. Se prohíben
`append(buffer, fragmento...)`, copia completa, división global, `Scanner`,
reserva proporcional o recorrido de una cola enorme posterior a la
monoframa. El coste queda acotado por el prefijo útil, la capacidad fija y el
contador O1b.

## Prohibiciones comprobables de G3

G3 y su nuevo grafo de llamadas no pueden:

- abrir, leer, duplicar o cerrar FDs;
- crear procesos o ejecutar Bash;
- usar `os/exec`, `syscall`, `unix`, `prctl` o pidfd;
- usar señales, goroutines, canales, sincronización, reloj o plazos;
- acceder a `/proc`, argumentos o entorno;
- usar red, HTTP, DNS, Docker, PostgreSQL o SQL;
- escribir logs, auditoría, terminales o ficheros;
- interpretar selectores o comparar el PID con el proceso real;
- consumir o entregar el ticket;
- admitir `ARMAR`, `INICIAR`, `CANCELAR`, `ACK_LISTO` o `ACK_CASO`;
- activar `--supervisar-m38`.

La revisión estática se aplica al nuevo G3 y al grafo O2a; G1/G2 ya contienen
primitivas anteriores ajenas al corte.

El bloque de imports completo de G3 admite exactamente, en orden de `gofmt`:

```go
import (
	"errors"
	"fmt"
	"strings"
)
```

Cualquier import adicional detiene antes del build. `errors` pertenece a los
centinelas y comprobaciones `errors.Is`; `fmt` y `strings` solo pueden servir a
la autoprueba. Las raíces productivas y sus ayudantes nuevos no pueden llamar
`fmt` o `strings`.

## Matriz mínima de autoprueba G3

### Positivos

- constructor real y clase exacta;
- entrada vacía sin EOF;
- sobre canónico directo con EOF;
- sobre completo sin EOF y confirmación con EOF vacío;
- todos los puntos de corte y un byte por llamada;
- LF aislado sin EOF deja S0/L2 y devuelve necesita datos; el EOF posterior
  hace fallar O1b y enclava el error;
- ticket con `|`;
- selector `NOMINAL` y sintáctico `A00`;
- mínimo canónico exacto de 149 bytes y máximo canónico exacto de 2212 bytes,
  incluido LF;
- contadores exactos y S0 durante L0/L1/L2;
- transición única a S1 y conservación exacta de los seis valores;
- mutación del fragmento original sin alterar lo retenido;
- limpieza O1b tras aceptar y rechazo estable después de S1.

### Negativos

- EOF inicial o parcial de uno y varios fragmentos;
- sobre inválido en L2 rechazado al EOF;
- cualquier byte desde L2;
- sobre más byte, NUL, LF o segundo sobre, con y sin EOF;
- NUL, CR, TAB, `0x1f`, `0x7f` y `0x80`;
- fronteras físicas 4096/4097 y fragmentos enormes con/sin LF;
- versión, clase y cardinalidad incorrectas;
- nonce, identidad, PID, selector y longitud adversos;
- ticket vacío, hostil o con longitud incoherente;
- error seguido por sobre válido;
- error con contador, resultado o retención no cero;
- receptor nulo sin `panic`;
- resultado O1b imposible;
- doble recepción sin sobrescribir la primera.

La matriz O1a/O1b continúa obligatoria; no se elimina ni duplica
artificialmente.

## Mutantes y controles estructurales obligatorios

Los mutantes conductuales deben morir, como mínimo:

1. constructor `CONTROL`;
2. S1 al LF sin EOF;
3. EOF vacío inicial en L0 aceptado sin monoframa previa;
4. EOF parcial convertido en éxito/cancelación;
5. cola ignorada o segundo sobre truncado;
6. `TRAMA` o `EOF_LIMPIO` aceptados como final;
7. codec O1a omitido;
8. S1 fijado antes de retener;
9. campos intercambiados o ticket borrado;
10. solo clase o ticket retenidos;
11. alias mutable del fragmento;
12. L4 recuperable o error pegajoso sustituido;
13. error con contador o resultado útil;
14. segunda recepción que sobrescribe la primera;
15. catálogo local o comparación con PID real;
16. ARMAR, ACK, terminal o causa incorporados;
17. modo operativo distinto de 64.

Los mutantes se aplican a copias temporales; no se añade una interfaz de
inyección a producción.

Las prohibiciones no observables solo por salida se acreditan además mediante
dos comprobaciones reproducibles sobre AST y fuentes privadas:

1. **Fichero G3 completo**: lista positiva exacta de imports; ausencia de APIs
   de log, terminal, FD, proceso, red, reloj, serialización o getter; y ninguna
   función de prueba alcanzable desde las raíces productivas.
2. **Grafo productivo nuevo**, con raíces `nuevoReceptorSobreS0M38` y
   `(*receptorSobreS0M38).consumir`: una sola construcción de
   `sobreRetenidoM38`; ningún `string`, `[]byte`, `append`, `Clone` o conversión
   adicional del ticket; ninguna llamada a `fmt`, `strings`, autopruebas o
   ayudantes de prueba. El lector O1b ya revisado es dependencia terminal y no
   amplía el grafo nuevo al resto de G2.

`autoprobarSobreS0M38` y sus ayudantes quedan excluidos únicamente de las
reglas de construcción de fixtures y copia/retención de tickets. No quedan
excluidos de imports, APIs absolutamente prohibidas, falta de alcanzabilidad
desde producción ni del resto de puertas del fichero.

Las transformaciones adversas insertan de una en una una segunda copia en el
grafo productivo, un getter, un log y cada familia de import prohibido; cada
copia debe ser rechazada por el control estructural antes del build. La lista
exacta de transformaciones y sus huellas queda registrada en la evidencia del
candidato.

## Write-set exacto

### Runner

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

Solo puede declarar ruta/SHA de G3; capturar y manifestar la séptima fuente;
acreditarla antes/después; incluirla en `go vet` y ambos builds privados; y
actualizar SHA de G1, G3 y binario. Ninguna otra conducta.

### G1

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
```

Solo tres líneas legibles para invocar `autoprobarSobreS0M38()` desde la
autoprueba existente.

### G3 nuevo

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go
```

Contendrá producción O2a y su autoprueba focal.

### Límite del candidato material

El productor solo puede modificar runner, G1 y G3. Las actas pertenecen a los
revisores y los relevos, tablero y estado transversal a dirección después de
integrar. Ningún documento transversal forma parte del write-set del productor.

### Invariantes byte a byte

G2, capturador, adaptador M38, D2d, D2c, H0b, migraciones, SQL y aplicación.

## Ledger físico

| Unidad | Base | Delta autorizado | Resultado |
| --- | ---: | ---: | ---: |
| Runner | 783 | +11 | 794 |
| G1 | 686 | +3 | 689 |
| G2 | 798 | 0 | 798 |
| G3 | 0 | previsión 300--460 | 300--460 |
| Capturador | 799 | 0 | 799 |
| D2d | 164 | 0 | 164 |
| Adaptador | 527 | 0 | 527 |
| D2c | 588 | 0 | 588 |
| H0b | 580 | 0 | 580 |

G3 tiene parada dura de 500 líneas. Un resultado inferior a 300 es admisible
si cumple el contrato; no se rellena. Superar 460 exige revalidar el ledger;
superar 500 detiene el trabajo. El manifiesto pasa de seis a siete fuentes
exactas, con orden fijo y sin descubrimiento por glob.

Huellas materiales de la base:

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 783 | `da6871ca174890c85eb93ee4cfac15f32ecd1ac046d84d24fa68170ac34c52e9` |
| G1 | 686 | `9fab2cae4edd0b5cf8cd5d67fd7a1f9643b81085c815b0c10cb477f67a7e1afe` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| Capturador | 799 | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |
| D2d | 164 | `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e` |
| Adaptador | 527 | `98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7` |
| D2c | 588 | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | 580 | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| Binario compuesto | -- | `4ae175f326145be4f9cc81908bc3fa381abedc21576aaab1ade1ca8551284419` |

## Puertas previas a integración

- base material y padre documental exactos, con sus dos CI verdes;
- `gofmt` y `go vet` sobre G1+G2+G3;
- dos builds privados aislados con SHA binaria idéntica;
- fuentes estables antes/después y manifiesto de siete fuentes;
- inventario `go list`: solo capturador en `GoFiles` y G1/G2/G3 en
  `IgnoredGoFiles`, sin `-tags=ignore`;
- autoprueba, matriz negativa, mutantes y controles estructurales verdes;
- sin variación de FD ni hijos;
- `--supervisar-m38` y modo desconocido en 64;
- Bash y ShellCheck del runner;
- `git diff --check` y Gitleaks;
- pruebas globales, carrera y `scripts/verificar_calidad.sh`;
- H0 existente y residuos cero;
- dos revisiones de código independientes con `P0=P1=P2=0`.

H0 solo acredita que el cambio del runner no rompe la puerta existente; O2a
no incorpora Docker o PostgreSQL a su lógica.

## Paradas obligatorias

Detener sin confirmar si:

- HEAD no coincide con el SHA completo del padre documental comunicado por
  dirección y registrado en la evidencia del candidato antes de editar;
- cualquier blob material difiere de las huellas de `ef027cf` antes de editar;
- la CI de la base material o del padre documental no está verde;
- hace falta modificar G2 o cualquier invariante;
- runner no termina en 794, G1 en 689 o G2 en 798 líneas;
- G3 supera 460 sin revisión o 500 en todo caso;
- el manifiesto no tiene siete fuentes o aparece una octava;
- G3 no queda en `IgnoredGoFiles`, cambia el único `GoFiles` o aparece
  `-tags=ignore`;
- se minifica, encadena o retira una prueba para caber;
- se duplica gramática, límite o catálogo;
- existe más de una copia del sobre/ticket o se expone un dato;
- un mutante sobrevive o un error deja de ser pegajoso;
- S1 se alcanza antes de EOF;
- aparece FD, proceso, Bash, ACK, ARMAR, goroutine, reloj, red o log;
- cambia el estado 64 del modo operativo;
- falla una puerta o una revisión devuelve `NO-GO`.

## Cierre limitado

O2a solo acreditará:

```text
fragmentos de memoria
-> O1b SOBRE
-> EOF limpio
-> retención opaca única
-> S0 -> S1
```

No cerrará O2b, G2-O, C4b-2, H0b, F0, O4-05 ni producción. No modifica F0
`10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa productiva `1/14`
ni el `NO-GO` de producción.

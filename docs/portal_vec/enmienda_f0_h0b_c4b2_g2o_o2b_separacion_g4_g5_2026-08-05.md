# Enmienda O2b: separación estricta entre producción G4 y autoprueba G5

Fecha: 5 de agosto de 2026.

Estado: **doble `GO` documental final, `P0=P1=P2=0`**. La reanudación del
código sigue condicionada a confirmar y publicar esta enmienda con su acta y
a comprobar verde la CI de ese padre exacto.

## Motivo y trazabilidad de la parada

El contrato O2b publicado en `acf6016e63ce0b1458f75b7cd0f937c0f25327d8`
ordenaba detener sin confirmar si G4 superaba 600 líneas. El primer prototipo
formateado cumplió esa parada:

```text
rama: agent/f0-h0b-o2b-control-20260805
padre: acf6016e63ce0b1458f75b7cd0f937c0f25327d8
G4: 647 líneas
SHA-256 G4: fcddd5ba2e748e490f44fc0eee2f364f396c5056b02eca3613a1bee904702cd7
estado: sin seguimiento, sin commit
otros ficheros modificados: ninguno
```

La causa no es una ampliación funcional: el fichero mezclaba unas 343 líneas
de máquina y limpieza productivas con unas 304 líneas de autoprueba. No se
acepta elevar el límite para ocultar esa mezcla. Se separan las dos
responsabilidades y se mantiene íntegra la matriz exigida por el contrato
publicado.

El prototipo detenido no es candidato, evidencia ni base aceptada. Solo puede
usarse como material de trabajo después de publicar esta enmienda y comprobar
verde la CI de su padre exacto.

## Prevalencia limitada

Esta enmienda sustituye únicamente en el contrato O2b publicado:

- la afirmación de que G4 contiene también la autoprueba;
- el bloque de imports único de G4;
- las menciones al build `G1--G4`;
- el write-set, manifiesto y ledger de una sola fuente nueva;
- las paradas físicas asociadas a 799 líneas del runner, ocho entradas de
  manifiesto y 600 líneas de G4.

Conserva sin cambios la API, máquina S1--S5, propiedad del sobre, limpieza,
precedencias, contadores, matriz funcional y física, 25 mutantes, prohibiciones
y puertas del contrato `acf6016`. La división no permite reducir pruebas ni
trasladar lógica productiva al fichero de autoprueba.

## Fuentes y responsabilidad única

### G4: producción O2b

Ruta exacta:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go
```

Contiene exclusivamente los tipos, constantes, centinelas, constructor,
`consumir` y ayudantes productivos autorizados. Su bloque de imports exacto es:

```go
import "errors"
```

No declara fixtures, funciones `probar*`, `autoprobar*`, generadores de texto
ni ayudantes que solo sean alcanzables desde pruebas.

### G5: autoprueba O2b

Ruta exacta:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go
```

Empieza exactamente con:

```go
//go:build ignore && linux && amd64

package main
```

Su bloque de imports admite exactamente, en orden de `gofmt`:

```go
import (
	"errors"
	"fmt"
	"strings"
)
```

G5 contiene `autoprobarControlPreinicioM38` y todos sus fixtures y ayudantes.
No declara otra API productiva, no altera estado global productivo y no es
alcanzable desde `--supervisar-m38`, el ayudante interno ni ninguna raíz que no
sea `autoprobar`. `fmt` y `strings` quedan confinados al grafo de autoprueba.

G4 y G5 comparten el paquete privado solo para probar el comportamiento real,
sin interfaces falsas, duplicados de la máquina ni adaptadores de demostración.

`go list` debe mantener el capturador como único `GoFiles` y clasificar G1,
G2, G3, G4 y G5 como `IgnoredGoFiles`, sin `-tags=ignore`.

## Write-set exacto revisado

El productor solo puede modificar:

1. el runner PostgreSQL F0-H0b;
2. G1, únicamente para llamar la autoprueba;
3. la nueva G4 productiva;
4. la nueva G5 de autopruebas.

G2, G3, capturador, adaptador M38, D2d, D2c, H0b, migraciones, SQL,
aplicación y documentos permanecen byte a byte.

### Runner

El runner pasa de 794 a exactamente 802 líneas mediante ocho líneas legibles:

1. ruta G4;
2. ruta G5;
3. SHA-256 G4;
4. SHA-256 G5;
5. comprobación de G4 en el índice 5 del manifiesto;
6. comprobación de G5 en el índice 6;
7. par G4 en el bucle de acreditación;
8. par G5 en el bucle de acreditación.

Las asignaciones `fuente_g4` y `fuente_g5` se incorporan como dos palabras de
asignación simple en una línea ya existente; no se emplean `;`, tuberías,
condiciones ni órdenes encadenadas. La línea queda por debajo de 160
caracteres. El total nuevo es `2 + 2 + 2 + 2 = 8`. La captura, cardinalidad,
`go vet` y ambos builds
incorporan G4 y G5 modificando líneas existentes sin minificar ni encadenar
órdenes. Cambian las huellas esperadas de G1 y del binario compuesto.

El manifiesto de nueve queda exactamente:

| Índice | Fuente |
| ---: | --- |
| 0 | `ruta_helper_sql`, D2c |
| 1 | `ruta_helper_h0b`, H0b |
| 2 | `ruta_adaptador_m38` |
| 3 | `ruta_helper_operativo`, D2d |
| 4 | `ruta_supervisor_m38`, G1 |
| 5 | `ruta_supervisor_m38_control_preinicio`, G4 |
| 6 | `ruta_supervisor_m38_control_preinicio_pruebas`, G5 |
| 7 | `ruta_supervisor_m38_operativo`, G2 |
| 8 | `ruta_supervisor_m38_sobre`, G3 |

### G1

G1 conserva el límite exacto de tres líneas nuevas para invocar
`autoprobarControlPreinicioM38()` desde la autoprueba existente.

Los identificadores de runner son exactamente:

```text
ruta_supervisor_m38_control_preinicio
ruta_supervisor_m38_control_preinicio_pruebas
sha256_supervisor_m38_control_preinicio
sha256_supervisor_m38_control_preinicio_pruebas
fuente_g4
fuente_g5
```

## Ledger físico revisado

| Unidad | Base | Delta autorizado | Resultado/parada |
| --- | ---: | ---: | ---: |
| Runner | 794 | exactamente +8 | exactamente 802 |
| G1 | 689 | exactamente +3 | exactamente 692 |
| G2 | 798 | 0 | exactamente 798 |
| G3 | 431 | 0 | exactamente 431 |
| G4 producción | 0 | objetivo 300--400 | revisar >400; parar >420 |
| G5 autoprueba | 0 | objetivo 300--500 | revisar >500; parar >540 |
| D2d | 164 | 0 | exactamente 164 |
| Capturador | 799 | 0 | exactamente 799 |
| Adaptador M38 | 527 | 0 | exactamente 527 |
| D2c | 588 | 0 | exactamente 588 |
| H0b | 580 | 0 | exactamente 580 |
| Manifiesto | 7 | +2 | exactamente 9 |

No se rellena una fuente para alcanzar el objetivo inferior. Superar 400 en
G4 o 500 en G5 exige revisión explícita del ledger; superar 420 o 540,
necesitar una novena línea del runner o tocar un invariante obliga a detener
sin commit y volver a decidir la estructura.

## Pruebas y evidencia revisadas

Antes de cualquier mutante, la línea base completa
`G1+G4+G5+G2+G3` debe:

1. compilar y superar `go vet`;
2. producir dos binarios `-a -trimpath` idénticos;
3. superar la autoprueba completa O1a/O1b/O2a/O2b;
4. superar el control AST de separación G4/G5;
5. conservar en 64 los modos `--supervisar-m38` y desconocido.

Los 25 identificadores de mutante del contrato permanecen obligatorios. Cada
ejecución compila las cinco fuentes y mata el mutante por autoprueba o AST; un
fallo por omitir G5 no cuenta. El AST debe demostrar además:

- bloques de imports exactos comprobados por separado, con `fmt` y `strings`
  exclusivos de G5;
- ninguna función de G5 alcanzable desde producción;
- ninguna función de prueba declarada en G4;
- ninguna lógica productiva duplicada en G5;
- todas las prohibiciones productivas sobre el grafo de G4;
- G2 y G3 byte a byte y manifiesto real de nueve;
- una única construcción de lector CONTROL y una única copia fija del nonce,
  ambas en G4;
- limpieza y transiciones exactas del contrato original.

## Correcciones obligatorias del prototipo detenido

La separación mecánica no basta para autorizar el código. El siguiente
candidato debe corregir y probar, como mínimo:

1. la invariante de cada llamada activa vuelve a exigir nonce de 64 bytes y
   presencia de PID, selector, identidad, longitud y ticket del único sobre;
   vaciar cualquiera mediante un alias posterior al constructor produce fallo
   interno, tupla cero, limpieza completa y primer error pegajoso;
2. receptores S1 con cada campo ausente, nonce de longitud distinta, lector
   nulo o incoherente, buffer sucio, error previo y receptor ya retirado;
3. vacíos sin EOF en S1/S2/S3/L1, ARMAR e INICIAR repetidos, ARMAR en S3 y
   nonce distinto en INICIAR/CANCELAR desde S2 y S3;
4. `ARMAR+INICIAR+parcial(CANCELAR)`, EOF en la misma entrega desde S2/S3,
   cancelación completa antes de EOF y EOF parcial acumulado;
5. CR, TAB, no ASCII, límites 1024/1025, clase/cardinalidad/causa/estado
   inválidos y trama inválida seguida de otra válida;
6. los cuatro errores normalizables, directos y envueltos, los tres no
   normalizables, L3/L4 preexistentes y combinaciones físicas imposibles;
7. fallo interno después de progreso previo con tupla externa cero, mutación
   posterior de cada fragmento A/I/C, causa inmutable y conservación/retirada
   exacta del sobre en cada fase.

La autoprueba usa tablas y ayudantes focales para cubrir la matriz dentro del
ledger. No se eliminan casos para cumplir el límite y no se duplican las
pruebas ya autoridad de O1a, O1b u O2a.

Los nombres de evidencia conservan el patrón O2b ya publicado y registran las
huellas separadas de G4 y G5. Toda referencia a build `G1--G4` se interpreta,
para el candidato posterior a esta enmienda, como build completo
`G1+G4+G5+G2+G3`.

## Reanudación y paradas

Solo se reanuda el candidato después de:

1. dos revisiones documentales independientes con `P0=P1=P2=0`;
2. commit y publicación de esta enmienda y sus actas;
3. CI de cinco puertas verde sobre ese commit;
4. avance por fast-forward de la rama candidata al nuevo padre exacto;
5. verificación de árbol sin cambios salvo el prototipo G4 no confirmado;
6. separación física inmediata de ese prototipo antes de seguir programando.

Se detiene sin confirmar ante cualquier incumplimiento del contrato original o
de esta enmienda, pérdida de cobertura, import productivo no autorizado,
acceso desde G5 fuera de autoprueba, ledger distinto, mutante superviviente o
revisión no verde.

## Efecto limitado

Esta separación mejora mantenibilidad y revisión, pero no cierra O2b ni ningún
hito funcional. Las métricas permanecen F0 `10/23`, O4-05 `3/5`,
Contratación temporal `24/46`, Bolsa productiva `1/14` y producción `NO-GO`.

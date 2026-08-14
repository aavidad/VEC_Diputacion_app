# Revisión funcional O3a V5 CND: estados causales con Go 1.26.6

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-ESTADOS-CAUSALES-GO1.26.6`.

Dictamen: **NO-GO funcional, `P0=0`, `P1=1`, `P2=0`**.

El candidato distingue correctamente las siete salidas agregadas de TUPLA y
las ramas exteriores de C16--C18, conserva sus predicados y cierra el falso
verde detectado en la escritura CONTROL de C16. Sin embargo, permanece el
mismo patrón de falso verde en la primera escritura CONTROL de la vuelta
tardía C18. Un fallo de esa escritura puede retornar `nil` y convertir C18 en
éxito exterior, en vez de devolver el estado causal 111. El corte no cumple
todavía su único criterio de atribución fiable.

Este dictamen no evalúa ni acredita estabilidad del conductor, el parche de
toolchain, O4, publicación, CI remota o producción. En particular, una corrida
verde nueva no revoca el P1 durable de C21 bajo el conductor completo.

## Independencia, lectura y write-set revisor

La revisión se realizó en el worktree exclusivo
`/srv/fabrica/revisiones/o3a-v5-cnd-estados-causales-go1.26.6-funcional-20260814`,
rama
`revision/o3a-v5-cnd-estados-causales-go1.26.6-funcional-20260814`, creada
directamente en el SHA objetivo. La rama y el worktree productores
permanecieron limpios y no se editaron, movieron, integraron ni rebasaron.

Antes de editar se leyeron completos `AGENTS.md`, los cuatro documentos
obligatorios de relevo/mapa/tablero, el expediente RRHH normalizado, la hoja de
ruta y la matriz normativa. También se revisaron íntegros la decisión O3a, su
enmienda V5, la revisión final V5, el parche Go 1.26.6, sus dos actas `NO-GO`
y la enmienda candidata.

El único write-set revisor es esta acta. No se modificaron los seis ficheros
del candidato, código productivo, conductores, evidencias históricas,
workflow, SQL, estado transversal, porcentajes ni credenciales.

## Identidad y alcance del candidato

| Propiedad | Valor reproducido |
| --- | --- |
| Candidato | `a997acbb9a51c02168f1648225c0e69c84e648d4` |
| Padre exacto | `74f249587d5c0da41092a2c017ccd6cb54817248` |
| Árbol | `38473c53cd1838113e549bf53233136a7e1da3cd` |
| Distancia | un commit lineal; padre antepasado directo |
| Delta | seis rutas, `+211/-25`, sin cambio de modo |
| Limpieza productora | limpia en el SHA exacto |

El write-set contiene exactamente:

| Ruta | Delta | Líneas | SHA-256 final |
| --- | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go` | `+33/-6` | 742 | `4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7` |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go` | `+22/-13` | 741 | `084e5363ec969ef705caed7dcc213b5d7a54483574cf18bc2b0382fd6cccadf5` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_go1_26_6_2026-08-14.md` | `+150/-0` | 150 | `2c3688fce80014027a5b3192ff0fd944579e3cd963843e9669d3ad151c6b8064` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `+2/-2` | 11 | `96907cac6ef96c9c7bc4770385cfd054c638039739d459c9ea51d5988b5b5ceb` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `+2/-2` | 23 | `2191a7b06e50da65df36e58bd8403a6bec6110f4b5dd229a272f5cb6172c29b5` |
| `tools/o3c_p6_conductor/fuentes.tsv` | `+2/-2` | 33 | `4f60e1ec8e3c09a928f56eba2fce0a6d26c5fb331776117b19bd858562f4e7ef` |

Los dos Go conservan `//go:build ignore && linux && amd64`: son test-only.
No cambia ninguna fuente productiva. En los tres ledgers solo cambian las dos
entradas correspondientes; el resto es byte a byte idéntico al padre. G7a y
G7b quedan respectivamente en 742 y 741 líneas, por debajo de la parada local
750 y del tope duro 800 de DEC-051.

## Auditoría semántica

Los quince estados nuevos forman el intervalo cerrado 99--113 y no colisionan
con los códigos anteriores 64--98 ni con el estado 124 de `timeout`. Todos son
fallos test-only; ninguno coincide con el éxito esperado por los conductores.

La comparación del diff con el padre confirmó:

- TUPLA conserva el orden `netpoll → snapshot inicial → preparación → avance
  → clase/origen → limpieza → snapshot final → cardinal FD → hijos`;
- separar los `||` conserva el cortocircuito previo: resultado inválido no
  ejecutaba limpieza y error de snapshot final impedía consultar cardinalidad
  e hijos;
- C16 conserva alias preparado antes de alias de retirada;
- C17 conserva testigos antes de inventario de hijos;
- C18 conserva parcial antes de plazo y vuelta tardía solo si ambos bordes son
  verdes;
- no cambian estados esperados, oráculos, 100 procesos C21, cardinalidades,
  plazos, watchdogs, conductores ni lógica productiva;
- los estados de preparación 77, 78 y 93--98 y los códigos contractuales 0,
  65 y 72--76 permanecen intactos;
- la corrección de C16 ahora asigna a `err` el resultado de
  `escribirControlPruebaO3aM38` antes de devolverlo, por lo que ese fallo ya no
  puede aparecer como éxito.

La sonda C16 directa bajo `flock` queda expresamente descartada: su estado 66
era ambiental, causado por el FD 3 del candado heredado sin `CLOEXEC`.
`conductor.sh` cierra todo FD mayor o igual que 3 antes de ejecutar cada
bloque. Este dictamen no usa esa sonda como defecto ni como evidencia causal.

## Hallazgo P1: C18 conserva un falso verde de escritura

En
`supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go`,
líneas 727--728 del candidato, `probarVueltaTardiaExternaO3aM38` contiene:

```go
if err = prepararFixtureO3aM38(f); err != nil || escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|") != nil {
    return err
}
```

Si la preparación termina verde y falla `escribirControlPruebaO3aM38`, el
segundo operando hace verdadera la condición, pero `err` continúa siendo
`nil`. La función retorna `nil`; `ejecutarCasosLinealesExternosO3aM38` puede
terminar con estado 0, en lugar de atribuir el fallo al estado 111
`estadoVueltaTardiaExternoO3aM38`.

Es la misma clase de defecto que el candidato sí corrige en
`probarAliasRetiradaExternaO3aM38`. Aunque el patrón ya existía en el padre,
queda dentro de C18, que este corte modifica y declara causal. Por ello el
contrato no puede afirmar que la siguiente ejecución canónica localizará con
fiabilidad el primer predicado fallido.

Corrección mínima exigida: separar la preparación y la primera escritura,
asignar ambas a `err` y retornar el error de escritura. Debe mantenerse el
estado exterior 111, sin añadir retry, espera, tolerancia, exclusión ni nuevo
estado. La corrección cabe dentro de la parada G7b=750 y requiere repetir los
ledgers vivos, gates y doble revisión sobre un SHA nuevo. No se corrige el
productor desde esta acta.

## Evidencia y puertas

Se validaron con `sha256sum -c` las tres evidencias durables del productor:

| Evidencia | Resultado y huellas principales |
| --- | --- |
| O3a V5 Go 1.26.6 | `GO`, 14/14 bloques, 74 registros, FD 5→5, residuos cero; resumen `c6a6a854…`, manifiesto `7245d0c4…`, C21 race `ee530c3f…`, sumas `5825ffc2…` |
| O3b P7 Go 1.26.5 | `GO`, 234 casos y 6 BF; resumen `0d018ec1…`, casos `bac12a92…`, O17 `602341bc…`, sumas `d40a0f37…` |
| O3c P6 Go 1.26.5 | `GO`, 244 casos y 6 BF; resumen `75398451…`, casos `87388777…`, BF `73d34dab…`, sumas `e7c666bd…` |

La reproducción funcional propia usó un clon limpio del candidato propiedad
de `orquesta` y serialización mediante
`flock --close /srv/fabrica/proyectos/VEC_Diputacion_app/.sec-toolchain-review-gates.lock`:

- O3a V5 con Go 1.26.6: `GO`, 14/14, 74 registros, FD 4→4 y residuos cero;
  C21 race completó 100/100 y su sidecar tuvo el mismo SHA-256
  `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91`
  del productor;
- O3b P7 con Go 1.26.5: `GO`, 234 casos, seis BF, 100 capturas normales y
  100 race, residuos cero; el resumen fue byte a byte el mismo
  `0d018ec1c8a6f73259c87e912b8892fdfc46f41b1cab5ad4459abbecddc69784`;
- O3c propio no llegó a ejecutar un caso: esperó el lock mientras otro corte
  corría O3a y se canceló antes de crear evidencia. Su evidencia durable
  exacta sí quedó íntegramente verificada como se indica arriba. Con el P1
  estático ya demostrado, otra puerta pesada no podía cambiar el dictamen.

También terminaron verdes:

- `gofmt -d` sobre las diez fuentes: cero bytes;
- `go vet` sobre las diez fuentes;
- build normal `CGO_ENABLED=0 -trimpath`;
- build race `CGO_ENABLED=1 -race -trimpath`;
- `git diff --check` del rango exacto;
- genealogía, modos, líneas, SHA-256, enlaces locales y limpieza de ambos
  worktrees.

Las primeras invocaciones O3a del revisor abortaron antes de crear evidencia
o ejecutar casos por errores del comando de revisión —PATH sin `go`, HOME no
escribible y destino sin permiso—. Se corrigió el entorno y se ejecutó una sola
corrida contractual; no hubo retry de ningún caso ni búsqueda de verde.

La calidad global del productor quedó sellada verde, incluidos normal, race,
vet, build, grafos y `govulncheck`. Gitleaks quedó sellado verde tanto para el
commit candidato —un commit, 9,72 KiB— como para el rango acumulado —22
commits, 178,28 KiB—. El binario Gitleaks no estaba instalado en este entorno
revisor; esos dos barridos no se repitieron y se registran como evidencia
aportada, no como reproducción propia.

## Dictamen y relevo

El candidato recibe **NO-GO funcional, `P0=0`, `P1=1`, `P2=0`**. El cambio es
acotado, test-only, conserva oráculos y presupuestos y mejora sustancialmente
la atribución; el único bloqueo es el falso verde C18 descrito.

Dirección debe asignar una corrección mínima sobre un SHA nuevo y someterla a
revisión funcional y de seguridad independientes. Ni este candidato ni una
corrida verde nueva acreditan estabilidad o el parche Go 1.26.6. El P1 C21
durable de `74f2495` permanece abierto hasta que una ejecución canónica causal
identifique el propietario y una corrección posterior sea revisada.

No se hizo push, integración, despliegue, cambio de producción, credenciales,
seguridad, estado transversal ni métricas.

## Comandos principales

```bash
git rev-parse HEAD HEAD^ 'HEAD^{tree}'
git merge-base --is-ancestor 74f249587d5c0da41092a2c017ccd6cb54817248 a997acbb9a51c02168f1648225c0e69c84e648d4
git diff --name-status 74f249587d5c0da41092a2c017ccd6cb54817248..a997acbb9a51c02168f1648225c0e69c84e648d4
git diff --numstat 74f249587d5c0da41092a2c017ccd6cb54817248..a997acbb9a51c02168f1648225c0e69c84e648d4
git diff --check 74f249587d5c0da41092a2c017ccd6cb54817248..a997acbb9a51c02168f1648225c0e69c84e648d4
gofmt -d FUENTES_O3A
go vet FUENTES_O3A
CGO_ENABLED=0 go build -trimpath FUENTES_O3A
CGO_ENABLED=1 go build -race -trimpath FUENTES_O3A
flock --close LOCK runuser -u orquesta -- tools/o3a_v5_conductor/conductor.sh CLON EVIDENCIA_NUEVA
flock --close LOCK runuser -u orquesta -- env GOTOOLCHAIN=go1.26.5 tools/o3b_p7_conductor/conductor.sh CLON EVIDENCIA_NUEVA
(cd EVIDENCIA && sha256sum -c SHA256SUMS)
```

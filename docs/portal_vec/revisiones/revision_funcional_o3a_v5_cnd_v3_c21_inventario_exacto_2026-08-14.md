# Revisión funcional O3a V5 CND V3: inventario FD exacto de C21

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-V3-C21-INVENTARIO-EXACTO`.

Dictamen: **NO-GO funcional, `P0=0`, `P1=1`, `P2=0`**.

El cambio estático cumple su criterio acotado: sustituye la igualdad de dos
cardinalidades por igualdad exacta de un mapa `FD -> tupla`, conserva orden,
estados y cierre seguro, y el mutante de sustitución con cardinalidad constante
muere con estado 104 en normal y race. Sin embargo, una de las puertas
obligatorias del propio contrato terminó roja en su única ejecución: O3c P6
falló en `CAP_NORMAL_021`. Ese rojo no se compensa con O3a/O3b verdes, con el
mutante ni con otra muestra; el SHA candidato no puede recibir GO.

Este dictamen se limita a V3. No acredita estabilidad O3a, el toolchain Go
1.26.6, O4, publicación, CI remota ni producción.

## Independencia, lectura y write-set

La revisión se realizó en el worktree exclusivo
`/srv/fabrica/revisiones/o3a-v5-cnd-v3-c21-inventario-exacto-funcional-20260814`,
rama
`revision/o3a-v5-cnd-v3-c21-inventario-exacto-funcional-20260814`, creada
directamente desde el candidato exacto. La rama productora
`trabajo/o3a-v5-cnd-v3-c21-inventario-exacto-20260814` y su worktree estaban
limpios y no se editaron, movieron, integraron ni rebasaron.

Antes de editar se leyeron completos `AGENTS.md`, los documentos obligatorios
de relevo, mapa, tablero y contratación temporal, la especificación RRHH y la
matriz normativa. También se revisaron la decisión O3a íntegra, la autoridad
V5, las enmiendas causales V1 y V2, sus actas funcionales y de seguridad, y la
[enmienda V3 candidata](../enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md).

El único write-set revisor es esta acta. No se modificaron las cinco rutas
candidatas, el productor, conductores, código productivo, workflows,
evidencias históricas, PostgreSQL, estado transversal, métricas ni
credenciales.

## Identidad y alcance exactos

| Propiedad | Valor reproducido |
| --- | --- |
| Candidato | `fea52f3ddf796991c93c85cae992ca695db1ae63` |
| Padre directo | `ec02febaa3080cb8e7894340075beb850ea54348` |
| Árbol candidato | `c9d5a393f39d48a5015848d5fd49a0b69959c622` |
| Distancia | un commit lineal; el padre es antepasado directo |
| Delta | cinco rutas, `+127/-16`, sin cambios de modo |
| Limpieza productora | limpia en el SHA exacto |

El write-set candidato es:

| Ruta | Delta | Líneas | Bytes | SHA-256 final |
| --- | ---: | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go` | `+21/-13` | 750 | 22607 | `2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md` | `+103/-0` | 103 | 4605 | `10e1eda8a6a741326be9fb53c2566b655ae316b4af819c224f7b4f368a8a2753` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `+1/-1` | 11 | 1554 | `ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `+1/-1` | 23 | 4400 | `238e019a8fe31c7de830e174a4308083452ef22ceb54999ca4a8aeff5aac5f4f` |
| `tools/o3c_p6_conductor/fuentes.tsv` | `+1/-1` | 33 | 6421 | `5603a529fadbe424ccbb17a9be98dda58871bb31c3168a83048b61ab3a47064f` |

Los cinco objetos son blobs ordinarios `100644`. G7a conserva
`//go:build ignore && linux && amd64`, por lo que es test-only. G7b queda byte
a byte en 744 líneas y SHA-256
`5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e`.
No cambia fuente productivo ni conductor.

Los tres ledgers solo sustituyen la fila de G7a. La comprobación de todas sus
entradas contra el árbol candidato dio 10/10 en O3a, 22/22 en O3b y 32/32 en
O3c. Ninguna fuente ajena cambió de huella o cardinalidad.

## Auditoría semántica del inventario

`inventarioFDVivosPruebaO3aM38` devuelve un `map[int][10]uint64`. La clave es
el número de FD y el valor fija, en orden, `F_GETFD`, `F_GETFL`, `Dev`, `Ino`,
`Rdev`, `Mode`, `Uid`, `Gid`, `Nlink` y `Size`. Clave y array son comparables;
`maps.Equal` exige el mismo conjunto de números y la misma tupla completa para
cada uno.

La secuencia observada es:

1. `os.ReadDir("/proc/self/fd")` termina y cierra su descriptor efímero;
2. cada nombre se convierte estrictamente a entero;
3. `F_GETFD` confirma vida; únicamente `EBADF` omite una entrada ya obsoleta;
4. cualquier otro error de `F_GETFD`, `F_GETFL` o `Fstat` retorna
   `errInventarioO3aM38`;
5. un FD confirmado incorpora la tupla sin abrirlo, duplicarlo, convertirlo en
   `*os.File` ni cambiar sus flags.

La ventana de carrera entre las consultas no abre el caso: si el FD deja de
ser consultable después de `F_GETFD`, la función falla cerrada. El inventario
es exacto respecto de la tupla contractual, no una afirmación sobre
identidades internas del kernel que esos diez campos no exponen.

En `ejecutarTuplaExternaO3aM38` se conserva el orden `netpoll -> inventario
inicial -> preparación -> avance -> clase/origen -> limpieza -> inventario
final -> igualdad -> hijos`. Un fallo inicial sigue siendo estado 100; un
fallo final, 103; una desigualdad estructurada, 104; hijos restantes, 105. Los
estados 99--113 no cambian y ningún estado negativo es aceptado como éxito por
los conductores.

El envoltorio `contarFDVivosPruebaO3aM38` ahora devuelve `len(inventario)` y
continúa siendo consumido desde G7b por `ejecutarSelectorExternoO3aM38`. Se
auditó ese alcance indirecto: en el camino verde conserva la cardinalidad y
en fallo solo endurece la lectura mediante `F_GETFL/Fstat`; no introduce
tolerancia, retry ni éxito nuevo. G7b, sus estados y sus oráculos permanecen
sin modificación textual.

El único import nuevo es `maps`, biblioteca estándar disponible en las
toolchains autorizadas. La fuente test-only no entra en el grafo productivo.
G7a queda exactamente en la parada local de 750 líneas y por debajo del tope
duro 800 de DEC-051; cualquier corrección posterior en ese fichero exige una
separación previa o una decisión expresa de presupuesto.

## Mutante determinista

Se contrastó el mutante efímero en
`/srv/fabrica/revisiones/o3a-v5-cnd-v3-inventario-mutante-v2-20260814`.
Difiere de G7a candidato solo por siete líneas: después de la limpieza cierra
el FD 1 y abre `/dev/null` con `O_CLOEXEC`, exigiendo que conserve el mismo
número. La cardinalidad queda constante, pero identidad/flags cambian.

| Artefacto | SHA-256 |
| --- | --- |
| G7a mutado | `8fbf6220318444f97d3aea669aea4bc7329145f2a31c106480eade04443a90be` |
| Binario normal | `bdd71e17883c4d7794136765d81695df87f520103900d512e58b3897e9da026c` |
| Binario race | `8dd990ba67c7f15322ae20b23f6ea3a9ebfa520324f082694775520040209515` |

La reproducción funcional propia ejecutó una sola vez cada binario desde el
checkout exacto propiedad de `orquesta`, con el candado cerrado antes del
proceso. Normal y race devolvieron estado 104; stdout y stderr fueron cero en
ambos. Los cuatro ficheros de salida tienen SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

## Gates verdes reproducidos

La compilación focal propia de los diez fuentes, serializada mediante
`flock --close /srv/fabrica/proyectos/VEC_Diputacion_app/.sec-toolchain-review-gates.lock`,
terminó verde:

- `gofmt -d`: cero bytes;
- `go vet` sobre los diez fuentes;
- build normal `CGO_ENABLED=0 -trimpath`;
- build race real `CGO_ENABLED=1 -race -trimpath`;
- `git diff --check ec02feb..fea52f3`;
- genealogía, modos, líneas, hashes, ledger y enlace local del documento;
- Gitleaks v8.30.0 del candidato, un commit y 6,12 KB, sin fugas;
- Gitleaks acumulado `5345d5d..fea52f3`, 24 commits y 189,21 KB, sin fugas.

La evidencia O3a productora en
`/srv/fabrica/revisiones/evidencia-o3a-v5-cnd-v3-inventario-fea52f3-r1`
validó íntegramente sus 21 entradas de `SHA256SUMS`. Registra Go 1.26.6, GO
14/14, 74 casos, FD 4->4 y residuos cero. C21 contiene 100/100 filas GO tanto
normal como race, todas con estado/stdout/stderr cero. Sus huellas principales
son:

| Artefacto O3a | SHA-256 |
| --- | --- |
| `resumen.txt` | `d81a1995132b40c83fe572d6072e26fd9017ac6a389631c2f1878ecde41727c2` |
| `casos.ndjson` | `e7b33a2dd6e8c2f7fcefde80c94814e6885ba006f0aee1126b51b368b743e000` |
| `manifiesto.tsv` | `e5aba4e06f6ff75583c94feb76e4dc14ff0b22773b4dec0436e63a04b8316398` |
| cada sidecar C21 | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `36e0d01cc361e4ffd1461cc104bc7a18fee9a1c1fac6afaeb024ae663097f773` |

La evidencia O3b productora en
`/srv/fabrica/revisiones/evidencia-o3b-p7-v3-inventario-fea52f3/r1`
validó sus seis entradas: Go 1.26.5, GO, 234 casos, 100+100 capturas, seis BF
directos y residuos cero. `resumen.txt` tiene SHA-256
`cecf22bc5065ef893d78b5792e26be0f1733fd9b992a633f735b322fe8246af2`
y `SHA256SUMS`,
`99e6adc183f0ce0dea7b605f0369157bf1681f7debf158720bd6d7359f480ce4`.

## Hallazgo P1: la puerta O3c obligatoria terminó roja

La única ejecución productora de O3c P6 sobre este SHA terminó:

```text
caso=CAP_NORMAL_021
modo=normal
estado=1
stdout=197
stderr=0
grupo=si
inventario=5/5,0/0,0/0,0/0,0/0
resultado=NO-GO
```

Es una puerta expresamente requerida porque O3c comparte G7a y su ledger vivo.
El estado no es un delta FD ni un residuo de grupo: es el fallo del caso
normal, con salida no vacía. Atribuir su causa exige la evidencia de esa
primera ejecución; no corresponde repetir el caso para buscar verde ni
inferir que O3a/O3b lo compensan.

El destino
`/srv/fabrica/revisiones/evidencia-o3c-p6-v3-inventario-fea52f3` está vacío.
La inspección del conductor confirma que construye la evidencia en staging y
solo la mueve al destino después de completar todas las filas en GO; su trap
elimina staging al primer error. Por ello fue posible verificar el registro
terminal exacto aportado para esta revisión, pero no recalcular hashes ni
leer los 197 bytes de stdout. Esta limitación probatoria no reduce el rojo ni
autoriza una repetición.

El hallazgo es P1, no P0, porque afecta una puerta test-only y no una autoridad
productiva; tampoco es P2 porque incumple directamente el criterio de cierre
del candidato. La acción siguiente es conservar y diagnosticar el primer rojo
O3c mediante el registro externo original si sigue disponible, o asignar un
corte nuevo que haga durable la evidencia de fallo antes de cualquier nueva
ejecución. Esta acta no modifica el productor ni el conductor.

## Puertas omitidas y límites

No se repitieron O3a, O3b ni O3c. O3a/O3b ya tenían una única evidencia
exacta con sumas verificables; O3c tenía un rojo contractual que por mandato no
se repite. Tampoco se ejecutaron una ráfaga adicional, `go test ./...`,
`go test -race ./...`, Docker, PostgreSQL, E2E o CI remota: no pueden convertir
en GO el gate O3c ya fallido y no son proporcionales a este delta test-only.

El P1 histórico de estabilidad C21 y el NO-GO del parche de toolchain siguen
abiertos. Una muestra O3a verde con el inventario nuevo solo demuestra
compatibilidad; no revoca la ejecución roja histórica ni acredita Go 1.26.6.
No se abre O4 ni ninguna dependencia posterior.

## Comandos principales reproducidos

```text
git rev-parse HEAD HEAD^ HEAD^{tree}
git merge-base --is-ancestor ec02febaa3080cb8e7894340075beb850ea54348 fea52f3ddf796991c93c85cae992ca695db1ae63
git diff --name-status HEAD^..HEAD
git diff --numstat HEAD^..HEAD
git diff --check HEAD^..HEAD
git ls-tree HEAD -- RUTAS_CANDIDATAS
wc -l -c RUTAS_CANDIDATAS
sha256sum RUTAS_CANDIDATAS
gofmt -d FUENTES_O3A
flock --close LOCK bash -c 'go vet FUENTES; CGO_ENABLED=0 go build -trimpath FUENTES; CGO_ENABLED=1 go build -race -trimpath FUENTES'
BINARIO_MUTANTE --autoprueba-o3a-caso TUPLA_C
(cd EVIDENCIA_O3A && sha256sum -c SHA256SUMS)
(cd EVIDENCIA_O3B && sha256sum -c SHA256SUMS)
go run github.com/zricethezav/gitleaks/v8@v8.30.0 git . --no-banner --redact --no-color --log-opts=RANGO
```

## Relevo

El candidato exacto recibe **NO-GO funcional, `P0=0`, `P1=1`, `P2=0`**.
El inventario estructurado es conforme en su alcance, pero la puerta O3c roja
impide cerrar V3. Dirección debe asignar diagnóstico/corrección sobre un corte
nuevo y someterlo de nuevo a revisión funcional y de seguridad independientes;
no debe reintentar este SHA ni integrar por mayoría.

No se hizo push, integración, despliegue, cambio de producción, credenciales,
seguridad, estado transversal ni métricas.

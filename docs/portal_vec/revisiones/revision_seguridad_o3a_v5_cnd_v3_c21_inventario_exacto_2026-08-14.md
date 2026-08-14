# Revisión de seguridad O3a V5 CND V3: inventario exacto de C21

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3A-V5-CND-V3-C21-INVENTARIO-EXACTO`.

Dictamen: **NO-GO de seguridad, `P0=0`, `P1=1`, `P2=0`**. El inventario
estructurado y su mutante focal cumplen el criterio acotado, pero la única
corrida contractual O3c P6 del candidato terminó roja. La autoridad V3 exige
esa puerta y prohíbe repetirla para buscar verde. No se atribuye el rojo a una
causa sin evidencia ni se compensa con O3a u O3b verdes.

## Identidad y alcance congelado

La revisión se realizó sobre el candidato exacto
`fea52f3ddf796991c93c85cae992ca695db1ae63`, cuyo padre directo es
`ec02febaa3080cb8e7894340075beb850ea54348` y cuyo árbol es
`c9d5a393f39d48a5015848d5fd49a0b69959c622`. La distancia es un único
commit lineal. La rama productora
`trabajo/o3a-v5-cnd-v3-c21-inventario-exacto-20260814` se comprobó limpia y
no fue modificada. Los cinco objetos candidatos son blobs ordinarios `100644`.

| Ruta candidata | Delta | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go` | `+21/-13` | 750 | `2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3` |
| `docs/portal_vec/enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md` | `+103/-0` | 103 | `10e1eda8a6a741326be9fb53c2566b655ae316b4af819c224f7b4f368a8a2753` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `+1/-1` | 11 | `ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `+1/-1` | 23 | `238e019a8fe31c7de830e174a4308083452ef22ceb54999ca4a8aeff5aac5f4f` |
| `tools/o3c_p6_conductor/fuentes.tsv` | `+1/-1` | 33 | `5603a529fadbe424ccbb17a9be98dda58871bb31c3168a83048b61ab3a47064f` |

El delta total es `+127/-16`. G7a es test-only por `//go:build ignore`; queda
en la parada local de 750 líneas y por debajo del tope duro 800. G7b permanece
byte a byte en 744 líneas y SHA-256
`5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e`.
Los tres ledgers fueron contrastados contra todas sus fuentes vivas: 10/10,
22/22 y 32/32 huellas y conteos coinciden. Sus únicos cambios son la huella y
las líneas de G7a; los tres conductores permanecen byte a byte.

El único write-set revisor es esta acta. No se editaron el productor, código
productivo, conductores, workflow, evidencias históricas, PostgreSQL, estado
transversal, métricas ni credenciales.

## Lectura y criterio de seguridad

Antes de editar se leyeron completos `AGENTS.md`, relevo, mapa, tablero,
expediente, hoja de ruta y matriz normativa; la decisión O3a íntegra, la
autoridad V5, la enmienda de estados causales, V2 C18 y la enmienda V3. Se
revisó además el código completo que implementa y consume C21 y las puertas
O3a, O3b y O3c aplicables.

El criterio acotado de V3 es eliminar el falso verde por mera cardinalidad:
los mapas inicial y final deben tener los mismos números de FD y una huella
estructurada igual para cada número. V3 no identifica la causa del rojo C21
histórico, no corrige estabilidad y no cambia conducta productiva.

## Auditoría del inventario

`inventarioFDVivosPruebaO3aM38` opera de forma cerrada:

1. `os.ReadDir("/proc/self/fd")` obtiene la lista y cierra su descriptor antes
   de devolverla;
2. cada nombre debe convertirse íntegramente a entero;
3. `F_GETFD` acredita que la entrada siga viva y obtiene `FD_CLOEXEC`;
4. únicamente `EBADF` permite omitir una entrada obsoleta de procfs;
5. para todo FD vivo, `F_GETFL` y `Fstat` deben terminar verdes;
6. cualquier otro fallo devuelve `errInventarioO3aM38`, que los llamadores
   convierten en un estado negativo y que ningún conductor acepta como GO.

La huella comparable es
`{FD flags, status flags, dev, ino, rdev, mode, uid, gid, nlink, size}` y la
clave es el número de FD. `maps.Equal` exige igualdad de cardinalidad, claves y
diez campos. Esto detecta un FD añadido o retirado y también una sustitución
de igual cardinalidad por otra identidad física, tipo, modo o flags. No se
abre, duplica ni convierte en `*os.File` ningún descriptor examinado; tampoco
se mutan flags, posiciones, custodia o permisos.

La fotografía no es una primitiva atómica del kernel: hay una ventana entre
la enumeración, `F_GETFD`, `F_GETFL` y `Fstat`. Es segura para el caso
contractual porque C21 ejecuta cada entrega en un proceso aislado, calienta
netpoll antes de la fotografía inicial y toma la final después de la limpieza,
sin productor concurrente de FD en esos bordes. No debe reutilizarse como
prueba de identidad frente a aperturas concurrentes ajenas. Asimismo, una
reapertura del mismo inode con los mismos flags y metadatos es
observacionalmente indistinguible para esta tupla; V3 acredita identidad
física conforme a su contrato, no identidad de la instancia `open file
description` para un adversario concurrente.

La fotografía inicial se toma antes de preparar la fixture y la final después
de `limpiarFixtureO3aM38`; por ello el intervalo incluye íntegramente el efecto
y su limpieza. El orden preexistente comprueba el error de fotografía inicial
después de intentar preparar la fixture. Ese orden no convierte el fallo en
GO: devuelve el estado 100 y el proceso aislado termina, por lo que el kernel
retira sus recursos. Es un límite del auxiliar, no una autorización ni una
filtración introducida por V3.

## Consumo indirecto y privacidad

`contarFDVivosPruebaO3aM38` reutiliza ahora el inventario y devuelve su
cardinalidad. G7b, aunque permanece byte a byte, consume ese auxiliar en sus
barreras. El cambio indirecto solo añade los fallos cerrados de `F_GETFL` y
`Fstat`; no altera orden de casos, códigos de éxito, oráculo, plazo, cleanup,
cardinalidad esperada o código productivo. Los gates O3a y O3b sellados son
compatibles con ese consumo.

El mapa vive solo en memoria del proceso test-only. No usa `readlink`, no lee
contenido de archivos y no incluye rutas, nombres, comandos, datos personales
ni secretos. Ninguna tupla se imprime o persiste. El estado 104 es la única
observación exterior de una desigualdad y stdout/stderr deben permanecer
vacíos. No aparecen nuevos `Print`, log, `Wait`, señal, sleep, fallback,
goroutine, red o syscall productiva.

## Mutante focal de igual cardinalidad

Se inspeccionó el mutante efímero preservado fuera de Git en
`/srv/fabrica/revisiones/o3a-v5-cnd-v3-inventario-mutante-v2-20260814`. Tras
la limpieza canónica cierra exclusivamente stdout y abre `/dev/null` con
`O_WRONLY|O_CLOEXEC`, exigiendo que ocupe el mismo número. La cardinalidad se
conserva, pero la huella física cambia.

| Artefacto | SHA-256 |
| --- | --- |
| G7a mutado | `8fbf6220ce905056f4b75448418dfd48a9b10c11bc2a78136b17b2e7f75f4818` |
| Binario normal | `bdd71e17883c4d7794136765d81695df87f520103900d512e58b3897e9da026c` |
| Binario race | `8dd990ba67c7f15322ae20b23f6ea3a9ebfa520324f082694775520040209515` |

La reproducción independiente válida se hizo una vez por modo, como usuario
`orquesta`, desde un clon exacto de runtime y cerrando antes todos los FD
mayores o iguales que 3. Normal y race devolvieron exactamente estado 104;
stdout y stderr fueron cero. Los cuatro ficheros de salida tienen SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
No se repitió una puerta canónica roja.

## Evidencia O3a y O3b

La evidencia productora O3a está en
`/srv/fabrica/revisiones/evidencia-o3a-v5-cnd-v3-inventario-fea52f3-r1`.
Su `SHA256SUMS` valida íntegramente y declara Go 1.26.6, GO 14/14 bloques, 74
casos, C21 normal/race 100+100, FD 4→4 y residuos cero.

| Artefacto | SHA-256 |
| --- | --- |
| `resumen.txt` | `d81a1995132b40c83fe572d6072e26fd9017ac6a389631c2f1878ecde41727c2` |
| `manifiesto.tsv` | `e5aba4e06f6ff75583c94feb76e4dc14ff0b22773b4dec0436e63a04b8316398` |
| C21 índices normal | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| C21 índices race | `ee530c3fa1d78a2467529bcfec729ad12edee28c28c640f4be055f7ed0fb2c91` |
| `SHA256SUMS` | `36e0d01cc361e4ffd1461cc104bc7a18fee9a1c1fac6afaeb024ae663097f773` |

La evidencia O3b está en
`/srv/fabrica/revisiones/evidencia-o3b-p7-v3-inventario-fea52f3/r1` y su
`SHA256SUMS` también valida íntegramente: Go 1.26.5, GO, 234 casos, 100+100
capturas, seis BF directos con estado 65, salida cero y residuos cero.

| Artefacto | SHA-256 |
| --- | --- |
| `resumen.txt` | `cecf22bc5065ef893d78b5792e26be0f1733fd9b992a633f735b322fe8246af2` |
| `casos.tsv` | `79c06bba100844851c80cd7d8625303d5182d15fa17c2c67a16a6f319843e732` |
| BF directos | `573e56278149447e8e8267f8d24ff700b764f63d5498d8ad163ac6aa5929408c` |
| `SHA256SUMS` | `99e6adc183f0ce0dea7b605f0369157bf1681f7debf158720bd6d7359f480ce4` |

## Hallazgo P1: puerta O3c roja

La única ejecución O3c P6 terminó:

```text
NO-GO caso=CAP_NORMAL_021 modo=normal estado=1 stdout=197 stderr=0 grupo=si inventario=5/5,0/0,0/0,0/0,0/0
```

Es una puerta obligatoria porque O3c consume G7a y su ledger vivo. Estado 1 no
es GO; el grupo seguía presente y el primer inventario era 5/5. Los demás
inventarios registrados fueron 0/0. Conforme a la cláusula de parada, no se
ejecutó un retry ni se usó mayoría o una corrida de otro conductor como
compensación.

El conductor escribe primero en staging y solo mueve el destino final después
de superar todos sus casos. Al cortar en `CAP_NORMAL_021`, el trap retiró el
staging y el destino
`/srv/fabrica/revisiones/evidencia-o3c-p6-v3-inventario-fea52f3` quedó vacío.
No existe log durable externo ni copia de los 197 bytes de stdout. Por tanto la
evidencia acredita inequívocamente la puerta roja, pero no permite conocer el
contenido de la salida ni atribuir causalidad a inventario, fixture, caso,
entorno u otra rama. Esta revisión no inventa ese contenido ni imputa a V3 una
causa no demostrada.

El P1 se cierra únicamente con una minitarea propietaria que preserve de forma
durable la salida del primer caso rojo, diagnostique `CAP_NORMAL_021` y aporte
un candidato corregido. La siguiente ejecución O3c debe ser única sobre ese
nuevo SHA; repetir `fea52f3` para buscar verde contradiría su autoridad.

Una sonda diagnóstica posterior y no compensatoria queda preservada en
`/srv/fabrica/revisiones/o3c-p6-cap021-selectores-fea52f3-r1`. Ejecutó una vez
cada uno de los siete selectores: los tres positivos devolvieron 0 y los cuatro
negativos 65, todos con stdout/stderr cero, PGID ausente, FD 4→4, hijos,
zombis y grupos cero. Sin embargo, su propio `runtime_tmp` acumuló una entrada
por selector, de 0→1 hasta 6→7, y el runner diagnóstico terminó NO-GO. No
reproduce el agregado `CAP_NORMAL_021`, no recupera los 197 bytes perdidos y
no cambia el dictamen. Sus huellas son: manifiesto
`700b6b6140a6e4bd87b0f85c2f7b42cded96adf47f603907eb7aefc4cfc71aae`,
runner `2eff0f7913db2dde88e75f8074c4544c264ff4cc7ab7a9de425c3f55feb3efaa`
y `SHA256SUMS`
`4e4f211e76b5b16f3796f0bdb692763e327cc8ed5abd1a7523f0ab4f86000131`.
No se repitió esa sonda.

## Gates revisores

La reproducción focal verificó identidad, parent, árbol, ancestro, un solo
commit, modos, líneas, hashes y ledgers completos. `gofmt` no produjo diff;
con Go 1.26.6 pasaron `go vet`, build normal `CGO_ENABLED=0` y build race
`CGO_ENABLED=1` de los diez fuentes O3a. Los binarios se descartaron fuera del
repositorio. `git diff --check` quedó verde.

Gitleaks v8.30.0 terminó sin fugas en el delta `ec02feb..fea52f3` (un commit,
6,12 KB) y en el rango acumulado `5345d5d..fea52f3` (24 commits, 189,21 KB).
La búsqueda focal no halló secreto, token, clave privada, DSN, credencial ni
ruta sensible nueva. El worktree no conserva binarios ni residuos.

Comandos principales reproducidos:

```text
git rev-parse HEAD HEAD^ HEAD^{tree}
git merge-base --is-ancestor ec02febaa3080cb8e7894340075beb850ea54348 fea52f3ddf796991c93c85cae992ca695db1ae63
git diff --name-status HEAD^ HEAD
git diff --numstat HEAD^ HEAD
git diff --check HEAD^ HEAD
git ls-tree -r HEAD -- RUTAS_CANDIDATAS
wc -l RUTAS_CANDIDATAS
sha256sum RUTAS_CANDIDATAS
gofmt -d FUENTES_O3A
GO1_26_6 vet FUENTES_O3A
CGO_ENABLED=0 GO1_26_6 build -trimpath -o /dev/null FUENTES_O3A
CGO_ENABLED=1 GO1_26_6 build -race -trimpath -o /dev/null FUENTES_O3A
(cd EVIDENCIA_O3A && sha256sum -c SHA256SUMS)
(cd EVIDENCIA_O3B && sha256sum -c SHA256SUMS)
BINARIO_MUTANTE --autoprueba-o3a-caso TUPLA_C
gitleaks git . --no-banner --redact --no-color --log-opts=RANGO
```

O3c no se repitió. Calidad global, PostgreSQL, Docker, E2E, publicación y CI
remota son N/A al write-set test-only/documental y no pueden volver verde una
puerta proporcional roja.

## Rectificación y límites

No se reutiliza el estado 66 de una antigua sonda C16 directa bajo `flock`:
quedó acreditado que heredaba el FD 3 del candado sin `CLOEXEC`, a diferencia
del conductor completo que cierra todos los FD mayores o iguales que 3. Esa
sonda ambiental está retirada del razonamiento.

La corrida O3a verde de V3 no revoca el P1 histórico de estabilidad sellado
sobre el conductor completo de `74f2495`, y la mejora causal C18 de `ec02feb`
no acredita C21. Este dictamen tampoco acredita toolchain Go 1.26.6, O4,
publicación, CI, producción o despliegue.

## Relevo

El candidato recibe **NO-GO de seguridad, `P0=0`, `P1=1`, `P2=0`** por su
puerta O3c obligatoria roja. No hay hallazgo adicional de privilegio,
privacidad o fail-open en el inventario estructurado dentro de su entorno
controlado. El productor no debe corregir este acta ni reejecutar el mismo SHA;
dirección debe asignar diagnóstico/corrección O3c con evidencia durable y una
nueva revisión independiente. Este revisor no integra ni autoacredita. No se
autoriza push, despliegue, producción, credenciales ni cambio de métricas.

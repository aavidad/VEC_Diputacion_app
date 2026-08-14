# Revisión funcional del parche de seguridad Go 1.26.6 para CI

Fecha: 13 de agosto de 2026.

Identificador: `SEC-TOOLCHAIN-PATCH-1.26.6-REVISION-FUNCIONAL`.

Estado: **NO-GO, P0=0, P1=1, P2=1**.

Este dictamen independiente revisa exclusivamente el candidato
`74f249587d5c0da41092a2c017ccd6cb54817248`, hijo exacto de
`d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c`, con árbol
`bdbd0b2b05c43d7c14230a5c9b0a3114f44a4bb6`. El parche eleva correctamente
la toolchain de entrega y elimina las vulnerabilidades alcanzables observadas,
pero una ejecución independiente y sellada de la puerta activa O3a V5 falló de
forma no determinista en C21 bajo `-race`. Dos pases funcionales posteriores
no borran ese fallo. El corte no puede acreditarse mientras la misma CI pueda
terminar verde o roja sobre los mismos bytes y el mismo entorno autorizado.

No se revisan ni acreditan O4, sus candidatos materiales o descendientes,
integración, publicación, CI remota, producción, despliegue ni métricas.

## Independencia, lectura y write-set

La revisión se realizó en el worktree exclusivo
`sec-toolchain-go1.26.6-funcional-20260813`, rama
`revision/sec-toolchain-go1.26.6-funcional-20260813`, creada directamente en
el SHA objetivo. La rama productora
`trabajo/sec-toolchain-go1.26.6-ci-20260813` permaneció limpia y no fue
editada, movida, integrada ni rebasada.

Antes de editar se leyó `AGENTS.md` completo y toda la lectura obligatoria:
relevo de sesión, mapa/roadmap, tablero, relevo de contratación temporal,
matriz normativa, expediente normalizado y hoja de ruta RRHH. Se auditó
además la enmienda candidata completa, el workflow, el Dockerfile, el
conductor y sus oráculos, las puertas PostgreSQL y los verificadores de
artefactos y calidad.

El único write-set revisor es esta acta. No se modificaron los cinco ficheros
del candidato, código, pruebas, workflows, SQL, autoridades históricas,
estado transversal, porcentajes ni credenciales.

## Identidad y write-set del candidato

`git rev-parse`, `merge-base` y `ls-tree` confirmaron commit, padre, árbol,
ancestry y modos. El delta contiene exactamente cinco rutas:

| Ruta | Estado/delta | Líneas | SHA-256 final |
| --- | ---: | ---: | --- |
| `Dockerfile` | `+1/-1` | 253 | `6045bd15bf061de0db0e70c3858344b48db39fc31dbe9fc1f76c6e397f81bf5d` |
| `docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md` | `+168/-0` | 168 | `377d5eb7e8393ca05c7f13f1c85ac88abb7ee45e6a38e33761824e07ce0b242b` |
| `go.mod` | `+1/-1` | 45 | `a447f8cf5217130ed336bf78bcd93211659e1c7ba547673e9a1753ef6ba7eeca` |
| `tools/o3a_v5_conductor/README.md` | `+3/-2` | 242 | `7933b7617a92e22fddf6f888b9fddcc74ea0224616e0340d9c9f36d7d2533ce8` |
| `tools/o3a_v5_conductor/conductor.sh` | `+1/-1` | 205 | `cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7` |

Total: `+174/-5`. El modo del conductor continúa `100755`; las otras cuatro
rutas son `100644`. No cambian `.github/workflows/ci.yml`, `go.sum`, fuentes
Go, fixtures, ledgers, evidencias históricas, PostgreSQL ni ningún oráculo de
las cinco puertas CI. `git diff --check` es verde.

## Contrato de toolchain y cadena de suministro

| Requisito | Resultado |
| --- | --- |
| Compatibilidad mínima | `go.mod` conserva `go 1.25.12`; `GOTOOLCHAIN=go1.25.12 go version` resolvió y ejecutó `go1.25.12 linux/amd64`. |
| Toolchain de entrega | `go.mod` fija `toolchain go1.26.6`; `go version` devolvió `go1.26.6 linux/amd64`. |
| `setup-go` | Los dos usos activos están fijados a `actions/setup-go@924ae3a...` y leen `go-version-file: go.mod`. El código exacto de esa acción V6 prioriza la directiva `toolchain` y después exporta `GOTOOLCHAIN=local`; selecciona 1.26.6, no el mínimo 1.25.12. |
| Docker | La única etapa Go usa `golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`. |
| OCI oficial | `docker buildx imagetools inspect` reprodujo el índice anterior y el manifiesto `linux/amd64` `sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976`. |
| SBOM | SPDX JSON: 14.058.287 bytes, SHA-256 `84f32382aea8aef3f57e450df50d771579960b1c16ab9a782e30fece18633f5c`; contiene `pkg:golang/stdlib@1.26.6` y `pkg:generic/go@1.26.6`. |
| Provenance | SLSA JSON: 55.072 bytes, SHA-256 `d10d3493e1911c7cce542dfb1fb8cbca9e90315d0fada3add0b94600c432ff1c`; contiene `GOLANG_VERSION=1.26.6` y descarga amd64 SHA-256 `708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89`. |
| Autoridad oficial Go | `go.dev` registra 1.26.6 estable, publicada el 13/08/2026, con correcciones de seguridad en `crypto/tls`, `encoding/asn1`, `encoding/xml`, `net/http` y `net/url`, entre otros. |
| Conductor activo | Falla cerrado salvo `go version go1.26.6 ...`; conserva fuentes, hashes, normal, `-race`, watchdogs, inventarios, resultados y checksums. |
| Historia | Las fijaciones 1.26.5 restantes pertenecen a herramientas/evidencias congeladas O3a AST, O3b y O3c o a la genealogía del README; ninguna sustituye la selección de toolchain de las cinco rutas CI activas. Cero ficheros históricos de evidencia cambiaron. |

`go mod verify`, `bash -n` y ShellCheck fueron verdes. `go mod tidy -diff`
devuelve la misma reclasificación de `github.com/nkiri/xls` en base 1.26.5 y
candidato 1.26.6: ambos diffs tienen SHA-256
`30979b90b3562891f6e7921c78f39cca11c2031a049ca5ac7b72f5a742463ed9`
y `cmp` confirma identidad. Es deuda preexistente, no una mutación ni una
relajación introducida por este parche.

## Vulnerabilidades

La reproducción actual sobre el padre con Go 1.26.5 terminó en estado 3. La
base de vulnerabilidades vigente encontró seis vulnerabilidades alcanzables de
la biblioteca estándar:

| ID | Paquete | Corregida en |
| --- | --- | --- |
| `GO-2026-6218` | `net/url` | Go 1.26.6 |
| `GO-2026-6090` | `crypto/tls` | Go 1.26.6 |
| `GO-2026-6089` | `net/http` | Go 1.26.6 |
| `GO-2026-6088` | `encoding/xml` | Go 1.26.6 |
| `GO-2026-5972` | `encoding/asn1` | Go 1.26.6 |
| `GO-2026-5026` | `net/http` | Go 1.26.6 |

Sobre el candidato, el mismo comando
`go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...` devolvió
`No vulnerabilities found`. La puerta no se eliminó, silenció ni sustituyó.

La enmienda enumera exactamente cinco hallazgos históricos, pero una
reproducción sin versión/fecha durable de la base devuelve ahora seis. Esta
deriva no debilita la necesidad ni la eficacia del parche, pero sí hace
irreproducible el conteo exacto documentado; se clasifica como P2 y debe
corregirse mediante un corte temporal/base de datos durable o una redacción
que distinga la corrida histórica del inventario vigente.

## Puerta canónica, artefactos y PostgreSQL

Todas las cargas pesadas se serializaron con
`flock /srv/fabrica/proyectos/VEC_Diputacion_app/.sec-toolchain-review-gates.lock`
y se ejecutaron en un clon limpio del SHA exacto, propiedad del usuario de
runtime no privilegiado.

`scripts/verificar_calidad.sh` terminó verde e incluyó:

- formato y verificación de módulos;
- `go test ./... -count=1 -timeout 20m`;
- `go test -race ./... -count=1 -timeout 30m`, incluido
  `internal/vec/ports` en 477,687 s;
- `go vet ./...` y `go build ./cmd/...`;
- grafos público/interno y sus autopruebas negativas;
- manifiestos web, carga TLS, unitarias Python, `govulncheck`, tamaños y
  `git diff --check`.

Las superficies se construyeron con los targets `runtime-publico` y
`runtime-interno`. BuildKit resolvió el índice Go 1.26.6 fijado. Los dos
verificadores concluyeron:

```text
Artefactos productivos publico e interno aislados y conformes con sus manifiestos.
Artefactos productivos fallan cerrados y sin revelar configuracion.
```

Las dos puertas PostgreSQL 18.4 fijadas por digest terminaron verdes:

```text
OK: registro ContextoActor/PDP V3 PostgreSQL 18
Integracion PostgreSQL 18 de Bolsa publica superada con TLS verificado.
```

Incluyeron ACL, concurrencia, replay, retiradas/downs, rollback, TLS y limpieza.
El clon quedó limpio y no quedaron contenedores de prueba.

## Hallazgo P1: conductor O3a V5 no determinista

La reproducción funcional hizo dos ejecuciones completas, nuevas e
independientes del conductor sobre el mismo clon propietario exacto. Ambas
terminaron `GO`, 14/14 bloques, 74 casos, normal y `-race`, FD 5→5, residuos
cero y `SHA256SUMS` válidos:

| Corrida | `sha_casos` | `sha_manifiesto` | Resultado |
| --- | --- | --- | --- |
| funcional R1 | `2e1f9c2f2a7c7674eaa97aa24b8c966a4f768df6ffd9aaa4720688ccd55893ae` | `506e4dde5ea8e9ed855d32f7ad8fd43ec3e12e0efcfe926526716beb3371d064` | GO |
| funcional R2 | `3a2a582c6edb55e4492ce57882b54a56d6efb08b409ea860e6e3d29e91bf3195` | `5695d712c8de4dcaaeee3d112e64bf357f412790ae34518fe69f950c23a5950d` | GO |

Sin embargo, el contraste independiente de seguridad sobre otro clon limpio,
propietario, con el mismo HEAD, Go 1.26.6 y sin carga competidora produjo
evidencia durable `NO-GO`. Esta revisión leyó y validó sus checksums:

```text
resultado=NO-GO
head=74f249587d5c0da41092a2c017ccd6cb54817248
sha_conductor=cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7
sha_fuentes=fc654291ffda8c9dae2ef690267b842568082644cfe2d384d718238ac85d492a
bloques=14
casos_registrados=74
fd_conductor_inicio=5
fd_conductor_fin=5
residuos=cero
```

El único bloque rojo fue `c15_c21` en modo `race`: C15, C16, C17 y C18
pasaron; `C21_CIEN_INVENTARIOS` falló en la entrega 44 de `TUPLA_C`, con
esperado 0, estado 66, stdout 0 y stderr 0. El manifiesto, el sidecar de cien
índices, el resumen y el resto de artefactos pasan `sha256sum -c`.

La CI ejecuta exactamente una corrida del conductor. Por tanto, dos pases no
pueden convertir el fallo sellado en ruido descartable: con el mismo candidato
la puerta puede bloquear o permitir la publicación según planificación. El
README conserva además el antecedente de una evidencia V25 revocada por
intermitencia C21, de modo que este borde era conocido y exigía estabilidad,
no reintento.

Clasificación: **P1**. No es una exposición directa ni una relajación de
seguridad, pero invalida el criterio verificable de una CI determinista y la
evidencia candidata que presenta el conductor como verde.

Corrección exigida: aislar la causa de estado 66 en `TUPLA_C` bajo `-race`,
obtener un candidato nuevo con estabilidad proporcional y conservar el
oráculo C21, sus cien entregas, el modo `-race`, el corte en el primer índice,
los watchdogs y la prohibición de reintento. No se acepta ocultarlo con
retries, mayor tolerancia, exclusión de C21 o reclasificación del estado.

## Secretos, diff y comandos reproducidos

Gitleaks recorrió el commit candidato (`d16c0f6..74f2495`) y los 21 commits
acumulados (`5345d5d..74f2495`) sin filtraciones. Un barrido adicional de
directorio encontró 59 coincidencias históricas tanto en el árbol padre como
en el candidato; la igualdad exacta demuestra que no pertenecen al delta ni
al rango acumulado revisado. No se introdujeron credenciales ni se alteró la
configuración de Gitleaks.

Comandos principales:

```bash
git rev-parse HEAD HEAD^ HEAD^{tree}
git merge-base --is-ancestor d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c 74f249587d5c0da41092a2c017ccd6cb54817248
git diff --name-status d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c..74f249587d5c0da41092a2c017ccd6cb54817248
git diff --numstat d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c..74f249587d5c0da41092a2c017ccd6cb54817248
git diff --check d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c..74f249587d5c0da41092a2c017ccd6cb54817248
go version
GOTOOLCHAIN=go1.25.12 go version
go mod verify
go mod tidy -diff
bash -n tools/o3a_v5_conductor/conductor.sh
shellcheck tools/o3a_v5_conductor/conductor.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
scripts/verificar_calidad.sh
tools/o3a_v5_conductor/conductor.sh CLON_PROPIETARIO EVIDENCIA_NUEVA
docker build --target runtime-publico -t PUBLICO .
docker build --target runtime-interno -t INTERNO .
scripts/verificar_contenido_artefactos_productivos.sh PUBLICO INTERNO
scripts/probar_fallo_cerrado_artefactos_productivos.sh PUBLICO INTERNO
VEC_POSTGRES_TEST_IMAGE=postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296 \
  bash deploy/postgresql/autorizacion/probar_integracion_contexto_actor_v3.sh
VEC_POSTGRES_TEST_IMAGE=postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296 \
  bash deploy/postgresql/bolsa_publica/probar_integracion.sh
gitleaks git . --no-banner --no-color --redact --config .gitleaks.toml \
  --log-opts='d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c..74f249587d5c0da41092a2c017ccd6cb54817248'
gitleaks git . --no-banner --no-color --redact --config .gitleaks.toml \
  --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..74f249587d5c0da41092a2c017ccd6cb54817248'
```

## Dictamen y relevo

El cambio de versión, el digest, la compatibilidad mínima, la eliminación de
vulnerabilidades, la calidad global, los artefactos y PostgreSQL son
correctos. No se debe revertir a Go 1.26.5 ni relajar `govulncheck`.

El candidato exacto recibe **NO-GO funcional, P0=0, P1=1, P2=1** por la
intermitencia sellada del conductor activo y por el conteo de vulnerabilidades
no reproducible sin corte durable de base. Dirección debe asignar una
corrección acotada y someter el nuevo SHA a revisión funcional y de seguridad
independientes. Este acta no autoaprueba el parche, no acredita código O4 ni
sus descendientes y no modifica porcentajes o estado transversal.

No se hizo push, deploy, cambio de producción o credenciales.

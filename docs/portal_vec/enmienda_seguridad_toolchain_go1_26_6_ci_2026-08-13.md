# Enmienda de seguridad: toolchain Go 1.26.6 en la CI de O4

Fecha: 13 de agosto de 2026.

Tarea: `SEC-TOOLCHAIN-PATCH-1.26.6`.

Corrección de evidencia: `SEC-TOOLCHAIN-P2-CORTE-DB-VULN`, sobre el padre
exacto `74f249587d5c0da41092a2c017ccd6cb54817248`.

Estado: **CANDIDATO A REVISIÓN INDEPENDIENTE**. Este corte no publica, no
despliega, no acredita O4 y no autoriza un nuevo candidato material O4A-P4.

## Base y criterio único

La base exacta es el árbol documental O4 integrado localmente en
`d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c`. Su publicación ejecutaría las
cinco puertas de `.github/workflows/ci.yml`.

La reproducción del preflight con la fijación anterior `go1.26.5` alcanzó
`govulncheck@v1.6.0` y terminó en estado 3 por seis vulnerabilidades
alcanzables de la biblioteca estándar. Este inventario corresponde al corte
observado el 13 de agosto de 2026 a las 23:11 UTC, después de que el registro
de `GO-2026-5026` se actualizara a las 21:43:54 UTC:

| Identificador | Paquete alcanzable |
| --- | --- |
| `GO-2026-6218` | `net/url` |
| `GO-2026-6090` | `crypto/tls` |
| `GO-2026-6089` | `net/http` |
| `GO-2026-6088` | `encoding/xml` |
| `GO-2026-5972` | `encoding/asn1` |
| `GO-2026-5026` | `net/http` |

La [revisión Go 1.26.6](https://go.dev/doc/devel/release#go1.26.6), publicada
el mismo 13 de agosto, contiene las correcciones que cierran los seis
intervalos observados. El criterio CI no es que el conteo histórico permanezca
inmutable, sino que la toolchain seleccionada termine con cero vulnerabilidades
alcanzables según la base oficial vigente.
El criterio único de este corte es que todas las rutas activas de la CI de O4
usen esa revisión corregida sin variar código de negocio, oráculos, permisos,
porcentajes ni estados del roadmap.

## Write-set exacto

```text
go.mod
Dockerfile
tools/o3a_v5_conductor/conductor.sh
tools/o3a_v5_conductor/README.md
docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
```

No se modifican `.github/workflows/ci.yml`, `go.sum`, fuentes Go, fixtures,
ledgers, evidencias históricas, PostgreSQL, configuración de ejecución,
credenciales ni documentos transversales de estado.

## Cambio cerrado

1. `go.mod` conserva `go 1.25.12` como compatibilidad mínima y eleva solamente
   la toolchain de entrega a `go1.26.6`. `actions/setup-go` lee esa autoridad
   mediante `go-version-file`.
2. La etapa de build del `Dockerfile` usa la imagen oficial
   `golang:1.26.6-bookworm` fijada por el índice OCI
   `sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`.
   El manifiesto `linux/amd64` resuelto es
   `sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976`.
3. El conductor O3a V5 sigue fallando cerrado por versión exacta, ahora
   `go1.26.6`. No aprende una versión del entorno ni acepta revisiones
   anteriores o posteriores.
4. Su README actualiza solo la autoridad activa. Las reproducciones y
   referencias históricas que acreditaron Go 1.26.5 permanecen byte a byte y
   siguen describiendo su corte original, no el actual.

No se eleva ninguna dependencia del módulo. `go mod tidy -diff` conserva una
reclasificación preexistente de `github.com/nkiri/xls` que aparece también con
Go 1.26.5; esta minitarea no la incorpora porque es disjunta del parche de
seguridad y la puerta canónica no depende de ese cambio.

## Evidencia local del candidato

Las pruebas que modifican manifiestos o requieren que el checkout pertenezca
al usuario de ejecución se reprodujeron en clones efímeros exactos, propiedad
de `orquesta`. Cada clon recibió únicamente el diff de este candidato y se
eliminó mediante un `trap` acotado. El worktree productor no cambió de
permisos.

### Toolchain, vulnerabilidades y calidad

```text
go version
go mod verify
bash -n tools/o3a_v5_conductor/conductor.sh
shellcheck tools/o3a_v5_conductor/conductor.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
scripts/verificar_calidad.sh
git diff --check
```

Resultado:

- `go version go1.26.6 linux/amd64`;
- módulos verificados;
- Bash y ShellCheck verdes;
- `govulncheck`: `No vulnerabilities found`;
- calidad completa verde: formato, normal, race, vet, build, grafos,
  manifiestos, carga TLS, unitarias Python, vulnerabilidades, tamaños y diff.

El baseline 1.26.5 del mismo árbol llegó hasta `govulncheck` después de dejar
normal y race globales verdes, pero fue rechazado por los seis hallazgos
anteriores. No se relajó ni omitió esa puerta.

### Corte durable del inventario

La salida completa del baseline tiene 4.522 bytes y SHA-256
`4cfc5d433267d239479cf6f77c9bd5b7fd8b9825792be014d30276fdbf10eb86`.
La salida sobre Go 1.26.6 tiene 26 bytes y SHA-256
`3016e51e4eac0d421674d2128bbbdefb2924b4646e0c14a1ab034977ad73fae5`.
Ambas proceden del mismo comando y del mismo árbol; solo cambia la toolchain:

```text
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

El corte baseline conserva al menos una traza alcanzable por identificador:

| Identificador | Símbolo o frontera mínima observada | Versión corregida |
| --- | --- | --- |
| `GO-2026-6218` | `osrm.Calculador.Calcular` → `http.Client.Do` → `url.URL.Parse` | `go1.26.6` |
| `GO-2026-6090` | `ServidorInterno.EscucharYServir` → `http.Server.ServeTLS` → `tls.Conn.HandshakeContext` | `go1.26.6` |
| `GO-2026-6089` | `vec.main` → `http.Server.ListenAndServe` | `go1.26.6` |
| `GO-2026-6088` | `docx.relacionExterna` → `xml.Decoder.Token` | `go1.26.6` |
| `GO-2026-5972` | `cargarMaterialEmisorCapacidadPostgreSQLV4` → `x509.ParsePKIXPublicKey` → `asn1.Unmarshal` | `go1.26.6` |
| `GO-2026-5026` | `osrm.Calculador.Calcular` → `http.Client.Do` | `go1.26.6` |

La base de vulnerabilidades es mutable: este hash y esta tabla hacen durable
la observación que motivó y revisó el parche, pero no convierten seis en un
máximo futuro. Una actualización posterior de la base debe volver a evaluarse
fail-closed; no autoriza a ignorar identificadores nuevos ni a reescribir este
corte histórico.

### Conductor durable O3a V5

```text
tools/o3a_v5_conductor/conductor.sh CHECKOUT_PROPIETARIO EVIDENCIA_VACÍA
```

Resultado interno del conductor:

```text
resultado=GO
go_version=go version go1.26.6 linux/amd64
bloques=14
casos_registrados=74
fd_conductor_inicio=4
fd_conductor_fin=4
residuos=cero
sha_conductor=cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7
sha_fuentes=fc654291ffda8c9dae2ef690267b842568082644cfe2d384d718238ac85d492a
```

Los siete bloques pasaron en modo normal y con un build `-race` real. El
conductor validó su propio `SHA256SUMS` antes de devolver GO. Una ejecución
del mismo target desde un worktree ajeno al usuario de runtime falló cerrada
en la preparación de fixtures; el control positivo sobre el checkout O3a
propietario y la reproducción en clon propietario aislaron ese efecto de
permisos y no se cambiaron los oráculos para ocultarlo.

### Artefactos y PostgreSQL

Se reprodujeron las otras tres puertas funcionales de la CI:

```text
docker build --target runtime-publico
docker build --target runtime-interno
scripts/verificar_contenido_artefactos_productivos.sh PUBLICO INTERNO
scripts/probar_fallo_cerrado_artefactos_productivos.sh PUBLICO INTERNO

VEC_POSTGRES_TEST_IMAGE=postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296 \
  bash deploy/postgresql/autorizacion/probar_integracion_contexto_actor_v3.sh

VEC_POSTGRES_TEST_IMAGE=postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296 \
  bash deploy/postgresql/bolsa_publica/probar_integracion.sh
```

Resultado:

- imágenes pública e interna construidas con Go 1.26.6, inventarios conformes
  y arranque fail-closed sin revelar configuración;
- ContextoActor/PDP V3 verde sobre PostgreSQL 18.4 real, incluidas ACL,
  concurrencia, reintento, replay, revocación y downs;
- Bolsa pública verde sobre PostgreSQL 18.4 real con TLS, publicación,
  rollback, ACL y limpieza;
- contenedores, etiquetas de prueba y directorios efímeros retirados.

## Límites y relevo

Este candidato necesita revisión funcional y de seguridad independientes
sobre el SHA exacto. Las revisiones deben repetir, como mínimo, identidad de
base y write-set, versión/digest, búsqueda de fijaciones activas, conductor
normal/race, `govulncheck`, calidad completa, artefactos, las dos puertas
PostgreSQL, Gitleaks y `git diff --check`.

Un doble GO de esta enmienda no equivale a publicación ni CI remota. Dirección
debe integrar el parche y el árbol documental O4, publicar normalmente y
obtener las cinco puertas verdes antes de asignar un nuevo O4A-P4 material.
O4B-P1, O4A-P5 y O4C-P1 continúan cerrados hasta cumplir sus dependencias.
La corrección de inventario tampoco resuelve por sí sola el P1 independiente
de estabilidad del conductor O3a V5 ni acredita el candidato de toolchain.

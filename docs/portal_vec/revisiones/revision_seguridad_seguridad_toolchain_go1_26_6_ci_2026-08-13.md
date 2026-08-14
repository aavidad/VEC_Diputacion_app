# Revisión de seguridad SEC-TOOLCHAIN-PATCH-1.26.6

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/seguridad-toolchain-go1-26-6-20260813`, sin edición del worktree o
de la rama productora.

Dictamen: **NO-GO**, `P0=0`, `P1=1`, `P2=1`.

## Corte exacto y alcance

Se revisó exclusivamente el candidato
`74f249587d5c0da41092a2c017ccd6cb54817248`, con padre
`d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c` y árbol
`bdbd0b2b05c43d7c14230a5c9b0a3114f44a4bb6`. Es un único commit y su
write-set exacto es:

| Fichero candidato | Delta | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `Dockerfile` | `+1/-1` | 253 | `6045bd15bf061de0db0e70c3858344b48db39fc31dbe9fc1f76c6e397f81bf5d` |
| `docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md` | `+168/-0` | 168 | `377d5eb7e8393ca05c7f13f1c85ac88abb7ee45e6a38e33761824e07ce0b242b` |
| `go.mod` | `+1/-1` | 45 | `a447f8cf5217130ed336bf78bcd93211659e1c7ba547673e9a1753ef6ba7eeca` |
| `tools/o3a_v5_conductor/README.md` | `+3/-2` | 242 | `7933b7617a92e22fddf6f888b9fddcc74ea0224616e0340d9c9f36d7d2533ce8` |
| `tools/o3a_v5_conductor/conductor.sh` | `+1/-1` | 205 | `cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7` |

El total es `+174/-5`. Los cinco objetos son blobs ordinarios; el conductor
conserva modo ejecutable. No se modifican fuentes Go, pruebas, `go.sum`,
workflow, SQL, permisos, configuración de ejecución, estado transversal ni
métricas.

La lectura obligatoria se completó antes de crear esta acta. La reproducción
dinámica se hizo en un clon limpio exacto propiedad del usuario `orquesta`, no
en el worktree root-owned, para no convertir la propiedad del checkout en un
falso defecto del conductor. Los gates pesados se serializaron mediante
`flock /srv/fabrica/proyectos/VEC_Diputacion_app/.sec-toolchain-review-gates.lock`.

## Procedencia oficial y fijación de la cadena de suministro

La [historia oficial de versiones de Go](https://go.dev/doc/devel/release#go1.26.6)
publica Go 1.26.6 el 13 de agosto de 2026 y enumera correcciones de seguridad
en el comando `go` y en `crypto/tls`, `encoding/asn1`, `encoding/xml`,
`html/template`, `net`, `net/http` y `net/url`. La API oficial de descargas
marca la versión estable y devuelve para `go1.26.6.linux-amd64.tar.gz`:

```text
size=66890545
sha256=708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89
```

La consulta autenticada y de solo lectura al registro OCI oficial de Docker
reprodujo la resolución que usa el `Dockerfile`:

```text
tag=1.26.6-bookworm
index_mediaType=application/vnd.oci.image.index.v1+json
index=sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36
linux/amd64=sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976
config=sha256:df664c2b56a98910721a529a9a74e20181c607ac32528e758a1dcfd522a9f011
```

El índice expone además atestaciones para `linux/amd64`. El SBOM SPDX JSON
tiene 14.058.287 bytes y SHA-256
`84f32382aea8aef3f57e450df50d771579960b1c16ab9a782e30fece18633f5c`,
con `pkg:golang/stdlib@1.26.6` y `pkg:generic/go@1.26.6`. La procedencia SLSA
JSON tiene 55.072 bytes y SHA-256
`d10d3493e1911c7cce542dfb1fb8cbca9e90315d0fada3add0b94600c432ff1c`;
declara `GOLANG_VERSION=1.26.6` y la descarga amd64 con la misma huella
`708effb…` de la fuente oficial.

El `Dockerfile` fija el índice, no solo la etiqueta mutable, y Docker resuelve
el manifiesto amd64 anterior. `go.mod` conserva `go 1.25.12` como versión de
lenguaje y fija `toolchain go1.26.6`. Las dos rutas `actions/setup-go` del
workflow están fijadas por SHA, leen `go-version-file: go.mod`, prefieren su
directiva `toolchain` y exportan `GOTOOLCHAIN=local` después de instalarla. No
hay downgrade, versión flotante ni fallback en esas rutas.

## Vulnerabilidades alcanzables

Con el binario local exacto 1.26.5 y `GOENV=off GOTOOLCHAIN=local`, la ejecución
actual de `govulncheck@v1.6.0 ./...` termina en estado 3. Confirma las cinco
vulnerabilidades que motivan el candidato, todas alcanzables y corregidas en
1.26.6:

| ID | Paquete | Ejemplo de alcance observado |
| --- | --- | --- |
| `GO-2026-6218` | `net/url` | `osrm.Calculador.Calcular` → `http.Client.Do` → `url.URL.Parse` |
| `GO-2026-6090` | `crypto/tls` | servidor TLS, lector documental y clientes HTTP |
| `GO-2026-6089` | `net/http` | `ListenAndServe`, `Serve` y `ServeTLS` |
| `GO-2026-6088` | `encoding/xml` | S3, DOCX y lectura PostgreSQL |
| `GO-2026-5972` | `encoding/asn1` | carga de material criptográfico documental |

La base oficial de vulnerabilidades fija para las cinco el intervalo afectado
`1.26.0-0 <= versión < 1.26.6`. La misma ejecución con el binario exacto
1.26.6 termina en estado 0 y `No vulnerabilities found`.

## Hallazgo P1 — el conductor activo no es estable con Go 1.26.6

El conductor O3a es parte de `puerta-calidad` y su propio contrato prohíbe
reintentos: cada uno de los cien índices C21 se ejecuta exactamente una vez y
cualquier divergencia debe cerrar el gate. Sobre el clon propietario exacto,
la reproducción integral obtuvo:

```text
resultado=NO-GO
go_version=go version go1.26.6 linux/amd64
head=74f249587d5c0da41092a2c017ccd6cb54817248
bloques=14
casos_registrados=74
fd_conductor_inicio=5
fd_conductor_fin=5
residuos=cero
```

Los siete bloques normales y seis de los siete bloques race fueron verdes.
`c15_c21` race falló en el índice 44 de C21:

```text
44  TUPLA_C  esperado=0  estado=66  stdout=0  stderr=0  NO-GO
```

El estado 66 es `estadoErrorExternoO3aM38`, no un timeout traducido ni ruido
de salida. El conductor mantuvo su semántica fail-closed, selló evidencia
íntegra y no dejó FD o procesos propios; por ello la falla no puede omitirse.
Otra ejecución exacta, también propietaria y serializada en el mismo host,
había completado los catorce bloques; la discrepancia demuestra intermitencia,
no una incompatibilidad determinista del checkout. El cambio de toolchain es
el único byte ejecutable de este corte que puede afectar al binario race del
conductor.

Una segunda reproducción diagnóstica propia ejecutó solo `c15_c21` con build
race nuevo y sin reintentar ningún caso. Falló todavía antes, en `C16_ALIAS`,
también con estado 66 y salidas vacías. Por tanto no se trata de un único índice
C21 defectuoso ni de evidencia reciclada: el mismo bloque race admite al menos
dos fallos externos en ejecuciones nuevas mientras otras corridas exactas son
verdes.

Se clasifica P1 porque el candidato pretende habilitar la publicación que
ejecutará precisamente esta puerta: hoy puede devolver verde o rojo sobre los
mismos bytes sin que la enmienda identifique ni cierre la causa. No es P0: el
fallo observado deniega y no concede autoridad, no hubo residuo, secreto ni
efecto productivo. Tampoco se autoriza a ocultarlo con retry, `SKIP`, aumento
del oráculo o aceptación del estado 66. Hace falta diagnosticar la causa bajo
1.26.6, corregirla sin relajar C21 y presentar un nuevo candidato reproducible.

## Hallazgo P2 — el inventario de vulnerabilidades queda incompleto

La misma reproducción 1.26.5 encuentra una sexta vulnerabilidad alcanzable:
`GO-2026-5026`, en `net/http`, a través de `http.Client.Do` y del transporte
HTTP. El registro oficial fue modificado el 13 de agosto a las 21:43:54 UTC
para añadir a la biblioteca estándar los mismos límites: afectado antes de
1.26.6 y corregido en 1.26.6. El candidato se creó después de esa modificación,
pero su criterio, tabla y evidencia dicen de forma cerrada «cinco».

El impacto se clasifica P2 porque la actualización a 1.26.6 sí corrige también
esta sexta ruta y `govulncheck` queda verde; no persiste exposición. Sin
embargo, el acta de cambio debe inventariarla y no atribuir al escaneo actual
un resultado incompleto. La corrección documental debe decir seis y conservar
las seis trazas/rangos oficiales.

## Rutas activas y evidencias históricas

El workflow tiene cinco jobs y no fue modificado:

- `puerta-secretos` no ejecuta Go y mantiene checkout sin credenciales;
- `puerta-calidad` instala 1.26.6 desde `go.mod`, ejecuta calidad global y
  después el conductor, que además rechaza cualquier revisión distinta;
- `puerta-artefactos-productivos` construye las superficies pública e interna
  desde el build fijado del `Dockerfile`;
- ContextoActor/PDP V3 es SQL/PostgreSQL y no selecciona otra toolchain;
- Bolsa pública instala de nuevo la toolchain de `go.mod` antes de su runner
  PostgreSQL/TLS.

Las fijaciones 1.26.5 que permanecen en O3a AST, O3b, O3c, runners históricos
y evidencia durable no son rutas activas del workflow. Sus objetos Git son
byte a byte iguales entre padre y candidato. El README del conductor separa
la autoridad activa 1.26.6 de su genealogía histórica 1.26.5; no se recalculan
binarios, hashes ni conclusiones antiguas con la nueva revisión.

## Puertas reproducidas

| Puerta | Resultado |
| --- | --- |
| `git rev-parse HEAD HEAD^ 'HEAD^{tree}'` | SHA, padre y árbol exactos |
| ancestry, `rev-list`, `diff --name-status`, `--numstat` | un commit; cinco ficheros; `+174/-5` |
| `wc -l`, SHA-256 y tipos Git | líneas, huellas y modos de la tabla exactos |
| fuente oficial Go y API `go.dev/dl` | 1.26.6 estable; SHA de descarga exacto |
| registro OCI, índice y manifiesto amd64 | digests exactos; SBOM y SLSA coherentes |
| `go version`, `go mod verify` | `go1.26.6 linux/amd64`; módulos verificados |
| `go mod tidy -diff` | solo reclasificación preexistente de `github.com/nkiri/xls`; no incorporada |
| `bash -n` y ShellCheck 0.11.0 | verdes para conductor y siete bloques |
| `govulncheck` con Go 1.26.5 | estado 3; seis vulnerabilidades alcanzables |
| `govulncheck` con Go 1.26.6 | estado 0; cero vulnerabilidades |
| `scripts/verificar_calidad.sh` | verde completo, incluido normal, race, vet y build globales |
| conductor O3a normal/race | **NO-GO**: C21 race índice 44, estado 66 |
| diagnóstico aislado `c15_c21` race | **NO-GO**: `C16_ALIAS`, estado 66 |
| build `runtime-publico`/`runtime-interno` | verde; binarios `go1.26.6 linux/amd64` |
| inventarios y fallo cerrado de artefactos | verdes; usuario 10001, listas positivas y diagnóstico saneado |
| ContextoActor/PDP V3 PostgreSQL 18.4 | verde, incluidas ACL, carreras, replay, revocación y downs |
| Bolsa pública PostgreSQL 18.4/TLS | verde, incluida publicación, ACL, rollback y limpieza |
| Gitleaks padre→candidato | 1 commit, 7,14 KB, cero filtraciones |
| Gitleaks base O4→candidato | 21 commits, 168,55 KB, cero filtraciones |
| Gitleaks sobre el árbol completo | 59 hallazgos históricos; ninguno se atribuye al delta |
| secretos focales y `git diff --check` | cero patrón sensible nuevo; verde en ambos rangos |
| limpieza final | cero contenedor, imagen, proceso o directorio temporal propio |

La calidad global verde no ejecuta el conductor, que el workflow invoca en el
paso inmediatamente posterior. Por tanto no compensa sus dos reproducciones
rojas. Los 59 hallazgos del escaneo amplio son material histórico ya presente;
los dos rangos que juzgan este candidato están limpios y no se ha ocultado el
resultado del árbol.

Los artefactos revisados fueron:

```text
runtime-publico image=sha256:f194b6112ffe0e227859e732caf28a9840a27eac9d2bc3d3846a10285d09a3be
runtime-interno image=sha256:014e4df6e48def1de8353206ab99cf9a8ea751a8202be0c95ef2cea17f7ad85e
vec-publico sha256=8d595e19487fe9f4682096b1414dd3463cd84211f960fad07805428bfb818dcc
vec-interno sha256=6b514233d4169db3cded3090db1165d79c65a84381727ba5d67e0932f8ebd42e
```

Las imágenes no incorporan credenciales, DSN, claves ni configuración
administrativa. Las superficies siguen separadas, con manifiestos exactos,
filesystem de prueba de solo lectura y red deshabilitada en el arranque
negativo. El parche no cambia autoridad funcional, identidad, autorización,
datos, i18n ni accesibilidad.

## Cierre y corrección requerida

El candidato exacto queda **NO-GO**, `P0=0`, `P1=1`, `P2=1`. Antes de una
nueva revisión debe:

1. diagnosticar y cerrar la intermitencia de `c15_c21` race con Go 1.26.6,
   conservando cien procesos, ejecución única, estado esperado y fail-closed;
2. corregir la enmienda para inventariar las seis vulnerabilidades alcanzables
   observadas con Go 1.26.5 y su ausencia bajo 1.26.6.

No se corrige el productor desde esta acta y no se acepta retry, `SKIP`,
fallback de toolchain ni relajación del oráculo. Este dictamen no acredita O4,
código o descendientes; no autoriza integración, push, publicación, CI remota,
despliegue, producción, credenciales ni cambio de métricas. Dirección y una
nueva doble revisión independiente conservan esas decisiones.

# Revisión de seguridad SEC-TOOLCHAIN P2: corte de base de vulnerabilidades

Fecha: 14 de agosto de 2026.

Tarea revisada: `SEC-TOOLCHAIN-P2-CORTE-DB-VULN`.

Dictamen: **GO de seguridad**, `P0=0`, `P1=0`, `P2=0`, exclusivamente para
la corrección documental del inventario. Este GO no acredita el parche de
toolchain: el P1 independiente del conductor O3a V5 permanece abierto.

## Identidad y alcance

El candidato exacto es `f49c00376a49692991ebc3556b1dc5e8dcfd6f52`, su
padre es `74f249587d5c0da41092a2c017ccd6cb54817248` y su árbol es
`824aaf57358dc4350a0d59dbc82a5259ac156e04`. La ascendencia es directa y el
productor está limpio.

El delta modifica una sola ruta Markdown, `+45/-5`:

```text
M docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
```

El documento final ocupa 208 líneas, es un blob ordinario `100644` y tiene
SHA-256 `fddceda75312e0308be3fbb81181f219ad376a731878e00054b9213001130731`.
No cambia `go.mod`, `Dockerfile`, workflow, conductor, README, fuente Go,
prueba, oráculo, permiso, credencial, estado transversal ni métrica. El único
write-set revisor es esta acta.

Se leyeron íntegros `AGENTS.md`, las lecturas obligatorias y autoridades
normativas, la enmienda candidata y los dos NO-GO previos:

- funcional `ac1a95c25a0793e8ff0387fd5a186d54d9011b1b`;
- seguridad `ef40b1fd404eec0c2f6096f7d5d513c46a5d18dc`.

Los bytes ejecutables del parche de toolchain y del conductor son idénticos
al padre. Por tanto esta revisión solo puede cerrar el P2 documental; no vuelve
a juzgar ni compensa el P1.

## Corte oficial y seis hallazgos

La consulta de las seis entradas JSON de `https://vuln.go.dev/ID/` confirma
para todas `modified=2026-08-13T21:43:54Z`, un intervalo 1.26 corregido en
`1.26.6` y estos paquetes/símbolos alcanzables:

| ID | Paquete | Símbolo oficial presente | Cierre 1.26 |
| --- | --- | --- | --- |
| `GO-2026-6218` | `net/url` | `URL.Parse` | `1.26.6` |
| `GO-2026-6090` | `crypto/tls` | `Conn.HandshakeContext` | `1.26.6` |
| `GO-2026-6089` | `net/http` | `Server.ListenAndServe` | `1.26.6` |
| `GO-2026-6088` | `encoding/xml` | `Decoder.Token` | `1.26.6` |
| `GO-2026-5972` | `encoding/asn1` | `Unmarshal` | `1.26.6` |
| `GO-2026-5026` | `net/http` | `Client.Do` | `1.26.6` |

La autoridad oficial de versiones registra Go 1.26.6 como publicado el 13 de
agosto de 2026 y enumera correcciones de seguridad en esos paquetes. La
enmienda sitúa la observación a las 23:11 UTC, después de la modificación de
la base, e incluye exactamente los seis IDs. En particular ya no omite
`GO-2026-5026` ni lo confunde con una ruta no alcanzable de `x/net`: el corte
actual de stdlib lo expone mediante `net/http`.

Las seis trazas documentadas terminan en símbolos incluidos por la base y sus
fronteras de repositorio existen en el árbol revisado:

- `osrm.Calculador.Calcular` llama a `http.Client.Do`, y la ruta URL alcanza
  `url.URL.Parse`;
- `ServidorInterno.EscucharYServir` usa `http.Server.ServeTLS`, que alcanza
  `tls.Conn.HandshakeContext`;
- `vec.main` alcanza `http.Server.ListenAndServe`;
- `docx.relacionExterna` alcanza `xml.Decoder.Token`;
- `cargarMaterialEmisorCapacidadPostgreSQLV4` alcanza
  `x509.ParsePKIXPublicKey` y `asn1.Unmarshal`;
- `osrm.Calculador.Calcular` alcanza también `GO-2026-5026` mediante
  `http.Client.Do`.

## Reproducción de las salidas

Se reprodujo el mismo comando sobre el mismo árbol con las dos toolchains,
sin modificar el checkout:

```text
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

| Toolchain | Resultado | Bytes completos | SHA-256 |
| --- | --- | ---: | --- |
| Go 1.26.5 | seis vulnerabilidades; estado interno 3 | 4.522 | `4cfc5d433267d239479cf6f77c9bd5b7fd8b9825792be014d30276fdbf10eb86` |
| Go 1.26.6 | `No vulnerabilities found` y estado 0 | 26 | `3016e51e4eac0d421674d2128bbbdefb2924b4646e0c14a1ab034977ad73fae5` |

En el baseline, los 4.522 bytes son la salida completa combinada: 4.508 bytes
de `govulncheck` más los 14 bytes `exit status 3` que `go run` escribe al
propagar el estado interno; el proceso exterior `go run` devuelve 1. Esta
distinción explica el hash sin reinterpretar el resultado rojo. En 1.26.6 no
hay stderr y el proceso devuelve 0.

El documento no convierte seis en un máximo ni congela una autorización por
conteo. Distingue el corte histórico mediante hora y huellas, reconoce que la
base es mutable y exige volver a ejecutar contra la base vigente. El criterio
positivo es cero vulnerabilidades alcanzables; cualquier ID futuro conserva
el fallo cerrado. Tampoco reescribe evidencias históricas 1.26.5.

## Puertas proporcionales

| Puerta | Resultado |
| --- | --- |
| HEAD, padre, árbol, ascendencia y limpieza | exactos y verdes |
| `git diff --name-status`, `--numstat`, modos, líneas y SHA | una M Markdown, `+45/-5`, exactos |
| seis entradas oficiales y hora de modificación | seis de seis; todas `21:43:54Z` |
| intervalos y símbolos oficiales | seis corregidos en 1.26.6; tabla coherente |
| `govulncheck` Go 1.26.5 | 4.522 bytes y SHA declarado; fallo cerrado |
| `govulncheck` Go 1.26.6 | 26 bytes y SHA declarado; cero hallazgos |
| criterio ante base mutable | cero, sin techo, fallback ni excepción |
| P1 O3a | expresamente abierto; no acreditado por este corte |
| enlace oficial Go 1.26.6 | HTTP 200 y contenido coherente |
| `git diff --check` | verde |
| Gitleaks v8.30.0, padre..candidato | 1 commit, 2,55 KB, cero fugas |
| Gitleaks v8.30.0, `5345d5d..candidato` | 22 commits, 171,11 KB, cero fugas |

No se repitieron calidad global, conductor, Docker ni PostgreSQL: el candidato
solo cambia evidencia Markdown, deja esos bytes intactos y no puede resolver
el P1. Ejecutarlos no es proporcional al criterio único de esta revisión.

Comandos principales:

```text
git rev-parse HEAD HEAD^ HEAD^{tree}
git diff --name-status HEAD^ HEAD
git diff --numstat HEAD^ HEAD
git diff --check HEAD^ HEAD
wc -l docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
sha256sum docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
curl -fsSL https://vuln.go.dev/ID/ID.json | jq ...
GOENV=off GOTOOLCHAIN=go1.26.5 go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./... 2>&1
GOENV=off GOTOOLCHAIN=go1.26.6 go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./... 2>&1
go run github.com/zricethezav/gitleaks/v8@v8.30.0 git . --no-banner --redact --no-color --log-opts=RANGO
```

## Dictamen y relevo

La corrección exacta recibe **GO de seguridad**, `P0=0`, `P1=0`, `P2=0`.
Cierra el P2 documental de conteo/corte para esta revisión, sujeto al GO
funcional independiente y a integración por dirección.

El ancestro `74f2495` continúa en NO-GO por la inestabilidad sellada de O3a
V5; este acta no acredita estabilidad, toolchain, O4, publicación ni CI. No se
autoriza push, despliegue, producción, credenciales ni cambio de métricas.

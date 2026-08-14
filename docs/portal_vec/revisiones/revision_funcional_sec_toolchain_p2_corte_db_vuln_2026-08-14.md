# Revisión funcional SEC-TOOLCHAIN P2 — corte de base de vulnerabilidades

Fecha: 14 de agosto de 2026.

Tarea: `SEC-TOOLCHAIN-P2-CORTE-DB-VULN`.

Dictamen: **GO funcional acotado**, `P0=0`, `P1=0`, `P2=0`, únicamente
para la corrección documental del inventario. El parche de toolchain conserva
el **NO-GO** anterior: el P1 independiente de estabilidad del conductor O3a
V5 continúa abierto y este acta no lo compensa ni lo acredita.

## Identidad, independencia y alcance

Se revisó el candidato exacto
`f49c00376a49692991ebc3556b1dc5e8dcfd6f52`, hijo directo de
`74f249587d5c0da41092a2c017ccd6cb54817248`, con árbol
`824aaf57358dc4350a0d59dbc82a5259ac156e04`. La rama productora
`trabajo/sec-toolchain-p2-corte-db-vuln-20260814` permaneció limpia.

La revisión se hizo en el worktree exclusivo
`/srv/fabrica/revisiones/sec-toolchain-p2-corte-db-vuln-funcional-20260814`,
rama `revision/sec-toolchain-p2-corte-db-vuln-funcional-20260814`, creada
directamente desde el SHA candidato. No se editó, movió, integró ni rebasó la
rama productora.

Antes de editar se leyeron íntegros `AGENTS.md`, el relevo de sesión, el mapa
de objetivos, el tablero, el relevo de contratación temporal, la especificación
normalizada de RRHH, la hoja de ruta y la matriz normativa. También se
releyeron la enmienda completa y los dos NO-GO independientes del parche
`74f2495`: funcional `ac1a95c25a0793e8ff0387fd5a186d54d9011b1b` y de
seguridad `ef40b1fd404eec0c2f6096f7d5d513c46a5d18dc`.

El delta candidato modifica una sola ruta Markdown, `+45/-5`:

```text
M docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
```

El documento final es un blob `100644`, ocupa 208 líneas y 8.915 bytes, y su
SHA-256 es
`fddceda75312e0308be3fbb81181f219ad376a731878e00054b9213001130731`.
El único write-set de esta revisión es esta acta. No cambian código, pruebas,
`go.mod`, Dockerfile, workflow, conductor, README, oráculos, permisos,
credenciales, estado transversal ni métricas.

## Cierre del P2 anterior

La consulta de las seis entradas oficiales en `https://vuln.go.dev/ID/`
confirma que todas tienen `modified=2026-08-13T21:43:54Z`. Para la línea
1.26, las seis declaran el intervalo afectado desde `1.26.0-0` y cierre en
`1.26.6`; también conservan los intervalos oficiales anterior y posterior,
cerrados en `1.25.13` y `1.27.0-rc.3`, respectivamente.

La enmienda fija su observación el 13 de agosto de 2026 a las 23:11 UTC, es
decir, después de la actualización de la base, y contiene exactamente estos
seis IDs, paquetes y trazas compatibles con los símbolos oficiales:

| ID | Frontera alcanzable documentada | Símbolo oficial | Cierre 1.26 |
| --- | --- | --- | --- |
| `GO-2026-6218` | `osrm.Calculador.Calcular` → `http.Client.Do` → `url.URL.Parse` | `net/url.URL.Parse` | `1.26.6` |
| `GO-2026-6090` | `ServidorInterno.EscucharYServir` → `http.Server.ServeTLS` → `tls.Conn.HandshakeContext` | `crypto/tls.Conn.HandshakeContext` | `1.26.6` |
| `GO-2026-6089` | `vec.main` → `http.Server.ListenAndServe` | `net/http.Server.ListenAndServe` | `1.26.6` |
| `GO-2026-6088` | `docx.relacionExterna` → `xml.Decoder.Token` | `encoding/xml.Decoder.Token` | `1.26.6` |
| `GO-2026-5972` | `cargarMaterialEmisorCapacidadPostgreSQLV4` → `x509.ParsePKIXPublicKey` → `asn1.Unmarshal` | `encoding/asn1.Unmarshal` | `1.26.6` |
| `GO-2026-5026` | `osrm.Calculador.Calcular` → `http.Client.Do` | `net/http.Client.Do` | `1.26.6` |

Las fronteras propias citadas existen en el árbol revisado. La autoridad de
versiones de Go registra 1.26.6 como publicada el 13 de agosto de 2026 y
enumera correcciones de seguridad en `crypto/tls`, `encoding/asn1`,
`encoding/xml`, `net/http` y `net/url`, entre otros paquetes. El enlace usado
por la enmienda respondió HTTP 200.

El corte durable registra dos salidas completas del mismo comando y árbol,
variando únicamente la toolchain:

| Toolchain | Resultado | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| Go 1.26.5 | seis vulnerabilidades alcanzables; fallo | 4.522 | `4cfc5d433267d239479cf6f77c9bd5b7fd8b9825792be014d30276fdbf10eb86` |
| Go 1.26.6 | `No vulnerabilities found` | 26 | `3016e51e4eac0d421674d2128bbbdefb2924b4646e0c14a1ab034977ad73fae5` |

El hash de 26 bytes se reprodujo además sobre la cadena exacta
`No vulnerabilities found.\n`. La salida baseline es la captura combinada de
la detección roja, incluida la propagación de su estado; no se interpreta como
éxito por el hecho de conservarse mediante hash.

El documento ya no presenta cinco como un máximo ni como autorización
persistente. Separa expresamente el corte histórico de la base mutable y
mantiene el criterio dinámico: cada ejecución futura debe consultar la base
oficial vigente y obtener cero vulnerabilidades alcanzables. Cualquier ID
nuevo vuelve a bloquear la puerta; no existe excepción, fallback, lista
permitida ni reescritura de la evidencia previa.

## Puertas proporcionales

| Puerta | Resultado |
| --- | --- |
| HEAD, padre, árbol y ascendencia | exactos; padre directo |
| Limpieza productora y revisora antes del acta | limpias |
| `diff --name-status`, `--numstat`, modo, líneas, bytes y SHA | una M Markdown, `+45/-5`, exactos |
| Seis registros oficiales | seis de seis; hora, rangos, paquetes y símbolos coherentes |
| Corte temporal | `21:43:54Z < 23:11Z` |
| Baseline Go 1.26.5 | seis IDs, 4.522 bytes, SHA declarado, resultado rojo |
| Go 1.26.6 | cero IDs, 26 bytes, SHA declarado, resultado verde |
| Criterio dinámico y fallo cerrado | conforme; cero obligatorio y sin techo futuro |
| P1 del conductor | explícitamente abierto; no acreditado |
| Enlace oficial 1.26.6 | HTTP 200; fecha y paquetes coherentes |
| `git diff --check` | verde |
| Gitleaks padre→candidato | sellado: 1 commit, 2,55 KB, cero fugas |
| Gitleaks base O4→candidato | sellado: 22 commits, 171,11 KB, cero fugas |

No se repitieron conductor, calidad global, Docker, PostgreSQL ni otros gates
pesados: este candidato modifica únicamente evidencia Markdown, deja sus bytes
ejecutables idénticos al padre y no puede cerrar el P1. Tampoco se reintentó la
puerta inestable buscando un verde. Gitleaks no estaba disponible como binario
local; se contrastaron las dos ejecuciones selladas sobre los rangos exactos.

Comandos principales reproducidos:

```text
git rev-parse HEAD HEAD^ 'HEAD^{tree}'
git merge-base --is-ancestor 74f249587d5c0da41092a2c017ccd6cb54817248 f49c00376a49692991ebc3556b1dc5e8dcfd6f52
git status --short --branch
git diff --name-status HEAD^..HEAD
git diff --numstat HEAD^..HEAD
git ls-tree HEAD docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
wc -l -c docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
sha256sum docs/portal_vec/enmienda_seguridad_toolchain_go1_26_6_ci_2026-08-13.md
for id in GO-2026-6218 GO-2026-6090 GO-2026-6089 GO-2026-6088 GO-2026-5972 GO-2026-5026; do curl -fsS "https://vuln.go.dev/ID/${id}.json" | jq ...; done
curl -fsS https://go.dev/doc/devel/release
git diff --check HEAD^..HEAD
```

## Dictamen y relevo

La corrección exacta recibe **GO funcional acotado**, `P0=0`, `P1=0`,
`P2=0`. Cierra para esta revisión el P2 documental de inventario, corte y
reproducibilidad, sujeto a la revisión de seguridad independiente y a la
decisión de integración de dirección.

El parche `74f2495` y su descendiente documental no reciben GO de toolchain:
el P1 de estabilidad C21/O3a V5 sigue vigente. Este acta no acredita O4,
publicación, CI, estabilidad ni descendientes, y no autoriza push, despliegue,
producción, credenciales, estado transversal o cambio de métricas.

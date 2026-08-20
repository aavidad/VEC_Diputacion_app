# Revisión funcional independiente O4A-P4-REVISION-781

Fecha: 20 de agosto de 2026.

Identificador: `O4A-P4-REVISION-781-FUNCIONAL`.

Estado: **NO-GO, P0=1, P1=2, P2=0**.

Este dictamen revisa exclusivamente el candidato
`781bb5891ba3304bfed9d104e71d48846a5b6679`, hijo exacto de
`2b7eaf498f8f68a90b66c004f166a62f070b5064`, con árbol
`707065f5e2d8a19fd8d37e79af889be52e9bf4b5`. No corrige producto, no revisa
descendientes y no autoriza integración, publicación, CI, despliegue,
producción, credenciales, estado transversal ni métricas.

## Independencia, lectura y write-set

La revisión se realizó en el worktree exclusivo
`/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o4a-p4-funcional-781-20260820`,
rama `revision/o4a-p4-funcional-781-20260820`, sin editar ni mover ramas
productoras. El único write-set de revisión es esta acta.

Antes de editar se leyeron completos `/srv/fabrica/AGENTS.md`, `AGENTS.md`, el
relevo de sesión vigente, mapa/roadmap, tablero, relevo de contratación
temporal, expediente RRHH, hoja de ruta y matriz normativa. También se
releyeron completos las decisiones O4a/O4b, las dos actas `NO-GO` exactas de
`2b7eaf4`, la enmienda O4AB-P0 V3 `2abba91`, su checkpoint y sus dos actas
independientes `GO` funcional y de seguridad.

## Identidad y huellas reproducidas

La genealogía desde la base publicada es lineal y contiene seis commits:

```text
5345d5d097b51ab3567983f048feabeceaf2957b
  -> 1e75c829215c43b4472908e9e00acc255aa016d9
  -> 4f5b5a1736a4e2a90cc728b03a9fa57b0b20e7f9
  -> 1f8186cf0705043fea638db4ba1ab4ff086455a3
  -> 6a7a83b252a24971d1255f19c0e30ae7f4e4eb90
  -> 2b7eaf498f8f68a90b66c004f166a62f070b5064
  -> 781bb5891ba3304bfed9d104e71d48846a5b6679
```

El delta `2b7eaf4..781bb58` modifica solo el arnés focal, `+29/-1`; no modifica
producto. El rango completo desde `5345d5d` añade únicamente estos dos Go:

| Fichero | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go` | 547 | 22018 | `42ad068a627e003b6d356b5488d1df8f464e55ade83fa566b0c7ee046be5ef7f` |
| `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go` | 600 | 26833 | `c6d3782e011885048f74b41f998906b7a8a8f3f7ccf9d64602b677c97a22912c` |

El blob productivo es el mismo en padre y candidato:
`aab44954a0b82591b44adad6687171726a1e1352`. El test cambia de
`8e43e72a334d2511eb572c62a4016d8e6b1d526e` a
`750f2e875237014e8ec57a3e09c0ef47ae253b22`.

Las autoridades publicadas O4a y O4b conservan, respectivamente, 535 líneas y
SHA-256 `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`,
y 443 líneas y SHA-256
`675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`.

Las actas históricas exactas permanecen:

| Objeto | Commit | Líneas | SHA-256 | Dictamen |
| --- | --- | ---: | --- | --- |
| revisión funcional de `2b7eaf4` | `e20a5fe597d9c172d59b99e75878f8e99e196c76` | 176 | `f72faa38f2c4543b46b4d8318a19a87149b84d15e4a3f7044317baa4df96de48` | `NO-GO`, `P0=1,P1=2,P2=0` |
| revisión de seguridad de `2b7eaf4` | `a62ee60f70315a1fc0d290290b681c9c3f35227b` | 147 | `f01503425807851a9efb3a6542dd110054e387d9760626b3b921538a7f80135e` | `NO-GO`, `P0=0,P1=1,P2=0` |

La enmienda V3 exacta tiene 282 líneas y SHA-256
`2b44bd03d0422a4aecff78ad6400873ecb17686901c40a6b0a68bdabd91d769f`;
su checkpoint tiene 125 líneas y SHA-256
`ce5cf0d0248cf1ac98e7580ec210e434b7f1610cb70515606b8e2f035134bc6a`.
Recibió `GO` funcional en
`c0ba2e78bbae2155876e2319196c281054b850b2` y `GO` de seguridad en
`d16c0f6bd42078abe090c554ab42b1f9bb8d3a1c`, ambos con
`P0=P1=P2=0`. Esos dictámenes acreditan solamente los bytes documentales V3:
no acreditan código, publicación ni CI.

## Prevalencia y estado de la enmienda V3

La V3 se declara candidata documental y exige doble revisión, publicación y CI
5/5 antes de un nuevo candidato material O4A-P4 (líneas 7--11). Sus dos
revisiones satisfacen la primera condición, pero sus propios cierres reservan
publicación y CI a dirección. No existe acreditación de publicación ni CI 5/5
para V3 en el corte recibido. Además, ni `2abba91`, ni sus actas `c0ba2e7` y
`d16c0f6` son ancestros de `781bb58`.

Por tanto, `781bb58` no puede apoyarse en V3 como autoridad material vigente.
Incluso si se usa V3 únicamente como contrato corrector pendiente, el
candidato no materializa sus aristas: conserva exactamente el producto
rechazado en `2b7eaf4`.

## Auditoría funcional completa

| Requisito | Resultado |
| --- | --- |
| Autoridad opaca, consumo one-shot, sellos, owner, causa, raws y estados A3/A4/A5/A7 | Sin cambio productivo respecto de `2b7eaf4`; los focales existentes pasan. |
| SALIDA/terminalidad anterior a señales llega A7 sin efecto | Conforme en la rama ya existente. |
| STOP inicial estable abre gracia; error o `NO_ESTABLE` salta a KILL rápido | La nueva focal cubre `NO_ESTABLE`; la semántica productiva permanece. |
| TERM y CONT conservan orden, cardinalidad y raws | Conforme para las ramas ya implementadas. |
| `TERMINAL` posterior a TERM/CONT acepta `observado <= finGracia` | No conforme: la línea 457 usa comparación estricta `<`. |
| `GRUPO_PRESENTE` en igualdad con `finGracia` abre `PARADA_FINAL` | No conforme: la línea 458 también exige `<`, por lo que la igualdad es fatal. |
| `PARADA_FINAL` autoriza cardinalidad máxima 1 y consume cardinalidad real 0/1 | No conforme: las líneas 261--262 y 478--490 fijan y aceptan solo cardinalidad 1. |
| Terminal final antes de STOP, cardinal 0, llega A7 sin STOP/KILL/incidente | No conforme y sin focal. |
| Terminal final posterior a STOP, cardinal 1, llega A7 sin KILL/incidente | No conforme y sin focal. |
| STOP final estable autoriza KILL; no estable/error enclava incidente y autoriza KILL | Implementado; la nueva focal cubre `NO_ESTABLE`. |
| Preflight que cruza `finParadaFinal` no produce resultado ni transición | No materializado en O4b y no consumible por este P4; sin focal en el candidato. |
| Cero syscall, señal, Wait, FD, red, SQL o API pública dentro de O4a-P4 | Conforme por inspección estructural y arnés AST. |
| Replay/carrera/falsificación fail-closed | Los casos ya presentes y los dos nuevos fatales pasan; falta la matriz V3 E01--E14 aplicable a O4a. |

## Hallazgos

### P0-01 — se conserva la autorización de KILL después de terminalidad final

La autoridad O4a publicada ordena en su fila 308 que terminalidad observada en
`PARADA_FINAL` llegue A7 con cero KILL. La enmienda V3 pendiente conserva y
precisa esa prohibición para terminalidad antes y después de STOP, con
cardinalidad real 0 o 1 (líneas 191--205).

El candidato no modifica las líneas 478--490 del producto: solo admite
`ESTABLE`, `NO_ESTABLE` o raw de error, nunca `TERMINAL`, y todos los resultados
admitidos emiten `MATAR_GRUPO`. El arnés, líneas 181--190, sigue fijando STOP
final estable seguido de KILL. La nueva prueba de STOP final `NO_ESTABLE`
también afirma KILL; no corrige la arista terminal. Es el mismo efecto no
autorizado que causó el P0 de `e20a5fe`, no una ausencia meramente documental.

### P1-01 — siguen cerrados el borde de gracia y la cardinalidad final 0/1

`marcaResultadoO4aM38(..., true)` exige `observado.Before(limite)`. Se usa tanto
para `TERMINAL` como para `GRUPO_PRESENTE` en las líneas 457--458, de modo que
ambos resultados fallan en igualdad con `finGracia`. V3 exige terminalidad
`<= finGracia`, presencia `== finGracia` como borde de parada final y un sobre
final cuya cardinalidad real sea 0 o 1. El tipo conserva un único campo
`cardinalidad`, la especificación fija 1 para `PARADA_FINAL` y su consumidor
rechaza 0. Ninguna de estas incompatibilidades cambia en `781bb58`.

### P1-02 — falta el prerequisito autoritativo y la matriz correctora

V3 tiene dos GO documentales, pero no publicación ni CI 5/5 acreditada; por su
propio DAG no autoriza todavía material O4A-P4. El candidato nace directamente
del histórico `2b7eaf4` y sólo añade cuatro casos: dos ramas positivas
`NO_ESTABLE` y dos rechazos de STOP inicial. No añade los focales contractuales
de terminalidad final cardinal 0/1, igualdad de gracia, cruce del preflight,
presencia/terminalidad posteriores ni las forjas asociadas. Las puertas verdes
solo demuestran coherencia con la semántica histórica rechazada.

## Puertas reproducidas

Se ejecutaron, sin retries ni puertas globales:

```text
git status --short --branch
git show --no-patch --format='HEAD=%H%nPARENT=%P%nTREE=%T%nSUBJECT=%s' HEAD
git merge-base --is-ancestor 2b7eaf498f8f68a90b66c004f166a62f070b5064 HEAD
git merge-base --is-ancestor 5345d5d097b51ab3567983f048feabeceaf2957b HEAD
git rev-list --count 5345d5d097b51ab3567983f048feabeceaf2957b..HEAD
git diff --name-status 2b7eaf498f8f68a90b66c004f166a62f070b5064..HEAD
git diff --numstat 2b7eaf498f8f68a90b66c004f166a62f070b5064..HEAD
wc -lc <producto> <test>
sha256sum <producto> <test>
mapfile -t fuentes < <(find deploy/postgresql/autorizacion_atestada_v3/pruebas_sql -maxdepth 1 -type f -name '*contexto_actor_v1*.go' ! -name '*_test.go' ! -name 'capturar_snapshot_*' -print | sort)
fuentes+=(deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go)
go test -count=1 -run '^TestEtapasO4aP4' "${fuentes[@]}"
go test -race -count=1 -run '^TestEtapasO4aP4' "${fuentes[@]}"
go vet "${fuentes[@]}"
gofmt -d <producto> <test>
git diff --check 2b7eaf498f8f68a90b66c004f166a62f070b5064..HEAD
git diff --check 5345d5d097b51ab3567983f048feabeceaf2957b..HEAD
```

Resultados: identidad, padre, árbol, ancestry, seis commits lineales, write-set,
líneas, bytes, hashes y blobs exactos; normal verde (`0.141s`), race verde
(`1.839s`), vet verde, gofmt sin salida y ambos diff-check verdes. Se usaron 26
fuentes explícitas; ningún paquete global entró en el gate.

Gitleaks no está disponible y, por mandato, no se instaló ni ejecutó.
PostgreSQL, Docker, E2E, HTTP, mutantes y las puertas Go globales no se
ejecutaron: están fuera del alcance permitido de esta revisión focal y no
podrían compensar los hallazgos causales.

## Seguridad, privacidad, i18n y accesibilidad

El delta añade sólo pruebas internas y no introduce credenciales, datos
personales, interfaz, texto visible, red, persistencia o API. i18n y
accesibilidad no aplican. El P0 sí afecta privilegio mínimo: una terminalidad
final acreditable no bloquea la autorización posterior de KILL.

## Relevo inequívoco

`781bb5891ba3304bfed9d104e71d48846a5b6679` queda **NO-GO** y no debe
integrarse ni publicarse. La siguiente minitarea correctora canónica propuesta
es `O4A-P4-CORRECCION-TERMINALIDAD-V3`, condicionada primero a la publicación y
CI 5/5 de los bytes exactos O4AB-P0 V3 ya doblemente revisados.

Su write-set máximo y estrecho será exclusivamente:

1. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go`.

Dirección deberá fijar como base el SHA publicado que acredite V3; no se inventa
ese SHA desde esta revisión. El criterio de cierre es materializar y matar los
mutantes de las aristas V3 aplicables a O4a: igualdad de gracia, cardinalidad
final 0/1, terminalidad antes/después de STOP hacia A7 sin KILL/incidente,
preservación de las tres ramas no terminales y rechazo fail-closed de marcas,
raws, etapas y cardinalidades forjadas; después requiere normal/race/vet/gofmt,
dos revisiones independientes, publicación y CI 5/5.

Esta acta no autoaprueba V3, el candidato, la corrección futura ni sus
descendientes.

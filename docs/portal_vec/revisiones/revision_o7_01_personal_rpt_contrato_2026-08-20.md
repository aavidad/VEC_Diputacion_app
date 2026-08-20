# Revisión independiente O7-01 — contrato Personal/RPT

Fecha: 20 de agosto de 2026.

## Dictamen

**GO**, con `P0=0`, `P1=0` y `P2=0`.

El candidato define un puerto neutral para solicitar a Personal un alta ligada
a RPT y validar su resultado. El contrato usa exclusivamente referencias
opacas, versión y huella de la fuente RPT, una capacidad nominal no
autoritativa, correlación e idempotencia exactas y una unión cerrada entre
confirmación y rechazo gobernado.

Esta acta acredita solo el contrato técnico aislado `O7-01`. No confirma una
incorporación, no implementa adaptador, persistencia, outbox/inbox, API, red o
efecto en Personal/RPT, no cierra O7 y no autoriza datos reales ni producción.

## Identidad exacta revisada

| Elemento | Valor |
| --- | --- |
| Tarea | `O7-01-CONTRATO-PERSONAL-RPT` |
| Candidato | `a312375bf548621aa70bb3da7a588f0ee13ec7ea` |
| Padre único | `781bb5891ba3304bfed9d104e71d48846a5b6679` |
| Árbol base | `707065f5e2d8a19fd8d37e79af889be52e9bf4b5` |
| Árbol candidato | `20fbb701479f6ea840c085996149fdc9320343c5` |
| Rama de revisión | `revision/o7-01-personal-rpt-20260820` |
| Worktree | `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o7-01-personal-rpt-revision-20260820` |
| Asunto | `feat(O7-01): define contrato Personal RPT` |

`git rev-list --parents -n 1` confirma que el candidato tiene como único padre
la base declarada y `git merge-base` devuelve esa misma base. El delta exacto
contiene dos altas y ninguna modificación adicional.

## Write-set del candidato

| Fichero | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `internal/modules/contrataciontemporal/ports/integracion_personal.go` | 190 | 7391 | `a3ecd386ba5e1da0b9f3eb93cd3b6af8f6695cb45a92b03d6ead38fa6df719bb` |
| `internal/modules/contrataciontemporal/ports/integracion_personal_test.go` | 187 | 6400 | `972767dbdf294a4761dc54572ad48210b71b979e5367d8bcab2860959627267a` |
| **Total** | **377** | **13791** | — |

Ambos ficheros quedan por debajo del objetivo de 500 líneas y del tope duro de
800 de DEC-051. No hay cambios en O4A ni fuera del write-set productor.

## Autoridades contrastadas

Se leyeron completas antes de editar esta acta:

- `/srv/fabrica/AGENTS.md` y `AGENTS.md`;
- `relevo_sesion_2026-07-29_inicio_ct47.md`;
- `mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md`;
- `tablero_tareas_contratacion_temporal_2026-07-23.md`;
- `relevo_contratacion_temporal_2026-07-23.md`;
- `expediente_contratacion_temporal_rrhh.md`;
- `objetivos_y_hoja_ruta_rrhh_2026-07-23.md`;
- `matriz_normativa_contratacion_temporal_2026-07-23.md`.

El contraste específico incluye O7-01 en el tablero; la solicitud de alta a
Personal y su confirmación de relación/ocupación en el expediente RRHH; y la
reserva de relación jurídica, ocupación, puesto, incorporación y cese a la
autoridad de Personal en la hoja de ruta. La matriz normativa mantiene
cerradas las puertas organizativas y el uso de datos reales.

## Revisión funcional y arquitectónica

### Hexagonalidad, autoridad y alcance

- El fichero productivo pertenece a `ports` e importa solo biblioteca estándar
  y el dominio de contratación temporal. No contiene una implementación del
  puerto ni conoce HTTP, SQL, proveedor, transporte o almacenamiento.
- Contratación temporal transporta puesto y plaza por referencia; Personal
  conserva la creación y autoridad de relación y ocupación. Incorporación y
  cese no se modelan como efectos locales: quedan para sus tareas y autoridades
  posteriores.
- `CapacidadRef` es material nominal opaco. La validación sintáctica no concede
  permiso y el propio contrato exige que proceda de una frontera confiable.
  Este corte no fabrica, verifica, consume ni persiste una autorización.
- La interfaz solo declara `SolicitarAlta` con `context.Context`; no hay
  adaptador, composición, I/O, red, envío ni efecto.

### Solicitud, RPT y ligaduras

- Esquema y versión contractual deben coincidir exactamente con V1; ausencia,
  versión cero, versión no interoperable o esquema desconocido fallan cerrados.
- Solicitud, expediente, capacidad, correlación, idempotencia, puesto y plaza
  usan la gramática acotada de referencias opacas.
- La fuente RPT conserva referencia, versión positiva interoperable y huella
  SHA-256 no nula. No copia una RPT ni un agregado de Personal.
- El material canónico incluye, en orden fijo, los trece componentes de la
  solicitud: esquema, versión contractual, solicitud, expediente y versión,
  capacidad, correlación, idempotencia, terna RPT, puesto y plaza.
- La huella SHA-256 del material liga por tanto el resultado a la solicitud
  completa, no solo a sus identificadores de coordinación.

### Resultado como unión cerrada

- Todo resultado debe repetir esquema y versión exactos, conservar referencias
  opacas de resultado y recibo, y coincidir byte a byte en solicitud,
  correlación, idempotencia y huella de la solicitud completa.
- `confirmada` exige simultáneamente relación y ocupación opacas y prohíbe
  motivo de rechazo.
- `rechazada` prohíbe relación y ocupación y exige un motivo gobernado como
  referencia, versión y huella; no admite texto libre.
- Cualquier estado distinto, mezcla de variantes, ausencia o discordancia se
  reduce al error opaco `ErrResultadoAltaPersonalRPTInvalido`.

### Canon, límites, copias y datos

- La serialización canónica usa nombres y valores prefijados por longitud en
  bytes, enteros decimales y orden fijo; no depende de mapas, locale o JSON.
- `MaterialCanonico` devuelve una copia defensiva y la huella se recalcula
  desde material validado. Las pruebas acreditan determinismo, copia y
  sensibilidad al cambio de plaza.
- Las referencias están limitadas por la gramática compartida a 160 bytes, las
  huellas a 64 caracteres hexadecimales y las versiones al entero seguro JSON
  `9007199254740991`. No hay slices, mapas ni cardinalidad variable en el
  contrato.
- Los fixtures son sintéticos y opacos. No aparecen DNI, nombres, contacto,
  credenciales, secretos, DSN ni datos personales reales.

## Hallazgos

| Severidad | Cantidad | Detalle |
| --- | ---: | --- |
| P0 | 0 | Ninguno. |
| P1 | 0 | Ninguno. |
| P2 | 0 | Ninguno. |

## Gates reproducidos sobre el candidato

| Puerta | Resultado | Evidencia |
| --- | --- | --- |
| `gofmt -d internal/modules/contrataciontemporal/ports/integracion_personal.go internal/modules/contrataciontemporal/ports/integracion_personal_test.go` | PASS | código 0, salida vacía |
| `go test -count=1 -run '^TestO701' ./internal/modules/contrataciontemporal/ports` | PASS | `ok`, paquete `ports`, 0.012 s |
| `go test -race -count=1 -run '^TestO701' ./internal/modules/contrataciontemporal/ports` | PASS | `ok`, paquete `ports`, 1.045 s |
| `go vet ./internal/modules/contrataciontemporal/ports` | PASS | código 0, salida vacía |
| `git diff --check 781bb5891ba3304bfed9d104e71d48846a5b6679..a312375bf548621aa70bb3da7a588f0ee13ec7ea` | PASS | código 0, salida vacía |

## Pruebas omitidas y motivo

No se ejecutaron suites globales, PostgreSQL, Docker, red, E2E, Gitleaks,
instalaciones, publicación ni despliegue. Estaban expresamente fuera del
encargo y el candidato solo añade un puerto neutral y unitarias focales.

## Seguridad, privacidad, i18n y accesibilidad

El contrato minimiza datos mediante referencias opacas, no recibe identidad o
autoridad desde presentación y no incorpora texto visible. No abre superficies
de i18n o accesibilidad. Tampoco acredita licitud, EIPD, ENS, ENI, retención,
seguridad del conector, aceptación RRHH ni conformidad organizativa.

## Limitaciones y siguiente paso

- No existe alta real, confirmación durable de incorporación, registro de
  efectos, reconciliación, adaptador Personal/RPT, API o E2E.
- Esta revisión no autoriza O7-02 ni otra tarea posterior por sí sola; solo
  dirección decide dependencias, integración y actualización transversal.
- Producción continúa en `NO-GO` y el uso de datos reales permanece prohibido.

## Independencia y modificaciones

La revisión fue realizada en rama y worktree exclusivos, sin subdelegación. No
se editaron los dos ficheros candidatos. El único write-set del revisor es esta
acta.

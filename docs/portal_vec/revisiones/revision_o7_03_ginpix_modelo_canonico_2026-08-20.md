# Revisión independiente O7-03 — modelo canónico GINPIX

Fecha: 20 de agosto de 2026.

## Dictamen

**GO**, con `P0=0`, `P1=0` y `P2=0`.

El candidato implementa un modelo canónico y un mapeo GINPIX versionados,
sellados y deterministas, junto con un puerto de transformación neutral. El
corte no contiene adaptadores, I/O, envío, exportación, persistencia, red ni
efecto sobre GINPIX o cualquier otro sistema.

Esta acta acredita solo el contrato técnico aislado O7-03. No cierra O7, no
autoriza O7-04/O7-05, no habilita datos reales ni cambia el estado de
producción.

## Identidad exacta revisada

| Elemento | Valor |
| --- | --- |
| Tarea | `O7-03-GINPIX-MODELO-CANONICO` |
| Candidato | `2b15fe87a73aa9ce7c1746a5a8aca752ae3c9cd3` |
| Padre único | `781bb5891ba3304bfed9d104e71d48846a5b6679` |
| Árbol candidato | `589f9bf48ddfcbe02e1d090401eda18747a79108` |
| Rama de revisión | `revision/o7-03-ginpix-20260820` |
| Worktree | `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o7-03-ginpix-revision-20260820` |
| Asunto | `feat(O7-03): fija modelo canonico GINPIX` |

`git diff-tree --no-commit-id --name-status -r 2b15fe87...` devuelve cuatro
altas y ninguna modificación adicional.

## Write-set del candidato

| Fichero | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `internal/modules/contrataciontemporal/domain/ginpix.go` | 496 | 17620 | `5b1886565d267e348f131c5033e855abcb10ac586687d258c0dd537daea8803d` |
| `internal/modules/contrataciontemporal/domain/ginpix_test.go` | 303 | 11804 | `efacd7aa579cb9454a68ef3a0380884ac143d053000ce5e4e8cac7cdfc9aa11b` |
| `internal/modules/contrataciontemporal/ports/ginpix.go` | 64 | 1928 | `940b240c30e19b23f43a4be3101ef579b9818944c8123f81de6b0353d8bb076c` |
| `internal/modules/contrataciontemporal/ports/ginpix_test.go` | 144 | 4552 | `ce3b711168c3b853931bcd5c34e47fdc1d87efc0a1b84218b60de99a434d0de4` |
| **Total** | **1007** | **35904** | — |

Los cuatro ficheros quedan por debajo del tope duro de 800 líneas de DEC-051;
el fichero productivo mayor queda en 496, dentro del objetivo general de 500.

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

El contraste específico incluye la especificación RRHH de GINPIX en las
líneas 176-181 del expediente, O7-03 en el tablero, la habilitación anticipada
sin envío de las líneas 69-72 y 141 del mapa, O7 en la hoja de ruta y las
puertas completas de la matriz normativa.

## Revisión funcional y arquitectónica

### Hexagonal y alcance

- `domain/ginpix.go` depende solo de biblioteca estándar y primitivas del
  propio dominio. No importa aplicación, puertos, adaptadores, HTTP, SQL ni
  proveedor.
- `ports/ginpix.go` importa el dominio y define una solicitud inmutable y el
  contrato mínimo `MapeadorGINPIX` para una transformación pura.
- No aparecen cliente, fichero, red, autenticación externa, reintento,
  conciliación, recibo externo, persistencia ni operación de envío. Esas
  responsabilidades permanecen fuera del candidato.

### Modelo, versión y ligaduras

- Modelo, mapeo y carga usan esquemas V1 exactos. Los constructores rechazan
  cualquier esquema desconocido.
- El modelo liga versión de expediente, expediente, incorporación,
  procedencia, correlación e idempotencia mediante referencias opacas.
- El mapeo liga referencia, versión explícita, procedencia y huella.
- La carga conserva las dos procedencias, ambas huellas, versión de mapeo,
  correlación, idempotencia y versión de expediente; su propia huella cubre
  todo ese material.
- Modelo y mapeo se restauran recalculando su huella y comparándola en tiempo
  constante. Una publicación adulterada falla cerrada.

### Ausente, nulo y vacío

`EstadoCampoGINPIXAusente`, `EstadoCampoGINPIXNulo` y
`EstadoCampoGINPIXValor` son estados diferentes. Un valor presente puede ser
la cadena vacía y su codificación y huella difieren de ausente y nulo. La
compatibilidad distingue los tres casos mediante `Obligatorio`,
`PermiteNulo` y `PermiteVacio`.

### Cobertura y compatibilidad exactas

- Modelo y mapeo rechazan claves de origen duplicadas; el mapeo rechaza además
  destinos duplicados.
- La aplicación exige igual cardinalidad y que cada origen del mapeo exista
  en el modelo. Dada la unicidad en ambos lados, esas dos condiciones prueban
  igualdad exacta de conjuntos: no puede omitirse ni inventarse un campo.
- Ausente obligatorio, nulo no permitido y vacío no permitido se deniegan.
  La incompatibilidad no produce una carga parcial.

### Canon, límites y copias

- El modelo se ordena por clave canónica, el mapeo por origen y la carga por
  destino. El material usa campos prefijados por longitud, representación
  decimal estable y booleanos explícitos.
- Un vector SHA-256 fijo prueba la serialización de carga y las pruebas
  comparan permutaciones de entrada del modelo y del mapeo.
- Antes de mapas, copias u ordenación internos se aplican cardinalidad máxima
  128, valor máximo 4096 bytes, carga funcional máxima 65536 bytes, versión
  positiva acotada y gramáticas limitadas para claves y referencias.
- Constructores, publicaciones, getters y serializaciones realizan copias
  defensivas de slices y bytes. Los valores cero y las huellas inválidas se
  rechazan.

### Datos y privacidad

- Todos los fixtures son sintéticos y están marcados como tales; no hay DNI,
  nombres, contacto, credenciales, secretos, DSN ni datos reales.
- Expediente, incorporación, procedencias, correlación e idempotencia viajan
  como identificadores opacos. Este contrato no otorga acceso ni autoridad.
- El corte no acredita licitud, EIPD, ENS, ENI, retención, conectores reales o
  conformidad organizativa. La matriz normativa mantiene esas puertas.

## Hallazgos

| Severidad | Cantidad | Detalle |
| --- | ---: | --- |
| P0 | 0 | Ninguno. |
| P1 | 0 | Ninguno. |
| P2 | 0 | Ninguno. |

## Gates reproducidos sobre el candidato

| Puerta | Resultado | Evidencia |
| --- | --- | --- |
| `gofmt -d` de los cuatro ficheros | PASS | código 0, salida vacía |
| `go test -count=1 ./internal/modules/contrataciontemporal/domain ./internal/modules/contrataciontemporal/ports` | PASS | `domain 0.773s`; `ports 0.760s` |
| `go test -race -count=1 ./internal/modules/contrataciontemporal/domain ./internal/modules/contrataciontemporal/ports` | PASS | `domain 8.348s`; `ports 8.091s` |
| `go vet ./internal/modules/contrataciontemporal/domain ./internal/modules/contrataciontemporal/ports` | PASS | código 0, salida vacía |
| `git diff --check 781bb5891ba3304bfed9d104e71d48846a5b6679 2b15fe87a73aa9ce7c1746a5a8aca752ae3c9cd3` | PASS | código 0, salida vacía |

## Pruebas omitidas y motivo

No se ejecutaron suites globales, PostgreSQL, Docker, red, E2E, Gitleaks,
instalaciones ni despliegue. Estaban expresamente fuera del encargo y el
candidato solo añade dominio, puerto neutral y unitarias focales.

## Limitaciones y siguiente paso

- No existe adaptador API ni de fichero, no hay envío y no hay recuperación de
  efectos externos; corresponden a tareas posteriores independientes.
- La aprobación funcional de campos y el uso de datos reales siguen sujetos a
  RRHH y a las puertas organizativas de la matriz normativa.
- Esta revisión deja el candidato técnicamente apto para que dirección decida
  su integración. Solo dirección puede actualizar tablero, métricas o estado.
- No se declara ninguna tarea posterior automáticamente desbloqueada por esta
  acta; O7-04 y O7-05 conservan sus dependencias y write-sets propios.

## Independencia y modificaciones

La revisión fue realizada en rama y worktree exclusivos, sin subdelegación.
No se editaron los cuatro ficheros candidatos. El único write-set del revisor
es esta acta.

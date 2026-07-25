# Revisión independiente final O3-04

Fecha: 24 de julio de 2026.

Commit revisado: `2834783`.

## Veredicto

`GO`. La persistencia, auditoría y outbox del análisis de RRHH quedan cerrados
sin hallazgos críticos, altos ni medios abiertos.

Esta acta no sustituye el `NO-GO` del candidato `88d3250`. Lo conserva como
antecedente y acredita que el commit corrector posterior resolvió sus seis
bloqueantes.

## Frontera aceptada

- El dominio y los puertos permanecen independientes de PostgreSQL.
- La única función productiva ejecutable es
  `confirmar_operacion_analisis_v3`.
- El rol de ejecución es mínimo, la transacción es serializable y de escritura,
  y el `search_path` queda cerrado.
- Las versiones V1 y V2 no pueden usarse como atajo de ejecución.
- Las migraciones `000009` a `000011` son idénticas al estado anterior al
  candidato rechazado; las correcciones se incorporan mediante `000013` a
  `000015`.

## Evidencia funcional y de seguridad

- Las invariantes Go y PostgreSQL coinciden para referencias, grupo, huellas,
  Unicode NFC, tiempos, período, RC, moneda, coste y actuación.
- La autorización VEC liga exactamente el análisis derivado, las fuentes, la
  política, la unidad, el motivo, la segregación, el artefacto y la versión.
- El replay exacto devuelve el mismo recibo y el replay divergente falla
  cerrado.
- Los trece puntos de escritura durable revierten el estado completo ante
  fallo.
- La cancelación anterior a `COMMIT` no deja efectos parciales.
- Cuatro sesiones concurrentes producen un único efecto y recibo terminal.
- Se recorren registro con RC validada y coste, y rectificación por un actor
  distinto, con historia de versiones 1–2–3.
- Las alteraciones estructuralmente válidas de unidad y motivo se rechazan al
  romper el vínculo VEC.
- La reversión con historia queda protegida; la destrucción explícita y la
  reinstalación limpia están probadas.

## Pruebas ejecutadas

- Runner PostgreSQL 18 integral: `exit 0`, con cierre
  `OK: autorización atestada, efecto único, atomicidad, ACL y rollback`.
- `go test ./...`.
- Pruebas de carrera del módulo de Contratación Temporal.
- `go vet ./...`.
- `go mod verify`.
- `shellcheck -e SC1091` sobre los runners modificados.
- `git diff --check`.
- Puerta de tamaño de ficheros.

Durante la revisión se reprodujo una inestabilidad del propio ensayo: el
runner compilaba el emisor Go después de iniciar una capacidad con vigencia
máxima de cinco segundos. El commit revisado precompila el binario antes de
abrir esa ventana. No amplía la vigencia ni modifica el contrato productivo.
La repetición integral posterior quedó verde y no dejó contenedores
efímeros residuales.

## Alcance

El `GO` cierra O3-04 y eleva el procedimiento de contratación temporal a
17 de 46 tareas verificadas, un 37 % redondeado. No autoriza producción ni
cierra O3-05; API, formulario, composición, identidad real y aceptaciones
formales conservan sus propias puertas.

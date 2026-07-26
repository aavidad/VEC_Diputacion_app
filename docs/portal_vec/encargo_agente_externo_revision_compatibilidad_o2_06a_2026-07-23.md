# Encargo externo — compatibilidad O2-06A con O2-05 final

## Mandato

Realiza una revisión de compatibilidad, en solo lectura, entre el diseño O2-06A
y el candidato SQL O2-05. No implementes el adaptador, no cambies ninguno de
los dos candidatos, no integres ni publiques. Tu salida permitirá actualizar
el diseño inmediatamente después del `GO` de O2-05.

Lee completos `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`ESTADO_PROYECTO.md`, `docs/instruccion_direccion_2026-07-18.md`,
`docs/portal_vec/cola_trabajo_agentes_contratacion_temporal_2026-07-23.md`,
el diseño O2-06A y sus dos revisiones.

## Subagentes obligatorios

Usa dos subagentes en solo lectura:

- especialista PostgreSQL 18/pgx, transacciones, ACL y reconciliación;
- especialista Go, hexagonalidad, errores, concurrencia y seguridad.

El principal contrasta ambas matrices y entrega una única tabla normativa.

## Fuentes exactas

- Diseño: rama `agent/ct-o2-06-diseno`, SHA `4cc4422`, worktree
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-06-diseno`.
- SQL candidato: rama `agent/ct-o2-05-sql-atomico`, SHA `cfc935a`, worktree
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-05-sql-atomico`.
- Integración de referencia: `54027d0`.

No uses cambios sin commit. `cfc935a` sigue pendiente de doble revisión; por
ello el resultado es una matriz de compatibilidad, no autorización para
implementar.

## Trabajo requerido

Extrae y compara:

1. nombre y firma exactos de confirmación y reconciliación;
2. orden, tipo, límite, codificación y nulabilidad de cada entrada/salida;
3. cánones, hashes, referencias cruzadas y marcador del agregado;
4. semántica nuevo/replay y validación de las ocho piezas;
5. barrera utilizada por escritura/reconciliación;
6. roles, `GRANT`, `SECURITY DEFINER`, `search_path` y aislamiento;
7. resultados cerrados, errores y SQLSTATE;
8. `COMMIT` exitoso, rollback concluyente e indeterminado;
9. reintentos `40001`/`40P01`, plazos, cancelación y conexión nueva;
10. reconciliación 0/1/>1, recibo adulterado y ausencia de reparación;
11. reinicio, rotación, revocación y replay con nueva concesión;
12. divergencias entre el diseño y el SQL real.

La tabla final debe indicar por elemento: evidencia SQL, previsión del diseño,
estado `coincide/no coincide/ausente`, severidad y cambio exacto requerido. No
inventes una firma Go ni declares estable el SHA.

Aplica castellano, arquitectura hexagonal, puertos intercambiables,
neutralidad web/escritorio/API/CLI/MCP, denegación por defecto, redacción y
cero secretos/datos personales.

## Validación

Reproduce las puertas de análisis que no muten el candidato: sintaxis shell,
inventario de migraciones y firmas, `go test`/`vet` focales pertinentes,
Gitleaks de ambos rangos, tamaños, `diff-check` y `merge-tree`. No necesitas
repetir el runner PostgreSQL completo si los revisores O2-05 lo tienen activo;
si lo omites, dilo expresamente.

## Entrega

Crea `.worktrees/rev-compat-o2-06a`, rama
`review/rev-compat-o2-06a`, desde `54027d0`. Solo añade:

`docs/portal_vec/revisiones/o2_06a_matriz_compatibilidad_o2_05_2026-07-23.md`

Entrega dictamen de compatibilidad, tabla completa, cambios exactos,
dependencias y pruebas. Commit documental único. No modifiques los candidatos
ni marques O2-05/O2-06 cerrados.

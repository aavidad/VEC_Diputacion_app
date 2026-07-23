# Encargo externo — revisión independiente O2-08A

## Mandato

Revisa de forma hostil la API interna aislada del alta de contratación. No
corrijas, integres, publiques ni registres la ruta. Debes decidir si el
candidato puede entrar como adaptador definitivo aún no compuesto.

Lee completos `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`ESTADO_PROYECTO.md`, `docs/instruccion_direccion_2026-07-18.md`,
`docs/portal_vec/cola_trabajo_agentes_contratacion_temporal_2026-07-23.md`,
el encargo O2-08A, su OpenAPI y su memoria técnica.

## Subagentes obligatorios

Usa dos subagentes en solo lectura:

- revisor de protocolos HTTP, OpenAPI, fuzzing, límites y Unicode;
- revisor de autorización, privacidad, hexagonalidad y neutralidad de canal.

El agente principal reproduce los hallazgos y emite un único dictamen.

## Fuente exacta

Rama `agent/ct-o2-08-api`, SHA `3fb1d5c`, worktree:

`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-08-api`

Base de comparación: `747e7ad`. No leas cambios sin commit.

## Criterios bloqueantes

Comprueba:

1. método, ruta, `Content-Type`, `Accept`, query, compresión y tamaño cerrados;
2. JSON con lista positiva, campos duplicados, caja alternativa, `null`,
   segundo documento, UTF-8, NFC, controles, profundidad y tokens;
3. fechas civiles, periodos, importes, moneda, adjuntos y referencias;
4. ausencia de identidad, rol, permiso, capacidad, decisión, idempotencia o
   credencial aportada por cuerpo, query, `Authorization`, cookies o cabeceras;
5. autoridad obtenida solo de `context.Context` confiable y copias defensivas;
6. recibo interno completo validado antes de la proyección pública minimizada;
7. recibo válido posterior al efecto frente a cancelación competitiva y
   resultado indeterminado neutral;
8. errores i18n, correlación opaca, redacción y cabeceras de respuesta seguras;
9. OpenAPI 3.1 cerrado y semánticamente válido;
10. que no exista ruta registrada, servicio falso, éxito simulado ni
    dependencia HTTP dentro de aplicación/dominio.

Aplica castellano coherente, arquitectura hexagonal, adaptadores
intercambiables, denegación por defecto, cero cookies/almacenamiento web,
canales neutrales y cero secretos/datos personales.

## Puertas mínimas

```bash
go test ./internal/modules/contrataciontemporal/adapters/httpinterno -count=50
go test -race ./internal/modules/contrataciontemporal/adapters/httpinterno -count=1
go test ./...
go vet ./...
git diff --check 747e7ad..3fb1d5c
```

Ejecuta además el análisis semántico OpenAPI 3.1 con una herramienta ya
disponible y aprobada. No instales dependencias globales ni modifiques el
equipo. Si no existe, registra el bloqueo. Comprueba Gitleaks, tamaños y
`merge-tree` contra `54027d0`.

## Entrega

Crea `.worktrees/rev-o2-08a`, rama `review/rev-o2-08a`, desde `54027d0`.
Solo añade:

`docs/portal_vec/revisiones/o2_08a_revision_independiente_final_2026-07-23.md`

Incluye `GO/NO-GO`, contraejemplos, evidencia, puertas y riesgos. Commit
documental único; no modifiques el candidato.


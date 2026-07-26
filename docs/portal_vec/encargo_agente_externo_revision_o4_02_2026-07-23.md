# Encargo externo — revisión independiente O4-02

## Mandato

Actúa como revisor independiente y hostil del candidato O4-02. No programes,
no corrijas el candidato, no integres, no publiques y no borres ramas. Tu
resultado es una revisión reproducible con `GO` o `NO-GO`.

Lee completos, por este orden:

1. `AGENTS.md`;
2. `ORQUESTACION_AGENTES.md`;
3. `ESTADO_PROYECTO.md`;
4. `docs/instruccion_direccion_2026-07-18.md`;
5. `docs/portal_vec/cola_trabajo_agentes_contratacion_temporal_2026-07-23.md`;
6. la documentación O4-01/O4-02 y sus relevos.

## Subagentes obligatorios

Usa dos subagentes en solo lectura:

- uno especializado en concurrencia, tiempo, cancelación, replay e
  idempotencia;
- otro especializado en seguridad, criptografía, arquitectura hexagonal,
  segregación y fuga de datos.

No les permitas editar. Reproduce personalmente sus hallazgos antes de
incluirlos.

## Fuente exacta

Revisa la rama `agent/ct-o4-02-rework2`, SHA `a0c7ecf`, situada en:

`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o4-02-rework2`

El árbol debe estar limpio. No revises cambios posteriores sin commit. Compara
contra el SHA de integración `54027d0`.

## Criterios bloqueantes

Intenta demostrar, mediante pruebas permanentes o temporales:

1. regresión del reloj entre cualquier lectura del caso de uso;
2. reapertura de catálogo, atestación o confirmación ya caducados;
3. acceso al consumidor tras un retroceso o evidencia inválida;
4. `COMMIT` válido ambiguo cuando compite cancelación o plazo;
5. aceptación de recibo adulterado, petición distinta o resultado incompatible;
6. doble efecto con mismo identificador y nuevo recibo;
7. divergencia bajo concurrencia y replay;
8. autoridad aportada por el cliente o identidad no segregada;
9. orquestación concreta de colaboradores dentro de `ports`;
10. referencias, claves, errores o datos sensibles expuestos.

Revisa además castellano coherente, i18n, canales web/escritorio/API/CLI/MCP
neutrales, ausencia de cookies y almacenamiento del navegador, adaptadores
intercambiables y denegación por defecto.

## Puertas mínimas

Ejecuta:

```bash
go test ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=50
go test -race ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=1
go test ./...
go vet ./...
git diff --check 54027d0..a0c7ecf
```

Comprueba tamaños, Gitleaks solo del rango y `git merge-tree`. Una prueba
omitida se declara, nunca se considera verde.

## Entrega

Crea un worktree propio dentro de `.worktrees/rev-o4-02`, rama
`review/rev-o4-02`, desde `54027d0`. Solo puedes añadir:

`docs/portal_vec/revisiones/o4_02_revision_independiente_final_2026-07-23.md`

El informe contiene dictamen, severidad, pasos exactos, salida, líneas
afectadas, puertas, riesgos residuales y requisitos transversales. Haz un
commit documental. No modifiques la rama candidata.


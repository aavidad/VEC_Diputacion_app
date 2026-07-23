# Encargo aislado O8-01 — dominio gobernado de seguimiento laboral

## Mandato

Lee completo este documento y ejecuta el encargo sin ampliar alcance. Lee
`AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`docs/instruccion_direccion_2026-07-18.md`, el tablero de contratación
temporal, `internal/modules/contrataciontemporal/domain/expediente.go` y
`tipos.go`.

Implementa un modelo de dominio definitivo, no una demo. No integres ni
empujes tu rama.

## Subagentes obligatorios

El agente principal es el único editor y crea dos revisores de solo lectura:

1. revisor de invariantes temporales/CAS/copias/concurrencia y arquitectura;
2. revisor de administración pública, trazabilidad, minimización, calendarios,
   i18n y ausencia de reglas laborales inventadas o rígidas.

Ambos emiten GO/NO-GO con archivo/línea y pruebas adversariales.

## Worktree

```bash
cd /home/alberto/Trabajo/VEC_Diputacion_app
git worktree list
test ! -e .worktrees/ct-o8-01-seguimiento
git worktree add .worktrees/ct-o8-01-seguimiento \
  -b agent/ct-o8-01-seguimiento feature/contratacion-temporal
cd .worktrees/ct-o8-01-seguimiento
```

Si ya existe, detente. El worktree no puede salir del proyecto.

## Objetivo

Cerrar O8-01 con un agregado de seguimiento independiente y versionado para:

- incorporación;
- prórrogas;
- incidencias;
- suspensión o espera administrativa cuando la definición publicada lo
  permita;
- cese previsto y cese efectivo;
- rectificación motivada sin reescribir historia.

No incorpores normativa concreta no aprobada ni plazos fijos. Estados,
motivos, transiciones y requisitos deben proceder de una definición de flujo
publicada, versionada e inmutable. Una nueva transición configurada no debe
exigir recompilar el núcleo.

## Zona exclusiva

Crea únicamente:

```text
internal/modules/contrataciontemporal/domain/seguimiento.go
internal/modules/contrataciontemporal/domain/seguimiento_canon.go
internal/modules/contrataciontemporal/domain/seguimiento_test.go
internal/modules/contrataciontemporal/domain/seguimiento_transiciones_test.go
internal/modules/contrataciontemporal/domain/seguimiento_rehidratacion_test.go
internal/modules/contrataciontemporal/domain/seguimiento_concurrencia_test.go
docs/portal_vec/o8_01_dominio_seguimiento_2026-07-23.md
```

Puedes dividir pruebas adicionales dentro de esa familia. No modifiques
`expediente.go`, otros archivos de dominio, aplicación, `ports`, adaptadores,
SQL, API, web, manifiesto ni documentación compartida.

## Diseño requerido

Modela un agregado separado ligado por referencias opacas a organización,
expediente y relación/nombramiento, sin nombres, DNI ni contacto.

Debe contener:

- versión CAS y secuencia append-only;
- definición de seguimiento: referencia, versión, huella SHA-256, publicación
  y vigencia;
- estado actual como clave de catálogo, no `switch` cerrado;
- transición aplicada: clave, origen, destino, motivo catalogado, actor,
  unidad, instante efectivo, registrada en y documentos por referencia;
- periodo laboral previsto y periodos resultantes;
- evidencia de calendario: referencia, versión, huella, ámbito territorial y
  resultado computado; el dominio no codifica festivos;
- referencias de actuación, recibo y correlación para futura persistencia;
- rectificación que añade un acto compensatorio y nunca edita el anterior.

La definición publicada debe validar genéricamente:

- unicidad de estados/transiciones;
- estado inicial y finales;
- transición origen→destino;
- requisitos de motivo, documentos, periodo y calendario;
- límites de tamaño y ausencia de ciclos silenciosos cuando estén prohibidos;
- nueva clave administrable sin cambios de código.

No atribuyas automáticamente efectos jurídicos a una incidencia. El flujo
gobernado decide qué transición permite; O8-02 coordinará autorización y tareas.

## Invariantes

- UTC canónico y orden temporal explícito;
- una prórroga no borra ni reduce historia y no crea solapamientos incoherentes;
- un cese efectivo no puede preceder incorporación ni su propio acto;
- estados finales impiden nuevas transiciones salvo rectificación/reapertura
  expresamente publicada;
- versión esperada incorrecta produce conflicto sin mutar;
- mismos datos de actuación no se aplican dos veces; referencia repetida con
  otro contenido es conflicto;
- rehidratación reproduce exactamente estado, versión, periodos y huellas;
- copias defensivas: ningún slice/map devuelto muta el agregado;
- serialización canónica determinista, cerrada, con límites y sin datos
  personales;
- errores en castellano y sin detalles privados.

## Pruebas obligatorias

1. incorporación→prórroga→incidencia→cese según definición publicada;
2. transición nueva añadida a la definición funciona sin modificar código;
3. transición ausente, estado/motivo/documento/calendario incorrectos;
4. CAS, replay exacto y colisión semántica;
5. solapamientos, fechas límite, UTC y extremos de rango;
6. cese antes de incorporación y operación tras final;
7. rectificación append-only y segregación de actor cuando la definición la
   exige;
8. rehidratación completa y adulteración de cualquier evento;
9. copias defensivas y uso concurrente sin carrera;
10. canon determinista y rechazo de campos/colecciones excesivos;
11. ausencia de nombres, DNI, correo, ubicación, fichajes y causas médicas;
12. archivos menores de 800 líneas.

Ejecuta:

```bash
gofmt -w internal/modules/contrataciontemporal/domain/seguimiento*.go
go test -race ./internal/modules/contrataciontemporal/domain -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Ejecuta Gitleaks si está disponible. No añadas dependencias.

## Entrega

Commits pequeños en castellano; árbol limpio; no push. Entrega SHA, pruebas,
dictámenes de revisores, decisiones, límites y:

**O8-01 modela dominio; no persiste, expone API ni habilita producción.**


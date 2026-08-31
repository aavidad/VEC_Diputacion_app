# CT-LITE-O5-01-RUTAS-ASIGNACION — registro modular

Fecha: 31 de agosto de 2026.

Estado del productor: candidato local pendiente de revisión independiente del
hash exacto. Este documento no concede `GO`, no cierra O5-01 y no acredita
alcanzabilidad desde la raíz real, E2E ni producción.

## Punto de partida

- Worktree exclusivo:
  `.worktrees/ct-lite-o5-01-rutas-asignacion-20260831`.
- Rama: `trabajo/ct-lite-o5-01-rutas-asignacion-20260831`.
- Base y `HEAD` exactos antes de editar:
  `2b96b64602ba82c904dbf1a0532542f50db3f032`.
- Árbol inicial: limpio.
- Toolchain: `go1.26.5 linux/amd64`, con `GOPROXY=off`.

No se usaron Git de red, Docker, PostgreSQL ni otros worktrees.

## Capability e invariante

`NuevasRutas` registra conjuntamente, al final de la lista existente:

```text
POST /api/vec/contratacion-temporal/asignaciones
POST /api/vec/contratacion-temporal/reasignaciones
```

Ambas entradas reciben una única instancia construida mediante
`NuevoManejadorAsignacion`. Si la autoridad o el ejecutor faltan, también como
`nil` tipado, el constructor conserva la denegación predeterminada: devuelve
una lista `nil` y `ErrRutasContratacionTemporalInvalidas`, sin publicar una API
parcial.

El registro no crea autoridad de identidad o despacho. La autorización de la
ruta exacta sigue perteneciendo al `httpapi.Handler` exterior; solo después de
una decisión positiva puede ejecutarse la autoridad de canal ya inyectada en
el manejador de asignación.

## Write-set

- `internal/app/composicion/interna/contrataciontemporal/rutas.go`;
- `internal/app/composicion/interna/contrataciontemporal/rutas_test.go`, solo
  dependencias de prueba y lista esperada;
- `internal/app/composicion/interna/contrataciontemporal/rutas_asignacion_test.go`;
- `docs/portal_vec/ct_lite_o5_01_rutas_asignacion_2026-08-31.md`.

`rutas_seguridad_test.go` no se modificó. No se tocaron el adaptador HTTP, la
aplicación, puertos, persistencia, manifiesto, composición raíz ni interfaz.

## Matriz focal

La regresión nueva prueba:

1. un único registro de cada ruta, en las dos posiciones finales;
2. la identidad física del manejador compartido;
3. rechazo atómico de autoridad y ejecutor `nil`;
4. rechazo atómico de autoridad y ejecutor `nil` tipados; y
5. denegación exterior de cada ruta antes de invocar la autoridad de canal o
   cualquiera de las dos operaciones del ejecutor de asignación.

La lista completa existente se conserva en el mismo orden y continúa
verificando ausencia de rutas duplicadas y manejadores no nulos.

## Puertas del productor

Se ejecutaron con `GOTOOLCHAIN=go1.26.5 GOPROXY=off` y terminaron en verde:

```text
gofmt -w rutas.go rutas_test.go rutas_asignacion_test.go
go test -count=1 ./internal/app/composicion/interna/contrataciontemporal
go test -count=50 -run 'Asignacion|Reasignacion' ./internal/app/composicion/interna/contrataciontemporal
go test -race -count=5 -run 'Asignacion|Reasignacion' ./internal/app/composicion/interna/contrataciontemporal
go vet ./internal/app/composicion/interna/contrataciontemporal
go test -count=1 -run 'Asignacion|Reasignacion' ./internal/modules/contrataciontemporal/adapters/httpinterno
git diff --check
```

La búsqueda focal de secretos usa
`/tmp/vec-gitleaks-20260831`, cuya huella SHA-256 verificada es
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`,
y termina sin hallazgos sobre el diff preparado.

## Seguridad, privacidad, i18n y accesibilidad

La autoridad exterior de rutas exactas continúa siendo única y previa al
manejador. El corte no obtiene identidad, perfil, organización, actor, rol o
permiso desde HTTP y no incorpora credenciales, cookies, almacenamiento web,
datos personales, conectores o secretos.

No se añaden respuestas ni textos visibles, por lo que no cambia el contrato
i18n ni la accesibilidad. Los errores conservan el centinela opaco existente de
la composición.

## Límites y revisión requerida

Este corte solo hace alcanzables las dos rutas dentro del conjunto modular que
devuelve `NuevasRutas`. La raíz real todavía no consume necesariamente ese
conjunto con autoridades corporativas; tampoco se acredita persistencia,
notificación durable, SQL, web, E2E, cierre de O5-01 ni vertical productiva.

Un agente distinto debe revisar el hash local exacto, reproducir las puertas y
emitir `GO` o `NO-GO`. El productor no integra, publica ni se autoaprueba.

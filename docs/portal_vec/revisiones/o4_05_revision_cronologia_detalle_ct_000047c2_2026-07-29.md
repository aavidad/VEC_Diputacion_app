# Revisión de cronología de detalle CT-000047C2

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para la minitarea CT-000047C2.

| Elemento | Valor |
| --- | --- |
| Base | `ce3f3d4` |
| Commit integrado | `e3e12d5` |
| Ficheros | 2 |
| P0 / P1 / P2 | 0 / 0 / 0 |

## Defecto corregido

El caso de detalle capturaba el reloj antes de resolver el contexto y
reutilizaba ese instante después de autorizar. Una capacidad VEC-AD-3 emitida
durante la autorización podía comenzar después del instante de la orden.

El flujo queda:

```text
resolver contexto
→ leer instante canónico de autorización
→ autorizar
→ leer instante canónico de orden
→ comprobar monotonía y ventana de capacidad
→ construir orden
→ consultar y registrar
```

Instantes nulos, no canónicos, regresivos, vencidos o cancelados fallan antes
de invocar `SesionConsultaRRHH`.

## Evidencia

Productor, revisor y dirección ejecutaron pruebas deterministas de capacidad
emitida después del primer instante, retroceso, límite exclusivo de
vencimiento, valores no canónicos y cancelación. Quedaron verdes:

```text
pruebas focales repetidas
go test del paquete application
go test -race focal
go vet del paquete
git diff --check
verificación de tamaños
Gitleaks del commit
```

La minitarea no modifica puertos, HTTP, PostgreSQL, autoridad, PDP, raíz ni
web. Por sí sola no cambia el avance funcional de O4-05.

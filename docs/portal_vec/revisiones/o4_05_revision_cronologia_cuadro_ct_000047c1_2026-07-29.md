# Revisión de cronología de cuadro CT-000047C1

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para la minitarea CT-000047C1.

| Elemento | Valor |
| --- | --- |
| Base | `ce3f3d4` |
| Commit integrado | `ce77db3` |
| Ficheros | 2 |
| P0 / P1 / P2 | 0 / 0 / 0 |

## Defecto corregido

El cuadro reutilizaba un instante leído antes de resolver el contexto y de
autorizar. El flujo corregido es:

```text
resolver contexto
→ leer instante canónico de autorización
→ autorizar
→ leer instante canónico de orden
→ comprobar monotonía
→ construir orden y volver a validar la ventana
→ consultar y registrar
```

Las dos lecturas deben llegar ya en UTC y con precisión de microsegundo. El
caso falla cerrado ante valores nulos, no canónicos, regresivos, vencidos o
cancelados; no corrige silenciosamente un reloj defectuoso.

## Evidencia

Productor, revisor y dirección cubrieron de forma determinista la emisión de
la capacidad después del primer instante, retroceso, vencimiento half-open,
valores no canónicos y cancelación. Ningún fallo alcanza
`SesionConsultaRRHH`.

Quedaron verdes:

```text
pruebas focales repetidas
go test del paquete y del módulo
go test -race focal
go vet del paquete
git diff --check y gofmt
verificación de tamaños
Gitleaks del commit
```

La minitarea no modifica puertos, HTTP, PostgreSQL, autoridad, PDP, raíz ni
web. Por sí sola no cambia el avance funcional de O4-05.

# Revisión final CT-000047C1: cápsula de identidad

Fecha: 30 de julio de 2026.

## Corte

```text
base: f791ca5
candidato final: de4653e
integración: 4471e3a
```

Veredicto de dos revisores independientes: **GO**.

| Severidad | Hallazgos abiertos |
| --- | ---: |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Correcciones exigidas durante la revisión

El primer candidato recibió `NO-GO` porque no ligaba el consumo al canal
actual, no revalidaba revocación/caducidad, permitía sustitución y perdía los
errores de contexto.

Una segunda revisión detectó que la misma cápsula podía vincularse a dos
contextos raíz distintos y que faltaban XML, CBOR y YAML. La corrección final:

- conserva la identidad privada y vuelve a ejecutar
  `ProyectarCuentaAutenticada` en cada extracción;
- valida instancia, canal y auditoría sin aceptar selectores libres;
- transporta cápsula y vínculo bajo una clave privada de `context.Context`;
- comparte un `atomic.Bool` entre todas las copias y permite una única
  vinculación mediante `CompareAndSwap`;
- rechaza sustitución, cero, ausencia, cruce, adulteración, revocación,
  caducidad y contexto terminado;
- bloquea JSON, texto, binario, Gob, XML, CBOR y YAML en ambos sentidos;
- redacta `fmt` y `slog`.

La extracción solo recibe `context.Context`, por lo que puede implementar la
autoridad del caso de uso sin introducir HTTP, cookies, cabeceras o
contratación temporal en C1.

## Pruebas reproducidas

```text
go test -count=50 ./internal/vec/adapters/httpseguridad
go test -count=10 -race ./internal/vec/adapters/httpseguridad/...
go test -count=1 ./...
go vet ./...
gofmt
git diff --check
gitleaks
```

La carrera de 64 vinculaciones produjo exactamente un éxito. Ambos revisores
reprodujeron las pruebas focales, de carrera, formato y análisis estático; uno
de ellos repitió además 25 veces la carrera. Dirección volvió a ejecutar la
suite global sobre el candidato final.

## Alcance

C1 no define el transporte de la aserción, no selecciona perfil u
organización, no consulta el PDP y no abre listeners. C2 y C3 siguen
pendientes. Las métricas funcionales y el `NO-GO` productivo no cambian.

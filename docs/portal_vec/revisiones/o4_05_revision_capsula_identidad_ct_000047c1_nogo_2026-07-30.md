# Revisión independiente CT-000047C1: cápsula de identidad

Fecha: 30 de julio de 2026.

## Corte revisado

```text
rama: agent/ct69-capsula-identidad-20260730
commits: cc12361, 22bd709
base: f791ca5
```

Veredicto: **NO-GO**.

| Severidad | Hallazgos |
| --- | ---: |
| P0 | 0 |
| P1 | 3 |
| P2 | 2 |

El candidato no se integró ni se publicó.

## Hallazgos P1

1. La cápsula se autocotejaba con el canal conservado al emitirla, pero no con
   el canal real de la petición que la consumía. Una cápsula del canal A podía
   vincularse a un contexto perteneciente al canal B.
2. La sesión y la cuenta solo se revalidaban al emitir. Una cápsula ya creada
   seguía siendo aceptable después de caducar o revocarse la sesión.
3. El resumen SHA-256 solo comprometía el canal, no el contenido completo, y
   una segunda vinculación podía sustituir silenciosamente la cápsula ya
   presente.

## Hallazgos P2

- Vinculación y extracción convertían `context.Canceled` y
  `context.DeadlineExceeded` en un error de sesión genérico.
- Faltaba la matriz negativa de cápsula cero, cruce de servicio y canal,
  adulteración, revocación, caducidad, doble vinculación y redacción en todos
  los serializadores y sistemas de registro.

## Corrección exigida

La cápsula debe conservar privadamente la identidad emitida y solo puede
vincularse o consumirse mediante la misma instancia de
`ServicioIdentidad`, aportando el canal autenticado actual. Cada uso vuelve a
ejecutar `ProyectarCuentaAutenticada`, por lo que revalida política, reloj,
sesión y cuenta contra la autoridad durable. No puede existir un método
alternativo que entregue los datos sin contexto y canal.

Una segunda vinculación se rechaza, la cancelación se conserva exactamente y
las pruebas negativas deben reproducir cada ataque sobre una cápsula ya
emitida.

## Pruebas reproducidas por el revisor

```text
go test -count=1 ./internal/vec/adapters/httpseguridad
go test -count=1 -race ./internal/vec/adapters/httpseguridad/...
go test -count=1 ./...
go vet ./...
gofmt -d <ficheros del corte>
git diff --check cc12361^..22bd709
```

Las puertas existentes eran verdes, pero no cubrían los escenarios que
originan el `NO-GO`. La corrección deberá recibirse como commits adicionales y
ser reproducida de nuevo por un agente distinto del productor.

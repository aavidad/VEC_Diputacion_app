# Revisión de Dirección O2-05

Fecha: 23 de julio de 2026.

## Dictamen

**GO 1/2** para el candidato exacto
`cbe7299127bb747a7553353f86926935ba0a4814`.

Dirección no encontró hallazgos críticos, altos, medios ni bajos en la
corrección ni en la confirmación SQL completa. Este dictamen no autoriza por sí
solo la integración: O2-05 exige un segundo GO independiente.

## Correcciones verificadas

- El canon histórico V3 conserva `RFC3339Nano` y los instantes SQL de seis
  microsegundos se convierten de forma determinista, incluidos ceros finales.
- La capacidad completa se construye mediante el emisor Go productivo y SQL la
  consume sin recalcular ni reparar su MAC.
- El agregado confirma en una única transacción consumo, expediente, versión,
  actuación, auditoría, outbox, reserva y marcador.
- Un fallo inyectado en cualquiera de las ocho escrituras revierte las ocho.
- Una cancelación concluyente antes de `COMMIT` no deja efectos.
- Una respuesta perdida después de `COMMIT` conserva el agregado y permite
  recuperar el mismo recibo mediante reconciliación.
- Un proceso y conexión nuevos obtienen el mismo recibo sin memoria local.
- El replay compara el agregado completo, sus cadenas y prueba durable; no
  rellena ni repara piezas ausentes o adulteradas.
- El propietario de contratación temporal ya no recibe `REFERENCES` sobre
  tablas de la autoridad de atestación y no puede crear una clave foránea entre
  ambos esquemas.
- Migraciones repetidas, retirada protegida, destrucción explícita y
  reinstalación quedan ensayadas.

## Evidencia reproducida por Dirección

```text
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh
go test ./internal/vec/adapters/seguridad/confianzaatestacion -count=20
go test -race ./internal/vec/adapters/seguridad/confianzaatestacion -count=2
go vet ./internal/vec/adapters/seguridad/confianzaatestacion
git diff --check 209ae720..cbe7299
```

Resultados:

- PostgreSQL 18 efímero, sin red: verde;
- pruebas focales y carrera: verdes;
- Gitleaks sobre los catorce commits: cero fugas;
- archivos dentro del máximo de 800 líneas;
- candidato y worktree limpios.

La integración virtual contra el estado actual presenta conflictos únicamente
en los tres documentos vivos de estado. Si llega el segundo GO, se incorporará
la serie funcional y Dirección actualizará esos documentos sobre su versión
actual, sin aceptar versiones antiguas por resolución automática.

## Alcance no acreditado

O2-05 no implementa el adaptador Go O2-06, la composición O2-07, el registro
HTTP O2-08, la conexión web O2-09 ni el E2E O2-10.

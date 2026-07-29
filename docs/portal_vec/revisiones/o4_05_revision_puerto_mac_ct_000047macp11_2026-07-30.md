# Revisión del puerto MAC CT-000047 MAC-P1.1

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente** para el contrato de cálculo MAC con clave no
exportable.

| Elemento | Valor |
| --- | --- |
| Base original | `d8f2671` |
| Candidato inicial | `dd97e3a` |
| Corrección del NO-GO | `cfa1bc6` |
| Commits integrados | `3f63599`, `eb6fbfd` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

Esta revisión sucede al
[NO-GO inicial](o4_05_revision_puerto_mac_ct_000047macp1_nogo_2026-07-30.md)
sin ocultarlo ni reescribir su evidencia.

## Cierre del hallazgo

El constructor valida perfil, instante y longitud `1..32768` antes de clonar
la preimagen o calcular SHA-256. Una carga mayor de 32 KiB se rechaza antes de
recorrerla siquiera para comprobar ceros.

`Validar` aplica el mismo orden y solo calcula la huella tras superar los
límites. La detección de material todo cero usa una acumulación lineal sin
reservar un búfer proporcional.

`TestSolicitudMACCapacidadV3RechazaSobredimensionSinAsignar` compara mediante
`testing.AllocsPerRun` el coste base de validar el perfil con el rechazo de
32.769 bytes y exige que este no añada asignaciones.

## Contrato acreditado

- El perfil público contiene metadatos gobernados, no un secreto, credencial
  o referencia física KMS/HSM.
- El calculador se inyecta preligado a una sola clave, versión y audiencia; no
  acepta selectores libres.
- Solicitud y resultado ligan perfil, instante, preimagen, huella y MAC
  exactos mediante portadores opacos y copias defensivas.
- Los portadores no implementan el calculador ni un exportador.
- Nulos tipados, codecs, `fmt` y `slog` fallan o quedan redactados.
- Las operaciones lentas reciben `context.Context`; el adaptador concreto
  deberá demostrar cancelación, indisponibilidad y ciclo de vida.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron focales veinte veces, detector de
carreras tres veces, el paquete completo `internal/vec/ports`, `go vet`,
formato, revisión del diff, tamaños y Gitleaks. Todo terminó en verde.

## Límite productivo

El puerto no es un adaptador KMS/HSM y no resuelve cómo comprobará PostgreSQL
la capacidad dentro de la transacción final. Continúa vigente la
[decisión MAC/PostgreSQL](../decision_mac_capacidad_y_postgresql_ct_000047_2026-07-29.md)
y su `NO-GO` hasta que Sistemas y DBA elijan y aporten la solución.

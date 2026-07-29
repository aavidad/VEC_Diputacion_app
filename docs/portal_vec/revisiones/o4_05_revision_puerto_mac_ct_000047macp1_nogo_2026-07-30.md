# Revisión del puerto MAC CT-000047 MAC-P1

Fecha: 30 de julio de 2026.

## Resultado

**NO-GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `d8f2671` |
| Commit candidato no integrado | `dd97e3a` |
| P0 | 0 |
| P1 | 1 |
| P2 | 0 |

El candidato permanece fuera de la rama integradora.

## Hallazgo P1

`NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3` clona la preimagen
y calcula su SHA-256 antes de comprobar el límite de 32 KiB. El método
`Validar` también calcula la huella antes de comprobar la longitud. Una entrada
sobredimensionada puede, por tanto, forzar reserva y recorrido proporcionales
antes de ser rechazada.

Esto incumple la barrera común que exige aplicar límites antes de reservar
memoria, recorrer cargas o llamar a conectores.

## Corrección exigida

1. Validar perfil, instante y longitud `1..32768` antes de clonar o calcular la
   huella.
2. Comprobar la longitud al principio de `Validar`, antes de SHA-256.
3. Añadir una regresión que demuestre rechazo sin clon ni asignación
   proporcional para una entrada sobredimensionada.
4. Repetir revisión independiente sobre un nuevo commit candidato.

## Garantías sin hallazgos adicionales

La revisión no encontró otro P1 o P2. El contrato conserva:

- metadatos informativos sin secreto, credencial o referencia física KMS;
- alias público gobernado separado de la correspondencia privada del
  adaptador;
- portadores opacos que no implementan el calculador ni un exportador;
- nulos tipados, copias defensivas, UTC canónico, huellas y ligadura exacta de
  preimagen;
- bloqueo de serialización y redacción de `fmt` y `slog`;
- API hexagonal y vocabulario coherente en castellano.

La cancelación de las operaciones lentas se deberá acreditar también en cada
adaptador concreto.

## Evidencia

Aunque no subsana el P1, el candidato superó focales veinte veces, carrera tres
veces, el paquete completo de puertos VEC, `go vet`, formato, revisión del
diff, límites de fichero y Gitleaks. El revisor no modificó el candidato.

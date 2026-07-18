# Revisión cruzada Orquesta del servicio Go de borradores

Fecha: 2026-07-18  
Estado: **NO-GO; no integrar esta revisión**  
Alcance: alta y actualización de borradores de convocatorias de Bolsa.

## Propósito

Esta evidencia conserva el control previo a integración del contrato Go
DEC-090. Orquesta trabajó en modo de solo lectura: sus agentes no modificaron el
repositorio. La corrección corresponde a un tramo posterior y deberá recibir
otra revisión sobre hashes nuevos.

## Revisión fuente

| Fichero | SHA-256 revisado |
|---|---|
| `internal/modules/bolsa/application/gobiernoconvocatorias/intenciones_operaciones.go` | `1821b497240207aa24f775a53f44e66f7eb3f7a2ae5a249044671e6756202af6` |
| `internal/modules/bolsa/application/gobiernoconvocatorias/contratos_reserva.go` | `aeceffd49c25192dbc42797e23a0e8db779875cf508f35917698e51d6cd500a6` |
| `internal/modules/bolsa/application/gobiernoconvocatorias/servicio_borradores.go` | `44f48ad48d47e912b53dccc944093014d41fe0f07843677901788d300457e27c` |
| `internal/modules/bolsa/application/gobiernoconvocatorias/servicio_borradores_test.go` | `f79d083613d4a0eeee75427940c0656bd1b08195b9c3345b8793522fdf750cb0` |

Los hashes coincidieron antes y después de las dos tandas.

## Tanda primaria

- Goal: `goal:405a74e946ba55bd489cf113a080a3e2`, revisión 8,
  estado `succeeded`.
- AppSpec: generación 1,
  `8efdb7222b6fd6c7abee9f5144aef7a67d239d6435e411be28b1a76a185d006d`.
- Seguridad y protocolo:
  - ejecución `execution:cd97f0a87015bebe1e551adc466eb77a`;
  - artefacto
    `artifact:sha256:4c529e550b0064a1e6e12469296cd7988e336eb2cd45c2f04951adb4e2ac1c7c`;
  - veredicto `NO-GO`.
- Dominio y concurrencia:
  - ejecución `execution:87470e2fcccf3be34b443128de9dacf8`;
  - artefacto
    `artifact:sha256:ccde264d19d15e0d2b19d2d75c0c901fe824b24f2feb403eb115ed0009b74e07`;
  - veredicto `NO-GO`.

## Metarrevisión independiente

- Goal: `goal:85ed2816ddff15cdf3531ee54045dfa2`, revisión 8,
  estado `succeeded`.
- AppSpec: generación 1,
  `9a2d59847d5ee93ffe2ba5dcc3f5da760a6b1b56ae3fa0ca28e6bcae4e4bfaca`.
- Validez factual:
  - ejecución `execution:b2b82c480731e4b3baee4857c2f59dba`;
  - artefacto
    `artifact:sha256:d6f2b87e28e34cc68a228a2a422b75b0af858281adc5010e9966572e57e2328e`;
  - veredicto `NO-GO`.
- Riesgo para administración pública:
  - ejecución `execution:13ea2536580083fe94e99a0c77acbbf3`;
  - artefacto
    `artifact:sha256:8c7acdb782344a669e33cb3aaa5d171b9eb694f86fc16d8d5c3040dfd8ad9a9e`;
  - veredicto `NO-GO`.

Orquesta verificó en los cuatro artefactos la codificación Base64, tamaño,
pertenencia al Goal, ámbito y digest recalculado. Los metarrevisores recibieron
la fuente limpia y los dos artefactos primarios.

## Hallazgos confirmados

1. No existe un camino contractual completo para reconciliar un COMMIT dudoso
   y reclamar por CAS una reserva expirada con revisión y cercado crecientes.
2. Un solo instante se reutiliza en PDP, reserva, sellado y confirmación. La
   validación local puede aceptar una capacidad ya vencida en tiempo real.
3. La atomicidad productiva PostgreSQL/PDP/HSM todavía no está acreditada.
4. La respuesta inmediata y el replay no conservan las mismas comprobaciones
   temporales del recibo.
5. La identidad idempotente no acredita todavía aislamiento por actor y ámbito
   ni continuidad segura durante rotaciones de claves L, F y motivo.
6. Faltan pruebas multiinstancia, crashes entre fases, lease expirado,
   reconciliación, fault injection y comparación nominal completa de L/F.
7. Los errores del PDP mezclan denegación con indisponibilidad.
8. Material V2 de creación no rechaza expresamente todo estado relacionado.

También se confirmó por inspección independiente que construir F después de
crear la versión con la hora del intento vuelve inestable un replay con reloj
avanzado. Relajar la comparación F sería inseguro; la intención semántica debe
estabilizarse antes de los datos generados por la ejecución.

## Falso positivo retirado

Los mensajes `Failed to create stream fd` pertenecían al entorno de lectura y
no a los bytes de la fuente. Los metarrevisores retiraron la afirmación de que
el código no compilaba por esos mensajes.

## Gate para la revisión sucesora

La siguiente revisión no podrá integrarse hasta acreditar conjuntamente:

1. intención idempotente estable, aislada por actor/ámbito y resistente a
   rotación dentro de la ventana de retención;
2. reconciliación antes de retry y reclaim CAS únicamente tras lease expirado
   y rollback/no aplicación demostrados;
3. reloj fresco y relectura autoritativa en reserva, HSM y COMMIT;
4. recibo ligado a L/F, decisión, sellado, cercado y límites temporales, con el
   mismo veredicto inmediato y en replay;
5. transacción PostgreSQL única para agregado, consumo del sellado, auditoría,
   outbox, recibo y estado terminal;
6. separación estable de denegación, indisponibilidad, conflicto e
   indeterminación;
7. pruebas Go con `-race -count=20` y pruebas PostgreSQL multiinstancia con
   restart, replay, expiración, fencing y fallos alrededor del COMMIT;
8. nueva revisión primaria y metarrevisión Orquesta sobre los hashes corregidos.


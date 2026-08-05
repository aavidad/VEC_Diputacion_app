# Revisión documental O1b: corrección final 800

Fecha: 5 de agosto de 2026.

Estado: **GO para corregir exclusivamente O1b**.

Base candidata: `56c0ac079419b187f851de8183d5f87b5b367b71`.

Contrato final: `a6db81808fcf840ad4556bff770a9c9a5eed264d`.

## Trazabilidad

| Corte | Funcional/seguridad | Ledger/reproducibilidad | Resultado |
| --- | --- | --- | --- |
| `878d724` | NO-GO: P0=0, P1=2, P2=0 | GO condicionado | NO-GO |
| `a6db818` | GO: P0=P1=P2=0 | GO: P0=P1=P2=0 | GO final |

El primer corte colocaba las puertas globales después de integrar y no decía
qué umbrales del ledger 790 sustituía. La corrección exige doble GO focal,
puertas globales sobre el candidato, integración, repetición proporcional y
solo después publicación. También sustituye exclusivamente:

```text
previsión 713..783  → 798..800
parada 790          → 800
delta +390          → +400
```

El resto del contrato, matriz, write-set, invariantes, puertas y prohibiciones
permanece vigente.

## Alcance autorizado

- G2 puede corregir únicamente los huecos probatorios documentados;
- G2 se detiene en 800 líneas y no pierde cobertura o diagnóstico;
- runner solo cambia al final los SHA-256 de G2 y binario;
- G1 y las demás unidades permanecen byte a byte invariantes;
- seis fuentes, modo operativo 64 y O2 cerrado;
- si no cabe, se diseña G3 mediante otra decisión; no se toca G1 ni se supera
  DEC-051.

El GO no acepta `56c0ac0`, no integra código y no cambia métricas.

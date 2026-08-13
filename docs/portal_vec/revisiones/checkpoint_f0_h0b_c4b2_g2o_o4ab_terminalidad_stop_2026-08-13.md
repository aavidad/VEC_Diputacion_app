# Checkpoint F0-H0b/C4b-2/G2-O — O4AB-P0-ENMIENDA-TERMINALIDAD-STOP

Fecha: 13 de agosto de 2026.

Estado: **CANDIDATA DOCUMENTAL A REVISIÓN**. No acredita O4A-P4, no autoriza
implementación O4a/O4b/O4c, integración, producción, despliegue ni métricas.

## Corte exacto y write-set

- Base y padre: `5345d5d097b51ab3567983f048feabeceaf2957b`.
- Rama: `trabajo/o4ab-p0-enmienda-terminalidad-stop-20260813`.
- Enmienda: [terminalidad y STOP final](../enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md).
- Write-set: la enmienda anterior y este checkpoint, ambos altas Markdown.
- O4a/O4b publicados, código, pruebas, herramientas, runner, workflows, SQL,
  O3, O4c, `AGENTS.md`, handoffs, roadmap, ledger transversal, métricas y
  candidatos históricos permanecen byte-inmutables.

La base conserva O4B-P0 con doble GO y CI `31546649383`, cinco de cinco
puertas verdes. La enmienda no reescribe ese contrato: define una prevalencia
limitada para corregir dos aristas incompatibles con O4a.

Huellas fijadas para revisión:

| Documento | Líneas | SHA-256 |
| --- | ---: | --- |
| Enmienda O4AB | 234 | `d1edcd4b1468000577577cbe5f86037c64b6b9ade65e3f9d98d4edc9fa2985f9` |
| Decisión O4a | 535 | `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc` |
| Decisión O4b | 443 | `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f` |

## Evidencia que abre la corrección

Objeto rechazado: `O4A-P4-ETAPAS`
`2b7eaf498f8f68a90b66c004f166a62f070b5064`, cuya base es este mismo
`5345d5d097b51ab3567983f048feabeceaf2957b`.

- revisión funcional: `e20a5fe597d9c172d59b99e75878f8e99e196c76`,
  `NO-GO`, `P0=1, P1=2, P2=0`;
- revisión de seguridad: `a62ee60f70315a1fc0d290290b681c9c3f35227b`,
  `NO-GO`, `P0=0, P1=1, P2=0`.

El candidato y ambas actas permanecen separados. Este checkpoint no acredita
ningún descendiente ni sustituye las revisiones.

## Contrato corregido resumido

1. `TERMINAL` post-CONT observado hasta la igualdad de `finGracia` llega A7
   con drenaje cooperativo y cero señal posterior.
2. `GRUPO_PRESENTE` anterior a gracia espera; en igualdad permite una única
   autorización condicional `PARADA_FINAL`.
3. La presencia anterior no prueba el borde. `PARADA_FINAL` ejecuta un
   preflight no recolector después de validar el límite y antes del efecto.
4. Terminalidad en preflight devuelve cardinalidad 0 y raws cero: no hay
   STOP, KILL ni incidente.
5. Presencia en preflight permite un STOP inmediato. Terminalidad posterior
   al STOP devuelve cardinalidad 1 y llega A7 sin KILL ni incidente.
6. STOP final estable, no estable o raw error conserva sus ramas publicadas.
7. STOP inicial permanece distinto: no admite `TERMINAL` ni cardinalidad 0.

O4a conserva la decisión; O4b solo acredita la condición física y ejecuta el
efecto autorizado. Cada syscall mantiene permiso lease separado y
consolidación previa; duda física es OBF, nunca presencia o éxito.

## DAG y bloqueos

```text
O4AB-P0 enmienda
  -> doble revisión + publicación + CI 5/5
  -> nuevo candidato O4A-P4 corregido
  -> doble revisión material
  -> O4B-P1 -> ... -> O4B-P5

O4A-P4 + O4B-P5 + O4C-P0 -> O4A-P5 -> O4C-P1
```

La corrección documental O4C-P0 `de4ee5a11c3611e56ab09899da860ef88647ea0c`
es un corte independiente y no elimina este bloqueo. O4A-P5, O4C-P1, O5 y O6
permanecen cerrados.

## Puertas del productor

Antes del commit local se reproducen:

1. base/padre/rama y write-set exactos;
2. SHA-256 y líneas de enmienda, O4a y O4b;
3. enlaces Markdown locales;
4. `git diff --check`;
5. Gitleaks sobre el rango exacto de un commit.

Go normal/race, gofmt, vet, PostgreSQL y E2E no aplican a un corte de solo dos
Markdown. No se declaran verdes ni se sustituyen por pruebas documentales.

## Revisión requerida

Dos revisores independientes deben releer completos O4a, O4b, esta enmienda,
el checkpoint y los dos NO-GO; reproducir genealogía, hashes, enlaces,
cardinalidades, bordes, permisos y puertas; y emitir sobre los mismos bytes
`GO` o `NO-GO` con `P0/P1/P2`.

El productor no crea actas de revisión, no integra, no hace push, no cambia
porcentajes y no se autoaprueba. Hasta doble GO el estado es candidato y no se
abre un corte material.

# Checkpoint F0-H0b/C4b-2/G2-O — O4AB-P0-ENMIENDA-TERMINALIDAD-STOP

Fecha: 13 de agosto de 2026.

Estado: **CANDIDATA DOCUMENTAL V3 A REVISIÓN**. No acredita O4A-P4, no autoriza
implementación O4a/O4b/O4c, integración, producción, despliegue ni métricas.

## Corte exacto y write-set

- Base: `5345d5d097b51ab3567983f048feabeceaf2957b`.
- Padre V3: `9de3ba320dfa4beede58e5a1d34aa02a3459f073`.
- Rama: `trabajo/o4ab-p0-v3-linealizacion-20260813`.
- Enmienda: [terminalidad y STOP final](../enmienda_f0_h0b_c4b2_g2o_o4ab_terminalidad_stop_2026-08-13.md).
- Write-set V3: la enmienda anterior y este checkpoint, ambos Markdown
  modificados; ningún otro fichero cambia respecto del padre V3.
- O4a/O4b publicados, código, pruebas, herramientas, runner, workflows, SQL,
  O3, O4c, `AGENTS.md`, handoffs, roadmap, ledger transversal, métricas y
  candidatos históricos permanecen byte-inmutables.

La base conserva O4B-P0 con doble GO y CI `31546649383`, cinco de cinco
puertas verdes. La enmienda no reescribe ese contrato: define una prevalencia
limitada para corregir dos aristas incompatibles con O4a.

Huellas V3 fijadas para revisión:

| Documento | Líneas | SHA-256 |
| --- | ---: | --- |
| Enmienda O4AB V3 | 282 | `2b44bd03d0422a4aecff78ad6400873ecb17686901c40a6b0a68bdabd91d769f` |
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

La V1 de la enmienda `a89a3228554f53b32f5d81fc8b0438835f35b0f6`
recibió doble `NO-GO P0=0, P1=1, P2=0`: funcional
`cdcedc44b31f0f997139f0e9e1210f29d1eb08ca` y seguridad
`e7e06423941807a41c40946e5a4af12e1b30bb11`. Sus gates documentales quedaron
verdes, pero ambos revisores demostraron un STOP potencialmente posterior a
`finParadaFinal`. V2 no integra ni altera esas actas.

La V2 `9de3ba320dfa4beede58e5a1d34aa02a3459f073` recibió `GO` de
seguridad `569cc9ccfb94d3f195a6c08340b0e0e10429da96` y `NO-GO funcional
P0=0, P1=1, P2=0` en `876b7c8e330dc28003dab1d0057ca4bc9d83a727`.
V3 conserva el orden seguro y corrige la atribución temporal señalada; no
integra ni altera V2 o sus actas.

## Contrato corregido resumido

1. `TERMINAL` post-CONT observado hasta la igualdad de `finGracia` llega A7
   con drenaje cooperativo y cero señal posterior.
2. `GRUPO_PRESENTE` anterior a gracia espera; en igualdad permite una única
   autorización condicional `PARADA_FINAL`.
3. La presencia anterior no prueba el borde. `PARADA_FINAL` ejecuta un
   preflight no recolector después de validar el límite y antes del efecto.
4. Tras toda evidencia de presencia, una lectura monotónica final exige
   `ahoraFinal < finParadaFinal`; igualdad/vencimiento es OBF, sin resultado,
   cardinal, raw, incidente, STOP ni efecto posterior.
5. La presencia se linealiza en el último sondeo; `ahoraFinal` posterior
   acredita exclusivamente vigencia y no traslada la presencia a ese instante.
6. Con lectura final verde, solo se prepara el permiso lease y STOP es la
   siguiente syscall literal, sin otra lectura, sonda, espera o log.
7. Terminalidad posterior al último sondeo, incluso antes de `ahoraFinal`, es
   posterior a la decisión física y no revoca STOP si el reloj queda verde.
8. Terminalidad en preflight devuelve cardinalidad 0 y raws cero: no hay
   STOP, KILL ni incidente.
9. Presencia en preflight permite un STOP inmediato. Terminalidad posterior
   al STOP devuelve cardinalidad 1 y llega A7 sin KILL ni incidente.
10. STOP final estable, no estable o raw error conserva sus ramas publicadas.
11. STOP inicial permanece distinto: no admite `TERMINAL` ni cardinalidad 0.

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
el checkpoint y los cinco NO-GO más el GO de seguridad V2; reproducir genealogía, hashes, enlaces,
cardinalidades, bordes, permisos y puertas; y emitir sobre los mismos bytes
`GO` o `NO-GO` con `P0/P1/P2`.

El productor no crea actas de revisión, no integra, no hace push, no cambia
porcentajes y no se autoaprueba. Hasta doble GO el estado es candidato y no se
abre un corte material.

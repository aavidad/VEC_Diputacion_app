# Revisión independiente F0-H0b/C4b-2 G2-O/O2a S0/G3

Fecha: 5 de agosto de 2026.

Estado: **GO documental condicionado a publicación y CI verde**.

Base material:
`ef027cf2f94955cdf6de9c091c531a85d05c2e04`, CI `31007941124`, cinco de
cinco puertas verdes.

Corte documental previo:
`c09b1533bafdcc38d22ec73f274800a4f0f685a4`, CI `31009787813`, cinco de
cinco puertas verdes y blobs materiales idénticos a la base.

Contrato:
[enmienda O2a/S0/G3](../enmienda_f0_h0b_c4b2_g2o_o2a_sobre_s0_g3_2026-08-05.md).

## Resultado final

Dos revisores independientes han releído el contrato completo final:

| Perfil | P0 | P1 | P2 | Dictamen |
| --- | ---: | ---: | ---: | --- |
| Funcional, seguridad y privacidad | 0 | 0 | 0 | GO |
| Ledger, aislamiento y reproducibilidad | 0 | 0 | 0 | GO |

El primer perfil comprobó API y estados S0/S1; EOF; errores tipados y
pegajosos; retención opaca; confinamiento serial; matrices; mutantes; dos
ámbitos AST; lista positiva de imports; privacidad; prohibiciones y autoridad.

El segundo perfil reprodujo base, CI, líneas, nueve huellas, dos builds
privados, inventario Go, capacidad del capturador, manifiesto, deltas,
aislamiento, modos cerrados y paradas.

## Historial de corrección

Ningún NO-GO intermedio autorizó código.

### Primera revisión funcional

Resultado: `P0=0`, `P1=5`, `P2=3`.

Se corrigieron:

- distinción entre error O1b/L4, receptor nulo, invariante O2a y uso S1;
- EOF vacío inicial frente al EOF válido desde L2;
- único respaldo inmutable compartido por subcadenas O1a;
- separación de mutantes conductuales y controles estructurales;
- write-set del productor limitado a runner, G1 y G3;
- receptor no concurrente y transición sin estado intermedio observable;
- fronteras canónicas exactas 149/2212;
- secuencia LF aislado más EOF posterior.

### Primera revisión de ledger

Resultado: `P0=1`, `P1=2`, `P2=0`.

Se corrigieron:

- separación entre base material, corte documental y futuro padre autorizador;
- build tag, `GoFiles`, `IgnoredGoFiles` y prohibición de `-tags=ignore`;
- tabla completa de líneas y SHA-256 materiales.

### Segunda revisión funcional

Resultado: `P0=0`, `P1=1`, `P2=0`.

Se separaron los controles AST de fichero completo y grafo productivo; se fijó
la lista exacta `errors`, `fmt`, `strings`; las pruebas quedaron inaccesibles
desde producción y su excepción se limitó a construir fixtures.

### Revisiones finales

Ambos perfiles releyeron el documento completo corregido y devolvieron
`P0=P1=P2=0`, sin hallazgos ni edición.

### Control del mecanismo de autorización

Un control funcional posterior devolvió `P0=0`, `P1=1`, `P2=0`: una parada
todavía decía que el SHA del padre futuro quedaría fijado dentro de la propia
acta, algo imposible antes de confirmar el commit que la incluye. Se sustituyó
por el mecanismo no circular vigente: dirección comunica el SHA completo tras
confirmar y publicar; el productor lo registra en la evidencia antes de editar
y acredita que coincide con HEAD.

Los dos perfiles revisaron después contrato y acta completos. El perfil de
ledger devolvió `P0=P1=P2=0`. El perfil funcional confirmó que el contrato ya
no tenía defectos y señaló únicamente que este control faltaba en el historial
del acta (`P0=0`, `P1=0`, `P2=1`); esta sección cierra esa omisión. El control
final posterior debe devolver `P0=P1=P2=0` antes de confirmar.

## Autorización limitada

La autorización solo será efectiva cuando el commit que contenga contrato y
acta esté publicado y su CI termine completamente verde. Dirección comunicará
al productor el SHA completo de ese commit, que deberá ser el padre exacto del
candidato, y el productor acreditará antes de editar:

1. igualdad de HEAD con ese SHA;
2. igualdad de los blobs materiales con las nueve huellas de `ef027cf`;
3. CI verde de base material y padre documental.

El write-set material queda limitado a runner, G1 y G3. No se autoriza O2b,
FD 9, procesos, Bash, ACK, `ARMAR`, terminales, señales, modo operativo, SQL,
migraciones, aplicación ni documentación transversal.

El candidato requerirá después dos revisiones de código independientes,
`P0=P1=P2=0`, todas las puertas del contrato, integración, publicación y CI
verde. Este GO no modifica F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` ni el `NO-GO` de producción.

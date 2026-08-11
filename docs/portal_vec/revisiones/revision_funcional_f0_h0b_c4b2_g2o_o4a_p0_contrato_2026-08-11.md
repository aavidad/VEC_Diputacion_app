# Revisión funcional O4A-P0-CONTRATO

Fecha: 11 de agosto de 2026.

Estado: **GO, P0=0, P1=0, P2=0**.

## Snapshot y aislamiento

Revisión independiente y read-only desde worktree propio. El productor no se
autoaprobó. Se releyó íntegramente la decisión de 535 líneas, SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`,
sobre base P7 `0029cfe03b2f2f637169be8340985e38b1fa6557`.

## Resultado

Se verificaron entrada C5 y ownership lineal; preservación de raw CONT y
primera observación; causa CAS única e incidente de cierre separado;
precedencias raw→CONTROL→señal→pidfd→reloj; bootstrap, 180 s y subplazos sin
reinicio; permisos lease por lectura; autómata total, partición TERM/CONT y
KILL acotado; fronteras O4a/O4b/O4c; DAG, matriz, mutantes y residuos.

Los NO-GO V1--V4 detectaron contradicciones de latch, consolidación, deadlines
y etapas. Todos quedaron corregidos causalmente en V5. La relectura final no
detectó API pública, getter, PII/tenant, efecto O4b/O4c ni cambio de O3.

## Puertas

Base, remoto, padre y hashes O3c/ledger exactos; enlaces locales y
`git diff --check` verdes; write-set exclusivamente documental y O3/P6
byte-inmutables. No se ejecutó implementación porque P0 solo define contrato.

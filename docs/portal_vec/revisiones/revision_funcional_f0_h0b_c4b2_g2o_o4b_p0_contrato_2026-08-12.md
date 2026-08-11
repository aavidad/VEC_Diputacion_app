# Revisión funcional independiente — O4B-P0-CONTRATO

Fecha: 12 de agosto de 2026.

Tarea: revisión funcional independiente de `O4B-P0-CONTRATO`.

Estado: **GO**. `P0=0`, `P1=0`, `P2=0` sobre el snapshot V3 completo.

## Objeto exacto revisado

- Base: `1946b671afbcc47c681d6d6bb22e3c1122935247` (`O4A-P3 arbitra una
  ronda inmediata no recolectora`).
- Decisión candidata:
  `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4b_senales_grupo_2026-08-12.md`,
  443 líneas, SHA-256
  `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`.
- Checkpoint candidato:
  `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4b_p0_contrato_2026-08-12.md`,
  48 líneas, SHA-256
  `1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45`.

## Resultado de la revisión

La decisión conserva la frontera obligatoria: O4a decide causa, plazos y
etapa; O4b solo consume la autorización opaca y ejecuta como máximo STOP,
TERM, CONT o KILL de grupo por pidfd primario; O4c conserva Wait, drenaje,
TERMINAL y liberación. No introduce PID/PGID como autoridad, fallback,
recogida, cierre/escritura, creación de plazos, parser nuevo ni una API
pública.

La máquina OB0--OBF es total y lineal: consume una vez
`EMITIDO→CONSUMIENDO→CONSUMIDO`, no devuelve resultados parciales y convierte
propiedad, identidad, lease o plazo no acreditables en fatalidad heredada. Cada
syscall, incluidas las sondas, queda subordinada a un permiso lease distinto y
consolidado antes de interpretar su raw; `EINTR` consolidado cuenta como intento
y no habilita reintento.

La tabla de etapas respeta el autómata O4a. STOP solo devuelve `ESTABLE` o
`NO_ESTABLE`, normalizando la terminalidad durante STOP a la única fila O4a
admisible. TERM con raw cero exige una segunda comprobación estricta de
`finGracia` antes de CONT; si vence, termina fatal sin CONT ni resultado
parcial. KILL queda como la siguiente syscall tras su comprobación temporal y
no tiene sonda posterior. La evidencia posterior reutiliza exclusivamente las
primitivas O3 publicadas y sigue siendo no recolectora.

El DAG queda acíclico y respeta la autoridad de O4a: `O4A-P4` condiciona solo
la materialización `O4B-P1`; `O4C-P0` depende de los contratos O4a y O4b, no
de O4A-P4 material. Se conserva el bloqueo de O4A-P4 hasta el cierre de este
P0 y de toda implementación O4b hasta su publicación y CI.

## Comprobaciones reproducidas

- SHA-256 y número de líneas de ambos artefactos V3.
- SHA-256 de O3a, O3b, O3c y O4a, todos coincidentes con los declarados.
- Existencia de enlaces Markdown locales y ancestro de O3a/O3b/O3c/O4a,
  O4A-P1, P2 y P3 en la base exacta.
- `git diff --check --no-index` sobre ambos Markdown: sin errores de espacios.
- Lectura integral de los dos artefactos V3 y contraste con las decisiones
  O3a, O3b, O3c y O4a, además de las primitivas reales reutilizadas.

## Límites

Es una revisión documental. No acredita implementación O4b, conductor,
mutantes, E2E, producción, despliegue ni el cierre de O4A-P4/O4C-P0. Gitleaks
no estaba instalado en el entorno de revisión; no se ha afirmado esa puerta.

Seguridad, privacidad, i18n y accesibilidad: no se introducen datos personales,
secretos, interfaz visible ni texto de producto; la decisión mantiene
autoridades opacas y denegación fatal ante incertidumbre.

Siguiente tarea desbloqueable: tras la segunda revisión independiente,
publicación y CI 5/5 del mismo SHA, dirección podrá asignar
`O4A-P4-ETAPAS`; no queda autoasignada.

Revisión independiente: funcional, rama
`revision/o4b-p0-funcional-20260812`.

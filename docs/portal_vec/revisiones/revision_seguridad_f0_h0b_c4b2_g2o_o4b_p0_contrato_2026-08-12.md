# Revisión de seguridad O4B-P0-CONTRATO

Fecha: 12 de agosto de 2026.

Estado: **GO, P0=0, P1=0, P2=0**.

## Snapshot y aislamiento

Revisión independiente, completa y read-only desde worktree propio, creado
desde la base exacta `1946b671afbcc47c681d6d6bb22e3c1122935247`. Se auditó
la decisión de 443 líneas, SHA-256
`675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`, y su
checkpoint de 48 líneas, SHA-256
`1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45`, sin
editar el worktree productor ni esos dos documentos.

## Comprobaciones

- señalización exclusiva por pidfd primario, con flag de grupo literal y sin
  PID/PGID numérico, fallback, promoción de reserva ni cuarta referencia;
- autorización opaca one-shot, CAS lineal, TID/owners/generaciones/pending y
  custodia física acreditados antes de todo syscall;
- permiso lease distinto para cada syscall, consolidado antes de interpretar
  raw; duda de permiso, identidad o consolidación conduce a OBF no retornable;
- deadlines monotónicos prestados, sin recreación: KILL es inmediato y el
  borde posterior a TERM queda sellado por una segunda lectura que fataliza
  sin CONT ni resultado parcial;
- evidencia estrictamente no recolectora, con dos muestras completas para STOP
  y sin Wait, waitid, drenaje, cierre ni escritura de recursos de O4c;
- resultados cerrados y consumibles: STOP normaliza terminalidad a
  `NO_ESTABLE`, y duda física tras TERM/CONT es OBF, por lo que no queda una
  arista A5 sin consumidor;
- fatalidad 65/EOF/stdout=0/stderr=0 sin efecto posterior, autoridad sin
  getters ni serialización, y ausencia de secretos, PII, red, SQL o producción;
- DAG O4A/O4B/O4C acíclico, mutantes y matriz que cubren el vencimiento
  post-TERM, carrera, residuos y APIs prohibidas.

Las huellas de O3a, O3b, O3c y O4a declaradas por el snapshot coinciden con
los documentos congelados de la base. Enlaces locales, formato y
`git diff --check` se verificaron sin hallazgos. Gitleaks no está instalado
localmente; sigue siendo una puerta obligatoria antes de cierre remoto.

# Revisión de seguridad O4A-P1B-CONTRATO-SELLOS

Fecha: 11 de agosto de 2026.

Resultado: **GO**, `P0=P1=P2=0`.

## Alcance y procedencia

Revisión independiente y de solo lectura sobre el productor, realizada desde
un worktree propio basado exactamente en P1
`0e750d41cbf72b0f0952341fa18e25474c3269fb`. La rama remota
`trabajo/o4a-p1-autoridad-20260811` resuelve al mismo SHA. El contrato O4a
publicado tiene SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.

Bytes documentales finalmente aprobados:

- decisión P1B: SHA-256
  `e279786ccbd302a1d9fc6bddf53013d98855b0f4e5b7839dbc936a76c203fba9`;
- checkpoint P1B: SHA-256
  `d7491d9cc58a52ce87ecabcf676a79f477c67add14bb997431e23718f8db763b`.

La API remota confirma que la CI P1 `31516149966` corresponde a la rama y SHA
anteriores, terminó `completed/success` y contiene cinco jobs concluidos en
`success`.

## Reproducción del defecto

La revisión directa de O3c y P1 confirma el NO-GO: O3c instala `senal_raw`
cuando su comprobación deja de ser verde, pero P1 exige palabra igual a
baseline y signo cero. P1 tampoco conserva la palabra observada. Para
`control_raw`, O3c deja el canon en CONTROL y en `custodia.primeraCausa`, pero
P1 conserva solo el puntero mutable, insuficiente para una traducción posterior
sin relectura.

## Propiedades de seguridad acreditadas

- El sello añade solo una palabra completa y un enum CONTROL cerrado; no copia
  errores, buffers, texto libre, nonce, ticket, PID, pidfd ni autoridad.
- Una única carga pre-CAS es el punto de linealización histórico. No hay
  segunda lectura post-CAS ni captura perezosa en P2.
- Baseline conserva estado 2 y signo cero. La palabra observada conserva estado
  2 y no admite retroceso de contador. Solo delta uno con INT o TERM es exacto;
  cualquier pareja raw ambigua queda destinada a `INCIDENTE`, nunca autoriza
  por disponibilidad o inferencia.
- Los otros cuatro discriminantes siguen exigiendo palabra exactamente igual a
  baseline y signo cero.
- CONTROL terminal funcional exige canon cerrado y coincidencia exacta entre
  `control.causa` y `custodia.primeraCausa`.
- CONTROL activo solo normaliza `PROTOCOLO_65` si permanece íntegro, en S3,
  con causa, primera causa y fallo vacíos. S1, S2, terminal interno, punteros
  residuales o combinación imposible producen AF.
- Un discriminante no CONTROL exige enum `VACIO` y no inspecciona ni copia una
  causa CONTROL.
- Owners O4A, CAS irreversible 2→3, anti-alias/replay, custodia, recursos,
  primera observación y raw CONT permanecen separados e invariantes.
- No se autoriza syscall, reloj, poll, señal, Wait, drenaje, TERMINAL,
  liberación, parser, getter, ticket, API, log, serialización, concurrencia ni
  código P2/P3.

## Historial de hallazgos

V1 recibió `NO-GO P1`: exigía a la vez una carga única y una relectura
post-CAS; aceptaba sin relación suficiente palabra/baseline; y rechazaba como
AF el CONTROL activo que puede representar presupuesto o EINTR.

V2 recibió `NO-GO P1`: atribuía de forma incorrecta delta cero a una salida O3
legítima y el predicado de CONTROL activo también admitía fases S1/S2. V3
describe delta cero solo como pareja raw ambigua destinada a `INCIDENTE` y
exige fase S3 exacta. Todos los hallazgos quedaron corregidos.

## Límites y puertas

El write-set de `O4A-P1C-SELLOS-RAW` queda limitado al Go P1 y su prueba
existentes. P2 solo podrá reabrirse tras doble GO y CI 5/5 propios de P1C; P3
y fronteras posteriores siguen bloqueados.

Se reprodujeron base y rama remota, SHA del contrato, CI P1 5/5, enlaces
locales, hashes y `git diff --check`. Gitleaks no está instalado en este
entorno revisor; la puerta remota `puerta-secretos` de P1 terminó `success` y
la nueva rama documental deberá superar esa puerta antes del cierre.

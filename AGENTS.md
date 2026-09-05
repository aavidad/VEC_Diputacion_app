# Instrucciones del repositorio para agentes

## Punto operativo vigente — desarrollo local, 5 de septiembre de 2026

Este bloque sustituye las ubicaciones y órdenes históricas inferiores.
Lea las instrucciones de desatasco del operador y el plan vigente en
`ESTADO_PROYECTO.md`; solo Contratación temporal y dependencias imprescindibles.
La bandeja y el análisis se publicaron en `b2effba`; esta entrega añade el
registro de respuesta declarada por RRHH. La línea de trabajo sigue siendo
`trabajo/ct-app-llamamiento-b4a-20260905`, ahora en el equipo local del operador,
worktree `.worktrees/ct-app-llamamiento-b4a-20260905`.
Las dos bases sintéticas están aquí, con consultas instaladas. Principal:
recorrido (8443/55433), 50 solicitudes y análisis desde una solicitud existente
v1, recibo v2 único conservado tras reiniciar aplicación y PostgreSQL.
Secundaria: consultas (8444/55432), 21 expedientes; no mezclar ambas bases.
Las instancias remotas de desarrollo están detenidas y conservadas. No crear
otra línea ni reactivar allí programación; el WIP ajeno D1 sigue intacto.
Rutas privadas y comandos exactos quedan en la bitácora local, fuera de Git.
Registro de respuesta demostrado: navegador `201`, recuperación `200` tras
reiniciar aplicación y PostgreSQL, mismo justificante/recibo/fecha y un solo
registro, historial y evento. Material cambiado con la misma clave: `409`.
CT56 y autorización14 están instaladas en ambas bases; no reaplicarlas.
La comparación de fechas equivalentes está corregida y no queda diagnóstico
temporal en SQL. La declaración conserva referencia y SHA256 del correo,
no su contenido ni su custodia; no resuelve aceptación/renuncia ni cambia Bolsa.
Siguiente corte: aceptación válida, reutilizando la resolución existente y
comprobando el justificante con permiso propio. No convertir una declaración
en aceptación automática. Continúan cinco pasos completos y parte del sexto.
No reconstruir bandeja ni análisis ya cerrados. No reabrir O3a.

## Prioridad vigente

La prioridad es el módulo `contrataciontemporal`, basado en el procedimiento
remitido por RRHH. Bolsa no se borra: mantiene convocatorias, candidaturas,
posiciones, reglas y llamamientos.

Si una tarea de contratación temporal necesita una capacidad común de VEC,
Bolsa, Personal, documentos o firma:

1. se define una tarea dependiente y acotada;
2. se implementa en el módulo que posee esa autoridad;
3. se prueba e integra;
4. se vuelve inmediatamente al camino crítico de contratación temporal.

No se amplía otro módulo por conveniencia ni se cambia la prioridad sin
instrucción de dirección.

## Checkpoint operativo vivo — O6 y cableado O2 — 31 de agosto de 2026

Este bloque prevalece sobre los bloques cronológicamente anteriores o
históricos que aparecen después en este archivo; nunca prevalece sobre relevos
futuros. La base y el `HEAD` de producto publicados al abrir el corte son
exactamente `3a8550eac9324168594bc1e36378015922ec5c4b`. Este documento parte de
ese mismo commit en la rama `docs/ct-checkpoint-o6-r3a-rutas-20260831`; no
integra ni publica por sí solo ningún cambio de producto.

### Capability, invariante, write-set y condición contable

- Capability: alinear la autoridad operativa viva con hechos ya integrados y
  publicados, cerrar documentalmente solo `O6-02` y preservar toda deuda real.
- Invariante: una sola subida contable pertenece a `O6-02`. `R3A`,
  `O5-rutas`, `O4C-P0` y `RUTA-A` son subcortes: no cierran `O2-06`, `O5-01`,
  `O2-07`/`O2-08`, `O4C-P1` ni una vertical E2E.
- Write-set: este `AGENTS.md`,
  `docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md` y
  `docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md`.
- Condición contable: el cierre formal anterior es `19/46` (`41,3 %`). Solo
  después de integrar y publicar el hash exacto de este checkpoint pasa a
  `20/46` (`43,5 %`), exclusivamente por `O6-02`.
- Siguiente corte documental: registrar el cierre de `R3B`/`R3C` únicamente
  cuando estén integrados; este checkpoint no lo anticipa.

### Genealogía local acreditada en el producto publicado

- `CT-O2-06-R3A`: candidato `1babd86ab891349640657c7e98305958f55e2c10`,
  integrado mediante `2b96b64602ba82c904dbf1a0532542f50db3f032`.
- Las piezas propias integradas de `O6-02` son
  `433df1f8b5f8dab670a3d7ee2a5a617fea6b35c9` (`CT-LITE-O6-02A`) y
  `908e7907a1c838dd4c0a508c73123ffa22571ec6` (`CT-LITE-O6-02B-HTTP`).
- La cadena continúa con `02221b47a126762250d4fda791d2b0276e72d713`
  (`CT-LITE-O6-03-GO`), adaptador de ejecuciones;
  `fc8ab4616630749fcb8c7d912bc31bf8fdacb00f` (`CT-LITE-O6-REM-01`), replay
  autorizado de selección; y
  `d72254e36eb4737db4c44b2481522cac2f5260f8`
  (`CT-LITE-O6-REM-02-R8`), ejecuciones revisadas, integrado mediante
  `ca71902192afe3448e71eadd3fee125b52ca11ef`. La cadena completa permite a
  Dirección contabilizar `O6-02`; estos tres seams posteriores no acreditan
  por sí solos comunicación, aceptación, renuncia y siguiente candidato, por
  lo que `O6-03` continúa abierta.
- `O5-rutas`: candidato `ffc0717b59d5e07b6be272179e00ce9d4882c48e`,
  integrado mediante `de2c9be8ea25c25a4e4173d1fdf6f5dcdfb769c8`.
- `O4C-P0`: candidato `5555885d7a57d473ac7b7f224bc7a62c0a7bee01`,
  integrado y publicado mediante
  `7945de3968c118633e3a15ba5a97145939050ffb`.
- `CT-O2-RUTA-A-R1`: candidato
  `8bb15bcc78d0c002cff9cd17099db0945805b674`, integrado y publicado en
  `3a8550eac9324168594bc1e36378015922ec5c4b`.

### Métricas y límites vigentes del checkpoint

- Cierre formal previo: `19/46` (`41,3 %`).
- Cierre formal tras integrar y publicar este checkpoint: `20/46` (`43,5 %`).
- Cierre técnico conservador: `22/46` (`47,8 %`).
- Primera vertical: `5/10`; `O2-06` continúa activa.
- Verticales E2E productivas completas: `0`.
- Aplicación arrancable: `NO`.
- Producción: `NO-GO`.

Formal, técnico y E2E son magnitudes distintas. Una ruta aislada, un contrato,
un adaptador o una prueba no acreditan composición raíz, persistencia, web ni
una vertical productiva.

### Orden de reanudación

1. `O2-06` sigue activo. `R3A` aporta la candidatura técnica opaca, no un
   implementador productivo; `R3B` pertenece a una producción candidata
   separada y todavía no está integrada.
2. `O5-01` sigue abierto. Las rutas integradas no acreditan raíz,
   persistencia ni web.
3. `O6-02` queda cerrado documentalmente solo por este checkpoint y su cadena
   exacta. `O6-03` continúa abierto.
4. `O4C-P0` está integrado y publicado; `O4C-P1` permanece bloqueado.
5. `RUTA-A` está integrada de forma aislada, sin composición raíz, y no cierra
   `O2-07` ni `O2-08`.

## Relevo operativo histórico — corte previo del 31 de agosto de 2026

Este bloque se conserva como historia del corte anterior y queda sustituido por
el checkpoint operativo vivo precedente. Sus referencias a sesiones,
worktrees, limpieza, métricas y siguiente tarea no describen el estado vigente.

### Producto e integraciones acreditadas en aquel corte

- Repositorio remoto: `/srv/fabrica/proyectos/VEC_Diputacion_app`.
- Worktree de producto:
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-producto-ligero-20260821`.
- Rama de producto: `integracion/ct-producto-ligero-20260821`.
- `HEAD`, referencia local de `origin` y rama remota coinciden exactamente en
  `5af095d7c1e9cd095eaf653d544fc4fafc098109`; el producto está limpio. No
  equivale a despliegue ni autoriza producción.
- La secuencia ya integrada y publicada en la rama de producto es:
  - `b2390a5179ec5cb807707766d90a8188fc3f4c66`: `CT-LITE-O7-06A`,
    recuperación de GINPIX;
  - `d69b4ea0325f58909f5c2cbe86ee8dadfb9770aa`: `CT-LITE-O8-02B`,
    cierre administrativo HTTP;
  - `c1afe81b164e3a9f86eb7a7a2fd5b014317db711`: `CT-LITE-O7-06B`,
    puerto de API recuperable de GINPIX;
  - `00bf61ac77fc358dff668042d4ef78747d3371c8`: `CT-LITE-O8-02C`,
    composición de rutas de cierre administrativo;
  - `3f72adc33d8a44203ae5785e14b033668975b266`: `VEC-DOC-CONS-01`,
    política de conservación documental, con revisión independiente `GO`; y
  - `02221b47a126762250d4fda791d2b0276e72d713`:
    `CT-LITE-O6-03-GO`, adaptador de ejecuciones, con revisión independiente
    `GO`; y
  - `64cede515db67cb2cf236ded95518fa67c27f9bf`:
    `CT-LITE-O8-03B`, consumo de la política de conservación, con padre exacto
    `3f72adc33d8a44203ae5785e14b033668975b266` y revisión independiente
    `GO`, `P0=0`, `P1=0` y `P2=0`. Se integró y publicó en producto mediante
    el merge `5af095d7c1e9cd095eaf653d544fc4fafc098109`.
- Se verificó que `64cede515db67cb2cf236ded95518fa67c27f9bf` es ancestro del
  merge de producto. Solo después se retiraron el worktree, la sesión
  `vec-produce-o8-03b` y las ramas temporales local y remota
  `trabajo/ct-lite-o8-03b-consumo-20260831`. No queda trabajo O8 pendiente de
  ese productor.

### Capability, invariante y write-set de este relevo

- Capability: corregir y conservar un relevo operativo verificable que permita
  continuar el camino crítico sin reconstruir ni duplicar trabajo ya integrado.
- Invariante: este corte documental no cambia producto, no convierte trabajo
  activo o SQL pendiente en capacidad cerrada y conserva denegación por defecto,
  autoridades únicas, trazabilidad y revisión independiente.
- Write-set: únicamente el `AGENTS.md` de
  `trabajo/ct-relevo-operativo-20260831`; ningún otro archivo.
- Siguiente corte: cerrar un candidato focal de consultas y rutas `R2`,
  someter su hash exacto a revisión independiente e integrarlo únicamente si
  recibe `GO`.

### Trabajo que estaba activo en aquel corte

- Sesión `vec-produce-r2-consultas`, worktree
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-lite-r2-consultas-rutas-20260831`,
  rama `trabajo/ct-lite-r2-consultas-rutas-20260831`, desde
  `5af095d7c1e9cd095eaf653d544fc4fafc098109`.
- Sesión `vec-test-o6-pg`, worktree
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-test-o6-pg-20260831`,
  en `detached HEAD` exacto
  `93b2fed3849f70b804f739f4772c6e010cdc632d`, con PostgreSQL 18.4 aislado
  para la puerta dinámica obligatoria.
- Sesión `vec-fix-relevo`, este worktree
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-relevo-operativo-20260831`,
  rama `trabajo/ct-relevo-operativo-20260831`; su único write-set es este
  `AGENTS.md`.

No fusionar, rebasar, borrar, limpiar ni reutilizar esos worktrees o ramas
mientras sus sesiones estén activas. Revalidar su estado vivo antes de actuar.

### Trabajo concluido sin candidato

- La auditoría de solo lectura recogida por `vec-produce-o6-next` concluyó que
  no existe otro seam `O6` no SQL honesto. La sesión, su worktree y su rama
  fueron retirados sin cambios; no hay candidato que esperar, revisar o
  integrar de ese carril.
- La auditoría `vec-audit-completion` también ha terminado y ya no constituye
  trabajo activo ni autoridad de implementación.

### Evidencia durable preservada y candidato rechazado

- `CT-LITE-O6-03` PostgreSQL queda preservado en
  `origin/trabajo/ct-lite-o6-03-ejecuciones-pg-20260831`, commit
  `93b2fed3849f70b804f739f4772c6e010cdc632d`; no se integra sin la prueba
  obligatoria en PostgreSQL real desechable.
- `VEC-DOC` SQL R4 queda preservado en
  `origin/trabajo/vec-doc-registro-autoridad-sql-r3-20260831`, commit
  `2536f32ccbeb915724671d7a25f204a31f8d2310`; tampoco se integra sin su
  prueba PostgreSQL real desechable.
- El candidato rechazado `444ff6ebf0d1a395c88207ca92e78e63d4b61dc8`
  permanece en el worktree
  `/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/vec-doc-esquema-efectos-v4-20260831`,
  rama `trabajo/vec-doc-esquema-efectos-v4-20260831`. No integrar, borrar,
  limpiar ni usar como base; se conserva como evidencia del `NO-GO`.

### Métricas conservadoras de aquel corte y estado de la interfaz

- Cierre formal: `19/46`, `41,3 %`.
- Cierre técnico: `21/46`, `45,7 %`.
- Avance operativo conservador: `33 %`.
- Arrancable: `NO`.
- Verticales productivas completas: `0`.

Estas magnitudes no son equivalentes: el cierre formal exige las autoridades y
aprobaciones aplicables; el técnico cuenta capacidades verificadas dentro de
su alcance; el operativo estima el recorrido utilizable; y una vertical solo
es productiva cuando queda compuesta de extremo a extremo con dependencias
reales. Ninguna cifra autoriza producción ni datos reales.

La auditoría de interfaz ejecutó `103/103` pruebas Node verdes, pero acredita
solo una superficie sintética. El manifiesto de Contratación temporal no está
registrado, la raíz real no consume sus rutas y el frontend real permanece
bloqueado hasta que la composición fail-closed las haga alcanzables con sus
autoridades verdaderas.

### Bloqueo raíz exacto

La aplicación no está cerrada al cien por cien. El camino raíz exige todavía,
sin sustitutos ficticios ni atajos en HTTP:

1. componer en la raíz real el punto de decisión de políticas y fijar una única
   autoridad de rutas, con denegación por defecto;
2. disponer de un registro durable de operaciones GINPIX; un recibo local no
   acredita por sí solo confirmación del sistema externo;
3. ejecutar el cierre como una transacción durable que una autorización,
   versión, cambio de estado, auditoría y outbox; y
4. cerrar el contrato de entrada de `O7-06` en la composición real antes de
   declarar alcanzable el flujo externo.

Una ruta registrada, un puerto o una prueba aislada no cierran estos cuatro
puntos si la composición raíz no los consume con sus autoridades reales.

### Ciclo obligatorio de entrega

Para cada corte: producir un commit coherente y limpio; someter ese hash exacto
a revisión independiente; integrar solo con `GO`; repetir las pruebas focales
y `git diff --check` en producto; publicar la rama de producto; verificar que
`HEAD`, la referencia de seguimiento y el hash remoto coinciden; y solo
entonces retirar el worktree y la rama local ya integrados. Una rama pendiente
de PostgreSQL o con `NO-GO` se preserva y no entra en esa limpieza.

### Orden de reanudación de aquel corte

1. Revalidar producto, `origin`, limpieza, sesiones, rutas, ramas y hashes; no
   reconstruir trabajo existente.
2. Cerrar el candidato focal `R2` de consultas y rutas; revisar su hash exacto
   de forma independiente e integrarlo y publicarlo solo si recibe `GO`.
3. Obtener `GO` dinámico en PostgreSQL 18.4 real desechable y `GO` estático
   independiente para `CT-LITE-O6-03` SQL antes de considerar su integración.
4. Someter después `VEC-DOC` SQL a su propia validación PostgreSQL real
   desechable y revisión independiente; no hereda el resultado de O6.
5. Registrar el manifiesto y componer rutas y políticas en la raíz real con
   denegación predeterminada y una única autoridad de rutas.
6. Conectar el frontend de lectura únicamente después de esa composición
   fail-closed, con contrato, seguridad, accesibilidad y revisión visual.
7. Cerrar las autoridades corporativas y los efectos durables restantes,
   incluidos GINPIX y la transacción de cierre, antes de declarar una vertical
   arrancable o productiva.

Tras cada integración se actualiza este relevo con hashes verificables y solo
se limpia lo ya integrado. Las ramas SQL pendientes y el candidato con
`NO-GO` se preservan.

## Relevo operativo histórico — 9 de agosto de 2026

- Worktree obligatorio:
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-stable-docs`.
- Rama integradora: `integracion/ct-o4-04e-20260726`. El directorio raíz
  conserva la rama histórica y no se edita.
- O2b queda publicado en `c1ca5aa`, con material y evidencia en el par exacto
  `d86aea8`–`4b39265`.
  Código y evidencia recibieron doble `GO`, `P0=P1=P2=0`: 31/31 mutantes,
  analizador AST tipado, dos builds Go 1.26.5 reproducibles, H0 PostgreSQL
  18.4, puerta global y Gitleaks verdes, con cero residuos. La
  [revisión final](docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o2b_codigo_final_2026-08-09.md)
  conserva el ledger y los límites. No abre Bash ni el modo operativo.
- O3a-P1 queda publicado en `ce0848e`, sobre la cadena exacta `758e66f` →
  `a1aeab7` → `f3a1e96` → `ce0848e`; la CI `31298943127` terminó 5/5.
  El material R-only `a1aeab7` conserva R en 702 líneas y SHA `7ad65a66…`; la
  evidencia `f3a1e96` fija seis artefactos, la matriz 100/600 + 10/10 + 1, H0
  PostgreSQL 18.4, calidad global, Gitleaks y residuos cero. Dos revisores
  independientes dieron `GO`, `P0=P1=P2=0`. La
  [revisión final](docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o3a_p1_codigo_final_2026-08-09.md)
  conserva hashes, ledger y límites. El siguiente único trabajo de desarrollo
  es corregir y revisar el contrato O3a completo sobre `ce0848e`; O3a,
  `Start` y mapa FD siguen bloqueados hasta doble GO documental, publicación y
  CI 5/5 de ese contrato.
- C4b-1 queda acreditado en la rama integradora hasta `c34db61`: decisión de señales
  `fb45b93`, código `84de42f`–`ffce19c` y acta
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b1_senales_regimen_2026-08-02.md`.
  Obtuvo doble `GO`, `P0=P1=P2=0`, 600 ráfagas reales, H0 PostgreSQL 18.4
  y residuos cero. El commit posterior que modifica este `AGENTS.md` es solo
  el relevo de cierre.
- La evidencia histórica Q5a queda aceptada sobre `649ee46` y su
  `docs/portal_vec/revisiones/revision_f0_h0b_q5a_captura_supervisor_2026-08-05.md`:
  tres `GO` independientes, `P0=P1=P2=0`, captura privada exacta de cinco,
  build Go cerrado, autoprueba ABI pidfd real, runner 789, D2d 145, adaptador
  527, Go 131 y residuos cero. La corrección posterior de empaquetado ocupa
  132 líneas y queda documentada en
  `docs/portal_vec/correccion_f0_h0b_q5a_paquete_go_2026-08-05.md`; recibió
  doble `GO`, `P0=P1=P2=0`, sobre `6a95b07`. No se ejecutaron Docker,
  PostgreSQL ni E2E.
- C4b-2/G1 queda integrado localmente en `f3d928d`–`d28d37d`. Dos revisores
  independientes dieron `GO`, `P0=P1=P2=0`; dirección reprodujo 100/100
  autopruebas, y las pruebas globales, carrera, `go vet` y
  `scripts/verificar_calidad.sh` terminaron verdes. El supervisor ocupa 754
  líneas y acredita primitivas pidfd y rollback sintético, pero no el protocolo
  operativo, Docker, PostgreSQL ni E2E.
- C4b-2/G2-S queda integrado localmente en `452f4f0`–`5808e18`, con ledger
  aceptado en `2409620`–`0e179e1`, doble `GO`, `P0=P1=P2=0`, G1 100/100,
  dos builds privados reproducibles y puertas globales verdes. El runner ocupa
  exactamente 800 líneas, G1 683 y G2 91. G2 sigue cerrado en estado 64; este
  corte no implementa el protocolo operativo ni cierra C4b-2.
- C4b-2/G2-O0/O1a queda integrado localmente en `dd029ad`–`52c8852`, después
  de contrato canónico, ledger basado en evidencia y triple `GO` final. O1a
  aporta únicamente codec de trama completa y autoprueba: runner 800, G1 686,
  G2 400, dos builds privados reproducibles, modo `--supervisar-m38` cerrado
  en 64 y puertas globales/carrera/calidad verdes. No abre FD, proceso, Bash
  operativo, Docker, PostgreSQL, SQL o red.
- La semántica O1b quedó aceptada en `6f4a118`–`4b765eb`. Su primer árbol
  material se detuvo sin commit al superar el ledger. La propuesta y corrección
  `docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o1b_ledger_correctivo_790_2026-08-05.md`
  quedaron en `e5e69e8`–`fb9e966`: un NO-GO aritmético fue corregido y dos
  revisores dieron GO final, `P0=P1=P2=0`. El candidato `56c0ac0` llegó a 788
  líneas y superó reproducibilidad, pero recibió NO-GO funcional porque tres
  mutantes seguían verdes y faltaban estados/precedencias.
  La corrección final `878d724`–`a6db818` obtuvo doble GO documental final,
  `P0=P1=P2=0`. El código final quedó integrado en `eb2bba0`–`98b753e`:
  G2 798, runner 800, G1 686, cuatro mutantes muertos, dos builds privados
  reproducibles y doble GO final `P0=P1=P2=0`. La puerta global terminó verde
  tanto en la candidata como en el árbol conjunto; la CI `31002229666`
  terminó después con sus cinco puertas verdes. La
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o1b_codigo_final_2026-08-05.md`
  conserva la evidencia. O1b no abre FD, procesos, Bash, Docker, PostgreSQL,
  SQL o red.
- La primera propuesta conjunta de separación/protocolo recibió `NO-GO`
  documental: no demostraba el presupuesto del runner y dejaba incompletos
  delimitación de tramas, `ACK_CASO` y causa/estado. Se sustituyó sin programar
  por dos documentos acotados: la
  `docs/portal_vec/enmienda_f0_h0b_c4b2_g2s_separacion_capturada_2026-08-05.md`
  y la
  `docs/portal_vec/especificacion_f0_h0b_c4b2_g2o_protocolo_operativo_2026-08-05.md`.
  G2-S obtuvo primero `GO` documental y después doble `GO` de implementación.
  G2-O recibió doble `NO-GO` documental y fue dividido. G2-O0, O1a y O1b ya
  están cerrados. El análisis O2a acreditó que G2 798/800 y runner 800/800 no
  admiten código nuevo. O2a-P0 y su enmienda local de ShellCheck obtuvieron
  doble GO documental. El candidato `df9a422` obtuvo después doble GO de
  código, `P0=P1=P2=0`, y quedó integrado localmente como `48a46a3`: runner
  783, D2d 164, H0 PostgreSQL 18.4, residuos cero y puerta global verde. Está
  publicado en `ef027cf` y la CI `31007941124` terminó con cinco puertas
  verdes. No crea G3 ni cambia conducta. El contrato O2a/S0/G3 quedó autorizado
  en `d148d76`. El primer candidato se rechazó por limpieza sensible y evidencia
  no durable; el corregido `50aec51` obtuvo doble GO final. La evidencia
  portable quedó confirmada en `6a8c1ee` y el código integrado localmente en
  `0caa140`: runner 794, G1 689, G2 798, G3 431, binario reproducible, AST,
  17 mutantes conductuales y 10 estructurales, H0 PostgreSQL 18.4 y puerta
  global verdes, con residuos cero. Quedó publicado en `ef1f08b` y la CI
  `31021785711` terminó con cinco puertas verdes.
  O2b, «ARMAR y cancelación sin Bash», queda cerrado técnicamente en
  `d86aea8`–`4b39265`, con doble GO y evidencia durable. O3a-P1 queda cerrado
  en `a1aeab7`–`f3a1e96`, con doble GO, y publicado en `ce0848e`; la CI
  `31298943127` terminó 5/5. Después solo se corrige y revisa el contrato O3a
  completo sobre `ce0848e`. `Start`, mapa FD y fases posteriores siguen
  cerradas hasta el doble GO, publicación y CI 5/5 de ese contrato.
  Las evidencias están en
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2s_implementacion_2026-08-05.md`,
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o1a_codigo_final_2026-08-05.md`,
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o1b_codigo_final_2026-08-05.md`
  y
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o2a_sobre_s0_g3_codigo_final_2026-08-05.md`,
  además de la revisión final O2b enlazada arriba.
- Después de G2-O2–O6 siguen, en secuencia, C4b-3, C4c y C4d. No se paralelizan porque
  comparten runner y adaptador. C4b-1 no cierra C4b, H0b, C2 ni F0.
- Métricas sin incremento: F0 `10/23`, O4-05 `3/5`, Contratación temporal
  `24/46`, Bolsa productiva `1/14`; producción permanece en `NO-GO`.
- No fusionar las ramas candidatas o de revisión de C4b-1: su contenido
  aceptado ya está en la integradora. Los SHAs `db240a5`, `075610f` y
  `1524feb` conservan su condición histórica `NO-GO`.
- No se desplegó ni se tocó el servidor remoto en este corte.

## Lectura obligatoria

Antes de editar:

1. instrucciones de mayor prioridad del entorno;
2. `AGENTS.md`;
3. `docs/portal_vec/relevo_sesion_2026-07-29_inicio_ct47.md`;
4. `docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md`;
5. `docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md`;
6. `docs/portal_vec/relevo_contratacion_temporal_2026-07-23.md`;
7. la especificación y matriz normativa enlazadas desde esos documentos.

El relevo fechado del 29 de julio es histórico y no vigente: no identifica la
rama integradora ni los worktrees actuales y no sustituye el corte operativo
del 31 de agosto. El directorio raíz conserva una rama histórica y no es el
lugar de desarrollo. El relevo anterior del 28 de julio también es histórico.

Un agente recibe un identificador de tarea. No toma otra por iniciativa propia.

## Trabajo paralelo

- Rama y worktree exclusivos fuera de `/tmp`.
- Write-set declarado y disjunto.
- Un agente productor no integra ni da el visto bueno a su propio trabajo.
- El revisor reproduce pruebas y emite `GO` o `NO-GO`.
- Solo dirección modifica los documentos transversales de estado durante la
  integración.
- No fusionar, rebasar ni limpiar ramas ajenas.
- Commits pequeños con el identificador funcional, pruebas y documentación.

## Granularidad obligatoria

Todo VEP se desarrolla mediante minitareas, también fuera de contratación
temporal:

- una minitarea tiene una sola responsabilidad observable y un único criterio
  de cierre;
- un agente recibe una sola minitarea y no amplía su alcance por iniciativa
  propia;
- código y pruebas forman un commit local autónomo; la evidencia de revisión y
  el estado transversal se confirman inmediatamente después por integración;
- como señal de alarma, una tarea que modifica más de tres ficheros de
  producción, añade más de unas doscientas líneas productivas o necesita dos
  motivos distintos en el mensaje se divide o justifica antes de programarse;
- cada commit debe compilar y superar sus pruebas focales; no se admiten cortes
  intermedios que dejen una API pública incompleta, una migración sin consumidor
  coherente o una ruta registrada sin autoridad;
- una capacidad grande se expresa como un grafo de minitareas con dependencias,
  no como un identificador que acumula contratos, adaptadores, composición,
  interfaz y E2E;
- las piezas independientes se asignan a subagentes con write-sets disjuntos;
  otro agente revisa el resultado antes de que dirección lo integre;
- si dos minitareas necesitan el mismo fichero, se ejecutan en secuencia o se
  separa primero una abstracción propietaria; nunca se resuelve permitiendo
  edición concurrente del mismo archivo.

Dividir no significa producir commits rotos o cambios cosméticos sin valor. La
unidad mínima es una capacidad compilable, protegida y verificable de extremo a
extremo dentro de su alcance.

## Arquitectura no negociable

- Hexagonal estricta.
- `domain` no importa aplicación, puertos, adaptadores, HTTP, SQL ni
  proveedores.
- `application` coordina dominio y puertos; no conoce HTTP, PostgreSQL, Docker
  ni SDK concretos.
- `ports` contiene contratos mínimos y neutrales; no implementaciones.
- `adapters` traduce proveedores y transportes sin redefinir reglas de negocio.
- Ningún módulo lee o escribe tablas de otro módulo.
- Los intercambios usan referencias opacas, comandos, eventos e
  outbox/inbox.
- El dominio y los casos de uso son neutrales al cliente. Web, escritorio,
  CLI y MCP consumen las mismas capacidades de aplicación mediante
  adaptadores distintos; ninguna regla funcional vive en HTTP, DOM o una
  sesión de navegador.
- Base de datos, almacenamiento, firma, antivirus, identidad, comunicaciones,
  documentos, calendarios y sistemas externos son adaptadores intercambiables.
- Fases, opciones, plantillas, formatos y reglas funcionales se gobiernan por
  catálogos versionados. Solo las invariantes técnicas quedan compiladas.
- No crear una segunda autoridad de identidad, roles, permisos, auditoría,
  i18n o temas.

## Seguridad y protección de datos

- Denegación predeterminada y privilegio mínimo.
- La identidad y el perfil proceden de una frontera confiable; nunca de JSON,
  cookies, parámetros, cabeceras libres o datos del navegador.
- Cookies, `localStorage` y `sessionStorage` no son autoridades ni mecanismos
  de sesión. La composición real no depende de ellos. Los clientes usan
  credenciales breves, ligadas al emisor y verificadas por la API; escritorio
  puede emplear certificado/mTLS y Kerberos corporativo mediante conectores.
- La API rechaza por defecto `Cookie`, credenciales persistidas por el
  navegador y cualquier cabecera de identidad que no esté atestada por la
  frontera confiable. Ninguna respuesta real emite `Set-Cookie`.
- Operaciones internas sensibles exigen garantía alta y superficie interna o
  administrativa coherente.
- Una decisión positiva queda ligada a acción, recurso, finalidad, ámbito,
  motivo, actor, perfil, correlación y vigencia exactos.
- Todo efecto consume la autorización dentro de la misma transacción que
  escribe estado, auditoría y outbox.
- Idempotencia semántica, control optimista de versión e historia de solo
  adición.
- Ningún secreto, certificado, clave, token, DSN, dato personal real o ruta
  privada entra en Git, fixtures, errores o logs.
- Datos minimizados y referencias opacas en listados, auditoría y eventos.
- Cifrado en tránsito y reposo; claves fuera del proceso y rotables.
- Límites de tamaño, tiempo, profundidad y cardinalidad antes de reservar
  memoria o llamar conectores.
- La indisponibilidad nunca se interpreta como autorización, validación,
  firma, entrega o éxito.
- No usar datos reales hasta cerrar EIPD, categorización ENS, análisis de
  riesgos, registro de actividades y autorizaciones formales.
- IA aplicada a selección, empleo, baremación, evaluación o asignación queda
  cerrada hasta completar la clasificación y obligaciones del Reglamento de
  IA. El bot público solo accede a corpus público gobernado.

La matriz vigente del módulo es
`docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md`.

## i18n, idioma y presentación

- Código de dominio en castellano coherente, salvo términos técnicos
  universalmente adoptados y convenciones de Go.
- No mezclar castellano e inglés en el mismo vocabulario.
- Todo texto visible, mensaje de validación, estado, ayuda, documento y
  notificación usa claves i18n.
- Fechas, números, moneda, zonas horarias y plurales se formatean por
  localización.
- El tema común es la autoridad visual; ningún módulo duplica CSS estructural.
- Accesibilidad desde diseño: teclado, foco, contraste, zoom, lector de
  pantalla, estados alternativos y documentos descargables accesibles.

## Calidad

- Código documentado cuando la intención o el contrato no sean obvios.
- Sin adaptadores ficticios en la composición real.
- No declarar E2E, producción o cumplimiento por tener una pantalla o una
  prueba aislada.
- Objetivo de 500 líneas y tope duro de 800 por fichero conforme a DEC-051.
- Sin funciones monolíticas, estados globales mutables ni errores que filtren
  detalles internos.
- Contextos y cancelación en todas las fronteras lentas.
- Copias defensivas al cruzar límites mutables.
- Dependencias externas solo si están mantenidas, son compatibles en licencia
  y reducen riesgo real.

Puertas mínimas:

```text
gofmt
go test del paquete afectado
go test -race del paquete afectado
go vet del paquete afectado
git diff --check
```

Antes de integrar se añaden `go test ./...`, `go vet ./...` y
`scripts/verificar_calidad.sh` cuando el alcance lo permita. PostgreSQL se
prueba en instancia efímera real con roles, ACL, concurrencia, reintento y
reversión protegida. HTTP/web exige contrato, seguridad, accesibilidad y
revisión visual.

## Entrega

```text
Tarea:
Estado: GO / NO-GO / bloqueada
Commit(s):
Archivos modificados:
Resultado:
Pruebas ejecutadas:
Pruebas omitidas y motivo:
Seguridad, privacidad, i18n y accesibilidad:
Limitaciones:
Riesgos:
Siguiente tarea desbloqueada:
Revisión independiente:
```

La documentación, pruebas y código se entregan juntos. El agente no cambia una
tarea a cerrada ni actualiza el porcentaje; lo hace dirección tras verificar e
integrar el commit.

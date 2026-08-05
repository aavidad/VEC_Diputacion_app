# Instrucciones del repositorio para agentes

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

## Relevo operativo inmediato — 5 de agosto de 2026

- Worktree obligatorio:
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-stable-docs`.
- Rama integradora: `integracion/ct-o4-04e-20260726`. El directorio raíz
  conserva la rama histórica y no se edita.
- Último corte remoto completamente verde: `ef1f08b`, ejecución
  `31021785711`, cinco de cinco puertas superadas. Incluye la evidencia, el
  código y el relevo integrado de O2a/S0/G3; las métricas funcionales no
  cambian.
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
  El siguiente trabajo es contratar O2b, que conserva «ARMAR y cancelación sin
  Bash»; los ACK vivos fueron eliminados por la corrección O1a. O2 y fases
  posteriores siguen cerradas. Las evidencias están en
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2s_implementacion_2026-08-05.md`,
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o1a_codigo_final_2026-08-05.md`,
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o1b_codigo_final_2026-08-05.md`
  y
  `docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o2a_sobre_s0_g3_codigo_final_2026-08-05.md`.
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

El relevo fechado identifica la rama integradora y los worktrees vigentes. El
directorio raíz conserva una rama histórica y no es el lugar de desarrollo.
El relevo anterior del 28 de julio es histórico y no sustituye al corte
vigente del 29 de julio.

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

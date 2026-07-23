# Relevo del frente de contratación temporal — 23/07/2026

Documento de entrada obligatorio para cualquier agente que continúe este
frente. Debe actualizarse en cada commit que cambie alcance, arquitectura,
estado o siguiente paso.

## Objetivo

Implementar el procedimiento remitido por RRHH para gestionar expedientes de
contratación temporal desde la petición del centro hasta GINPIX, conservando
íntegramente las capacidades existentes de Bolsa.

## Rama y base

- Rama: `feature/contratacion-temporal`.
- Base técnica: `real/c3-convergencia`, commit `eeac3a2`.
- Motivo: contiene C3 público auditado y la cápsula interna C4.
- No desarrollar este frente sobre `vec-orquesta-20260619`: en el momento de
  abrirlo tenía cuatro commits documentales propios, pero estaba 73 commits por
  detrás de C3.

Las ramas `real/t17a-postgres-importador` y
`real/t20-borradores-postgresql` siguen siendo trabajos separados e
incompletos. No deben mezclarse sin auditoría y reconciliación explícitas.

## Fuente de requisitos

Original local recibido:

```text
/home/alberto/Trabajo/VEC_Diputacion_app/
Pantalla de procedimiento de gestión de contratación y gestión de bolsas.docx
```

El fichero temporal de LibreOffice cuyo nombre empieza por `.~lock` no es
documentación y no debe versionarse.

La lectura normalizada se conserva en:

- `docs/portal_vec/expediente_contratacion_temporal_rrhh.md`.

## Dirección y cumplimiento

Todo agente debe leer `AGENTS.md`. El alcance se gobierna mediante:

- `docs/portal_vec/objetivos_y_hoja_ruta_rrhh_2026-07-23.md`;
- `docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md`;
- `docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md`;
- `docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md`.

La matriz normativa es una puerta técnica y organizativa. El módulo no puede
usar datos reales ni declararse conforme o productivo sin las validaciones
formales de DPD, Seguridad, Sistemas, Jurídico, Archivo, RRHH y demás
responsables competentes.

## Decisión principal

Se crea `internal/modules/contrataciontemporal`. No se amplía Bolsa hasta
convertirla en un monolito.

```text
contrataciontemporal
  coordina → bolsa
  coordina → personal
  coordina → documentos/firma
  coordina → comunicaciones
  coordina → intervención
  coordina → GINPIX
```

Cada módulo mantiene su autoridad y solo intercambia referencias opacas,
comandos y eventos. No se permiten lecturas o escrituras directas en tablas
ajenas.

## Estado comprobado

| Entregable | Estado |
| --- | --- |
| Especificación inicial | Implementada |
| Manifiesto del módulo | Implementado y probado |
| Dominio de expediente | Primer corte implementado y probado |
| Puertos de alta | Implementados y probados por el caso de uso |
| Caso de uso: registrar solicitud | Implementado y probado |
| Resto de casos de uso | Pendiente |
| Preparación idempotente PostgreSQL | Cerrada y revisada: rotación v1→v2, replay, concurrencia, ACL y límites reales |
| Autorización VEC durable de la preparación | Cerrada y revisada: V3, par HMAC activo exacto, revalidación y clientes neutrales |
| Confianza y capacidad breve VEC-AD-3 | Integradas y revisadas: Ed25519/COSE estricto, audiencia, capacidad HMAC ≤5 s, rotación y revocación |
| Contrato autenticado con Bolsa | Cerrado y revisado: referencias opacas, seudónimos, eventos, pruebas durables e inbox idempotente |
| Confirmación atómica PostgreSQL | Cerrada en `77743a7`: GO 2/2 y puertas PostgreSQL/globales conjuntas verdes |
| Diseño de adaptador y reconciliación | GO condicionado; debe acoplarse a la firma real de O2-05 antes de implementar |
| API interna | Adaptador O2-08B revisado con GO e integrado; falta registrarlo mediante O2-07 |
| Web conectada | O2-09B integrada en `764fd52` tras GO independiente; falta registrar la ruta real y el E2E |
| E2E administrativo | Pendiente |

## Cortes locales y revisiones pendientes

- `825e251`, `ec87e27` y `ff6011f` crean el corte VEC-AD-3 para decisiones V3
  ligadas a contexto de actor V2. Tras dos NO-GO, `99d3396`–`b25e134`
  incorporan parser estricto, canonicidad única, ligadura de compromisos,
  validación semántica y cruce nominal completo con el contexto firmado.
  El conjunto obtuvo GO independiente y quedó integrado en `fe00ed9`.
- `4022152`–`9114f76` añaden verificación de confianza VEC-AD-3 con
  Ed25519/COSE estricto y una capacidad HMAC canónica de 37 campos y vida
  máxima de cinco segundos, ligada a decisión, efecto, audiencia, secuencia de
  configuración y versión de raíz. La revisión independiente terminó sin
  hallazgos y el corte quedó integrado en `8461aee`. Falta el consumidor SQL
  atómico; no cierra O2-05 ni habilita producción.
- `cfc935a` recibió dos NO-GO coincidentes por el canon temporal SQL/Go, un
  privilegio `REFERENCES` entre autoridades y una matriz de fallos incompleta.
  La corrección `cbe7299` conserva RFC3339Nano V3, elimina el privilegio,
  automatiza los ocho fallos y cubre cancelación pre-`COMMIT`, respuesta
  perdida, reconciliación y reinicio real. El productor superó PG18 ×4, foco
  ×100, carrera y puertas globales. Dirección y un segundo revisor repitieron
  PostgreSQL 18, foco, carrera, `go vet`, tamaños y secretos, y emitieron GO
  2/2 sin hallazgos. La serie funcional se integró en `77743a7`; PostgreSQL 18,
  pruebas globales, carrera focal, `go vet`, scripts, tamaños y secretos
  quedaron verdes sobre el árbol conjunto. O2-05 está cerrada y contabilizada.
- `2c800fa`–`4cc4422` describen O2-06A, incluido resultado indeterminado,
  reconciliación, reintentos y ACL. Sus revisiones son GO condicionado. El
  diseño suponía catorce entradas y doce columnas; O2-05 ha congelado doce
  entradas y ocho columnas. O2-06 debe ejecutar ahora la puerta de
  acoplamiento y adaptar el mapeo documental antes de escribir código.
- O2-08A y O2-09A se crearon como candidatos aislados en sus respectivas
  capas. La API no registra ruta y la vista no usa un adaptador falso.
  Dirección detectó entonces que discrepaban sobre el origen y transporte de
  la clave idempotente; la decisión neutral quedó documentada en
  `decision_idempotencia_canales_alta_o2_08b_2026-07-23.md`.
- O2-08B `3e2885c` resuelve la incompatibilidad de idempotencia con O2-09A:
  todos los canales envían el mismo sobre cerrado con una UUIDv4 de intención
  no autoritativa; la autoridad de servidor aporta solo autenticación, sesión,
  perfil y organización. La aplicación liga UUID, contexto efectivo y
  contenido mediante HMAC. No usa cookies ni almacenamiento web. La revisión
  independiente terminó con GO y cero hallazgos; focal ×50, carrera ×5,
  pruebas globales, `go vet`, OpenAPI, secretos y el ensayo conjunto con
  O2-09A quedaron verdes. Se integró como `42dc3ac`–`94e09e8`.
  La integración virtual con O2-09A `1323b4b` produjo el árbol
  `750bd6d479ddcf9e5564cc25d8603b0b5aa62e06` sin conflictos; en ese árbol
  pasaron veinte repeticiones del contrato HTTP, carrera, las veinte pruebas
  de la vista y todos los paquetes de contratación temporal. El worktree
  desechable se eliminó después de la prueba. La ruta sigue sin registrar:
  O2-08 permanece funcionalmente abierta hasta la composición O2-07.
- O2-09A `1323b4b` superó las puertas visuales, pero recibió NO-GO por admitir
  periodos de más de cien años e importes superiores a `922337203685477`
  céntimos. O2-09B `228df6f` corrigió ambos bordes con aritmética decimal
  exacta, doble barrera e i18n. Un revisor distinto reprodujo los extremos,
  ejecutó 260/260 pruebas del portal y emitió GO. La interfaz quedó integrada
  en `764fd52`. O2-09 sigue abierta hasta registrar la ruta real y superar
  O2-10; el corte no usa cookies, almacenamiento ni adaptadores DEMO.
- `2cd3da1` inicia O3-02 con rectificación motivada, control optimista,
  cronología de solo adición y bloqueo de retroacciones implícitas. Un primer
  corte de aplicación posterior recibió NO-GO: aceptaba autoridad RC/coste
  aportable, su orden no limitaba el único efecto y un test copió una
  credencial real. No se integró ni se publicó. La rama está en cuarentena y
  O3-02 se reconstruye desde `209ae72` sin reutilizar sus commits. La barrera y
  el procedimiento están en
  `docs/seguridad/barrera_secretos_git_2026-07-23.md`.
- El corte reconstruido O3-02 recibió tres revisiones `NO-GO`. La tercera
  detectó preconsumo durable de RC+coste antes del commit final, validación de
  vigencia posterior al commit y un orquestador concreto en `ports`. El
  rediseño candidato deja el artefacto sin consumir, mueve el orquestador a
  `application` y transporta orden pendiente de fuentes, contexto y concesión
  V3 a una única frontera O3-04. Esa transacción debe validar con reloj de base
  de datos y consumir RC+coste+V3 junto con CAS, agregado, historia, auditoría,
  recibo y outbox en un solo `COMMIT`, con rollback total. Aplicación sólo
  valida defensivamente el recibo devuelto contra la orden, sin consultar un
  reloj nuevo. Sigue pendiente de revisión independiente y del adaptador
  PostgreSQL; no habilita producción.
  Dos bloqueos posteriores obligaron a precisar el corte: aplicación valida
  primero cualquier recibo retornado, de modo que uno válido prueba el commit
  aunque coexista cancelación o error de transporte, mientras uno adulterado
  nunca se expone. Además, toda coordinación de los desafíos frescos se movió
  a `application`; `ports` limita su papel a materiales y confirmaciones
  opacos, copias y validación criptográfica local neutral. La corrección final
  `df537b4` sella además actor, perfil, UUID, operación, organización,
  expediente, versión, artefacto, datos funcionales y motivo frente al replay
  temprano. La revisión independiente emitió GO. El candidato quedó integrado
  en `e9d461c` y volvió a superar focales ×20, carrera ×2, pruebas globales,
  `go vet`, tamaños y secretos sobre el árbol conjunto. O3-02 está cerrada;
  O3-04 es el siguiente efecto durable. Permanece como limpieza baja retirar
  dos ayudantes privados sin llamadas, sin efecto funcional ni exposición.
- `1faa8e7` implementa O3-03 con puertos neutrales para validación
  presupuestaria y cálculo de coste, ligadura de petición, copias defensivas,
  cancelación y fallo cerrado. La indisponibilidad nunca se convierte en «RC
  no requerida» ni en coste válido. El rework `fca5d41`–`af124fb` corrigió
  autenticidad de respuesta, replay, vigencia, errores y catálogo.
  `4e6d14a`–`6d1be36` cerraron la separación TCB con credenciales Ed25519
  institucionales, prueba de posesión, tres autoridades/backend/claves
  distintas, rotación/revocación y límites interoperables. La revisión final
  terminó sin hallazgos y el conjunto quedó integrado en `4c33336`.
- `a34d462` inicia O4-02 con un puerto único para consultar fuentes de
  cobertura configuradas por catálogo. Liga organización, expediente y
  versión, catálogo y huella, vía, procedencia, categoría, periodo y
  comprobación; impone timeout, reloj inyectado, cancelación, respuesta
  minimizada y errores públicos sin detalles del proveedor. No contiene listas
  compiladas de Bolsa/SAE ni autoriza lecturas de sus tablas. Faltan los
  adaptadores concretos y la revisión independiente.
- `fd766a2` completa la ligadura de la respuesta O4-02 con organización,
  procedencia, categoría y periodo. La frontera rechaza texto libre del
  proveedor para impedir que una consulta minimizada transporte identidad o
  detalles personales; el diagnóstico completo permanece tras el recibo opaco
  en el sistema fuente.
  Una revisión independiente posterior dio NO-GO: estas ligaduras siguen
  siendo nominales y copiables, no existe atestación de la respuesta ni
  consumo anti-replay, tampoco se acredita que catálogo, vía y comprobación
  estén publicados y vigentes. Deben corregirse además el timeout a cinco
  segundos, el periodo máximo, la versión interoperable, la prioridad de
  cancelación y la ocultación total de errores privados. O4-02 no está cerrado.
- El candidato O4-02 posterior `a0c7ecf` incorpora consumo/replay durable,
  identidad semántica de petición, suelo temporal monótono y mueve el
  autenticador concreto a infraestructura. Una nueva revisión independiente
  reprodujo tres bloqueos: la vigencia se comprueba antes de una presentación
  que puede consumir tiempo y no al terminar; el replay histórico vuelve a
  verificar con la clave actual y falla tras rotación; y `ports` conserva una
  orquestación residual de desafío, presentación y verificación.
- La quinta corrección O4-02 `0a3604c`–`5bb4aaf` arregló la ruta nueva, pero
  recibió NO-GO independiente: `ports.ValidarRCConFuente` y
  `ports.CalcularCosteConFuente` siguen coordinando varios puertos y verifican
  con un instante capturado antes de presentar. Una credencial válida hasta
  `t0+6 s` puede aceptarse indebidamente si la presentación consume dos
  segundos y el horizonte es cinco. La sexta corrección debe retirar esa ruta
  residual y añadir la regresión determinista. Nada de esta serie está
  integrado.
- O8-01 recibió tres NO-GO por integridad, límites y coste multiplicativo. La
  corrección final `7b56962` limita actuaciones, periodos y colecciones
  anidadas antes de reservar, copiar u ordenar; valida la definición una sola
  vez y reproduce con índices en `O(D+A)`. Un revisor distinto reprodujo
  entradas adversariales de 65.536 elementos, focales ×50, carrera ×5,
  globales, `go vet`, tamaños y secretos, y emitió GO. Dirección repitió las
  puertas sobre el árbol fusionado. O8-01 quedó integrada en `ec8e758` y está
  cerrada; no acredita todavía persistencia, API ni producción.
- `2b67c7a`–`20935bd` integran O6-01 tras dos revisiones independientes.
  Contratación temporal y Bolsa solo intercambian contratos versionados,
  referencias opacas, seudónimos HMAC, evidencias autenticadas y eventos
  durables. Los decodificadores rechazan esquemas desconocidos, cargas no
  canónicas y estados parciales; el consumidor vuelve a comprobar vigencia
  dentro de la transacción. No existe lectura directa de tablas de Bolsa.

Estos contratos no dependen de HTTP, cookies ni almacenamiento de navegador.
Los consumirán por igual web, escritorio, CLI y MCP a través de los casos de
uso comunes.

## Permisos iniciales

- `contratacion_temporal.cuadro.consultar`
- `contratacion_temporal.expediente.consultar`
- `contratacion_temporal.solicitud.crear`
- `contratacion_temporal.analisis.validar`
- `contratacion_temporal.cobertura.decidir`
- `contratacion_temporal.unidad.asignar`
- `contratacion_temporal.flujo.configurar`
- `contratacion_temporal.auditoria.consultar`

Son capacidades técnicas publicadas por el módulo. No se asignan a perfiles
desde el manifiesto ni conceden acceso sin una decisión positiva del PDP.

## Siguiente corte exacto

1. Alinear e implementar O2-06 contra la firma real de O2-05.
2. Componer O2-07 y registrar el adaptador O2-08B.
3. Conectar la interfaz O2-09B sin cookies ni autoridad de cliente.
4. Cerrar O2-10 con navegador → API → autorización → PostgreSQL → recibo,
   reinicio, concurrencia y aceptación RRHH.

## Dominio implementado

Paquete:

```text
internal/modules/contrataciontemporal/domain
```

Incluye:

- referencias de flujo con versión y huella;
- claves funcionales abiertas a catálogos gobernados;
- estados operativos técnicos cerrados;
- periodos civiles UTC e importes exactos en céntimos;
- solicitud del centro y declaración de RC;
- análisis RRHH, jornada en diezmilésimas, coste y validación de RC;
- comprobaciones y decisión de vía de cobertura;
- asignación a unidad y responsable;
- agregado con versión optimista y cronología encadenada;
- copias defensivas y orden mínimo de los cuatro primeros hitos.

## Alta de solicitud implementada

Paquetes:

```text
internal/modules/contrataciontemporal/ports
internal/modules/contrataciontemporal/application
```

El caso `ServicioRegistroSolicitud.Registrar` realiza, por este orden:

1. valida la orden y clona defensivamente la solicitud;
2. resuelve identidad y perfil desde la sesión interna;
3. exige garantía alta y vigencia;
4. resuelve la definición gobernada y versionada del flujo;
5. deriva una huella HMAC sin usar el cuerpo en claro como clave;
6. prepara referencias estables bajo idempotencia semántica;
7. devuelve el mismo recibo si la operación ya estaba confirmada;
8. obtiene una autorización ligada al expediente exacto;
9. construye el agregado y una orden opaca de confirmación;
10. exige un recibo durable coherente con la versión creada.

La interfaz `TransaccionAltas` obliga al futuro adaptador a cotejar y consumir la
autorización y a confirmar reserva, expediente, auditoría y outbox en un único
`COMMIT`. La preparación idempotente ya tiene persistencia PostgreSQL revisada;
la confirmación atómica y su reconciliación siguen pendientes.

Pruebas superadas en el corte:

```text
go test ./internal/modules/contrataciontemporal/...
go test -race ./internal/modules/contrataciontemporal/...
go vet ./internal/modules/contrataciontemporal/...
git diff --check
```

## Invariantes

- Las fases y opciones funcionales proceden de definiciones y catálogos
  gobernados; no se añaden como listas cerradas en el código.
- Los estados técnicos de seguridad y consistencia sí permanecen cerrados.
- Importes en unidades monetarias menores, nunca `float64`.
- Instantes UTC canónicos; los calendarios civiles se resuelven por puerto.
- Historial de solo adición.
- Una transición exige versión esperada y falla ante concurrencia.
- Ninguna identidad, rol, ámbito o decisión procede del navegador.
- No se copian DNI, teléfonos o correos en cuadros de mando.
- La integración GINPIX será un puerto con adaptadores API y fichero.
- Una pantalla DEMO no se contabiliza como integración.

## Puertas antes de commit

Como mínimo:

```text
gofmt
go test ./internal/modules/contrataciontemporal/...
go test -race ./internal/modules/contrataciontemporal/...
go vet ./internal/modules/contrataciontemporal/...
git diff --check
```

Al tocar composición, seguridad o artefactos deben ejecutarse además las
puertas focales de C3/C4 y `scripts/verificar_calidad.sh`.

## Regla de documentación

Cada corte debe actualizar:

1. este relevo;
2. la especificación si cambia una decisión;
3. `README.md` si cambia el estado visible;
4. la matriz operativa cuando una capacidad cambie realmente de nivel;
5. las pruebas y evidencia exacta del commit.

Nunca sustituir «pendiente» por «terminado» porque exista una pantalla, un
puerto o una prueba aislada.

## Estado O4-02 tras cuarta revisión

O4-02 continúa en `NO-GO` pendiente de nueva revisión independiente. La rama
aislada incorpora dos correcciones:

- suelo de reloj monotónico por consulta; cualquier retroceso entre
  autenticación, catálogo, fuente, verificación, preconsumo y salida falla
  cerrado, y el caso `t+5 → t+2` prueba consumo cero;
- el autenticador concreto de autoridades se ubica en
  `adapters/seguridad`; `ports/fuentes_analisis_autenticador.go` queda limitado
  a identidad inmutable, copias defensivas, segregación local y contrato.

No existe todavía consumidor durable productivo ni composición con fuentes
reales. Tampoco hay autoridad procedente de cookies, navegador o DTO cliente:
la confianza, organización y audiencia se fijan exclusivamente en composición.

## Estado O4-02 tras quinta corrección

O4-02 sigue en `NO-GO` a la espera de otra revisión independiente. La rama
aislada corrige:

- TOCTOU de presentación: cada autoridad se verifica con una lectura
  autoritativa posterior y el fin temporal es exclusivo;
- replay K1→K2: el recibo conserva credencial, firma institucional, desafío,
  prueba de posesión, identidad y clave pública originales, sin secretos;
- separación hexagonal: `application` coordina el desafío y la presentación;
  el adaptador de seguridad solo verifica la evidencia pública ya obtenida;
- pruebas de timeout deterministas mediante contexto controlado, sin esperas
  de 5 ms ni dependencia del planificador.

La evidencia histórica solo verifica el efecto anterior. Fuente, verificador y
publicador actuales vuelven a autenticarse antes de cualquier consumo; una
confianza actual inválida no queda autorizada por un recibo K1. Continúan
pendientes el consumidor durable productivo, la composición con conectores
reales, las pruebas E2E y las conformidades organizativas.

## Estado O4-02 tras sexta corrección candidata

La sexta corrección parte exactamente de
`ba80e8f766e1054f35db15b47c2f4f13ea6b2221`, no de la rama divergente de la
quinta corrección. Así preserva la arquitectura O3-02 ya integrada: la
orquestación de validación RC y cálculo de coste permanece en `application` y
la versión histórica de `ports` solo compila en `_test.go`.

O4-02 se ha trasladado sobre esa base sin recuperar la ruta productiva antigua.
Además, el coordinador común de autoridades O3:

- lee el reloj antes de crear el desafío;
- obtiene la presentación;
- vuelve a leer el reloj autoritativo;
- rechaza retrocesos y verifica credencial, raíz, revocación y horizonte con
  el instante posterior.

La API local ya no permite validar una presentación sin aportar ese instante
posterior. Una regresión determinista cubre ambas rutas: `t0`, presentación en
`t0+2 s`, credencial hasta `t0+6 s` y horizonte de cinco segundos. RC y coste
se rechazan antes de invocar la operación funcional o el consumidor.

El candidato continúa en `NO-GO` hasta que un agente distinto revise la sexta
corrección y repita las puertas O3+O4 sobre el SHA exacto.

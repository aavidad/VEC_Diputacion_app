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
| Confirmación atómica PostgreSQL | Pendiente |
| API interna | Pendiente |
| Web conectada | Pendiente |
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
- `2cd3da1` inicia O3-02 con rectificación motivada, control optimista,
  cronología de solo adición y bloqueo de retroacciones implícitas. Un primer
  corte de aplicación posterior recibió NO-GO: aceptaba autoridad RC/coste
  aportable, su orden no limitaba el único efecto y un test copió una
  credencial real. No se integró ni se publicó. La rama está en cuarentena y
  O3-02 se reconstruye desde `209ae72` sin reutilizar sus commits. La barrera y
  el procedimiento están en
  `docs/seguridad/barrera_secretos_git_2026-07-23.md`.
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

1. Implementar la frontera consumidora SQL atómica de VEC-AD-3.
2. Asegurar en una sola transacción la reserva, autorización consumida,
   expediente, primera actuación, auditoría y outbox.
3. Probar repetición, conflicto semántico, concurrencia y resultado
   indeterminado de `COMMIT`.
4. Exponer después la API interna; no crear antes una segunda autoridad de
   identidad en el manejador.

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

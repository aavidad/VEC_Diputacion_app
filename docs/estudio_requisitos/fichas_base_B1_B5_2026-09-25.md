# Fichas de la base B1–B5 — 25 de septiembre de 2026

Denominador de los cinco frentes de base de la hoja de ruta aprobada el 24/09/2026
(`/home/alberto/.codex-vec-modulos/CONSENSO_HOJA_RUTA_MODULOS_20260924.md`, «Plan
consensuado», punto 1): B1 identidad/persona/composición, B2 empleado y relaciones,
B3 organización, B4 calendarios y B5 documentos (con firma, registro, notificación y
archivo como capacidades comunes, sin nueva autoridad universal: ronda 2, punto 9).

Estas listas son cortas a propósito: una fila es una capacidad observable con un solo
criterio de cierre. Se contrastan con `origin/main` = `2ad54ce4` (hasta el PR #52).

## Cómo se lee cada fila

- **Formal**: cerrada según la definición de «terminado» del consenso del 24/09 (recorrido
  humano en PostgreSQL real con datos sintéticos, negativos y mismo resultado tras reiniciar;
  puerta verde; dos E10 donde corresponda; un PR; desplegada en cidonia y barrida por
  Dirección). A 25/09 **ninguna fila de B1–B5 cumple todas esas condiciones**.
- **Técnico**: existe en `main` con pruebas (Go, Node o ensayo PostgreSQL 18.4). No implica
  instalación ni uso.
- **Uso real**: servido hoy en la principal (inventario de solo lectura del 25/09: binario
  `main@35d1cd17`, hasta el PR #48) **y con barrido o prueba documentados**. Lo servido sin
  barrido documentado se anota en la fila, pero no se cuenta.

Criterio común a toda fila: además de su criterio propio, sin segunda autoridad (AGENTS.md,
«Arquitectura no negociable») y con la denegación probada.

## B1 — Identidad, persona y composición (6)

| Id | Capacidad | Criterio de cierre | Técnico | Uso real |
| --- | --- | --- | --- | --- |
| B1.1 | Autenticación interna con certificado personal en la frontera mTLS, sin cookies ni almacenamiento web | Cadena cliente con CA aceptada en todos los módulos; certificado ajeno o ausente → 401 | Sí (PR #46; decisión provisional de acceso por certificado hasta Kerberos) | No contado: solo camino positivo en el barrido de Bolsa (23/09); negativos sin barrido |
| B1.2 | Persona, cuenta y contexto de actor durable con alcance `{empleado}` | Proyección persona→empleado consumida por el contexto; ambiguo o ausente → denegado | Sí (Personal 000016 y ContextoActor 000007, PR #36) | No acreditado: el inventario de hoy no incluye ContextoActor |
| B1.3 | Autorización positiva V3 consumida en la misma transacción que estado, auditoría y outbox | Todo efecto nuevo usa su consumidor AD3 | Sí (AD3, núcleo en uso por CT y Bolsa) | Sí: ejercido en el barrido de Bolsa en cidonia (ESTADO, 23/09) |
| B1.4 | Revalidación corporativa del vínculo RRHH en cada petición | ContextoActor 000009 instalada y vínculo revocado → denegado | Sí (PR #41) | No: «nada aplicado en la principal» (PR #41) |
| B1.5 | Matriz de papeles y concesiones por rol (quién ve cada bandeja) | Pestañas ofrecidas solo a quien tiene concesión | No: dudas 29, 31, 47 y 48 (`web/static/portal-empleado/modulos/cronos/INTEGRACION.md`, «Pendiente») | No |
| B1.6 | Identidad definitiva: Kerberos interno; Cl@ve/DNIe externo (acceso del candidato, B11 de Bolsa) | Proveedor real compuesto en la frontera | No: portada pública con las opciones deshabilitadas (ESTADO, 19/09) | No |

B1: formal 0/6, técnico 4/6, uso real 1/6.

## B2 — Empleado y relaciones (Personal) (5)

| Id | Capacidad | Criterio de cierre | Técnico | Uso real |
| --- | --- | --- | --- | --- |
| B2.1 | Proyección gobernada persona→empleado con historia, versiones y revocación | Consulta desde el contexto sin construir el contexto con ella (consenso 24/09, ronda 2, punto 4) | Sí (Personal 000016, PR #36) | No: instalada (Personal 7–21 salvo huecos), sin consumidor activo |
| B2.2 | Registro RRHH de relaciones, ocupaciones, servicios y situaciones, bitemporal y de solo adición | Alta y hecho con recibo; lista por organismo | Sí (Personal 000017–000021, AD3-54/55/56, PR #41 y #42) | No: gobierno B2 tras `VEC_PERSONAL_B2_GOBIERNO_ENABLED`, sin activación acreditada |
| B2.3 | «Mis datos»: ficha propia del empleado | `GET /api/interna/personal/mi-ficha` con V3 sobre el propio `empleado_ref` | Sí (Personal 000022 y AD3-74, PR #48) | No: inactivo; Personal 000022 y AD3-74 sin instalar |
| B2.4 | Relaciones autorizadas para consumidores (Dietas) | `GET /api/vec/personal/relaciones-dietas` | Sí (Personal 000012–000013, que declara la ruta, y AD3-49 `consumidor_personal_dietas`) | No: Dietas inactiva |
| B2.5 | Carga de empleados desde la fuente corporativa y conciliación | Importación con recibo, idempotente, sin datos reales hasta EIPD | No | No |

B2: formal 0/5, técnico 4/5, uso real 0/5.

## B3 — Organización (5)

| Id | Capacidad | Criterio de cierre | Técnico | Uso real |
| --- | --- | --- | --- | --- |
| B3.1 | Estructura organizativa histórica e importación | Consulta e importación con historia | Sí (`internal/modules/personal/application/consulta_organizacion_historica.go`, `importacion_organizacion_historica.go`; AD3-51/52) | No acreditado |
| B3.2 | RPT pública y categorías profesionales servidas desde la raíz real | Fuentes fijadas por huella; 503 si faltan | Sí (PR #44) | No acreditado: Personal no figura entre los activos |
| B3.3 | Plazas y puestos con versiones como maestro | Alta y consulta versionada que B2 referencia | No: B2 guarda `PlazaRef`/`VersionPlazaRef`, pero no hay maestro (`docs/estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md` es estudio) | No |
| B3.4 | Vacantes como proyección de plazas y ocupaciones | Proyección consultable | No | No |
| B3.5 | Competencias por unidad (administrativo, responsable, jefatura) publicadas | Catálogo competente autorizado consumido por Dietas y Cronos | No: fachadas AD3-61 y Personal 000014/000015, sin fuente de competencia (dudas 40, 46 y 47) | No |

B3: formal 0/5, técnico 2/5, uso real 0/5.

## B4 — Calendarios (4)

| Id | Capacidad | Criterio de cierre | Técnico | Uso real |
| --- | --- | --- | --- | --- |
| B4.1 | Calendario laboral y hábil histórico por organización y centro | `/api/vec/calendarios/centros` y `/centro` | Sí (Calendarios 000001–000003) | No contado: Calendarios activo, sin barrido documentado |
| B4.2 | Cómputo de plazos hábiles | `/api/vec/calendarios/plazo`; indeterminado → 422 | Sí (`internal/modules/calendarios/domain/plazo.go`, PR #40) | No contado: ruta servida, sin barrido documentado |
| B4.3 | Consumo por los módulos mediante puerto | Cronos, CT y Bolsa leen B4 y no tablas propias | No: Cronos mantiene `vec_cronos_v1.calendario_dia` propio (Cronos 000008) | No |
| B4.4 | Mantenimiento por RRHH con versión (años siguientes, jornadas especiales) | Publicación versionada sin migración | No: los datos son la migración 000002 (2026) | No |

B4: formal 0/4, técnico 2/4, uso real 0/4 (servido sin barrido: 2/4).

## B5 — Documentos, firma, registro, notificación y archivo (7)

| Id | Capacidad | Criterio de cierre | Técnico | Uso real |
| --- | --- | --- | --- | --- |
| B5.1 | Documento común con custodia VEC o externa (referencia + SHA-256) y almacén de ficheros | Alta y consulta con V3 en la misma transacción | Sí (Documentos 000001–000004, AD3-60/62, PR #50) | No: ninguna migración de Documentos instalada |
| B5.2 | Generación desde plantilla como servicio común | Plantilla versionada consumida por más de un módulo | No como servicio común: los seis borradores PDF/Word son propios de CT | No (los de CT se cuentan en CT) |
| B5.3 | Verificación de firma con el validador de AutofirmaV2 como servicio aparte | Dictamen v1 interpretado; indisponible ≠ válida | Sí (PR #52), **sin componer** | No |
| B5.4 | Firma en navegador por protocolo AutoFirma | Documento firmado y verificado por B5.3 | No | No |
| B5.5 | Registro de entrada y salida | Asiento con justificante | No | No |
| B5.6 | Notificación y constancia de entrega | Acuse del canal | No: una referencia o borrador no acredita entrega (consenso 24/09, ronda 2, punto 9) | No |
| B5.7 | Conservación y archivo | Plazos aplicados por catálogo aprobado | No: catálogo provisional, plazos pendientes (duda 60) | No |

B5: formal 0/7, técnico 2/7, uso real 0/7.

## Resumen

| Frente | Filas | Formal | Técnico | Uso real |
| --- | ---: | ---: | ---: | ---: |
| B1 | 6 | 0 | 4 | 1 |
| B2 | 5 | 0 | 4 | 0 |
| B3 | 5 | 0 | 2 | 0 |
| B4 | 4 | 0 | 2 | 0 |
| B5 | 7 | 0 | 2 | 0 |

Esta ficha la actualiza Dirección al integrar; una fila solo pasa a formal con la evidencia
completa de la definición del 24/09.

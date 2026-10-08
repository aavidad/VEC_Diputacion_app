# Certificados y analítica: inventario y continuación

**Aparcado hasta cerrar Bolsa y CT (orden de Alberto, 07/10/2026).** Los objetivos vigentes están en [OBJETIVOS.md](OBJETIVOS.md).

Estado comprobado sobre `origin/main` `0a62a3ea6` el 1 de octubre de 2026. El catálogo funcional es `docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md`; este plan no da por emitido ningún certificado ni por registrada una descarga.

## Qué existe

| Pieza en main | Rutas y estado |
| --- | --- |
| Certificados | `internal/modules/personal/manifest.go` declara hoy un enlace al menú; **J es dueño del módulo y de la entrada de Certificados**, Personal solo enlaza. `web/static/area-personal/` presenta certificados, pero su catálogo propio llega vacío. No hay emisión de servicios conectada. |
| Documentos y firma | `internal/vec/adapters/documentos/pdf/renderizador.go` genera PDF; `internal/modules/contrataciontemporal/application/firma_documento.go` y `ports/custodia_firmado.go` aportan contratos de firma y custodia para CT. Son capacidades reutilizables mediante puertos; no acreditan por sí solas un certificado firmado. |
| Estadísticas CT | `internal/modules/contrataciontemporal/adapters/postgres/estadisticas_rrhh_postgresql.go`, migración CT 000107, `adapters/httpinterno/estadisticas_rrhh.go` y `web/static/portal-empleado/modulos/contratacion-temporal/{cliente-http-estadisticas,contrato-estadisticas,vista-estadisticas}.js` dan consulta agregada por semana, mes o año y ámbito autorizado, con gráfico, tabla y CSV local. Funciona como cuadro CT; no es un cuadro transversal. |

## Huecos frente al catálogo y minitareas

| Orden | Capacidad y tarea pequeña | Responsable, archivos previstos y dependencia | Cierre verificable |
| --- | --- | --- | --- |
| 1 | **ANA-001:** mostrar definición y fórmula observable de SQL107, fuente, periodo, calidad y versión de los cinco indicadores CT existentes. El responsable institucional y la aprobación siguen pendientes de RRHH. | J: `web/static/portal-empleado/modulos/analitica/`, `vista-estadisticas.js`, textos ES/EN. Terminar la cadena `?v=` y los manifiestos en el turno J de compartidos. | Ayuda visible junto al cuadro real, sin cambiar cifras; Chrome escritorio/móvil, revisión independiente, calidad y CI verdes. |
| 2 | **CER-001, preparación:** plantilla de servicios versionada y borrador PDF con ejemplo sintético, sin emisión oficial. La CLI es el preparador que servirá como **vista previa de RRHH en el producto**; el corte actual solo admite ensayo sintético local y aún no está montado en el portal. | J: `internal/modules/certificados/`, `cmd/vec-certificados-borrador/`, `data/certificados/plantillas/`, textos ES/EN. Reutiliza PDF común; entrada autorizada de servicios depende de Personal B. | La CLI prepara y comprueba un borrador identificado como tal; exige una muestra marcada como sintética, sin poder verificar que un nombre sea ficticio. No afirma firma, código de verificación ni entrega. |
| 2b | **CER-002 (hecho):** el borrador acepta la respuesta del contrato V1 de Personal (`LectorServiciosParaCertificadosV1`): periodos semiabiertos o abiertos, cobertura, certeza y acto, sin días porque V1 no los trae; marca qué servicios pueden sustentar un certificado. | J: `internal/modules/certificados/adapters/personalv1/`, dominio, aplicación, CLI y textos ES/EN. El lector de Personal sigue sin implementación. | El CLI prepara el borrador desde una muestra sintética con forma V1 y la traduce con el mismo adaptador que usará la respuesta real. |
| 3 | **ANA-002:** hacer visible en el cuadro CT el rango efectivo, zona horaria y corte recibidos. | J con cesión de F para `vista-estadisticas.js` y catálogos CT; sin nueva pantalla ni SQL. El servidor ya determina el ámbito y no devuelve su etiqueta. | Una consulta autorizada muestra contexto y totales coherentes; carga, error, denegación y móvil comprobados. |
| 4 | **CER-001, emisión:** obtener hechos de servicios autorizados de Personal por un puerto, revisión RRHH, documento original, firma/sello, registro, código seguro de verificación (CSV), entrega y conservación. | Personal B posee servicios; Certificados J posee plantilla/expediente; Documentos/firma aporta capacidades de PDF, firma y custodia por puertos nuevos o adaptados. Requiere permisos fijos, SQL nuevo **solo en borrador**, número reservado y orden en `ORDEN_SQL_NUCLEO.md` por D, clon y doble revisión antes de integrar. La verificación por código no abre la ficha Personal; RRHH debe aprobar sus datos visibles. | Navegador → identidad/permiso → datos de oficio → revisión → firma real → documento y código verificables → recibo e historia conservados tras reinicio. |
| 5 | **ANA-005:** registrar consulta y exportación nominales antes de entregar resultados. | Propietario CT F, con J para el consumidor y D para orden SQL; puerto de consulta atestada y exportación separada con `corte_esperado`. SQL 000107 solo lee el corte actual; el archivo CSV de hoy se genera en el navegador y carece de registro de descarga. SQL nuevo queda en borrador hasta reserva, ensayo y revisiones. Alertas de uso anómalo quedan pendientes de regla aprobada. | Revalidar identidad, permiso y corte; confirmar antes de devolver datos un registro durable mínimo de actor, finalidad, ámbito, versión, corte, campos, filas y resultado, sin contenidos de certificados ni filas nominales. Cambio de corte rechaza exportar lo visto antes; la traza acredita la puesta a disposición o el intento de respuesta, no recepción ni guardado. |

**Fuera del corte actual:** ANA-003 (transparencia) y ANA-004 (informes regulatorios/ISPA). Requieren proyección, plantillas, autoridad y decisiones propias; no están incluidas en las horas siguientes ni se deducen del cuadro CT.

## Estimación

Estimación inicial de los cortes descritos, pendiente de contrastar con los equipos dueños: un equipo Codex con varios subagentes, PR pequeñas de 1–3 horas, revisión independiente y CI; jornada de referencia de ocho horas. Las horquillas incluyen consumidores, integración y pruebas de J, **suponiendo disponibles** fuentes, decisiones y servicios externos. No estiman el cierre completo del gobierno ANA-001, del cuadro transversal ANA-002 ni las alertas ANA-005 sin regla aprobada.

| Minitarea | Horas de un equipo |
| --- | ---: |
| 1. ANA-001: montaje, Chrome, revisión y CI restantes | 3–5 h |
| 2. CER-001: revisión final, calidad y PR del borrador ya preparado | 3–5 h |
| 3. ANA-002: contexto visible del cuadro existente | 5–8 h |
| 4. CER-001: emisión por cortes de fuente, permisos, revisión, firma, verificación y recuperación | 48–72 h |
| 5. ANA-005: permiso, registro atómico, exportación separada, pantalla y recuperación | 40–64 h |

**Total de implementación pendiente:** 99–154 horas, unas **13–20 jornadas de un equipo**. Con dos equipos que trabajen en paralelo en Certificados y Analítica, con D/Personal/F como dueños de sus contratos y una integración final, **8–13 jornadas de calendario de trabajo**. No se ganan todas las horas en paralelo: ANA-005 depende de la frontera CT y CER emisión depende de Personal y Documentos.

**Fuera de nuestro control:** Personal B debe aportar servicios autorizados (8–16 h de su equipo tras acordar el contrato); D/F deben fijar perfiles, orden SQL y transacción de estadísticas (16–24 h de sus equipos, coordinadas con la minitarea 5); Documentos/firma y servidor deben aportar el circuito de firma/verificación (16–32 h de sus equipos si faltan adaptadores); RRHH debe decidir fuentes, plantillas, firmantes, gobierno y finalidades. Son **5–9 jornadas agregadas de esfuerzo ajeno**, que pueden solaparse con nuestras 8–13 jornadas y no se suman sin más al plazo. Las horas se conciliarán con los dueños para no contar dos veces un mismo contrato. La espera de decisiones institucionales y de disponibilidad del servidor no tiene fecha acreditada; no cabe convertirla en días de calendario cerrados.

## Decisiones pendientes de RRHH

Se trasladarán a `dudas.md` con números consecutivos **solo en el turno J**, después de las preguntas que añadan los equipos precedentes:

1. ¿Qué certificados de servicios y plantillas aprueba RRHH, quién revisa y firma/sella cada tipo, y qué datos de oficio son obligatorios?
2. ¿Qué fuentes acreditan periodos, jornada e interrupciones y cómo se distinguen los servicios declarados, comprobados y reconocidos?
3. ¿Quién responde de cada indicador, aprueba su fórmula y fija calidad, periodicidad, umbrales de revelación y ámbitos del cuadro?
4. ¿Qué finalidades y perfiles permiten consultar y descargar agregados, cuánto se conserva la traza y quién revisa alertas de uso anómalo?

## Por dónde empezar mañana y trabajo en curso

Primero: cerrar **ANA-001** en `trabajo/codexj-ana001-ficha-20261001` (`ff8b18ea7`, fuente remota conservada; falta cadena de versiones, Chrome y puerta), después revisar el borrador **CER-001** en `trabajo/codexj-cer001-borrador-20261001` (`a1d263a84`, fuente remota conservada; dos revisiones en curso) y abrir PR separada solo si supera sus puertas. No hay PR J abierta en este corte. ANA-002 y ANA-005 no tienen rama de implementación; no abrirlas antes de cerrar esas piezas y sus dependencias. La cola de compartidos vigente es B → A → E → G → F → M → H → I → J; A liberó a E con `749f3567e` y J aún no tiene turno.

## Consenso Astra

Astra y J acordamos que Personal conserva la autoridad de servicios; Certificados prepara y gobierna su expediente; Documentos aporta PDF, firma y custodia mediante contratos propios. El ensayo CER es solo un borrador. El [SAS distingue consulta de servicios sin firma de certificado oficial firmado](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/guia-laboral/servicios-previosprestados), y la [Seguridad Social permite verificar documentos por código](https://sede.seg-social.gob.es/wps/portal/sede/sede/Inicio/Serviciosverficacion?changeLanguage=es); VEC necesita su propia aprobación sobre qué muestra esa verificación y no expondrá la ficha Personal.

ANA-001 y ANA-002 son cortes parciales del cuadro CT: faltan responsable/fórmula aprobados, modelo transversal y reglas de revelación. Las [fichas metodológicas del INE](https://ine.es/dynt3/metadatos/es/RespuestaDatos.html?oe=30800) muestran el valor de fuente, periodo, unidad y control de identificación indirecta. [Mi Carpeta Ciudadana](https://estaticoscarpeta.carpetaciudadana.gob.es/) ilustra consulta desde organismos fuente; VEC conservará esa separación entre módulos. Para ANA-005 acordamos confirmar autorización y traza durable antes de responder, con alertas solo cuando RRHH apruebe su regla. Este acuerdo orienta el plan; no es GO de código, SQL, emisión ni firma.

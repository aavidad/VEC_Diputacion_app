# Cobertura de auditoría de VEC — 3 de octubre de 2026

VEC ya tiene una auditoría común de consumo V3, cadenas de huellas, consultas autorizadas y registros técnicos. La cobertura es parcial: una operación permitida que consume V3 puede dejar evidencia común, pero eso no acredita que todos los errores, denegaciones, consultas y descargas de cada módulo tengan el registro nominal completo exigido el 3 de octubre.

Este inventario permite repartir los huecos sin crear otra autoridad. El recorrido inicial se hizo sobre `main` en `936aac665084b57b0d71adad15672e85e0efbb6a`. La actualización del 04/10 contrasta los estados de integración y las lecturas y descargas CT/Bolsa con `296f78373f4874b9a81def9df744df12c1096af1`. Las filas de otros módulos conservan el alcance del recorrido inicial. No certifica instalación SQL, despliegue, datos de la principal ni cobertura exhaustiva de cada ruta. En esta revisión documental no se ha consultado una base de datos ni ejecutado una operación funcional.

El [contrato de auditoría común](../portal_vec/contrato_auditoria_comun.md) recoge las firmas vigentes y las condiciones de integración para K y los demás módulos.

## Criterio y fuentes

La orden «AUDITORÍA TOTAL CON VALOR DE PRUEBA» de `comun.md`, 03/10, exige identidad y perfil activo, acción, recurso opaco, finalidad, instante, resultado, proceso, canal y correlación. El registro se añade a la auditoría común y se confirma con el efecto. Los intentos denegados y los errores también deben conservarse; una reversión del efecto no debe hacer desaparecer su constancia. No se inventa una identidad para una autenticación que no llegó a verificarse.

Se han leído [ESPECIFICACIONES_AGENTES.md](../../ESPECIFICACIONES_AGENTES.md), especialmente E03–E07 y E10, [cumplimiento y seguridad](../portal_vec/cumplimiento_y_seguridad.md), [persistencia PostgreSQL](../portal_vec/seguridad_persistencia_postgresql.md) y las dudas [67, 71 y 84](../../dudas.md). La 71 mantiene pendiente la fuente autorizada del nombre visible; una referencia nominal estable no debe sustituirse por el nombre aportado en una cabecera. La 84 no aprueba los dos años propuestos para accesos ni autoriza borrados.

La búsqueda empezó en `codebase-memory-mcp` (`list_projects`, `search_graph`, `search_code`). El índice disponible pertenecía a otro worktree; sus resultados se usaron para localizar archivos y se contrastaron con el árbol del commit indicado. No había `gopls` en el `PATH` de esta revisión documental.

En las tablas, **común V3** significa registro en la autoridad de autorización V3 ligado al consumo, **local** significa historia o bitácora del propietario y **ensayo** significa capacidad sintética. «Pendiente» significa que este recorrido no acredita esa propiedad; no afirma que falte en todas las demás operaciones del módulo. Un recibo con `auditoria_ref` permite seguir la relación, pero no sustituye comprobar su productor SQL.

## Autoridades que deben ampliarse

| Pieza existente | Evidencia | Qué acredita y qué falta |
| --- | --- | --- |
| Modelo común `AuditEntry` y puerto `AuditStore` | [`internal/vec/domain/types.go`](../../internal/vec/domain/types.go), `AuditEntry`; [`internal/vec/ports/ports.go`](../../internal/vec/ports/ports.go), `AppendAudit` / `ListAudit` | Tiene actor, perfil, finalidad, resultado, recurso, instante y correlación, además de huellas y secuencia. No tiene campos propios de proceso/canal. Sus campos opcionales y `Metadata` no garantizan que cada productor los complete. |
| Auditoría durable interna V3 | [AD3-1](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql), `auditoria_consumo_v3` / `control_cadena_auditoria`; [AD3-2](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql), `registrar_y_consumir_decision_v3_atestada` | Confirma decisión, consumo y cadena bajo bloqueo de la cabeza. La fila auditada contiene referencia de decisión/efecto, huellas, secuencia e instante. El contexto nominal se conserva relacionado con la atestación; la fila no expone por sí sola perfil, finalidad, canal o resultado de error. |
| Linaje externo V3 | [AD116](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000116_consumo_candidato_externo.up.sql), `auditoria_consumo_v3_externa` / `control_cadena_auditoria_externa` | Hay separación del consumo externo. La ampliación debe respetarla, sin copiar fichas de aspirantes a un almacén interno de usuarios. No se ha ensayado la cadena externa en este corte. |
| Auditoría temprana de frontera | [`auditoria_frontera_ruta_exacta.go`](../../internal/vec/ports/auditoria_frontera_ruta_exacta.go), `RegistradorAuditoriaFronteraRutaExacta`; [CT108](../../deploy/postgresql/contratacion_temporal/migraciones/000108_auditoria_frontera_ruta_exacta.up.sql) y sus ampliaciones | Conserva correlación, motivo, superficie, ruta, actor opcional e instante. La tabla es de solo adición. No transporta perfil, finalidad ni canal. En la consulta de auditoría se escribe después de observar 401/403; un fallo de registro produce `slog` y no cambia la denegación. Ese caso no acredita auditoría durable total. |
| Consulta común | [`internal/vec/auditoria/contrato.go`](../../internal/vec/auditoria/contrato.go), `Filtro` / `FuenteAuditoria`; [`servicio.go`](../../internal/vec/auditoria/servicio.go), `Consultar`; [`filtro.go`](../../internal/vec/auditoria/filtro.go), `Validar` | Solo admite fuentes `ct` y `bolsa`, expediente obligatorio, actor opcional y fechas. Liga filtros y cursor a autorización. No incluye filtro por acción ni consulta general por persona sin expediente. La proyección no devuelve perfil, finalidad, proceso, canal o correlación de cada evento. |
| Auditoría en memoria | [`internal/vec/adapters/memory/store.go`](../../internal/vec/adapters/memory/store.go), `appendAuditLocked` | Encadena `sha256-chain-v1` en memoria. No acredita durabilidad, TSA ni protección frente a un administrador del almacén. |
| Adaptador durable del puerto antiguo | [`bolsa/adapters/postgres/registroaccesos/registro.go`](../../internal/modules/bolsa/adapters/postgres/registroaccesos/registro.go), `AppendAudit` / `ListAudit` | Ambas operaciones genéricas deniegan expresamente: no ligan una capacidad al efecto. La consulta administrativa usa un contrato específico y transacción. K no puede usar este adaptador como si fuera un registrador nominal general disponible. |
| Intentos nominales comunes integrados | [`ports/auditoria_intento_nominal.go`](../../internal/vec/ports/auditoria_intento_nominal.go), `RegistradorIntentosAuditoria.AppendIntentoAuditoria`; [`postgres/auditoria_intento_nominal.go`](../../internal/vec/adapters/postgres/auditoria_intento_nominal.go); [AD169](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000169_intentos_nominales_auditoria_comun.up.sql) | Registra únicamente `denegado` o `error` tras cerrar el intento original, con contexto/vínculo nominales y proceso/canal configurados. Devuelve acuse tras COMMIT y admite recuperación con la misma orden. No acepta el éxito de una consulta o descarga ni sustituye su consumo V3. El puerto en `main` no acredita el montaje de las cuatro PR pendientes. |

Los registros locales existentes son historia que hay que conservar. No se trasladan ni se borran para cumplir la nueva orden. Las piezas nuevas deben ampliar la autoridad común; la relación con la historia del módulo se hace mediante referencias verificables, sin otra tabla de auditoría paralela.

## Inventario por módulo

| Módulo y propietario de coordinación | Operaciones comprobadas | Cobertura observable y transacción | Lecturas, descargas y hueco por cerrar |
| --- | --- | --- | --- |
| Contratación temporal — E en firma/circuito; dirección en el resto | Alta: [`confirmacion_alta.go`](../../internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta.go), `NuevaTransaccionAltasPostgreSQLCandidata`. Seguimiento: [`seguimiento_operaciones.go`](../../internal/modules/contrataciontemporal/adapters/postgres/seguimiento_operaciones.go), repositorio de operación. Cuadro, detalle, resumen y original: [`consulta_rrhh_postgresql_sql.go`](../../internal/modules/contrataciontemporal/adapters/postgres/consulta_rrhh_postgresql_sql.go). | Las consultas usan fachadas atestadas y devuelven `auditoria_vec_ref` y su huella. Alta y seguimiento tienen adaptadores transaccionales propios. No se atribuye cobertura completa a todas las fases a partir de estos ejemplos. | La lectura del original conserva consumo propio; la generación y entrega del PDF debe distinguirse de la consulta del expediente. Registrar cada descarga por acción/recurso/canal y comprobar sus errores. Auditoría de frontera temprana parcial, CT108/136/162. No duplicar firma ni originales de E. |
| Bolsa — A en baremo/selectivos; dirección en llamamientos | Gobierno de convocatorias: [`convocatorias_consulta.go`](../../internal/modules/bolsa/adapters/postgres/convocatorias_consulta.go). Panel: [`panel_interno.go`](../../internal/modules/bolsa/adapters/postgres/panel_interno.go), `ConsultarPanel`. Llamamientos: [`llamamientos_transaccion.go`](../../internal/modules/bolsa/adapters/postgres/llamamientos_transaccion.go). | Consumo autorizado y transacción en los adaptadores. Hay historia local de llamamientos y sus replays. [`registroaccesos/registro.go`](../../internal/modules/bolsa/adapters/postgres/registroaccesos/registro.go), `ConsultarAccesosAdministrativos`, confirma autorización, lectura y auditoría juntas. | Corte del 04/10: Mi Bolsa y su historial en `nuevaRutaMiBolsaDesarrollo` añaden intentos nominales comunes después del cierre de las lecturas fallidas. Los permisos y consumos de lecturas permitidas se conservan. Archivo privado `auditoria-intentos-externa.json`, canal `externa_personal` y LOGIN dedicado con CA26/IS13 comunes; no se reutiliza la cuenta corporativa. El portal exterior separado usa autoridades externas distintas y queda pendiente de su registrador propio compatible; no se declara cubierto por este montaje sintético. Pendiente ensayo nominal con K. La fuente de la consulta común de Bolsa es [`auditoriaconsulta/fuente.go`](../../internal/modules/bolsa/adapters/auditoriaconsulta/fuente.go). Ampliar campos nominales y fallos sin sustituir las funciones existentes. Revisar por separado los documentos y descargas; esta fila no certifica esas rutas. |
| Personal y RPT — B | Altas/hechos: [`registro_empleado_b2_actos.go`](../../internal/modules/personal/adapters/postgres/registro_empleado_b2_actos.go), `RegistrarEmpleadoRRHH` / `RegistrarHechoEmpleadoRRHH`. Ficha/vacantes: [`registro_empleado_b2_consulta.go`](../../internal/modules/personal/adapters/postgres/registro_empleado_b2_consulta.go), `ConsultarFichaRRHH` / `ListarVacantesRRHH`. Listado: [`registro_empleado_b2_lista.go`](../../internal/modules/personal/adapters/postgres/registro_empleado_b2_lista.go), `ListarEmpleadosRRHH`. | Material V3 y recibos con auditoría propia de acceso actual, también al recuperar operaciones. Las funciones diferencian recibo histórico y acceso actual. La existencia de los recibos no certifica aquí todas las ramas SQL de error. | Datos personales en fichas y listas: mantener consumo/auditoría antes de devolverlos. Relación/organización histórica tienen adaptadores y fronteras propios. No duplicar el lector nominal RPT de #437/#438. Incorporar su fuente a la consulta común cuando B cierre el contrato. |
| Aspirantes — coordinación A/B; identidad común K | [`aspirantes/adapters/postgres/ficha.go`](../../internal/modules/aspirantes/adapters/postgres/ficha.go), `ConsultarPropia`, `Alta`, `Rectificar`, `RegistrarAuditoriaFronteraRutaExacta`. | Las operaciones se ejecutan en transacción serializable. Las denegaciones de frontera usan `registrar_denegacion_frontera_v1` y no aceptan actor libre; la auditoría local de acceso existe en la migración de ficha. | No inferir el nombre o el perfil de la carga HTTP. Verificar la proyección nominal externa y sus vínculos a auditoría común antes de ampliar la administración. No hay descarga de ficha acreditada en este recorrido. |
| Usuarios, perfiles y revocación — K | Preferencias: [`usuarios/adapters/postgres/preferencias.go`](../../internal/modules/usuarios/adapters/postgres/preferencias.go), `ConsultarPropias`, `Guardar`, `RecuperarOperacion`. Correos/avisos: [`correo_avisos.go`](../../internal/modules/usuarios/adapters/postgres/correo_avisos.go). Frontera: [`auditoria_frontera_preferencias.go`](../../internal/modules/usuarios/adapters/postgres/auditoria_frontera_preferencias.go). | Preferencias confirma fachadas V3 mediante transacción serializable. Frontera tiene registrador separado. Identidad y perfiles son autoridades comunes, no se sustituyen por las preferencias del usuario. | K debe ligar provisión, cambio de perfil y revocación al contrato común completado por L. Consultas de correos e imagen propia requieren registro, también lectura/retirada y fallos. Este inventario no da por cerrado el nuevo trabajo de K. |
| Méritos — A | [`meritos/application/servicio.go`](../../internal/modules/meritos/application/servicio.go), `Declarar`, `Rectificar`, `Rechazar`, `Verificar`; [`ports/auditoria.go`](../../internal/modules/meritos/ports/auditoria.go), `AuditoriaIntentos`. [RUM03](../../deploy/postgresql/meritos/migraciones/000001_registro_hechos_v1.up.sql). | El caso de uso construye actor/perfil/finalidad/correlación y registra intentos no confirmados. Los éxitos pertenecen al efecto durable. La migración conserva `auditoria_operacion`, `acceso_operacion` y outbox, vinculados al consumo V3. `Verificar` permanece pendiente de acreditación en el caso de uso revisado. | Consulta propia con recibo nominal de #435 integrada; no repetirla. El estado de integración no amplía la evidencia de este recorrido. Revisar que su proyección y las descargas futuras no se limiten al historial local y no expongan documentos completos en la auditoría. |
| Selección — A | [`seleccion/application/consultar_convocatoria.go`](../../internal/modules/seleccion/application/consultar_convocatoria.go), consulta; [`preparacion_bases.go`](../../internal/modules/seleccion/application/preparacion_bases.go), `PrepararMaterialBases`. | La preparación revisada es propuesta sintética: declara pendientes de fuente/firma/custodia y no crea borrador gobernado. No acredita un efecto institucional auditado. | Las PR #442–#446 preparan frontera, servicio, SQL y montaje; no repetirlas. La auditoría nominal de guardar/consultar bases pertenece a esa entrega y a su consumidor común V3. |
| Cronos — E | [`cronos/adapters/postgres/transaccion_marcaje.go`](../../internal/modules/cronos/adapters/postgres/transaccion_marcaje.go), `ejecutarTransaccionMarcaje` / `auditarFallo`. Consultas: [`consulta_saldo.go`](../../internal/modules/cronos/adapters/postgres/consulta_saldo.go), [`empleado_solicitudes.go`](../../internal/modules/cronos/adapters/postgres/empleado_solicitudes.go). Permisos y avisos: [CRN10](../../deploy/postgresql/cronos_v1/migraciones/000010_circuito_y_notificaciones_rrhh.up.sql). | Marcajes distinguen fallo confirmado de resultado indeterminado del COMMIT y registran el resultado sin inventar reversión. CRN10 llama consumidores V3 de bandeja, resolución y notificaciones; conserva accesos locales con `auditoria_ref`. | Cerrar y proyectar en la común saldo, movimientos, permisos, bandejas, avisos y lecturas personales. El fallo posterior al rollback tiene transacción de registro distinta del efecto revertido; conservar esa distinción. No duplicar la cola E. |
| Dietas — E; relaciones del empleado B | [`dietas/adapters/postgres/acceso_rutas.go`](../../internal/modules/dietas/adapters/postgres/acceso_rutas.go), `ConsumirAccesoRutasDietas`; [`auditoria_frontera.go`](../../internal/modules/dietas/adapters/postgres/auditoria_frontera.go), `RegistrarAuditoriaFronteraComision`. Personal: [`solicitud_rectificacion_dietas_auditoria.go`](../../internal/modules/personal/adapters/postgres/solicitud_rectificacion_dietas_auditoria.go). | Acceso a rutas confirma consumo V3 y auditoría antes de retornar. Fronteras de comisión y rectificación conservan motivo, actor verificado cuando existe, recurso y estado en contratos propios. | Documento de comisión, consulta del detalle y exportaciones requieren acción diferenciada y auditoría nominal común. El registro de una denegación de frontera no acredita por sí mismo auditoría de la descarga permitida. |
| Documentos comunes — E en originales/firmados; L en consulta de auditoría | [`documentos/adapters/postgres/repositorio.go`](../../internal/vec/documentos/adapters/postgres/repositorio.go), `transaccion`; [DOC1](../../deploy/postgresql/documentos/migraciones/000001_documentos_comunes.up.sql), `confirmar_alta_v1`, `listar_expediente_v1`, `obtener_original_v1`, `preparar_notificacion_v1`. | Las fachadas consumen V3 en la misma transacción. La bitácora local guarda actor/perfil/finalidad/correlación y referencia a auditoría AD3, con acciones diferentes de alta, lista y descarga. `documentos.original.descargar` está expresamente registrado. | El permiso de obtener el original no prueba que la transmisión completa acabara bien. Añadir resultado de entrega/fallo cuando corresponda sin reescribir la descarga autorizada. Frontera [`frontera.go`](../../internal/vec/documentos/adapters/postgres/frontera.go) tiene registro propio: ampliar la común. #457/#460/#466 integradas; no se repiten ni se atribuye su instalación por esta revisión. |
| Administración y copias — K en copias; L en logs/auditoría | [`administracion/application/registrocopias/servicio.go`](../../internal/modules/administracion/application/registrocopias/servicio.go), `Reservar`, `Aplicar`, `Consultar`, `Listar`. | El diario offline recibe actor/correlación y coordina el registro; su comentario excluye decisiones de permiso. No se atribuye a esta capa un consumo V3 ni un registro común nominal completo. | #392/#404–#407 integradas: captura, vista/API, abandono y separación de claves. Seguir esos contratos; no crear otra vista de copias. La vista técnica para Sistemas y exportación judicial corresponden a L. |
| Calendarios — dirección | [`calendarios/application/consulta.go`](../../internal/modules/calendarios/application/consulta.go), `Centros`, `CalendarioCentro`, `CalcularPlazo`. | Consultas de calendario por repositorio y reloj; el servicio no recibe identidad ni auditoría. Es cálculo compartido, no un productor de registro nominal. | Registrar la operación de la persona en su caso de uso consumidor. No auditar cada cálculo interno como si fuese una nueva acción de usuario. Cambios administrativos de calendario quedan fuera del recorrido revisado. |
| Carrera — coordinación A | [`carrera/application/preparar.go`](../../internal/modules/carrera/application/preparar.go), `Preparar` / `Ejecutar`. | Preparación de escenario y entrada/salida de ensayo; esta pieza no acredita registro durable nominal. | PR #388 prepara expediente sintético. Antes de habilitar consulta personal o cambio administrativo debe consumir autoridad y auditoría comunes. No abrir una vertical paralela desde L. |
| Formación — dirección | [`formacion/application/preparar.go`](../../internal/modules/formacion/application/preparar.go), `Preparar`. | Preparación de datos para revisión. No se ha trazado una operación HTTP de registro/consulta durable desde esta pieza. | Cobertura de cursos personales, validación, consulta y descargas pendiente de trazado cuando se componga la vertical real. No inventar operaciones productivas a partir del preparador. |
| Certificados — dirección | [`certificados/application/preparar.go`](../../internal/modules/certificados/application/preparar.go), `PrepararEnsayo`. | Produce PDF sintético; el contrato excluye expedición, reconocimiento, autorización, firma, registro y notificación. | No contar ese PDF como descarga nominal de un certificado emitido. La futura consulta de servicios y la entrega al usuario necesitan auditoría común, separadas de la firma documental. |
| Provisión — coordinación B | [`provision/application/concursos_simular.go`](../../internal/modules/provision/application/concursos_simular.go), `Simular`; resto de preparadores de ciclo/concurso. | Valoración local; no dispone de puertos institucionales ni acepta persistencia como efecto. | No hay una escritura institucional acreditada por este recorrido. Antes de habilitar consulta o adjudicación real debe recibir contrato nominal y auditoría ligada al efecto. |

## Campos y resultados que aún no se acreditan de forma general

| Requisito del 03/10 | Situación comprobada | Trabajo acotado |
| --- | --- | --- |
| Identidad y perfil activo | `AuditEntry` tiene ambos; los consumos V3 validan contexto y perfil. Frontera temprana admite actor ausente y no lleva perfil. | Construir la orden desde contexto verificado. Distinguir intento sin identidad acreditada de una operación nominal; no admitir datos libres para completar el actor. |
| Acción, recurso opaco y finalidad | Existen en el modelo común y las fachadas nominales. La bitácora temprana conserva ruta/motivo, no finalidad. | Descriptor gobernado de acción y recurso para el evento, con finalidad propia; evitar guardar cuerpo, URL completa o documento. |
| Permitido, denegado y error | El consumo común está ligado a una decisión consumida. Méritos usa `no_confirmado`; Cronos diferencia indeterminación y fallo. Los estados no forman un contrato transversal completo. | Catálogo común de resultado, con motivo cerrado. Mantener el detalle de resultado indeterminado sin convertirlo en éxito ni rollback confirmado. |
| Instante, proceso, canal y correlación | Modelo común conserva instante/correlación. Procesos y canales no son campos propios de `AuditEntry`; incidencias generan una correlación técnica distinta. | Fijar estos campos desde composición/contexto y propagarlos sin cabeceras libres. La correlación técnica debe enlazar con la de la petición. |
| Atomicidad y solo adición | Consumo V3 y múltiples fachadas se confirman juntos. Frontera escribe después de la respuesta; el log de su fallo no garantiza registro durable. | Cada efecto autorizado debe confirmar su registro común antes de devolver datos o recibo. Los fallos/denegaciones sin efecto requieren registro separado y política explícita de indisponibilidad, sin conceder permiso. |
| Consultas y descargas personales | Hay ejemplos positivos: ficha B2, ficha propia aspirante, cuadro/detalle CT, originales Documentos, saldos/avisos Cronos. | Verificar cada familia; ni consultar un expediente ni generar un PDF sustituye registrar la descarga concreta. |

## Integridad, consulta y conservación

La cadena interna V3 ya tiene secuencia única, huella anterior y cabeza bajo `FOR UPDATE`. En AD3-2, la preimagen encadena secuencia, huella anterior, decisión, efecto, huella del efecto y huella de consumo. `registrada_en` se almacena, pero no aparece como campo separado de esa preimagen. Una ampliación no puede declarar protegidos todos los campos nominales o el instante solo porque existe `huella_sha256`.

El adaptador [`tsa_desarrollo.go`](../../internal/app/bootstrap/tsa_desarrollo.go), `Timestamp`, genera un HMAC rotulado `no_autoritativa` y `migrable_a_produccion=false`. El [`KMS de desarrollo`](../../internal/app/bootstrap/kms_desarrollo.go) distingue emisor, revalidador y verificador, con claves fijadas. Son capacidades reutilizables de desarrollo; no acreditan sello cualificado ni exportación judicial firmada. [`internal/candidate/domain/audit.go`](../../internal/candidate/domain/audit.go), `VerifyAuditChain`, verifica una cadena SHA256 construida con una referencia de firma; no es prueba de firma asimétrica de la auditoría común.

El verificador común, la comprobación de continuidad y el sello periódico de desarrollo están integrados. Falta acreditar una TSA independiente admitida y la cobertura de cada nuevo productor; los formatos aceptados se contrastan con el lector compartido antes de ampliar la exportación. Para detectar una eliminación del final hace falta comparar con un anclaje conservado fuera de la porción examinada; verificar solo que los elementos recibidos se enlazan no acredita que estén todos. Un evento anterior conserva su esquema y su huella; no se recalcula historia como si ya llevara campos nuevos.

La consulta actual pagina como máximo 100 registros e intervalos de 31 días. Su fuente CT llama `consultar_auditoria_ct_atestada_v1`; confirma la transacción antes de devolver la página. La fuente Bolsa sigue el mismo puerto. Las rutas del contrato común son opciones y consultas: no se ha encontrado en este contrato una operación de exportación judicial. Ampliar filtros por acción/persona/recurso requiere su propia autorización ligada a filtros, y la exportación debe dejar un registro nominal antes de entregar el archivo.

La política documental existente usa [`conservacion/catalogo.go`](../../internal/vec/adapters/conservacion/catalogo.go) y catálogos versionados, con serie, procedimiento, plazo y protección. Sus valores están rotulados provisionales. No es todavía una política aprobada de conservación de auditoría. La ampliación debe enlazar cada serie con su resolución, fecha inicial, bloqueo judicial y versión. Mientras no haya resolución aplicable no se programa expurgo. La rotación de logs técnicos no debe confundirse con borrar evidencia funcional.

## Registros técnicos: reutilización y huecos

| Capacidad | Evidencia comprobada | Pendiente de L |
| --- | --- | --- |
| Incidencias comunes y minimizadas | [`domain/incidencia_tecnica.go`](../../internal/vec/domain/incidencia_tecnica.go), `NuevaIncidenciaTecnica` / `ClasificarIncidenciaTecnica`: esquema `vec.incidencia_tecnica.v1`, catálogo de códigos, severidad, componente, etapa, entorno, versión, correlación y recuento acotado. | La nueva capa de logs no debe aceptar errores libres, secretos, nombres, certificados, cuerpos ni datos personales para completar eventos. |
| Emisión estructurada y contadores | [`observabilidad/emisor_jsonl.go`](../../internal/vec/adapters/observabilidad/emisor_jsonl.go), `EmisorJSONLines`: cola acotada, JSONL, saneamiento, contadores y declaración `RECOLECCION_DEGRADADA` por descartes. | Recogida durable común con rotación/retención configurables. Los descartes admitidos en telemetría no cumplen por sí mismos auditoría funcional total. |
| Errores HTTP y pánicos | [`server/supervision/supervision.go`](../../internal/app/server/supervision/supervision.go), `SupervisarServidor` / `SupervisarRespuestas`: 5xx, pánicos y ErrorLog saneado. | Su correlación la genera el emisor y no conserva ruta/IP/cuerpo. Enlazar petición→BD→efectos mediante contexto común, sin perder minimización. |
| Registros `slog` existentes | [`bootstrap/auditoria_consulta.go`](../../internal/app/bootstrap/auditoria_consulta.go), `ServeHTTP`: registra fallo de bitácora con correlación/ruta/causa fija. | Formato, nivel, componente y límites homogéneos para procesos. Un `slog.Error` no sustituye el registro nominal de la operación denegada. |
| Vista y alertas | El catálogo ya contempla errores de arranque, dependencia V3, PostgreSQL y alertas no entregadas. Las rutas revisadas de auditoría son funcionales CT/Bolsa. | Vista técnica solo Sistemas, filtros propios y auditoría de sus consultas; umbrales configurados para arranque, denegaciones anómalas y validador caído. No inferir que exista una instalación central por la presencia del emisor. |

## Lecturas y borradores revisados el 04/10

Los seis PDF de desarrollo de CT comparten
`POST /api/vec/contratacion-temporal/expedientes/consultas`. La lectura y su
consumo V3 se confirman antes de renderizar. La acción auditada es
`contratacion_temporal.expediente.consultar`: no distingue qué borrador se
ha descargado. Permanecen pendientes el registro nominal de los fallos de
renderizado y la auditoría positiva de la descarga concreta. El transporte
tampoco registra actualmente el resultado de escritura de bytes. No se atribuye
entrega completa por confirmar el permiso de lectura. Originales y firma
pertenecen al circuito E/Documentos.

En Bolsa, la lectura RRHH de solicitudes documentales pendientes consume
Bolsa77/AD155 y confirma antes de devolver metadatos. El corte nuevo conecta
el registrador común AD169 para denegaciones y errores posteriores al contexto
acreditado, con recurso de participación, acción de consulta y finalidad ya
existentes. Usa el archivo interno `auditoria-intentos.json`, su LOGIN dedicado
y su preflight. El acuse se coteja antes de publicar las referencias; un fallo
del registro o una proyección inválida cierra sin datos. Pruebas sintéticas
focales; ensayo nominal, reinicio y provisión de K pendientes.

La lista B5 de aspirantes combina lectores y staging, sin consumo nominal único
acreditado en este inventario. No se la presenta como cubierta por un decorador
de errores. El histórico B13 usa una acción de cambio preexistente y requiere
su propio corte. Bolsa conserva referencias y huellas de documentos; no se ha
localizado aquí una descarga privada de sus bytes fuera de los circuitos
excluidos. Mi Bolsa e historial y la consulta de auditoría conservan sus cortes
independientes, sin repetirlos en esta pieza.

## Cola de piezas sin solapamiento

| Orden | Pieza útil de L | Dependencia y límite |
| --- | --- | --- |
| 1 | Contrato común e intentos AD169 integrados; consumidores pendientes en #540/#581/#582/#591. | Ensayar cada consumidor con la fuente nominal de K; no activar `AppendAudit` genérico ni usar AD169 para éxito. |
| 2 | Verificador común, continuidad y sello periódico AD186 integrados. | Cada formato nuevo necesita admisión y comprobación en el lector compartido. TSA de desarrollo sigue rotulada; no acredita una TSA independiente. |
| 3 | Consulta administrativa común con proyección nominal y filtros por acción/persona/recurso; fuentes registradas por capacidad. | No leer tablas de A/B/E/K desde L. Cada propietario aporta adaptador autorizado. Conservar la minimización y el permiso propio de consulta. |
| 4 | Paquete verificable de desarrollo integrado; captura nominal y entrega pendientes. | Fuente propia de Aplicación, concesión exacta y acuse durable. Ni la captura periódica ni su firma conceden exportación judicial. |
| 5 | Recogida/consulta técnica para Sistemas, con rotación, contadores y umbrales configurados. | Reutilizar incidencias/emisor/supervisión. No mezclar permisos de administrador funcional y Sistemas ni duplicar la pantalla de copias de K. |
| 6 | Preservación técnica provisional AD187 integrada; conservación por serie y bloqueo judicial pendientes. | Resolución de Archivo/DPD y duda 84; sin eliminación. |
| Después | Huecos de cada módulo, una familia por PR. | A: Méritos/selectivos; B: Personal/RPT; E: firma/Cronos/Dietas; K: usuarios/perfiles/revocación. Dirección asigna CT/Bolsa restantes. L mantiene contratos y consulta común. |

## Estados contrastados — 04/10

GitHub confirma integradas #435 (Méritos), #442–#446 (baremo/preparación de bases y frontera), #437/#438 (lector Personal→RPT), #448/#451/#453/#456/#457/#458/#460/#464–#466 (firma, circuito y originales) y #392/#404–#407 (copias). La lista anterior de PR abiertas queda sustituida. Integración en código y evidencia funcional de cada entrega conservan alcances distintos.

Las cuatro PR de lecturas siguen abiertas y en borrador. Sus estados y hashes se consultaron el 04/10; no están en la base `296f78373`:

| PR | Candidata | Consumidor pendiente de integrar |
| --- | --- | --- |
| [#540](https://github.com/aavidad/VEC_Diputacion_app/pull/540) | `3c70d887aaab2a988df6a4848c7ddf9725946edb` | Denegaciones y errores de la consulta común RRHH de auditoría CT/Bolsa. |
| [#581](https://github.com/aavidad/VEC_Diputacion_app/pull/581) | `7bb963c42df269edbf772bde9c82778e09db93c6` | Mi Bolsa e historial en la composición sintética con CA/IS comunes; no el portal exterior separado. |
| [#582](https://github.com/aavidad/VEC_Diputacion_app/pull/582) | `d06adba863395203708332bb75354471631e2138` | Consulta del recibo de respuesta y comunicaciones del expediente CT. |
| [#591](https://github.com/aavidad/VEC_Diputacion_app/pull/591) | `e40bd055d9e450a5c717ece5fb221a4c7f512317` | Consulta RRHH de solicitudes documentales pendientes de Bolsa. |

Las cuatro conservan pendiente el ensayo de su consumidor con la fuente nominal de K y el registrador dedicado. Sus pruebas previas no se repiten por esta actualización; no se declara recorrido PostgreSQL o navegador del consumidor a partir del ensayo separado de AD169. Los rechazos anteriores al contexto acreditado conservan su frontera existente.

La revisión de seguridad de este inventario es estática y focal: comprueba propiedad de autoridades, separación de fuentes, minimización y las limitaciones de atomicidad descritas. No es una auditoría completa ni acredita una vulnerabilidad en producción. El único cambio es este documento; no requiere ensayos SQL, `gosec`, Semgrep de código, pruebas Go ni navegador. Se comprueban enlaces a archivos existentes y `git diff --check`.

## Exportación, sello periódico y preservación — 04/10

El paquete de desarrollo dispone de manifiesto tipado, proveedor KMS/TSA y
comprobación de firma, archivo exacto y eslabones en `verificar-exportacion`.
Los tramos históricos mantienen el aviso de fecha no ligada. #561 y #602 están
integradas mediante `466ed84306a3bac4d2c6f244425ac80a7e68b9b0` y
`c2356409b4e2a5695ef9408dce04e626eb62ac65`, ambos ancestros de la base revisada.
El origen de la captura sigue sin acreditarse por la firma del paquete. La
emisión de aplicación exige `FuenteCapturaExportacionAuditoria`; aún falta su
adaptador de captura nominal y el catálogo de acciones de Aplicación. No hay
emisión CLI desde una captura JSON libre. La consulta CT/Bolsa y las capturas
técnicas siguientes no conceden extracción judicial de registros personales.

#609 está integrada en `8c19cc162bbb875892fd412f10d3cf414c15f4b9`: [AD186](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000186_sello_periodico_auditoria.up.sql),
[`FuenteCheckpointPeriodico`](../../internal/vec/ports/auditoria_periodica.go),
adaptador PostgreSQL y CLI `ejecutar-periodico`. Captura y acuse técnicos se
confirman juntos; el recibo firmado se conserva con su confirmación y puede
recuperarse. Usa autoridad técnica propia y no devuelve filas personales.
Mantiene TSA HMAC de desarrollo, sin tiempo independiente ni firma legal.

#618 está integrada en `3ba1a77a79ab550c98756e873ea899571e80d7df`: [AD187](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000187_preservacion_provisional_auditoria.up.sql),
[`FuentePreservacionAuditoria`](../../internal/vec/ports/preservacion_auditoria.go),
adaptador y CLI de publicación/consulta versionadas. La medida es
`conservar_todo_sin_expurgo`, provisional y auditada. No fija serie documental,
plazo aprobado ni resolución del Archivo/DPD; la duda 84 sigue abierta.
Ambos merges son ancestros de la base revisada. Este estado describe código
integrado; no acredita instalación o despliegue por esta revisión.

El punto 4 sigue abierto por la captura nominal, su entrega auditada y la
consulta por Aplicación. El punto 6 dispone de preservación técnica provisional;
la conservación documental aprobada permanece pendiente. Las [instrucciones
del comando](../../cmd/vec-auditoria-checkpoint/README.md) describen las cuentas
y límites de las operaciones técnicas.

## Descargas de borradores CT: contrato pendiente para E/K

El [manejador de detalle](../../internal/modules/contrataciontemporal/adapters/httpinterno/consulta_rrhh.go),
`ServeHTTP` → `responderBorrador`, representa el detalle autorizado y entrega
los bytes por la ruta de consulta. La lectura confirma su consumo antes de
retornar mediante
[`ServicioConsultaDetalleRRHH`](../../internal/modules/contrataciontemporal/application/consulta_detalle_rrhh.go),
`SesionConsultaRRHH.ConsultarDetalleYRegistrar`. No hay en ese recorrido un
segundo consumo o registro común que identifique la descarga concreta.

La base revisada admite diez tipos de borrador: los seis originales (informe,
resolución, diligencia, toma de posesión, notificación y comunicación al centro)
y contrato laboral, nombramiento, cese y modificación. PDF y DOCX comparten
la lectura autorizada. Desde los detalles v8/v9 puede recuperar el original
v7 antes de renderizar; la auditoría de esa segunda lectura conserva su acción
de consulta. No representa por sí sola una descarga. Originales, custodia y
firmados mantienen la autoridad de Documentos y el ámbito exclusivo de E.

El siguiente corte necesita estas entradas antes de montar el consumidor:

- E/dirección fija el descriptor gobernado de descarga de borrador: referencia
  opaca del expediente, versión consultada y versión representada, tipo,
  formato, finalidad, ámbito y campos autorizados. Identidad, perfil,
  correlación, proceso y canal proceden de las autoridades comunes. No se
  guardan contenido, nombre de persona ni cuerpo HTTP en la auditoría.
- K y el propietario del consumo aportan la concesión exacta y la fachada
  durable que confirme la descarga permitida en la auditoría común. El permiso
  actual de consulta y `documentos.original.descargar` no se reutilizan para
  una acción de borrador distinta. Este documento no asigna claves nuevas,
  publica permisos ni reserva una migración.
- Aplicación y HTTP entregan bytes sólo después del acuse confirmado para esa
  acción. Renderizado, cancelación, fallo de registro y rechazo se conservan
  como resultados propios. AD169 permite el intento nominal `denegado` o
  `error` después de cerrar el intento; no admite `permitido` ni demuestra
  que el cliente recibiera todos los bytes. Un resultado de transmisión sería
  un evento relacionado posterior, sin reescribir el acceso confirmado.

La aceptación debe recorrer cada tipo/formato admitido, la recuperación v7,
la denegación, el fallo de renderizado, la cancelación y el fallo de auditoría
sin bytes parciales. El ensayo nominal debe contrastar acción/recurso y acuse
en la auditoría común, incluida la recuperación de una respuesta perdida.
No se añade un decorador sin montaje ni se atribuye esa cobertura a #582.

## Siguiente lectura de Bolsa sin acción SQL nueva

Las consultas de contactos de participación y de oferta están montadas en
[`bolsa_borrador_llamamiento_desarrollo.go`](../../internal/app/bootstrap/bolsa_borrador_llamamiento_desarrollo.go)
y expuestas por
[`HandlerContactoParticipacion`](../../internal/modules/bolsa/adapters/httpinterno/contacto_participacion.go).
[`ServicioContactoParticipacion`](../../internal/modules/bolsa/application/contacto_participacion.go)
usa la acción existente `bolsa.contacto_participacion.consultar`, finalidad
`consulta_contactos_participacion` y contexto nominal registrado.
[`RepositorioContactoParticipacionPostgreSQL.listar`](../../internal/modules/bolsa/adapters/postgres/contacto_participacion.go)
llama `listar_contactos_participacion_v2` y confirma su transacción antes de
devolver contactos. El éxito ya tiene consumo autorizado; sus errores y
denegaciones no pasan por `RegistradorIntentosAuditoria` en el servicio revisado.

El corte útil es registrar esos resultados fallidos con AD169 tras retornar
del lector, conservando el consumo favorable existente. Incluye ambas lecturas,
acuse validado antes de exponer referencias, indisponibilidad sin datos si
falla el registrador y proceso/canal del montaje. Los fallos de preparación o
anteriores al contexto verificado requieren su frontera propia. El montaje
comparte archivo con #591: coordinar su cesión después de esa PR y verificar
el resto de ramas abiertas antes de editar. No se ha implementado este corte
ni se ha ensayado su consumidor en esta revisión.

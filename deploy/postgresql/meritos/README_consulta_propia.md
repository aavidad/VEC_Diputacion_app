# RUM04-P1: consulta propia interna — WIP

Este corte prepara la consulta nominal de un hecho propio y su ficha actual. Está pendiente de revisión y de completar el orden causal de migraciones; no tiene GO de entrega ni acredita una instalación institucional.

## Auditoría común: continuación del 3 de octubre

Este apartado sustituye las instrucciones de auditoría e instalación de los cortes históricos de abajo. La lectura confirmada conserva el consumo V3 y su auditoría común en la misma transacción. El recibo devuelve la referencia real de esa entrada común, sin generar una referencia de auditoría local. Los errores y denegaciones se registran después del retorno del repositorio mediante `ports.RegistradorIntentosAuditoria` de #502, sin crear otra tabla de auditoría en Méritos.

El receptor obtiene primero el contexto acreditado. Un selector o JSON rechazado registra el intento sobre la referencia configurada del circuito, sin copiar entradas del cliente. Si ese registro falla responde 503 en lugar de anunciar un 400 confirmado.

La orden lleva identidad y perfil del contexto acreditado, acción, recurso opaco, finalidad, motivo de catálogo, correlación, proceso configurado y canal del vínculo autenticado. También se registra una emisión fallida antes de que exista concesión positiva. El registro usa un plazo independiente de la cancelación de la petición. Si falla o devuelve un acuse no ligado, el servicio responde 503 sin ficha ni recibo; una denegación registrada responde 403.

Se retiran el adaptador local de intentos, AUT29 y la función `registrar_intento_consulta_propia_v1` de la candidata Méritos000002. El rol propio nuevo sólo permite consultar; los intentos usan el rol y la configuración de la autoridad común. Las reservas históricas se conservan y no se reutilizan. No se ejecuta DOWN ni se modifica el clon anterior que ya tiene AUT29 instalado.

El ensayo requiere la auditoría común instalada (AD169 y sus contratos históricos de ContextoActor y sesiones), un `audit_login` segregado y la configuración privada `auditoria.proceso` y `auditoria.plazo_ms`. La configuración incluye dos motivos de catálogo distintos (`motivo_denegacion` y `motivo_error`) y `recurso_consulta_ref`. El validador común resuelve positivamente el motivo elegido antes de registrar. Un error compuesto con indisponibilidad o cancelación se clasifica como error técnico y responde 503. La fábrica conecta el adaptador común PostgreSQL y comprueba su preflight antes de montar el handler. No acepta un sustituto en memoria ni un registrador local.

Orden de los objetos nuevos en una copia causal compatible: rol lector → AD145 → Méritos000002. Los prerrequisitos concretos están en `deploy/principal/lista_sql_codexa_rum04.txt`; dirección valida la postimagen exacta antes de instalar. La declaración RUM01 conserva su contrato anterior y no forma parte de este cambio.

El producto corregido `ee36b00b8` tiene dos revisiones estáticas favorables. Las pruebas normales, race y vet de Méritos pasaron; las pruebas focales de los callbacks y la clasificación también pasaron. Gosec y Semgrep local no encontraron incidencias. #435 permanece en borrador.

En una copia nueva de H7 se instaló una vez la cadena causal con AD145, la candidata Méritos000002 que devuelve la auditoría común y los contratos de L (CA26/IS13/AD169). Se conservaron las huellas de 477 tablas anteriores; la única diferencia estructural fue la ampliación de `auditoria_consumo_v3`. La proyección de sus columnas anteriores conserva exactamente las filas y su huella. La provisión de claves del ejercicio confirmó el protocolo común existente; no creó permisos por petición.

El recorrido se detuvo al preparar la identidad: `registrar_sesion.filas`, esperado al menos 1, observado 0. La cuenta A del ejercicio anterior no existe en H7: esperado 1, observado 0. La cuenta B está activa y tiene un alias, pero el alias restaurado del clon anterior no coincide con la fuente actual. No se cambió un vínculo ni se inventó esa correspondencia. No hay sesión positiva, alta M1 nueva, HTTP 200, Chrome ni recuperación funcional demostrados en este corte.

La continuación necesita la fuente nominal vigente de Identidad/Sesiones para las mismas persona, cuenta y perfil del ejercicio, y el cotejo histórico de ContextoActor compatible con esa fuente. Se conserva un checkpoint frío y un acta privada sin secretos. Los servicios y temporales propios se retiran al cerrar; las fuentes y los respaldos privados quedan disponibles para dirección.

El cliente envía únicamente `hecho_ref`. El servidor revalida la sesión, reconstruye el contexto y obtiene la persona del vínculo autenticado. La acción fija es `meritos.hecho.consultar_propio`, la finalidad `consulta_hecho_propio` y la audiencia `vec_meritos.hecho.consultar_propio.v1`. El perfil admite solamente `hecho_actual` y `recibo_consulta`, con obligación de auditar. No exige la condición de empleado para leer el hecho de la persona autenticada.

La función PostgreSQL consume la autorización nominal y recupera la versión actual en una misma transacción serializable. Devuelve una ficha minimizada y un recibo de consulta nuevo después del commit. La ficha excluye las referencias de persona, declarante y actor revisor. Una referencia ajena y una ausente producen la misma respuesta de ausencia tras una autorización positiva. La lectura conserva la historia de negocio; solamente añade la constancia de consulta y la auditoría de autorización.

La provisión del perfil, las concesiones y las claves se realiza separadamente, con huella y CAS. Ninguno de esos valores se acepta en una petición HTTP. El rol técnico propio carece de LOGIN, herencia y privilegios elevados; el login de ejecución tiene una única membresía nominal y no obtiene acceso directo a las tablas.

La pantalla y el receptor loopback están preparados para el ensayo interno con identidades sintéticas y las autoridades reales del clon. El receptor no se monta en la raíz institucional. La superficie externa y la verificación de méritos continúan cerradas. Una ficha consultada no acredita el mérito ni sustituye una revisión o firma.

El receptor de ensayo fija el actor en su configuración privada y revalida cuenta, contexto y sesión mediante las autoridades PostgreSQL. El transporte de Chrome no observa un certificado de cliente ni autentica por mTLS. El recorrido acredita handler, interfaz y PostgreSQL reales en el clon; la frontera de autenticación del portal institucional queda pendiente. Un fallo 503 devuelve un JSON con código de error saneado, sin ficha ni recibo, no un cuerpo vacío.

## Estado comprobado

En un clon privado nuevo, las migraciones nuevas se instalaron una vez tras AD142 y Méritos000001. La consulta nominal real recuperó el hecho propio en versión 3 y una ausencia; la consulta de otra persona al mismo hecho devolvió ausencia. Tras reiniciar PostgreSQL y ejecutar otro proceso, la consulta propia volvió a recuperar la versión 3 y la ausencia. Las cuatro tablas de negocio conservaron sus huellas y sus recuentos: un hecho, tres versiones, tres operaciones y tres eventos de outbox. Los recibos de consulta pasaron de cuatro a seis por las dos nuevas lecturas.

Las pruebas focales de aplicación, adaptadores y HTTP pasaron en Go normal/race/vet, con análisis local de seguridad sin hallazgos. La interfaz pasó sus 22 pruebas con Node20 y los catálogos por idioma. Estas comprobaciones no sustituyen Chrome ni las revisiones independientes pendientes.

## Retoma del 2 de octubre

La continuación une el WIP `f25aac133ea44e54aa1edb88e883bcb09f634ccf` con `main@7225e8782`. El cambio respecto a esa base conserva los 31 archivos propios de RUM04. RUM03, AD142 y Méritos000001 coinciden con la fuente final de la PR #400, `770c288cf43a14f7026d77d13c4867107c1aef24`; no se vuelven a incluir como cambios nuevos.

Dos regresiones focales comprueban el rechazo después de emitir la autorización. El adaptador envía el material, recibe un rechazo SQL, revierte la transacción y devuelve una denegación sin ficha, recibo ni diagnóstico privado. Aplicación también conserva ese resultado vacío. Son pruebas con dobles de contrato: no prueban el rechazo en PostgreSQL real ni su auditoría durable.

La constancia de la emisión no acredita el rechazo posterior de SQL. La transacción rechazada revierte también cualquier consumo o constancia de consulta que hubiera iniciado. El servicio exige ahora el puerto común `AuditoriaIntentos`: después de retornar el repositorio registra el fallo con identidad del contexto validado, acción, finalidad, recurso, correlación y decisión emitida. Distingue denegación de resultado no confirmado y omite documentos, material de autorización y diagnósticos. El registro usa un contexto independiente con un límite de dos segundos; una cancelación de la petición no borra el intento.

La aplicación devuelve indisponibilidad y datos vacíos si esa auditoría falla. Las consultas obtenidas o ausentes no añaden otra auditoría fuera de su transacción SQL. El receptor de ensayo admite una autoridad inyectada y usa, mientras falta la autoridad común persistente, un cierre explícitamente indisponible para los fallos posteriores a la emisión. No registra en memoria ni simula PostgreSQL. El cierre durable de estos intentos sigue pendiente.

AUT29 y la extensión de Méritos000002 preparan ese registro persistente, todavía sin instalar ni ensayar. Autorización reconstruye identidad, perfil, recurso y motivo desde su concesión positiva histórica, acreditada con ContextoActor V2 al emitir. No lee tablas de otros propietarios ni revalida la vigencia actual: una revocación posterior puede explicar el fallo que se registra.

El registrador técnico tiene un pool separado y una única membresía nominal, sin lectura de tablas ni acceso a la consulta de hechos. Sólo envía decisión, correlación y resultado fijo. Méritos escribe en su tabla `auditoria_operacion`, sin otra tabla o autoridad, y conserva la misma referencia y fecha en un replay exacto. Un resultado diferente para el mismo intento se rechaza. La constancia identifica la concesión como positiva y el resultado del intento como observación del registrador confiable; no afirma una denegación del PDP ni demuestra por sí sola que SQL produjo ese rechazo.

En el ensamblaje de ensayo, `audit_login` conecta el adaptador real cuando dirección haya instalado y provisionado sus dependencias. Si se omite, continúa el cierre indisponible; no se sustituye la auditoría por memoria. Los guiones de AUT29 necesitan concesiones sintéticas reales y comprueban ACL, cruces de referencias, replay y roles mezclados. Terminan en rollback y no acreditan durabilidad ni reinicio.

No se puede convertir cualquier rechazo SQL en una consulta confirmada: la tabla actual exige un consumo válido y admite únicamente obtención o ausencia. Si SQL rechaza la firma o la ligadura de actor, el material recibido tampoco sirve como identidad validada para una auditoría nueva.

## Cierre ordenado del 2 de octubre, 18:00

El driver único instaló una vez roles propios → AUT29 → AD145 → Méritos000002, con cuatro códigos 0, sobre POST144 del clon. Las huellas reales de POST145 coincidieron con las propuestas y conservaron propietario, ACL y configuración. No se reaplicaron AD142 ni Méritos000001.

El fixture creó el hecho mediante `Servicio.Declarar`, con las autoridades y el repositorio reales. Resultado `confirmada`, recibo `recibo:7ba8fdcb-554e-4220-b46a-3d0cf1d483b1`, versión 1, estado `declarado`, fecha `2026-10-02T15:58:24.217223Z`. Fuente del helper `ebbcce4bb51e3ca1a82d56bb5376d4c3fc9f58eb`; no se insertó negocio directamente por SQL.

La consulta propia y la referencia ausente devolvieron HTTP 200/200, con `PASS` del driver real. Fuente `0ba3d791f9fdcb9530cd90b9ea88487bbe52ec39`; binario SHA256 `66dd264d961c678277269aa731289549af3f1eb9154de3e1481c523c5839b0dd`. El primer fallo quedó antes del PDP positivo, sin concesión ni consumo de lectura; tras renovar las sesiones por la API real se completaron ambos casos. La normalización privada de fechas UTC de `+00:00` a `Z` conservó los mismos instantes, SPKI y huella de configuración; no se cambiaron claves ni gobierno.

La orden de cierre detuvo los casos ajeno, rechazo SQL con PDP positivo y auditor segregado, 503 sin auditoría, replay, Chrome y reinicio. Estaban preparados y siguen sin acreditar. La PR #435 permanece en borrador. AD149 entró después en main: AD145 de esta fuente protege POST144 y necesita reanclaje a POST149 real y nuevo ensayo causal antes de entregarse sobre esa base. El núcleo instalado del clon se conserva; no se ejecuta DOWN ni se sustituye para adaptar la evidencia.

## Pendientes conservados del corte del 2 de octubre

- Conservar las reservas propias AD000145, AUT000029 y Méritos000002. La orden del 2 de octubre a las 13:50 permite avanzar por dependencias de objetos: AD144 se aplicó en el clon sobre POST142, sin AD143. Copias no es prerrequisito de RUM04. Para esta pieza, el orden de instalación es roles de consulta/registro de intentos → AUT29 → AD145 sobre POST144 → Méritos000002.
- Conservar la evidencia de AD145 sobre POST144 ya instalada en este clon. Para la entrega siguiente, obtener POST149 real, reanclar el delta con conservación exacta, repetir las dos revisiones afectadas y ensayarlo en una copia causal mínima. No relajar guardas ni reaplicar UP/DOWN de migraciones con historia.
- Completar el recorrido con Chrome del sistema y la revisión independiente de usabilidad.
- Obtener dos revisiones del hash exacto, incluida SQL, autorización y seguridad, y ejecutar la puerta de calidad de cierre correspondiente.
- Completar la auditoría común persistente de la denegación SQL posterior a la emisión y probarla después del rollback en el clon autorizado. Las regresiones focales de la retoma sólo acreditan propagación, cierre y ausencia de exposición.
- Preparar la PR funcional sólo después de resolver esos pendientes. La rama actual es WIP y RUM04 permanece en cola, no cerrado.

No se incluyen claves, configuraciones privadas, SQL de provisión ni datos del clon en esta entrega.

## Retoma del 3 de octubre

La rama se reconcilió sin conflictos con `origin/main@936aac665084b57b0d71adad15672e85e0efbb6a`. El commit `930c589ab3ddeaf585db6557acd843123ee546c2` reancla AD145 a POST161, la convergencia del núcleo posterior a POST155. La lista causal está en `deploy/principal/lista_sql_codexa_rum04.txt`: exige POST155, AD161 y Méritos000001 ya instalados, y sólo permite instalar los objetos nuevos confirmados ausentes. Este reanclaje todavía no acredita instalación ni recorrido real.

Los catálogos ES/EN de consulta se añadieron a ambos manifiestos. La revisión de usabilidad detectó que volver a la página desde la caché de navegación dejaba la ficha y el selector de idioma vacíos. El montaje vuelve ahora a obtener el contexto y sólo consulta después de una respuesta válida. Cancela la petición anterior, descarta su respuesta y mantiene la ficha vacía si la nueva revalidación se deniega.

Comprobaciones locales: Go 1.26.6 pasó las pruebas normales, race y vet de `internal/modules/meritos/...`; las 22 pruebas de consulta web previas al parche, las seis pruebas de montaje posteriores y las cuatro de manifiestos pasaron. Gosec en los paquetes afectados y Semgrep con reglas locales no encontraron incidencias. Estas pruebas usan dobles donde corresponde; el ensayo `TestConsultaIntegracionReal` se omite si no recibe su configuración privada.

En el clon nuevo sin hechos de Méritos, dirección permite una declaración sintética nueva mediante `Servicio.Declarar`, con otra referencia y otra clave. No se repite la operación M1 original ni se atribuye su recibo al clon nuevo. Quedan pendientes la instalación causal aprobada, las lecturas reales, el rechazo posterior al PDP con auditor segregado, el 503 sin auditoría, el replay del intento, Chrome ES/EN a 1440/390 y la recuperación tras reinicio. Cada lectura autorizada crea un recibo nuevo; el replay exacto corresponde al registro del intento.

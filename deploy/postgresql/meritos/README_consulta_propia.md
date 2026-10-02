# RUM04-P1: consulta propia interna — WIP

Este corte prepara la consulta nominal de un hecho propio y su ficha actual. Está pendiente de revisión y de completar el orden causal de migraciones; no tiene GO de entrega ni acredita una instalación institucional.

El cliente envía únicamente `hecho_ref`. El servidor revalida la sesión, reconstruye el contexto y obtiene la persona del vínculo autenticado. La acción fija es `meritos.hecho.consultar_propio`, la finalidad `consulta_hecho_propio` y la audiencia `vec_meritos.hecho.consultar_propio.v1`. El perfil admite solamente `hecho_actual` y `recibo_consulta`, con obligación de auditar. No exige la condición de empleado para leer el hecho de la persona autenticada.

La función PostgreSQL consume la autorización nominal y recupera la versión actual en una misma transacción serializable. Devuelve una ficha minimizada y un recibo de consulta nuevo después del commit. La ficha excluye las referencias de persona, declarante y actor revisor. Una referencia ajena y una ausente producen la misma respuesta de ausencia tras una autorización positiva. La lectura conserva la historia de negocio; solamente añade la constancia de consulta y la auditoría de autorización.

La provisión del perfil, las concesiones y las claves se realiza separadamente, con huella y CAS. Ninguno de esos valores se acepta en una petición HTTP. El rol técnico propio carece de LOGIN, herencia y privilegios elevados; el login de ejecución tiene una única membresía nominal y no obtiene acceso directo a las tablas.

La pantalla y el receptor loopback están preparados para el ensayo interno con identidades sintéticas y las autoridades reales del clon. El receptor no se monta en la raíz institucional. La superficie externa y la verificación de méritos continúan cerradas. Una ficha consultada no acredita el mérito ni sustituye una revisión o firma.

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

## TODO antes de cerrar el corte

- Conservar las reservas propias AD000145, AUT000029 y Méritos000002. El orden central pendiente es AD142 → AD143 (K) → AD144 (G) → AD145 (A). Para esta pieza, el orden de instalación es roles de consulta/registro de intentos → AUT29 → AD145 → Méritos000002, tras sus prerrequisitos.
- Actualizar la preimagen de AD145 cuando AD143 y AD144 estén publicadas y ensayar el conjunto en un clon nuevo. La candidata actual protege la preimagen posterior a AD142 y no acredita compatibilidad posterior a AD144. No relajar las guardas ni reaplicar UP/DOWN de migraciones con historia.
- Completar el recorrido con Chrome del sistema y la revisión independiente de usabilidad.
- Obtener dos revisiones del hash exacto, incluida SQL, autorización y seguridad, y ejecutar la puerta de calidad de cierre correspondiente.
- Completar la auditoría común persistente de la denegación SQL posterior a la emisión y probarla después del rollback en el clon autorizado. Las regresiones focales de la retoma sólo acreditan propagación, cierre y ausencia de exposición.
- Preparar la PR funcional sólo después de resolver esos pendientes. La rama actual es WIP y RUM04 permanece en cola, no cerrado.

No se incluyen claves, configuraciones privadas, SQL de provisión ni datos del clon en esta entrega.

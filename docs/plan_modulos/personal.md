# Personal: continuación acotada

Propuesta para consenso de Dirección, Astra y Claude. El módulo se detiene tras acordar este plan; solo se rematan las piezas ya casi terminadas. Ninguna tarea de la cola autoriza código nuevo por sí sola. La prioridad de Contratación temporal y la cola H6 B205/B206/B204/B209 siguen separadas.

## Punto de partida

Base del inventario: `0a62a3ea68e58fbf890885a2f59cc80343107e18`, consultada por Git; main puede avanzar después. «Mi ficha» consulta las relaciones y los servicios propios con el empleado canónico de ContextoActor y una concesión V3; RRHH dispone de lista, detalle y actos de relaciones, ocupaciones, situaciones y servicios con historia bitemporal. RPT publica organización propia. Documentos conserva su autoridad común. Es una consulta parcial: faltan una historia propia uniforme, documentos vinculados a hechos de Personal, la fuente canónica de datos personales y un cauce general de rectificación. La competencia y la fuente del empleado para Dietas (`AUT-27`) siguen pendientes de decisión.

Las piezas abiertas se verifican por su hash exacto antes de afirmar integración:

| Pieza | Estado comunicado y remate permitido |
| --- | --- |
| PR #298, `trabajo/codexb-personal-ficha-recuperacion-20261001` @ `8f9a978a7` | Implementada; dos revisiones sensibles del producto `ab3442f77` y 65 pruebas focales. El sucesor cambia solo dos fixtures de ausencia en el coordinador: 26/26 pruebas. Producto idéntico; calidad y CI pendientes. |
| PR #299, `trabajo/codexb-personal-traza-rrhh-20261001` @ `4cdb40d7f` | Implementada; dos `GO` sensibles, `GO` de UX y 61 pruebas focales. Producto revisado en `437cbfb18`, CSS corregido en `0ab7fff8a` y los mismos dos fixtures incorporados en el sucesor: 26/26. Sin solapamientos a 1440/390 px y con reflujo CSS al 200 %. No se ha acreditado zoom nativo. La secuencia local HTTP con fixture dio 503 → 200 al reintentar → 200 al actualizar → 403 tras revocación simulada; no acredita mTLS, PostgreSQL ni navegador con backend real. Calidad/CI y prueba de carrera siguen pendientes. |
| PR #300, `trabajo/codexb-personal-expediente-borrador-20261001` @ `3baa47910` | Solo JSON, comentarios de SQL 29 y apéndice; `GO` de preparación. Sin ensayo SQL ni activación. Comprobar CI y fusión documental sin atribuirle persistencia operativa. |

`Personal 000025` se reservó antes del borrador para la proyección histórica propia y el vínculo documental. Queda después de M3 y del contrato nominal acordado con D; no es una migración de competencia de Dietas. La reserva no aprueba la fuente, los permisos ni la migración. No editar el hito 6 ni activar SQL por este plan.

## Orden al reanudar

1. Verificar CI, hashes y estado de fusión de #298, #299 y #300; resolver solo sus remates concretos. Dirección integra en la rama canónica tras revisión del candidato final. Registrar por separado código fusionado, SQL ensayado/instalado y recorrido acreditado.
2. Obtener de RRHH y Sistemas el contrato exacto de fuente y competencia de Personal, incluido `AUT-27` para Dietas, antes de otra implementación. Resolver las dudas existentes 27/28/29 (fuente, campos y roles), 34 (perfil) y 39 (sistemas corporativos) en su seguimiento propio. Una cuenta, certificado o cargo no acredita relación ni concede acceso. No duplicar identidad, RPT o Documentos.
3. Solo con ese contrato, encargar una pieza visible por PR. Cada pieza llevará un dueño de archivos exclusivo, consumidor real, autorización V3 por acción/ámbito/finalidad/campos, fuente y fechas expuestas, historia conservada y comprobación focal. Si toca SQL, reservar antes el número, ensayar en clon y obtener revisión SQL independiente y dos revisiones sensibles del hash final; si toca pantalla, aplicar las skills visuales y revisión independiente de usabilidad. Semgrep local sobre lo cambiado antes del PR.

## Garantías de todos los cortes

Una sesión actúa con un único perfil activo. Autorización conserva perfiles fijos y asignaciones nominales; su provisión usa huella y CAS, nunca permisos publicados por petición. Cada lectura consume la concesión vigente y registra la auditoría en la misma transacción. En escrituras, estado, versión, auditoría y outbox aplicable se confirman juntos; la historia y la auditoría son de solo adición. No se crea un outbox nuevo para una consulta. Personal se sirve solo en el proceso interno; Cronos exterior y cualquier carpeta agregada no amplían su exposición ni sus campos.

## Cola propuesta, sujeta a las decisiones anteriores

| Pieza visible y propietaria | Archivos orientativos exclusivos | Dependencia y comprobación proporcionada |
| --- | --- | --- |
| **Dependencia de Dietas: competencia acreditada.** Comprobar primero las consultas ya existentes de relación propia y competencias de Personal. Completar solo el contrato nominal que falte con D y G. Entrega coordinada: el consumidor real de Dietas muestra si la relación y la competencia son válidas o por qué no están acreditadas, sin copiar la ficha. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters}`. Dueño de integración posterior: puerto consumidor de Dietas; no editar ambos en un mismo encargo. | Fuente/competencia 27–29 y 39 y AUT27 de D; no reutilizar la reserva Personal000025 para esta dependencia. Probar dos relaciones, relación cesada, revocación, denegación y referencia ajena en aplicación/HTTP; PostgreSQL real solo tras migración autorizada. |
| **Datos de la persona en su ficha.** Mostrar los campos internos acordados de identidad y contacto por los puertos autorizados de Persona y Usuarios, con fuente y fecha; el registro de Personal conserva solo su vínculo laboral. No crear campos personales nuevos en tablas compartidas ni usar el perfil externo como fuente del interno. | Dueño Personal: proyección y vista propia; dueños Persona/Usuarios: sus contratos de lectura, en encargos dependientes. | Dudas 27/28/34 y fuente admitida. Probar minimización por perfil, dos identidades sintéticas, negativa cruzada entre procesos y retirada de datos al revocar. No repetir el alta de la persona ni inferir empleo de su certificado. |
| **Primer incremento lector: periodos propios y procedencia.** «Mi ficha» muestra una página de relaciones y servicios del titular, con fechas efectivas, corte de conocimiento, estado y fuente. Personal proyecta sus hechos con una autorización propia; no presta la consulta B2 de RRHH. No incluye la decisión de rectificación ni la historia administrativa de terceros. Una segunda pieza añade puestos y situaciones propios con su contrato específico. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters/postgres}` y sus vistas propias bajo `web/static/portal-empleado/modulos/personal/`. | Depende de Personal000025 y de campos/periodos autorizados. Probar corte efectivo frente al de conocimiento, varias relaciones, revocación entre páginas, hecho ajeno y fuente incompleta. Validar la lectura de una rectificación ya registrada, sin escribirla desde este corte. Ensayo SQL cuando exista candidata ejecutable y recorrido real de lectura; ninguna capacidad activa por el borrador. |
| **Documentos de Personal.** Desde un acto o periodo se lista y descarga el original o representación permitida, con versión, estado y procedencia. Documentos custodia el fichero; Personal conserva solo el vínculo `hecho–documento–versión`. | Dueño Personal: puerto y adaptador de vínculo en `internal/modules/personal/{ports,application,adapters}` y vista propia. Dueño Documentos, en encargo dependiente distinto, solo si su contrato actual necesita ampliación. | Depende de la terna y de la clasificación acordadas. Probar original frente a representación, versión sustituida, acceso propio/RRHH, denegación de terceros y descarga real; adjuntar no concede permiso. |
| **Solicitud general de rectificación.** El titular propone la corrección de un hecho de Personal con evidencia; RRHH competente decide y crea nueva versión. Reutiliza el circuito y recibos comunes de VEC. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters}` y pantalla propia; dueño del circuito común solo si aparece una carencia concreta, en tarea dependiente. | Depende de campos rectificables, decisor y fuente. Probar propuesta, denegación, decisión, idempotencia, versión concurrente y lectura de ambas versiones tras reinicio. No presentar la solicitud como corrección ya aprobada. |
| **Historia RRHH y competencias gobernadas.** RRHH consulta la secuencia completa y actúa solo con concesión central vigente, por unidad, relación, finalidad y campos. | Dueño Personal: proyección y pantalla RRHH en `internal/modules/personal/` y `web/static/portal-empleado/modulos/personal/`; Autorización gobierna la concesión en encargo propio si falta contrato. | Depende de 27–29 y 34, de la fuente vigente y de suplencias definidas. Probar actor competente, otro ámbito, revocación, delegación vencida y auditoría. No otorgar permiso por petición del navegador ni por nombre del cargo. |

Formación, Registro Único de Méritos, nómina, cotización, Cronos y Dietas conservan sus propios datos y planes. La carpeta personal podrá reunir consultas mínimas de esas autoridades cuando existan contratos y fuentes admitidos; no se abre aquí un banco paralelo de cursos, méritos o pagos. RPT conserva estructura, plazas, puestos y publicación. Personal conserva la reserva nominal vinculada a relación, ocupación y acto; RPT recibe solo la proyección mínima necesaria. No confundir una reserva nominal con una vacante disponible.

## Estimación

Horas de trabajo de un equipo Codex con varios subagentes, revisiones y CI, como trabajamos hoy. Cada bloque se divide en PR de 1–3 horas; no es una migración grande ni una espera continua de pruebas globales.

| Minitarea | Horas de equipo |
| --- | --- |
| Cerrar #298–300, incorporar únicamente sus correcciones y comprobar CI | 1–3 |
| Contrastar fuente, campos y vínculos existentes; acordar el contrato nominal con D | 2–4 |
| Coordinar competencia de Personal y consumidor Dietas sobre los puertos existentes | 3–5 |
| Datos internos autorizados de la persona y contacto, sin duplicar sus autoridades | 3–5 |
| Dominio y puerto de historia propia de relaciones/servicios, con cortes separados | 2–4 |
| Proyección paginada y consumidor durable propios; SQL nuevo, ensayo y dos revisiones | 5–8 |
| Cliente y vista de historia propia, sin selector de empleado | 3–5 |
| Extender la lectura propia de puestos/situaciones con versiones RPT admitidas | 3–5 |
| Vínculo hecho–terna documental y listado por el servicio común | 3–5 |
| Descarga original autorizada desde la ficha, con versión y estado documentales | 2–4 |
| Solicitud de rectificación y decisión RRHH por etapas, conservando ambas versiones | 5–8 |
| Consulta de historia RRHH y competencia fija por ámbito; reutilizar los perfiles comunes | 3–5 |
| Recorrido interno completo con fuentes admitidas, reinicio y manual de uso final | 3–5 |

Total del alcance de expediente aquí definido: **38–66 horas**, aproximadamente **5–9 días de un equipo** o **4–7 días con dos equipos**. Dos equipos pueden separar cliente/vistas de proyección/documentos; la cadena nominal y SQL sigue siendo secuencial. La cifra no incluye construir nómina, cotización, Formación o Carrera ni migrar todo el histórico corporativo.

No dependen del equipo: decisiones RRHH de las dudas 27–29/34/39, contrato nominal y orden SQL de D, interfaces y diccionario de Sistemas/RPT, acceso al servidor y validación de fuentes documentales. Si se entregan al iniciar cada bloque, reservar **2–5 días laborables adicionales** para intercambio y aceptación. Si no hay fecha de respuesta o la fuente no tiene interfaz utilizable, la espera es indeterminada y esta horquilla no fija una fecha de finalización. El ensayo SQL y el recorrido exigen los accesos acordados; una aprobación del plan no los sustituye.

## Consenso Astra

Ronda 1 de Astra: pidió actualizar las fuentes de Git, distinguir cambios de fixtures de cambios de producto, separar historia propia lectora de rectificación y consulta RRHH, coordinar la dependencia de Dietas con su consumidor y precisar la reserva nominal de Personal. El plan incorpora esos cambios. La provisión por huella y CAS pertenece a las asignaciones de Autorización; las lecturas consumen su concesión y auditoría de forma transaccional, sin añadir un outbox por consulta. Astra dio GO en la segunda ronda al alcance, orden y fronteras. Astra dio GO en la tercera ronda a la estimación y fronteras; una última precisión añade la lectura de datos internos de Persona/Usuarios para cubrir todo el encargo, sin duplicar datos ni usar el perfil externo. Estimación actual: 38–66 horas; 5–9 días con un equipo, 4–7 con dos; Astra confirmó también esta precisión y su estimación en la ronda final. Claude todavía no ha dado GO al plan.

Dirección deberá confirmar el orden y el primer encargo antes de reabrir código. El GO del plan no activa Personal000025 ni autoriza despliegue.

Fuentes: `AGENTS.md`; `ESPECIFICACIONES_AGENTES.md` E02–E08; `docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md` (PER, RPT y EMP); `docs/estudio_requisitos/analisis_integral_rrhh.md` §§15–22; `docs/estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md`. Los estados de PR proceden del encargo de Dirección y requieren comprobación en Git/CI.

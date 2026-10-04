# Lote ordinario de perfiles — AUT44, AD190 y CA35

El contrato v3 acepta entre uno y 32 cambios ordinarios de **una persona en
una unidad**. La organización la fija la configuración privada del servidor;
el navegador no la envía. Cada alta declara inicio `inmediato` sin fecha de
inicio, o `programado` con fecha futura. La baja no añade modo ni fecha nueva.
La fecha efectiva inmediata y la confirmación proceden del instante privado
único que AUT44 elige tras los bloqueos. Una alta programada conserva su fecha
exacta, que se vuelve a comprobar después de esperar por los bloqueos.

El canon de solicitud incluye orden, roles, persona, cuenta, organización,
unidad, modos, vigencias, preimágenes CAS, motivo y referencia de acto. Su SHA
entra como atributo del recurso V3 `persona`; la capacidad compromete la huella
del contexto canónico `{ambitos,atributos}` que calculan Go y AD190. Los
descriptores de organización y unidad proceden de recibos propietarios
privados, viajan por otro argumento y se cotejan con AUT37/Personal31. Nunca
conceden actor, rol, destino o fecha por sí mismos. Una tabla AUT inmutable
conserva sus versiones y SHA junto al canon y recibo comunes.

AUT44 verifica primero el actor y la decisión nominal Rol6, consume una
autorización V3 nueva en AD190 y, dentro de la misma transacción serializable,
comprueba las fuentes, todos los CAS y todas las clases ordinarias antes de
escribir. CA35 aplica las versiones de contexto con un instante común; AUT44
escribe asignaciones, historia, auditoría vinculada, recibo y un evento de
outbox `lote_perfiles_aplicado`. Una petición distinta con la misma referencia
choca; un replay idéntico exige otro acceso vigente, coteja **las fuentes
originales guardadas** y recupera el mismo recibo sin efectos adicionales.
Un COMMIT incierto no entrega recibo ni genera reintento automático.

El corte permanece **cerrado para instalación** mientras la revisión sensible
y el ensayo PostgreSQL del escritor único no acrediten hashes, ámbito,
fuentes y recibos con fixture real. La migración AUT44 conserva una parada
explícita de borrador. Tampoco está publicada la clave HMAC de la audiencia
`vec_autorizacion.administracion_perfiles.lote_ordinario.v1`: AD188 gobierna
solo las dos audiencias de lectura de usuarios. La publicación de esa clave y
el montaje HTTP pertenecen a cortes separados. Los actos sensibles de ADMIN
e Intervención siguen sujetos a propuesta y otra persona aprobadora. La
operación multiunidad o por centro queda fuera de esta versión.

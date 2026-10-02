# CRN14: enlace y revisión de justificación C8

Borrador bloqueado y no instalable. CRN14 está reservada por Dirección fuera de
Git. Esta pieza prepara el contrato y el adaptador de persistencia; no abre rutas.

Fuente C8: `5b51f06338fc4bdfcc1452d724770b1de782ddbc`, puertos, dominio y
aplicación de justificación. Requisitos: ficha Cronos C8 y seguridad Cronos
§§5, 9 y 10. CRN11, CRN12 y CRN13 conservan su alcance.

Contrato: [contrato_v1.json](contrato_v1.json), SHA256:
`1a1420481e2940621785becd4a656ceea6c70f4ba5936d1c41ff4c4f96b2c75f`.
El archivo `.sql.borrador` contiene una candidata SQL para ensayos PostgreSQL 18
desechables. No pertenece a una lista de instalación. Sus precondiciones exigen
las fachadas nominales de AD146 y el servicio documental DOC12/AD148. El ensayo
estructural con sustitutos sintéticos permite comprobar sintaxis y ACL; no
acredita autorización V3 real, la cadena causal ni una instalación en la principal.

Documentos confirma el registro administrativo de una referencia externa en
su propia transacción. El adaptador C8 transporta el documento devuelto por
ese servicio —identidad, número interno, fecha y política de conservación—
hasta el repositorio Cronos. Ese registro no acredita que el custodio externo
guarde el original, ni firma o entrega. Si falla el enlace, el registro de
Documentos permanece y se recupera con la misma clave y material, tras
autorización actual. El recibo Cronos aparece sólo al confirmar consumo V3,
versión, historia, auditoría y outbox juntos. La revisión conserva el permiso
concedido y el saldo. Su recuperación por la misma clave consulta el material
original con una autorización actual para el actor y perfil originales. Compara
decisión, motivo y versión solicitados antes de devolver el recibo histórico,
aunque después se haya anexado otro documento. No confirma otra revisión.

Faltan fuente Persona/Personal, enclave acreditado, gobierno nominal y
conservación aprobada. El consumidor V3 de las dos operaciones y sus
concesiones deben estar definidos antes de activar el enlace. El número
reservado no acredita esa autoridad ni fija el orden de instalación.

Antes de instalar una candidata quedan dos revisiones independientes
sobre su versión exacta y ensayo PostgreSQL real, con estos casos mínimos:

- Alta documental confirmada y fallo del enlace; recuperación sin duplicados.
- Confirmación devuelta por Documentos sobre ID, versión, huella, custodio,
  referencia, expediente y tipo; número, fecha y política originales.
- Misma clave con material cambiado, vínculo ajeno, catálogo o custodio distinto.
- Revocación, caducidad o pérdida de competencia antes del efecto y del replay.
- Dos revisiones concurrentes: una sola versión ganadora; rollback y reinicio.
- Recibo histórico idéntico, acceso auditado y permiso/saldo sin cambios.

Antes de presentar una migración instalable hay que ensayar la fuente exacta
con la postimagen causal de AD146/148 y DOC12 en el clon local de la principal,
y obtener dos revisiones sensibles. No se traslada a ese ensayo la evidencia
estructural sintética. Estos archivos no conceden autoridad ni contienen datos
personales reales o bytes documentales.

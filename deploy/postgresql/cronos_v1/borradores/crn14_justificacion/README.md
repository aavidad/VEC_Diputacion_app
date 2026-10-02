# CRN14: enlace y revisión de justificación C8

Borrador bloqueado y no instalable. CRN14 está reservada por Dirección fuera de
Git. Esta pieza conserva el contrato futuro; no añade SQL activo ni abre rutas.

Fuente C8: `5b51f06338fc4bdfcc1452d724770b1de782ddbc`, puertos, dominio y
aplicación de justificación. Requisitos: ficha Cronos C8 y seguridad Cronos
§§5, 9 y 10. CRN11, CRN12 y CRN13 conservan su alcance.

Contrato: [contrato_v1.json](contrato_v1.json), SHA256:
`1a1420481e2940621785becd4a656ceea6c70f4ba5936d1c41ff4c4f96b2c75f`.
El archivo `.sql.borrador` contiene exclusivamente comentarios.

Documentos confirma el registro administrativo de una referencia externa en
su propia transacción. El adaptador C8 transporta el documento devuelto por
ese servicio —identidad, número interno, fecha y política de conservación—
hasta el repositorio Cronos. Ese registro no acredita que el custodio externo
guarde el original, ni firma o entrega. Si falla el enlace, el registro de
Documentos permanece y se recupera con la misma clave y material, tras
autorización actual. El recibo Cronos aparece sólo al confirmar consumo V3,
versión, historia, auditoría y outbox juntos. La revisión conserva el permiso
concedido y el saldo.

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

Comprobación de esta pieza: JSON válido, gate cerrado, líneas SQL comentadas y
`git diff --check`. `ensayar-sql` no aplica al borrador no instalable; no se
acredita ensayo PostgreSQL. Estos archivos son datos y documentación, jamás
fuente de autoridad real. No contienen datos personales ni bytes documentales.

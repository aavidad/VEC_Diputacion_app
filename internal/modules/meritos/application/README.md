# Declaración y revisión de hechos

Estos casos de uso coordinan declaración, rechazo y rectificación mediante los tipos V3 y el emisor comunes. El servicio exige contexto registrado, vínculo de autenticación, acción, finalidad, campos y audiencia exactos. La huella liga actor, clave, contenido, versión esperada, motivo versionado y fecha de corte. Un reintento exige nueva autoridad y recupera el recibo original mediante el puerto de registro.

Declarar se limita al hecho propio. La rectificación conserva persona, declarante y origen; crea una versión pendiente sin la revisión anterior. El rechazo exige una persona revisora distinta del declarante y conserva el contenido rechazado. La operación transaccional consume V3, carga el antecedente confiable y aplica el cambio bajo CAS; no hay consulta de negocio previa fuera de esa transacción.

`Verificar` devuelve `meritos.error.acreditacion_pendiente` sin consultar ni modificar el registro. Los puertos comunes de Documentos conservan versiones y custodia; no acreditan correspondencia del hecho, fuente y competencia revisora. Esa comprobación positiva aún no tiene un circuito RUM acordado.

RUM03 aporta el adaptador PostgreSQL y la migración propia, dependientes del consumidor nominal común AD142. El adaptador coteja el resultado con estas reglas puras antes de COMMIT. Los rechazos de negocio confirmados contienen únicamente código y referencia de auditoría; un commit fallido o incierto nunca devuelve recibo ni denegación acreditada. La auditoría de rechazos previos usa el actor y la correlación nominales y omite los campos de negocio inválidos.

Las pruebas unitarias usan las fábricas de prueba comunes y dobles definidos solo en archivos `_test.go`. Comprueban cotejo de material, denegaciones, recuperación, separación de personas, aislamiento de recibos y preparación de cambios. Los ensayos PostgreSQL y su alcance se registran en `deploy/postgresql/meritos/README.md`; una prueba unitaria no acredita firma COSE ni una base instalada. El montaje HTTP, la consulta propia RUM04 y la acreditación siguen pendientes. Requisitos, admisión y puntos pertenecen a cada proceso.

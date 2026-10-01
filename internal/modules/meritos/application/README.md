# RUM02 en preparación

Estos casos de uso preparan declaración, rechazo y rectificación mediante los tipos V3 y el emisor comunes. El servicio exige contexto registrado, vínculo de autenticación, acción, finalidad, campos y audiencia exactos. La huella liga actor, clave, contenido, versión esperada, motivo versionado y fecha de corte. Un reintento exige nueva autoridad y recupera el recibo original mediante el puerto de registro.

Declarar se limita al hecho propio. La rectificación conserva persona, declarante y origen; crea una versión pendiente sin la revisión anterior. El rechazo exige una persona revisora distinta del declarante y conserva el contenido rechazado. La consulta previa y la confirmación deben mantener el CAS y los mismos datos de autoridad.

`Verificar` devuelve `meritos.error.acreditacion_pendiente` sin consultar ni modificar el registro. Los puertos comunes de Documentos conservan versiones y custodia; no acreditan correspondencia del hecho, fuente y competencia revisora. Esa comprobación positiva aún no tiene un circuito RUM acordado.

Esta entrega es WIP. No tiene consumidor nominal, adaptador SQL, permisos publicados ni montaje HTTP o CLI. Los códigos de acción, finalidad, audiencia y campos son candidatos pendientes de gobierno central. Los contratos del registro requieren consumo y revalidación V3, CAS, idempotencia, historia aditiva, auditoría, outbox y recibo en una transacción; RUM03 debe implementarlos y demostrarlos. Una firma de interfaz no acredita esa transacción.

Las pruebas unitarias usan las fábricas de prueba comunes y dobles definidos solo en archivos `_test.go`. Comprueban cotejo de material, denegaciones, recuperación, separación de personas y preparación de cambios. No prueban firma COSE, base de datos, recuperación tras reinicio ni acreditación de hechos. RUM02 permanece abierto hasta sus consumidores reales; requisitos, admisión y puntos siguen en cada proceso.

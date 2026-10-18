# Consulta del perfil propio de Bolsa

La fuente `bolsa_configuracion_rrhh_v1.json` incorpora una acción: `bolsa.configuracion.consultar`, sobre `perfil_rrhh_bolsa`, con finalidad `consulta_configuracion_bolsa_rrhh`. Las hojas exactas del perfil, las operaciones, los canales y la fecha de la consulta coinciden con el descriptor del servidor. Mantiene garantía alta, obligación de auditoría y los ámbitos `unidad_ref` y `ambito_ref` del soporte RRHH. El recurso procede de la sesión propia; la pantalla no permite elegir otra persona ni perfil.

Dirección incorpora esta fuente al catálogo completo, conservando sus fuentes y el censo de perfiles vigentes. La admite por el circuito existente de catálogo de acciones y publica con dos ADMIN la siguiente versión del mismo rol RRHH de Bolsa, conservando permisos y asignaciones anteriores mediante huella y CAS. Puede hacerlo junto a la acción del origen CT preparada por K. Este archivo no concede ni publica permisos por sí solo.

La ventana de la entrada reproduce la fuente RRHH B103 vigente. Cambiarla requiere otra versión y recalcular las huellas, sin editar una fuente ya publicada. No es un plazo de un expediente.

Tras publicar, dirección prueba `GET /api/vec/bolsa/bolsas?perfil=propio`. El servidor comprueba la sesión, el perfil, el rol, los ámbitos y la concesión exacta antes de entregar datos; la consulta tiene su asiento nominal común. Si devuelve 403, el registro técnico indica `bolsa_configuracion_denegada`, la clave que falló y los valores booleanos esperado/actual, sin nombres, documentos ni referencias personales. El GET general sin parámetros pertenece al cuadro de Bolsas; su autorización se comprueba aparte.

SQL nueva: no. CONFIG NUEVA: ninguna; se reutilizan el lector B103, la sesión de Bolsa y el registrador ya configurados. La publicación administrativa de esta entrada y de su concesión queda a cargo de dirección.

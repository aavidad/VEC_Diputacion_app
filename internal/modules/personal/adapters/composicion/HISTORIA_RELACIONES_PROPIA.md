# Historia propia de relaciones de servicio

Mi ficha permite abrir la historia desde Relaciones y consultar un intervalo de
fechas. Conserva todas las revisiones conocidas que solapan ese intervalo, con
sus efectos, fecha de registro, acto, fuente y versión. No deduce puestos o
situaciones anteriores de la ficha actual ni acredita documentos firmados.

La composición interna selecciona la capacidad mediante `historia_relaciones`
en el material privado ya existente de Personal. Este bloque fija motivo e
intentos de auditoría con proceso y canal confiables; no concede permisos.
Sin él, la ruta y el botón quedan ausentes. Si se solicita sin la fachada SQL o
sin su material nominal propio, el arranque falla.

Contrato nominal propio:

- Acción: `personal.registro_empleado.relaciones.historia_propia.consultar`.
- Audiencia: `vec_personal.registro_empleado.relaciones.historia_propia.v1`.
- Finalidad: `consultar_historia_relaciones_propias`.
- Entrada HTTP: solo `efectos_desde` y `efectos_hasta`; el intervalo excluye la
  fecha final. El servidor fija el corte de conocimiento y resuelve el empleado.
- Fachada pendiente: `vec_personal.consultar_historia_relaciones_propias_empleado_v1`.

La lectura requiere una concesión central positiva propia y confirma consumo y
auditoría común en su transacción. El repositorio valida toda la respuesta antes
de COMMIT. Los fallos se registran por el puerto común después de cerrar la
lectura; un reintento del registro conserva identidad, orden y referencia.
El límite es 200 revisiones, sin truncar: un exceso pide reducir el intervalo.

Este corte prepara código, cliente y vista. No instala ni activa la capacidad.
Personal35 y AD182 están reservadas para una PR SQL separada, tras la postimagen
común de K/L y provisión fija por huella y CAS. Faltan ensayo PostgreSQL, recorrido
nominal y recuperación tras reinicio. CA32, vínculos documentales y rectificación
general conservan sus dependencias propias.

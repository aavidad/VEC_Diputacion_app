# Lecturas de preferencias con campos cerrados

F1-13. AD296 prepara las dos lecturas positivas existentes de preferencias:
la externa y la interna. Cada una admite exactamente los ocho campos de Bolsas
o los cinco de modalidades. Mantiene la acción, finalidad, audiencia y permiso
propio de la persona candidata. La concesión de AUT104 puede contener diez
campos; la orden auditada contiene solo una de las dos proyecciones. No añade
un selector `tipo` ni publica una acción nueva.

## Activación

1. Instalar AD296 una sola vez, al final de la lista completa de main tras
   HZ37. Su guarda solo exige lo que dejan AD246, AD275 y AD294. No ejecutar
   un DOWN ni reaplicar una migración con historia.
2. AD296 exige las definiciones y los permisos exactos de las cuatro funciones
   de lectura e intento. Reconstruye solo las dos lecturas positivas. Los
   intentos conservan su cuerpo y `transaccion_origen` nulo. La lectura positiva
   sigue anotando la transacción real en la auditoría.
3. No instalar AUT104 sin B138 en la misma ventana. Las fachadas actuales de
   Bolsa devuelven siempre los ocho campos de Bolsas; con AD296 y AUT104 pero
   sin B138, una orden de modalidades quedaría auditada como lectura de
   modalidades y devolvería los datos de Bolsas. B138 liga la respuesta a la
   proyección pedida.
4. El comentario `vec.f1-13.preferencias.proyecciones-fijas.8-5.v1` en las
   cuatro funciones sirve para comprobar el orden de instalación de B138.
   Ese comentario no concede acceso; cada petición conserva el cotejo del
   perfil, recurso y ámbito actuales.

El focal `ad296_proyecciones_preferencias_fijas.sql` coteja cuerpo, propietario,
permisos, comentario y los dos arrays exactos con ROLLBACK. Se prepararon las
posthuellas de ambas lecturas en una transacción revertida de PostgreSQL 18:
externa `014ea7fbf8229619c736d0ebcea7964634dbd7c545b9b6a76d659d5a2a334bf8`;
interna `0a6ec140c5b7b4c52a2242269418397c527f446fc21c131af91a7cd0d728c5dc`.
La prueba estructural no finge una sesión, una decisión ni un recibo V3.

Ensayo del 19 de octubre, con main unido: una copia nueva instaló las 98 SQL
de la lista, con AD296 la última. Las huellas finales de las dos lecturas
coinciden con las de arriba, la focal de AD296 pasó y la aplicación respondió
en sus consultas públicas. Una revisión SQL independiente repitió la lista en
su propio PostgreSQL 18.4. Llamando a las fachadas reales, solo las proyecciones
de ocho y de cinco campos pasan el contrato. La mezcla, los campos de más, el
orden cambiado, los repetidos y la lista vacía se rechazan. Reaplicar AD296 se
detiene en su guarda sin efectos. La lectura de cinco campos sigue denegada
mientras no llegue AUT104.

Antes de activar las modalidades, la fila de auditoría debe decir por sí sola
qué proyección se leyó, con un motivo propio o una columna. Hoy solo se deduce
de la orden original conservada.
La lectura real de modalidades necesita AUT104, B138 y la API de preferencias
de modalidades con sus cinco campos. Esta entrega prepara las fachadas SQL,
pero no acredita un GET de modalidades ni el recorrido de esa API.

Vuelta atrás: recuperar el binario y la configuración anteriores; conservar
las migraciones y la historia. La activación de la API de modalidades y su recorrido siguen pendientes.

# Lecturas de preferencias con campos cerrados

F1-13. AD296 prepara las dos lecturas positivas existentes de preferencias:
la externa y la interna. Cada una admite exactamente los ocho campos de Bolsas
o los cinco de modalidades. Mantiene la acción, finalidad, audiencia y permiso
propio de la persona candidata. La concesión de AUT104 puede contener diez
campos; la orden auditada contiene solo una de las dos proyecciones. No añade
un selector `tipo` ni publica una acción nueva.

## Activación

1. Instalar AD296 una sola vez después de AD288, AD294, AD293 y AD295, y antes
   de AD291, según la lista completa de main tras HZ37 más las SQL propias.
   No ejecutar un DOWN ni reaplicar una migración con historia.
2. AD296 exige las definiciones y los permisos exactos de las cuatro funciones
   de lectura e intento. Reconstruye solo las dos lecturas positivas. Los
   intentos conservan su cuerpo y `transaccion_origen` nulo. La lectura positiva
   sigue anotando la transacción real en la auditoría.
3. El comentario `vec.f1-13.preferencias.proyecciones-fijas.8-5.v1` en las
   cuatro funciones sirve para comprobar el orden de instalación de B138.
   Ese comentario no concede acceso; cada petición conserva el cotejo del
   perfil, recurso y ámbito actuales.

El focal `ad296_proyecciones_preferencias_fijas.sql` coteja cuerpo, propietario,
permisos, comentario y los dos arrays exactos con ROLLBACK. Se prepararon las
posthuellas de ambas lecturas en una transacción revertida de PostgreSQL 18:
externa `014ea7fbf8229619c736d0ebcea7964634dbd7c545b9b6a76d659d5a2a334bf8`;
interna `0a6ec140c5b7b4c52a2242269418397c527f446fc21c131af91a7cd0d728c5dc`.
La prueba estructural no finge una sesión, una decisión ni un recibo V3.

La focal estructural de AD296 pasó con ROLLBACK en la copia anterior de 97
SQL. La candidata `601686060` instaló 98 SQL en dos copias nuevas y ambas
reiniciaron con la comprobación del guion; no se repitió aquella focal en 98.
La lectura real de modalidades necesita AUT104, B138 y la API de preferencias
de modalidades con sus cinco campos. Esta entrega prepara las fachadas SQL,
pero no acredita un GET de modalidades ni el recorrido de esa API.

Vuelta atrás: recuperar el binario y la configuración anteriores; conservar
las migraciones y la historia. La activación de la API de modalidades y su recorrido siguen pendientes.

# Intentos de consulta de preferencias de Mi Bolsa

F1-12 prepara AD294 para que el intento denegado de consulta externa quede
registrado sin atribuirle la transacción de una lectura autorizada. Conserva
el GET actual de preferencias de Mi Bolsa. La lectura positiva sigue usando
su transacción real; un intento no acredita acceso a los datos.

## Activación prevista

1. Usar una copia local de la principal con PostgreSQL 18 y datos ficticios.
   La lista única contiene main tras HZ37 y las SQL propias en su orden exacto:
   el bloque F1-12 (AD279, AUT96, B116 y AD275), después AD288, B128 y AD294.
   En una copia que ya recibió AD246 o AD275, cotejar su historia y no
   reaplicarlas. AD294 se instala una sola vez, sin lista parcial.
2. AD294 coteja la función instalada de AD246 antes de reconstruir solo
   `registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)`.
   Exige la huella previa
   `d72e98e0858d29535de28eb3ca06dcf166ac9a96c9c7827351499c2a73e49b75`
   y la posterior
   `819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27`.
   Si la preimagen, el propietario o los permisos difieren, la instalación
   se detiene. No se elimina historia ni se fabrica material de autorización.
3. Comprobar el mismo GET con un permiso vigente y con una denegación. El
   intento denegado debe dejar auditoría sin transacción de origen; la lectura
   positiva debe conservar su identificador de transacción. Comprobar además
   arranque con y sin la configuración existente y reinicio vigilado.

El cambio solo afecta a la inserción del intento nominal externo: la columna
`transaccion_origen` queda vacía y su restricción exige ese valor. No cambia
la función de lectura autorizada, el núcleo de autorización, audiencias,
permisos ni configuración. No se añade otra ruta ni otra pantalla.

## Evidencia pendiente

AD294 quedó instalada en las dos copias nuevas de `601686060`, con 98 SQL
en orden. Ambas arrancaron y reiniciaron con la comprobación pública del guion.
En la externa, un único GET real devolvió 403 `acceso_denegado` y una sola
fila de intento nominal denegado con `transaccion_origen` nulo. La prueba
no acredita acceso a los datos: falta publicar el rol externo y comprobar
el GET 200 y la transacción real de la lectura positiva. Vuelta atrás:
conservar copia y binario anteriores; si falla la preimagen, corregir la
causa y preparar otra candidata. No ejecutar DOWN ni reaplicar AD246 o AD275.

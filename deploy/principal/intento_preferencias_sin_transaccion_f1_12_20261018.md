# Intentos de consulta de preferencias de Mi Bolsa

F1-12 prepara AD294 para que el intento denegado de consulta externa quede
registrado sin atribuirle la transacción de una lectura autorizada. Conserva
el GET actual de preferencias de Mi Bolsa. La lectura positiva sigue usando
su transacción real; un intento no acredita acceso a los datos.

## Activación prevista

1. Usar una copia local de la principal con PostgreSQL 18 y datos ficticios.
   Dirección usa la lista única en este orden: el bloque F1-12
   (AD279, AUT96, B116 y AD275), AD294 y después AD288. En una copia que ya
   recibió AD246 o AD275, coteja su historia y no las reaplica. AD294 se
   instala una sola vez, sin lista parcial.
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

La prueba SQL de fuente y permisos está preparada, pero falta ejecutarla en
una copia nueva. También faltan el GET real autorizado y denegado, el arranque
con y sin configuración y el reinicio. Esta guía no acredita instalación ni
recorrido. Vuelta atrás: conservar la copia y el binario anteriores; si falla
la preimagen, corregir la causa y preparar otra candidata. No ejecutar DOWN
ni reaplicar AD246 o AD275.

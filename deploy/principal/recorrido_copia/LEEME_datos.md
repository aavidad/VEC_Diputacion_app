# Datos de una copia nueva

La fábrica llama a `preparar_roles(copia)` tras restaurar HZ37 y sus permisos,
antes de aplicar las migraciones. La función prepara los dos LOGIN de origen
que AD278 exige. Obtiene sus nombres de las filas restauradas, crea claves
nuevas con vigencia de 90 días y guarda el resultado en
`roles-previos.json` dentro de la copia. El SQL recibe las claves por la
entrada estándar; el registro y el fichero quedan en la carpeta privada.

Después de generar el material de identidad y crear las conexiones, la fábrica
llama a `preparar_datos(copia)`. Esta función sustituye los dos LOGIN
temporales por los de origen, conserva el resto de conexiones, prepara los
registradores de intentos y el manifiesto de Bolsa, y actualiza la huella del
entorno. En modo de material nuevo vacía los punteros de gobierno heredados
para que el arranque publique su propio gobierno. En modo `rrhh_original`
conserva el manifiesto original y aplica el ajuste de datos HZ37, protegido
por preimagen y limitado a 97 claves del emisor de desarrollo. Para ese modo
el conductor deja `sid_copia` en el estado: se coteja con PostgreSQL antes
de ejecutar el ajuste. Ambas rutas exigen la
marca del volcado HZ37.

La configuración de Documentos se prepara con
`preparar_documentos(copia, base)` cuando un productor de la misma copia
aporte cuentas y motivos completos. El generador actual de credenciales no
produce esos datos. El preparador documental no lee material de otra copia.

Un fallo durante estas etapas invalida la copia nueva: se descarta y se crea
otra. No se reintenta la escritura SQL sobre la misma instancia.

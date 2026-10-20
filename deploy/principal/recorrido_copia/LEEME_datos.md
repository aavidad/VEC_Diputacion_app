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
entorno. Solo admite material nuevo y la marca del volcado HZ37. Dentro de
la misma transacción comprueba el identificador PostgreSQL, la base y el
usuario antes de sustituir conexiones o vaciar punteros de gobierno heredados.
El arranque publica después el gobierno propio de la copia.

El resolutor de motivos de RRHH conserva una excepción heredada: su contrato
instalado exige un LOGIN sin fecha de caducidad. El ajuste afecta solo a la
conexión propia de ese resolutor, dentro de la transacción ligada a la copia.
Dirección retirará la excepción cuando el contrato admita una caducidad.
Los nuevos pools de Administración mantienen su límite de 90 días.

El recorrido prepara las cuentas, motivos y conexiones de Documentos en la
misma copia. No lee capacidades ni material de otra instancia.

Un fallo durante estas etapas invalida la copia nueva: se descarta y se crea
otra. No se reintenta la escritura SQL sobre la misma instancia.

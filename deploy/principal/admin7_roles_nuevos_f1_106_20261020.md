# Publicar roles nuevos con dos administradores (F1-106)

AD304 añade el conjunto 7 de claves: las diez del conjunto 6 y las dos del
gobierno de roles nuevos. Conserva las filas anteriores, sus permisos y el
catálogo de audiencias. El consumidor, AUT60 y las rutas de propuesta y
cierre ya existen; la publicación usa una entrada de un catálogo admitido.

## Activación

1. Instalar AD304 una vez, al final de `lista_sql_main_tras_hz37.txt`.
   Solo UP. No reaplicar AD198, AD220, AD248 ni AUT60.
2. Preparar y aplicar el gobierno técnico con `conjunto_capacidades: 7`,
   configuración aprobada por la huella del plan y material propios de la
   copia. Conservar el recibo y verificar la publicación. Los conjuntos
   anteriores siguen disponibles y no adquieren estas dos claves.
3. Preparar pools nominales distintos para `vec_admin_gobierno_roles_ejecutor`
   y `vec_admin_catalogo_acciones_lector`, y sus dos motivos publicados.
   Añadir el bloque `gobierno_roles` a la configuración privada y ejecutar
   la fase `gobierno-roles` con el acuse confirmado de aplicar. Produce
   `gobierno-roles.json` y dos archivos de clave 0600 separados.
4. Parar Administración, conservar su entorno anterior y montar
   `VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE` con el archivo exportado. Reiniciar.
   Sin esa configuración, Administración mantiene el resto de funciones y
   no ofrece este acto.
5. Seleccionar el perfil de Administración de la aplicación con el
   certificado propio de cada una de las dos personas administradoras.
   Consultar el catálogo vigente y elegir una entrada de perfil ordinario.
6. La primera persona presenta
   `POST /api/admin/perfiles/v1/gobierno-roles/propuestas`, con referencia y
   huella del catálogo y su entrada, identificador y nombre del rol nuevo,
   motivo y referencia de operación. El servidor construye el plan; la
   petición no entrega concesiones libres.
7. La segunda persona confirma
   `POST /api/admin/perfiles/v1/gobierno-roles/cierres`, con referencia y
   huella de la propuesta, motivo y operación propia. Conservar el recibo.
   Repetir o conciliar con las mismas referencias y el mismo contenido.
   La aprobación no asigna el perfil a ninguna persona.

CONFIG NUEVA: se reutiliza el selector existente
`VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE`. El nuevo bloque de la herramienta de
mantenimiento solo sirve para exportar su archivo privado; no se lee desde
el portal ni concede permisos al arrancar.

## Comprobación

En una copia propia con PostgreSQL 18 y la lista completa: probar AD304,
preparar el conjunto 7, publicar las claves y recuperar su recibo. Comprobar
arranque de Administración antes y después de montar roles. En Chrome,
propuesta y cierre con certificados de personas diferentes; el mismo
administrador debe tener denegado cerrar su propuesta. Recuperar el cierre
con el mismo recibo y verificar el rol publicado después de reiniciar.

El exportador rechaza un conjunto anterior sin esas dos audiencias, un acuse
no confirmado o ajeno al material y un destino ya ocupado. La prueba focal
comprueba esos límites del material y la conservación del conjunto 6. No se
usa una semilla de rol para acreditar el acto de dos administradores.

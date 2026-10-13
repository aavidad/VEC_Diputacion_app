# Preparar la confianza del portal externo

Este paso permite arrancar «Mi Bolsa» del aspirante con su autoridad propia.
Se ejecuta con la herramienta administrativa `vec-server preparar-portal-externo`,
antes de arrancar el proceso web. No entrega al proceso web credenciales de publicación.

Reutiliza la preparación y el inventario `externo/v3`, los descriptores de audiencias,
las funciones de AD140 y la cadena de auditoría común. AD263 añade una fachada
nominal limitada a publicar la confianza externa y consultar su estado. No cambia
los punteros internos ni concede permisos funcionales al aspirante. AD265 alinea
las listas de los dos lectores técnicos con los consumidores actuales, conservando
sus funciones, permisos y guardas.

## Orden

1. Conservar una copia de la base. Instalar HZ37 según la lista de dirección;
   las entradas `OMITIR` no se aplican. Después instalar únicamente la SQL nuevas
   de `lista_sql_codexp_confianza_externa_20261014.txt`. No repetir SQL instalada.
2. Preparar la identidad externa y su material con el kit externo-combinado
   existente. Mantener su idempotencia propia; no copiar claves internas.
3. Sistemas provisiona un LOGIN exclusivo para la persona que prepara la
   publicación. Este ejemplo identifica al operador Alberto; para otra persona
   se cambia el sufijo y se deja constancia en el registro de provisión:

   ```sql
   CREATE ROLE vec_confianza_externa_publicador_alberto
     LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
   GRANT vec_autorizacion_atestada_v3_publicador_externo
     TO vec_confianza_externa_publicador_alberto
     WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
   ```

   La contraseña SCRAM se fija por el mecanismo privado habitual, fuera de Git
   y de la salida del terminal. El LOGIN tiene **una sola membresía**, no recibe
   permisos directos sobre tablas ni pertenece al propietario AD3. El grupo
   publicador no pertenece a ningún otro grupo. El LOGIN no tiene `TEMP` en la
   base; si `PUBLIC` lo concede, Sistemas debe retirarlo conservando los permisos
   explícitos que necesiten otros procesos. No se retiran permisos a ciegas.
4. Guardar la conexión administrativa en un fichero privado `0600`, en un
   directorio `0700` fuera de los materiales montados por la aplicación. Exigir
   `sslmode=verify-full`, CA verificada y el LOGIN nominal anterior. No crear una
   variable `VEC_*` ni añadir esta conexión al entorno del servidor web.
5. Reconstruir el binario desde la PR integrada y usar el entorno interno privado
   ya preparado por dirección. En el host, ejecutar:

   ```sh
   vec-server preparar-portal-externo \
     --material-externo "$MATERIAL_EXTERNO" \
     --publicador-confianza-dsn "$DSN_PUBLICADOR_PRIVADO" \
     --consumidores mi_bolsa,portal_candidato,preferencias_bolsas,inscripcion_bolsa,meritos_cartera
   ```

   Los valores son rutas absolutas elegidas por dirección. El directorio externo
   tiene la marca de proceso externo y está separado del interno. Si el kit añade
   `--seudonimos` y `--cuentas`, conservarlos: se cotejan con la provisión existente.
   No se da de alta otra cuenta desde esta herramienta.
6. Se publican juntas las 23 audiencias externas actuales. El inventario sólo
   entrega las claves de los consumidores pedidos. Repetir el comando con el
   mismo material conserva raíz, configuración, claves y coordenadas; cada
   ejecución deja un asiento nominal. Si caduca la configuración o se renueva
   el material externo, se añade una versión y se conserva la historia. Renovar sólo la configuración
   conserva las coordenadas de las claves para que el portal ya arrancado pueda
   continuar; cambiar los secretos exige recomponer sus proveedores.
7. Cerrar el LOGIN administrativo cuando finalice la preparación (`ALTER ROLE
   vec_confianza_externa_publicador_alberto NOLOGIN`) y retirar su DSN privado
   según el procedimiento de dirección. Para renovar se habilita por Sistemas
   de forma controlada. El servidor sólo conserva sus conexiones de ejecución
   y preflight externo. El kit debe pasar el nuevo argumento también en su
   etapa `inscripcion`, que vuelve a preparar el material.

CONFIG NUEVA: fichero privado `0600` con el DSN del LOGIN
`vec_confianza_externa_publicador_alberto`, miembro exclusivo de
`vec_autorizacion_atestada_v3_publicador_externo`, pasado con
`--publicador-confianza-dsn` en cada preparación. Ninguna variable nueva para el
proceso web. Mantener `VEC_MATERIAL_PORTAL_EXTERNO_DIR` y las DSN externas del kit.

## Comprobación local

La prueba `TestPreparacionConfianzaExternaPostgreSQL` usa una copia desechable
post-HZ37/AD263/AD265 con TLS. Prepara el material, repite sin cambiar el inventario,
lee `leer_configuracion_externa_v1` y crea los proveedores V3 de cada consumidor
pedido. También publica una configuración de vigencia corta y comprueba que un
proveedor ya creado se recupera al renovar, sin reiniciar. Se activa sólo con `VEC_HZ38_PG_DESECHABLE=1`,
`VEC_HZ38_PUBLICADOR_DSN_FICHERO` y `VEC_HZ38_PREFLIGHT_DSN_FICHERO`, ambos ficheros
privados. No instala SQL ni crea LOGIN desde la prueba.

La preparación no mide una pantalla ni sustituye el recorrido de aspirante de
dirección. «Mi Bolsa», sus preferencias y la inscripción existentes son los
consumidores web de esta publicación.

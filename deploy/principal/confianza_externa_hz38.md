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

   Desactivar el registro de parámetros para el LOGIN antes de usarlo:

   ```sql
   ALTER ROLE vec_confianza_externa_publicador_alberto SET log_parameter_max_length=0;
   ALTER ROLE vec_confianza_externa_publicador_alberto SET log_parameter_max_length_on_error=0;
   ```

   La contraseña SCRAM se fija por el mecanismo privado habitual, fuera de Git
   y de la salida del terminal. El LOGIN tiene **una sola membresía**, no recibe
   permisos directos sobre tablas ni pertenece al propietario AD3. El grupo
   publicador no pertenece a ningún otro grupo. El LOGIN no tiene `TEMP` en la
   base; si `PUBLIC` lo concede, Sistemas debe retirarlo conservando los permisos
   explícitos que necesiten otros procesos. No se retiran permisos a ciegas.
4. Con `umask 077`, guardar la conexión administrativa mediante `install -m 600`
   en un fichero privado, en un
   directorio `0700` fuera de los materiales montados por la aplicación. Exigir
   `sslmode=verify-full`, CA verificada y el LOGIN nominal anterior. La contraseña
   vive en un `passfile` separado `0600`; el DSN incluye `passfile=/ruta/privada`,
   nunca `password=...`. No añadir esta conexión al entorno del servidor web.
5. Reconstruir el binario desde la PR integrada y usar el entorno interno privado
   ya preparado por dirección. En el host, ejecutar:

   ```sh
   vec-server preparar-portal-externo \
     --material-externo "$MATERIAL_EXTERNO" \
     --publicador-confianza-dsn "$DSN_PUBLICADOR_PRIVADO" \
     --consumidores mi_bolsa,portal_candidato,preferencias_bolsas,inscripcion_bolsa,meritos_cartera,bolsa_justificantes
   ```

   Los valores son rutas absolutas elegidas por dirección. El directorio externo
   tiene la marca de proceso externo y está separado del interno. Si el kit añade
   `--seudonimos` y `--cuentas`, conservarlos: se cotejan con la provisión existente.
   No se da de alta otra cuenta desde esta herramienta.
6. Se publican juntas las 28 audiencias externas actuales. El inventario sólo
   entrega las claves de los consumidores pedidos. Repetir el comando con el
   mismo material conserva raíz, configuración, claves y coordenadas; cada
   ejecución deja un asiento nominal. Si caduca la configuración o se renueva
   el material externo, se añade una versión y se conserva la historia. Renovar sólo la configuración
   conserva las coordenadas de las claves para que el portal ya arrancado pueda
   continuar; cambiar los secretos exige recomponer sus proveedores.
7. Instalar y activar la renovación administrativa automática descrita abajo.
   Mientras el temporizador esté activo, el LOGIN publicador permanece habilitado
   para ese proceso separado. Para retirar el portal o suspender la preparación,
   Sistemas detiene el temporizador, pasa el LOGIN a `NOLOGIN` y retira sus ficheros
   privados. No hace falta una intervención manual cada día. El kit debe pasar el
   nuevo argumento también en su etapa `inscripcion`.

## Renovación automática

Sistemas instala la tarea con la cuenta de servicio `openclaw`; la identidad de
la publicación sigue siendo el LOGIN nominal asignado al operador. Cada quince
minutos y al arrancar se ejecuta el mismo preparador, en un proceso administrativo
separado de la web. Si quedan más de treinta minutos, recupera la publicación;
cuando se acerca el vencimiento publica una configuración vigente durante las
siguientes 24 horas por defecto, conservando las claves y sus coordenadas. No espera al corte
de medianoche. Un fallo se registra, conserva lo anterior y se reintenta en el
siguiente disparo; nunca se presenta como renovación.

En desarrollo, la [configuración de vigencias de F1-64](vigencias_publicaciones_f1_64_20261016.md)
permite publicaciones nuevas de hasta 90 días con las tres llaves de desarrollo.
La renovación debe recibir el mismo valor de `VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS`;
una publicación vigente conserva sus fechas hasta que corresponda renovarla.

Ficheros: `confianza_externa_hz38/renovar.sh`, `vec-confianza-externa.service` y
`vec-confianza-externa.timer`. Como `openclaw`, con las rutas del paquete de HZ38:

```sh
umask 077
install -d -m 700 "$HOME/.config/vec/confianza-externa" "$HOME/.local/libexec/vec" "$HOME/.config/systemd/user"
install -m 755 "$BINARIO_HZ38" "$HOME/.local/libexec/vec/vec-server"
install -m 755 "$REPO/deploy/principal/confianza_externa_hz38/renovar.sh" "$HOME/.local/libexec/vec/renovar-confianza-externa.sh"
install -m 644 "$REPO/deploy/principal/confianza_externa_hz38/vec-confianza-externa.service" "$HOME/.config/systemd/user/"
install -m 644 "$REPO/deploy/principal/confianza_externa_hz38/vec-confianza-externa.timer" "$HOME/.config/systemd/user/"
install -m 600 "$ENTORNO_RENOVACION_PREPARADO" "$HOME/.config/vec/confianza-externa/renovacion.env"
systemctl --user daemon-reload
systemctl --user enable --now vec-confianza-externa.timer
systemctl --user start vec-confianza-externa.service
systemctl --user list-timers vec-confianza-externa.timer
journalctl --user -u vec-confianza-externa.service -n 20
```

Contenido de `renovacion.env` en cidonia (sin contraseñas):

```text
VEC_RENOVACION_BINARIO=/home/openclaw/.local/libexec/vec/vec-server
VEC_RENOVACION_MATERIAL_EXTERNO=/home/openclaw/.local/state/vec-desarrollo-20260906/config/portal-externo
VEC_RENOVACION_PUBLICADOR_DSN=/home/openclaw/.config/vec/confianza-externa/publicador.dsn
VEC_RENOVACION_CONSUMIDORES=mi_bolsa,portal_candidato,preferencias_bolsas,inscripcion_bolsa,meritos_cartera,bolsa_justificantes
VEC_PORTAL_PROCESO=interno
VEC_AUTH_MODE=desarrollo
VEC_EXECUTION_PROFILE=desarrollo
VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
VEC_DEVELOPMENT_MATERIAL_DIR=/home/openclaw/.local/state/vec-desarrollo-20260906/material/desarrollo-fiscalizacion-20260904
```

La DSN administrativa, su `passfile` y la CA deben usar rutas accesibles desde
este proceso del host, con TLS verificado y el nombre certificado de PostgreSQL.
No copiar la conexión del pod sin adaptar esas rutas. Sistemas deja habilitada
la permanencia de las unidades de usuario (`loginctl enable-linger openclaw`).
El temporizador no arranca la aplicación, no crea un perfil ni incorpora una
cuenta. Los alias se cotejan antes de la primera publicación; la renovación de
confianza no recibe `--seudonimos` ni permisos sobre la provisión de cuentas.

Comprobar `LastTrigger`/próximo disparo, salida correcta del servicio, expiración
futura del inventario y asientos nominales de publicación/recuperación. Ante
fallo, revisar el journal y corregir la conexión o el material; el siguiente
intervalo vuelve a intentarlo. Al cambiar el binario, actualizar también el
binario administrativo instalado. La retirada detiene primero el temporizador.

CONFIG NUEVA: fichero privado `0600` con el DSN del LOGIN
`vec_confianza_externa_publicador_alberto`, miembro exclusivo de
`vec_autorizacion_atestada_v3_publicador_externo`, pasado con
`--publicador-confianza-dsn` en cada preparación. Variables `VEC_RENOVACION_*` sólo en la tarea administrativa anterior; ninguna
variable nueva para el proceso web. Mantener `VEC_MATERIAL_PORTAL_EXTERNO_DIR` y las DSN externas del kit.

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

## Motivo documental y alcance de F1-46

El consumidor `bolsa_justificantes` entrega las tres claves propias de subida,
descarga y preparación del justificante de renuncia. Publicar confianza no
publica el permiso del aspirante ni el motivo documental. La configuración
privada exige `motivos_autorizacion_justificante_renuncia_bolsa`, con su versión,
huella y entrada exactas. AUT99 admite ese catálogo en la autoridad externa
únicamente para los actos propios del justificante; AD284 conserva la
confianza y la auditoría externas al consumirlos. Una entrada ausente,
retirada o divergente no autoriza la operación.

La [guía del justificante](justificante_renuncia_activacion.md) recoge los
permisos, el almacén y el antivirus que necesita la subida. Sus precondiciones
de gobierno no se sustituyen por esta publicación de confianza. Las admisiones
y asignaciones ficticias de la copia tampoco acreditan actos de Administración.

El [ensayo de F1-46](../../docs/estudio_requisitos/cuenta_externa_f1_46_20261020.md)
recoge el acceso externo a Mi Bolsa y su recuperación tras reiniciar.

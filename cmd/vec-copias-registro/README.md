# Registro externo de operaciones de copia

Este CLI guarda el progreso declarado de un ejercicio sintético CS07-A. Permite
reservar, aplicar un comando, consultar y listar operaciones. Conserva las claves,
el historial, los recibos y un registro local de accesos y rechazos semánticos
después de cerrar el proceso. Ninguna acción captura ni restaura datos.

El actor se declara en la entrada. El programa no autentica a esa persona ni le
concede permisos. Su auditoría local debe integrarse con la autoridad común y un
destino segregado antes del uso operativo. El estado `verificada_declarada`
mantiene pendiente la autenticidad de las evidencias. No declara una copia válida.

## Preparar un ejercicio

Compila `go build ./cmd/vec-copias-registro`. Copia `config.ejemplo.json` fuera de
Git, sustituye sus rutas por dos directorios sintéticos existentes y asigna
permisos `0700` al directorio de control. Es una configuración retirable; Sistemas
debe aprobar el destino externo y su custodia para uso real. No incluyas bases,
documentos, secretos o carpetas privadas en los datos de entrada.

Declara en `raices_restauradas` todas las raíces que una restauración podría
sustituir. El control se rechaza si queda dentro de alguna de ellas, resolviendo
también enlaces simbólicos. Esta comprobación de rutas no acredita volúmenes,
bind mounts o que el inventario de Sistemas esté completo. La cuenta del ejecutor
debe conservar el control y sus dos archivos fuera del rollback.

Ejemplo de reserva en un archivo `reserva.json` sintético:

```json
{
  "sintetica": true,
  "declaracion": {"actor_declarado": "actor:ensayo", "correlacion": "correlacion:ensayo"},
  "solicitud": {
    "operacion": "op:ensayo",
    "clave": "reserva:ensayo",
    "solicitud_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "conjunto": "conjunto:ensayo",
    "destino": "destino:ensayo",
    "politica": "politica:ensayo"
  }
}
```

La huella de este ejemplo es sintética. El consumidor real debe calcularla sobre
la solicitud canónica y conservar su procedencia. Ejecuta:

```bash
./vec-copias-registro -config config-privada.json -textos web/static/textos/es/copias_registro.json -idioma es -accion reservar < reserva.json
```

Para iniciar la captura declarada, usa `-accion aplicar` con esta entrada:

```json
{
  "sintetica": true,
  "declaracion": {"actor_declarado": "actor:ensayo", "correlacion": "correlacion:ensayo"},
  "operacion": "op:ensayo",
  "comando": {
    "clave": "comando:captura",
    "version_esperada": 0,
    "solicitud_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "accion": "iniciar_captura"
  }
}
```

La consulta `-accion consultar` conserva `sintetica`, `declaracion` y `operacion`,
sin `solicitud` ni `comando`. El listado `-accion listar` conserva `sintetica` y
`declaracion`, y añade `"consulta": {"limite": 25}`. Para la página siguiente
añade `despues` con la referencia de `siguiente`. El orden es ascendente por
operación y el cursor es exclusivo. La configuración admite de 1 a 100 elementos
por página; un límite cero usa el configurado. Para inglés usa su catálogo y
`-idioma en`. Todos los mensajes proceden del catálogo i18n común.

## Persistencia y recuperación

`operaciones.jsonl` contiene registros canónicos encadenados por SHA256. Cada
registro une solicitud o comando, evento de dominio cuando existe, recibo y
auditoría. `confirmaciones.sha256` conserva las marcas de confirmación por
adición. El ejecutor bloquea el diario con `flock` entre procesos, comprueba toda
su historia y sincroniza ambos archivos antes de devolver éxito. También
sincroniza la entrada del directorio. Los archivos son regulares, privados,
sin enlaces simbólicos ni enlaces duros. Se requiere Linux y almacenamiento
local con garantías de `flock` y `fsync`; NFS y destinos remotos no están acreditados.

La reserva exige unicidad de operación y clave. Repetir la misma solicitud o
comando devuelve el recibo original y añade otra auditoría; cambiar sus datos
produce conflicto. La versión esperada se compara bajo el mismo bloqueo. Una
reserva nueva se rechaza si el destino tiene otra operación en curso. Sólo los
estados terminales declarados liberan ese destino; liberarlo no acredita que la
copia sea válida. El replay de una reserva existente conserva el recibo incluso
cuando continúa en curso. Una
consulta conserva el recibo del último efecto y añade únicamente su acceso.
Entradas mal formadas se rechazan antes de escribir para no guardar rutas o
contenidos libres. Los rechazos semánticos de entradas válidas quedan auditados.

Tras reiniciar, consulta la misma operación o repite exactamente su clave y
datos. Un error de salida o sincronización deja un resultado incierto; nunca
inventes una clave nueva para superar ese error. Si hay escritura parcial,
huella distinta, truncado completo de un diario o marcas que no coinciden,
el programa devuelve `registro_historia_incompleta` sin modificar lo conservado.
No recorta ni reconstruye automáticamente una operación dudosa. Conserva los
dos archivos y concilia con la evidencia externa antes de una reparación
administrativa revisada. Esa reparación no se implementa en este CLI.

Las huellas detectan corrupción y truncado de uno de los archivos. No autentican
al escritor ni resisten la reescritura o sustitución coordinada de ambos archivos
por su propietario. La custodia, el almacenamiento protegido y la auditoría
central externos siguen pendientes de Sistemas. El diario tiene un límite de
64 MiB y cada registro de 16 KiB; agotarlo bloquea nuevas entradas. La rotación
debe conservar la historia y necesita un procedimiento posterior.

## Contrato para consumidores

El puerto `ports/registrocopias.Registro` expone `Reservar`, `Aplicar`, `Consultar`
y `Listar` con contexto, declaración de actor y DTO de dominio. El adaptador
`adapters/registrocopias.Abrir(Config)` recibe configuración explícita; no lee
variables de entorno ni toca PostgreSQL. La capa de aplicación conserva las
validaciones de referencias. Estos contratos no reciben una autorización
inventada. La integración HTTP y el ejecutor operativo deben utilizar identidad,
permisos y auditoría centrales vigentes, también durante mantenimiento.

El puerto separado `AceptadorOrden.AceptarOrden` recibe `RecepcionOrden` con
referencias de orden y operación, huella de orden, huella de solicitud, destino,
época y contador `Fence`. El consumidor CS08 debe verificar y revalidar la orden
con la autoridad común antes de cada llamada, incluido el replay. El registro
exige reserva previa y vínculos exactos, una orden por operación y contador
estrictamente creciente por destino. La época queda fijada por el primer dato
del proveedor confiable; un cambio se rechaza. Repetir la misma orden devuelve
su aceptación durable original, aunque el contador posterior haya avanzado.
La aceptación no modifica el progreso de copia, no ejecuta la orden ni consume
otra vez un permiso. Un reinicio no convierte esa aceptación en autorización.
Este puerto queda fuera del CLI sintético. La transacción SQL de autoridad y
su publicación exterior son responsabilidad del consumidor; aquí no se promete
una transacción atómica entre PostgreSQL y ficheros.

`AbandonadorCaptura.AbandonarCaptura` permite cerrar una captura fallida sin
inventar un manifiesto o ensayo. Recibe operación, clave, versión esperada,
huella de solicitud, destino y referencia/huella del fallo. Sólo se habilita al
componer `AbrirConObservadorAbandono` con un proveedor confiable. Ese proveedor
revalida el actor y la autoridad actuales y observa el efecto detenido, la
reserva de ejecución cancelada y la ausencia de efectos pendientes de escritura,
mantenimiento o restauración. Su observación se comprueba bajo el bloqueo del
diario y queda en el evento, con el recibo y la auditoría nominales.

Un resultado incierto, una reserva vigente o efectos pendientes impiden el
abandono y conservan ocupado el destino. El estado `abandonada_declarada` sólo
libera el destino después de esa comprobación; conserva todo el historial.
Una nueva operación puede reservar ese destino con su propia clave. Repetir el
abandono original exige revalidar la autoridad y devuelve el mismo recibo.
La CLI y `Aplicar` genérico no habilitan esta acción por datos declarados.
`Abrir` sin observador tampoco la autoriza. La comprobación real del ejecutor
y de su reserva corresponde a la composición operativa, no a este adaptador.

Desde `capturada` o `verificando`, el abandono exige además observar el
verificador propio `detenido` y la ventana `inactiva`. La fase procede del estado
conservado de CS07; no se elige en la solicitud. La referencia gobernada
`verificacion_fallida` exige esas mismas observaciones aunque CS07 siga en
`capturando`, si la publicación se adelantó a su confirmación. Una observación omitida, activa o
incierta mantiene ocupado el destino. Los campos nuevos son opcionales en la
serialización de una captura anterior para conservar sus bytes y su huella.
Si se observan como activos o inciertos, se rechaza el abandono en cualquier
fase y con cualquier referencia de fallo, incluida `captura_fallida`.

Las propuestas y aprobaciones de restauración pertenecen al contrato CS10.
Este registro de progreso CS07-A no las sustituye ni emite autorización FULL.

Pruebas focales:

```bash
GOMAXPROCS=8 GOCACHE=/dev/shm/go-build go test -race -p 8 ./internal/modules/administracion/{domain/operacionescopias,ports/registrocopias,application/registrocopias,adapters/registrocopias} ./cmd/vec-copias-registro
GOMAXPROCS=8 GOCACHE=/dev/shm/go-build go vet -p 8 ./internal/modules/administracion/{ports,application,adapters}/registrocopias ./cmd/vec-copias-registro
```

Cubren reapertura tras cada transición, recibos idénticos, rechazo de claves
alteradas, concurrencia CAS e interproceso, consultas/listado, truncado parcial y
completo, corrupción, exclusión del rollback y recorrido CLI castellano/inglés.
No acreditan restauración de plataforma, autorización ADMIN ni destino operativo.

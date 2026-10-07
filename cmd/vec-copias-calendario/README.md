# Calendario y retención de copias

Este consumidor permite guardar una política sintética, consultar su historial y
planificar próximas fechas y retención. Conserva la configuración en un directorio
externo privado. No copia, restaura, verifica ni borra contenido; no concede permisos ADMIN.

La política fija frecuencia, fecha inicial, días de la semana opcionales, zona horaria,
ventana, destino opaco y retención. Todos esos valores proceden del fichero de entrada.
Los números de los ejemplos sirven para el ensayo; no son plazos legales ni valores
aprobados por Sistemas. `doble_control` exige otra persona por defecto cuando se omite.

## Ensayo local

Compile una vez desde la raíz del repositorio:

```sh
go build -o /tmp/vec-copias-calendario ./cmd/vec-copias-calendario
```

Elija un directorio de control que quede fuera del conjunto que se restaurará.
Debe pertenecer a la cuenta operadora y tener permisos `0700`. La herramienta solo
crea el último directorio; sus padres deben existir. Use un directorio nuevo dedicado
a una sola política. No use una carpeta de copias o de documentos como registro.
Declare todas las raíces que se restaurarán con `--raiz-restaurada`, repetible.
El CLI rechaza un registro contenido en cualquiera de esas raíces, también con alias.
En el ejemplo, ambas carpetas vacías están separadas y representan un alcance sintético.

```sh
ambito_ensayo=$(mktemp -d)
registro_ensayo=$(mktemp -d)
chmod 700 "$registro_ensayo"
/tmp/vec-copias-calendario --registro "$registro_ensayo" --raiz-restaurada "$ambito_ensayo" \
  --textos web/static/textos/es/copias_calendario.json \
  < cmd/vec-copias-calendario/testdata/configurar.json
```

La respuesta contiene versión `1`, política normalizada, referencias declaradas e
instante, además del aviso de alcance sintético. `autorizacion_admin`, `habilita_copia`
y `habilita_borrado` son falsos. El fichero de ejemplo omite `doble_control`; la política
guardada lo devuelve activado.

Consulte el mismo registro desde una nueva ejecución:

```sh
printf '%s\n' '{"sintetica":true,"accion":"consultar"}' |
  /tmp/vec-copias-calendario --registro "$registro_ensayo" --raiz-restaurada "$ambito_ensayo" \
    --textos web/static/textos/es/copias_calendario.json
/tmp/vec-copias-calendario --registro "$registro_ensayo" --raiz-restaurada "$ambito_ensayo" \
  --textos web/static/textos/es/copias_calendario.json \
  < cmd/vec-copias-calendario/testdata/agenda.json
/tmp/vec-copias-calendario --registro "$registro_ensayo" --raiz-restaurada "$ambito_ensayo" \
  --textos web/static/textos/es/copias_calendario.json \
  < cmd/vec-copias-calendario/testdata/retencion.json
```

Para cambiar la configuración, prepare otra entrada `configurar` con la política
completa, la versión actual en `version_esperada` y un instante no anterior al último.
Si otra escritura cambió la versión, el cambio se rechaza. Consulte y revise antes
de reintentar. Un fallo después de guardar puede haber dejado la nueva versión;
consulte el historial antes de repetir. No hay opción de forzar o rebobinar el registro.

Para inglés seleccione `web/static/textos/en/copias_calendario.json`.
Los campos, estados y motivos del JSON son claves estables; los avisos y errores se
leen del catálogo. Las entradas rechazan campos desconocidos, repetidos o con otra
capitalización. Se limitan a 1 MiB y el proceso termina a los 30 segundos.

## Agenda y retención

`cada_dias` cuenta fechas civiles desde `fecha_inicial`, con valores entre 1 y 366.
`dias_semana`, si existe, restringe esas fechas: domingo es `0`, lunes `1`, sábado `6`.
La ventana empieza y termina el mismo día. No se aceptan ventanas nocturnas que
crucen medianoche. La agenda devuelve hasta 64 eventos dentro de 100 años de búsqueda.
Si una hora desaparece al cambiar a horario de verano, se omite esa fecha. Una hora
repetida en otoño tiene una única ejecución, en su primera aparición; el fin de la
ventana toma su última aparición si también se repite. Las fechas de salida son UTC.

La planificación de retención ordena las copias por fecha, conserva el mínimo de
copias verificadas y considera su edad en la zona configurada. Conserva copias previas
activas, protegidas, pendientes de conciliación o necesarias para otras copias.
Las no verificadas y las de otro destino no son candidatas. La última copia verificada
siempre se conserva. Todos los candidatos juntos mantienen el mínimo configurado.

El campo `candidata` propone una revisión. El plan exige autorización actual y el
control configurado antes de cualquier efecto. Los metadatos del ensayo son declarados:
`verificada: true` no acredita una restauración. El consumidor productivo debe obtener
el catálogo autenticado y revalidar sus datos al aplicar un plan; este CLI no lo aplica.

## Contratos para la composición

- `domain/politicacopias`: validación, normalización, agenda y selección de retención.
- `ports/politicacopias`: repositorio, reloj, autoridad, reserva y ejecutor inyectados.
- `application/politicacopias.Servicio`: configura con CAS, autoriza y revalida bajo
  exclusión del repositorio. Sin autoridad, deniega. La consulta autoriza cada versión
  y destino del historial. El cambio de retención exige dos personas si la política
  anterior o la nueva lo exige; no permite desactivar ese control unilateralmente.
- `adapters/politicacopias.Archivo`: diario Linux de una política, lock entre procesos,
  escritura temporal sincronizada, renombrado atómico, sincronización del directorio
  y lectura completa del historial. Cada versión enlaza la huella anterior. Un marcador
  independiente detecta la pérdida de la última versión completa. Una interrupción
  entre ambos registros bloquea nuevas escrituras para revisión, sin acortar historia.

La agenda utiliza una clave estable por política, destino y fecha civil. Cambiar la
versión durante ese día conserva la clave y exige conciliación si cambia la solicitud.
`Ejecutar` solo acepta la fecha programada, la versión actual y su ventana. Mantiene
la versión bloqueada, revalida antes de reservar y antes de ejecutar, y delega en
CS07 y el caso de uso real de copias. El ejecutor debe ser idempotente y conciliar
una reserva existente antes de repetir efectos. No debe volver a entrar en el
repositorio de política mientras mantiene su bloqueo.

## Límites de la entrega

El diario guarda historia y procedencia local. Su SHA256 detecta incoherencias;
no autentica el historial ni sustituye la auditoría central segregada. Su permanencia
fuera de una restauración depende del directorio elegido y la infraestructura.
El control del sistema operativo debe impedir manipular o borrar versiones históricas.
La CLI usa identidades declaradas únicamente para el ejercicio local.

Faltan la composición con permisos centrales ADMIN, consumición de autoridad y auditoría
externa, catálogo autenticado y ejecución real de captura/verificación. El puerto de
aviso y su adaptador al emisor común conservan el fallo de ejecución; distinguen
`fallido`, `no_configurado` y `emitido_sin_acuse`, que no acredita entrega. El dueño
de la operación debe conservar ese resultado en CS07 y conectar el aviso operativo.
`ReservadorCS07` conecta la reserva global y la exclusión por destino de CS07
sin duplicar su diario. Su prueba reabre el registro real de ficheros, conserva
una sola reserva y rechaza un destino ocupado o una solicitud alterada. Este corte no instala
servicios, permisos ni SQL; no acredita una copia válida, restauración o E2E ADMIN.

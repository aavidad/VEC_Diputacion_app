# Captura física fría local (CS05)

`vec-copias-fisica -config /sandbox/control.json` detiene los escritores
configurados, observa su exclusión, detiene PostgreSQL y captura PGDATA, los
almacenes y los componentes de la release. Reanuda PostgreSQL antes de devolver
el resultado y después reabre los escritores si observa PostgreSQL activo. CS04 puede usar el capturador
Go dentro de su propia ventana, sin ejecutar esta CLI ni duplicar su exclusión.

El resultado tiene estado `pendiente_cifrado_y_verificacion`. CS03 debe cifrar y
autenticar el conjunto; CS06 debe comprobar las dos restauraciones y el arranque
del binario archivado. Esta CLI no autoriza restauraciones ni publica una copia
válida. Solo se ha ensayado con datos sintéticos en recursos propios.

## Configuración privada

El archivo JSON requiere permisos `0600`, claves conocidas y un solo objeto.
No se guarda en Git. Todas las rutas son ejemplos que el operador sustituye por
su inventario autorizado. La configuración permite ejecutar herramientas de
plataforma: solo debe escribirla el responsable local de esas herramientas.

```json
{
  "captura": {
    "pgdata": "/sandbox/origen/pgdata",
    "destino": "/sandbox/destino",
    "fuentes": [
      {"id": "vec-server", "tipo": "binario", "ruta": "/sandbox/release/vec-server"},
      {"id": "descriptor", "tipo": "release", "ruta": "/sandbox/release/descriptor.json"},
      {"id": "configuracion", "tipo": "configuracion", "ruta": "/sandbox/configuracion.json"},
      {"id": "material", "tipo": "material", "ruta": "/sandbox/material.tar"},
      {"id": "almacen:documentos", "tipo": "ficheros", "ruta": "/sandbox/documentos"}
    ],
    "max_bytes": 268435456,
    "max_entradas": 10000,
    "tiempo_recuperacion_segundos": 30
  },
  "control": {
    "pgdata": "/sandbox/origen/pgdata",
    "tiempo_segundos": 30,
    "herramientas": {
      "escritores_detener": {"ejecutable": "/sandbox/tools/escritores-detener", "argumentos": []},
      "escritores_observar": {"ejecutable": "/sandbox/tools/escritores-observar", "argumentos": []},
      "escritores_reanudar": {"ejecutable": "/sandbox/tools/escritores-reanudar", "argumentos": []},
      "pg_detener": {"ejecutable": "/usr/lib/postgresql/18/bin/pg_ctl", "argumentos": ["-D", "/sandbox/origen/pgdata", "-m", "fast", "-w", "stop"]},
      "pg_estado": {"ejecutable": "/usr/lib/postgresql/18/bin/pg_ctl", "argumentos": ["-D", "/sandbox/origen/pgdata", "status"]},
      "pg_control": {"ejecutable": "/usr/lib/postgresql/18/bin/pg_controldata", "argumentos": ["/sandbox/origen/pgdata"]},
      "pg_reanudar": {"ejecutable": "/usr/lib/postgresql/18/bin/pg_ctl", "argumentos": ["-D", "/sandbox/origen/pgdata", "-l", "/sandbox/pg.log", "-w", "start"]}
    }
  },
  "inventario": {},
  "bloqueo": "/sandbox/control/captura.lock",
  "tiempo_segundos": 300
}
```

El ejemplo completo está en `testdata/config.sintetica.json`; sus rutas y
huellas son ficticias y también deben ajustarse a los archivos de ensayo.

Es un esquema de configuración: `inventario` debe contener el inventario CS02
completo. `fuentes` debe cubrir exactamente los identificadores de sus binarios,
componentes y almacenes. Añada también web, catálogos y los demás componentes
que figuren en ese inventario. Si archiva el descriptor de release, inclúyalo como
componente del inventario. Los componentes de release CS02 son archivos regulares
con SHA256 y tamaño comprobados; para varios archivos puede declararse un paquete
regular y, aparte, un almacén de directorio que capture su árbol instalado.

El destino existe previamente, tiene permisos `0700` y queda separado de todas
las fuentes. Las fuentes tampoco se solapan entre sí. El bloqueo está fuera de
las fuentes y del destino. El proceso necesita lectura de todo el conjunto y
privilegios mínimos para las herramientas de parada y arranque.

Los alias de escritores deben detener y drenar aplicaciones, consumidores,
migradores y editores que puedan cambiar base, configuración o almacenes.
`escritores_observar` devuelve `excluded` únicamente tras comprobar que esos
procesos ya no escriben. Un archivo de bandera o una afirmación del operador no
cumplen ese contrato. Su implementación depende del inventario de plataforma.

Las herramientas usan ejecutables absolutos sin enlaces simbólicos y sin permiso
de escritura para grupo u otros. No se admite un intérprete de shell como
herramienta. Los argumentos proceden solo de esta configuración privada; no hay
shell libre ni parámetros tomados de una copia. Los programas reciben únicamente
`LC_ALL=C`, `LANG=C` y un `PATH` fijo. No heredan secretos del entorno.

## Archivos y contrato Go

El directorio privado `captura-<aleatorio>` contiene `indice.json` y los archivos
que ese índice enumera. El índice todavía no está autenticado. No se usa por sí
solo como autoridad de integridad o autorización. Tanto índice como contenido
requieren la protección de CS03 antes de salir del espacio privado.

Cada fuente tiene un TAR `componente-NNNN.tar`: raíz `contenido`, seguida del
árbol cuando es un directorio. Conserva contenido, directorios vacíos, UID/GID,
modo y fecha de modificación. Rechaza enlaces simbólicos y archivos especiales.
Las huellas y tamaños son los del TAR completo. El PGDATA tiene identificador
`fisica:pgdata` y tipo `base_fisica`.

Cada componente regular de release tiene además `componente-NNNN.bin`, con sus
bytes y artefacto originales. Se extrae del TAR capturado y se comprueba otra vez
contra la huella y el tamaño del inventario. Así el manifiesto CS01 puede conservar
los IDs originales y distinguirlos de los TAR `fisica:<id>`. El índice establece
la correspondencia entre cada artefacto y su archivo; no deduzca esa relación de
la posición en la lista. `max_bytes` limita el contenido copiado, incluidas estas
copias regulares; el TAR añade sus propias cabeceras y relleno.

```go
capturador := &capturafisica.Capturador{Config: configuracion, Control: plataforma}
artefactos, err := capturador.Capturar(ctx, inventario)
```

El método satisface `CapturadorComponentes` de CS04. `Control` exige comprobación
de exclusión, parada, comprobación fría y reanudación. CS04 conserva su ventana
durante captura lógica, parada y captura física. El método reanuda PG para que
CS04 pueda hacer su observación final. Ante error elimina su salida parcial e
intenta reanudar PG con un plazo independiente de la cancelación original.

La comprobación fría exige `pg_ctl status` con código 3, `pg_controldata` en estado
`shut down` y ausencia de `postmaster.pid`. Se repite antes de reanudar. La captura
relee contenido y metadatos semánticos para detectar cambios. No usa ctime,
inodes ni xattrs para decidir equivalencia.

## Ensayo ejecutado y límites

Las pruebas focales con `-race` comprueban contenido, metadatos y recuperación;
rechazan huella distinta, symlinks, FIFO, tablespaces y límites de tamaño/entradas.
También se ejecutaron vet, build, gopls y gosec sobre estos paquetes.

El ensayo de CLI usó PostgreSQL 18.4 en un contenedor propio: sin red, raíz de
solo lectura, usuario no privilegiado, CPU 1, memoria 384 MiB, 96 procesos y SHM
propia de 64 MiB. Un escritor sintético quedó detenido y observado. La captura
reanudó PG y ese escritor. Una comparación de SHA256 de todos los archivos del
PGDATA, antes y después de copiar y antes de reanudar, confirmó fuente intacta.
El TAR frío tuvo 42 179 584 bytes y SHA256
`99c43578ec772b36339a198523ca1bdad7b1468ad1f2f5b8f340a31412b838e8`.

Se extrajo ese TAR a un PGDATA nuevo del ensayo y arrancó PG18.4 con otro socket.
La consulta recuperó `1|synthetic`, la secuencia conservó `23` y seguían presentes
NOT NULL, PRIMARY KEY y CHECK. Es una recuperación física sintética; no acredita
la restauración lógica, el arranque de VEC, las ACL completas ni la validez final
de una copia. Los recursos de ese ensayo se retiraron tras la comprobación.

V1 bloquea cualquier entrada en `pg_tblspc`, incluso si está declarada, y también
un `pg_wal` enlazado. No omite tablespaces ni otros enlaces para aparentar una
captura completa. Su soporte necesita un mapeo acordado con el restaurador.
No captura ACL extendidas ni garantiza recuperación de efectos externos. No
expone HTTP, modifica bases ajenas, cifra contenido ni sustituye fuentes.

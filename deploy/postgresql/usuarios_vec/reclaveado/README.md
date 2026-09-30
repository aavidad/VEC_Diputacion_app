# Reclavar los correos externos históricos

Esta herramienta convierte las direcciones cifradas del proceso combinado a las
claves propias del externo. Usa el adaptador criptográfico de Usuarios y conserva
la versión del sobre, la dirección, su estado y la versión del conjunto. Incluye
las direcciones retiradas. La herramienta se ejecuta fuera de los servidores;
el proceso externo recibe únicamente su material propio al arrancar.

Los HMAC de códigos y peticiones históricos quedan intactos como evidencia sellada
por clave primaria y huella de fila. Los desafíos pendientes se sustituyen. Una
petición con una clave histórica ocupada y otro HMAC conserva el conflicto P1409;
para iniciar una operación nueva hacen falta una clave nueva y la versión actual.
El sello de transición no permite confirmar códigos ni recuperar una operación
como si se hubiera recalculado su HMAC.

## Preparar el ensayo

1. Instalar las dependencias y Usuarios 000013 una sola vez en el clon local de
   la principal. La lista causal acompaña la migración. Conservar la preimagen.
2. Detener ambos procesos y comprobar sus conexiones PostgreSQL. La operación
   requiere una sesión de instalación superusuario. No concede permisos a los
   ejecutores ni añade una ruta HTTP.
3. Conciliar previamente los envíos reservados por su procedimiento nominal.
   La herramienta rechaza cualquier envío reservado o contexto abierto. No envía
   correo, ni afirma si un destinatario recibió un mensaje.
4. Preparar un plan privado con las personas externas que deben convertirse.
   Usar el inventario autoritativo del clon; no deducirlas de identificadores ni
   inventar referencias. Guardar ese plan y la conexión con permisos `0600`.

La conexión se lee desde un fichero privado. El plan autoriza expresamente
usuario, host, puerto, base y modo TLS. La excepción `sslmode=disable` exige
`clon_local=true` y host `127.0.0.1`. Fuera de esa excepción se exige
`verify-full`: la CA llega por otro descriptor y su SHA-256 debe coincidir con
el plan. Se verifica el nombre del servidor y se desactivan los destinos de
respaldo. Dirección debe comprobar el contenedor y los destinos del clon antes
del ensayo. No se admiten otras opciones libpq ni variables `PG*` que influyan
en pgx. La conexión y las claves no aparecen en argumentos ni mensajes.

Las dos semillas se leen de sus ficheros privados de 32 bytes mediante
descriptores heredados. La anterior es la semilla KMS del proceso combinado:
primero se deriva `vec.kms.desarrollo.envoltura.v1`, como en el proveedor actual.
La nueva es la semilla propia del proceso externo. No copiar las semillas a otro
fichero, registrarlas en terminal ni incorporarlas al repositorio.

## Inventario, aprobación y ejecución

El plan JSON usa estos campos:

```json
{
  "esquema": "usuarios.correos.reclaveado.plan.v1",
  "base": "nombre_del_clon",
  "sistema_postgresql": "identificador_del_cluster",
  "conexion": {
    "host": "127.0.0.1", "puerto": 55441, "usuario": "operador",
    "sslmode": "disable", "clon_local": true, "ca_sha256": ""
  },
  "conexion_sha256": "huella_de_configuracion",
  "lote_ref": "referencia:operacion:aprobada",
  "aprobacion_ref": "referencia:aprobacion:vigente",
  "personas": [
    {"persona_ref": "referencia_del_inventario", "preimagen_sha256": ""}
  ]
}
```

El ejemplo describe el formato: hay que sustituir las referencias por las que
consten en el inventario. No se acepta una misma persona dos veces. El plan tiene
un límite de 10 000 personas; cada persona se convierte en su propia transacción.

`sistema_postgresql` es el valor de `pg_control_system().system_identifier`,
obtenido al identificar el clon. Se compara antes de procesar personas. La huella
de conexión excluye la contraseña: es SHA-256 del JSON UTF-8 compacto con forma
`{"base":...,"conexion":...}` y claves en el orden del ejemplo. La aprobación
comprende ese destino, la versión del plan y las preimágenes individuales.

Compilar `./cmd/vec-reclavar-correos` con salida fuera del repositorio. Abrir los
cuatro ficheros privados en los descriptores 3, 4, 5 y 6: plan, conexión, semilla
anterior y semilla externa. Invocar el binario mediante:

```text
vec-reclavar-correos --modo inventario --plan-fd 3 --dsn-fd 4 --anterior-fd 5 --externa-fd 6
```

Para TLS, abrir también el fichero privado de la CA en otro descriptor y añadir
`--ca-fd 7`. El programa rechaza una CA sin la huella aprobada o un descriptor
que coincida con alguno de los otros cuatro.

`inventario` comprueba que cada dirección puede descifrarse, valida la huella de
igualdad anterior y prueba el nuevo sobre. Devuelve por persona la preimagen
SHA-256 y los recuentos de las siete tablas implicadas. No cambia filas. Guardar
la salida fuera de Git y trasladar cada huella a `preimagen_sha256` del plan.
Dirección revisa esa preimagen y deja su referencia de aprobación en el plan.

Ejecutar después el mismo comando con `--modo ensayo`: aplica el efecto completo
y devuelve el recibo que produciría, pero revierte cada transacción. Esa salida
es una simulación; no acredita un recibo instalado. Comprobar la conservación
exacta de las filas y repetir las comprobaciones de ACL y de preflight.

Solo después del ensayo revisado se usa `--modo aplicar`. La comparación de
preimagen impide convertir una población que cambió desde el inventario. La
salida incluye el recibo confirmado de cada persona. Conservar el plan, los
recibos y las huellas en el material privado de la entrega.

## Reanudar y cerrar

Si se pierde la salida o falla una persona, repetir `aplicar` con el mismo plan,
lote y aprobación. El programa consulta primero el recibo: recupera el original
sin otro nonce ni otro efecto. Si cambió la postimagen, detiene la recuperación;
hay que revisar la situación antes de volver a actuar. Las personas todavía
pendientes se convierten con su preimagen original. Una transición parcial deja
el preflight cerrado hasta cubrir toda la población histórica.

Los códigos de salida de error son claves de máquina: `reclaveado_entrada_invalida`
indica un descriptor, conexión o plan incompatible; `reclaveado_cripto_incompatible`
indica una dirección o huella que no se pudo validar; y
`reclaveado_operacion_no_confirmada` exige conservar el plan y comprobar el recibo
antes de repetir. Nunca interpretar un error como prueba de ausencia de efecto.

Al terminar, comprobar el preflight externo con su login propio, recuperar los
recibos y verificar que las filas internas no cambiaron. Retirar binario y
ficheros temporales propios, cerrar descriptores y destruir el clon de ensayo al
cerrar la entrega. El diario SQL se conserva. No ejecutar DOWN sobre esa historia.

La implementación borra los búferes y el material de las fuentes al terminar;
Go y el sistema operativo pueden conservar copias transitorias. Ejecutar en una
ventana controlada, sin trazas de depuración, volcados de memoria ni registro de
entradas privadas. Esta pieza no declara un recorrido de correo ni un despliegue
en la principal.

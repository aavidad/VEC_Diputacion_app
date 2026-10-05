# Frontera ADMIN: primer corte

`vec-admin` arranca un proceso separado con mTLS directo. Prepara la pantalla
`/admin/usuarios/` (también accesible por `/administracion-perfiles/`) y sus
lecturas nominales mediante la sesión ADMIN y el PDP V3. Las escrituras de
perfiles siguen cerradas. El selector reutiliza la observación de esa misma
frontera; no añade un login.

El modo heredado exige configuración privada y siete LOGIN segregados. Si falta una
función, un perfil vigente, material criptográfico o una fuente, el arranque o
la operación se cierran. No crea cuentas ni publica permisos al recibir una
petición. La CLI privada prepara o coteja el bootstrap; su aplicación sigue cerrada
hasta disponer del proveedor y el circuito admitidos.

El arranque requiere estas variables privadas:

| Variable | Contenido |
| --- | --- |
| `VEC_ADMIN_ENTORNO` | `desarrollo` (ver nota) |
| `VEC_ADMIN_ESCUCHA` | Dirección IP concreta y puerto; no admite `0.0.0.0` ni `::` |
| `VEC_ADMIN_HOST` | Nombre del subdominio ADMIN, o `nombre:puerto` si se publica en un puerto propio (ver nota) |
| `VEC_ADMIN_AUDIENCIA` | Audiencia propia de ADMIN |
| `VEC_ADMIN_EMISOR_IDENTIDAD` | El mismo valor que `identidad.espacio_identidad` |
| `VEC_ADMIN_TLS_CERT_FILE`, `VEC_ADMIN_TLS_KEY_FILE` | Certificado y clave del servidor |
| `VEC_ADMIN_CA_FILE` | Único certificado de la CA cliente ADMIN |
| `VEC_ADMIN_CRL_FILE` | Lista de revocación vigente, firmada por esa CA |
| `VEC_ADMIN_REDES_PERMITIDAS` | CIDR explícitas, separadas por comas |
| `VEC_ADMIN_RETIRADA_EN` | Fin de la excepción temporal, RFC 3339 en UTC |
| `VEC_ADMIN_PERFILES_CONFIG_FILE` | Archivo privado de configuración de perfiles |

Notas sobre estas variables, comprobadas en el ensayo del 5 de octubre de 2026:

- `VEC_ADMIN_ENTORNO` admite `cidonia`, pero IS15 sólo registra la política de
  acceso para `desarrollo` y el selector compara con este valor. Con `cidonia`
  todas las peticiones se deniegan hasta que exista una política para ese
  entorno.
- `VEC_ADMIN_HOST` fija el puerto público por el que entra el navegador, que
  no tiene por qué coincidir con el de `VEC_ADMIN_ESCUCHA`. Con `admin.ejemplo.es`
  (o `admin.ejemplo.es:443`) la cabecera `Host` debe ser `admin.ejemplo.es`
  exactamente. Con `admin.ejemplo.es:8444` debe ser `admin.ejemplo.es:8444`, y
  el `Origin` de las peticiones, `https://admin.ejemplo.es:8444`. Cualquier otra
  forma (sin puerto, otro puerto, `:443` explícito) recibe 403 o 401. El nombre
  va en minúsculas; si el valor no es un nombre DNS válido o el puerto no está
  entre 1 y 65535 sin ceros a la izquierda, el arranque falla.
- La política de certificado de IS15 (`host_admin`) guarda solo el nombre, sin
  puerto, y se compara con el nombre de `VEC_ADMIN_HOST`.
- El puente que lleve el puerto público hasta el proceso no puede terminar TLS:
  el certificado cliente tiene que llegar en el mismo handshake.
- El emisor de la aserción es el espacio de identidad de la sesión y el
  registro de sesiones lo compara con `identidad.espacio_identidad`. Si
  `VEC_ADMIN_EMISOR_IDENTIDAD` no coincide, el arranque falla con
  `etapa=emisor_identidad`.

Si el arranque falla, el registro dice `administracion: configuracion no valida: etapa=…`. La
etapa es un código fijo (por ejemplo `perfiles_config`, `pool_6_grupo`,
`lector_usuarios` o `servidor`) que señala el paso de la composición donde se
cerró. Nunca incluye rutas, valores de la configuración ni mensajes de
PostgreSQL.

La lista de revocación se lee en cada petición. Si falta, caduca o incluye
el certificado cliente, se deniega el acceso. El servidor solo admite TLS 1.3
y certificados de cliente verificados contra la CA ADMIN. Ignora cabeceras de
identidad y no comparte rutas con `vec-interno`.

La excepción de certificado como garantía HIGH solo está disponible en
`desarrollo` y `cidonia`, hasta la fecha indicada. En producción el proceso
falla al arrancar: faltan todavía el segundo factor Kerberos y la autoridad V3.
Antes de habilitarlo allí harán falta además rangos corporativos en la
aplicación y el proxy del subdominio.


La configuración de perfiles usa archivos `0600` en directorios propios `0700`
fuera de Git. Contiene rutas a siete DSN: fuente y registro de autorización,
motivos, registro y revalidación de sesiones, cuentas ADMIN y auditoría
de frontera. Los LOGIN son distintos y sus grupos y permisos se acreditan al
arrancar. Los DSN incluyen host, usuario, base, contraseña explícita y modo TLS;
se comprueban también los destinos alternativos y se desactiva `pgpass`.

La clave `pools.actos_admin` se conserva en el formato y puede ser una cadena
vacía: este proceso no lee su DSN ni abre ese pool. Si se aporta una ruta, se
comprueba su forma y que no repita otra. No abre una autoridad de actos.

El bloque `confianza` reúne cabecera, raíz pública, gobierno y once entradas
de capacidades. Cada clave HMAC llega por `material_archivo`; la semilla del
firmante y el material de seudónimos también se leen de archivos privados. No
se generan claves al arrancar. Las audiencias, vigencias, huellas y motivos son
valores expresos del aprovisionamiento. La raíz y el firmante deben coincidir.

La carpeta `activos_directorio` es un ensamblaje ADMIN privado, fuera de Git.
Se copian únicamente los archivos admitidos por la lista fija de
`internal/app/administracion/perfiles.go`, respetando sus rutas relativas. No
se incorporan fixtures, pruebas, otros portales ni material privado. Los
manifiestos públicos e internos comunes no se amplían para servir ADMIN.

Las fuentes `Lecturas` y `FuenteSeleccion` se inyectan por el compositor. La
segunda debe confirmar lectura o selección y auditoría común en una sola
transacción. En el modo heredado no hay una implementación conectada: la
ausencia conserva 503. Las fuentes no se sustituyen por datos de prueba ni
se habilitan desde un parámetro HTTP.

La composición y los activos están preparados para revisión. Un recorrido
real exige fuentes auditadas, instalación causal, configuración privada y
bootstrap aprobado por su canal de operador. Aquí no se ejecutan SQL,
bootstrap ni servidor. Una preparación, un GO estático o pruebas con dobles
no acreditan el recorrido.

## Modo privado de metadatos de usuarios

`VEC_ADMIN_USUARIOS_CONFIG_FILE` puede apuntar a un segundo archivo privado
`0600`, fuera de Git. Si se aporta, su campo `modo` debe ser exactamente
`metadatos_v1`: cualquier fallo cierra el arranque. El archivo heredado de
`VEC_ADMIN_PERFILES_CONFIG_FILE` sigue aportando identidad, firmante, pools
centrales y activos, con el mismo formato que antes.

El overlay fija organización y unidad, proceso y canal ADMIN, los dos motivos
de lectura, los motivos de denegación y error, un plazo acotado y los nueve
destinos de auditoría. La referencia del conjunto de usuarios debe corresponder
al ámbito privado. También señala tres archivos DSN distintos para el lector
AUT43/AD185, el registrador común AD169 y el selector IS14. Cada uno usa un
LOGIN propio y el proceso verifica su grupo antes de montar la fuente. AUT43
no concede `CONNECT` sobre la base al grupo `vec_admin_usuarios_lector`; hasta
que una migración lo haga, el DBA debe concederlo, porque `PUBLIC` no lo tiene
en la principal. Todos los pools fijan los límites de sesión que exige
VEC-AD-3 (10 s por sentencia, 2 s de bloqueo y 15 s de inactividad en
transacción). La
configuración `confianza` del overlay admite exactamente las dos audiencias
de usuarios; el material HMAC, la semilla de firma y los seudónimos siguen
procediendo de archivos protegidos.

Con ese modo, el proceso conecta el emisor V3, la fuente PostgreSQL nominal,
el mapper de metadatos y el auditor común. La API queda acotada a GET de lista
y ficha; las escrituras continúan cerradas. El selector conserva su circuito
auditado. Sin pool, permiso, clave o fuente real no hay lectura sustitutiva.
El montaje no acredita todavía un recorrido de navegador ni instalación SQL
en el entorno de destino. La frontera anterior a V2 sigue devolviendo
indisponibilidad hasta que disponga de una autoridad técnica propia.


## Configuración del runtime nominal

El arranque requiere también `VEC_ADMIN_RUNTIME_CONFIG_FILE`. Es un JSON privado
0600, dentro de un directorio propio 0700 y fuera de Git. Si falta o no pasa el
cotejo, el proceso termina sin recurrir a los constructores heredados. El formato
de `VEC_ADMIN_PERFILES_CONFIG_FILE` conserva sus campos.

| Campo | Contenido |
| --- | --- |
| `version` | `1`, formato de este fichero. |
| `pool_contexto` | Ruta del JSON de conexión del LOGIN exclusivo de CA36. |
| `fuente_identificadores_archivo` | Ruta del archivo de originales entregado por el productor de fuentes ADMIN. |
| `fuente_identificadores_sha256` | SHA256 aprobado de los bytes de ese archivo. |
| `proceso_contexto` | Proceso configurado para el contexto ADMIN. |

El pool de contexto tiene un LOGIN distinto del de cuentas, registro,
revalidación, selector, lecturas y auditoría. Los constructores acreditan las
fachadas de IS16 y CA36; no conceden acceso a sus tablas. La fuente privada
conserva SujetoID y las cuentas privilegiada y ordinaria originales, ligados a
Persona, certificados y fuente HMAC. No se obtiene desde nombres ni digests.

La composición usa esa fuente y el proveedor existente para resolver la cuenta.
El contexto utiliza su propio pool y transmite el vínculo de sesión auditado de
IS16. La selección del perfil propio sigue el circuito IS14/CA31. Instalar las
estructuras o aceptar esta configuración no acredita todavía una sesión
favorable, garantía alta, PDP ni recorrido HTTPS; el ensayo nominal está pendiente.

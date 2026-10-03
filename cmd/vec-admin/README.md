# Frontera ADMIN: primer corte

`vec-admin` arranca un proceso separado con mTLS directo. Monta la pantalla
`/administracion-perfiles/` y su API nominal mediante la sesión ADMIN, el PDP V3
y los proveedores PostgreSQL centrales. Cada operación revalida permiso y
ámbito. Las propuestas sensibles conservan el control de dos personas.

El proceso exige configuración privada y ocho LOGIN segregados. Si falta una
función, un perfil vigente, material criptográfico o una fuente, el arranque o
la operación se cierran. No crea cuentas ni publica permisos al recibir una
petición. El bootstrap se aplica aparte con la CLI privada existente.

El arranque requiere estas variables privadas:

| Variable | Contenido |
| --- | --- |
| `VEC_ADMIN_ENTORNO` | `desarrollo` o `cidonia` |
| `VEC_ADMIN_ESCUCHA` | Dirección IP y puerto exclusivos del proceso |
| `VEC_ADMIN_HOST` | Host HTTP exacto del subdominio ADMIN |
| `VEC_ADMIN_AUDIENCIA` | Audiencia propia de ADMIN |
| `VEC_ADMIN_EMISOR_IDENTIDAD` | Emisor propio de ADMIN |
| `VEC_ADMIN_TLS_CERT_FILE`, `VEC_ADMIN_TLS_KEY_FILE` | Certificado y clave del servidor |
| `VEC_ADMIN_CA_FILE` | Único certificado de la CA cliente ADMIN |
| `VEC_ADMIN_CRL_FILE` | Lista de revocación vigente, firmada por esa CA |
| `VEC_ADMIN_REDES_PERMITIDAS` | CIDR explícitas, separadas por comas |
| `VEC_ADMIN_RETIRADA_EN` | Fin de la excepción temporal, RFC 3339 en UTC |
| `VEC_ADMIN_PERFILES_CONFIG_FILE` | Archivo privado de configuración de perfiles |

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
fuera de Git. Contiene rutas a ocho DSN: fuente y registro de autorización,
motivos, registro y revalidación de sesiones, cuentas ADMIN, actos y auditoría
de frontera. Los LOGIN son distintos y sus grupos y permisos se acreditan al
arrancar. Los DSN incluyen host, usuario, base, contraseña explícita y modo TLS;
se comprueban también los destinos alternativos y se desactiva `pgpass`.

El bloque `confianza` reúne cabecera, raíz pública, gobierno y once entradas
de capacidades. Cada clave HMAC llega por `material_archivo`; la semilla del
firmante y el material de seudónimos también se leen de archivos privados. No
se generan claves al arrancar. Las audiencias, vigencias, huellas y motivos son
valores expresos del aprovisionamiento. La raíz y el firmante deben coincidir.

La pantalla puede localizar a una persona por su referencia canónica aunque
la fuente maestra todavía no aporte su nombre. Lo indica de forma explícita;
la referencia se muestra separada del nombre y no concede permiso. La búsqueda
por nombre permanece cerrada hasta disponer de esa fuente. Los ejemplos de
prueba conservan nombres sintéticos.

La composición Go y las pruebas focales están preparadas. El recorrido real
exige instalar y ensayar las fuentes causales de identidad, contexto, AUT24 y
AD150, publicar los motivos admitidos y aprovisionar dos administradores por el
canal privado. Un borrador SQL o un GO estático no acredita ese recorrido.

# Frontera ADMIN: primer corte

`vec-admin` arranca un proceso separado con mTLS directo. Prepara la pantalla
`/admin/usuarios/` (también accesible por `/administracion-perfiles/`) y sus
lecturas nominales mediante la sesión ADMIN y el PDP V3. Las escrituras de
perfiles siguen cerradas. El selector reutiliza la observación de esa misma
frontera; no añade un login.

El proceso exige configuración privada y siete LOGIN segregados. Si falta una
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
transacción. Todavía no se ha conectado una implementación en este proceso;
la ausencia conserva 503. Las fuentes no se sustituyen por datos de prueba ni
se habilitan desde un parámetro HTTP.

La composición y los activos están preparados para revisión. Un recorrido
real exige fuentes auditadas, instalación causal, configuración privada y
bootstrap aprobado por su canal de operador. Aquí no se ejecutan SQL,
bootstrap ni servidor. Una preparación, un GO estático o pruebas con dobles
no acreditan el recorrido.

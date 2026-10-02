# Frontera ADMIN: primer corte

`vec-admin` arranca un proceso separado con mTLS directo. Solo atiende
`GET /livez` (`204`). Las rutas de consulta o cambio de perfiles siguen
cerradas hasta que existan el rol fijo, la lectura V3 y el control de dos
administradores. El proceso no usa una cuenta de base de datos.

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

La lista de revocación se lee en cada petición. Si falta, caduca o incluye
el certificado cliente, se deniega el acceso. El servidor solo admite TLS 1.3
y certificados de cliente verificados contra la CA ADMIN. Ignora cabeceras de
identidad y no comparte rutas con `vec-interno`.

La excepción de certificado como garantía HIGH solo está disponible en
`desarrollo` y `cidonia`, hasta la fecha indicada. En producción el proceso
falla al arrancar: faltan todavía el segundo factor Kerberos y la autoridad V3.
Antes de habilitarlo allí harán falta además rangos corporativos en la
aplicación y el proxy del subdominio.

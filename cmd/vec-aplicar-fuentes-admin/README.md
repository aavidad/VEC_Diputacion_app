# Aplicar fuentes iniciales de ADMIN

Este comando invoca AUT39 desde un LOGIN técnico privado. Aplica las fuentes
sintéticas y la titularidad cuenta–Persona, sin conceder perfiles. El DBA prepara
por separado el LOGIN y su configuración aprobada. El comando no hace GRANT,
no escribe esa configuración y no aprueba su propio plan.

Primero prepare y coteje el documento con `vec-preparar-fuentes-admin`. El plan
contiene `plan` y `huella_plan_sha256`. Los bytes que recibe PostgreSQL son el
canon calculado por el dominio, sin salto de línea. El comando exige que esa
huella coincida con la del documento. La aprobación se lee de otro archivo
privado; se envía sin sustituirla por la huella calculada. AUT39 coteja la
aprobación, el LOGIN, la configuración privada y la preimagen reales.

La conexión privada tiene exactamente estos campos:

```json
{
  "dsn": "postgres://operador_sintetico@localhost/ensayo?host=/ruta/socket&sslmode=disable",
  "permitir_socket_desarrollo": true
}
```

El ejemplo usa un socket local de desarrollo autorizado expresamente. Para un
servidor remoto se exige TLS con verificación del nombre del servidor, también
en todos los destinos alternativos de la conexión. No se aceptan `role` ni
`options` como parámetros de sesión. Proteja las credenciales reales en el
archivo privado; no las ponga en argumentos, documentación ni registros.

La aprobación privada contiene exactamente `huella_plan_sha256`, emitida por
el operador que prepara la configuración externa. Conocer una huella no concede
por sí solo permiso: AUT39 la coteja con la aprobación fija del DBA.

```sh
GOCACHE=$HOME/.cache/go-build go build -p 8 \
  -o /ruta/privada/vec-aplicar-fuentes-admin ./cmd/vec-aplicar-fuentes-admin
/ruta/privada/vec-aplicar-fuentes-admin \
  --plan /ruta/privada/plan.json \
  --conexion /ruta/privada/conexion.json \
  --aprobacion /ruta/privada/aprobacion.json \
  --acuse /ruta/privada/acuse-primero.json \
  --textos /ruta/app/web/static/textos/es/admin-fuentes-aplicar.json \
  --timeout 30s
```

El timeout es obligatorio y positivo; el operador fija el límite de su sesión.
Las entradas privadas y el acuse usan rutas absolutas, archivos propios 0600 y
un directorio propio 0700 fuera de Git. Se rechazan enlaces, JSON con claves
repetidas, campos adicionales, campos ausentes y documentos de más de 64 KiB.

El comando reserva un archivo de acuse nuevo antes de conectar. No admite un
destino existente, ni siquiera cuando el plan es el mismo. Cada invocación tiene
su propio acuse porque la auditoría de su intento es nueva.

La operación usa una transacción SERIALIZABLE de escritura, UTC y sin SET ROLE.
Hace una única consulta parametrizada a
`vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(text,text)`. La envoltura
es cerrada: `estado`, `codigo`, `recibo`, `replay` y `auditoria_intento`. Se valida
su forma y el vínculo del recibo con el plan antes de COMMIT.

Los tres estados del servidor se confirman con COMMIT:

- `permitido`: efecto nuevo o recuperación, recibo y auditoría del intento;
- `denegado`: sin recibo, con auditoría del rechazo;
- `error`: sin recibo, con auditoría del error controlado.

Solo después de COMMIT se guarda la respuesta original completa en el acuse
privado. La salida de consola contiene estado y mensaje traducido; no vuelca
DSN, errores SQL, referencias de personas, material HMAC ni recibos completos.
Existen catálogos en castellano e inglés.

En un replay se conserva el recibo original de fuentes y se añade un intento de
auditoría distinto. Para recuperarlo use el mismo plan y aprobación con otro
archivo de acuse. El comando nunca reintenta automáticamente.

Si COMMIT no se confirma, el resultado se indica como `indeterminado`: no se
guarda ni se muestra un recibo como persistido. Conserve el plan y compruebe su
estado por el canal autorizado. Si COMMIT está confirmado pero falla el archivo
de acuse, el diagnóstico distingue ese caso para recuperar el recibo original.

Códigos de proceso: 0 operación permitida y acuse guardado; 1 entrada rechazada,
operación no confirmada antes de COMMIT o respuesta denegado/error confirmado;
2 catálogo o salida fallidos, COMMIT sin confirmar o acuse no guardado tras
COMMIT. Un código 1 de rechazo del servidor sí puede tener un acuse y auditoría
persistidos: consulte `confirmado` y `acuse_guardado` en el diagnóstico.

Dependencias: IS15, CA33, AD174 con auditoría de intentos y AUT39 con respuesta.
La preparación offline y las pruebas con dobles no acreditan instalación SQL.
Este comando no ofrece producción ni asegura auditoría de errores locales,
conexiones fallidas, cancelaciones o invocaciones que no alcanzan COMMIT.

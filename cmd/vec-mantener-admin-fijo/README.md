# Mantener el perfil fijo de Administración de Aplicación

La CLI invoca la operación técnica de mantenimiento con un plan y una aprobación
externos: AUT42 (plan versión 1, Rol4 a Rol5), AUT45 (versión 2, Rol5 a Rol6,
lote ordinario), AUT51 (versión 3, Rol6 a Rol7, gobierno del plan nominal de
firma), AUT59 (versión 4, Rol7 a Rol8, gobierno de definiciones) o AUT64
(versión 5, Rol8 a Rol9, versión de Bolsa). Actualiza los dos perfiles
existentes de Aplicación, con historia y
auditoría común. No solicita perfiles nuevos ni actúa por HTTP.

```sh
go run -p 8 ./cmd/vec-mantener-admin-fijo \
 --plan /ruta/privada/plan.json \
 --conexion /ruta/privada/conexion.json \
 --aprobacion /ruta/privada/aprobacion.json \
 --acuse /ruta/privada/acuse-nuevo.json \
 --timeout 30s --idioma es \
 --textos /ruta/repositorio/web/static/textos/es/admin-fijo-mantener.json
```

Los tres archivos de entrada son propios, `0600`, en directorio `0700` fuera de
Git y sin enlaces. Conexión: `dsn` y `permitir_socket_desarrollo` explícito.
El socket local sólo se admite cuando se autoriza desarrollo; TCP exige TLS
verificado, también en destinos alternativos. No hay SET ROLE ni opciones SQL
libres. Aprobación: `huella_plan_sha256`, SHA256 de los bytes exactos del plan.
La configuración positiva del DBA es independiente de ese archivo; el cliente
no publica una aprobación propia ni concede EXECUTE.

El acuse es nuevo por llamada y se reserva con O_EXCL antes de enviar. Incluye
los cinco campos del sobre AUT42: estado/código/recibo/replay/auditoria_intento.
Éxito, denegación y error gestionados se confirman por COMMIT antes de informar.
COMMIT incierto no se presenta como confirmado ni se reintenta automáticamente.
Un replay conserva el recibo original y tiene un intento común nuevo. No se
recupera un perfil revocado ni se reemplaza un acuse anterior.

Los mensajes están en catálogos ES/EN. Diagnósticos cerrados no incluyen DSN,
plan completo, nombres, claves ni error bruto del servidor. Esta CLI técnica
no acredita firma jurídica, despliegue ni cobertura de errores anteriores a
la invocación o cancelaciones externas.

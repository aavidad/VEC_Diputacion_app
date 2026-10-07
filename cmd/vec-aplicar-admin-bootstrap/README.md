# Aplicar el arranque 2+1 de Administración

Este comando invoca AUT40 desde un LOGIN técnico preparado por el DBA. Usa un
plan V3 ya preparado y cotejado por `vec-preparar-admin-bootstrap`, con cuentas,
Personas, certificados y fuentes de ámbito reales. No publica LOGIN, GRANT,
configuración ni aprobación propia.

```sh
GOCACHE=$HOME/.cache/go-build go build -p 8 \
  -o /ruta/privada/vec-aplicar-admin-bootstrap ./cmd/vec-aplicar-admin-bootstrap
/ruta/privada/vec-aplicar-admin-bootstrap \
  --plan /ruta/privada/plan.json \
  --conexion /ruta/privada/conexion.json \
  --aprobacion /ruta/privada/aprobacion.json \
  --acuse /ruta/privada/acuse-primero.json \
  --textos /ruta/app/web/static/textos/es/admin-bootstrap-aplicar.json \
  --timeout 30s
```

El plan contiene `plan` y `huella_plan_sha256`. El comando comprueba el canon y
esa huella. La aprobación separada contiene exactamente `huella_plan_sha256`:
se envía su valor externo sin sustituirlo por el cálculo local. AUT37/40 coteja
la configuración fija del LOGIN, las fuentes y la preimagen completa.

La conexión contiene `dsn` y `permitir_socket_desarrollo`. Un socket local solo
se acepta cuando ese permiso está activado; las conexiones remotas y sus
destinos alternativos requieren TLS con verificación del servidor. No se
aceptan parámetros `role` ni `options`. El timeout positivo es obligatorio.

Las entradas usan archivos privados propios 0600 en un directorio propio 0700,
con rutas absolutas fuera de Git. JSON cerrado, sin claves repetidas ni campos
adicionales. Se reserva un archivo de acuse nuevo antes de enviar la operación.
Un destino existente se rechaza antes de conectar.

La transacción es SERIALIZABLE de escritura y UTC, sin SET ROLE. La consulta es
única y parametrizada. La respuesta tiene cinco campos: estado, código, recibo,
replay y auditoría del intento. Se valida contra las dos Personas, las tres
referencias de perfil/vínculo y sus roles del plan. Se confirma COMMIT tanto en
permiso como en denegación o error; solo entonces se guarda la respuesta original
completa en el acuse privado.

Cada replay conserva el recibo original y crea otro intento nominal. Para
recuperar use el mismo plan y aprobación con otro acuse. No hay reintento
automático. Si COMMIT no se confirma, se informa `indeterminado` y no se guarda
ni muestra un recibo como persistido. Un fallo posterior del archivo distingue
COMMIT confirmado para permitir la recuperación posterior.

La consola solo muestra códigos y mensajes del catálogo, en castellano o inglés.
No vuelca DSN, errores SQL, datos de cuentas ni recibos completos. Sin catálogo
emite únicamente `catalogo_no_disponible` a stderr.

Salida de proceso: 0 permitido y acuse guardado; 1 rechazo/error confirmado o
fallo anterior a COMMIT; 2 catálogo/salida fallidos, COMMIT incierto o acuse no
guardado tras COMMIT. Consulte `confirmado` y `acuse_guardado` para distinguirlos.
Dependencias: AUT36/37/38/40 y AD179. La cobertura excluye fallos locales,
conexiones fallidas, cancelaciones y llamadas que no alcanzan COMMIT.

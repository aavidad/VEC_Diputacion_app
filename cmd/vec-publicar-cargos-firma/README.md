# Publicar los cargos de quien firma

La CLI prepara y aplica la operación de AUT53: publica la versión de rol de cada
cargo del plan nominal de firma y la registra como perfil asignable. Después,
Administración asigna el cargo a una persona con el lote ordinario. La CLI no
asigna cargos, no concede permisos, no crea el LOGIN ni la aprobación y no actúa
por HTTP. El detalle de las reglas está en
`deploy/postgresql/autorizacion/CARGOS_FIRMA_AUT53.md`.

## 1. Preparar el plan

```sh
go run -p 4 ./cmd/vec-publicar-cargos-firma preparar \
 --cargos /ruta/privada/cargos.json \
 --plan /ruta/privada/plan.json \
 --caduca 2h --idioma es \
 --textos /ruta/repositorio/web/static/textos/es/admin-cargos-firma.json
```

El fichero de cargos sale del plan nominal de firma (un cargo por `rol_id`):

```json
{"esquema":"vec.admin.cargos-firma.cargos.v1","cargos":[{
 "rol_id":"ct_direccion_rrhh","version":1,"nombre":"Jefatura de Servicio de RRHH",
 "version_anterior_sha256":"","operaciones_v2":["consultar_r5","firmar_vec"],
 "competencias":[{"accion":"contratacion_temporal.documento.firma_vec.registrar",
   "tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}],
 "regla_asignacion":"lote_ordinario","organizacion_ref":"organizacion:…",
 "vigente_desde":"…Z","vigente_hasta":"…Z","duracion_propuesta_segundos":86400}]}
```

`competencias` copia `accion_competencial`, `tipo_recurso` y `finalidad` de las
entradas del plan con ese `rol_id`. `operaciones_v2` puede ir vacío (ver el aviso
de ámbitos en el documento de AUT53). La CLI calcula la huella de cada versión de
rol con el dominio de Go, genera una `operacion_ref` aleatoria y escribe el plan
en un archivo privado nuevo, sin salto de línea final, con el mismo texto que
imprime PostgreSQL para `jsonb`. Muestra la huella SHA256 del plan, que es lo que
se aprueba. Caducidad máxima: un día.

## 2. Aprobar y configurar

Alberto aprueba esa huella. El DBA crea el LOGIN técnico, miembro único de
`vec_admin_cargos_firma_ejecutor`, e inserta su fila en
`vec_autorizacion.config_cargos_firma_admin_v1` (pasos en el documento de AUT53).

## 3. Aplicar

```sh
go run -p 4 ./cmd/vec-publicar-cargos-firma aplicar \
 --plan /ruta/privada/plan.json \
 --conexion /ruta/privada/conexion.json \
 --aprobacion /ruta/privada/aprobacion.json \
 --acuse /ruta/privada/acuse-nuevo.json \
 --timeout 30s --idioma es \
 --textos /ruta/repositorio/web/static/textos/es/admin-cargos-firma.json
```

Archivos de entrada propios, `0600`, en un directorio `0700` fuera de Git y sin
enlaces. Conexión: `dsn` y `permitir_socket_desarrollo`; por TCP exige TLS
verificado. Aprobación: `{"huella_plan_sha256":"…"}`. Antes de conectar, la CLI
comprueba que el plan es canónico, que sus huellas de rol son las que calcula Go
y que coincide con la aprobación. El acuse se reserva nuevo (O_EXCL) y se escribe
sólo tras COMMIT; un COMMIT dudoso se informa como `indeterminado` y no se
reintenta. Repetir con otro acuse devuelve el mismo recibo con `replay: true`.

Los mensajes están en los catálogos ES/EN `admin-cargos-firma.json`. Los
diagnósticos no incluyen DSN, plan, nombres ni el error del servidor.

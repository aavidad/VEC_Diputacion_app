# Catálogo de acciones de la copia local

`catalogo_acciones.py` prepara dos admisiones AUT58 independientes, después del
gobierno inicial de Administración y antes de publicar los roles de cargo de
firma. Primero admite `personal_organizacion_historica_rrhh_v1.json` con el modo
`fuentes`. Después sustituye `bolsa_origen_ct_rrhh_v1.json` por la versión 2 con
el modo `sustituir`. Cada preparación lee la cabeza y el censo actuales de la
copia. Conserva todas las demás fuentes, entradas y versiones de rol.

El conductor de la copia valida previamente la revisión, la copia de seguridad,
el SID, el destino y su candado. Crea un directorio privado `0700` distinto para
cada acto y exporta `VEC_COPIA_RAIZ`. El preparador usa `Copia` de
`copia_local.py` para comprobar la marca de propiedad y la etiqueta del
contenedor. Ejecuta únicamente las consultas de lectura declaradas, con una orden fija de
Docker y `psql`; no acepta ejecutable, host, base ni argumentos por el entorno.
Los valores de `--backup-ref`, `--backup-sha` y `--encargo-ref` deben
identificar la copia y el encargo técnico `F1-131`. `--sid-copia` es el
identificador de sistema de PostgreSQL de esa copia: entre 15 y 20 cifras y
distinto de cero. Los SHA de archivo que siguen
corresponden a las fuentes versionadas de este corte; el conductor los coteja
con su manifiesto antes de invocar el preparador.

```sh
python3 deploy/principal/admin_copia/catalogo_acciones.py preparar \
  --dir "$DIRECTORIO_ACTO" --fuentes deploy/principal/fuentes \
  --modo fuentes --acto 11 --encargo-ref F1-131 --sid-copia "$SID_COPIA" \
  --backup-ref "$BACKUP_REF" --backup-sha "$BACKUP_SHA" \
  --suelta personal_organizacion_historica_rrhh_v1.json:c04d3041921c03ec32d36970d479ad8b4ec729ac9161d79652b1afeba547dfe7
```

Para el segundo acto, se usan otro directorio, `--acto 12`, `--modo sustituir`
y `--suelta bolsa_origen_ct_rrhh_v2.json:239b0dd9dab236396f675247f344c91863c839ab56d2d068165bd45cd930bd9b`.
El segundo acto se prepara solo después de confirmar el recibo del primero.

`preparado.json` contiene `catalogo_sha256`, `paquete_sha256`, el SID, la
cabeza y el censo esperados y las referencias de copia y encargo. El conductor
aporta un archivo privado `0600` de aprobación técnica ligado a ese material:

```json
{
  "esquema": "vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1",
  "aprobacion_ref": "referencia-del-encargo-tecnico",
  "aprobador_ref": "fabrica:copias:desarrollo",
  "catalogo_sha256": "sha256-del-catalogo-preparado",
  "paquete_sha256": "sha256-del-paquete-preparado",
  "backup_ref": "referencia-de-la-copia",
  "backup_sha256": "sha256-de-la-copia",
  "encargo_ref": "F1-131",
  "sid_copia": "identificador-de-sistema-de-postgresql",
  "esperado_version": 1,
  "esperado_sha256": "sha256-de-la-cabeza-previa",
  "censo_preparado_sha256": "sha256-del-censo-preparado"
}
```

El preparador lee los bytes de esa aprobación y fija su SHA en el plan y en el
SQL de configuración. `fabrica:copias:desarrollo` identifica esta receta local;
no representa a una persona o actor V3, ni confiere permisos o acredita una
firma. El preparador rechaza otra referencia. La aprobación liga el material y la
preimagen antes de conocer la huella del plan final.

```sh
python3 deploy/principal/admin_copia/catalogo_acciones.py revisar \
  --dir "$DIRECTORIO_ACTO" --fuentes deploy/principal/fuentes \
  --huella "$CATALOGO_SHA"
python3 deploy/principal/admin_copia/catalogo_acciones.py configurar \
  --dir "$DIRECTORIO_ACTO" --fuentes deploy/principal/fuentes \
  --aprobacion "$APROBACION_EXTERNA" --pg-ca "$PG_CA" \
  --pg-puerto "$PG_PUERTO" --ttl 120
```

`configurar` vuelve a leer la cabeza, el censo y los archivos fuente. También
lee el SID de PostgreSQL en la copia validada y detiene la fase si ha cambiado.
Solo después produce
`plan.json`, `conexion.json`, `configurar.sql` y `configurado.json` con permisos
`0600`. `configurado.json` es el registro final, previo al efecto: guarda la
huella exacta del plan, SID, encargo, cabeza y censo preparados, aprobación y
referencia técnica de la fábrica. El DBA de la copia revisa y aplica el SQL en una transacción. Crea un
LOGIN exclusivo, miembro del grupo ejecutor sin permiso de cambio de rol, y
una fila protegida de `config_catalogo_acciones_admin_v1` ligada a la huella
exacta del plan, paquete y aprobación. No hay migración ni concesión de perfil.

La aplicación y recuperación se hacen con la CLI existente:

```sh
vec-catalogo-acciones-admitir --fase aplicar \
  --config "$DIRECTORIO_ACTO/conexion.json" \
  --plan "$DIRECTORIO_ACTO/plan.json" --sha "$PLAN_SHA" --idioma es
vec-catalogo-acciones-admitir --fase consultar \
  --config "$DIRECTORIO_ACTO/conexion.json" \
  --ref catalogo:bolsa:carga_convoca:b1 --version "$VERSION_NUEVA" \
  --catalogo-sha "$CATALOGO_SHA" --idioma es
```

Si se pierde la respuesta, primero se consulta la versión y huella exactas.
Un reintento usa los mismos bytes de `plan.json`, su SHA y `operacion_ref`.
`acuses --primero <acuse1.json> --replay <acuse2.json>` exige el mismo recibo y
que la segunda respuesta indique repetición. Los acuses son privados `0600`.

La admisión deja la acción disponible en el catálogo. La propuesta y el cierre
de la nueva versión del rol RRHH de Bolsa, sus pools y el overlay de gobierno
pertenecen a la etapa B103 posterior. Ninguna de esas operaciones la ejecuta
este preparador.

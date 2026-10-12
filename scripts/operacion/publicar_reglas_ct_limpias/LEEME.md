# Publicación de reglas de operación

Este corte prepara tres versiones nuevas a partir de las fuentes privadas usadas en el ejercicio: 36 reglas CT, 50 de Bolsa y 11 pasos del circuito de firma. Conserva claves, valores, plazos y referencias. La referencia `operacion:datos-inventados:20261012` documenta la orden de cargar datos inventados. No representa una aprobación de RRHH ni una firma.

El guion publica y activa **solo la base CT v2** por CT190/191. Bolsa v6 y circuito v2 son ficheros que consume el arranque; este corte no les atribuye una tabla SQL nueva. La publicación CT usa el preparador y el publicador ya existentes. Necesita CT148/190/191 instaladas y la cuenta técnica con acceso al propietario CT. Detenga la admisión durante la activación de una nueva base.

```bash
GOCACHE="$HOME/.cache/go-build" go build -p 6 \
  -o /var/tmp/vec-preparar-catalogo-reglas-ct \
  ./cmd/vec-preparar-catalogo-reglas-ct
export VEC_REGLAS_CT_PREPARADOR=/var/tmp/vec-preparar-catalogo-reglas-ct
# Configure libpq mediante PGHOST, PGPORT, PGDATABASE y PGUSER en su canal privado.
scripts/operacion/publicar_reglas_ct_limpias/publicar.sh --comprobar
scripts/operacion/publicar_reglas_ct_limpias/publicar.sh --aplicar PUBLICA-REGLAS-CT-LIMPIAS
```

`--comprobar` no crea ficheros ni inicia transacciones de escritura. Coteja las tres fuentes exactas y lee la activación CT y los ajustes en una transacción `READ ONLY`. `--aplicar` repite ese cotejo, prepara el canónico con el comando Go vigente, publica mediante `deploy/principal/publicar_catalogo_ct190.sh` y verifica la cabeza. Si se perdió la respuesta tras el `COMMIT`, repetir la misma orden recupera o completa la operación. Cualquier argumento adicional se rechaza.

Si CT148 contiene ajustes, el guion conserva sus valores en una **versión nueva** vinculada a la base v2; no altera ni borra las filas anteriores. Este rebase de datos de operación no pasa por `operar_ajustes_reglas_v1`, pues esa operación exige un cambio real de valor y una decisión V3. La fila nueva registra la referencia `operacion:datos-inventados:20261012`, la cuenta técnica, un recibo y un evento; no registra cambios de valor ni afirma una decisión V3 o aprobación RRHH. Si el publicador CT confirma el `COMMIT` pero falla el rebase, la misma orden completa el vínculo tras cotejar de nuevo ambas cabezas. Mantenga detenida la admisión hasta que `--comprobar` devuelva `replay` y coteje recibos e historia en el clon y destino.

Después de la publicación, copie los tres JSON junto con los demás catálogos de `/app/data/catalogos/` y cambie únicamente estas tres líneas del lanzador privado:

```bash
export VEC_BOLSA_REGLAS_SOURCE_PATH=/app/data/catalogos/bolsa/reglas.v6.json
export VEC_CT_REGLAS_SOURCE_PATH=/app/data/catalogos/contratacion-temporal/reglas.v2.json
export VEC_CT_CIRCUITO_FIRMA_SOURCE_PATH=/app/data/catalogos/contratacion-temporal/circuito_firma.v2.json
```

El arranque coteja la versión y la huella CT con la base activa. Una divergencia impide habilitar la edición de reglas. Los pasos del circuito describen datos configurados; su ejecución y una firma efectiva requieren el sistema corporativo y su autoridad propia.

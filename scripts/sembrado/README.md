# Sembrado de ejemplo de Peticiones de personal temporal

`sembrar_ejemplo_ct.py` crea o completa los expedientes de `casos_ejemplo_ct.json` usando solo la
API de VEC con los certificados mTLS de desarrollo (RRHH e Intervención). Exige la CA local y
comprueba el certificado y el nombre del servidor. No escribe en tablas.
Es idempotente: claves derivadas de `espacio_claves` y del código de cada caso, fechas fijas, y
reanudación a partir de los hitos del expediente.

```bash
python3 scripts/sembrado/sembrar_ejemplo_ct.py --base https://localhost:18443 \
  --material <material de desarrollo> --casos scripts/sembrado/casos_ejemplo_ct.json --plan
```

`--plan` no escribe y `--resumen` solo lee el cuadro de RRHH (fase y tramo de plazo).
Para sembrar en el entorno sintético, añada `--ejecutar --confirmar-entorno-sintetico` y
compruebe antes que `--base` apunta a ese entorno. La escritura solo acepta HTTPS en loopback;
también exige que la API publique `preparacion_vias.es_ejemplo=true`. Si falta la marca o
algún centro, categoría, contacto, grupo o modalidad no coincide con el catálogo, no empieza
el sembrado. Dentro del contenedor se puede pasar el JSON con `--casos-b64`.

Un túnel local puede apuntar a otro servidor: ni loopback ni `es_ejemplo`, que describe el
catálogo de vías, acreditan por sí solos que la base de datos sea sintética. Confirme el
destino antes de ejecutar. Si la API no publica esa marca, `--ejecutar` queda bloqueado;
este guion no ofrece un modo para saltarse la comprobación.

Límites del circuito actual: el análisis no mueve el expediente a «Análisis RRHH»; el llamamiento
no lo mueve a «Obtención del candidato»; la aceptación en Bolsa solo admite la categoría de
desarrollo C2, ausente del catálogo RPT, así que nombramiento, incorporación y seguimiento no se
alcanzan. Los plazos se cuentan desde la entrada en la fase: lo sembrado hoy queda en plazo.

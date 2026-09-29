# Sembrado de ejemplo de Peticiones de personal temporal

`sembrar_ejemplo_ct.py` crea o completa los expedientes de `casos_ejemplo_ct.json` usando solo la
API de VEC con los certificados mTLS de desarrollo (RRHH e Intervención). No escribe en tablas.
Es idempotente: claves derivadas de `espacio_claves` y del código de cada caso, fechas fijas, y
reanudación a partir de los hitos del expediente.

```bash
python3 scripts/sembrado/sembrar_ejemplo_ct.py --base https://127.0.0.1:18443 \
  --material <material de desarrollo> --casos scripts/sembrado/casos_ejemplo_ct.json --plan
```

`--plan` no escribe, `--ejecutar` siembra y `--resumen` solo lee el cuadro de RRHH (fase y tramo
de plazo). Dentro del contenedor de la aplicación se pasa por la entrada estándar con
`--casos-b64`.

Límites del circuito actual: el análisis no mueve el expediente a «Análisis RRHH»; el llamamiento
no lo mueve a «Obtención del candidato»; la aceptación en Bolsa solo admite la categoría de
desarrollo C2, ausente del catálogo RPT, así que nombramiento, incorporación y seguimiento no se
alcanzan. Los plazos se cuentan desde la entrada en la fase: lo sembrado hoy queda en plazo.

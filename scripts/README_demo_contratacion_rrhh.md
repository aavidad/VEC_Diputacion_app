# Datos de demostración de Contratación temporal

`demo_contratacion_rrhh.py` crea exclusivamente datos sintéticos a través de la API de VEC. No usa SQL ni admite identidad o permisos en cabeceras.

Primero se revisa el plan sin escribir, contra los proxies locales de la principal:

```bash
python3 scripts/demo_contratacion_rrhh.py \
  --url-base http://127.0.0.1:8082 \
  --solo-inventario --salida /tmp/vec-demo-c6-plan.json
```

El JSON enumera doce centros distintos, las seis categorías, periodos 2026--2027, jornadas de 100 %, 50 % y 75 %, y el objetivo de cada expediente. La ejecución requiere la revisión de dirección y `--ejecutar`; con HTTPS hay que aportar los dos pares de certificado y clave mTLS. El script conserva los recibos que devuelva la API en `--salida` y corta cada caso en la última operación confirmada.

La propuesta de nombramiento no se falsifica: necesita una aceptación de llamamiento vinculada y verificada por la cadena de comunicaciones. Si la API no aporta esas referencias, el caso queda documentado en llamamiento.

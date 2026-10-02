# Preparar un plan de formación

`go run ./cmd/vec-formacion-preparar < cmd/vec-formacion-preparar/ejemplo.json > /tmp/formacion-preparada.json`

La entrada contiene exclusivamente un escenario sintético de preparación.
Edite el JSON de ejemplo para cambiar necesidades, presupuesto, acciones y ediciones.
Las fechas, plazas, horas o presupuesto ausentes quedan pendientes de completar.
La salida JSON incluye el plan y una lista de comprobaciones pendientes.
Las fuentes oficiales se configuran en `internal/modules/formacion/adapters/jsonio/fuentes.json`.
La CLI no publica, inscribe, selecciona, certifica ni concede puntos o grados.

Para revisar el borrador en Chrome desde la raíz del repositorio:

```sh
python3 scripts/servir_preparacion_rrhh.py --modulo formacion
```

El comando muestra una dirección local. El visor permite filtrar, revisar pendientes
y descargar el borrador; no se publica en los portales ni modifica expedientes.

# Preparar hechos de RUM para un proceso selectivo

El CLI selecciona hechos de un paquete sintético RUM, conserva su versión actual
exacta y devuelve procedencia, estado, vigencia y evidencias por referencia.
La salida omite nombres, persona, denominaciones y actores de revisión.

Ejecute desde la raíz del repositorio con Go 1.26.6 disponible:

```sh
python3 - <<'PYTHON' | go run -p 8 ./cmd/vec-seleccion-hechos-preparar
import json
with open("data/ejemplos/meritos/preparacion.json") as archivo:
    paquete = json.load(archivo)
print(json.dumps({
    "paquete": paquete,
    "solicitud": {
        "alcance": "preparacion_sintetica",
        "contexto": {
            "convocatoria_ref": "convocatoria:ensayo",
            "bases_ref": "bases:ensayo",
            "bases_version": 1,
            "solicitud_ref": "solicitud:ensayo",
            "hito_ref": "hito:ensayo",
            "uso": "merito"
        },
        "selector": {
            "persona_ref": "persona:ensayo:01",
            "fecha_corte": paquete["fecha_corte"],
            "hechos": [
                {"referencia": "hecho:ensayo:01", "version_esperada": 2},
                {"referencia": "hecho:ensayo:02", "version_esperada": 1}
            ]
        }
    }
}))
PYTHON
```

El primer hecho sigue pendiente; el segundo conserva la afirmación de acreditado
aportada por el ejemplo y sus comprobaciones pendientes. `uso` admite `requisito`
o `merito`: cada ejecución conserva ese contexto sin decidir cumplimiento ni
calcular puntos. `hito_ref` y `fecha_corte` son referencias y fecha civil aportadas
por el ensayo; no establecen una excepción ni un plazo jurídico.

Una versión anterior, un hecho ajeno o ausente, referencias duplicadas o un
paquete contradictorio producen error sin resultado parcial. La entrada se limita
a 1 MiB; el paquete admite los límites existentes de RUM y la selección hasta
1000 referencias. No se abren documentos, enlaces ni rutas indicados en el JSON.

`lectura_autorizada: false` indica que el CLI no obtiene autorización real;
no representa una denegación observada del sistema de permisos. Tampoco persiste,
acredita fuentes, decide admisión ni conecta con el baremador o el portal.
La consulta real RUM05 requiere su perfil nominal por proceso, lectura auditada
y correspondencias gobernadas con las bases y catálogos del baremador.

# Capturas anotadas para manuales

`capturar_y_anotar.py` abre un servidor local con Chrome del sistema y guarda una captura a 1440 × 900 y otra a 390 × 844 por pantalla. Cada número dibujado corresponde al texto de una marca en `manifiesto.json`. El fichero incluye dimensiones y SHA-256 de cada PNG. No guarda URL, cookies, consola ni contenido de las respuestas.

## Comprobar la herramienta

Desde la raíz del repositorio:

```bash
python3 scripts/manuales/capturar_y_anotar.py --ensayo --salida /tmp/vec-capturas-ensayo
python3 -m unittest scripts/tests/test_manual_capturas.py
```

El ensayo levanta una página local mínima y genera dos PNG. Lleva la etiqueta `ensayo-sintetico`: sus imágenes no son capturas de VEC y no deben incluirse en un manual. Necesita Python Playwright, Pillow y `/usr/bin/google-chrome`.

## Capturar un recorrido local

Prepare una instancia local con **datos exclusivamente sintéticos**. Revise cada pantalla y señale en `ocultar` cualquier dato que deba taparse. El guion tapa también campos de entrada, contenido editable y elementos marcados con `data-private` o `data-sensitive`. Si ve correos, DNI o secretos en el texto de la página, se detiene. Esta comprobación automática no reconoce todos los datos personales; revise los PNG antes de incorporarlos a un manual.

Ejemplo de `escenario.json`:

```json
{
  "pantallas": [
    {
      "clave": "peticiones",
      "ruta": "/portal-empleado/peticiones-centro/",
      "pasos": [
        {"accion": "esperar", "selector": "main h1"},
        {"accion": "rellenar", "selector": "#busqueda", "valor": "ejemplo sintético"},
        {"accion": "clic", "selector": "[data-vista='detalle']"},
        {"accion": "esperar", "selector": "#detalle:visible"}
      ],
      "marcas": [
        {"numero": 1, "selector": "main h1", "texto": "Título de la página", "tipo": "recuadro"},
        {"numero": 2, "selector": "#detalle", "texto": "Detalle de la petición", "tipo": "flecha"}
      ],
      "ocultar": [".datos-personales"]
    }
  ]
}
```

```bash
python3 scripts/manuales/capturar_y_anotar.py \
  --base-url http://127.0.0.1:8081 \
  --escenario escenario.json \
  --salida /tmp/vec-capturas-manual \
  --datos-sinteticos-confirmados
```

`ruta` debe empezar por `/` y no admite parámetros. Los pasos admiten `clic`, `esperar`, `rellenar` y `seleccionar`; los dos últimos requieren `valor` sintético. `seleccionar` usa el valor de una opción HTML. Los selectores son de Playwright. Cada marca admite `recuadro` o `flecha`. Los mismos pasos y selectores se ejecutan en **ambos tamaños**; los elementos marcados deben quedar visibles en los dos.

Si el servidor local exige certificado de cliente, añada `--mtls-certificado /ruta/externa/cliente.crt --mtls-clave /ruta/externa/cliente.key`. Los archivos deben estar fuera del repositorio. Si la clave tiene frase, expóngala en una variable del proceso y pase solo su nombre mediante `--mtls-frase-env NOMBRE_VARIABLE`. El guion no copia esos archivos ni guarda sus rutas o la frase en el manifiesto. Los recursos internos con `?v=` se cargan sin registrar sus parámetros.

La navegación se limita al mismo origen de loopback. Chrome usa un contexto nuevo por captura, sin perfil ni almacenamiento persistente. Un clic o un formulario pueden activar una escritura: use una instancia sintética y configure solo las acciones necesarias para el recorrido. Compruebe en la aplicación el resultado real de cada paso antes de describirlo en el manual; una captura por sí sola no acredita una operación ni su persistencia.

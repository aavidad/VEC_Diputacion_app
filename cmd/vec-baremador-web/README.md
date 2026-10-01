# Simulador local de baremo

Abre el editor de reglas sobre dos ejemplos sintéticos: experiencia y méritos.
El cálculo usa los mismos servicios Go que `vec-baremador`. No consulta personas
ni registra baremaciones, activa reglas o modifica una convocatoria.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-baremador-web --web-dir web/static
```

El programa muestra la dirección que debe abrirse en Chrome. Escucha solo en
`127.0.0.1`; por defecto el sistema asigna un puerto libre. Para elegirlo, añadir
`--puerto 49153`. La ruta de los recursos la fija el operador con `--web-dir`.
Solo se cargan los recursos del editor enumerados en `http.go`; el servidor no
publica el directorio completo. Los catálogos del editor se cargan para los
idiomas declarados en `textos/idiomas.json`.

`GET /ejemplos` devuelve las reglas de cada ejemplo. `POST /simular` recibe
`modo`, `ejemplo_ref` y el objeto `reglas`. El servidor fija los datos sintéticos
de entrada, valida las reglas y calcula su representación canónica en Go.
Devuelve el resultado con su huella. Un bloqueo conserva sus motivos y no tiene
total; una regla inválida devuelve 422. El POST exige el origen y el Host exactos
de la dirección mostrada al arrancar, con `Content-Type: application/json`.

No admite ficheros, URLs, datos personales ni credenciales. No emite cookies,
CORS ni caché. Cada solicitud tiene un máximo de 256 KiB, 128 elementos por
colección, 32 niveles y 8192 valores; se permiten dos simulaciones simultáneas.
El servidor limita la lectura de cabeceras a dos segundos, la lectura completa
a cinco y la escritura a diez. Se detiene con Ctrl+C.

Comprobaciones focales:

```sh
go test -race ./cmd/vec-baremador-web ./internal/modules/bolsa/adapters/simuladorlocal
go vet ./cmd/vec-baremador-web ./internal/modules/bolsa/adapters/simuladorlocal
```

Las pruebas comparan la salida HTTP con el servicio común y comprueban origen,
Host, método, campos desconocidos, exceso de tamaño y escapes del directorio de
recursos. Esta herramienta local no se monta en el portal ni acredita permisos
institucionales, aprobación de bases o una valoración oficial.

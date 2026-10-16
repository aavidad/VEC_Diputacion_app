# Ensayo de la candidata HZ39

`recorrer.py` recorre las pantallas del caso único de `FLUJO_APP.md`, apartado
4.0. El destino y los certificados son parámetros. Puede apuntar a una copia
local o a una segunda instancia autorizada por dirección a través de un túnel
local. No accede directamente a cidonia.

La preparación del guion no demuestra que la candidata arranque ni que termine
el caso. Esos resultados se obtienen al ejecutarlo sobre el destino congelado
con su configuración, datos y actos ADMIN. El material privado lo entrega
dirección; no se obtiene del servidor desde este guion.

Reutiliza la captura y auditoría de `scripts/revision_web`, el registro HTTP de
`recorridos/convoca_integrado.py`, los ficheros privados y filtro sin redirección
de `recorridos/intervencion/recorrer.py`, y las acciones/selectores de los tramos
de centro, análisis, llamamiento, Intervención, firma y B2 existentes.

## Entradas privadas

Python y Playwright ya instalados, `/usr/bin/google-chrome` y OpenSSL. Chrome
confía en la CA del destino por el almacén preparado por dirección. La validación
TLS permanece activa. No hay respuesta simulada, cookie ni estado web guardado.

- `--base-url`: origen HTTPS de loopback con puerto, sin credenciales ni ruta;
  si se usa un túnel, dirección fija antes qué instancia hay detrás.
- `--ca`: certificado de la CA que verifica ese destino. También se admite
  `NODE_EXTRA_CA_CERTS` cuando ya apunta a la misma CA de la copia. El transporte
  usa el certificado del actor y conserva estado, cabeceras y bytes de la
  respuesta real; no sigue redirecciones ni reintenta un POST. Reutiliza la
  conexión HTTPS del ensayo de preparación de bases. Evita la cabecera
  `Connection` que añade `route.fetch` y que la frontera de CT rechaza.
  Los PEM temporales se crean privados y se eliminan al terminar.
- `--certificados`: carpeta de P12, sólo lectura. Nombres del juego actual:
  `candidato.p12`, `centro-solicitante.p12`, `centro-ratificador.p12`,
  `cliente.p12`, `intervencion.p12`.
- `--claves`: JSON privado `0600` fuera de Git. Las claves son `aspirante`,
  `centro`, `ratificador`, `rrhh` e `intervencion`; sus valores son las
  contraseñas P12 entregadas por dirección. No ponerlas en argumentos ni Git.
  El guion comprueba los certificados públicos y rechaza personas repetidas.
- `--datos`: JSON privado `0600` con `candidata` (hash Git exacto de 40 dígitos hexadecimales), `valores` y
  `extraer`. Los valores contienen el MOAD, categoría/puesto/centro publicados,
  fechas, RC, motivo, bolsa, datos de mérito, archivos aportados y opciones B2
  de ese caso. K proporciona la selección y receta de los datos existentes.
  La petición usa el código RPT `9`, plaza `2026/0033`, clasificación orgánica
  y funcional, proyecto de gasto y porcentaje de financiación. El centro
  indica que no tiene retención de crédito; RRHH la registra en el análisis.
- `extraer`: lista de `{metodo,ruta,campo,variable}`. Coteja la ruta observada y
  copia una referencia real del JSON de respuesta, por ejemplo
  `campo: data.expediente_ref` a `variable: expediente_ref`. Las referencias de
  petición, solicitud e incorporación se obtienen de sus respuestas reales;
  no se inventan ni se supone que una ruta haya creado un expediente.
- `--salida`: carpeta nueva fuera de Git, con padre privado `0700`.
  Cada ejecución tiene su salida; las anteriores no se sobrescriben.

`pasos_hz39.json` mantiene el caso aprobado: el recorrido principal comienza
en la petición con Antonio disponible en la bolsa y contiene los pasos 1 a 21, con 7b (firma de remisión) y 17 → 18 → 16a → 16b (incorporación,
relación de Personal, descarga y confirmación GINPIX). Los datos que falten se ajustan por los guiones/semillas
existentes en la copia; este guion no fabrica filas ni otra implementación.

## Uso

```sh
umask 077
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --ca "$CA_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_NUEVA" --modo comprobar

# Sólo después de que dirección fije destino, candidata y material:
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --ca "$CA_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_NUEVA" --modo ejecutar --visible --video
```

`comprobar` no abre Chrome ni conecta: revisa entradas y enumera lo disponible.
`--pasos` limita la observación. Los tramos e1–e5 son demostraciones opcionales
que se ejecutan aparte al pedir sus IDs y no bloquean el principal (orden de
Alberto del 14/10 a las 00:20). Sólo se cierra el principal si todos sus pasos
terminan; ejecutar únicamente la demostración nunca lo cierra. Cada paso ejecutado guarda HTTP, tiempo, dueño, recibos y capturas de
1440/390. Los vídeos son privados y se graban por contexto/certificado. Las
contraseñas, cuerpos y cabeceras de identidad no van a la evidencia.

Si falla un paso, queda con su causa y dueño; los dependientes quedan pendientes,
y los independientes continúan. Una firma por H se observa en el panel y en el
estado real de firmas. 7b exige informe verificado; 15 exige informe/remisión y
resolución de tres cargos. Los otros cuatro documentos se descargan y comparan,
con firma pendiente, según la decisión de dirección del 14/10 a las 20:55. No se
considera una firma completada porque se abra el panel o el operador lo afirme.

`continuar` conserva los pasos que ya terminaron. Un acto previo iniciado pero
sin paso cerrado se bloquea: primero se recupera el resultado original, nunca
se pulsa otra vez con una clave nueva para superar una respuesta perdida.
Durante la preparación se pueden incorporar correcciones y continuar con un
commit distinto. Ese avance no acredita la candidata final. La prueba de
aceptación se ejecuta una vez al final: commit congelado, copia restaurada
limpia, recorrido entero y reinicio. La recuperación de esa prueba exige el
mismo commit y conserva los recibos originales.

## Reinicio y recuperación

Dirección o quien posee la copia reinicia únicamente sus contenedores. Este
guion no reinicia servicios ni escribe SQL. Se mantiene la misma candidata y
los mismos datos; el origen debe seguir apuntando a la copia acordada.

```sh
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --ca "$CA_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_RECUPERACION_NUEVA" --modo recuperar \
  --anterior "$RESULTADO_ORIGINAL_JSON" --reinicio-acreditado \
  --centro-pre-reinicio "$BANDEJA_CENTRO_ANTES_REINICIO_JSON" \
  --postgres-contenedor "$CONTENEDOR_POSTGRES_PROPIO" --postgres-puerto "$PUERTO_POSTGRES_PROPIO"
```

Antes del reinicio, guardar en un JSON privado la respuesta real de la
bandeja del centro, obtenida con su certificado. Después, los pasos 01–02
exigen la misma referencia, versión y estado de esa bandeja y su fila visible.
El recibo de cada acto se coteja por SQL de solo lectura en el contenedor
propio: referencia, versión y fecha original deben coincidir tanto en las
columnas como en el JSON guardado. Esta consulta está autorizada por dirección
(14/10, 23:50) y no añade una ruta HTTP ni repite una operación. Sin la
evidencia anterior o sin los dos recibos completos, el cotejo se detiene.

La recuperación usa pantallas y lecturas de la proyección. Bloquea las peticiones
con efecto antes de enviarlas; permite los POST de consulta declarados. Compara
el recibo, fecha y versión del acto original con la ejecución anterior. Un acto
sin respuesta ni recibo sigue pendiente, aunque se vea una ficha; una recuperación
fallida conserva el bloqueo para impedir que `continuar` lo repita. El paso 14
vuelve a descargar los doce PDF/Word y exige identidad, tamaño y huella iguales;
una lista de descargas vacía nunca pasa. Si una
pantalla no permite recuperar un recibo, se informa y no se acredita ese paso.
La opción `--reinicio-acreditado` declara la actuación del dueño de la copia;
por sí sola no demuestra el reinicio.

Salida: `resultado.json`, `paso-*.json`, intenciones minimizadas antes de los
POST, capturas, documentos y vídeo. Sólo `completo: true` con todos los pasos principales
OK acredita el recorrido del guion; parcial o pendiente devuelve código 1.
Una precondición inválida devuelve código 2 sin atribuir una ejecución válida.

La focal de guardas del guion se ejecuta con:

```sh
python3 -m unittest discover -s deploy/principal/ensayo_candidata -p test_recorrer.py -v
```

No sustituye el ensayo único de la aplicación real congelada.

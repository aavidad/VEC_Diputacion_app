# Comparar dos cortes de organización y RPT

La herramienta recibe una preparación sintética por la entrada estándar y escribe JSON o un informe HTML autónomo. El ejemplo compara el Servicio de Cultura en dos fechas: cambia una dotación, una plaza tiene amortización explícita y un puesto individual sale de la selección. La salida del corte no acredita su supresión.

Desde la raíz del repositorio:

```sh
go build -o /tmp/vec-comparar-organizacion ./cmd/vec-comparar-organizacion
/tmp/vec-comparar-organizacion < cmd/vec-comparar-organizacion/ejemplo.json > /tmp/comparacion.json
python3 -c 'import json; p=json.load(open("cmd/vec-comparar-organizacion/ejemplo.json")); p["formato"]="html"; print(json.dumps(p))' | /tmp/vec-comparar-organizacion > /tmp/comparacion.html
```

Abra el HTML en Chrome. Las tablas se desplazan dentro de cada apartado, también con teclado. La procedencia y el manifiesto se despliegan al pulsar sus títulos. Puede imprimir el informe desde el navegador; esta herramienta no firma ni custodia un PDF.

Para cambiar el idioma, establezca `idioma` en uno de los códigos de `web/static/textos/idiomas.json`. Si se omite, se usa el idioma por defecto del catálogo común. `formato` admite `json` y `html`.

Cada corte contiene su selector temporal, las versiones RPT y plantilla resueltas, seis colecciones y su cobertura: `completa`, `parcial` o `sin_datos`. La cobertura procede de la preparación; la herramienta no comprueba una fuente externa. Las cifras cuentan registros recibidos, no puestos disponibles ni vacantes. Una dotación agrupada cuenta como un registro, aunque su cantidad sea mayor.

Las ausencias solo se comparan cuando ambas coberturas son completas. Con cobertura parcial quedan sin verificar. Las versiones documentales y la procedencia se diferencian de los cambios estructurales. No se reciben personas ni ocupaciones.

El manifiesto es idéntico en JSON y HTML. La huella de entrada usa los datos tipados validados, sin idioma ni formato. La huella del manifiesto usa su JSON compacto, con las claves y orden de la estructura Go. No es la huella de los bytes del HTML ni de una firma. Repetir exactamente la misma preparación produce el mismo informe.

La entrada tiene un límite de 8 MiB. Se rechazan claves desconocidas, duplicadas, más de un documento, una cadena de páginas sin terminar y datos inválidos para el dominio. El ensayo se ejecuta sin argumentos y no abre rutas indicadas por el documento. Los errores omiten los valores recibidos.

Comprobación técnica focal:

```sh
go test ./cmd/vec-comparar-organizacion ./web
go test -race ./cmd/vec-comparar-organizacion ./web
go vet ./cmd/vec-comparar-organizacion ./web
```

Las pruebas contrastan manifiestos y huellas, los dos idiomas, escape HTML, cobertura parcial, límites, datos inválidos y fallos de lectura o escritura. El ejemplo y la ejecución son preparación local: no conectan base de datos, red, autorización productiva ni auditoría.

## Preparación paginada sintética

Para ensayar una lectura paginada, use `ejemplo-paginado-sintetico.json`:

```sh
/tmp/vec-comparar-organizacion < cmd/vec-comparar-organizacion/ejemplo-paginado-sintetico.json > /tmp/comparacion-paginada.json
```

El esquema `vec.personal.comparacion-organizacion.paginas-sinteticas.v1` exige
`sintetico: true`, `antes_paginas` y `despues_paginas`. Omita `antes` y `despues`.
Cada página contiene `instantanea`, las versiones resueltas `version_rpt_ref` y
`version_plantilla_ref`, y `cursor_siguiente`. El selector de la primera página
lleva `cursor` vacío; los siguientes reproducen el cursor anterior. El último
`cursor_siguiente` queda vacío. `instantanea` conserva las seis colecciones,
el selector consultado y la cobertura declarada. Las versiones del selector
pueden quedar vacías si la página declara las versiones resueltas.

La reunión rechaza páginas ausentes, repetidas o posteriores al final;
identidades duplicadas entre páginas o colecciones; cambios de selector,
versiones o cobertura; páginas vacías con continuación y exceso de registros.
Admite hasta 10.000 hechos por corte, con el límite de hasta 100 hechos por página
y un máximo de 10.001 páginas. Terminar la cadena no convierte una cobertura
parcial en completa. El informe usa el mismo comparador y conserva todas las
páginas en el manifiesto JSON y HTML.

La CLI procesa una preparación local sin evidencia V3. No verifica la autoridad
de una fuente externa. El servicio de aplicación
`ServicioComparacionOrganizacionHistorica` reúne lecturas mediante
`ServicioConsultaOrganizacionHistorica.Consultar`: cada página pasa por su
concesión y consumo V3 y devuelve selector, recibo y auditoría. Este servicio
queda disponible para composición posterior; este corte no lo conecta al
servidor ni activa la pantalla histórica.

## Consulta nominal por HTTPS interno

El argumento `-consulta-nominal` activa un modo separado. Recibe la ruta absoluta
de un archivo JSON privado y consulta las páginas por la ruta interna fija del
cliente de Personal, con TLS y certificado de cliente. El servidor conserva la
autoridad sobre identidad, perfil activo, concesión V3 y auditoría de cada
lectura. Los selectores sirven para filtrar y comprobar la respuesta; no
conceden acceso ni envían un actor o un perfil.

```sh
/tmp/vec-comparar-organizacion -consulta-nominal "$CONFIGURACION_PRIVADA" < "$SELECTORES_PRIVADOS" > "$COMPARACION_PRIVADA"
```

Prepare esos tres valores fuera del repositorio. El archivo de configuración
debe tener permisos `0600`, dentro de un directorio `0700`, sin enlaces
simbólicos ni pertenencia a un árbol Git. Su tamaño máximo es 16 KiB. El JSON
admite únicamente estos campos:

- `version`: el número `1`.
- `origen`: origen HTTPS interno, sin ruta, consulta ni credenciales en la URL.
- `autoridad_ca`, `certificado_cliente` y `clave_cliente`: nombres relativos de
  los archivos TLS dentro del mismo directorio privado. La clave requiere `0600`.
- `maximo_bytes_pagina`: entero positivo con la cota de respuesta acreditada
  para el servidor configurado. Es obligatorio, no tiene valor por defecto y
  se comprueba que permita añadir el byte de control sin desbordar el entero.
  Los ensayos locales usan 1 MiB, que no acredita una cota para otro servidor.

La entrada estándar usa el esquema
`vec.personal.comparacion-organizacion.consulta.v1`, con `formato: "json"`,
`antes` y `despues`. Cada selector tiene `organismo_ref`, `unidad_clave`,
`vigente_en`, `conocido_en`, `version_rpt_ref`, `version_plantilla_ref`, `limite`
y `cursor`. Use el organismo y la unidad esperados de la configuración privada
del servidor. Ambos cortes deben coincidir en esos dos campos y comenzar con
el cursor vacío. Las fechas y versiones pueden diferir. No se admite
`sintetico`, `idioma`, HTML ni CSV en este modo.

La CLI valida los dos selectores antes de iniciar la primera consulta. Reúne
las páginas con las mismas reglas del dominio que usa el ensayo: hasta 100
hechos por página, 10.000 por corte y 10.001 páginas por corte. La consulta
completa dispone de un máximo técnico de dos minutos. Un fallo de lectura,
selector, versión, cobertura, cursor o identidad de hecho cancela la comparación
sin emitir un manifiesto parcial. También se rechaza repetir un recibo, decisión,
auditoría o huella de consumo entre páginas, incluidos ambos cortes. La referencia
del efecto puede repetirse porque corresponde al mismo organismo.

El JSON de salida contiene `manifiesto` y `huella_sha256`. El manifiesto usa el
esquema `vec.personal.comparacion-organizacion.consulta.manifiesto.v1` y el modo
`consulta_autorizada`. Incluye la comparación, los totales de hechos y páginas
por corte y dos listas de lecturas. Cada lectura conserva su selector, evidencia
de acceso, versiones resueltas, cobertura y siguiente cursor. Las colecciones
completas no se vuelven a exportar. La huella SHA256 corresponde al JSON compacto
de la estructura Go del manifiesto, con su orden de campos. No contiene el
origen, las rutas, los archivos TLS ni cabeceras.

El manifiesto conserva las evidencias recibidas del servidor. No añade un recibo
de comparación, firma legal ni acreditación de publicación de la fuente.
Una cobertura parcial sigue siendo parcial al terminar la paginación.

Este corte conecta la fábrica de cliente HTTPS a la CLI. Las pruebas de CLI
usan un cliente inyectado con datos de ensayo; las del adaptador verifican TLS
local. El uso nominal en un entorno concreto requiere un perfil y una asignación
internos con autorización positiva, CT162 para la auditoría de consultas y la
configuración privada validada. CT162 es una migración SQL de auditoría, no un
perfil de acceso. Esta documentación no acredita instalación, consulta de datos
reales ni autorización de producción.

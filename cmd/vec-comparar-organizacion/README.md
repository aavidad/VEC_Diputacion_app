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

La entrada tiene un límite de 8 MiB. Se rechazan claves desconocidas, duplicadas, más de un documento, un cursor pendiente y datos inválidos para el dominio. La herramienta no acepta argumentos ni abre rutas indicadas por el documento. Los errores omiten los valores recibidos.

Comprobación técnica focal:

```sh
go test ./cmd/vec-comparar-organizacion ./web
go test -race ./cmd/vec-comparar-organizacion ./web
go vet ./cmd/vec-comparar-organizacion ./web
```

Las pruebas contrastan manifiestos y huellas, los dos idiomas, escape HTML, cobertura parcial, límites, datos inválidos y fallos de lectura o escritura. El ejemplo y la ejecución son preparación local: no conectan base de datos, red, autorización productiva ni auditoría.

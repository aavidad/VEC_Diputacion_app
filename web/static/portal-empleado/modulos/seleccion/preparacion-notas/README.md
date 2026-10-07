# Ver una preparación de revisión de notas

El visor compara el antecedente y la propuesta producidos por
`vec-selectivos-preparar-notas`. Coteja sus referencias, las notas modificadas y
las diferencias. Requisitos, méritos y notas no propuestas se conservan.
La revisión competente, aprobación, firma y publicación siguen pendientes.

Desde la raíz del repositorio:

```sh
python3 scripts/servir_revision_notas.py
```

Abra en Chrome la dirección local que muestra el programa. Seleccione el JSON
preparado por el CLI. El archivo se lee en el navegador y no se envía al servidor.
«Descargar original» conserva sus bytes; «Cerrar archivo» retira el contenido y
devuelve el foco al selector.

La huella se muestra como aportada por el archivo. El visor no coteja su origen
ni una aprobación. El CLI la calcula sobre el JSON Go del resultado antecedente
recalculado; no sobre los bytes de entrada ni una salida anterior independiente.

El lector rechaza archivos de más de 1 MB, claves duplicadas, estados oficiales,
metadatos incompatibles y diferencias que contradigan ambos resultados. Una nota
pendiente no recibe un total ni un orden. Los textos están en castellano e inglés;
la tabla se apila en móvil y mantiene las etiquetas de cada valor.

El lanzador sirve sólo los recursos del visor en loopback, con los controles del
servidor común. Excluye pruebas, ejemplos y archivos preparados; no monta una API
institucional ni incorpora datos personales reales. Esta herramienta local no
acredita una reclamación registrada, calificación aprobada, firma o traspaso.

# Visor de las listas de admitidos y excluidos

Abre la salida `lista-provisional` o `lista-definitiva` del preparador de
admisión, la reconoce por su esquema y la muestra como la revisará RRHH. En la
provisional: un resumen con solicitudes, admitidas, excluidas y las
que pueden subsanar; el plazo de subsanación; las excluidas con sus motivos y
si pueden subsanar; las admitidas y lo que falta para publicar.

En la definitiva no hay plazo. Cada admitida indica si ya lo estaba en la
provisional o si entró tras subsanar o reclamar; cada excluida, los motivos que
siguen y si su escrito se desestimó o no presentó ninguno. El detalle técnico
añade la provisional de la que parte (referencia, revisión y huella).

```sh
go run ./cmd/vec-selectivos-preparar-admision --salida lista-provisional \
  < cmd/vec-selectivos-preparar-admision/testdata/lista-material.json \
  > lista-provisional.json
python scripts/servir_preparacion_rrhh.py --modulo selectivos-lista-admision-visor
```

Abra la dirección que imprime el servidor y seleccione `lista-provisional.json`.

Los textos de los motivos se leen de
`web/static/textos/<idioma>/motivos-<referencia del catálogo>.json`. Si falta
el texto de un motivo, el visor muestra su código. El resto de textos está en
`web/static/textos/<idioma>/selectivos-lista-admision-visor.json`; la ayuda
solo aparece tras el botón «?».

El archivo se lee en el navegador y no se guarda: cerrar, abrir otro o salir
lo retira. El visor no aprueba, publica ni cambia la lista, no usa cookies ni
almacenamiento web y no muestra nombres, que se añaden al publicar.

Pruebas:

```sh
node --test web/static/portal-empleado/modulos/seleccion/preparacion-lista-admision/modelo.test.mjs
python3 -m unittest scripts.tests.test_servir_preparacion_rrhh
```

El modelo exige el contrato exacto del CLI (borrador, sin aprobar ni publicar,
recuentos coherentes, subsanable igual a «todos sus motivos subsanables») y
rechaza claves duplicadas, campos de más y archivos de más de 8 MiB.

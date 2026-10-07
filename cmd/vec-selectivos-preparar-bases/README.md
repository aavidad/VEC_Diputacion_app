# Preparar material sintético de bases

Este CLI conserva una propuesta de bases y muestra qué falta para continuar.
Reutiliza la validación de contenido y referencias de Bolsa. No crea una
convocatoria ni un borrador gobernado: la salida siempre queda pendiente.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-selectivos-preparar-bases \
  --catalogos-dir web/static/textos --idioma es \
  < cmd/vec-selectivos-preparar-bases/testdata/material-completo.json

go run ./cmd/vec-selectivos-preparar-bases \
  --catalogos-dir web/static/textos --idioma en \
  < cmd/vec-selectivos-preparar-bases/testdata/material-incompleto.json
```

Los dos JSON son editables. El primero tiene contenido estructuralmente completo;
el segundo conserva la propuesta sin documentos, plazos ni referencias exactas.
El campo `contenido.tipo` puede usar la modalidad propuesta, por ejemplo
`oposicion`, `concurso` o `concurso_oposicion`. Este comando no interpreta sus
reglas ni calcula notas: las reglas de baremación quedan referenciadas para el
motor común existente.

`identidad_material` y `version_material` identifican revisiones locales de la
propuesta. No son la secuencia, revisión, historia ni control de concurrencia de
una convocatoria de Bolsa. El CLI no almacena versiones; cada salida puede
guardarse por separado mediante redirección.

La propuesta se devuelve sin rellenar datos ausentes. Solo aparece
`contenido_canonico_bolsa` cuando el contenido pasa `ClonarCanonico()` de Bolsa.
Las referencias con ID, versión y huella válida siguen sin verificar: el comando
no consulta su fuente, existencia ni vigencia. Las rutas de documentos son datos
propuestos; no se descargan ni se presentan como documentos disponibles.

Quedan pendientes los documentos admitidos, firma y custodia, acto de aprobación
y publicación oficial. Las preguntas 109–111 de `dudas.md` siguen abiertas.
Preparar material no concede permisos, acredita admisión ni produce derechos.
El gobierno real seguirá siendo responsabilidad de Bolsa y sus autoridades.

La entrada admite un único JSON UTF-8 de hasta 1 MiB, sin claves duplicadas,
alias de mayúsculas ni campos desconocidos. Los catálogos ES/EN propios viven en
`web/static/textos/<idioma>/selectivos-preparar-bases.json`; el traductor es el
común de VEC. Un error no devuelve material parcial. No hay conexión de red,
SQL, escritura de agregados ni montaje en el portal.

Pruebas focales:

```sh
go test -p 8 ./cmd/vec-selectivos-preparar-bases ./internal/modules/seleccion/...
```

El corte solo cubre la preparación parcial de S2 descrita en
`docs/plan_modulos/selectivos.md`. No cierra aprobación, firma o publicación.

## Exportar una petición para S2

Con `--salida solicitud-s2`, la entrada contiene tres campos: `material_propuesto`
con el mismo material local de los ejemplos, `esperada` con la preimagen del
guardado y `clave_operacion` con la clave estable elegida para esa intención.

```sh
go run ./cmd/vec-selectivos-preparar-bases \
  --catalogos-dir web/static/textos --idioma es --salida solicitud-s2 \
  < propuesta-con-preimagen.json > solicitud-s2.json
```

Para una preparación nueva, `esperada` contiene `preparacion_ref`, `revision: 0`
y `huella_material_sha256: ""`. Para editar una existente, debe contener la
referencia, revisión y huella recuperadas del servidor. Estos campos son
obligatorios: el comando no genera referencias, claves ni preimágenes. La
revisión `version_material` del archivo local no determina la revisión durable.

La salida contiene exclusivamente `esperada`, `material` y `clave_operacion`,
con la estructura que acepta la ruta S2 de guardado ya existente. Las diez
referencias propuestas conservan ID, versión y huella de contenido, incluido el
baremo; las ausencias se conservan. Siguen pendientes su comprobación y los
actos de aprobación, firma y publicación. Se aplica el canon de material de
Bolsa, que admite propuestas incompletas y limita tamaño y contenido.

La exportación no envía la petición ni la guarda. El servidor resuelve identidad,
perfil y ámbito, exige autorización actual, registra auditoría y comprueba la
preimagen. Una preimagen obsoleta permanece en la petición para que el CAS del
servidor la rechace. Reutilizar la misma petición y clave permite recuperar el
guardado original; cambiar la intención requiere otra clave.

Prácticas públicas consultadas para separar los pasos:

- [BOP de Granada: anuncio de admisión, tribunal y ejercicio](https://bop.dipgra.es/publica/buscador-anuncios/anuncio/LISTADO-DEFINITIVO-DE-PERSONAS-ADMITIDAS-TRIBUNAL-Y-FECHA-DEL-PRIMER-EJERCICIO-DEL-PROCESO-SELECTIVO-CONVOCADO-PARA-LA-COBERTURA-DE-3-PLAZAS-DE-TECNICO-A-DE-ADMINISTRACION-GENERAL-GRUPO-A-SUBGRUPO/): cada anuncio identifica administración, fecha y documento.
- [INAP: convocatoria publicada en BOE, TDF/569/2025](https://www.boe.es/buscar/doc.php?id=BOE-A-2025-11111): las plazas, bases y anexos se vinculan al acto de convocatoria; sus actuaciones tienen publicación propia.
- [GVA: convocatoria 74/26](https://sede.gva.es/es/detall-ocupacio-publica?id_emp=110138&id_info=info_basica): distingue bases, tribunal, admisión provisional y definitiva, y aprobados.
- [EAPC: recursos para órganos de selección local](https://eapc.gencat.cat/ca/seleccio/seleccio-administracio-local/espai-de-recursos-daprenentatge): distingue preparación de bases, órgano, pruebas, méritos y propuesta final.

El diseño de VEC toma de estas prácticas la separación de material y actos.
No incorpora sus plazos, reglas ni órganos al ejemplo sintético.

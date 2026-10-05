# Catálogo de admisión de Selección

`admision_ejemplo.json` reúne, por versión, los motivos de exclusión de una
convocatoria, si cada uno se puede subsanar y el plazo de subsanación. Es un
paquete de ejemplo: lo inventamos a partir de listas publicadas en el BOP de
Granada y se retira cuando RRHH confirme el suyo (pregunta 139 de `dudas.md`).

Para cambiar una regla no se toca el código. Se añade otra entrada en
`catalogos` con la misma `referencia` y una `version` nueva; las listas ya
preparadas conservan la versión con la que se hicieron.

- `plazo_subsanacion` usa las unidades de Calendarios (`dias_habiles`,
  `dias_naturales`, `meses`, `anios`). El vencimiento se calcula al publicar
  la lista, desde el día siguiente a la publicación.
- Una exclusión solo es subsanable si lo son todos sus motivos.
- Los textos de cada motivo están en el catálogo común de textos,
  `web/static/textos/<idioma>/motivos-<referencia>.json`, uno por código.
  Desde ahí los lee el visor de la lista.

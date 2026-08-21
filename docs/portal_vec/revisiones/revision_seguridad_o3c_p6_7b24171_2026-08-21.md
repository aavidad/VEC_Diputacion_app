# Revisión independiente de seguridad O3C-P6 — `7b24171`

Tarea: `O3C-P6`, revisión del candidato corregido exacto `7b24171`.

Estado: **NO-GO**.

## Alcance

Revisión read-only de la corrección de identidad de atestación, marcador FD,
staging y contexto. El único write-set es esta acta; no se modificaron fuentes,
pruebas, contratos ni documentos canónicos.

## Hallazgo bloqueante S-01 persistente — `stat` y lectura no están ligados

La corrección congela `atestacion_selectores_huella` con
`stat -c '%d:%i:%u:%a'`, y `validar_atestaciones_selectores` compara esa huella
contra el pathname. Sin embargo, la comparación y la lectura posterior son
operaciones separadas:

```text
stat(path) == huella_esperada
read(path)
```

No se abre el fichero antes del `fstat` ni se lee desde el descriptor cuya
identidad fue comprobada. Un proceso del mismo UID (incluido un descendiente
que sobreviva hasta la ventana de validación) puede sustituir el pathname
después del `stat` y antes de `read`; el objeto nuevo puede conservar el mismo
UID/mode, por lo que la defensa sigue siendo vulnerable a TOCTOU. La
revalidación del grupo y la comprobación de residuos no congelan ese pathname.

El mismo patrón afecta al marcador `fd-ambiental`: se hace `stat` y luego
`read`/`wc` sobre el pathname, sin descriptor ligado. Aunque el impacto
principal es la atestación de selectores, la evidencia del cierre de FD tampoco
queda criptográficamente ligada al objeto validado.

## Criterio de cierre

Abrir cada artefacto con una operación segura (descriptor no-follow cuando
corresponda), ejecutar `fstat` sobre ese descriptor, comparar dispositivo/inode,
UID y mode, y leer exclusivamente desde el mismo descriptor; comprobar tamaño
y contenido completo antes de aceptar. El marcador FD debe seguir el mismo
patrón o quedar generado por una primitiva que no dependa de pathname mutable.

## Resultado

La corrección mejora la identidad nominal, pero no elimina la ventana TOCTOU.
No se emite GO. Pruebas de conducta y ejecución integral no se realizaron por
ser una revisión de seguridad read-only y existir un hallazgo bloqueante.

Commit de acta: pendiente de commit local único.

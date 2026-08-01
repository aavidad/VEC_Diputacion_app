# Revisión CONFIG-CAT-01: catálogo operativo de configuración

Fecha: 2 de agosto de 2026.

## Dictamen

`NO-GO`, con `P0=0`, `P1=5` y `P2=0` en la revisión de seguridad, y
`P0=0`, `P1=3`, `P2=0` en la revisión operativa independiente.

Los candidatos `57bd7df` y `88aae846` permanecen aislados en la rama
`config-cat-01-20260802`. No se han integrado ni publicado en la rama estable.

## Alcance evaluado

El candidato inventariaba los 58 nombres de entorno observados en el código Go
productivo, sus lectores, valores predeterminados, secretos, anclas y uso por
las superficies pública, interna y emisora V4. También generaba plantillas
mínimas para Sistemas y mantenía separados los parámetros de infraestructura.

Las pruebas favorables reproducidas fueron:

- 12 pruebas focales y mutantes;
- plantilla pública con ocho asignaciones exactas;
- TLS público opcional con cardinalidad cero o dos;
- CLI anclada a un único catálogo;
- pruebas Go focales, carrera y `vet`;
- Ruff, `git diff --check` y Gitleaks sin hallazgos.

## Motivos del rechazo

Las revisiones adversariales demostraron que el control aceptaba:

1. incorporar variables internas o V4 a la superficie pública con una
   clasificación aparentemente válida;
2. alterar tipos, valores predeterminados, anclas o conjuntos de
   infraestructura sin conservar la semántica real;
3. construir nombres de entorno por concatenación o leerlos mediante un alias
   de `os` sin que el escáner de expresiones regulares los detectara;
4. omitir requisitos y grupos alternativos de la superficie interna;
5. derivar silenciosamente, porque el validador y sus pruebas no formaban parte
   de `scripts/verificar_calidad.sh`.

El cambio también ampliaba el `README.md` raíz hasta 839 líneas. Corregir el
escáner habría requerido incorporar y mantener un analizador semántico Go
propio junto a otro manifiesto manual. Esa solución aumentaría las fuentes de
verdad y no resolvería la causa: los cargadores siguen repartidos.

## Decisión de dirección

No se continúa ni se integra CONFIG-CAT-01. La rama estable conserva su código
y comportamiento anteriores. El inventario obtenido sirve como diagnóstico,
pero no se presenta como una garantía de seguridad ni como configuración
centralizada.

La corrección futura se dividirá en minitareas funcionales pequeñas:

1. declarar tipos y nombres de configuración en un paquete propietario por
   superficie, sin compartir secretos entre procesos;
2. sustituir lecturas directas por cargadores tipados que fallen cerrados;
3. generar la referencia operativa de Sistemas desde esas declaraciones, sin
   mantener un segundo catálogo manual;
4. integrar la comprobación de deriva y sus mutantes en la puerta de calidad;
5. retirar los nueve alias heredados solo mediante una migración documentada.

Cada cargador conservará segregación pública, interna y V4. Centralizar la
definición no autoriza a un proceso a recibir variables de otra superficie.

## Efecto sobre el avance

No cambia ninguna métrica funcional ni técnica de Bolsa o Contratación. El
experimento evitó integrar deuda, y queda documentado para que otro agente no
repita el mismo enfoque.

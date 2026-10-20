# Inventario de manuales y capturas

Revisión documental del 20 de octubre de 2026 sobre `origin/main@3c643ce34`.
Este inventario es para mantener la documentación; las instrucciones de uso
se consultan en el [índice por perfil](README.md).

## Cobertura por perfil

| Perfil | Documento existente | Qué necesita actualizarse |
| --- | --- | --- |
| Aspirante interno y externo | `aspirante.md` | Separar las dos entradas; explicar méritos y alegaciones; distinguir inscripción de incorporación. La entrada externa comprobada sin participaciones no acredita sus actos. |
| Centro solicitante | `centro.md` | Quitar la dirección de cidonia como acceso vigente; añadir selección del puesto publicado y acreditación de incorporación. |
| Ratificador | `ratificador.md` | Quitar el enlace operativo a cidonia y marcar las imágenes del 9/10 como históricas. Conservar separación de personas y recibo. |
| Técnica de RRHH | `tecnico_rrhh.md` | El texto salta del informe al llamamiento; incorporar firmas y fiscalización previas y explicar los pasos posteriores con sus límites. |
| Jefatura | `jefatura_rrhh.md` | Guía general extensa, mezcla usuario y activación técnica. Esperar las PR abiertas que la modifican antes de revisarla. |
| Responsable de RRHH | `responsable_rrhh.md` | Mantener consulta e historial; distinguir borradores de documentos firmados y retirar afirmaciones generales basadas en expedientes antiguos. |
| Intervención | `intervencion.md` | Conservar los resultados comprobados el 19 y 20/10; corregir la introducción que sigue dando por pendiente el reparo ya recorrido. |
| Administración | `sistemas.md` | Separar administración funcional de operación técnica. Falta guía de usuarios, perfiles y cambios pendientes basada en las pantallas actuales. |

## Carpetas que se solapan

| Carpeta | Contenido | Tratamiento |
| --- | --- | --- |
| `docs/manuales/` | Guías por perfil y pendientes; imágenes de octubre | Punto de entrada único. No acredita por sí solo un recorrido completo. |
| `docs/manual_usuario/` | Bolsa en Markdown, HTML y PDF; contratación temporal; generadores e imágenes | Manuales de septiembre, con ramas candidatas y estados antiguos. Marcar históricos y remitir al índice, conservando los originales. |
| `docs/manual_rrhh/` | Un README largo con resultados técnicos de septiembre y un correo ficticio | Marcar histórico: los recibos, contadores y versiones no describen el recorrido vigente. |
| `docs/manual/` | Capturas de diseño «antes» y «después», sin manual por perfil | Conservar como comparación histórica; no usarlas como instrucciones actuales. |
| `docs/capturas/` | Peticiones del 2/10 y recorrido del 5/10 | Conservar con fecha y advertencia de alcance. No demuestran el recorrido del 20/10. |

No se ha acreditado que una pantalla haya desaparecido por el mero hecho de
que su manual sea antiguo. Sí hay instrucciones desfasadas: cidonia como
acceso vigente, firma aún ausente, llamamiento sin fiscalización previa y
contadores de pantallas de septiembre. El contraste funcional se hace con
[el recorrido vigente](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación)
y los catálogos de `web/static/textos/es/`.

## Estructura que se mantiene

Se conservan los nombres de las guías existentes para no romper enlaces.
`README.md` reúne los perfiles; `aspirante.md` distingue acceso interno y
externo; `administracion.md` recoge el uso funcional y `sistemas.md` remite a
la operación técnica. `PENDIENTES.md` resume lo que falta para completar el
recorrido, sin trasladar códigos del tablero a los usuarios.

Las capturas conservan su fecha y su contexto. No se crean copias locales
ni se toman imágenes nuevas para esta actualización documental. La revisión
de Jefatura espera a que se integren los cambios de los otros equipos.

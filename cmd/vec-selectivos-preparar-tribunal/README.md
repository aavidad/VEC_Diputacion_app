# Preparar una propuesta de tribunal

Esta herramienta convierte un archivo JSON sintético en material para revisar
la composición del tribunal. Conserva miembros, fases y propuestas de abstención,
recusación o sustitución sin retirar ni designar a nadie. Los textos se cargan
del catálogo común en castellano o inglés.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-selectivos-preparar-tribunal \
  -catalogos-dir web/static/textos -idioma es \
  < cmd/vec-selectivos-preparar-tribunal/testdata/material.json
```

Cambie `-idioma es` por `-idioma en` para obtener los avisos en inglés.
El resultado sale por stdout y los errores por stderr, en JSON. Un resultado
válido devuelve código 0 y estado `pendiente`. Si la entrada es inválida,
devuelve código 1 sin material por stdout. No escribe archivos ni llama a servicios.
Si no puede escribir el diagnóstico por stderr, devuelve código 2.

## Datos de entrada

`bases_s2` reutiliza las coordenadas de `preparacionbases.Esperada`: referencia,
revisión positiva y huella del material. Identifica una revisión **local de
preparación**, aportada por el operador. No acredita bases aprobadas ni comprueba
la existencia de esa revisión en Bolsa. No admite la revisión cero de alta.

`fases_propuestas` contiene referencias opacas explícitas. Cada miembro aporta
su referencia local, una referencia de persona sintética, un rol propuesto y sus
fases. El flujo se conserva por referencia exacta; no permite deducir fases.
El baremo es opcional y permanece en su propietario. No se vuelve a calcular.

Cada incidencia enlaza un miembro y sus fases. `causa_catalogada` fija una
referencia de causa, versión y huella; su existencia y aplicación siguen
pendientes de comprobación. `evidencia_ref` conserva solo una referencia opaca,
sin documentos ni datos personales. Puede faltar y se muestra como pendiente.
Una sustitución añade `sustituto_ref`, que debe enlazar otro miembro propuesto
para esas fases. Los tipos admitidos son `abstencion`, `recusacion` y `sustitucion`.

Se rechazan miembros o personas repetidos, fases ajenas, duplicados de incidencia
por tipo/miembro/fase, sustituciones a sí mismo y ciclos de sustitución por fase.
Dos sustituciones hacia el mismo miembro en una fase también se rechazan.
La misma persona puede requerir otro modelo de representación en el circuito
institucional; esta preparación conserva una sola entrada por referencia.
Se permite completar una propuesta sin fases ni miembros y se indican los datos
pendientes. Una referencia parcial o mal formada se rechaza.

La entrada admite como máximo 1 MiB; los catálogos, 64 KiB. La estructura limita
las propuestas a 100 miembros, 100 fases y 200 incidencias. Son límites técnicos
de esta herramienta: no establecen el tamaño legal del tribunal ni su quórum.
Se rechazan campos desconocidos, claves repetidas y variantes en mayúsculas.
Las referencias y huellas del ejemplo son ficticias.

## Siguiente dependencia

RRHH y el órgano competente deben comprobar las fuentes, la composición y las
causas; resolver cada propuesta; y aportar el acto de designación. La autoridad
común habilitará el acceso por proceso y fase con permisos reales. Las actas y su
firma quedan para un corte posterior a admisión y habilitación.

Este corte no conserva historia durable, concede permisos, calcula elegibilidad
o quórum, ni produce designaciones, actas, firmas o publicaciones oficiales.
No acepta identidad nominal ni datos reales. Repetirlo produce la misma propuesta;
esa repetición no constituye recuperación de un recibo institucional.

Pruebas focales ejecutadas con resultado correcto:

```sh
go test -p 8 ./internal/modules/seleccion/domain \
  ./internal/modules/seleccion/application ./cmd/vec-selectivos-preparar-tribunal
```

La comprobación de gopls señala el contexto nulo del test
`TestPrepararTribunalRespetaContexto`. Esa entrada es intencional: comprueba
que la aplicación la rechaza antes de preparar material.

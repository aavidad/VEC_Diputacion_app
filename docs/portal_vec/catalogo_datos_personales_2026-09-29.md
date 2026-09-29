# Catálogo de datos personales por finalidad y momento

Fase 1, paso 2 del estudio de datos personales de las inscripciones (PR #132).
El catálogo dice, para cada tipo de convocatoria, qué datos personales se piden,
cuándo, para qué y de dónde salen. El módulo Aspirantes lo usa como única fuente:
lo que no figura para un tipo y un momento no se pide ni se admite.

## Dónde está

- Paquete de ejemplo: `data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json`.
  Lleva los valores provisionales del apartado 2 del estudio y la marca
  `paquete:ejemplo:vec:v1`. Está pendiente de RRHH y del DPD y se retira antes de
  producción, como los demás paquetes de ejemplo.
- Lectura: paquete Go `internal/vec/datospersonales`.
- Textos visibles: `web/static/textos/{es,en}/datos-personales.json`. El catálogo
  solo lleva códigos; la pantalla los traduce con estas secciones.

## Qué dice cada entrada

Una entrada es un dato que un tipo de convocatoria pide en un momento. La clave es
`<tipo>.<momento>.<dato>` y los atributos son:

| Atributo | Valores |
| --- | --- |
| `tipo_convocatoria` | `bolsa`, `selectivo_libre`, `promocion_interna`, `provision` |
| `momento` | `inscripcion`, `pago_tasa`, `admision`, `baremacion`, `pruebas`, `llamamiento`, `contratacion` |
| `momento_comprobacion` | Opcional. Momento posterior en que se comprueba lo declarado |
| `obligatoriedad` | `obligatorio`, `condicional` (con `condicion`), `voluntario` |
| `finalidad`, `base_rgpd` | Código de finalidad y artículos del RGPD (`6.1.e,9.2.b`) |
| `fuente` | `identificacion_electronica`, `persona`, `consulta_administracion`, `personal_diputacion`, `vigilancia_salud` |
| `consulta_servicio`, `consulta_regimen` | Solo si se consulta a otra Administración. `oposicion` (art. 28.2 de la Ley 39/2015: se consulta salvo que la persona se oponga) o `autorizacion` (datos tributarios) |
| `fuente_alternativa` | A quién se pide si la persona se opone. En una consulta es siempre `persona` |
| `categoria` | `ordinaria`, `especial` (art. 9), `penal` (art. 10), `protegida` (víctimas) |
| `custodia` | `aspirantes` (bolsa y selectivo libre), `procesos_empleado` (promoción interna y provisión, desde el portal interno) o `personal` (datos del nombramiento o contrato) |
| `origen`, `pendiente_de` | `ejemplo` con `rrhh`, `dpd` o `rrhh_dpd`; `aprobado` con `aprobacion_ref` |
| `norma`, `duda` | Cita de la norma y pregunta de `dudas.md` que lo confirma |

Reglas que el código comprueba (una entrada que las incumple invalida todo el
catálogo):

- Los datos del momento `contratacion` los pide y guarda Personal, y solo esos.
- Promoción interna y provisión nunca guardan datos en Aspirantes.
- Un dato penal nunca se guarda en Aspirantes.
- Un dato especial, penal o protegido solo es obligatorio si lo guarda Personal, y un
  dato especial cita una letra del art. 9.2 del RGPD.
- Una consulta a otra Administración nombra servicio y régimen y deja que la
  persona aporte el documento si se opone.
- Un paquete de ejemplo solo tiene valores de ejemplo, y un valor de ejemplo no
  puede ir en un catálogo aprobado.
- No se repite un dato en el mismo tipo y momento.

## Entradas que no vienen del apartado 2

Dos entradas del paquete no están en el inventario del estudio y RRHH debe
confirmarlas con la pregunta 80 de `dudas.md`:

- `bolsa.llamamiento.justificante_renuncia`: justificante de la causa de renuncia a
  un llamamiento (dudas 2 y 11). Puede revelar un dato de salud, por eso es
  categoría especial.
- `provision.baremacion.conciliacion_familiar`: mérito de conciliación, habitual en
  las bases de los concursos. Revela datos de terceros (hijos o familiares); se
  guarda el mérito reconocido.

## Cómo lo lee Aspirantes

```go
resolutor, err := datospersonales.NuevoResolutor(consulta, metadatos, reloj)
datos, procedencia, err := resolutor.Para(ctx, datospersonales.TipoBolsa, datospersonales.MomentoInscripcion)
catalogo, err := resolutor.Catalogo(ctx)
entrada, permitido := catalogo.Permitido(tipo, momento, "telefono")
```

- `consulta` es el mismo puerto de catálogos del resto de VEC
  (`ports.ConsultaCatalogosConfigurablesAcotada`). En desarrollo, el adaptador
  `fichero` sobre el paquete de ejemplo; en producción, el almacén de catálogos.
- `Para` devuelve los datos de ese tipo y momento y la procedencia (catálogo,
  versión y huella) para anotarla en el recibo de la solicitud.
- `Permitido` sirve para rechazar en la frontera un campo que el catálogo no prevé.
- Sin catálogo compuesto la respuesta es `ErrCatalogoNoConfigurado`; con un catálogo
  ilegible o no vigente, `ErrCatalogoNoDisponible`. Ninguno de los dos significa
  «se puede pedir cualquier cosa».
- Si `EsEjemplo()` es cierto, la pantalla lo rotula con el texto
  `aviso.paquete_ejemplo` o `pendiente_de.<valor>`.

La composición en el arranque (variable de entorno y doble llave de desarrollo,
como los demás paquetes de ejemplo) la hace el módulo que lo consume.

## Cómo se cambia

RRHH o el DPD corrigen un valor publicando una versión nueva del catálogo con los
cambios. El resolutor lee el catálogo en cada consulta, así que no hace falta
reiniciar. Cuando un valor quede aprobado, pasa a `origen: aprobado` con su
`aprobacion_ref` y deja de llevar la marca de ejemplo.

# Módulo Aspirantes: diseño del primer corte

Fecha: 29 de septiembre de 2026. Fase 1, paso 5 del plan del estudio
[`datos_personales_inscripciones_rgpd_2026-09-29.md`](../estudio_requisitos/datos_personales_inscripciones_rgpd_2026-09-29.md)
(apartados 4, 5, 6 y 8). Solo datos sintéticos; nada se instala en la principal
por estar en esta rama.

## Qué hace este corte

Aspirantes es el dueño de los datos personales de las personas externas
(aspirantes). Este corte le da:

- una ficha por persona con identificador aleatorio `asp_`, nunca derivado del
  documento ni del nombre;
- cada dato cifrado por separado en la aplicación, con la clave del portal externo;
- el documento de identidad con tipo, país y número, y su historial;
- un índice ciego del documento para encontrar la ficha y detectar duplicados sin
  descifrar;
- historia de solo adición: cada cambio es una versión nueva con motivo;
- registro de cada lectura, con su finalidad;
- el apartado «Perfil y contacto» del área personal: teléfono, móvil, domicilio y
  código postal, y solo los que pida el catálogo de campos.

Queda fuera: datos de categoría especial (discapacidad, víctimas; fase 2), vistas
de RRHH y tribunal, fusión de fichas, registro de correspondencias, «Mis datos» y el
paso de Bolsa a `asp_`. Todo eso tiene su hueco previsto en el modelo.

## Cortes

| Corte | Contenido | PR |
| --- | --- | --- |
| 1 | Este diseño, dominio, puertos, forma canónica y aplicación, con pruebas | 1 |
| 2 | Criptografía (AES-GCM e índice ciego), SQL (roles, esquema, consumidor V3), adaptador PostgreSQL y prueba PG18 | 2 |
| 3 | API, identidad del certificado, composición y pantalla «Perfil y contacto» | 3 |

## Paquetes Go

```
internal/modules/aspirantes/
  manifest.go                 permisos vec.aspirantes.ficha.*
  domain/                     referencias, documento, campos, identidad acreditada, ficha
  ports/                      errores, acciones y audiencias, material, sobres, puertos
  canonico/                   preimagen única del material V3 y del recurso
  application/                ServicioFichaPropia: Consultar, Alta, Rectificar
  adapters/seguridad/         AES-256-GCM por campo, HMAC del índice ciego y huellas
  adapters/postgres/          llamadas a las funciones de vec_aspirantes
  adapters/certificado/       lee nombre, apellidos y documento del certificado verificado
  adapters/httpapi/           GET y POST de «mi ficha» en el área personal
```

El dominio no importa aplicación, puertos ni adaptadores. La aplicación no conoce
HTTP, SQL ni X.509.

## Identificadores

| Referencia | Formato | Origen |
| --- | --- | --- |
| Ficha | `asp_` + 22 caracteres base64url (16 bytes aleatorios) | CSPRNG en Go |
| Documento | `aspdoc_` + 22 caracteres base64url | CSPRNG en Go |
| Recibo | `asprec_` + 32 hex | `gen_random_uuid()` en SQL |
| Acceso | `aspacc_` + 32 hex | `gen_random_uuid()` en SQL |
| Evento de salida | `aspevt_` + 32 hex | `gen_random_uuid()` en SQL |

Aspirantes no guarda `per_` en ninguna tabla. La ficha propia se encuentra por el
índice ciego del documento que acredita el certificado de la sesión.

## Datos de la ficha

Vocabulario cerrado. Añadir un campo exige otra migración.

| Campo | Grupo | Origen | Validación |
| --- | --- | --- | --- |
| `nombre`, `apellidos` | identidad | certificado | letras, espacios, apóstrofo, guion y punto; hasta 100 caracteres el nombre y 200 los apellidos |
| `telefono` | contacto | titular | 9 cifras nacionales (6, 7, 8 o 9 al inicio) o internacional `+` y 8 a 15 cifras; se guarda normalizado |
| `movil` | contacto | titular | como el teléfono, pero nacional solo con 6 o 7 al inicio |
| `domicilio` | contacto | titular | 5 a 200 caracteres imprimibles; espacios normalizados |
| `codigo_postal` | contacto | titular | código postal español `01000`–`52999` |

El certificado de la FNMT trae los dos apellidos juntos (atributo `surname`) y el
DNIe solo el primero, con los dos en el nombre común. Separarlos sería adivinar, así
que la ficha guarda un solo dato `apellidos` tal como lo acredita el certificado. Si
un día hace falta el primer apellido por separado (por ejemplo, al cargar CONVOCA), se
añadirá como campo nuevo con su fuente.

El código postal solo admite el formato español. Un domicilio extranjero necesitará
un campo de país: queda anotado para cuando el catálogo lo pida.

### Documento de identidad

- Tipo: `dni`, `nie`, `pasaporte` u `otro`. País: ISO 3166-1 alfa-2. DNI y NIE
  exigen país `ES` y letra de control correcta.
- Número normalizado: mayúsculas, sin espacios, guiones ni puntos.
- Enmascarado para pantalla (orientación AEPD de 2019, a confirmar por el DPD):
  DNI `***4567**`, NIE `****4567*`; pasaporte y otros, cuatro caracteres en las
  posiciones 4 a 7 si el número tiene al menos 8, y todo oculto si es más corto.
- El historial es de solo adición: un cambio de NIE a DNI añade un documento nuevo
  a la misma ficha (la fusión de fichas es un corte posterior).

## Cifrado

- AES-256-GCM en Go, nonce aleatorio de 12 bytes. PostgreSQL solo ve texto cifrado.
- Datos asociados de cada valor: `vec.aspirantes.valor.v1`, ficha, campo y versión.
  Del número de documento: `vec.aspirantes.documento.v1`, ficha y documento. Un
  sobre copiado a otra fila o a otra versión no se descifra.
- Cada sobre guarda la referencia de su clave. Las claves retenidas permiten leer lo
  cifrado antes de una rotación.
- Índice ciego: HMAC-SHA256 con clave propia sobre
  `vec.aspirantes.documento.indice.v1`, tipo, país y número normalizado. Rotar esta
  clave exige reindexar; mientras tanto el alta se cierra (igual que en «Mis correos»).
- Huella semántica de cada petición (para la repetición idempotente): HMAC con otra
  clave, nunca SHA-256 de datos adivinables.
- Las cuatro claves (cifrado, índice, huella y sus retenidas) llegan por el puerto
  `FuenteClavesAspirantes`, que la composición conecta con el material del portal
  externo (F1.1). El portal interno no las tiene.

## Autorización V3

Solo existe la superficie `externa_personal`. El recurso es la ficha propia de quien
actúa.

| Acción | Perfil V3 | Audiencia | Campos permitidos |
| --- | --- | --- | --- |
| `vec.aspirantes.ficha.consultar` | `aspirantes_ficha_consultar` | `vec_aspirantes.ficha.consultar.externa_personal.v1` | los ocho campos de la ficha |
| `vec.aspirantes.ficha.alta` | `aspirantes_ficha_alta` | `vec_aspirantes.ficha.alta.externa_personal.v1` | los ocho campos de la ficha |
| `vec.aspirantes.ficha.rectificar` | `aspirantes_ficha_rectificar` | `vec_aspirantes.ficha.rectificar.externa_personal.v1` | `codigo_postal`, `domicilio`, `movil`, `telefono`, `version` |

Los ocho campos: `apellidos`, `codigo_postal`, `documento`, `domicilio`, `movil`,
`nombre`, `telefono`, `version`.

- Módulo `aspirantes`, tipo de recurso `ficha_aspirante_propia`, finalidad
  `finalidad:aspirantes:ficha-propia:v1`, `recurso_ref` = persona que actúa.
- El material V3 (JSON que serializa `canonico`) lleva superficie, persona, perfil,
  acción, finalidad, versión esperada, clave de operación, huellas semánticas y el
  índice ciego del documento de la sesión. La decisión V3 queda ligada a ese índice
  por la huella del material. SQL lo recalcula todo y lo coteja.
- Migración `autorizacion_atestada_v3/000110_consumidor_aspirantes.up.sql`: añade
  los tres perfiles al núcleo con anclajes que no dependen de Usuarios (la principal
  no tiene AD3-106/107/108), exige que la sesión sea miembro exclusivo de
  `vec_aspirantes_ejecutor_externo` y registra las tres audiencias. El número 109 se
  deja libre para Usuarios por superficie (F1.3).

## SQL: esquema `vec_aspirantes`

Roles (DBA, una vez): `vec_aspirantes_propietario`, `vec_aspirantes_migrador` y
`vec_aspirantes_ejecutor_externo`. No hay ejecutor interno en este corte: las vistas
de RRHH tendrán su propio rol de lectura por finalidad.

Tablas, todas con RLS forzada y sin permisos directos para el ejecutor:

| Tabla | Contenido | Cambios |
| --- | --- | --- |
| `ficha` | `aspirante_ref`, versión, estado `activa`, fechas | versión y fecha |
| `valor` | ficha, versión, campo, estado (`presente` o `retirado`), origen, sobre cifrado | solo adición |
| `documento` | ficha, `documento_ref`, tipo, país, sobre del número, índice, origen, versión | solo adición |
| `indice_documento` | clave del índice, índice → ficha y documento; clave primaria única | solo adición |
| `historia` | ficha, versión, acción, motivo, campos cambiados, catálogo usado, recibo, decisión y auditoría V3 | solo adición |
| `recibo` | ficha y clave de operación, huella semántica, recibo, versión, decisión, auditoría y consumo V3 | solo adición |
| `acceso` | `aspacc_`, ficha, finalidad, tipo de actor, campos entregados, resultado, decisión y auditoría V3 | solo adición |
| `evento_salida` | `aspevt_`, ficha, tipo (`ficha.alta`, `ficha.contacto_rectificado`), versión, estado `pendiente` | solo adición |
| `contexto` | marcador de transacción tras consumir V3 (patrón de Usuarios) | se crea y se retira en la misma transacción |

El valor vigente de un campo es la fila de mayor versión. Retirar un dato añade una
fila `retirado` sin sobre. `evento_salida` solo lleva referencias opacas; lo leerá
Bolsa cuando pase a `asp_` (paso 8 del plan). No se inventa un relé antes.

Funciones (únicas con `EXECUTE` para el ejecutor, todas en `SERIALIZABLE READ WRITE`):

- `consultar_ficha_propia_v1(material, V3…)`: consume V3, busca por índice, anota el
  acceso con finalidad `consulta_propia` y devuelve sobres. Sin ficha devuelve
  `sin_ficha` y no anota acceso.
- `recuperar_ficha_operacion_v1(material, V3…)`: devuelve el recibo original de una
  clave ya usada, o nada.
- `alta_ficha_propia_v1(material, ficha, V3…)`: bloqueo por índice; si el índice ya
  existe, `P1411`. Crea ficha, valores, documento, índice, historia (motivo
  `alta_titular`), recibo y evento en una sola transacción.
- `rectificar_ficha_propia_v1(material, cambios, V3…)`: control de versión; añade
  valores nuevos o retirados, historia con motivo, recibo y evento.

Códigos: `P1409` conflicto (versión o clave reutilizada), `P1411` ficha ya existente,
`P1404` sin ficha, `22023` petición inválida, `42501` denegación, `40001` repetir.

## Aplicación

`ServicioFichaPropia` con tres casos de uso. Cada uno pide una V3 fresca ligada al
material exacto y comprueba el resultado del registro.

- **Consultar**: devuelve identidad (nombre, apellidos, tipo de documento y número
  enmascarado), los datos de contacto vigentes y lo que pide el catálogo. Sin ficha,
  devuelve la identidad del certificado para que la persona la vea antes de crearla;
  eso no se guarda.
- **Alta**: la identidad sale del certificado, nunca del JSON. La persona puede
  añadir en el mismo paso los datos de contacto que pida el catálogo. Una segunda
  alta del mismo documento devuelve «ya existe» y la pantalla recarga la ficha.
- **Rectificar**: solo campos que pida el catálogo; los obligatorios no se pueden
  vaciar. Motivo obligatorio de un vocabulario cerrado:
  `dato_nuevo` (no había valor), `cambio_de_dato` (ha cambiado) o
  `correccion_de_error` (estaba mal). La distinción importa: una corrección avisa de
  que los usos anteriores pudieron basarse en un dato erróneo (estudio, 6.3).

La persona, el perfil y la superficie salen del contexto de actor y del vínculo V2
certificado; la identidad acreditada sale del mismo certificado verificado de la
sesión. El JSON del navegador solo aporta operación, clave, versión, motivo y valores.

## Catálogo de datos personales (contrato con F1.2)

El catálogo está en `internal/vec/datospersonales` (PR #140) y su paquete de ejemplo
en `data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json`. Aspirantes no
lo lee directamente desde la aplicación: pregunta por un puerto propio.

```go
type ExigenciaCampo struct {
    Campo       domain.CampoFicha // telefono, movil, domicilio o codigo_postal
    Obligatorio bool
    Condicion   string            // código del catálogo; vacío si siempre
}
type ExigenciasContacto struct {
    CatalogoRef string           // catálogo, versión y huella; queda en la historia
    Campos      []ExigenciaCampo // vacío: no se pide ningún dato de contacto
    Ejemplo     bool             // paquete de ejemplo pendiente de RRHH y DPD
}
type CatalogoExigenciasFicha interface {
    ExigenciasContactoFichaPropia(ctx context.Context) (ExigenciasContacto, error)
}
```

El adaptador (corte 3) llama a `Resolutor.Para(ctx, tipo, MomentoInscripcion)` para
cada tipo de convocatoria configurado para el área personal. Mientras las
inscripciones no lleven `asp_` (paso 8 del plan), la lista de tipos es configuración
(por defecto, `bolsa`). Solo cuentan las entradas con custodia `aspirantes` y
categoría `ordinaria`. Correspondencia de datos:

| Dato del catálogo | Campo de la ficha |
| --- | --- |
| `telefono` | `telefono` |
| `telefono_secundario` | `movil` |
| `domicilio_notificacion` | `domicilio` y `codigo_postal` |

`obligatorio` en algún tipo hace obligatorio el campo; `condicional` y `voluntario`
lo dejan opcional y la condición viaja como código para que la pantalla la explique.
El correo sigue en Usuarios en este corte.

Si el catálogo falla, consultar sigue funcionando pero no se ofrece ningún dato de
contacto; el alta y la rectificación devuelven 503. Nunca se piden todos por defecto.

## API

| Método y ruta | Cuerpo | Respuesta |
| --- | --- | --- |
| `GET /api/vec/aspirantes/area-personal/mi-ficha` | — | `sin_ficha` con identidad del certificado, o ficha con versión, identidad, contacto y exigencias |
| `POST /api/vec/aspirantes/area-personal/mi-ficha` | `operacion` (`alta` o `rectificar`), `clave_operacion`, `version_esperada`, `motivo` (solo rectificar), `campos` | recibo: referencia, versión, fecha, repetición |

Cabeceras `no-store`, cuerpo máximo 4 KiB, campos exactos por operación. Errores:
401, 403, 404 (sin ficha al rectificar), 409, 422 y 503, sin revelar si un documento
tiene ficha fuera de la propia.

## Pantalla «Perfil y contacto»

- «Tus datos de identidad»: nombre y apellidos y documento enmascarado, con la
  indicación de que vienen del certificado y se corrigen en origen.
- «Teléfono y domicilio»: solo los campos que pide el catálogo, con marca de
  obligatorio. Si no pide ninguno, una frase lo explica.
- Al cambiar un dato que ya tenía valor se pregunta si ha cambiado o estaba mal.
- Sin ficha: se muestran los datos del certificado y un botón «Crear mi ficha».
- Textos en el catálogo i18n; la ayuda va en el botón «?».

## Seguridad

- Separación: esquema y roles propios; el ejecutor solo existe para el portal
  externo; ninguna clave foránea ni consulta cruza a otro módulo.
- Denegación por defecto en SQL (RLS por contexto de transacción tras consumir V3)
  y en HTTP (rutas exactas, cuerpo exacto).
- El proceso externo es la frontera de confianza del documento: toma nombre, apellidos
  y documento del certificado verificado de la misma sesión TLS. Quien comprometa ese
  proceso puede consultar fichas por índice o dar de alta una ficha con un documento
  ajeno; es el límite que el estudio (4.2 y 9) asume hasta tener gestor de claves y
  procesos separados.
- Regla de composición (corte 3): `IdentidadAcreditada` solo se construye en
  `adapters/certificado`, a partir de la hoja verificada de la misma conexión TLS que
  crea el vínculo V2 de la petición. Nunca desde cabeceras, JSON ni configuración. El
  vínculo V2 no lleva datos del certificado con los que cotejarla, así que no hay
  una segunda comprobación en la aplicación.
- La huella de un alta no incluye nombre ni apellidos: si el certificado cambia de
  nombre y se repite la misma clave de operación, se devuelve el recibo original.
- Reescribir el mismo valor de contacto con un motivo de cambio no se detecta (habría
  que descifrar dentro de la transacción). El motivo vale para toda la petición; la
  pantalla envía un cambio por formulario.
- Ningún dato en claro en errores, registros, historia, eventos ni auditoría.

## Pruebas previstas

- Dominio: DNI y NIE válidos e inválidos, normalización, enmascarado por tipo,
  teléfonos, código postal, vocabulario cerrado, referencias aleatorias.
- Aplicación con dobles: identidad del certificado frente a JSON, superficie
  interna rechazada, catálogo que no pide campos, obligatorio vaciado, motivo
  inválido, repetición con la misma clave, conflicto de versión, V3 ligada al
  material, errores del registro.
- PG18 efímero en `/dev/shm`: ACL y RLS reales, vector Go/SQL, alta, duplicado,
  repetición, conflicto, rectificación, retirada, inmutabilidad, acceso anotado.
- Ensayo de la lista SQL sobre el clon de la principal.

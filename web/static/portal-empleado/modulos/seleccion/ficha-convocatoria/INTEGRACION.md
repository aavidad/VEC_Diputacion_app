# Consulta exacta de convocatoria

La ficha consume `ports.LectorConvocatoriaExacta` mediante el caso de uso de
Selección. Bolsa conserva las bases y autoriza cada lectura por V3. Selección
presenta esa versión y su evidencia sin evaluar requisitos ni resolver el proceso.

El shell aporta la convocatoria y su secuencia exacta, la vuelta a su lista y
el idioma. No se extraen identidad, permisos ni credenciales de la URL.

```js
import { montarFichaConvocatoriaHTTP } from "./montaje.js?v=20261001-s1-ficha-v1";
const vista = montarFichaConvocatoriaHTTP(contenedor, {
  selector: seleccionActual,
  alVolver: volverALista,
});
// El handle ya permite desmontar durante la carga inicial.
await vista.ready;
// Al cambiar de versión:
await vista.consultar(otraSeleccionExacta);
// Al abandonar la pantalla:
vista.desmontar();
```

El montaje debe cargar `ficha.css` y los componentes y tokens comunes del portal.
`vista.js` también admite un lector inyectado con
`consultarExacta(selector, { signal })` y el objeto del catálogo común `textos`.
Los catálogos `textos/es/seleccion-ficha-convocatoria.json` y
`textos/en/seleccion-ficha-convocatoria.json` usan `cargarTextos` desde el
directorio común de textos. Conservan castellano e inglés.

La ruta interna fija es `POST /api/vec/seleccion/convocatorias/ficha`. El cuerpo
contiene únicamente `convocatoria_id` y `secuencia`. La respuesta válida contiene
`ficha` y `evidencia`, con nombres JSON explícitos. Se conserva la versión, la
huella, las referencias de documentos, los requisitos y el recibo de lectura.
La ficha no ofrece descarga: el contrato aporta referencias, sin bytes ni URL.

El adaptador HTTP requiere `ConfigFicha.Lector`, `ResolverContexto` y
`ValidarFrontera`. El montaje aporta las autoridades reales de canal, origen,
audiencia, identidad y correlación. Estas autoridades registran sus rechazos;
el lector registra y consume las decisiones V3 en su transacción. La presencia
de una función en la configuración no demuestra que esa autoridad esté montada.

Una denegación devuelve 403; una versión ausente, 404; una dependencia caída o
una respuesta incompatible, 503. La vista limpia los datos anteriores ante
carga o fallo y permite reintentar una caída. Las peticiones se cancelan al
cambiar de versión o desmontar. No se usa almacenamiento del navegador.

Las fases y las referencias de plaza, OEP y RPT siguen pendientes de sus fuentes.
La pantalla no inventa admisión, publicación, firma, calificación ni cobertura.
El montaje institucional, su autorización y el recorrido duradero se acreditan
por separado de las pruebas focales y de una vista con lector sintético.

Pruebas focales:

```sh
node --test web/static/portal-empleado/modulos/seleccion/ficha-convocatoria/ficha.test.mjs
go test ./internal/modules/seleccion/adapters/http
```

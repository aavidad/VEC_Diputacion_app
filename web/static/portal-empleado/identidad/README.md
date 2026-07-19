# Identidad interna compartida

Este directorio no define otro dominio de usuarios. `contexto-actor.js` es la
proyección frontend, cerrada y no autoritativa, del `ContextoActor` canónico ya
existente en `internal/vec/domain/contexto_actor.go`:

- `persona_ref` corresponde a `ContextoActor.PersonaRef`;
- `cuenta_ref` corresponde a `ContextoActor.Instantanea.CuentaRef`;
- `perfil_ref` corresponde a `ContextoActor.PerfilActivoRef`;
- actor, rol y ámbito son datos de presentación del contexto resuelto, no una
  fuente de permisos;
- las referencias son opacas: no se consulta ni se enlaza por un identificador
  civil, nombre o correo.

La autorización sigue perteneciendo al backend y es de denegación por defecto.
El navegador no debe decidir que una operación está permitida porque vea un rol
o un ámbito en esta proyección.

## Composición de la presentación

El punto de composición debe obtener primero la sesión que Bolsa ya usa y crear
una sola identidad:

```js
const contexto = crearContextoActorPresentacionDesdeSesion(DATOS_PANEL.sesion);
const proveedor = crearProveedorContextoActorFijo(contexto);
const identidades = compartirContextoActor(proveedor, ["bolsa", "cronos", "dietas"]);

iniciarBolsa({ contextoActor: identidades.bolsa });
iniciarCronos({ contextoActor: identidades.cronos });
iniciarDietas({ contextoActor: identidades.dietas });
```

Los tres valores son exactamente el mismo objeto inmutable. Cada módulo puede
usar `exigirContextoParaModulo(contextoActor, "cronos")` —con su propia clave—
para fallar cerrado si queda fuera del ámbito. No debe crear, clonar ni guardar
una identidad por módulo.

La lista de módulos no está compilada en este contrato. En producción procede
del ámbito resuelto por el backend y de sus manifiestos registrados; añadir un
módulo con una clave segura no exige modificar el núcleo de identidad. El
fixture enumera únicamente los módulos presentes en esta demostración.

`presentacion.js` es el único fixture: traduce los dos actores sintéticos que ya
declara `datos-presentacion.js`. No copia sus comodines de vistas u operaciones,
no persiste estado y no realiza comunicaciones.

## Sustitución productiva

En producción se elimina el adaptador `presentacion.js`, no el contrato que
consumen los módulos. La superficie interna autentica en el servidor mediante
la sesión corporativa reforzada (Kerberos y certificado según la política),
resuelve y registra de forma durable el `ContextoActor` canónico y devuelve su
proyección con:

- `demostracion: false`;
- referencias opacas emitidas por el backend;
- método primario `kerberos_ad` o `certificado` y garantía acreditada;
- revisión, perfil activo, ámbito y tiempo de resolución obtenidos de fuentes
  autoritativas.

El adaptador productivo valida la respuesta con
`validarYCongelarContextoActor` y expone `obtenerContexto()`. Bolsa, Cronos y
Dietas conservan así la misma dependencia y no cambian al sustituir el fixture.
La prueba de los factores, la vigencia de sesión, RBAC/ABAC y cada decisión de
acceso permanecen en el servidor; esta proyección solo sirve para composición y
presentación coherente.

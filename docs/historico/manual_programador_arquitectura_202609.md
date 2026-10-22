# Arquitectura de desarrollo: antecedente de septiembre de 2026

Referencia histórica extraída del manual anterior. Conserva el mapa técnico
del corte de septiembre; las rutas, plantillas y superficies actuales se
consultan en el [manual vigente](../manual_programador/README.md). Los relevos
y relatos de sesiones anteriores se mantienen únicamente en la historia de Git.

## Arquitectura real y propiedad

Esta sección describe la **composición de desarrollo** conservada. El ejecutable
de ese recorrido es [cmd/vec-server](../../cmd/vec-server/main.go).
Carga `config`, llama a `bootstrap.NewHTTPServerWithConfig` y, con el perfil
de desarrollo explícito, entra en la composición de desarrollo.

```text
Navegador: shell + módulo + cliente HTTP
  → servidor de desarrollo con certificado de cliente
  → lista de rutas y autorización de servidor
  → adaptador HTTP → caso de uso → dominio y puertos
  → adaptadores PostgreSQL / Bolsa / salida local
  → estado, historia y salida pendiente durables → recibo minimizado
```

No existe una base de negocio común que cualquier módulo pueda modificar:

- `internal/modules/contrataciontemporal/` posee el expediente y su procedimiento.
- `internal/modules/bolsa/` posee sus convocatorias, propuestas, órdenes y
  llamamientos. Contratación consume sus puertos y recibos; no escribe sus tablas.
- `internal/vec/` aporta contratos y autoridades compartidas: identidad,
  autorización, contexto del actor, documentos y evidencias.
- `internal/app/` conecta dependencias concretas y superficies.
- `internal/candidate/` es código heredado; no es un camino alternativo para
  duplicar el flujo de Contratación.

Dentro de un módulo: `domain` contiene reglas sin infraestructura;
`ports` declara contratos; `application` coordina dominio y puertos;
`adapters` traduce HTTP, SQL y proveedores. Un caso de uso no debe importar
PostgreSQL ni leer el DOM. El contrato de otro módulo se consume por su frontera,
no se reproduce con otra implementación equivalente.

La identidad sintética de desarrollo procede del canal validado y de sus
autoridades explícitas. No acredita Kerberos corporativo ni habilita producción.
No admitir cuenta, perfil o permisos declarados por el navegador.

## Dónde modificar y dónde conectar

| Necesidad | Fuente o directorio que debe inspeccionarse |
| --- | --- |
| Configuración y separación de conexiones | [config/postgresql_contratacion_temporal.go](../../config/postgresql_contratacion_temporal.go) y [config/config.go](../../config/config.go). |
| Arranque y selección del perfil | [bootstrap.go](../../internal/app/bootstrap/bootstrap.go) y [composicion_desarrollo.go](../../internal/app/bootstrap/composicion_desarrollo.go). |
| Conectar el procedimiento real | [contratacion_temporal_desarrollo.go](../../internal/app/bootstrap/contratacion_temporal_desarrollo.go); los archivos `contratacion_temporal_*_desarrollo.go` aportan cada dependencia. |
| Registrar rutas internas del módulo | [composicion/interna/contrataciontemporal/rutas.go](../../internal/app/composicion/interna/contrataciontemporal/rutas.go), función `NuevasRutas`, y su llamada en bootstrap. |
| Contrato HTTP y respuesta pública | [adapters/httpinterno](../../internal/modules/contrataciontemporal/adapters/httpinterno/): manejador, contrato cerrado y proyección del recibo. |
| Regla o coordinación funcional | [domain](../../internal/modules/contrataciontemporal/domain/), [application](../../internal/modules/contrataciontemporal/application/) y [ports](../../internal/modules/contrataciontemporal/ports/). |
| Persistencia y transacción | [adapters/postgres](../../internal/modules/contrataciontemporal/adapters/postgres/) y [deploy/postgresql/contratacion_temporal](../../deploy/postgresql/contratacion_temporal/). |
| Navegación y montaje web | [portal.js](../../web/static/portal-empleado/portal.js) y [portal-modulos-coordinador.js](../../web/static/portal-empleado/portal-modulos-coordinador.js). |
| Formulario, vista, contrato y cliente HTTP | [modulos/contratacion-temporal](../../web/static/portal-empleado/modulos/contratacion-temporal/): `formulario-*.js`, `vista*.js`, `contrato-*.js` y `cliente-http-*.js`. |
| Textos visibles | [portal-i18n.js](../../web/static/portal-empleado/portal-i18n.js); en el módulo, `i18n.js`, `i18n-expedientes.js` e `i18n-llamamiento.js`; backend compartido en [internal/shared/i18n](../../internal/shared/i18n/). |
| Texto de los documentos del expediente | [data/catalogos/contratacion-temporal/plantillas_documentos.v3.json](../../data/catalogos/contratacion-temporal/plantillas_documentos.v3.json) (versión 3; la v1 y la v2 siguen en `data/demo/plantillas/` porque la v1 es la preimagen instalada en el servidor de desarrollo): título, `parrafo.NN` con campos `{{campo}}` y bloques `{{?campo}}…{{/campo}}` / `{{!campo}}…{{/campo}}`, `modalidades`, `firmantes` y `requiere_accion`. `VEC_CT_PLANTILLAS_SOURCE_PATH` lo sustituye por otro fichero del mismo formato. La lista cerrada de campos y la validación al arrancar están en [plantillas_borrador.go](../../internal/modules/contrataciontemporal/adapters/informejuridico/plantillas_borrador.go) y [campos_borrador.go](../../internal/modules/contrataciontemporal/adapters/informejuridico/campos_borrador.go). |
| Publicar un recurso estático del recorrido | [web/produccion.manifest](../../web/produccion.manifest), leído por [estaticos_produccion.go](../../internal/app/server/estaticos_produccion.go). Revisar los otros manifiestos solo si cambia su superficie. |

Para ampliar una acción existente, seguir su cadena completa: contrato y
manejador → caso de uso y puertos → adaptador real → dependencia de bootstrap
→ ruta → cliente y formulario. Añadir una función o una ruta a `NuevasRutas`
no basta si la raíz no le entrega una implementación autorizada.

Ejemplo ya existente: selección llama a
`POST /api/vec/contratacion-temporal/llamamientos/seleccion` con
`expediente_ref`, `version_esperada` y `clave_idempotencia`.
El formulario de comunicación toma organización, llamamiento, versión y recibo
antecedente de la respuesta autenticada, no de referencias inventadas.

Cuando un nuevo import JavaScript devuelve `404`, comprobar el archivo y
todos sus imports transitivos contra el manifiesto. El servidor carga la lista
al arrancar: modificarla exige reiniciar el proceso para validar el resultado.
No quitar guardas que separan recursos reales y de presentación.

Conservar el shell y el tema compartidos: contexto del expediente, etiquetas,
confirmación explícita, estados de espera/error y recibo. No duplicar helpers,
CSS estructural ni reglas funcionales en la vista.

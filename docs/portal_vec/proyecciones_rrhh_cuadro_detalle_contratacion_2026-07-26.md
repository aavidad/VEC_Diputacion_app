# Proyecciones reales de cuadro y detalle de Contratación temporal

**Fecha:** 26 de julio de 2026
**Estado:** contrato técnico aprobado para implementación; acciones y
finalidades pendientes de ratificación funcional por RRHH y DPD
**Alcance:** primeras lecturas productivas de las pantallas remitidas por RRHH

## Resultado buscado

La misma interfaz de las diecisiete pantallas debe empezar a mostrar
expedientes reales sin incorporar el adaptador de presentación a producción.
El primer corte solo publicará hechos que ya formen parte del agregado durable:

- alta y solicitud del centro;
- análisis de RRHH;
- decisión de cobertura;
- asignación de unidad, cuando exista;
- actuaciones de solo adición realmente registradas.

Las fases de informe, fiscalización, llamamiento, formalización, GINPIX,
incorporación y seguimiento permanecerán ausentes hasta disponer de dominio,
persistencia y autorización propios. No se crearán tareas, documentos,
firmantes, responsables ni estados ficticios para rellenar la pantalla.

La revisión aplicó `admin-data-web`: se conserva el espacio administrativo
denso ya validado y se sustituye únicamente su fuente de datos. La primera
vista seguirá siendo el cuadro operativo, no una portada nueva.

## Separación de contratos

PostgreSQL no generará HTML, etiquetas localizadas ni el árbol visual de las
pantallas. Entregará una proyección neutral con referencias opacas, claves de
catálogo, versiones, estados e instantes. Aplicación:

1. revalidará identidad, perfil, organización, ámbito, acción y finalidad;
2. ejecutará lectura y registro de acceso dentro de una sola sesión sellada;
3. verificará que cada fila pertenece a la organización y al ámbito exactos de
   la capacidad concedida;
4. verificará el agregado, su versión observada y sus referencias de flujo y
   catálogo;
5. minimizará los campos según el alcance concedido;
6. entregará un DTO neutral al adaptador.

El cliente web resolverá etiquetas mediante i18n y compondrá el contrato visual
cerrado existente. Otro cliente —escritorio, CLI o MCP autorizado— consumirá la
misma API neutral sin depender de HTML.

## Rutas y entrada

Las consultas serán `POST` para impedir que referencias, filtros o cursores
terminen en URL, historial o registros ordinarios del proxy:

```text
POST /api/interno/v1/contratacion-temporal/cuadro/consultas
POST /api/interno/v1/contratacion-temporal/expedientes/consultas
```

Cuadro, cuerpo máximo 4 KiB:

```json
{
  "filtros": {
    "texto": "",
    "estado_clave": "",
    "fase_clave": ""
  },
  "paginacion": {
    "limite": 50,
    "cursor": ""
  }
}
```

En la versión inicial, `texto` solo buscará por prefijo de `numero_visible`.
No habrá búsqueda libre sobre personas, observaciones, documentos o
referencias internas. El límite estará entre 1 y 100.

Detalle:

```json
{
  "expediente_ref": "referencia-opaca",
  "version_observada": 3
}
```

La versión observada detecta una pantalla obsoleta; no concede acceso.
El valor cero identifica exclusivamente una primera carga sin versión previa.
Una versión distinta de cero deberá coincidir exactamente con la versión
recuperada o la respuesta se tratará como no confiable.
Expediente ausente, ajeno o no autorizado producirá la misma respuesta no
observable, sin revelar cuál de esas causas se dio.

Identidad, organización, perfil, roles, ámbito y autoridad no podrán aportarse
en el cuerpo, cookies, almacenamiento web ni cabeceras libres.

## Paginación y anti-enumeración

- orden estable por `actualizada_en DESC, expediente_ref DESC`;
- sin `OFFSET`;
- cursor opaco y cifrado autenticadamente, ligado a versión, ámbito, filtros,
  límite, última posición y caducidad máxima de cinco minutos;
- un cursor alterado, caducado o usado bajo otro ámbito falla como petición no
  válida;
- máximo de cien filas y presupuestos acotados de tiempo y tamaño;
- cada resumen conserva internamente la organización necesaria para validar el
  aislamiento, aunque esa referencia no tenga por qué publicarse al cliente;
- la aplicación vuelve a comprobar que todas las filas respetan el prefijo,
  estado y fase solicitados y el orden estable publicado;
- un ámbito de centro solo admite expedientes de ese centro y un ámbito de
  unidad solo los asignados a esa unidad;
- el cuadro nunca publica actor, contacto, DNI ni referencias personales;
- el detalle base tampoco publica actores, contacto u observaciones libres;
- las referencias internas de lectura y auditoría se validan en el servidor,
  pero no se serializan al cliente;
- los rechazos previos a PostgreSQL y las lecturas permitidas o vacías quedan
  registrados por sus fronteras correspondientes.

## Contenido mínimo del detalle

El detalle no expondrá el agregado de dominio completo. Usará DTO explícitos
que, cuando existan en el expediente durable, permitan alimentar la misma web
con:

- solicitud: centro, categoría, grupo/subgrupo, motivo y periodo;
- análisis: modalidad, causa, periodo, porcentaje de jornada, resultado de RC
  y coste previsto con su fuente cuando estén autorizados;
- cobertura: vía, procedimiento o bolsa y claves de las comprobaciones;
- asignación: unidad, instante y motivo de asignación;
- hitos: secuencia, versión, acción, fase, estado e instante.

Quedan excluidos contacto, responsable personal, actor, notificación, DNI,
observaciones, motivaciones o detalles de texto libre, documentos y recibos
internos. Cualquier ampliación exigirá una finalidad y una minimización
específicas; no se incorporará por reutilizar directamente una estructura del
dominio.

## PostgreSQL y mínimo privilegio

Se creará un rol NOLOGIN exclusivo de proyección y un LOGIN técnico miembro
únicamente de él. Dispondrá de:

- `CONNECT` a la base, sin `CREATE` ni `TEMP`;
- `USAGE` del esquema, sin `CREATE`;
- `EXECUTE` solo sobre las dos fachadas de proyección;
- ningún acceso directo a tablas, secuencias, vistas ni funciones de efecto;
- ningún `SET ROLE`, `BYPASSRLS`, superusuario, creación de roles, bases o
  replicación.

Las tablas conservarán `FORCE ROW LEVEL SECURITY`. Las fachadas
`SECURITY DEFINER` usarán `search_path=pg_catalog`, límites temporales y una
transacción serializable. Consumirán la decisión autorizada y registrarán la
lectura antes de devolver la respuesta. Un simple `SELECT` seguido de una
auditoría independiente no cumple el contrato.

La raíz deberá acreditar en una conexión física TLS `verify-full`, primario,
LOGIN, membresía y manifiesto exacto de privilegios antes de aceptar tráfico.
`Ping` no constituye evidencia suficiente. Además, cada operación adquirirá
una conexión, validará su configuración efectiva y su estado real y comenzará
la transacción sobre esa misma conexión. La liberará una sola vez después de
`COMMIT` o `ROLLBACK`; no podrá acreditar una conexión y ejecutar en otra.

El pool será dedicado y de composición controlada. Se rechazarán callbacks o
envoltorios capaces de cambiar host, TLS, identidad, sesión o conexión después
de la configuración acreditada. La prueba productiva incluirá una CA y un
certificado PostgreSQL de ensayo; el socket Unix sin TLS seguirá siendo solo
una prueba aislada mediante etiqueta de compilación.

## Vocabulario de autorización

Claves propuestas:

```text
contratacion_temporal.cuadro.consultar
contratacion_temporal.expediente.consultar
gestion_operativa_contratacion_temporal
tramitacion_expediente_contratacion_temporal
```

Las acciones ya forman parte del vocabulario de presentación, pero no son una
concesión. Las finalidades son propuestas y deberán publicarse en el catálogo
gobernado tras ratificación de RRHH y DPD. Hasta entonces la composición
productiva seguirá cerrada; no se introducirá una concesión temporal en código.

La capacidad tendrá una vigencia máxima de cinco minutos. En el ámbito de
organización, su referencia de ámbito deberá coincidir exactamente con la
organización autenticada. Contexto y capacidad se revalidarán también en el
instante durable del recibo de lectura; una consulta que termine después de su
caducidad no publicará datos.

## Orden verificable

1. contratos y puertos Go sellados;
2. migración PostgreSQL, roles, auditoría y cursores;
3. adaptadores PostgreSQL y acreditación del pool;
4. HTTP exacto y protegido;
5. fuente productiva `listar/obtener` en el cliente;
6. composición con identidad y PDP reales;
7. E2E navegador → HTTP → aplicación → PostgreSQL;
8. aceptación funcional de RRHH y revisión de DPD/Sistemas.

La capacidad no contará como productiva hasta completar el recorrido entero.

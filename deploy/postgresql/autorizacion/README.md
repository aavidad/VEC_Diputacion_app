# PostgreSQL del nucleo de autorizacion

Estado: primer adaptador durable implementado y probado, **no habilitado en la
composicion productiva**. Fecha de corte: 15 de julio de 2026.

Implementa exclusivamente `ports.FuenteAutorizacion` y
`ports.RegistroDecisionesAutorizacion` con `pgx` v5.10.0. No conecta HTTP, CLI,
MCP ni modulos de negocio y la aplicacion no ejecuta migraciones al arrancar.

## Contenido

- `roles_up.sql`: bootstrap DBA de una sola ejecucion para grupos tecnicos
  `NOLOGIN`, sin contrasenas.
- `migraciones/000001_autorizacion.up.sql`: esquema, invariantes, RLS,
  funciones cerradas y privilegios.
- `migraciones/000001_autorizacion.down.sql`: reversion destructiva del
  esquema, solo con copia y aprobacion.
- `../ejecucion_documental_v4/migraciones_autorizacion/000002_*`: evolucion
  del documento de decision de 30 a 31 claves, validacion cerrada del bloque
  autenticacion-actor y fuentes locales versionadas para revalidarlo.
- `roles_down.sql`: retirada final de grupos; falla si quedan membresias o
  dependencias.
- `probar_integracion.sh`: PostgreSQL efimero aislado, migracion ascendente,
  pruebas reales, migracion descendente y retirada de roles.

La prueba usa por defecto PostgreSQL 18.4 Bookworm con el indice OCI fijado a
`sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296`.
No busca, para ni modifica contenedores preexistentes: crea un nombre unico y
solo elimina ese contenedor mediante `trap`.

```bash
deploy/postgresql/autorizacion/probar_integracion.sh
```

También se pueden ejecutar las pruebas contra una instancia preparada:

```bash
export VEC_POSTGRES_TEST_FUENTE_DSN='postgresql://fuente:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_REGISTRO_DSN='postgresql://registro:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_ADMIN_DSN='postgresql://administrador_pruebas:...@host/base?sslmode=verify-full'
go test ./internal/vec/adapters/postgres -run TestIntegracionAutorizacionPostgreSQL -count=1
```

Esas variables solo existen para pruebas. No deben imprimirse, guardarse en el
repositorio ni reutilizarse en produccion.

## Aplicacion de migraciones

Se presupone una base dedicada. Un DBA ejecuta primero `roles_up.sql`. El
bootstrap toma un bloqueo transaccional para serializar ejecuciones concurrentes
y, antes de cualquier mutacion, aborta si ya existe uno solo de los cuatro
nombres reservados. No adopta ni corrige roles homonimos aunque sus atributos
parezcan validos: podrian conservar credenciales, ajustes, membresias,
privilegios o propiedad ajenos que un bootstrap no debe asumir ni revocar.

Por diseño no es idempotente: una segunda ejecucion falla. La comprobacion de
una instalacion existente debe hacerse con una auditoria separada y de solo
lectura. Una reconstruccion requiere la migracion descendente, retirar primero
las cuentas `LOGIN` y dependencias bajo procedimiento aprobado, ejecutar
`roles_down.sql` y solo entonces aplicar de nuevo el bootstrap. Este limite
evita tocar objetos extranjeros, incluso privilegios o propiedad situados en
otras bases del mismo cluster.

Después, una identidad `LOGIN` nominativa y temporal, miembro de
`vec_autorizacion_migrador`, aplica las migraciones por orden. La aplicacion no
recibe esa identidad.

Las identidades de ejecucion son cuentas `LOGIN` distintas por despliegue y
solo heredan lo necesario:

- `vec_autorizacion_fuente`: ejecutar la lectura de instantanea exacta;
- `vec_autorizacion_registro`: ejecutar exclusivamente el CAS de registro.

Ninguna recibe `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE` ni propiedad
sobre tablas. No son superusuario, no crean roles o bases, no replican y no
tienen `BYPASSRLS`. `PUBLIC` carece de uso del esquema y de ejecucion de sus
funciones; los privilegios predeterminados tambien quedan revocados.
El PDP no dispone de una funcion de lectura de decisiones. La futura lectura o
consumo pertenece al repositorio del efecto, con una operacion definidora
exacta, atestacion y consumo atomico; no al rol que registra.

RLS esta habilitada y forzada en todas las tablas. Solo existe una politica
positiva para el propietario `NOLOGIN`; las funciones `SECURITY DEFINER`
poseen `search_path` cerrado y exponen una operacion parametrizada cada una.
Versiones, controles y decisiones son append-only; los punteros solo avanzan y
el cambio de politicas exige actualizar el control de catalogo en la misma
transaccion.

## CAS implementado

`registrar_decision_si_vigente` bloquea y compara en una transaccion
serializable:

1. asignacion actual, principal, referencia y huella;
2. version y huella exactas del rol;
3. revision y huella del control de vigencia de esa version;
4. revision y huella del catalogo;
5. conjunto completo referencia/huella de politicas actuales;
6. forma canonica, listas, manifiestos, referencias, UTC microsegundo y
   vigencia contra `clock_timestamp()` de PostgreSQL.

La prueba aplica primero el contrato historico y despues la evolucion V2. Una
decision nueva incorpora exactamente el bloque `vinculo_autenticacion_actor`
de 25 campos, ligado a la persona, perfil, sesion, control, garantia y
ContextoActor. El registro comprueba su estructura y el disparador bloquea y
revalida los punteros locales actuales antes del `INSERT`; una revocacion
concurrente no puede atravesar esa transaccion.

El documento historico tiene una lista blanca cerrada de 30 claves y el actual
exactamente 31: las anteriores mas `vinculo_autenticacion_actor`. Todas son
obligatorias, aunque una lista o mapa sea vacio; no se admiten claves
adicionales. Booleanos, numeros, textos, arrays y objetos conservan su tipo JSON
exacto, por lo que valores como `"concedida":"yes"` no se convierten. Los
manifiestos solo contienen valores string SHA-256 y las revisiones son enteros
positivos del rango `uint64`.

`pg_column_size` limita cada decision a 512 KiB. Es un techo deliberadamente
amplio para una capacidad breve con hasta 512 entradas de catalogo, campos u
obligaciones habituales, pero impide usar el registro como almacen documental o
forzar filas TOAST patologicas. Si una instalacion alcanza el limite debe acortar
referencias o segmentar su catalogo; nunca elevarlo implicitamente desde datos de
entrada.

Cualquier ausencia, cambio, espera conflictiva o error falla cerrado. El
adaptador nunca devuelve SQL, DSN, nombres internos ni contenido personal en
sus errores. La tabla durable solo admite decisiones concedidas con codigo
`concedida`; el registro probatorio de denegaciones debe incorporarse como
puerto de auditoria separado y append-only.

La barrera durable rechaza cualquier asterisco, completo o parcial, en RBAC,
campos, obligaciones, finalidades y ambitos positivos. No existe
`global=["*"]`. Las politicas ABAC pueden conservar el comodin restrictivo
exacto porque solo reducen acceso; nunca conceden.

## Puerta productiva que sigue cerrada

El CAS demuestra frescura de la evidencia, pero no vuelve a ejecutar el PDP.
Eso es deliberado: duplicar RBAC/ABAC en SQL crearia dos semanticas y rompería
la portabilidad hexagonal. Con el contrato actual, la decision solo conserva
la huella del contexto; no contiene los ambitos y atributos necesarios para
rehacer `AsignacionPerfil.Cubre` y todas las restricciones ABAC.

Por ello una credencial de `vec_autorizacion_registro` robada podria clonar
una evidencia vigente, cambiar la tupla funcional y presentarla como otra
decision.

Eliminar la funcion general de lectura evita que ese rol recupere decisiones
ajenas, pero no cierra esta falsificacion: quien controle la llamada de
registro aun puede alterar el documento que recibe antes de ejecutar el CAS.

Hasta resolverlo se aplican estas prohibiciones:

- esa credencial solo pertenece a la identidad tecnica nominativa del PDP
  aislado; nunca al portal, a un modulo, a un trabajador general ni a HTTP;
- el adaptador no se monta en composicion productiva;
- una decision registrada no autoriza por si sola ningun efecto de negocio.

Antes de abrir la puerta se exige una atestacion criptografica versionada de la
decision completa, emitida con una credencial y clave separadas y verificada
dentro de la funcion definidora. No se presupone que `pgcrypto` verifique firmas
ni se aplica un fallback HMAC silencioso. Suite, representacion canonica,
gobierno de claves, rotacion, revocacion, pruebas y perfil candidato PostgreSQL
se especifican en
`docs/portal_vec/atestacion_criptografica_decisiones.md`.

## Punto de extension para consumo atomico

La segunda condicion pendiente es consumir/revalidar la decision dentro de la
misma transaccion que produce el efecto. El punto de extension no esta en HTTP
ni en este adaptador aislado, sino en cada repositorio PostgreSQL de negocio:

```text
BEGIN
  fijar contexto tecnico local
  verificar atestacion versionada
  bloquear decision por decision_ref
  comparar tupla funcional exacta y clock_timestamp() < valida_hasta
  revalidar asignacion + control de rol + catalogo completo
  insertar consumo unico decision_ref + operacion_ref + idempotencia
  escribir agregado + OCC + hecho + auditoria + outbox
COMMIT
```

La futura funcion definidora puede llamarse
`consumir_decision_para_efecto(decision_ref, operacion_ref, contexto, sello)`.
Debe devolver un resultado tipado, no detalles SQL, y afectar exactamente una
fila. Una decision mutadora es de un solo uso; la unicidad se impone en base.
Si el caso de lectura admite reutilizacion, necesita una politica explicita y
un registro append-only por cada uso, nunca un supuesto implicito.

El puerto/repositorio de negocio debe recibir la decision y el comando
completo para que no exista ventana entre comprobar y confirmar. No es seguro
llamar primero a este CAS, cerrar su transaccion y ejecutar despues el efecto.

## Fuentes y mantenimiento

- pgx v5.10.0, version estable comprobada el 15-07-2026:
  <https://pkg.go.dev/github.com/jackc/pgx/v5@v5.10.0>.
- PostgreSQL 18.4 corrige vulnerabilidades de versiones anteriores:
  <https://www.postgresql.org/about/news/postgresql-184-1710-1614-1518-and-1423-released-3297/>.

El resumen de imagen y la version de pgx se revisan con cada actualizacion de
seguridad; fijarlos hace reproducible una entrega, no autoriza a congelarlos.

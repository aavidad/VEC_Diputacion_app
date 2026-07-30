# Coordinación CT-000047C2.1a: rol selector de contexto RRHH

Fecha: 30 de julio de 2026.

Estado: **lista para producción acotada; no concede acceso funcional**.

## Responsabilidad única

Crear el grupo técnico `NOLOGIN`
`vec_contexto_actor_corporativo_rrhh_selector` que, en cortes posteriores,
recibirá exclusivamente la ejecución de la fachada nominal
`resolver_y_registrar_contexto_corporativo_rrhh_v1`.

Este corte no crea LOGIN, pool, función, migración, tabla, membresía ni acceso
a datos. Al terminar, el rol solo posee `CONNECT` sobre la base actual.

## Base y propiedad

- Rama base: `integracion/ct-o4-04e-20260726`.
- Decisión propietaria:
  `decision_contexto_corporativo_rrhh_ct_000047c2_2026-07-30.md`.
- El rol pertenece a la frontera `vec_contexto_actor_v1`.
- Identidad conserva sesión/cuenta; ContextoActor conserva selección,
  organización y recibo; Contratación temporal no posee este rol.

## Write-set exclusivo

```text
deploy/postgresql/contexto_actor_v1/
  roles_contexto_corporativo_rrhh_selector_v1_up.sql
  roles_contexto_corporativo_rrhh_selector_v1_down.sql
  probar_roles_contexto_corporativo_rrhh_selector_v1_pg18_4.sh
```

No se modifica ningún otro fichero. Dirección integrará y actualizará los
documentos transversales tras la revisión.

## Contrato del alta

El alta:

1. exige superusuario, PostgreSQL 18 o posterior, UTF-8 y base dedicada sin
   privilegios de `PUBLIC`;
2. toma un advisory lock nominal;
3. rechaza cualquier homónimo, incluso si parece correcto;
4. reacredita la topología, atributos, membresías, ajustes y ACL esenciales de
   los roles base y del esquema `vec_contexto_actor_v1`;
5. crea un solo rol `NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
   NOREPLICATION NOBYPASSRLS`, sin contraseña, caducidad ni ajustes;
6. fija una marca de propiedad inequívoca con `COMMENT ON ROLE`;
7. concede solo `CONNECT` en la base actual, sin opción de concesión;
8. postvalida atributos, marca, ausencia de membresías y la ACL exacta.

La reentrada falla cerrada y no adopta ni altera objetos preexistentes.

## Contrato de retirada

El `down`:

1. exige superusuario y toma el mismo advisory lock;
2. bloquea los catálogos necesarios para impedir carreras con `GRANT`;
3. reacredita atributos, marca, membresías, ajustes y ACL exacta;
4. rechaza cualquier función nominal C2 que ya dependa del rol, incluida
   cualquier sobrecarga con el mismo nombre;
5. rechaza dependencias, propiedades o ACL adicionales en cualquier base;
6. revoca exclusivamente el `CONNECT` creado por el alta;
7. elimina el rol sin `CASCADE` y comprueba que no quedan referencias.

No se añade un parámetro de «forzar» que permita borrar evidencia o
dependencias.

## Matriz PostgreSQL 18.4

El arnés usa la imagen fijada por digest ya adoptada en el proyecto y acredita:

- rechazo de base sin endurecer y de ejecutor no superusuario;
- homónimo exacto y hostil sin adopción;
- dos altas concurrentes: un único éxito y un único rol;
- atributos, contraseña, caducidad, comentario y ajustes alterados;
- ACL adicional en base actual/ajena, esquema, tabla, columna, secuencia,
  función o tipo;
- rol como grupo, miembro u otorgante, incluidas opciones `ADMIN`, `INHERIT`
  y `SET`;
- propiedad y dependencias SQL reales;
- funciones C2 nominales y sobrecargas que bloquean el `down`;
- carrera `down` contra `GRANT` sin residuo ni sustitución;
- `up → down → up`, OID nuevo, huella exacta y aislamiento funcional;
- cero lectura/escritura de tablas, ejecución de funciones, `CREATE`, `TEMP`,
  `MAINTAIN`, parámetros o `SET ROLE`;
- ausencia de alteración en los roles y objetos base;
- limpieza del contenedor y de temporales incluso ante fallo.

## Puertas

```text
bash -n deploy/postgresql/contexto_actor_v1/probar_roles_contexto_corporativo_rrhh_selector_v1_pg18_4.sh
shellcheck deploy/postgresql/contexto_actor_v1/probar_roles_contexto_corporativo_rrhh_selector_v1_pg18_4.sh
deploy/postgresql/contexto_actor_v1/probar_roles_contexto_corporativo_rrhh_selector_v1_pg18_4.sh
git diff --check
gitleaks detect sobre el corte
```

El arnés PostgreSQL debe pasar tres veces en total entre productor, revisores y
dirección. Productor, revisor e integrador son personas/agentes distintos.

## Fuera de alcance

- la fachada nominal de Identidad C2.1b;
- historia, puntero, publicación, revocación o recibo corporativo;
- LOGIN/pool y membresía consumidora;
- fachada pública de ContextoActor;
- PDP, Contratación temporal, HTTP, web o datos reales.

Este corte no aumenta las métricas funcionales ni habilita producción.

# Coordinación CT-000047M1.R: rol resolutor de motivos RRHH

Fecha: 30 de julio de 2026.

## Motivo

El rol histórico `vec_autorizacion_motivos_evaluador` conserva por diseño
`EXECUTE` sobre una función V2 genérica que recibe selectores de catálogo y
entrada. Reutilizarlo para M1.3 permitiría al mismo cliente evitar las dos
fachadas nominales de cuadro y detalle.

M1.3 queda condicionada a un rol nuevo y exclusivo. No se modifican los
scripts `roles_v2_up/down.sql` ya publicados.

## Resultado único

Crear el rol:

```text
vec_autorizacion_motivos_rrhh_resolutor
```

Será `NOLOGIN`, `NOSUPERUSER`, `NOCREATEDB`, `NOCREATEROLE`, `NOINHERIT`,
`NOREPLICATION`, `NOBYPASSRLS`, sin contraseña, caducidad, ajustes ni
membresías.

Su única ACL inicial será `CONNECT` no otorgable sobre la base VEC actual. No
recibe `USAGE` de esquema, ejecución de funciones ni acceso a tablas. M1.3
concederá después únicamente `USAGE` de `vec_autorizacion` y `EXECUTE` sobre
las dos fachadas nominales.

## Write-set exclusivo

```text
deploy/postgresql/autorizacion/
  roles_motivos_rrhh_resolutor_v1_up.sql
  roles_motivos_rrhh_resolutor_v1_down.sql
  probar_roles_motivos_rrhh_resolutor_v1_pg18_4.sh
```

Un productor posee esos tres ficheros. No edita roles históricos, migraciones,
Go, web ni documentación transversal.

## Instalación

El `up`:

1. abre una transacción y fija `search_path=pg_catalog`;
2. exige ejecutor superusuario;
3. adquiere el advisory exclusivo
   `vec_autorizacion:rol-motivos-rrhh-resolutor:v1`;
4. valida la base VEC dedicada, el propietario V1 y los roles V2 esperados;
5. rechaza cualquier homónimo, incluso si parece correcto;
6. crea el rol con los atributos exactos;
7. concede solo `CONNECT` mediante el propietario de la base;
8. postvalida atributos, comentario de procedencia, topología y ACL.

La segunda ejecución falla y no adopta ni repara. El rol evaluador V2 y sus
privilegios permanecen byte a byte sin cambios.

## Retirada

El `down` usa el mismo advisory y bloquea los catálogos de roles, membresías y
base en el orden ya acreditado por `roles_v2_down.sql`.

Antes de mutar:

- exige superusuario;
- revalida atributos, comentario, contraseña, caducidad y ajustes;
- rechaza cualquier membresía donde el rol sea grupo, miembro u otorgante;
- exige la ACL `CONNECT` exacta;
- rechaza las dos fachadas M1.3 y cualquiera de sus sobrecargas;
- no revoca ACL ni elimina dependencias ajenas.

Después revoca únicamente `CONNECT` y ejecuta `DROP ROLE` sin `CASCADE`. Toda
ACL, propiedad o dependencia no prevista hace fallar `DROP ROLE` y revierte
también la revocación.

## Matriz PostgreSQL 18.4

El runner debe acreditar:

- instalación limpia y huella exacta;
- reentrada rechazada sin cambios;
- homónimo exacto u hostil no adoptado;
- ejecutor no superusuario rechazado;
- rol V2 histórico sin ninguna modificación;
- solo `CONNECT`, sin `CREATE`, `TEMP`, esquema, tabla, tipo o función;
- `up → down → up` y segunda retirada rechazada;
- safe-down ante atributo, comentario, contraseña, caducidad o ajuste hostil;
- safe-down ante ACL adicional de base, esquema o función;
- safe-down ante membresía como grupo, miembro u otorgante;
- safe-down ante función M1.3, sobrecarga, propiedad o dependencia real;
- carreras `up/up`, `down/GRANT` y `down/000010` sin estado parcial;
- tras retirada limpia, ausencia total de rol, ACL, membresías y ajustes.

Las familias hostiles se prueban por separado para que una no oculte a otra.

## Orden

```text
despliegue:
roles_up → roles_v2_up → 000003…000009 → M1.R up → 000010

reversión:
000010 down → M1.R down → 000009…000003 down → roles_v2_down
```

## Puertas

```text
PostgreSQL 18.4: 3/3
bash -n
ShellCheck
git diff --check
máximo 800 líneas por fichero
Gitleaks
revisión independiente: P0=0 y P1=0
```

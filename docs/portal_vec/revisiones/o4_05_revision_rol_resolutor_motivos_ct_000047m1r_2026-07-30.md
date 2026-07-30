# Revisión final del rol resolutor de motivos CT-000047M1.R

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base estable | `b642bd7` |
| Candidato revisado | `1721572` |
| Commit integrado | `231648b` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Garantías verificadas

El corte crea `vec_autorizacion_motivos_rrhh_resolutor` como rol técnico
`NOLOGIN` exclusivo para M1.3. No hereda el acceso histórico con selectores
libres del evaluador V2.

Quedaron acreditados:

- atributos, comentario, contraseña nula, caducidad nula y ausencia de ajustes
  exactos;
- una única ACL inicial: `CONNECT` no otorgable en la base VEC;
- ausencia de membresías y de privilegios de esquema, tabla, columna, función
  o tipo;
- preservación byte a byte de los dos roles V2 y sus ACL;
- rechazo de reentrada, homónimos y ejecución por un principal no
  superusuario;
- retirada transaccional, sin `CASCADE`, que falla cerrada ante atributos,
  ACL, membresías, propiedad o dependencias no previstas;
- ausencia de OID, ACL, ajustes o membresías residuales después de la
  retirada;
- serialización causal de `up/up`, `down/GRANT ROLE` y
  `000010/down`.

La carrera inversa abre previamente las tres conexiones, hace que la retirada
posea los bloqueos de `pg_authid` y `pg_auth_members` mientras espera
`pg_database`, demuestra que bloquea directamente al `GRANT ROLE` y exige que
este termine de forma natural con SQLSTATE `42704` después de eliminar el rol.
No usa cancelaciones ni pausas como prueba de corrección.

## Evidencia reproducida

- productor: PostgreSQL 18.4 real, tres ejecuciones finales verdes;
- revisor independiente: PostgreSQL 18.4 real, una ejecución verde;
- dirección: PostgreSQL 18.4 real, una ejecución verde;
- `bash -n`, ShellCheck, `git diff --check` y límite de 800 líneas: verdes;
- Gitleaks sobre el commit: cero hallazgos;
- árbol del productor limpio y write-set limitado a los tres ficheros
  coordinados.

## Límites conservados

M1.R no concede `USAGE` ni `EXECUTE` y no resuelve motivos. M1.3 deberá crear
las dos fachadas nominales y otorgarlas únicamente a este rol. El cierre no
autoriza producción ni sustituye las conformidades de RRHH, DPD, Jurídico,
Sistemas y DBA.

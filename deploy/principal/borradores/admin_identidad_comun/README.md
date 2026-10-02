# Fuente nominal de identidad ADMIN para copias

CA22 e IS11 completan la base de #232 para resolver un administrador de copias por su cuenta privilegiada, su persona y su certificado. La asignación del perfil de copias se provisiona con una preimagen y una huella aprobada; su baja añade historia y avanza la revisión actual. Cada consulta vuelve a comprobar las cuatro referencias y sus versiones. IS11 enlaza la sesión común con esa asignación y la revalida antes de un efecto.

La secuencia de SQL nueva está en `deploy/principal/lista_sql_trabajo_codexe_admin_identidad_comun_20261002.txt`. Requiere antes CA19, AUT22, CA20, AUT23 e IS9 de #232. CA23 e IS12, que pertenecen al gestor de perfiles, consumen después el auxiliar propietario de contexto y conservan roles, asignaciones y sesiones propios.

La base no acredita por sí sola el certificado TLS ni consulta una CRL. Esas comprobaciones corresponden a la frontera ADMIN que recibe la petición. La revalidación SQL comprueba la sesión, la política y la edad máxima de la observación de revocación. Estas migraciones no activan copias ni perfiles sin sus consumidores y un recorrido con PostgreSQL.

La acreditación del LOGIN comprueba la membresía y los privilegios exactos en los esquemas de Contexto e Identidad. La revisión de permisos públicos de otros esquemas corresponde a la comprobación global de ACL del despliegue; CA22 no modifica esas concesiones.

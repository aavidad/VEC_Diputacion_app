# Administración de perfiles: AUT24 y AD150

Borrador de trabajo. No instalar ni incluir en listas de despliegue.

AUT24 fue reservada por F el 30/09 y cedida a E. AD137 queda sin usar. Dirección
reservó AD150 antes de escribir el consumidor. No se altera una migración instalada. Base revisada:
`d49e6adfcac7a1a68752c239efc6c68a4679191d` (#232). Mantener bootstrap de dos
personas, baja aprobada 2→1 y denegación de 1→0; el HTTP anterior no fija el umbral.

Orden causal: CA19 → AUT22 → AUT23 → CA20 → IS9 → AUT24 → CA23 → IS12;
AD150 va después de IS12 y de AD149 real publicado. AUT24 define funciones PL/pgSQL
que permanecen indisponibles hasta instalar el consumidor; no exige su existencia
al crear el contrato y así evita una dependencia circular. Dirección debe reconciliar este orden con
ORDEN_SQL_NUCLEO.md antes del ensayo. Los números no indican el orden causal.

Fachadas de once argumentos (material text y diez piezas V3):
`aplicar_acto_ordinario_admin_v1`, `proponer_acto_admin_v1` y
`cerrar_propuesta_admin_v1`, en `vec_autorizacion`, devuelven JSON.
La consulta interna `resolver_rol_administrable_v1(text)` devuelve un descriptor
sin personas ni facultad de publicar permisos. Las firmas conservan el contrato
con el adaptador PostgreSQL. El acto lleva el rol exacto y su huella; nunca una
lista de concesiones. El cierre no admite clase, operación ni participantes:
los reconstruye de la propuesta almacenada.

El rol técnico previsto es `vec_admin_perfiles_ejecutor`, sin LOGIN ni pertenencia
a propietarios. La aplicación recibe un LOGIN provisionado fuera de Git con una
única membresía INHERIT, sin SET ni ADMIN. Las fachadas CA20 y AD150 se conceden
únicamente al propietario de autorización; no al ejecutor ni al LOGIN.

Pendientes que impiden activar:

- Capturar las preimágenes reales de la definición V3 y su restricción de
  audiencias después de AD149 real publicado; no sustituirlas por una base aislada.
- Acreditar el enlace entre material V3 y la revalidación IS9 de certificado,
  revocación, audiencia y red. IS9 exige evidencias que no forman parte del DTO
  de negocio. No convertir constantes booleanas ni campos del cliente en esas
  evidencias. IS12 aporta el vínculo nominal de perfiles y AD150 exige su
  revalidación antes de devolver el consumo, también en recuperación.
- Provisionar dos administradores mediante el canal de huella+CAS aprobado por
  operador. No hay bootstrap HTTP ni semilla de personas en este borrador.
- Publicar los roles fijos de Dietas con unidad exacta mediante AUT27 y registrar
  su huella en el catálogo administrable. Sin esa publicación, no se asignan.
- Ensayo de transacción, ACL, concurrencia, replay, baja propia 2→1 y denegación
  1→0 en clon PG18 causal; después, dos revisiones del mismo hash.

El consumo V3 aporta la referencia de auditoría central. El registro de actos,
historia, outbox y recibos se escribe en la misma transacción; conserva referencias
sin copiar certificados, claves, nombres ni documentos de identidad.

La preimagen usa el esquema `vec.admin.perfiles.preimagen.v1`. Una sola rutina
propietaria obtiene cuenta, persona, perfil, vínculo, asignación, versión de rol,
control, continuidad y población administradora bajo bloqueo. La huella es
SHA256 de UTF-8 del JSONB que devuelve esa rutina. Incluye campos explícitos de
referencias, versiones, estados, vigencias y procedencias; no incluye el reloj
de consulta, una sesión, nombres ni certificados. El cliente trata la huella
como opaca. Go no vuelve a serializar ese JSONB para calcularla. La vigencia se
comprueba de nuevo al aplicar el cambio.

La preparación del formulario todavía necesita una lectura nominal autorizada
de esa preimagen. El borrador conserva la rutina interna sin EXECUTE runtime;
no abre consultas personales para facilitar el formulario.

La consulta propietaria `consultar_asignacion_admin_perfiles_v1(text)` sirve a
CA23 y devuelve versiones exactas sin conceder otro perfil. Su audiencia procede
del catálogo auditado `audiencia_administrativa`, cotejada con configuración
privada por el proveedor de sesión. El bootstrap debe conservar `cuenta_ref`,
`vinculo_ref` y `rol_huella_sha256` en las asignaciones.

Comprobaciones del borrador: `git diff --check` y Semgrep local sobre los dos
SQL, con métricas y comprobación de versión desactivadas. Las tres reglas
comprobaron concesiones a PUBLIC y pertenencia del runtime a propietarios:
cero hallazgos. Esta comprobación no acredita ACL efectivas, ejecución de
PL/pgSQL, transacciones, concurrencia ni recuperación. No se ejecutaron SQL
en un clon ni se abrió una PR.

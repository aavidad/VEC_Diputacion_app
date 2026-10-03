# Selector ADMIN sin perfil activo — IS14 y CA31

Este corte prepara el listado y la elección explícita de perfiles propios de
Administración de la aplicación. La cuenta se obtiene de IS9 mediante certificado
mTLS verificado, revocación vigente, host y política exactos. El listado no exige
perfil activo ni una decisión V3. La categoría procede del catálogo positivo AUT33;
los roles antiguos sin esa metadata quedan fuera. Sistemas continúa pendiente.

## Contrato para el adaptador

Las dos fachadas del esquema `vec_identidad_sesiones_v1` devuelven una sola fila
`resultado jsonb, auditoria_comun_ref text`:

- `listar_perfiles_admin_auditado_v1`: nueve argumentos de observación mTLS del
  contrato anterior, más `evento_ref` y `correlacion_ref`.
- `seleccionar_perfil_admin_auditado_v1`: esos nueve, `perfil_ref`,
  `revision_esperada`, `evento_ref` y `correlacion_ref`.

El listado contiene `revision`, `perfil_activo_ref` y `perfiles`. Cada perfil
contiene `perfil_ref`, `rol_version_ref`, `clave_i18n` y `categoria_admin`. La clave
`administracion.perfiles.rol` ya pertenece al catálogo; no se devuelve un nombre
visible escrito dentro de SQL. La selección contiene `perfil_ref`,
`seleccion_revision` y `seleccionada_en`.

La petición genera `evento_<32 hex>` nuevo en el adaptador confiable; la correlación
es `correlacion_<32 hex>` del contexto de petición. Ninguna cabecera libre concede
estos datos. Cada lectura y cada recuperación de selección dejan su evento común.
Una repetición semántica conserva revisión y fecha de elección; revalida identidad,
asignación, categoría y vigencia. Una revisión anterior a otro cambio se deniega,
incluso si el perfil actual vuelve a tener la misma referencia.

Las fachadas requieren `SERIALIZABLE READ WRITE` con `TimeZone=UTC`. El adaptador
valida el acuse y confirma la transacción antes de entregar datos o comunicar la
denegación. Una denegación posterior a resolver la identidad devuelve
`{"estado":"denegado","motivo_ref":"..."}` y su auditoría común. Un fallo de
identidad sin actor acreditado no fabrica un evento preperfil: su auditoría
corresponde a la frontera de autenticación. Un fallo técnico de AD171 o del COMMIT
impide devolver éxito. La aplicación debe tratar los códigos SQL de conflicto
sin atribuir persistencia a una transacción fallida. El código propio `VCA31`
identifica únicamente el CAS deliberado obsoleto o perdido de CA31. Todo `40001`
del motor, de CA20 o de la cadena AD171 se propaga; exige reintentar la transacción
completa y nunca se transforma en una denegación de negocio.

## Dependencias y acreditación técnica

Orden causal: AD171, CA31 e IS14. La lista está en
`deploy/principal/lista_sql_trabajo_codexk_identidad_preperfil_20261003.txt`.
AD171 pertenece a la rama del escritor de auditoría y debe estar en el árbol
integrado para ensayar esa lista. También se requieren IS9, CA20, AUT24 y AUT33
ya instaladas; no reaplicarlas. No se importan IS11, CA22, CA23 ni IS12.

Un listado favorable requiere además configuración privada gobernada para la
versión v4 en `vec_autorizacion.rol_administrable_exacto_v1`, con su huella,
audiencia, ámbitos fijos y vigencias auténticos, y una asignación actual de esa
versión. AUT24 crea el catálogo vacío; AUT33 publica categoría positiva de v4,
pero no aprovisiona esa configuración ni asignaciones. La configuración se
aporta por el procedimiento privado existente. Estas migraciones no insertan
permisos para conseguir un resultado favorable ni reclasifican v3.

El rol técnico nuevo es `vec_identidad_sesiones_v1_admin_preperfil`. Sólo tiene
CONNECT, USAGE en identidad y EXECUTE sobre las dos fachadas y el preflight
`acreditar_runtime_preperfil_admin_v1()`. Un LOGIN privado se incorpora a ese
único grupo con `INHERIT TRUE, SET FALSE, ADMIN FALSE`, sin privilegios directos.
No tiene acceso a tablas, a los escritores propietarios ni al append de AD171.

El canal privado aprovisiona una fila en
`vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1`: `identidad_login`,
`proceso`, `entorno`, `host_admin`, `audiencia`, `vigente_hasta`. `proceso` admite
`^[a-z][a-z0-9._-]{1,79}$`. No hay LOGIN ni configuración sembrados. El preflight
acredita la topología, las ACL exactas y la vigencia; devuelve `acreditada` y
`proceso`. La fecha de caducidad de esa configuración se incorpora al límite
común de identidad, certificado, CRL y perfiles, y se comprueba después de las
esperas de bloqueo y antes de devolver el resultado. Para sustituir un consumidor se usa otro LOGIN y otra fila; la fila
anterior permanece inmutable.

IS conserva la observación original en una tabla de sólo adición y construye el
JSON interno de AD171. La fuente es `observacion_admin_preperfil:<32 hex>` y su
SHA256 corresponde al documento de la observación. Nunca se copian certificados,
secretos ni cuerpos HTTP a la cadena. CA almacena la referencia común que recibe
IS, junto con la revisión y fecha; no fabrica un hash local como auditoría.

La selección CA31 aún no compone una sesión posterior con perfil. Esa composición
necesita su tarea propia; este corte no acredita ADMIN completo, una familia de
Sistemas, autenticación externa publicada ni un recorrido de navegador.

## Comprobaciones preparadas y pendientes

`pruebas_sql/000014_selector_admin_acl.sql` crea un LOGIN sintético dentro de
ROLLBACK. Comprueba preflight mínimo, ausencia de acceso a evidencia/tablas,
denegación por host ajeno o identidad ausente y rechazo al ampliar la ACL.
Resuelve los OID como DBA antes de cambiar al LOGIN y consulta los privilegios
mediante esos OID: el LOGIN conserva el cierre de USAGE de los esquemas de
auditoría y contexto. No se concede USAGE para facilitar la prueba.
Debe ejecutarse en el clon con PostgreSQL real tras instalar el árbol causal.
`pruebas_sql/000014_selector_admin_40001.sql` inyecta fallos de listado y de la
cadena dentro de ROLLBACK. Usa un LOGIN y observación sintéticos previamente
acreditados por el procedimiento privado; no crea identidades o asignaciones
fingidas. Comprueba que `40001` alcanza al consumidor sin intentar auditarlo
como denegación. `000014_selector_admin_config_caducidad.sql` comprueba la
caducidad del consumidor durante una demora del listado, con fuentes reales y
sin elección persistida. La prueba de la cadena requiere el perfil v4 gobernado real
del fixture para alcanzar ese punto.

El ensayo de Dirección sobre producto `2d1865fe0` instaló CA31/IS14 una vez en
el clon. La variante privada de la prueba de ACL por OID pasó, junto con la
recuperación tras reinicio. La prueba original fallaba al resolver el nombre de
una función en un esquema correctamente cerrado; el ajuste reproducible afecta
sólo a la prueba. Los casos 40001/caducidad quedaron omitidos porque no había
política IS9, vínculo, LOGIN y asignación v4 gobernados. No hay positivo del
consumidor con COMMIT acreditado. El acta está en la bitácora privada del ensayo.

Siguen pendientes los casos positivos con una asignación sintética gobernada del
catálogo AUT33: listado sin selección, elección CAS, replay con nueva auditoría,
revisión obsoleta, revocación, rollback del CAS y recuperación de esa elección tras
reiniciar. El script corregido por OID no se ha vuelto a ejecutar: conserva el
resultado de la variante equivalente del ensayo. El productor no ha conectado
PostgreSQL ni Go; Dirección coordina las ratificaciones sobre el nuevo hash.

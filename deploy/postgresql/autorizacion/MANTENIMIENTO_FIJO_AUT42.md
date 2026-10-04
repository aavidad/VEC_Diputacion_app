# AUT42 — mantenimiento aprobado del perfil fijo

La migración instala configuración y helpers privados; no publica Rol5. La
operación técnica `mantener_version_perfil_fijo_admin_v1(text,text)` compara el
SHA de los bytes del plan con la aprobación externa del LOGIN exclusivo. El
DBA prepara esa configuración fuera de Git: plan/preimagen/catálogo/SHA y
aprobación, entorno desarrollo y ventana. La CLI no concede permisos ni crea
esa aprobación.

Plan V1, diez campos cerrados:

```text
version = 1
operacion_ref = pmf_<22..124 caracteres opacos>
preparado_en, caduca_en = UTC con segundos
rol_origen_sha256
control_revision_esperada, control_huella_sha256
catalogo_sha256
rol_destino_doc
asignaciones[2] {
 perfil_ref, asignacion_origen_ref, asignacion_origen_sha256,
 persona_ref, cuenta_ref, vinculo_ref,
 cuenta_version, persona_version, perfil_version, vinculo_version,
 ambitos_fuente
}
```

`rol_destino_doc` debe ser copia exacta de Rol4, salvo versión5, publicador
`mantenimiento_operador:<LOGIN real>`, fecha igual a preparación y las cuatro
concesiones cerradas de los catálogos de denominación y usuarios. No se acepta un documento
arbitrario aunque su SHA esté aprobado. Los campos/vigencias/ámbitos originales
se conservan en ambas asignaciones v2; no se crea Persona, perfil o vínculo CA.

La preimagen revalida Rol4/control/categoría/gobierno y catálogo actuales, los
dos perfiles de Aplicación activos y su fuente CA. Organización y unidad se
cotejan por los puertos existentes de CA/Personal31; AUT no consulta sus tablas.
Las asignaciones no pueden tener historia revocada ni se amplía su vigencia.

El efecto une publicación Rol5/control/categoría/catálogo, versiones v2 de las
dos asignaciones, sellos de AUT24 y punteros CAS con AD183 en una transacción.
Sistemas y el bootstrap original permanecen en su historia. Los helpers
legados tienen dispatcher exacto4/5: la rama4 conserva el cuerpo literal; la
rama5 sólo admite catálogo5 heredado y controles propios, sin `version>=`.
No se modifica el núcleo AD ni se utiliza una tabla de auditoría propia.

La recuperación usa los sellos24 y el evento determinista AD183 para conservar
el acuse original. No repone un puntero revocado o cambiado después. Cada
invocación, incluido replay/rechazo/error gestionados, añade intento común.
Un fallo de append aborta todo; un fallo de efecto revierte el subbloque y deja
sólo el intento. La CLI comunica resultado únicamente después de COMMIT.

`validar_administrador_denominacion_persona_v1(d,m)` es privado para AD184:
rol5 fijo de Aplicación, actor/asignación/control vigentes y huellas exactas;
acción publicar/leer, campos `[denominacion]`/`[nombre_mostrar]`, obligación
`[auditar]`, módulo `vec`, tipo `persona_denominacion`, finalidad
`presentacion_persona`. Exige organización y unidad asignadas y el contexto
hash ligado a los bytes canónicos de M/procedencia. Los codecs puros de AD184
se resuelven al llamar; si faltan, el gate deniega. No hay ciclo de instalación:
42 crea el gate; 184 exige ese gate y aporta sus codecs. Ninguno concede
permisos por los datos JSON de la petición.

Orden causal: AUT33/34/37 y AD183 previas; después AUT42 y AD184/CA32 por sus
propietarios. La lista de esta rama sólo contiene la migración42. No se
reaplican SQL instaladas ni se ejecuta DOWN. Las actas SQL reales y dos revisiones
exactas las coordina dirección. La vigencia antigua del ensayo no se amplía:
el positivo se realiza en otro objetivo recuperado y aprobado si ha caducado.

Las acciones `administracion.usuarios.listar`/`consultar` tienen finalidad
`gestion_usuarios`, garantía `alto` y obligación `[auditar]`. Exigen organización
y unidad. Los campos cerrados de listado son
`[denominacion_version,perfiles,persona_ref,siguiente_cursor,unidad_ref]`; en
consulta se omite `siguiente_cursor`. `perfiles` sólo proyecta referencia,
versión de rol, versión CA, estado y vigencia. La denominación se autoriza
aparte por `vec.persona.denominacion.leer`; no se incluyen actos, certificados
ni historia. AUT43 aportará sus gates propios: los helpers legados siguen
admitiendo exactamente el catálogo heredado4/5, sin ampliar su ámbito.

Comprobación del 04/10 sobre el clon K posterior a AD183: UP42 y vector de
estructura terminaron con código0. La instalación conservó 2382 asignaciones,
65 punteros y las 6269 auditorías anteriores; Rol5 siguió ausente. Se conservaron
OID, propietario, ACL, configuración y volatilidad de los helpers legados.
El LOGIN mínimo sintético sin aprobación devolvió denegación sin recibo y sin
replay; COMMIT confirmó un intento nominal AD183, dejando 6270 auditorías.
El LOGIN de ensayo se retiró y no se sembraron aprobaciones favorables.

Las pruebas focales de la CLI pasaron en modo normal, race y vet con `-p 8` y
caché en disco. Gosec del paquete cambiado, Semgrep con reglas locales Go/SQL,
la guarda vecsilencio y diff terminaron sin hallazgos nuevos. Los catálogos
web pasaron siete comprobaciones Node. Se consultaron definiciones con gopls.

Todavía falta demostrar publicación, replay, reinicio y rollback de auditoría
en un objetivo fresco con plan aprobado. Las APP originales caducaron a las
03:26 UTC; se conservaron sus vigencias. Este ensayo de estructura y rechazo
no acredita todavía la actualización real de las dos asignaciones ni una
instalación en principal. Las dos revisiones independientes corresponden a
dirección antes de integrar.

La candidata posterior cierra la facultad de delegación del operador: exige
exactamente CONNECT, USAGE y EXECUTE de la fachada, los tres sin GRANT OPTION,
y ninguna ACL directa al LOGIN. Conserva el rechazo de CREATE/TEMP. El
vector `mantenimiento_perfil_fijo_000042_acl.sql` necesita el plan real aprobado
y su LOGIN en un objetivo fresco: parte de un permiso efectivo, prueba las
tres delegaciones dentro de savepoints y recupera el mismo recibo tras revertir
las ACL. Todo termina en ROLLBACK, sin fuentes ni aprobación fabricadas.
Esta guarda final aún no se ha reaplicado sobre el clon instalado anterior.

En el descriptor de usuarios, `proyeccion_perfiles.version` representa la
versión CA del perfil. Es la misma clave del DTO; no se publica `versionCA`.

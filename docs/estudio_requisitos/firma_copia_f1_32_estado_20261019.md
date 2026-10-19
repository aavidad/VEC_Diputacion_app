# F1-32: la copia local compone la firma. Hasta dónde se llegó (19/10)

Este documento es un relevo. No hay código nuevo en esta rama ni recorrido
acreditado. Recoge qué se comprobó, qué falta y por dónde seguir mañana.

## Qué se pretende

Que `deploy/principal/copia_local.py crear ... --modulos` deje la firma
compuesta sin pasos a mano, para que el paso 7b del caso 4.0 (dos firmas del
informe: técnico y Jefatura) funcione y el paso 8 (fiscalización de
Intervención) devuelva 201 con recibo en lugar de 409. Hacen falta:

- el circuito de RRHH (`VEC_CT_CIRCUITO_RRHH_SOURCE_PATH`);
- el plan de firma publicado por la vía gobernada (dos administradores);
- los firmantes con su certificado y su asignación;
- `VEC_CT_FIRMA_VEC_R5_ENABLED` y el verificador GrxFirma
  (`VEC_FIRMA_VERIFICACION_*`) encendidos.

## Qué quedó comprobado

1. Una copia nueva con `--modulos --segundo-rrhh --rrhh-original` se crea en
   unos 90 segundos.
2. Administración (kit H6 privado, fases H10 y gobierno del conjunto 6, que
   ya trae las audiencias de lote, plan de firma, cargo competencial y
   certificado nominal) arranca en esa copia con la unidad
   `unidad:desarrollo:rrhh`. Con esa unidad no hace falta crear un nodo de
   RRHH por SQL ni tocar disparadores de ámbito: cargos, plan, asignaciones y
   expediente comparten unidad.
3. Para que eso funcione, el kit H6 necesita tres ajustes privados: aceptar
   `unidad_ref` en la configuración de unidad, acotar la guarda de pools a
   los dos LOGIN externos y reintentar `verificar` del gobierno (la primera
   verificación falla a veces justo después de `aplicar`; a los 5 segundos
   pasa). El kit también fija la revisión candidata y debe volver a fijarse
   en cada cabeza.
4. `vec-admin` arranca sin los cuatro montajes de efectos. Sin ellos no se
   pueden publicar cargos, certificados ni el plan.

## Hallazgo para empezar mañana

El material de origen de la copia (`--rrhh-original`) ya trae
`identidad/firma-vec.json` con el plan `ct.plan.firma.hz39.codexp.ajuste1`
v1, cuatro firmantes y cuatro admisiones con retirada `2027-09-30`, y el kit
de presentación tiene un `preparar_firma.py` que lo copia a la copia. Si las
asignaciones, roles y certificados de ese respaldo siguen vigentes en la base
restaurada, el camino corto es:

1. copiar ese `firma-vec.json`, su plan y su CRL al material de la copia;
2. copiar `config/contratacion_temporal_circuito_rrhh_v2.json` y apuntar
   `VEC_CT_CIRCUITO_RRHH_SOURCE_PATH` a la copia privada;
3. levantar el verificador GrxFirma propio de la copia (`kit/grxfirma_p/` del
   kit de firma) y encender `VEC_FIRMA_VERIFICACION_*` y
   `VEC_CT_FIRMA_VEC_R5_ENABLED`;
4. reiniciar y aprobar la provisión de los perfiles fijos de RRHH
   (`firma_documento`, `lector_entrega_peticion`) con
   `VEC_CT_PROVISION_PERFILES_RRHH_APROBACION` y sus preimágenes;
5. comprobar que el preflight ofrece `certificado_vec` al técnico.

Lo primero es consultar en la copia la vigencia de esas cuatro asignaciones y
la publicación del plan. Si alguna caducó o no está, no se alarga por SQL: se
sigue el camino largo.

## Camino largo (si el respaldo no basta)

Es la receta de `deploy/principal/candidata_hz39/firma/LEEME.md`, ya probada
en H el 15/10 y el 18/10:

1. montajes de `vec-admin`: cuatro LOGIN `vec_adm_{lote,plan_firma,cargos,certificados}_<sufijo>`
   miembros exclusivos de su grupo, sus filas de origen de consumo y sus
   cuatro configuraciones (`VEC_ADMIN_{LOTE,PLAN_FIRMA,CARGOS,CERTIFICADOS}_CONFIG_FILE`);
   reiniciar `vec-admin`;
2. certificados de Jefatura, Diputación y Secretaría con
   `scripts/operacion/firma_ct_tres_pasos/certificados.py`;
3. semilla de identidad de los cuatro firmantes (personas y perfiles a 89
   días);
4. 16 roles por AUT53 con `cmd/vec-publicar-cargos-firma`, LOGIN técnico de
   dos horas que se cierra al acabar;
5. semilla final de asignaciones (87 días), semilla F1-63 (#1198) y
   `datos_empleados_firmantes.sql`;
6. selección de perfil de los dos administradores y 14 actos (cinco cargos,
   cinco enlaces, cuatro certificados; las fechas del descriptor de
   certificado llevan seis decimales);
7. plan `ct.plan.firma.hz39.operativa` v1 con `cmd/vec-plan-firma-validar
   preparar`: el administrador A lo crea y el B lo publica;
8. exportar admisiones, empaquetar el plan publicado y escribir
   `firma-vec.json`;
9. verificador, variables, reinicio y aprobación de perfiles fijos.

Ese orquestador debería vivir en `deploy/principal/` y llamarse desde una
orden `firma` de `copia_local.py`, con el verificador incluido en
`arrancar` y `parar`.

## Variables de las ocho

| Variable | Estado en la copia |
| --- | --- |
| `VEC_BOLSA_BORRADORES_ENABLED` | encendida |
| `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED` | apagada |
| `VEC_DOCUMENTOS_ENABLED` | encendida |
| `VEC_BOLSA_CESE_CT_ENABLED` | apagada |
| `VEC_PERSONAL_B2_GOBIERNO_ENABLED` | apagada |
| `VEC_CT_INCORPORACION_ACREDITADA_ENABLED` | encendida |
| `VEC_FIRMA_VERIFICACION_ENABLED` | apagada |
| `VEC_RRHH_AUDITORIA_ENABLED` | apagada |

## Límites

No se ejecutó ningún acto de firma ni el recorrido 1–8 en Chrome. No se
añadió SQL. La copia y los worktrees de este intento se borraron al cerrar.

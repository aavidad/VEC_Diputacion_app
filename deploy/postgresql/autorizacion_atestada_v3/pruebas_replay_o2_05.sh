#!/usr/bin/env bash

# Se carga desde probar_integracion_o2_05.sh después de definir sus auxiliares.

estado_agregado_o2_05() {
    local caso="$1"
    valor "SELECT
      (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3
        WHERE decision_ref='decision:ct:o205:${caso}')::text || ':' ||
      (SELECT count(*)
         FROM vec_contratacion_temporal.identidad_reserva_alta i
         JOIN vec_contratacion_temporal.reserva_alta_actual ra USING (ambito_hmac)
         JOIN vec_contratacion_temporal.reserva_alta_version rv
           ON rv.ambito_hmac=ra.ambito_hmac AND rv.revision=ra.revision
        WHERE i.expediente_ref='expediente:ct:o205:${caso}'
          AND rv.estado='confirmada') || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta
        WHERE expediente_ref='expediente:ct:o205:${caso}') || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta_version
        WHERE expediente_ref='expediente:ct:o205:${caso}' AND version=1) || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.actuacion_alta
        WHERE expediente_ref='expediente:ct:o205:${caso}' AND secuencia=1) || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta
        WHERE expediente_ref='expediente:ct:o205:${caso}') || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.outbox_alta
        WHERE expediente_ref='expediente:ct:o205:${caso}') || ':' ||
      (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta
        WHERE expediente_ref='expediente:ct:o205:${caso}')"
}

afirmar_agregado_completo_o2_05() {
    local caso="$1"
    [[ "$(estado_agregado_o2_05 "${caso}")" == '1:1:1:1:1:1:1:1' ]]
    [[ "$(valor "SELECT (
      SELECT count(*)=1
        FROM vec_contratacion_temporal.confirmacion_agregado_alta m
        JOIN vec_contratacion_temporal.reserva_alta_version rv
          ON rv.ambito_hmac=m.ambito_hmac
         AND rv.revision=m.reserva_revision
         AND rv.confirmacion_ref=m.confirmacion_ref
        JOIN vec_contratacion_temporal.expediente_alta e
          ON e.expediente_ref=m.expediente_ref
         AND e.confirmacion_ref=m.confirmacion_ref
        JOIN vec_contratacion_temporal.expediente_alta_version ev
          ON ev.expediente_ref=m.expediente_ref
         AND ev.version=m.version_expediente
         AND ev.confirmacion_ref=m.confirmacion_ref
        JOIN vec_contratacion_temporal.actuacion_alta ac
          ON ac.expediente_ref=m.expediente_ref
         AND ac.secuencia=m.actuacion_secuencia
         AND ac.confirmacion_ref=m.confirmacion_ref
        JOIN vec_contratacion_temporal.auditoria_alta au
          ON au.auditoria_ref=m.auditoria_ref
         AND au.confirmacion_ref=m.confirmacion_ref
        JOIN vec_contratacion_temporal.outbox_alta o
          ON o.evento_ref=m.evento_ref
         AND o.confirmacion_ref=m.confirmacion_ref
       WHERE m.expediente_ref='expediente:ct:o205:${caso}'
    )::text")" == 'true' ]]
}

comprobar_replay_sin_reparar_o2_05() {
    local caso="$1"
    local descripcion="$2"
    local antes
    local despues
    antes="$(estado_agregado_o2_05 "${caso}")"
    esperar_fallo "${descripcion}" invocar "${caso}"
    despues="$(estado_agregado_o2_05 "${caso}")"
    if [[ "${antes}" != "${despues}" ]]; then
        printf 'replay alteró agregado %s: %s -> %s\n' \
            "${caso}" "${antes}" "${despues}" >&2
        return 1
    fi
}

probar_canon_cerrado_o2_05() {
    local mutaciones=(
        'esquema|"vec.contratacion-temporal.efecto-alta.v3"'
        'reserva_ref|"reserva:ct:o205:alterada"'
        'expediente_ref|"expediente:ct:o205:alterado"'
        'numero_visible|"2026/alterado"'
        'recibo_ref|"recibo:ct:o205:alterado"'
        'organizacion_ref|"organizacion:otra"'
        'actor_ref|"per_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa"'
        'perfil_ref|"prf_sintetico_aaaaaaaaaaaaaaaaaaaaaaaaaa"'
        'version|2'
        'flujo.definicion_ref|"flujo:alternativo"'
        'flujo.version|2'
        'flujo.huella_sha256|"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"'
        'fase_actual|"validacion_inicial"'
        'estado_actual|"completado"'
        'solicitud.centro_ref|"centro:otro"'
        'solicitud.contacto_ref|"contacto:otro"'
        'solicitud.categoria_ref|"categoria:otra"'
        'solicitud.grupo_subgrupo|"C1"'
        'solicitud.motivo_clave|"sustitucion"'
        'solicitud.detalle|"Detalle alterado"'
        'solicitud.periodo.inicio|"2026-08-02"'
        'solicitud.periodo.fin|"2026-09-01"'
        'solicitud.rc.existe|true'
        'solicitud.rc.numero|"RC-2026-1"'
        'solicitud.rc.fecha|"2026-07-01"'
        'solicitud.rc.importe.centimos|1'
        'solicitud.rc.importe.moneda|"USD"'
        'solicitud.rc.documento_ref|"documento:rc:1"'
        'solicitud.documentos_adjuntos|["documento:adjunto:1"]'
        'solicitud.observaciones|"Observación alterada"'
        'creado_en|"2026-07-01T00:00:00.000000Z"'
        'actualizado_en|"2026-07-01T00:00:00.000000Z"'
        'actuacion.secuencia|2'
        'actuacion.version_expediente|2'
        'actuacion.accion_clave|"otra_accion"'
        'actuacion.actor_ref|"per_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa"'
        'actuacion.unidad_ref|"unidad:otra"'
        'actuacion.recibo_ref|"recibo:ct:o205:otro"'
        'actuacion.realizada_en|"2026-07-01T00:00:00.000000Z"'
        'actuacion.fase_origen|"inicio"'
        'actuacion.fase_destino|"validacion_inicial"'
        'actuacion.estado_origen|"en_curso"'
        'actuacion.estado_destino|"completado"'
        'actuacion.observaciones|"Observación alterada"'
        'actuacion.documentos_ref|["documento:actuacion:1"]'
    )
    local indice=0
    local mutacion caso ruta valor_json campo
    paso 'canon cerrado: cada campo mutable queda ligado al efecto autorizado'
    for mutacion in "${mutaciones[@]}"; do
        indice=$((indice + 1))
        caso="$(printf 'ef_%02d' "${indice}")"
        IFS='|' read -r ruta valor_json <<<"${mutacion}"
        preparar "${caso}"
        sql postgres \
            "SELECT public.mutar_efecto_o2_05('${caso}','${ruta}','${valor_json}'::jsonb)" \
            >/dev/null
        esperar_fallo "campo de efecto ${ruta}" invocar "${caso}"
        [[ "$(valor "SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:${caso}'")" == '0' ]]
    done
    paso 'tipos JSON numéricos estrictos, incluso con sobre canónico'
    for campo in version clave_version revision_gobierno \
        configuracion_secuencia raiz_version; do
        caso="tipo_${campo:0:8}"
        preparar "${caso}"
        sql postgres \
            "SELECT public.mutar_tipo_capacidad_o2_05('${caso}','${campo}')" \
            >/dev/null
        esperar_fallo "número serializado como texto: ${campo}" \
            invocar "${caso}"
    done
}

probar_integridad_replay_o2_05() {
    local casos=(
        der_reservada der_falsa der_res_version der_res_audit
        der_res_evento der_res_instante der_identidad der_expediente
        der_version_ausente der_version_huella der_act_ausente
        der_act_campo der_act_huella der_audit_ausente
        der_audit_decision der_audit_huella der_audit_eslabon
        der_out_ausente der_out_payload der_out_huella der_out_eslabon
        der_marker_ausente der_marker_huella der_cadenas
    )
    local caso
    paso 'agregados aislados para replay adversarial'
    for caso in "${casos[@]}"; do
        preparar "${caso}"
        invocar "${caso}" >/dev/null
        afirmar_agregado_completo_o2_05 "${caso}"
    done
    sql postgres "CREATE TABLE public.o205_respaldo_identidad AS
      SELECT * FROM vec_contratacion_temporal.identidad_reserva_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_actual AS
      SELECT ra.* FROM vec_contratacion_temporal.reserva_alta_actual ra
       JOIN public.o205_respaldo_identidad i USING (ambito_hmac);
      CREATE TABLE public.o205_respaldo_reserva AS
      SELECT rv.* FROM vec_contratacion_temporal.reserva_alta_version rv
       JOIN public.o205_respaldo_identidad i USING (ambito_hmac);
      CREATE TABLE public.o205_respaldo_expediente AS
      SELECT * FROM vec_contratacion_temporal.expediente_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_version AS
      SELECT * FROM vec_contratacion_temporal.expediente_alta_version
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_actuacion AS
      SELECT * FROM vec_contratacion_temporal.actuacion_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_auditoria AS
      SELECT * FROM vec_contratacion_temporal.auditoria_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_outbox AS
      SELECT * FROM vec_contratacion_temporal.outbox_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_marcador AS
      SELECT * FROM vec_contratacion_temporal.confirmacion_agregado_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      CREATE TABLE public.o205_respaldo_cadenas AS
      SELECT * FROM vec_contratacion_temporal.control_cadenas_alta" >/dev/null

    paso 'replay terminal expirado y no-cabeza conserva recibo exacto'
    if [[ "$(valor "SELECT clock_timestamp() >=
      (convert_from(capacidad,'UTF8')::jsonb->>'expira_en')::timestamptz
      FROM public.vectores_o2_05 WHERE caso='der_reservada'")" != 't' ]]; then
        sleep 6
    fi
    [[ "$(valor "SELECT clock_timestamp() >=
      (convert_from(capacidad,'UTF8')::jsonb->>'expira_en')::timestamptz
      FROM public.vectores_o2_05 WHERE caso='der_reservada'")" == 't' ]]
    invocar der_reservada >/dev/null

    paso 'puntero de reserva reservado o confirmado falso falla cerrado'
    sql postgres "UPDATE vec_contratacion_temporal.reserva_alta_actual ra
      SET revision=1 FROM vec_contratacion_temporal.identidad_reserva_alta i
      WHERE i.ambito_hmac=ra.ambito_hmac
        AND i.expediente_ref='expediente:ct:o205:der_reservada'" >/dev/null
    comprobar_replay_sin_reparar_o2_05 \
        der_reservada 'puntero actual a reserva todavía reservada'
    sql postgres "UPDATE vec_contratacion_temporal.reserva_alta_actual ra
      SET revision=m.reserva_revision
      FROM vec_contratacion_temporal.confirmacion_agregado_alta m
      WHERE m.ambito_hmac=ra.ambito_hmac
        AND m.expediente_ref='expediente:ct:o205:der_reservada'" >/dev/null
    sql postgres "SET session_replication_role='replica';
      INSERT INTO vec_contratacion_temporal.reserva_alta_version
        (ambito_hmac,revision,estado,version_expediente,auditoria_ref,
         evento_ref,confirmada_en,registrada_en,confirmacion_ref)
      SELECT ambito_hmac,99,'confirmada',9,'auditoria:falsa:o205',
             'evento:falso:o205',confirmada_en,confirmada_en,confirmacion_ref
        FROM vec_contratacion_temporal.confirmacion_agregado_alta
       WHERE expediente_ref='expediente:ct:o205:der_falsa';
      UPDATE vec_contratacion_temporal.reserva_alta_actual ra SET revision=99
       FROM vec_contratacion_temporal.identidad_reserva_alta i
       WHERE i.ambito_hmac=ra.ambito_hmac
         AND i.expediente_ref='expediente:ct:o205:der_falsa';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 \
        der_falsa 'puntero actual a confirmación falsa'
    sql postgres "UPDATE vec_contratacion_temporal.reserva_alta_actual ra
      SET revision=m.reserva_revision
      FROM vec_contratacion_temporal.confirmacion_agregado_alta m
      WHERE m.ambito_hmac=ra.ambito_hmac
        AND m.expediente_ref='expediente:ct:o205:der_falsa'" >/dev/null

    esperar_fallo 'reserva confirmada con NULL' sql postgres \
      "INSERT INTO vec_contratacion_temporal.reserva_alta_version
       (ambito_hmac,revision,estado,version_expediente,auditoria_ref,
        evento_ref,confirmada_en,registrada_en,confirmacion_ref)
       SELECT ambito_hmac,100,'confirmada',1,NULL,NULL,NULL,
              confirmada_en,NULL
         FROM vec_contratacion_temporal.confirmacion_agregado_alta
        WHERE expediente_ref='expediente:ct:o205:der_falsa'"

    paso 'refs e instante exactos de reserva'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.reserva_alta_version rv
         SET version_expediente=9
        FROM vec_contratacion_temporal.confirmacion_agregado_alta m
       WHERE rv.confirmacion_ref=m.confirmacion_ref
         AND m.expediente_ref='expediente:ct:o205:der_res_version';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_res_version \
        'versión de expediente divergente en reserva'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.reserva_alta_version rv
         SET auditoria_ref='auditoria:falsa:o205'
        FROM vec_contratacion_temporal.confirmacion_agregado_alta m
       WHERE rv.confirmacion_ref=m.confirmacion_ref
         AND m.expediente_ref='expediente:ct:o205:der_res_audit';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_res_audit \
        'referencia de auditoría divergente en reserva'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.reserva_alta_version rv
         SET evento_ref='evento:falso:o205'
        FROM vec_contratacion_temporal.confirmacion_agregado_alta m
       WHERE rv.confirmacion_ref=m.confirmacion_ref
         AND m.expediente_ref='expediente:ct:o205:der_res_evento';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_res_evento \
        'referencia de outbox divergente en reserva'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.reserva_alta_version rv
         SET confirmada_en=rv.confirmada_en+interval '1 second'
        FROM vec_contratacion_temporal.confirmacion_agregado_alta m
       WHERE rv.confirmacion_ref=m.confirmacion_ref
         AND m.expediente_ref='expediente:ct:o205:der_res_instante';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_res_instante \
        'instante confirmado divergente'

    paso 'identidad, expediente, versión y actuación fallan sin completar'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.identidad_reserva_alta
         SET recibo_ref='recibo:ct:o205:identidad:falsa'
       WHERE expediente_ref='expediente:ct:o205:der_identidad';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_identidad \
        'identidad de reserva divergente'
    sql postgres "SET session_replication_role='replica';
      DELETE FROM vec_contratacion_temporal.expediente_alta
       WHERE expediente_ref='expediente:ct:o205:der_expediente';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_expediente \
        'expediente ausente'
    sql postgres "SET session_replication_role='replica';
      DELETE FROM vec_contratacion_temporal.expediente_alta_version
       WHERE expediente_ref='expediente:ct:o205:der_version_ausente';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_version_ausente \
        'versión inicial ausente'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.expediente_alta_version
         SET alta_canonica=alta_canonica||convert_to(' ','UTF8'),
             huella_alta_sha256=encode(
               sha256(alta_canonica||convert_to(' ','UTF8')),'hex')
       WHERE expediente_ref='expediente:ct:o205:der_version_huella';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_version_huella \
        'canon y huella de versión divergentes'
    sql postgres "SET session_replication_role='replica';
      DELETE FROM vec_contratacion_temporal.actuacion_alta
       WHERE expediente_ref='expediente:ct:o205:der_act_ausente';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_act_ausente \
        'actuación inicial ausente'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.actuacion_alta
         SET accion_clave='PRUEBA_NO_SECRETO_ACCION_ALTERADA'
       WHERE expediente_ref='expediente:ct:o205:der_act_campo';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_act_campo \
        'campo de actuación divergente'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.actuacion_alta
         SET huella_sha256=repeat('e',64)
       WHERE expediente_ref='expediente:ct:o205:der_act_huella';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_act_huella \
        'huella de actuación divergente'

    paso 'auditoría, outbox, marcador y cadenas fallan cerrado'
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.auditoria_alta
         SET decision_ref='decision:ct:o205:falsa'
       WHERE expediente_ref='expediente:ct:o205:der_audit_decision';
      UPDATE vec_contratacion_temporal.auditoria_alta
         SET huella_sha256=repeat('d',64)
       WHERE expediente_ref='expediente:ct:o205:der_audit_huella';
      UPDATE vec_contratacion_temporal.auditoria_alta
         SET anterior_sha256=repeat('c',64)
       WHERE expediente_ref='expediente:ct:o205:der_audit_eslabon';
      UPDATE vec_contratacion_temporal.outbox_alta
         SET payload_canonico=payload_canonico||convert_to(' ','UTF8'),
             payload_huella_sha256=encode(
               sha256(payload_canonico||convert_to(' ','UTF8')),'hex')
       WHERE expediente_ref='expediente:ct:o205:der_out_payload';
      UPDATE vec_contratacion_temporal.outbox_alta
         SET huella_sha256=repeat('b',64)
       WHERE expediente_ref='expediente:ct:o205:der_out_huella';
      UPDATE vec_contratacion_temporal.outbox_alta
         SET anterior_sha256=repeat('a',64)
       WHERE expediente_ref='expediente:ct:o205:der_out_eslabon';
      UPDATE vec_contratacion_temporal.confirmacion_agregado_alta
         SET agregado_huella_sha256=repeat('9',64)
       WHERE expediente_ref='expediente:ct:o205:der_marker_huella';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_audit_decision \
        'decisión de auditoría divergente'
    comprobar_replay_sin_reparar_o2_05 der_audit_huella \
        'huella de auditoría divergente'
    comprobar_replay_sin_reparar_o2_05 der_audit_eslabon \
        'eslabón de auditoría divergente'
    comprobar_replay_sin_reparar_o2_05 der_out_payload \
        'payload y huella de outbox divergentes'
    comprobar_replay_sin_reparar_o2_05 der_out_huella \
        'huella de outbox divergente'
    comprobar_replay_sin_reparar_o2_05 der_out_eslabon \
        'eslabón de outbox divergente'
    comprobar_replay_sin_reparar_o2_05 der_marker_huella \
        'prueba durable divergente'

    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.control_cadenas_alta
         SET cabeza_auditoria_sha256=repeat('f',64);
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_cadenas \
        'cabeza de auditoría divergente'
    sql postgres "UPDATE vec_contratacion_temporal.control_cadenas_alta c
      SET secuencia_auditoria=x.secuencia,
          cabeza_auditoria_sha256=x.huella_sha256
      FROM (SELECT secuencia,huella_sha256
              FROM vec_contratacion_temporal.auditoria_alta
             ORDER BY secuencia DESC LIMIT 1) x
      WHERE c.control_id" >/dev/null
    sql postgres "SET session_replication_role='replica';
      UPDATE vec_contratacion_temporal.control_cadenas_alta
         SET cabeza_outbox_sha256=repeat('f',64);
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_cadenas \
        'cabeza de outbox divergente'
    sql postgres "UPDATE vec_contratacion_temporal.control_cadenas_alta c
      SET secuencia_outbox=x.secuencia,cabeza_outbox_sha256=x.huella_sha256
      FROM (SELECT secuencia,huella_sha256
              FROM vec_contratacion_temporal.outbox_alta
             ORDER BY secuencia DESC LIMIT 1) x
      WHERE c.control_id" >/dev/null

    sql postgres "SET session_replication_role='replica';
      DELETE FROM vec_contratacion_temporal.auditoria_alta
       WHERE expediente_ref='expediente:ct:o205:der_audit_ausente';
      DELETE FROM vec_contratacion_temporal.outbox_alta
       WHERE expediente_ref='expediente:ct:o205:der_out_ausente';
      DELETE FROM vec_contratacion_temporal.confirmacion_agregado_alta
       WHERE expediente_ref='expediente:ct:o205:der_marker_ausente';
      RESET session_replication_role" >/dev/null
    comprobar_replay_sin_reparar_o2_05 der_audit_ausente \
        'auditoría ausente'
    comprobar_replay_sin_reparar_o2_05 der_out_ausente \
        'outbox ausente'
    comprobar_replay_sin_reparar_o2_05 der_marker_ausente \
        'prueba durable ausente'

    paso 'restauración exacta de casos adversariales aislados'
    sql postgres "SET session_replication_role='replica';
      DELETE FROM vec_contratacion_temporal.confirmacion_agregado_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.outbox_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.auditoria_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.actuacion_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.expediente_alta_version
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.expediente_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      DELETE FROM vec_contratacion_temporal.reserva_alta_actual
       WHERE ambito_hmac IN (
         SELECT ambito_hmac FROM public.o205_respaldo_identidad
       );
      DELETE FROM vec_contratacion_temporal.reserva_alta_version
       WHERE ambito_hmac IN (
         SELECT ambito_hmac FROM public.o205_respaldo_identidad
       );
      DELETE FROM vec_contratacion_temporal.identidad_reserva_alta
       WHERE expediente_ref LIKE 'expediente:ct:o205:der\\_%';
      INSERT INTO vec_contratacion_temporal.identidad_reserva_alta
        SELECT * FROM public.o205_respaldo_identidad;
      INSERT INTO vec_contratacion_temporal.reserva_alta_version
        SELECT * FROM public.o205_respaldo_reserva;
      INSERT INTO vec_contratacion_temporal.reserva_alta_actual
        SELECT * FROM public.o205_respaldo_actual;
      INSERT INTO vec_contratacion_temporal.expediente_alta
        SELECT * FROM public.o205_respaldo_expediente;
      INSERT INTO vec_contratacion_temporal.expediente_alta_version
        SELECT * FROM public.o205_respaldo_version;
      INSERT INTO vec_contratacion_temporal.actuacion_alta
        SELECT * FROM public.o205_respaldo_actuacion;
      INSERT INTO vec_contratacion_temporal.auditoria_alta
        SELECT * FROM public.o205_respaldo_auditoria;
      INSERT INTO vec_contratacion_temporal.outbox_alta
        SELECT * FROM public.o205_respaldo_outbox;
      INSERT INTO vec_contratacion_temporal.confirmacion_agregado_alta
        SELECT * FROM public.o205_respaldo_marcador;
      UPDATE vec_contratacion_temporal.control_cadenas_alta c
         SET secuencia_auditoria=r.secuencia_auditoria,
             cabeza_auditoria_sha256=r.cabeza_auditoria_sha256,
             secuencia_outbox=r.secuencia_outbox,
             cabeza_outbox_sha256=r.cabeza_outbox_sha256,
             actualizada_en=r.actualizada_en
        FROM public.o205_respaldo_cadenas r WHERE c.control_id=r.control_id;
      RESET session_replication_role;
      DROP TABLE public.o205_respaldo_cadenas,
        public.o205_respaldo_marcador,public.o205_respaldo_outbox,
        public.o205_respaldo_auditoria,public.o205_respaldo_actuacion,
        public.o205_respaldo_version,public.o205_respaldo_expediente,
        public.o205_respaldo_reserva,public.o205_respaldo_actual,
        public.o205_respaldo_identidad" >/dev/null
    for caso in "${casos[@]}"; do
        afirmar_agregado_completo_o2_05 "${caso}"
    done
}

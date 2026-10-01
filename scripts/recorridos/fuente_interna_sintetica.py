#!/usr/bin/env python3
"""Build a private internal source proposal through an explicitly supplied RO clone.

This module has no connection CLI, provisioner, certificate issuer or runtime
output. Its caller must obtain the two exact sensitive reviews before supplying
the owned H6 SQL62 reader. A proposal never credits a certificate or a grant.
"""
from __future__ import annotations

from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import stat

from cryptography import x509
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.serialization import Encoding
from cryptography.x509.oid import ExtendedKeyUsageOID

from clon_sql import ReadOnlyDB
from fuente_sintetica_candidatos import private_root, revalidate_root, write_proposal

AUTHORITY = "direccion-vec:sintetico:20261001"
BASE_COMMIT = "268125ac2601f3dfce4306030cb91ea1dbf1fb7c"
ACT_SHA = "2a483496f7f5ea168917fb5aa4d67d991f0b85d72f44622b434d856ca1bfb84e"
MAX_FILE = 1024 * 1024
SOURCE_NAME = "fuente-sintetica-v1.json"
PRIVATE_ROOT = Path.home() / ".local/state/vec-codexm-h6-fuente-interna-20261001"
RO_SQL = "SELECT jsonb_build_object('transaction_read_only',current_setting('transaction_read_only'),'database',current_database())"
# One SELECT obtains a coherent statement snapshot. There is no arbitrary SQL
# supplied by the caller; ReadOnlyDB additionally forces BEGIN READ ONLY.
SNAPSHOT_SQL = """SELECT jsonb_build_object(
 'ro',current_setting('transaction_read_only'),
 'catalog',(SELECT jsonb_build_object('version',version,'revision',revision,
   'sha256',huella_sha256,'document',catalogo::jsonb)
   FROM vec_contratacion_temporal.organizacion_catalogo_revision
   ORDER BY version DESC,revision DESC LIMIT 1),
 'petitions',(SELECT coalesce(jsonb_agg(jsonb_build_object(
   'petition',p.peticion_ref,'version',p.version,'operation',p.operacion,
   'state',p.estado,'actor',p.actor_ref,'profile',p.perfil_ref,
   'center',p.centro_ref,'position',p.puesto_ref,'material_sha256',p.material_sha256,
   'assignment_ref',a.asignacion_ref,'assignment_version',a.version,
   'assignment_sha256',a.huella_sha256,'assignment',a.documento,
   'role',v.rol_id,'role_version',v.version,'role_sha256',v.huella_sha256,
   'control_revision',c.revision,'control_sha256',c.huella_sha256,'control_state',c.estado,
   'policies_revision',pol.revision,'policies_sha256',pol.huella_sha256)
   ORDER BY p.peticion_ref,p.version),'[]'::jsonb)
   FROM vec_contratacion_temporal.peticion_centro_revision p
   LEFT JOIN vec_autorizacion.asignacion_perfil_actual current_a ON current_a.perfil_activo_ref=p.perfil_ref
   LEFT JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=current_a.asignacion_ref
   LEFT JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=a.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual current_c ON current_c.version_rol_ref=v.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=current_c.version_rol_ref AND c.revision=current_c.revision
   LEFT JOIN vec_autorizacion.control_catalogo_politicas pol ON pol.control_id=true))"""
BLOCKERS = ["acuse_direccion_sha_pendiente", "provision_nominal_huella_cas_pendiente",
            "certificados_centro_pendientes", "vinculo_certificado_perfiles_h6_pendiente",
            "correspondencia_claves_privadas_no_comprobada"]


class InternalSourceError(RuntimeError):
    pass


def require(condition, code):
    if not condition:
        raise InternalSourceError(code)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode()


def unique(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate_json_key")
        result[key] = value
    return result


def decoded(data):
    require(isinstance(data, bytes) and 0 < len(data) <= MAX_FILE, "input_size")
    return json.loads(data, object_pairs_hook=unique)


def read_file(path):
    """Pin every path component and leaf; detect renames and byte changes."""
    path = Path(path)
    require(path.is_absolute() and ".." not in path.parts, "input_path")
    root_fd, ids = private_root(path.parent, create=False)
    try:
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=root_fd)
        try:
            before = os.fstat(fd)
            require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                    before.st_uid == os.getuid() and not before.st_mode & 0o077 and
                    0 < before.st_size <= MAX_FILE, "input_file")
            data = bytearray()
            while len(data) <= MAX_FILE:
                block = os.read(fd, min(65536, MAX_FILE + 1 - len(data)))
                if not block:
                    break
                data.extend(block)
            after = os.fstat(fd)
            leaf = os.stat(path.name, dir_fd=root_fd, follow_symlinks=False)
            fields = lambda s: (s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns, s.st_ctime_ns, s.st_nlink, s.st_mode)
            require(fields(before) == fields(after) == fields(leaf) and len(data) == before.st_size,
                    "input_changed")
            revalidate_root(path.parent, ids)
            return bytes(data)
        finally:
            os.close(fd)
    finally:
        os.close(root_fd)


def ref(value):
    require(isinstance(value, str) and re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", value), "reference")
    return value


def sha(value):
    require(isinstance(value, str) and re.fullmatch(r"[0-9a-f]{64}", value), "sha256")
    return value


def derived(subject, certificate_sha256):
    base = subject + "\0" + certificate_sha256
    def item(prefix, suffix):
        return prefix + digest(("vec.ct.alta.desarrollo.v1\0" + base + "\0" + suffix).encode())[:32]
    return {"cuenta_ref": item("cta_", "cuenta"), "persona_ref": item("per_", "persona"),
            "perfil_ref": item("prf_", "perfil"), "contexto_ref": item("vca_", "vinculo")}


def public_identity(identity_data, cert_data, ca_data, role):
    identity = decoded(identity_data)
    require(isinstance(identity, dict) and set(identity) == {
        "version", "autoridad", "subject", "certificate_sha256", "roles", "display_name"}, "identity_schema")
    require(identity["version"] == 1 and identity["autoridad"] == "no_autoritativo" and
            identity["roles"] == [role] and isinstance(identity["subject"], str) and
            identity["subject"].startswith("desarrollo:") and len(identity["subject"]) <= 159,
            "identity_contract")
    # Parse one public PEM certificate; private-key or multi-certificate material
    # is rejected before parsing. No private file is ever opened here.
    for pem in (cert_data, ca_data):
        require(pem.count(b"-----BEGIN CERTIFICATE-----") == 1 and
                pem.count(b"-----END CERTIFICATE-----") == 1 and
                b"PRIVATE" not in pem and pem.strip().startswith(b"-----BEGIN CERTIFICATE-----") and
                pem.strip().endswith(b"-----END CERTIFICATE-----"), "public_pem")
    cert = x509.load_pem_x509_certificate(cert_data)
    ca = x509.load_pem_x509_certificate(ca_data)
    try:
        cert.verify_directly_issued_by(ca)
    except (InvalidSignature, ValueError, TypeError) as error:
        raise InternalSourceError("public_certificate_issuer") from error
    now = datetime.now(timezone.utc)
    require(cert.not_valid_before_utc <= now < cert.not_valid_after_utc and
            ca.not_valid_before_utc <= now < ca.not_valid_after_utc and
            ca.extensions.get_extension_for_class(x509.BasicConstraints).value.ca and
            not cert.extensions.get_extension_for_class(x509.BasicConstraints).value.ca and
            ExtendedKeyUsageOID.CLIENT_AUTH in cert.extensions.get_extension_for_class(x509.ExtendedKeyUsage).value and
            "urn:vec:" + identity["subject"] in cert.extensions.get_extension_for_class(x509.SubjectAlternativeName).value.get_values_for_type(x509.UniformResourceIdentifier),
            "public_certificate_contract")
    cert_sha = digest(cert.public_bytes(Encoding.DER))
    require(cert_sha == sha(identity["certificate_sha256"]), "certificate_identity_mismatch")
    return {"sujeto": identity["subject"], "certificado": {"estado": "publico_h1_cotejado",
            "der_sha256": cert_sha, "pem_sha256": digest(cert_data), "clave_privada_cotejada": False},
            "referencias_derivadas_h1": derived(identity["subject"], cert_sha)}


def center_pair(snapshot):
    require(isinstance(snapshot, dict) and set(snapshot) == {"ro", "catalog", "petitions"} and snapshot["ro"] == "on", "snapshot_ro")
    catalog = snapshot["catalog"]
    require(isinstance(catalog, dict) and set(catalog) == {"version", "revision", "sha256", "document"} and
            catalog["version"] == 1 and catalog["revision"] == 4, "organization_preimage")
    sha(catalog["sha256"])
    rows = snapshot["petitions"]
    require(isinstance(rows, list) and len(rows) == 2, "petition_pair_missing_or_ambiguous")
    require([r["operation"] for r in rows] == ["presentar", "ratificar"] and
            [r["version"] for r in rows] == [1, 2] and
            [r["state"] for r in rows] == ["pendiente_ratificacion", "ratificada"], "petition_history")
    s, r = rows
    require(s["petition"] == r["petition"] and s["center"] == r["center"] and
            s["actor"] != r["actor"] and s["profile"] != r["profile"] and
            s["position"] != r["position"], "separation_of_duties")
    expected = {"petition", "version", "operation", "state", "actor", "profile", "center", "position",
                "material_sha256", "assignment_ref", "assignment_version", "assignment_sha256", "assignment",
                "role", "role_version", "role_sha256", "control_revision", "control_sha256", "control_state",
                "policies_revision", "policies_sha256"}
    for row, role in zip(rows, ("solicitante_centro", "ratificador_centro")):
        require(set(row) == expected and row["role"] == role and row["control_state"] == "habilitada", "published_assignment")
        for key in ("petition", "actor", "profile", "center", "position", "assignment_ref"):
            ref(row[key])
        for key in ("material_sha256", "assignment_sha256", "role_sha256", "control_sha256", "policies_sha256"):
            sha(row[key])
        for key in ("assignment_version", "role_version", "control_revision", "policies_revision"):
            require(type(row[key]) is int and row[key] > 0, "preimage_version")
        a = row["assignment"]
        require(isinstance(a, dict) and a["principal_id"] == row["actor"] and a["perfil_activo_ref"] == row["profile"] and
                a["estado"] == "activa", "assignment_actor")
        scopes = [x for x in a["ambitos"] if x["clave"] == "centro_ref"]
        require(len(scopes) == 1 and scopes[0]["valores"] == [row["center"]], "assignment_center")
        entries = catalog["document"]["entradas"]
        require(sum(e["clave"] == row["center"] and e["atributos"].get("tipo") == "centro" for e in entries) == 1 and
                sum(e["clave"] == row["position"] and e["atributos"].get("tipo") == "puesto_responsabilidad" and
                    e["atributos"].get("adscripcion_clave") == row["center"] for e in entries) == 1, "organization_binding")
    return rows


def prepare(ro, h1_root, act_path):
    require(type(ro) is ReadOnlyDB, "readonly_reader_required")
    act_bytes = read_file(act_path)
    require(digest(act_bytes) == ACT_SHA, "act_preimage")
    marker = b"\nEvidencia\n"
    require(act_bytes.count(marker) == 1, "act_evidence")
    evidence = decoded(act_bytes.split(marker)[1])
    require(evidence["installed_count"] == 62 and evidence["results"]["ro"] == {
        "database": "postgres", "transaction_read_only": "on"}, "act_sql62")
    clone = ro.system_identity()
    require(clone == evidence["clone_identity"] and
            ro.database_acl_digest() == evidence["clone_datacl_sha256"], "owned_clone_preimage")
    require(decoded(ro.query(RO_SQL)) == {"transaction_read_only": "on", "database": "postgres"}, "database_ro")
    snapshot = decoded(ro.query(SNAPSHOT_SQL))
    rows = center_pair(snapshot)
    require(snapshot["catalog"]["sha256"] == evidence["results"]["center_catalog"]["catalog_sha"], "catalog_changed")
    for row in rows:
        matches = [a for a in evidence["results"]["nominal_assignments"] if
                   a["profile_ref_sha256"] == digest(row["profile"].encode())]
        require(len(matches) == 1, "nominal_assignment_absent")
        match = matches[0]
        require(all(match[k] == v for k, v in {
            "role": row["role"], "role_version": row["role_version"], "role_sha": row["role_sha256"],
            "assignment_version": row["assignment_version"], "assignment_sha": row["assignment_sha256"],
            "assignment_active": True, "control_revision": row["control_revision"],
            "control_sha": row["control_sha256"], "control_state": "habilitada"}.items()), "assignment_changed")
    files = {"ca/ca.crt": read_file(Path(h1_root) / "ca/ca.crt")}
    people = []
    absences = []
    for key, identity_path, cert_path, role, name in (
        ("rrhh", "identidad/identidad.json", "mtls/cliente.crt", "tecnico_rrhh", "Lucía Robles Serrano"),
        ("intervencion", "identidad/intervencion.json", "mtls/intervencion.crt", "intervencion", "Andrés Vega Montes")):
        files[identity_path] = read_file(Path(h1_root) / identity_path)
        files[cert_path] = read_file(Path(h1_root) / cert_path)
        require(digest(files[identity_path]) == evidence["h1_public_files"][identity_path]["sha256"] and
                digest(files[cert_path]) == evidence["public_certificates"][key]["pem_file_sha256"] and
                digest(files["ca/ca.crt"]) == evidence["public_certificates"]["ca"]["pem_file_sha256"], "h1_preimage")
        identity = public_identity(files[identity_path], files[cert_path], files["ca/ca.crt"], role)
        absence = evidence["results"]["source_h1_" + key]
        require(set(absence) == {"derived_account_exists", "derived_account_projection_exists",
                               "derived_profile_assignment_exists", "derived_profile_context_exists"} and
                all(v is False for v in absence.values()), "h1_absence_preimage")
        # Reuse the act's closed derivation template, but pin it to its SHA and
        # only substitute references computed from the verified public identity.
        template = evidence["dynamic_query_templates"][key]["query_template"]
        query = template.replace("<CUENTA_DERIVADA_H1>", identity["referencias_derivadas_h1"]["cuenta_ref"]).replace(
            "<PERFIL_DERIVADO_H1>", identity["referencias_derivadas_h1"]["perfil_ref"])
        require(digest(query.encode()) == evidence["dynamic_query_templates"][key]["query_sha256"], "absence_query_preimage")
        require(decoded(ro.query(query)) == absence, "h1_absence_changed")
        absences.append((query, absence))
        people.append({"funcion": key, "nombre_visible_sintetico": name, **identity,
                       "perfil_publicado_h6": None, "preimagen_ausencia_h6": absence})
    for row, name in zip(rows, ("Marta Cárdenas Vidal", "Pablo Luque Ferrer")):
        people.append({"funcion": row["role"], "nombre_visible_sintetico": name,
            "sujeto_propuesto_desde_principal_h6": row["actor"], "sujeto_certificado": None,
            "certificado": {"estado": "pendiente", "der_sha256": None, "pem_sha256": None},
            "referencias_derivadas_certificado": None, "perfil_publicado_h6": row["profile"],
            "centro_ref": row["center"], "puesto_ref": row["position"], "preimagen_h6": row})
    subjects = [p.get("sujeto", p.get("sujeto_propuesto_desde_principal_h6")) for p in people]
    require(len(set(subjects)) == 4 and people[0]["certificado"]["der_sha256"] != people[1]["certificado"]["der_sha256"], "four_distinct_identities")
    # Detect concurrent publication before returning bytes. This is a measured
    # proposal preimage, not a transactional CAS guarantee for future effects.
    require(decoded(ro.query(SNAPSHOT_SQL)) == snapshot, "h6_preimage_changed")
    require(all(decoded(ro.query(q)) == absence for q, absence in absences), "h1_absence_changed")
    require(ro.system_identity() == clone and ro.database_acl_digest() == evidence["clone_datacl_sha256"], "owned_clone_changed")
    require(read_file(act_path) == act_bytes and all(read_file(Path(h1_root) / path) == data for path, data in files.items()), "source_changed")
    source = {"esquema": "vec.fuente.sintetica.interna.propuesta.v1", "version": 1,
              "estado": "propuesta", "acreditacion": None, "autoridad_maestra": AUTHORITY,
              "responsable": "Dirección VEC", "datos_sinteticos": True,
              "source_base_commit": BASE_COMMIT, "acta_preimagen_sha256": ACT_SHA,
              "organizacion_preimagen": snapshot["catalog"], "personas": people,
              "centros_propuestos": {"estado": "propuesta", "version": 1,
                  "solicitante": {"principal_h6": rows[0]["actor"], "perfil_h6": rows[0]["profile"],
                                  "centro_ref": rows[0]["center"], "puesto_ref": rows[0]["position"]},
                  "ratificador": {"principal_h6": rows[1]["actor"], "perfil_h6": rows[1]["profile"],
                                  "centro_ref": rows[1]["center"], "puesto_ref": rows[1]["position"]},
                  "ratificador_subject": None, "certificados_cotejados": False},
              "preimagen_ro_sha256": digest(encoded(snapshot)), "consulta_ro_sha256": digest(SNAPSHOT_SQL.encode()),
              "provision_ejecutada": False, "blockers": BLOCKERS}
    data = encoded(source)
    require(len(data) <= 64 * 1024, "proposal_size")
    return {SOURCE_NAME: data}


def write(root, outputs):
    require(Path(root) == PRIVATE_ROOT, "private_destination")
    require(set(outputs) == {SOURCE_NAME}, "output_schema")
    source = decoded(outputs[SOURCE_NAME])
    require(set(source) == {"esquema", "version", "estado", "acreditacion", "autoridad_maestra", "responsable",
        "datos_sinteticos", "source_base_commit", "acta_preimagen_sha256", "organizacion_preimagen", "personas",
        "centros_propuestos", "preimagen_ro_sha256", "consulta_ro_sha256", "provision_ejecutada", "blockers"}, "source_schema")
    require(source["estado"] == "propuesta" and source["acreditacion"] is None and
            source["provision_ejecutada"] is False, "proposal_only")
    write_proposal(Path(root), outputs)
    return {"estado": "propuesta", "acreditacion": None, "personas": 4,
            "fuente_sha256": digest(outputs[SOURCE_NAME]), "blockers": BLOCKERS}

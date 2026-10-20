"""Firmantes ficticios de una copia: fuentes propias, AUT53 y datos de Personal."""

from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import sys


ROLES = ("tecnico", "jefatura", "diputacion", "secretaria")
HEX = re.compile(r"[0-9a-f]{64}\Z")
NOMBRE = re.compile(r"[a-z][a-z0-9_-]{2,40}\Z")
H_SUCE = Path(__file__).parent / "sucesoras_roles.py"
H_SUCE_SHA = "0150fc925bf378346dbceb4cab3bac978e5bae98d9203a02a10e7b7224c9c1e1"
P_CONFIG = Path(__file__).parent / "config_aut53.py"


def _exigir(condicion, motivo):
    if not condicion:
        raise ValueError(motivo)


def _privado(ruta):
    ruta = Path(ruta)
    st = ruta.lstat()
    _exigir(ruta.is_absolute() and ruta.is_file() and not ruta.is_symlink()
            and st.st_uid == os.getuid() and not st.st_mode & 0o077,
            "fuente privada de firmantes ausente o insegura")
    return ruta


def _json(ruta):
    return json.loads(_privado(ruta).read_bytes())


def _poner(ruta, contenido):
    """Crear en exclusiva o cotejar los bytes idénticos de un reinicio."""
    ruta = Path(ruta)
    raw = contenido if isinstance(contenido, bytes) else (json.dumps(
        contenido, ensure_ascii=False, separators=(",", ":"), allow_nan=False) + "\n").encode()
    if ruta.exists():
        _exigir(_privado(ruta).read_bytes() == raw, "salida previa de firmantes divergente")
        return
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "wb") as out:
        out.write(raw)
        out.flush()
        os.fsync(out.fileno())


def _copiar(origen, destino):
    origen = Path(origen)
    _exigir(origen.is_file() and not origen.is_symlink(), "productor ausente")
    raw = origen.read_bytes()
    _poner(destino, raw)
    _exigir(hashlib.sha256(_privado(destino).read_bytes()).digest() == hashlib.sha256(raw).digest(),
            "productor privado divergente")


def _carpeta(ruta):
    ruta = Path(ruta)
    ruta.mkdir(mode=0o700, parents=True, exist_ok=True)
    _exigir(ruta.is_dir() and not ruta.is_symlink() and ruta.stat().st_uid == os.getuid()
            and not ruta.stat().st_mode & 0o077, "carpeta de firmantes insegura")
    return ruta


def _ejecutar(c, argv, log, *, cwd=None):
    resultado = c.ejecutar(list(map(str, argv)), log, cwd=cwd)
    _exigir(resultado.returncode == 0, f"falló {log}; conservar salida privada")


def _instante(valor):
    return datetime.strptime(valor, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)


def _ahora(c):
    valor = c.ahora()
    if isinstance(valor, str):
        valor = datetime.fromisoformat(valor.replace("Z", "+00:00"))
    _exigir(isinstance(valor, datetime) and valor.tzinfo is not None, "reloj UTC ausente")
    return valor.astimezone(timezone.utc).replace(microsecond=0)


def _limitar_asignaciones(sql, preparacion, retirada, ahora):
    desde = _instante(preparacion["roles_vigente_desde"])
    hasta = _instante(preparacion["roles_vigente_hasta"])
    fin = _instante(retirada)
    _exigir(timedelta(days=60) <= fin - ahora <= timedelta(days=90)
            and timedelta(days=60) <= fin - desde <= timedelta(days=90)
            and fin <= hasta and timedelta(days=60) <= hasta - desde <= timedelta(days=90),
            "autoridad y asignaciones fuera de 60 a 90 días")
    _exigir(sql.count(preparacion["roles_vigente_hasta"].encode()) == 32,
            "plantilla de asignaciones cambió; revisar sustituciones")
    salida = sql.replace(preparacion["roles_vigente_hasta"].encode(), retirada.encode())
    _exigir(b"2027-11-18T16:06:24Z" not in salida, "asignaciones de larga duración")
    return salida


def _roles(recibo, plan, cantidad):
    r, p = _json(recibo), _json(plan)
    filas = r.get("recibo", {}).get("cargos", [])
    _exigir(r.get("estado") == "permitido" and r.get("recibo", {}).get("auditoria_ref")
            and r.get("recibo", {}).get("plan_sha256") == hashlib.sha256(_privado(plan).read_bytes()).hexdigest()
            and len(filas) == cantidad and len(p.get("cargos", [])) == cantidad,
            "recibo AUT53 ausente o distinto del plan")
    plan_hash = {"rol:" + x["rol_id"] + ":v" + str(x["version"]): x["version_rol_sha256"]
                 for x in p["cargos"]}
    _exigir(len(plan_hash) == cantidad and all(
        HEX.fullmatch(x.get("version_rol_sha256", ""))
        and HEX.fullmatch(x.get("control_sha256", ""))
        and plan_hash.get(x.get("version_rol_ref")) == x["version_rol_sha256"] for x in filas),
        "huellas de roles AUT53 divergentes")
    return {x["version_rol_ref"]: x for x in filas}


def _confirmar_db(c, roles):
    """El acuse debe corresponder a versiones y controles activos de esta copia."""
    _exigir(roles and all(re.fullmatch(r"rol:[a-z0-9_:.-]+:v[12]", ref) for ref in roles),
            "referencias de roles inseguras")
    lista = ",".join("'" + ref + "'" for ref in roles)
    consulta = ("SELECT COALESCE(jsonb_object_agg(v.version_rol_ref,jsonb_build_object("
                "'huella',v.huella_sha256,'control',c.huella_sha256,'estado',c.estado)), '{}'::jsonb) "
                "FROM vec_autorizacion.version_rol v "
                "JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref) "
                "JOIN vec_autorizacion.control_vigencia_version_rol c "
                "ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision "
                f"WHERE v.version_rol_ref IN ({lista});")
    actual = json.loads(c.consultar(consulta))
    _exigir(set(actual) == set(roles) and all(
        actual[ref] == {"huella": row["version_rol_sha256"],
                        "control": row["control_sha256"], "estado": "habilitada"}
        for ref, row in roles.items()), "roles o controles de la copia divergen del recibo AUT53")


def _aut53(c, carpeta, fuente, prefijo, cantidad):
    """Plan, aprobación técnica declarada, LOGIN de dos horas y recibo propio."""
    cli = c.cli("vec-publicar-cargos-firma")
    textos = c.fuente / "web/static/textos/es/admin-cargos-firma.json"
    plan, acuse = carpeta / (prefijo + "-plan.json"), carpeta / (prefijo + "-acuse.json")
    if not plan.exists():
        _ejecutar(c, (cli, "preparar", "--cargos", fuente, "--plan", plan,
                     "--caduca", "2h", "--idioma", "es", "--textos", textos),
                 prefijo + "-preparar.log")
    p = _json(plan)
    _exigir(p.get("esquema") == "vec.admin.cargos-firma.plan.v2"
            and len(p.get("cargos", [])) == cantidad, "plan AUT53 inesperado")
    huella = hashlib.sha256(_privado(plan).read_bytes()).hexdigest()
    aprobacion = carpeta / (prefijo + "-aprobacion.json")
    declaracion = carpeta / (prefijo + "-declaracion.json")
    _poner(aprobacion, {"huella_plan_sha256": huella})
    _poner(declaracion, {"huella_plan_sha256": huella,
                         "alcance": f"Semilla técnica de copia ficticia {c.copia.nombre}: {cantidad} cargos {prefijo}",
                         "orden": "Dirección F1-131, 20/10/2026: excepción técnica AUT53; sin aprobación de dos administradores"})
    if not acuse.exists():
        login = "vec_f131_cargos_" + ("base" if cantidad == 16 else "consulta")
        _exigir(re.fullmatch(r"vec_[a-z0-9_]{3,55}", login), "LOGIN AUT53 inválido")
        clave = secrets.token_urlsafe(32)
        ventana = _ahora(c) + timedelta(hours=2)
        fin = ventana.strftime("%Y-%m-%d %H:%M:%S+00")
        referencia = "aprobacion:semilla:f131:" + prefijo
        if cantidad == 16:
            config_sql = carpeta / (prefijo + "-config-aut53.sql")
            if not config_sql.exists():
                _ejecutar(c, (sys.executable, carpeta / "config_aut53.py", "--plan", plan,
                             "--aprobacion", declaracion, "--aprobacion-ref", referencia,
                             "--login", login, "--salida", config_sql), prefijo + "-config-preparar.log")
            base = _privado(config_sql).read_text()
            _exigir(base.count("COMMIT;") == 1 and f"CREATE ROLE {login} LOGIN" in base
                    and f"'{huella}'" in base, "SQL AUT53 P divergente")
            sql = base.replace("COMMIT;", f"ALTER ROLE {login} PASSWORD '{clave}' VALID UNTIL '{fin}';\nCOMMIT;")
        else:
            # La fuente P exige 16 filas. Para las cuatro sucesoras se conserva
            # el mismo contrato AUT53 y se limita explícitamente la cardinalidad.
            sql = ("BEGIN;\n" +
                   f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;\n" +
                   f"GRANT vec_admin_cargos_firma_ejecutor TO {login} WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;\n" +
                   "GRANT CONNECT ON DATABASE postgres TO vec_admin_cargos_firma_ejecutor;\n" +
                   "INSERT INTO vec_autorizacion.config_cargos_firma_admin_v1 VALUES(" +
                   f"'{login}','{huella}','{referencia}','{hashlib.sha256(_privado(declaracion).read_bytes()).hexdigest()}'," +
                   "'desarrollo',clock_timestamp()-interval '1 minute',clock_timestamp()+interval '2 hours');\n" +
                   f"ALTER ROLE {login} PASSWORD '{clave}' VALID UNTIL '{fin}';\nCOMMIT;\n")
        c.sql(sql.encode(), prefijo + "-configuracion-aut53.log")
        puerto = c.copia.leer()["puerto_pg"]
        conexion = carpeta / (prefijo + "-conexion.json")
        _poner(conexion, {"dsn": f"host=localhost port={puerto} dbname=postgres user={login} password={clave} "
                          f"sslmode=verify-full sslrootcert={c.material / 'ca/ca.crt'}",
                          "permitir_socket_desarrollo": False})
        try:
            _ejecutar(c, (cli, "aplicar", "--plan", plan, "--conexion", conexion,
                         "--aprobacion", aprobacion, "--acuse", acuse, "--timeout", "30s",
                         "--idioma", "es", "--textos", textos), prefijo + "-aplicar.log")
        finally:
            conexion.unlink(missing_ok=True)
            c.sql(f"ALTER ROLE {login} NOLOGIN;\n".encode(), prefijo + "-cerrar-login.log")
    return _roles(acuse, plan, cantidad), plan, acuse


def _sucesoras(c, w, inicial, roles_v1):
    huellas = {}
    for tipo in ("firma", "consulta"):
        for role in ROLES:
            rid = f"ct_cargo_{role}_{tipo}_dev"
            ref = f"rol:{rid}:v1"
            _exigir(ref in roles_v1, "falta preimagen V1 de sucesora")
            huellas[rid] = roles_v1[ref]["version_rol_sha256"]
    origen = H_SUCE
    _exigir(hashlib.sha256(origen.read_bytes()).hexdigest() == H_SUCE_SHA,
            "productor H de sucesoras divergente")
    _copiar(origen, w / "preparar_fuentes_v2.py")
    _poner(w / "roles-v1-huellas.json", huellas)
    salida = w / "sucesoras"
    if not salida.exists():
        _carpeta(salida)
    if not (salida / "cargos-consulta-v2.json").exists():
        _ejecutar(c, (sys.executable, w / "preparar_fuentes_v2.py", "--plantilla",
                     inicial / "cargos-firma-fuente.json", "--roles-v1", w / "roles-v1-huellas.json",
                     "--salida", salida), "firmantes-sucesoras-preparar.log")
    fuente = salida / "cargos-consulta-v2.json"
    doc = _json(fuente)
    _exigir(len(doc.get("cargos", [])) == 4 and all(
        x["version"] == 2 and x["operaciones_v2"].count("consultar_detalle_selectores") == 1
        and "consultar_detalle" not in x["operaciones_v2"]
        and x["version_anterior_sha256"] == huellas[x["rol_id"]]
        for x in doc["cargos"]), "sucesoras consulta sin preimagen exacta")
    return _aut53(c, w, fuente, "consulta-v2", 4)


def _sql_sucesoras(c, w, datos, roles_v1, roles_v2):
    """Nueva versión de cuatro asignaciones; CAS contra la versión 1 exacta."""
    filas = []
    for role in ROLES:
        ref = datos["referencias"][role]
        perfil = ref["perfil"]["consulta"]
        viejo = f"rol:ct_cargo_{role}_consulta_dev:v1"
        nuevo = f"rol:ct_cargo_{role}_consulta_dev:v2"
        _exigir(viejo in roles_v1 and nuevo in roles_v2, "rol consulta ausente")
        filas.append({"perfil": perfil, "persona": ref["persona_ref"], "viejo": viejo,
                      "vieja_huella": roles_v1[viejo]["version_rol_sha256"], "nuevo": nuevo,
                      "nueva_huella": roles_v2[nuevo]["version_rol_sha256"]})
    _exigir(len({x["perfil"] for x in filas}) == 4, "perfil consulta repetido")
    payload = json.dumps(filas, separators=(",", ":"), ensure_ascii=False).replace("'", "''")
    sql = """BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:f131:sucesion-consulta',0));
DO $f131$
DECLARE f jsonb; antes vec_autorizacion.asignacion_perfil%ROWTYPE;
        doc jsonb; actual text; nueva text; h text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'F1-131: DBA requerido'; END IF;
 FOR f IN SELECT value FROM jsonb_array_elements('__PAYLOAD__'::jsonb) LOOP
  SELECT a.* INTO STRICT antes FROM vec_autorizacion.asignacion_perfil a
   JOIN vec_autorizacion.asignacion_perfil_actual p USING(perfil_activo_ref,asignacion_ref)
   WHERE p.perfil_activo_ref=f->>'perfil' FOR UPDATE OF p;
  actual := 'asignacion:'||antes.asignacion_id||':v1';
  nueva := 'asignacion:'||antes.asignacion_id||':v2';
  IF antes.asignacion_ref<>actual OR antes.version<>1 OR antes.principal_id<>f->>'persona'
     OR antes.version_rol_ref<>f->>'viejo' OR antes.documento->>'estado'<>'activa'
     OR antes.huella_sha256<>encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(antes.documento),'UTF8')),'hex')
     OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
        WHERE r.version_rol_ref=f->>'viejo' AND r.huella_sha256=f->>'vieja_huella')
     OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
        JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
        JOIN vec_autorizacion.control_vigencia_version_rol cv
          ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
        WHERE r.version_rol_ref=f->>'nuevo' AND r.huella_sha256=f->>'nueva_huella'
          AND cv.estado='habilitada')
  THEN RAISE EXCEPTION 'F1-131: preimagen consulta divergente'; END IF;
  doc := antes.documento || jsonb_build_object('version',2,'version_rol_ref',f->>'nuevo');
  h := encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex');
  INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,
    principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
  VALUES(nueva,antes.asignacion_id,2,antes.perfil_activo_ref,antes.principal_id,f->>'nuevo',h,antes.emitida_en,doc);
  UPDATE vec_autorizacion.asignacion_perfil_actual p SET asignacion_ref=nueva,
    actualizada_en=clock_timestamp(),actualizada_por='semilla:ensayo:datos-ficticios-f131',
    acto_ref='semilla:ensayo:f131:sucesion-consulta'
  WHERE p.perfil_activo_ref=f->>'perfil' AND p.asignacion_ref=actual;
  IF NOT FOUND THEN RAISE EXCEPTION 'F1-131: CAS de consulta perdido'; END IF;
 END LOOP;
END $f131$;
COMMIT;
""".replace("__PAYLOAD__", payload)
    _poner(w / "sucesoras-asignaciones.sql", sql.encode())
    c.sql(sql.encode(), "firmantes-sucesoras-asignaciones.log")


def preparar(c):
    """Preparar y aplicar exclusivamente la semilla técnica de firmantes."""
    w = _carpeta(c.carpeta / "firmantes")
    estado = w / "metadata.json"
    if estado.exists():
        meta = _json(estado)
        _exigir(meta.get("copia_ref") == c.copia.nombre
                and meta.get("organizacion_operativa_ref") == c.parametros["organizacion_operativa_ref"]
                and meta.get("firmantes_datos") == str(w / "semilla-final/firmantes-datos.json")
                and meta.get("roles_acuse") == str(w / "roles-acuse.json")
                and meta.get("consulta_v2_acuse") == str(w / "consulta-v2-acuse.json")
                and meta.get("entrada_admin") == str(c.raiz / "admin/h10/entrada-vec-admin.json"),
                "firmantes previos de otra copia")
        _privado(meta["firmantes_datos"])
        _exigir(timedelta(days=60) <= _instante(meta["autoridad_hasta"]) - _ahora(c) <= timedelta(days=90),
                "autoridad previa de firmantes ya no cubre el recorrido")
        _exigir(meta["roles_acuse_sha256"] == hashlib.sha256(_privado(meta["roles_acuse"]).read_bytes()).hexdigest()
                and meta["consulta_v2_acuse_sha256"] == hashlib.sha256(_privado(meta["consulta_v2_acuse"]).read_bytes()).hexdigest()
                and meta["unidad_h10_acuse_sha256"] == hashlib.sha256(_privado(c.raiz / "admin/h10/unidad/acuse-primero.json").read_bytes()).hexdigest(),
                "recibos previos de firmantes divergentes")
        return meta
    repo_semilla = c.fuente / "deploy/principal/candidata_hz39/firma/semilla_firmantes"
    fuente = _carpeta(w / "fuente")
    originales = _carpeta(w / "fuente-original")
    nombres = ("generar_semilla_p.py", "identidad.template.sql", "asignaciones.template.sql",
               "firmantes.template.json", "cargos-roles.template.json")
    huellas = {}
    for nombre in nombres:
        _copiar(repo_semilla / nombre, originales / nombre)
        huellas[nombre] = hashlib.sha256(_privado(originales / nombre).read_bytes()).hexdigest()
        if nombre != "firmantes.template.json":
            _copiar(repo_semilla / nombre, fuente / nombre)
    # Únicamente el dato ficticio de retirada cambia en el productor privado.
    # Se conservan los cinco originales y la guarda de plantilla del generador.
    dato_original = _json(originales / "firmantes.template.json")
    retiro_original = dato_original["retirada_en"]
    registro_fuentes = w / "fuentes-manifiesto.json"
    if registro_fuentes.exists():
        retiro_nuevo = _json(registro_fuentes)["retirada_nueva"]
    else:
        fecha = _ahora(c) + timedelta(days=88)
        retiro_nuevo = fecha.strftime("%Y-%m-%dT%H:%M:%SZ")
    dato_nuevo = {**dato_original, "retirada_en": retiro_nuevo}
    _poner(fuente / "firmantes.template.json", dato_nuevo)
    _poner(registro_fuentes, {"sha256_originales": huellas,
           "dato_transformado": "firmantes.template.json:retirada_en",
           "retirada_original": retiro_original, "retirada_nueva": retiro_nuevo,
           "sha256_resultante": hashlib.sha256(_privado(fuente / "firmantes.template.json").read_bytes()).hexdigest()})
    _copiar(P_CONFIG, w / "config_aut53.py")
    certificados = _carpeta(w / "certificados")
    if not (certificados / "certificados.json").exists():
        _ejecutar(c, (sys.executable, c.fuente / "scripts/operacion/firma_ct_tres_pasos/certificados.py",
                     "--aplicar", "PREPARA-CERTIFICADOS-CT", "--material", c.material,
                     "--salida", certificados, "--cert-dias", "397", "--crl-dias", "90"),
                 "firmantes-certificados.log")
    manifiesto = _json(certificados / "certificados.json")
    _exigir(manifiesto.get("cert_dias") == 397 and manifiesto.get("crl_dias") == 90
            and len(manifiesto.get("personas", [])) == 3, "certificados propios incompatibles")
    h10 = c.raiz / "admin/h10"
    plan_h10 = _json(h10 / "fuentes/plan.json")
    unidad_recibo = _json(h10 / "unidad/acuse-primero.json")
    unidad = unidad_recibo.get("recibo", {}).get("unidad", {})
    _exigir(unidad_recibo.get("estado") == "permitido" and unidad_recibo.get("recibo", {}).get("auditoria_ref")
            and unidad.get("organismo_ref", "").startswith("org_") and unidad.get("unidad_ref")
            and unidad.get("nodo_ref") and unidad.get("revision") == 1,
            "unidad H10 sin recibo propio permitido")
    procedencia = plan_h10["plan"]["procedencia"]
    _exigir(HEX.fullmatch(procedencia["huella_sha256"]), "procedencia H10 sin huella")
    operativa = c.parametros["organizacion_operativa_ref"]
    _exigir(isinstance(operativa, str) and operativa.startswith("organizacion:"), "organización operativa inválida")
    ref_copia = "f131-" + hashlib.sha256(c.copia.nombre.encode()).hexdigest()[:12]
    _exigir(NOMBRE.fullmatch(ref_copia), "referencia de copia no admite semilla")
    config = {"version": 1, "copia_ref": ref_copia,
              "organizacion_canonica_ref": unidad["organismo_ref"],
              "organizacion_procedencia_ref": procedencia["referencia"],
              "organizacion_procedencia_sha256": procedencia["huella_sha256"],
              "organizacion_operativa_ref": operativa, "unidad_ref": unidad["unidad_ref"],
              "nodo_ref": unidad["nodo_ref"],
              "hmac_g2_clave_archivo": str(c.material / "idempotencia/g2-huella-solicitud.bin"),
              "certificados": {"tecnico": str(c.material / "mtls/cliente.crt"),
                               **{role: str(certificados / ("rrhh_" + role) / "cliente.crt")
                                  for role in ROLES[1:]}},
              "acuse_unidad_h10": str(h10 / "unidad/acuse-primero.json")}
    _poner(w / "config-semilla.json", config)
    inicial = w / "semilla-inicial"
    if not inicial.exists():
        _ejecutar(c, (sys.executable, fuente / "generar_semilla_p.py", "--config",
                     w / "config-semilla.json", "--salida", inicial), "firmantes-semilla-inicial.log")
    # El productor deja datos de identidad hasta noviembre de 2027; esta copia
    # limita su autoridad a la retirada nominal (enero de 2027).
    identidad = _privado(inicial / "datos-identidad-firmantes.sql").read_bytes()
    retirada = _json(inicial / "firmantes-datos.json")["retirada_en"]
    _exigir(retirada == retiro_nuevo, "retirada de semilla no coincide con dato propio")
    _exigir(timedelta(days=60) <= _instante(retirada) - _ahora(c) <= timedelta(days=90),
            "retirada nominal fuera de 60 a 90 días")
    _exigir(identidad.count(b"2027-11-18T16:06:24Z") == 45
            and identidad.count(b"2026-10-12T00:00:00Z") == 44,
            "plantilla de identidad cambió; revisar sustituciones")
    inicio_identidad = _ahora(c).replace(hour=0, minute=0, second=0)
    identidad = (identidad.replace(b"2027-11-18T16:06:24Z", retirada.encode())
                .replace(b"2026-10-12T00:00:00Z", inicio_identidad.strftime("%Y-%m-%dT%H:%M:%SZ").encode()))
    _exigir(b"2027-11-18T16:06:24Z" not in identidad, "identidad de larga duración")
    marca = w / "identidad-aplicada.json"
    if not marca.exists():
        c.sql(identidad, "firmantes-identidad.log")
        _poner(marca, {"sql_sha256": hashlib.sha256(identidad).hexdigest()})
    else:
        _exigir(_json(marca)["sql_sha256"] == hashlib.sha256(identidad).hexdigest(), "identidad previa divergente")
    roles_v1, plan, acuse = _aut53(c, w, inicial / "cargos-firma-fuente.json", "roles", 16)
    _confirmar_db(c, roles_v1)
    final = w / "semilla-final"
    if not final.exists():
        _ejecutar(c, (sys.executable, fuente / "generar_semilla_p.py", "--config",
                     w / "config-semilla.json", "--salida", final, "--roles-recibo", acuse,
                     "--roles-plan", plan, "--roles-fuente-publicada",
                     inicial / "cargos-firma-fuente.json"), "firmantes-semilla-final.log")
    prep = _json(final / "preparacion.json")
    datos = _json(final / "firmantes-datos.json")
    _exigir(prep.get("roles_aut53_cotejados") is True and prep.get("certificados_cotejados") is True
            and datos["retirada_en"] == retirada and len(datos["firmantes"]) == 4,
            "semilla final sin cotejo de roles y certificados")
    asignacion = _privado(final / "semilla-asignaciones-firma.sql").read_bytes()
    asignacion = _limitar_asignaciones(asignacion, prep, retirada, _ahora(c))
    marca = w / "asignaciones-aplicadas.json"
    if not marca.exists():
        c.sql(asignacion, "firmantes-asignaciones.log")
        _poner(marca, {"sql_sha256": hashlib.sha256(asignacion).hexdigest(),
                      "autoridad_hasta": retirada})
    else:
        _exigir(_json(marca)["sql_sha256"] == hashlib.sha256(asignacion).hexdigest(), "asignaciones previas divergentes")
    roles_v2, plan_v2, acuse_v2 = _sucesoras(c, w, inicial, roles_v1)
    _confirmar_db(c, roles_v2)
    marca = w / "sucesoras-aplicadas.json"
    if not marca.exists():
        _sql_sucesoras(c, w, datos, roles_v1, roles_v2)
        _poner(marca, {"roles_acuse_sha256": hashlib.sha256(_privado(acuse_v2).read_bytes()).hexdigest(),
                      "sql_sha256": hashlib.sha256(_privado(w / "sucesoras-asignaciones.sql").read_bytes()).hexdigest()})
    # Personal exige inicio y fin a medianoche UTC y al menos 60 días restantes.
    marca = w / "empleados-aplicados.json"
    if marca.exists():
        previo = _json(marca)
        hoy = datetime.fromisoformat(previo["vigente_desde"].replace("Z", "+00:00")).date()
        fin = datetime.fromisoformat(previo["vigente_hasta"].replace("Z", "+00:00")).date()
    else:
        hoy = _ahora(c).date()
        fin = _instante(retirada).date() - timedelta(days=1)
    _exigir(timedelta(days=60) <= datetime.combine(fin, datetime.min.time(), timezone.utc) -
            datetime.combine(hoy, datetime.min.time(), timezone.utc) <= timedelta(days=90),
            "relación de Personal fuera de 60 a 90 días")
    valores = {"organismo_ref": unidad["organismo_ref"], "unidad_ref": unidad["unidad_ref"],
               "regimen_ref": "regimen:funcionario_interino", "modalidad_ref": "modalidad:vacante",
               "vigente_desde": f"{hoy.isoformat()}T00:00:00Z", "vigente_hasta": f"{fin.isoformat()}T00:00:00Z"}
    for role in ROLES:
        valores["persona_" + role] = datos["referencias"][role]["persona_ref"]
        valores["perfil_" + role] = datos["referencias"][role]["perfil"]["firma"]
    _exigir(all(re.fullmatch(r"[A-Za-z0-9:._-]+", x) for x in valores.values()), "parámetro Personal inseguro")
    cabecera = "\\set ON_ERROR_STOP on\n" + "".join(f"\\set {k} '{v}'\n" for k, v in valores.items())
    empleado = (cabecera.encode() + (c.fuente / "deploy/principal/candidata_hz39/firma/datos_empleados_firmantes.sql").read_bytes())
    if not marca.exists():
        c.sql(empleado, "firmantes-empleados.log")
        _poner(marca, {"sql_sha256": hashlib.sha256(empleado).hexdigest(),
                      "vigente_desde": valores["vigente_desde"], "vigente_hasta": valores["vigente_hasta"]})
    else:
        _exigir(_json(marca)["sql_sha256"] == hashlib.sha256(empleado).hexdigest(), "empleados previos divergentes")
    meta = {"copia_ref": c.copia.nombre, "semilla_copia_ref": ref_copia,
            "organizacion_operativa_ref": operativa,
            "unidad_ref": unidad["unidad_ref"], "firmantes_datos": str(final / "firmantes-datos.json"),
            "preparacion": str(final / "preparacion.json"), "roles_plan": str(plan), "roles_acuse": str(acuse),
            "consulta_v2_plan": str(plan_v2), "consulta_v2_acuse": str(acuse_v2),
            "roles_acuse_sha256": hashlib.sha256(_privado(acuse).read_bytes()).hexdigest(),
            "consulta_v2_acuse_sha256": hashlib.sha256(_privado(acuse_v2).read_bytes()).hexdigest(),
            "unidad_h10_acuse_sha256": hashlib.sha256(_privado(h10 / "unidad/acuse-primero.json").read_bytes()).hexdigest(),
            "certificados": str(certificados), "certificados_crl": str(certificados / "clientes.crl"),
            "entrada_admin": str(h10 / "entrada-vec-admin.json"),
            "autoridad_hasta": retirada, "roles_iniciales": 16, "sucesoras_consulta": 4,
            "alcance": "semilla técnica ficticia; sin aprobación de dos administradores ni firma"}
    c.guardar(estado, meta)
    return meta

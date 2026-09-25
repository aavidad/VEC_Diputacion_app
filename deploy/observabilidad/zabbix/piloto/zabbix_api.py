#!/usr/bin/env python3
"""Cliente mínimo de la API JSON-RPC de Zabbix 7.0 para el piloto M0.

Solo biblioteca estándar. Órdenes:
  esperar <url>                          espera a que la API responda
  preparar <url> <fichero_clave>         cambia la clave de Admin (aleatoria, en fichero 0600)
                                         y desactiva el host por defecto «Zabbix server»
  cargar <url> <fichero_clave> <plantilla.yaml> <host> <base_cgroup> <regex_cgroup>
                                         importa la plantilla VEC y crea el host del piloto
  panel <url> <fichero_clave> <host>     consulta que simula el adaptador de Administración (M4)
  verificar <url> <fichero_clave> <host> resume elementos, valores y visibilidad en JSON

No imprime credenciales. Los valores devueltos son técnicos (sin datos personales).
"""
import json
import os
import re
import secrets
import sys
import time
import urllib.request

CLAVE_INICIAL = "zabbix"  # clave por defecto de la imagen; se sustituye al preparar


def llamar(url, metodo, params, token=None, tiempo=20):
    cuerpo = json.dumps({"jsonrpc": "2.0", "method": metodo, "params": params, "id": 1}).encode()
    cab = {"Content-Type": "application/json-rpc"}
    if token:
        cab["Authorization"] = "Bearer " + token
    pet = urllib.request.Request(url, data=cuerpo, headers=cab)
    with urllib.request.urlopen(pet, timeout=tiempo) as r:
        resp = json.load(r)
    if "error" in resp:
        raise RuntimeError(f"{metodo}: {resp['error'].get('message')} {resp['error'].get('data')}")
    return resp["result"]


def entrar(url, clave):
    return llamar(url, "user.login", {"username": "Admin", "password": clave})


def leer_clave(fichero):
    with open(fichero) as f:
        return f.read().strip()


def esperar(url):
    for _ in range(120):
        try:
            print(llamar(url, "apiinfo.version", {}, tiempo=5))
            return
        except Exception:
            time.sleep(5)
    sys.exit("la API no respondió en 10 minutos")


def preparar(url, fichero):
    token = entrar(url, CLAVE_INICIAL)
    nueva = secrets.token_urlsafe(24)
    uid = llamar(url, "user.get", {"filter": {"username": "Admin"}, "output": ["userid"]}, token)[0]["userid"]
    llamar(url, "user.update", {"userid": uid, "current_passwd": CLAVE_INICIAL, "passwd": nueva}, token)
    fd = os.open(fichero, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        f.write(nueva)
    token = entrar(url, nueva)
    hosts = llamar(url, "host.get", {"filter": {"host": "Zabbix server"}, "output": ["hostid"]}, token)
    for h in hosts:
        llamar(url, "host.update", {"hostid": h["hostid"], "status": 1}, token)
    print(json.dumps({"admin_clave_cambiada": True, "host_por_defecto_desactivado": len(hosts)}))


def cargar(url, fichero, plantilla, host, base, regex):
    token = entrar(url, leer_clave(fichero))
    with open(plantilla) as f:
        fuente = f.read()
    reglas = {
        "template_groups": {"createMissing": True, "updateExisting": True},
        "templates": {"createMissing": True, "updateExisting": True},
        "discoveryRules": {"createMissing": True, "updateExisting": True},
        "items": {"createMissing": True, "updateExisting": True},
    }
    llamar(url, "configuration.import", {"format": "yaml", "rules": reglas, "source": fuente}, token, tiempo=120)
    nombres = ["Linux by Zabbix agent active", "Zabbix server health", "VEC cgroup contenedores"]
    plantillas = llamar(url, "template.get", {"filter": {"host": nombres}, "output": ["templateid", "host"]}, token)
    if len(plantillas) != len(nombres):
        sys.exit(f"faltan plantillas: {sorted(set(nombres) - {p['host'] for p in plantillas})}")
    grupo = llamar(url, "hostgroup.get", {"filter": {"name": ["VEC/Piloto"]}, "output": ["groupid"]}, token)
    gid = grupo[0]["groupid"] if grupo else llamar(url, "hostgroup.create", {"name": "VEC/Piloto"}, token)["groupids"][0]
    macros = [
        {"macro": "{$VEC.CGROUP.BASE}", "value": base},
        {"macro": "{$VEC.CGROUP.REGEX}", "value": regex},
        {"macro": "{$VEC.CGROUP.LLD.INTERVALO}", "value": "1m"},
        # Solo las sondas de disco montadas por el piloto; nada del contenido.
        {"macro": "{$VFS.FS.FSNAME.MATCHES}", "value": "^/sondas/"},
    ]
    llamar(url, "host.create", {
        "host": host, "groups": [{"groupid": gid}],
        "templates": [{"templateid": p["templateid"]} for p in plantillas],
        "macros": macros,
    }, token)
    print(json.dumps({"host": host, "plantillas": sorted(p["host"] for p in plantillas)}))


def hostid(url, token, host):
    return llamar(url, "host.get", {"filter": {"host": host}, "output": ["hostid"]}, token)[0]["hostid"]


def panel(url, fichero, host):
    """Consulta periódica equivalente a la que hará Administración (M4)."""
    token = entrar(url, leer_clave(fichero))
    hid = hostid(url, token, host)
    t0 = time.monotonic()
    llamar(url, "problem.get", {"hostids": [hid], "recent": True, "output": ["eventid", "severity", "name"], "limit": 50}, token)
    llamar(url, "host.get", {"hostids": [hid], "output": ["hostid", "status", "active_available"]}, token)
    llamar(url, "item.get", {"hostids": [hid], "search": {"key_": "system.cpu"}, "output": ["key_", "lastvalue", "lastclock"], "limit": 20}, token)
    llamar(url, "user.logout", {}, token)
    print(f"{(time.monotonic() - t0) * 1000:.0f}")


def verificar(url, fichero, host):
    token = entrar(url, leer_clave(fichero))
    hid = hostid(url, token, host)
    items = llamar(url, "item.get", {"hostids": [hid], "output": ["key_", "state", "status", "lastvalue", "lastclock", "value_type", "error"], "webitems": False}, token)
    activos = [i for i in items if i["status"] == "0"]
    soportados = [i for i in activos if i["state"] == "0"]
    no_sop = [i for i in activos if i["state"] != "0"]
    con_valor = [i for i in soportados if i["lastclock"] != "0"]
    cgroup = [i for i in activos if i["key_"].startswith(("vfs.file.", "vec.cgroup."))]
    cgroup_con_valor = [i for i in cgroup if i["state"] == "0" and i["lastclock"] != "0"]
    # Las claves creadas por LLD conservan la macro: ...[{$VEC.CGROUP.BASE}/<grupo>/fichero,...]
    contenedores = sorted({m.group(1) for i in cgroup for m in [re.search(r"\}/([^/]+)/", i["key_"])] if m})

    def valor(clave):
        for i in items:
            if i["key_"] == clave:
                return i["lastvalue"]
        return None

    muestra = {
        "system.cpu.util": valor("system.cpu.util"),
        "vm.memory.size[available]": valor("vm.memory.size[available]"),
        "vm.memory.size[total]": valor("vm.memory.size[total]"),
        "system.cpu.num": valor("system.cpu.num"),
        "proc.num": valor("proc.num"),
    }
    for i in items:
        if "/sondas/" in i["key_"] and "pused" in i["key_"]:
            muestra[i["key_"]] = i["lastvalue"]
    ejemplo = next((i for i in cgroup_con_valor if "memory.current" in i["key_"]), None)
    rendimiento = next((i["lastvalue"] for i in items if "requiredperformance" in i["key_"]), None)
    print(json.dumps({
        "elementos_activos": len(activos),
        "elementos_soportados": len(soportados),
        "elementos_con_valor": len(con_valor),
        "elementos_no_soportados": len(no_sop),
        "no_soportados_claves": sorted(i["key_"] for i in no_sop)[:40],
        "elementos_cgroup": len(cgroup),
        "elementos_cgroup_con_valor": len(cgroup_con_valor),
        "contenedores_descubiertos": len(contenedores),
        "muestra_host": muestra,
        "ejemplo_contenedor_memoria": ejemplo["lastvalue"] if ejemplo else None,
        "nvps_requerido": rendimiento,
    }, ensure_ascii=False, indent=1))


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    orden, args = sys.argv[1], sys.argv[2:]
    {"esperar": esperar, "preparar": preparar, "cargar": cargar, "panel": panel, "verificar": verificar}[orden](*args)


if __name__ == "__main__":
    main()

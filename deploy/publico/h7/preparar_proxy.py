#!/usr/bin/env python3
"""Render a host-scoped HTTPS proxy without installing or reloading it."""
import argparse
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import ssl
import stat
from string import Template
import sys

MAX_CONFIG = 65536
PROXY_KEYS = {
    "public_host", "public_port", "upstream_address", "upstream_port",
    "upstream_server_name", "upstream_ca_file", "edge_certificate_file", "edge_key_file",
}


def exact_keys(value, keys):
    if not isinstance(value, dict) or set(value) != set(keys):
        raise ValueError("schema")


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate_key")
        result[key] = value
    return result


def safe_path(value):
    if (not isinstance(value, str) or not re.fullmatch(r"/[A-Za-z0-9_./ -]+", value)
            or any(part in ("", ".", "..") for part in value.split("/")[1:])):
        raise ValueError("path")
    path = Path(value)
    if any(part.is_symlink() for part in (path, *path.parents)):
        raise ValueError("symlink")
    return path


def read_file(value, *, private=False, limit=MAX_CONFIG):
    path = safe_path(value)
    # Reject links and non-regular files before reading; O_NONBLOCK avoids FIFO waits.
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1
                or info.st_size > limit or info.st_mode & 0o022):
            raise ValueError("file")
        if private and (info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600):
            raise ValueError("private_file")
        data = stream.read(limit + 1)
        if len(data) > limit:
            raise ValueError("file_size")
        return data


def private_json(value):
    path = safe_path(value)
    info = path.parent.stat()
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise ValueError("private_directory")
    return json.loads(read_file(value, private=True), object_pairs_hook=unique_object)


def dns_name(value, *, complete=False):
    if (not isinstance(value, str) or len(value) > 253
            or not re.fullmatch(r"[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?", value)
            or any(not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
                   for label in value.split("."))
            or (complete and "." not in value)):
        raise ValueError("dns_name")
    try:
        ipaddress.ip_address(value)
    except ValueError:
        return value
    raise ValueError("dns_name")


def port(value):
    if type(value) is not int or not 1 <= value <= 65535:
        raise ValueError("port")
    return value


def loopback(value):
    if not isinstance(value, str) or "%" in value:
        raise ValueError("loopback")
    address = ipaddress.ip_address(value)
    if not address.is_loopback or getattr(address, "ipv4_mapped", None) is not None:
        raise ValueError("loopback")
    return str(address)


def tls_context(ca_file, certificate=None, key=None):
    # Load checked bytes for trust. No system trust, environment proxy or insecure mode.
    ca = read_file(ca_file).decode("ascii")
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    context.set_alpn_protocols(["http/1.1"])
    context.load_verify_locations(cadata=ca)
    if certificate is not None:
        read_file(certificate)
        read_file(key, private=True)
        context.load_cert_chain(certificate, key)
    return context


def validate_proxy(cfg):
    exact_keys(cfg, PROXY_KEYS)
    dns_name(cfg["public_host"], complete=True)
    dns_name(cfg["upstream_server_name"])
    port(cfg["public_port"])
    port(cfg["upstream_port"])
    loopback(cfg["upstream_address"])
    tls_context(cfg["upstream_ca_file"])
    read_file(cfg["edge_certificate_file"])
    read_file(cfg["edge_key_file"], private=True)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(cfg["edge_certificate_file"], cfg["edge_key_file"])
    return cfg


def render(cfg):
    validate_proxy(cfg)
    values = dict(cfg)
    address = loopback(cfg["upstream_address"])
    values["upstream_address"] = "[" + address + "]" if ":" in address else address
    for key in ("upstream_ca_file", "edge_certificate_file", "edge_key_file"):
        values[key] = json.dumps(cfg[key])
    template = Path(__file__).with_name("Caddyfile.publico.template").read_text()
    return Template(template).substitute(values)


def write_new(value, content):
    path = safe_path(value)
    info = path.parent.stat()
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise ValueError("private_directory")
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as stream:
        stream.write(content)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    try:
        content = render(private_json(args.config))
        write_new(args.output, content)
        print(json.dumps({"result": "prepared", "sha256": hashlib.sha256(content.encode()).hexdigest(),
                          "caddy_validated": False, "installed": False}))
    except (OSError, ValueError, TypeError, KeyError, ssl.SSLError):
        print("proxy_preparation_rejected", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

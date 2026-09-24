"""Generación y comprobación de una clave HMAC no exportable en PKCS#11.

Se usa la ABI estándar PKCS#11 mediante ctypes para pasar el PIN en memoria,
sin argumento de proceso ni importación de una clave creada fuera del token.
"""

from __future__ import annotations

import ctypes as c
from pathlib import Path


CK = c.c_ulong
BOOL = c.c_ubyte
CKR_OK = 0
CKR_USER_ALREADY_LOGGED_IN = 0x00000100
CKF_RW_SESSION = 0x00000002
CKF_SERIAL_SESSION = 0x00000004
CKU_USER = 1
CKO_SECRET_KEY = 4
CKK_GENERIC_SECRET = 0x10
CKM_SHA256_HMAC = 0x251
CKM_GENERIC_SECRET_KEY_GEN = 0x350


class Attribute(c.Structure):
    _fields_ = [("type", CK), ("pValue", c.c_void_p), ("ulValueLen", CK)]


class Mechanism(c.Structure):
    _fields_ = [("mechanism", CK), ("pParameter", c.c_void_p), ("ulParameterLen", CK)]


def check(result: int, operation: str) -> None:
    if result != CKR_OK:
        raise RuntimeError(f"PKCS#11 {operation} rechazó la operación (CKR={result:08x})")


def _attr(kind: int, value: object) -> tuple[Attribute, object]:
    return Attribute(kind, c.cast(c.pointer(value), c.c_void_p), c.sizeof(value)), value


def _bytes_attr(kind: int, value: bytes) -> tuple[Attribute, object]:
    buffer = c.create_string_buffer(value, len(value))
    return Attribute(kind, c.cast(buffer, c.c_void_p), len(value)), buffer


def _attributes(pairs: list[tuple[int, object]]) -> tuple[object, list[object]]:
    attributes = []
    keep = []
    for kind, value in pairs:
        a, retained = _bytes_attr(kind, value) if isinstance(value, bytes) else _attr(kind, value)
        attributes.append(a)
        keep.append(retained)
    return (Attribute * len(attributes))(*attributes), keep


def provision(module: Path, token_label: str, token_serial: str, object_id: bytes,
              object_label: str, pin: bytes) -> None:
    if not module.is_file() or module.is_symlink():
        raise RuntimeError("módulo PKCS#11 inválido")
    lib = c.CDLL(str(module))
    lib.C_Initialize.argtypes = [c.c_void_p]
    lib.C_Initialize.restype = CK
    lib.C_Finalize.argtypes = [c.c_void_p]
    lib.C_Finalize.restype = CK
    lib.C_GetSlotList.argtypes = [BOOL, c.POINTER(CK), c.POINTER(CK)]
    lib.C_GetSlotList.restype = CK
    lib.C_GetTokenInfo.argtypes = [CK, c.c_void_p]
    lib.C_GetTokenInfo.restype = CK
    lib.C_OpenSession.argtypes = [CK, CK, c.c_void_p, c.c_void_p, c.POINTER(CK)]
    lib.C_OpenSession.restype = CK
    lib.C_CloseSession.argtypes = [CK]
    lib.C_CloseSession.restype = CK
    lib.C_Login.argtypes = [CK, CK, c.c_void_p, CK]
    lib.C_Login.restype = CK
    lib.C_Logout.argtypes = [CK]
    lib.C_Logout.restype = CK
    lib.C_FindObjectsInit.argtypes = [CK, c.POINTER(Attribute), CK]
    lib.C_FindObjectsInit.restype = CK
    lib.C_FindObjects.argtypes = [CK, c.POINTER(CK), CK, c.POINTER(CK)]
    lib.C_FindObjects.restype = CK
    lib.C_FindObjectsFinal.argtypes = [CK]
    lib.C_FindObjectsFinal.restype = CK
    lib.C_GenerateKey.argtypes = [CK, c.POINTER(Mechanism), c.POINTER(Attribute), CK, c.POINTER(CK)]
    lib.C_GenerateKey.restype = CK
    lib.C_GetAttributeValue.argtypes = [CK, CK, c.POINTER(Attribute), CK]
    lib.C_GetAttributeValue.restype = CK
    lib.C_SignInit.argtypes = [CK, c.POINTER(Mechanism), CK]
    lib.C_SignInit.restype = CK
    lib.C_Sign.argtypes = [CK, c.c_void_p, CK, c.c_void_p, c.POINTER(CK)]
    lib.C_Sign.restype = CK
    check(lib.C_Initialize(None), "inicializar")
    session = None
    logged = False
    try:
        count = CK()
        check(lib.C_GetSlotList(BOOL(1), None, c.byref(count)), "enumerar tokens")
        if count.value < 1 or count.value > 64:
            raise RuntimeError("número de tokens PKCS#11 inválido")
        slots = (CK * count.value)()
        check(lib.C_GetSlotList(BOOL(1), slots, c.byref(count)), "leer tokens")
        selected = []
        for slot in slots[:count.value]:
            raw = c.create_string_buffer(256)
            check(lib.C_GetTokenInfo(slot, raw), "leer token")
            label = raw.raw[:32].decode("ascii", errors="ignore").strip()
            serial = raw.raw[80:96].decode("ascii", errors="ignore").strip()
            if label == token_label and serial == token_serial:
                selected.append(slot)
        if len(selected) != 1:
            raise RuntimeError("token label/serial ausente o ambiguo")
        handle = CK()
        check(lib.C_OpenSession(selected[0], CKF_RW_SESSION | CKF_SERIAL_SESSION,
                                None, None, c.byref(handle)), "abrir sesión")
        session = handle.value
        pin_buf = c.create_string_buffer(pin, len(pin))
        rv = lib.C_Login(session, CKU_USER, pin_buf, len(pin))
        c.memset(pin_buf, 0, len(pin_buf))
        if rv not in (CKR_OK, CKR_USER_ALREADY_LOGGED_IN):
            check(rv, "PIN")
        logged = rv == CKR_OK
        search, keep = _attributes([(0x000, CK(CKO_SECRET_KEY)), (0x102, object_id)])
        check(lib.C_FindObjectsInit(session, search, len(search)), "buscar clave")
        try:
            found = (CK * 2)()
            found_count = CK()
            check(lib.C_FindObjects(session, found, 2, c.byref(found_count)), "enumerar claves")
        finally:
            check(lib.C_FindObjectsFinal(session), "cerrar búsqueda")
        if found_count.value > 1:
            raise RuntimeError("ID de clave PKCS#11 ambiguo")
        if found_count.value == 1:
            key = found[0]
        else:
            mechanism = Mechanism(CKM_GENERIC_SECRET_KEY_GEN, None, 0)
            template, keep = _attributes([
                (0x000, CK(CKO_SECRET_KEY)), (0x100, CK(CKK_GENERIC_SECRET)),
                (0x001, BOOL(1)), (0x002, BOOL(1)), (0x003, object_label.encode()),
                (0x102, object_id), (0x103, BOOL(1)), (0x108, BOOL(1)),
                (0x162, BOOL(0)), (0x161, CK(32)),
            ])
            result = CK()
            check(lib.C_GenerateKey(session, c.byref(mechanism), template,
                                    len(template), c.byref(result)), "generar HMAC")
            key = result.value
        required = {0x000: CKO_SECRET_KEY, 0x100: CKK_GENERIC_SECRET,
                    0x001: 1, 0x002: 1, 0x103: 1, 0x108: 1,
                    0x162: 0, 0x164: 1, 0x165: 1, 0x163: 1}
        for kind, expected in required.items():
            value = CK() if kind in (0x000, 0x100) else BOOL()
            attr, retained = _attr(kind, value)
            check(lib.C_GetAttributeValue(session, key, c.byref(attr), 1), "verificar atributos")
            if value.value != expected:
                raise RuntimeError("atributos de clave HMAC incompatibles")
        length = CK()
        attr, retained = _attr(0x161, length)
        check(lib.C_GetAttributeValue(session, key, c.byref(attr), 1), "verificar longitud")
        if length.value < 32:
            raise RuntimeError("clave HMAC demasiado corta")
        mechanism = Mechanism(CKM_SHA256_HMAC, None, 0)
        check(lib.C_SignInit(session, c.byref(mechanism), key), "habilitar HMAC SHA256")
        challenge = c.create_string_buffer(b"vec-interno-pkcs11-preflight", 29)
        signature = c.create_string_buffer(64)
        size = CK(len(signature))
        check(lib.C_Sign(session, challenge, 29, signature, c.byref(size)), "firmar reto")
        if size.value != 32:
            raise RuntimeError("HMAC SHA256 de longitud inesperada")
    finally:
        if session is not None:
            if logged:
                lib.C_Logout(session)
            lib.C_CloseSession(session)
        lib.C_Finalize(None)

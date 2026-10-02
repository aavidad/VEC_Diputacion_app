#!/usr/bin/env python3
"""Child boundary: token is read locally and never appears in process arguments."""
import os
from pathlib import Path
import sys

kit = Path(sys.argv[1])
token = (kit / 'runtime/grxfirma/token').read_text().strip()
if len(token) < 32 or any(c.isspace() for c in token):
    raise SystemExit('credencial_sintetica_no_valida')
home = kit / 'runtime/grxfirma/home'
os.execve('/work/grxfirma', ['/work/grxfirma', '-rest-solo-verificacion',
    '-verificacion-anclas', str(kit / 'pki/ancla.pem'), '-verificacion-crl', str(kit / 'pki/crl'),
    '-direccion-rest', '127.0.0.1:63118'],
    {'HOME': str(home), 'PATH': '/usr/bin:/bin', 'TMPDIR': '/tmp', 'TZ': 'UTC', 'AUTOFIRMAV2_REST_TOKEN': token, 'GRXFIRMA_REST_TOKEN': token})

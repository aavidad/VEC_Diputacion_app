"""Candado compartido con ADMIN y parada limitada al proceso de la copia."""

import fcntl
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import copia_local_administracion as admin
import copia_local as conductor


class AdministracionCopiaTest(unittest.TestCase):
    def setUp(self):
        temporal = tempfile.TemporaryDirectory()
        self.addCleanup(temporal.cleanup)
        base = Path(temporal.name)
        self.raiz = base / 'l-admin'
        self.raiz.mkdir(mode=0o700)
        self.lock = (base / '.vec-copia-l-admin.lock').open('a')
        self.addCleanup(self.lock.close)
        fcntl.flock(self.lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        self.script = base / 'ejecutor.py'
        self.script.write_text('''import argparse,fcntl
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--raiz');p.add_argument('--fase');p.add_argument('--candado-fd',type=int)
p.add_argument('--puerto-admin',type=int)
a=p.parse_args()
fcntl.flock(a.candado_fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
r=Path(a.raiz)
with (r.parent/'.vec-copia-l-admin.lock').open('a') as otro:
 try: fcntl.flock(otro,fcntl.LOCK_EX|fcntl.LOCK_NB)
 except BlockingIOError: pass
 else: raise RuntimeError('La instancia quedó sin exclusión')
(r/'fase-confirmada').write_text(a.fase)
''')
        self.script.chmod(0o600)
        self.estado = dict(administracion=dict(ejecutor=str(self.script),
                          sha256=admin.huella_ejecutor(self.script), puerto=19485))
        self.copia = SimpleNamespace(raiz=self.raiz, nombre='vec-copia-l-admin',
                                     candado=self.lock.fileno(), leer=lambda: self.estado)

    def test_hijo_hereda_candado_y_otro_escritor_sigue_excluido(self):
        admin.ejecutar(self.copia, 'comprobar')
        self.assertEqual((self.raiz / 'fase-confirmada').read_text(), 'comprobar')

    def test_conductor_conserva_descriptor_para_preparador_externo(self):
        conductor.ejecutar([sys.executable, str(self.script), '--raiz', str(self.raiz),
                           '--fase', 'portal', '--candado-fd', str(self.lock.fileno()),
                           '--puerto-admin', '19485'], descriptores=(self.lock.fileno(),))
        self.assertEqual((self.raiz / 'fase-confirmada').read_text(), 'portal')

    def test_fuente_alterada_no_se_ejecuta(self):
        self.script.write_text("raise RuntimeError('no debe ejecutarse')")
        with self.assertRaises(ValueError):
            admin.ejecutar(self.copia, 'preparar')
        self.assertFalse((self.raiz / 'fase-confirmada').exists())

    def test_otro_descriptor_no_suple_el_candado_de_la_instancia(self):
        with (self.raiz / 'ajeno.lock').open('w') as ajeno:
            self.copia.candado = ajeno.fileno()
            with self.assertRaises(ValueError):
                admin.ejecutar(self.copia, 'preparar')
        self.assertFalse((self.raiz / 'fase-confirmada').exists())

    def test_parada_no_envia_senal_a_pid_con_otro_ejecutable(self):
        self.estado.update(pid_admin=os.getpid(), inicio_proceso_admin=
                           Path(f'/proc/{os.getpid()}/stat').read_text().split()[21])
        with patch.object(admin.os, 'kill') as kill:
            admin.parar(self.copia)
        kill.assert_not_called()

    def test_para_ejecutable_propio_solo_con_comienzo_coincidente(self):
        binario = self.raiz / 'vec-admin'
        shutil.copyfile(Path(sys.executable).resolve(), binario)
        binario.chmod(0o700)
        proceso = subprocess.Popen([str(binario), '-c', 'import time; time.sleep(30)'])
        try:
            inicio = Path(f'/proc/{proceso.pid}/stat').read_text().split()[21]
            self.estado.update(pid_admin=proceso.pid, inicio_proceso_admin='otro')
            admin.parar(self.copia)
            self.assertIsNone(proceso.poll())
            self.estado['inicio_proceso_admin'] = inicio
            admin.parar(self.copia)
            self.assertEqual(proceso.wait(timeout=2), -15)
        finally:
            if proceso.poll() is None:
                proceso.terminate()
            proceso.wait(timeout=2)

    def test_sin_administracion_no_necesita_ejecutor(self):
        self.estado.clear()
        admin.ejecutar(self.copia, 'comprobar')
        admin.parar(self.copia)


if __name__ == '__main__':
    unittest.main()

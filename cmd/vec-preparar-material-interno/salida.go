package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	errSalida     = errorPropio("salida: la ruta debe ser absoluta, sin enlaces, fuera de Git, en un directorio padre privado del usuario y no existir o estar vacía con 0700")
	errEscritura  = errorPropio("salida: no se pudo escribir el material en el directorio temporal")
	errValidacion = errorPropio("salida: el material compuesto no supera los cargadores reales de vec-interno")
	errActivacion = errorPropio("salida: activación rechazada; no queda material parcial")
	errCancelada  = errorPropio("preparación interrumpida antes de activar; no queda material parcial")

	nombreInventarioB2 = "personal_b2_v3.json"
	versionFormatoB2   = 4
	maximoSecretoB2    = 256
)

// capacidadInventario reproduce exactamente capacidadMaterial del cargador.
type capacidadInventario struct {
	ClaveID          string    `json:"clave_id"`
	Version          uint64    `json:"version"`
	Archivo          string    `json:"archivo"`
	SHA256           string    `json:"sha256"`
	EmisorID         string    `json:"emisor_id"`
	Desde            time.Time `json:"desde"`
	Hasta            time.Time `json:"hasta"`
	RevisionGobierno uint64    `json:"revision_gobierno"`
	HuellaGobierno   string    `json:"huella_gobierno"`
}

type inventarioB2 struct {
	Version         int       `json:"version"`
	CatalogoMotivos string    `json:"catalogo_motivos"`
	Motivos         motivosB2 `json:"motivos"`
	V3              struct {
		Capacidades map[string]capacidadInventario `json:"capacidades"`
	} `json:"v3"`
}

func nombreArchivoCapacidad(capacidad string) string {
	return "personal_b2_" + capacidad + ".hmac"
}

type preparacion struct {
	opciones
	dsnEntorno string
	dep        dependencias
}

// preparar valida todas las entradas, coteja con el gobierno, compone en un
// directorio temporal hermano de la salida, lo valida con los cargadores
// reales y solo entonces lo activa con un único rename(2).
func (p preparacion) preparar(ctx context.Context) error {
	if ctx == nil || p.dep.abrirGobierno == nil || p.dep.reloj == nil {
		return errCancelada
	}
	padre, salidaExiste, err := comprobarSalida(p.salida)
	if err != nil {
		return err
	}
	ct, err := leerDatosCT(p.inventarioCT)
	if err != nil {
		return err
	}
	motivos, err := leerMotivos(p.motivos, ct.catalogo)
	if err != nil {
		return err
	}
	dsn, err := leerDSN(p.dsnArchivo, p.dsnEntorno)
	if err != nil {
		return err
	}
	base, err := leerClaveBase(p.claveBase)
	if err != nil {
		return err
	}
	defer clear(base)
	if ctx.Err() != nil {
		return errCancelada
	}
	g, err := p.dep.abrirGobierno(ctx, dsn)
	if err != nil {
		return errGobiernoConexion
	}
	capacidades, err := cotejar(ctx, g, ct, base)
	g.cerrar()
	defer borrarCapacidades(&capacidades)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return errCancelada
	}
	temporal, err := os.MkdirTemp(padre, ".vec-preparar-material-")
	if err != nil {
		return errEscritura
	}
	activado := false
	defer func() {
		if !activado {
			_ = os.RemoveAll(temporal)
		}
	}()
	if err := escribirMaterial(temporal, ct.catalogo, motivos, capacidades); err != nil {
		return err
	}
	if err := validarConCargadores(temporal, ct.catalogo, motivos, capacidades, p.dep.reloj()); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return errCancelada
	}
	if p.dep.antesDeActivar != nil {
		if err := p.dep.antesDeActivar(); err != nil {
			return errCancelada
		}
	}
	if err := activar(temporal, p.salida, salidaExiste); err != nil {
		return err
	}
	activado = true
	return nil
}

// comprobarSalida exige una salida inexistente o vacía (0700, del usuario,
// no enlace) bajo un padre canónico del usuario sin escritura de terceros y
// fuera de cualquier árbol Git.
func comprobarSalida(salida string) (string, bool, error) {
	if salida == "" || !filepath.IsAbs(salida) || filepath.Clean(salida) != salida || salida == "/" {
		return "", false, errSalida
	}
	padre := filepath.Dir(salida)
	if !rutaCanonica(padre) || dentroDeGit(padre) {
		return "", false, errSalida
	}
	infoPadre, err := os.Lstat(padre)
	if err != nil || !infoPadre.IsDir() || !delUsuario(infoPadre) || infoPadre.Mode().Perm()&0022 != 0 {
		return "", false, errSalida
	}
	info, err := os.Lstat(salida)
	if os.IsNotExist(err) {
		return padre, false, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 || !delUsuario(info) {
		return "", false, errSalida
	}
	d, err := os.Open(salida)
	if err != nil {
		return "", false, errSalida
	}
	defer d.Close()
	if _, err := d.Readdirnames(1); err != io.EOF {
		return "", false, errSalida
	}
	return padre, true, nil
}

// leerDatosCT carga ct_v3.json con el cargador real de vec-interno y extrae
// solo coordenadas públicas: catálogo, emisor único y raíz.
func leerDatosCT(ruta string) (datosCT, error) {
	if !rutaCanonica(ruta) || filepath.Base(ruta) != "ct_v3.json" {
		return datosCT{}, errInventarioCT
	}
	if info, err := os.Lstat(ruta); err != nil || !info.Mode().IsRegular() || !delUsuario(info) {
		return datosCT{}, errInventarioCT
	}
	m, err := internactproveedores.CargarMaterial(filepath.Dir(ruta))
	if err != nil {
		return datosCT{}, errInventarioCT
	}
	defer m.Cerrar()
	c := m.V3.Capacidades
	emisor := c.Alta.EmisorID
	for _, e := range []string{c.Lectura.EmisorID, c.CT.EmisorID, c.Cuadro.EmisorID, c.Detalle.EmisorID} {
		if e == "" || e != emisor {
			return datosCT{}, errEmisorCT
		}
	}
	return datosCT{catalogo: m.CatalogoMotivos, emisor: emisor, raizID: m.V3.ClaveID, raizVersion: m.V3.ClaveVersion, audiencia: m.V3.Audiencia}, nil
}

func escribirMaterial(dir, catalogo string, motivos motivosB2, capacidades [8]capacidadCotejada) error {
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return errEscritura
	}
	defer raiz.Close()
	inv := inventarioB2{Version: versionFormatoB2, CatalogoMotivos: catalogo, Motivos: motivos}
	inv.V3.Capacidades = make(map[string]capacidadInventario, len(capacidades))
	for _, c := range capacidades {
		archivo := nombreArchivoCapacidad(c.Capacidad)
		if err := escribirPrivado(raiz, archivo, c.secreto); err != nil {
			return err
		}
		inv.V3.Capacidades[c.Capacidad] = capacidadInventario{
			ClaveID: c.ClaveID, Version: c.Version, Archivo: archivo, SHA256: c.SHA256, EmisorID: c.EmisorID,
			Desde: c.Desde.UTC(), Hasta: c.Hasta.UTC(), RevisionGobierno: c.RevisionGobierno, HuellaGobierno: c.HuellaGobierno,
		}
	}
	b, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return errEscritura
	}
	defer clear(b)
	// El inventario se escribe el último: sin él, el directorio no es material.
	if err := escribirPrivado(raiz, nombreInventarioB2, append(b, '\n')); err != nil {
		return err
	}
	return sincronizarDirectorio(dir)
}

// escribirPrivado crea el fichero de forma exclusiva y sin seguir enlaces,
// fija 0600 aunque la umask lo reduzca y lo sincroniza.
func escribirPrivado(raiz *os.Root, nombre string, contenido []byte) error {
	f, err := raiz.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errEscritura
	}
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return errEscritura
	}
	if _, err := f.Write(contenido); err != nil {
		_ = f.Close()
		return errEscritura
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return errEscritura
	}
	if f.Close() != nil {
		return errEscritura
	}
	return nil
}

func sincronizarDirectorio(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return errEscritura
	}
	defer d.Close()
	if d.Sync() != nil {
		return errEscritura
	}
	return nil
}

// validarConCargadores abre el temporal con el cargador real de formato 4 de
// vec-interno y reconstruye cada clave con los mismos constructores y
// comprobaciones que aplica su composición (crearCapacidad): huella del
// fichero, clave HMAC de emisión, audiencia y ventana de vigencia.
func validarConCargadores(dir, catalogo string, motivos motivosB2, esperadas [8]capacidadCotejada, ahora time.Time) error {
	m, err := internactproveedores.CargarMaterialPersonalB2(dir)
	if err != nil {
		return errValidacion
	}
	defer m.Cerrar()
	// capacidadMaterial no se exporta; sus coordenadas públicas se leen con
	// las mismas etiquetas JSON (no contienen secretos).
	c := m.V3.Capacidades
	var cargadas [8]capacidadInventario
	for i, v := range []any{c.Ficha, c.Vacantes, c.Alta, c.Hecho, c.CatalogoConsultar, c.CatalogoPublicar, c.CatalogoRetirar, c.Empleados} {
		b, err := json.Marshal(v)
		if err != nil || json.Unmarshal(b, &cargadas[i]) != nil {
			return errValidacion
		}
	}
	if m.Version != versionFormatoB2 || m.CatalogoMotivos != catalogo {
		return errValidacion
	}
	cargados := [8]core.ReferenciaEntradaCatalogo{m.Motivos.Ficha, m.Motivos.Vacantes, m.Motivos.Alta, m.Motivos.Hecho, m.Motivos.CatalogoConsultar, m.Motivos.CatalogoPublicar, m.Motivos.CatalogoRetirar, m.Motivos.Empleados}
	if cargados != motivos.lista() {
		return errValidacion
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return errValidacion
	}
	defer raiz.Close()
	descriptores := bootstrap.DescriptoresCapacidadPersonalB2V3Desarrollo()
	for i, x := range cargadas {
		e := esperadas[i]
		if descriptores[i].Capacidad != e.Capacidad || x.Archivo != nombreArchivoCapacidad(e.Capacidad) || x.ClaveID != e.ClaveID ||
			x.EmisorID != e.EmisorID || x.HuellaGobierno != e.HuellaGobierno || x.SHA256 != e.SHA256 || x.Version != e.Version ||
			x.RevisionGobierno != e.RevisionGobierno || !x.Desde.Equal(e.Desde) || !x.Hasta.Equal(e.Hasta) {
			return errValidacion
		}
		if err := comprobarCapacidad(raiz, x, descriptores[i].Audiencia, ahora); err != nil {
			return err
		}
	}
	return nil
}

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

func comprobarCapacidad(raiz *os.Root, x capacidadInventario, audiencia string, ahora time.Time) error {
	info, err := raiz.Lstat(x.Archivo)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() < 1 || info.Size() > maximoSecretoB2 {
		return errValidacion
	}
	f, err := raiz.Open(x.Archivo)
	if err != nil {
		return errValidacion
	}
	secreto, err := io.ReadAll(io.LimitReader(f, maximoSecretoB2+1))
	_ = f.Close()
	defer clear(secreto)
	h := sha256.Sum256(secreto)
	if err != nil || int64(len(secreto)) != info.Size() || hex.EncodeToString(h[:]) != x.SHA256 {
		return errValidacion
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(x.ClaveID, x.Version, secreto, x.EmisorID, audiencia,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, x.Desde, x.Hasta, time.Time{}, x.RevisionGobierno, x.HuellaGobierno)
	if err != nil || ahora.Before(x.Desde) || !ahora.Before(x.Hasta) {
		return errValidacion
	}
	if _, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, relojFijo(ahora)); err != nil {
		return errValidacion
	}
	return nil
}

// activar sustituye la salida por el temporal con un único rename(2). Si la
// salida no existía se crea vacía justo antes y se retira si el rename falla;
// si existía vacía, rename(2) solo la sustituye mientras siga vacía y siga
// siendo un directorio (un enlace o contenido nuevo lo hacen fallar).
func activar(temporal, salida string, existia bool) error {
	creada := false
	if !existia {
		if err := os.Mkdir(salida, 0700); err != nil {
			return errActivacion
		}
		creada = true
	}
	if err := syscall.Rename(temporal, salida); err != nil {
		if creada {
			_ = os.Remove(salida)
		}
		return errActivacion
	}
	return sincronizarDirectorio(filepath.Dir(salida))
}

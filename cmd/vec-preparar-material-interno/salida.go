package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// preparar valida todas las entradas, obtiene las claves B2 por la ruta de
// vec-server, las coteja con el gobierno, compone en un directorio temporal
// hermano de la salida, lo valida con los cargadores reales y solo entonces
// lo activa con un único renameat(2). Devuelve `sincronizado=false` cuando
// el material quedó activado pero el fsync del padre no se confirmó.
func (p preparacion) preparar(ctx context.Context) (bool, error) {
	if ctx == nil || p.dep.abrirGobierno == nil || p.dep.reloj == nil {
		return false, errCancelada
	}
	destino, err := abrirDestino(p.salida)
	if err != nil {
		return false, err
	}
	defer destino.cerrar()
	ct, err := leerDatosCT(p.inventarioCT)
	if err != nil {
		return false, err
	}
	motivos, err := leerMotivos(p.motivos, ct.catalogo)
	if err != nil {
		return false, err
	}
	dsn, err := leerDSN(p.dsnArchivo, p.dsnEntorno)
	if err != nil {
		return false, err
	}
	if err := comprobarIdempotencia(p.idempotencia); err != nil {
		return false, err
	}
	if ctx.Err() != nil {
		return false, errCancelada
	}
	claves, err := bootstrap.DerivarClavesPersonalB2V3DesdeMaterialDesarrollo(p.idempotencia, p.dep.reloj())
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	if err != nil {
		return false, errIdempotencia
	}
	g, err := p.dep.abrirGobierno(ctx, dsn)
	if err != nil {
		if errors.Is(err, errIdentidad) {
			return false, errIdentidad
		}
		return false, errGobiernoConexion
	}
	capacidades, err := cotejar(ctx, g, ct, &claves)
	g.cerrar()
	defer borrarCapacidades(&capacidades)
	if err != nil {
		return false, err
	}
	if ctx.Err() != nil {
		return false, errCancelada
	}
	temporal, raizTemporal, err := destino.crearTemporal()
	if err != nil {
		return false, err
	}
	activado := false
	defer func() {
		_ = raizTemporal.Close()
		if !activado {
			destino.retirar(temporal)
		}
	}()
	if err := escribirMaterial(raizTemporal, ct.catalogo, motivos, capacidades); err != nil {
		return false, err
	}
	ruta, ok := destino.rutaTemporal(temporal, raizTemporal)
	if !ok {
		return false, errActivacion
	}
	if err := validarConCargadores(ruta, ct.catalogo, motivos, capacidades, p.dep.reloj()); err != nil {
		return false, err
	}
	if ctx.Err() != nil {
		return false, errCancelada
	}
	if p.dep.antesDeActivar != nil {
		if err := p.dep.antesDeActivar(); err != nil {
			return false, errCancelada
		}
	}
	sincronizado, err := destino.activar(temporal, p.dep.sincronizarPadre)
	if err != nil {
		return false, err
	}
	activado = true
	return sincronizado, nil
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

func escribirMaterial(raiz *os.Root, catalogo string, motivos motivosB2, capacidades [8]capacidadCotejada) error {
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
	return sincronizarDirectorio(raiz)
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

func sincronizarDirectorio(raiz *os.Root) error {
	d, err := raiz.Open(".")
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

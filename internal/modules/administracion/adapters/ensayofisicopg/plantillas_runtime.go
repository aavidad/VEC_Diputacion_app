package ensayofisicopg

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

type anclajeFisicoRuntime struct {
	nombre, raiz, imagen, artefacto, sello string
	fase                                   sync.Mutex
	archivadosIniciados                    bool
}

func (r *RuntimeObservacion) anclarFisico(ctx context.Context, artefacto string) error {
	sello, err := r.ComprobarExclusion(ctx)
	if err != nil {
		return falloPlantilla("anclaje_fisico")
	}
	r.fisico = &anclajeFisicoRuntime{nombre: r.Nombre, raiz: r.Raiz, imagen: r.ImagenSHA256, artefacto: artefacto, sello: sello}
	return nil
}

var errPlantillaRuntime = errors.New("ensayo_plantilla_runtime_no_comprobable")

// RuntimePlantillas existe sólo durante el hook del ensayo físico. La fuente
// verificada pertenece a la aplicación; el adaptador no descifra ni concede permisos.
// El lease es un registro técnico desechable: no acredita auditoría nominal
// ni habilita copias o restauraciones en la administración de producto.
type RuntimePlantillas struct {
	runtime        RuntimeObservacion
	fuente         puertos.ContextoFisicoVerificado
	datos          puertos.DatosContextoFisico
	nombre         string
	marca          string
	mu             sync.Mutex
	propiedad      *metadatosClon
	pendiente      bool
	despuesTecnico func(string, error) error
	antesTecnico   func(string) error
}
type metadatosClon struct {
	OID         string `json:"oid"`
	Propietario string `json:"propietario"`
	Marca       string `json:"marca"`
}
type leasePlantilla struct {
	Nombre    string                      `json:"nombre"`
	Marca     string                      `json:"marca"`
	Contexto  puertos.DatosContextoFisico `json:"contexto"`
	Propiedad *metadatosClon              `json:"propiedad,omitempty"`
}

func falloPlantilla(etapa string) error {
	slog.Error("ensayo_plantilla_runtime_fallido", "etapa", etapa)
	return errPlantillaRuntime
}
func datosFisicosAdmitidos(d puertos.DatosContextoFisico) bool {
	for _, r := range []string{d.OperacionRef, d.ConjuntoRef, d.VentanaRef} {
		if r == "" || len(r) > 256 || strings.ContainsAny(r, "\r\n\x00") {
			return false
		}
	}
	return huellaValida.MatchString(d.ManifiestoSHA256) && huellaValida.MatchString(d.ArtefactoFisicoSHA256) && huellaValida.MatchString(d.ImagenSHA256)
}

// NuevoRuntimePlantillas recibe el Entorno construido por el restaurador real.
// Recalcula el SHA del TAR privado, además de comparar el descriptor registrado,
// evitando convertir una marca o un digest enviado por otro canal en procedencia.
func NuevoRuntimePlantillas(ctx context.Context, e puertos.Entorno, fuente puertos.ContextoFisicoVerificado) (*RuntimePlantillas, error) {
	if ctx == nil || fuente == nil {
		return nil, falloPlantilla("entrada")
	}
	r, ok := e.PostgreSQL.(RuntimeObservacion)
	if !ok || r.fisico == nil || r.fisico.nombre != r.Nombre || r.fisico.raiz != r.Raiz || r.fisico.imagen != r.ImagenSHA256 || !r.ConfiguracionOrigenExcluida || len(r.Componentes) == 0 || r.Componentes[0].ID != "fisica:pgdata" || r.Componentes[0].Tipo != "base_fisica" {
		return nil, falloPlantilla("runtime_fisico")
	}
	d, err := fuente.RevalidarContextoFisico(ctx)
	if err != nil || !datosFisicosAdmitidos(d) || d.ArtefactoFisicoSHA256 != r.Componentes[0].SHA256 || d.ArtefactoFisicoSHA256 != r.fisico.artefacto || d.ImagenSHA256 != r.ImagenSHA256 {
		return nil, falloPlantilla("contexto_verificado")
	}
	p := &RuntimePlantillas{runtime: r, fuente: fuente, datos: d}
	if _, err = p.ObservarProcedenciaFisica(ctx); err != nil {
		return nil, err
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, falloPlantilla("nombre_propio")
	}
	p.nombre = "cs06t_" + hex.EncodeToString(nonce[:])
	p.marca = "vec_cs06_" + hex.EncodeToString(nonce[:])
	if err = p.escribirLease(true); err != nil {
		return nil, err
	}
	return p, nil
}
func (p *RuntimePlantillas) NombreClon() string {
	if p == nil {
		return ""
	}
	return p.nombre
}
func (p *RuntimePlantillas) ComprobarExclusion(ctx context.Context) (string, error) {
	if p == nil {
		return "", errPlantillaRuntime
	}
	return p.runtime.ComprobarExclusion(ctx)
}
func (p *RuntimePlantillas) EjecutarPostgreSQL(ctx context.Context, herramienta string, args []string, entrada []byte, max int) ([]byte, error) {
	if p == nil {
		return nil, errPlantillaRuntime
	}
	if _, err := p.ObservarProcedenciaFisica(ctx); err != nil {
		return nil, err
	}
	return p.runtime.EjecutarPostgreSQL(ctx, herramienta, args, entrada, max)
}
func (p *RuntimePlantillas) ObservarProcedenciaFisica(ctx context.Context) (puertos.ProcedenciaFisica, error) {
	var out puertos.ProcedenciaFisica
	if p == nil || ctx == nil {
		return out, errPlantillaRuntime
	}
	d, err := p.fuente.RevalidarContextoFisico(ctx)
	if err != nil || d != p.datos {
		return out, falloPlantilla("contexto_cambiado")
	}
	sello, err := p.runtime.ComprobarExclusion(ctx)
	if err != nil || p.runtime.fisico == nil || sello != p.runtime.fisico.sello {
		return out, falloPlantilla("exclusion")
	}
	hash, _, err := huellaContenido(ctx, filepath.Join(p.runtime.Raiz, "componente-0000.tar"))
	if err != nil || hash != d.ArtefactoFisicoSHA256 {
		return out, falloPlantilla("artefacto_fisico")
	}
	return puertos.ProcedenciaFisica{OperacionRef: d.OperacionRef, ConjuntoRef: d.ConjuntoRef, VentanaRef: d.VentanaRef, ManifiestoSHA256: d.ManifiestoSHA256, ArtefactoFisicoSHA256: hash, ImagenSHA256: p.runtime.ImagenSHA256, SelloExclusion: sello}, nil
}
func (p *RuntimePlantillas) leerClon(ctx context.Context) (*metadatosClon, error) {
	sql := fmt.Sprintf("SELECT pg_catalog.json_build_object('oid',oid::text,'propietario',datdba::text,'marca',coalesce(pg_catalog.shobj_description(oid,'pg_database'),''))::text FROM pg_catalog.pg_database WHERE datname='%s'", p.nombre)
	b, err := p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-v", "ON_ERROR_STOP=1", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", sql}, nil, 4096)
	if err != nil {
		return nil, falloPlantilla("metadatos_clon")
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}
	var m metadatosClon
	if json.Unmarshal(b, &m) != nil || !oidPlantillaValido(m.OID) || !oidPlantillaValido(m.Propietario) {
		return nil, falloPlantilla("metadatos_clon")
	}
	return &m, nil
}
func (p *RuntimePlantillas) propietarioActual(ctx context.Context) (string, error) {
	b, err := p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-v", "ON_ERROR_STOP=1", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", "SELECT oid::text FROM pg_catalog.pg_roles WHERE rolname=current_user"}, nil, 4096)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return "", falloPlantilla("propietario")
	}
	return strings.TrimSpace(string(b)), nil
}

// tecnico ejecuta una sentencia cerrada sobre la copia aislada. PGOPTIONS sólo
// cambia esta sesión; la configuración global permanece en lectura.
func (p *RuntimePlantillas) tecnico(ctx context.Context, sql string) error {
	return p.tecnicoValidado(ctx, sql, nil)
}
func (p *RuntimePlantillas) tecnicoValidado(ctx context.Context, sql string, validar func(context.Context) error) error {
	if _, err := p.ObservarProcedenciaFisica(ctx); err != nil {
		return err
	}
	if p.antesTecnico != nil {
		if err := p.antesTecnico(sql); err != nil {
			return falloPlantilla("preimagen_tecnica")
		}
	}
	if _, err := p.ObservarProcedenciaFisica(ctx); err != nil {
		return err
	}
	if validar != nil {
		if err := validar(ctx); err != nil {
			return err
		}
	}
	_, err := docker(ctx, nil, 4096, "exec", p.runtime.Nombre, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGOPTIONS=-c default_transaction_read_only=off", "psql", "-X", "-At", "-v", "ON_ERROR_STOP=1", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", sql)
	if p.despuesTecnico != nil {
		err = p.despuesTecnico(sql, err)
	}
	if err != nil {
		return falloPlantilla("sentencia_tecnica")
	}
	return nil
}
func (p *RuntimePlantillas) escribirLease(exclusivo bool) error {
	root, err := os.OpenRoot(filepath.Join(p.runtime.Raiz, "control"))
	if err != nil {
		return falloPlantilla("registro_propio")
	}
	defer root.Close()
	flags := os.O_WRONLY | os.O_CREATE
	if exclusivo {
		flags |= os.O_EXCL
	} else {
		flags |= os.O_TRUNC
	}
	f, err := root.OpenFile("plantilla-"+p.nombre+".json", flags, 0600)
	if err != nil {
		return falloPlantilla("registro_propio")
	}
	err = json.NewEncoder(f).Encode(leasePlantilla{p.nombre, p.marca, p.datos, p.propiedad})
	cerrar := f.Close()
	if err != nil || cerrar != nil {
		return falloPlantilla("registro_propio")
	}
	return nil
}

// CrearClonPlantilla sólo admite template0 y el nombre opaco generado aquí.
// Antes de emitir CREATE registra ausencia y la intención propia. Cualquier
// nombre preexistente se conserva aunque el propietario coincida.
func (p *RuntimePlantillas) CrearClonPlantilla(ctx context.Context, origen, clon string) error {
	if p == nil || origen != "template0" || clon != p.nombre {
		return falloPlantilla("clon_no_admitido")
	}
	desbloquear, err := p.runtime.escritorPlantillas()
	if err != nil {
		return err
	}
	defer desbloquear()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.propiedad != nil || p.pendiente {
		return falloPlantilla("clon_ocupado")
	}
	if _, err := p.ObservarProcedenciaFisica(ctx); err != nil {
		return err
	}
	previo, err := p.leerClon(ctx)
	if err != nil {
		return err
	}
	if previo != nil {
		return falloPlantilla("clon_preexistente")
	}
	owner, err := p.propietarioActual(ctx)
	if err != nil {
		return err
	}
	p.pendiente = true
	crearErr := p.tecnico(ctx, fmt.Sprintf(`CREATE DATABASE "%s" WITH TEMPLATE template0`, p.nombre))
	// Un error no demuestra que CREATE fuese nuestro. La aparición posterior de
	// un nombre nunca autoriza COMMENT ni DROP. El ensayo completo retira su
	// contenedor propio, también cuando el commit tiene respuesta perdida.
	if crearErr != nil {
		return crearErr
	}
	// Confirmar sólo una creación cuyo comando terminó correctamente.
	// La exclusión técnica impide iniciar procesos archivados en esta fase.
	revisar, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	actual, err := p.leerClon(revisar)
	if err != nil {
		return err
	}
	if actual == nil {
		p.pendiente = false
		return falloPlantilla("clon_ausente")
	}
	if actual.Propietario != owner || (actual.Marca != "" && actual.Marca != p.marca) {
		return falloPlantilla("propiedad_no_comprobable")
	}
	// El nombre aleatorio reservado, la ausencia previa y la exclusión del runtime
	// ligan la creación a esta intención. La marca y el OID gobiernan su retirada.
	creado := *actual
	if err = p.tecnicoValidado(revisar, fmt.Sprintf(`COMMENT ON DATABASE "%s" IS '%s'`, p.nombre, p.marca), func(ctx context.Context) error {
		actual, err := p.leerClon(ctx)
		if err != nil {
			return err
		}
		if actual == nil || *actual != creado {
			return falloPlantilla("clon_sustituido")
		}
		return nil
	}); err != nil {
		return err
	}
	actual, err = p.leerClon(revisar)
	if err != nil || actual == nil || actual.Marca != p.marca || actual.Propietario != owner {
		return falloPlantilla("propiedad_no_comprobable")
	}
	p.propiedad = actual
	p.pendiente = false
	if err = p.escribirLease(false); err != nil {
		if limpiarErr := p.retirar(revisar, clon); limpiarErr != nil {
			return limpiarErr
		}
		return err
	}
	return nil
}
func (p *RuntimePlantillas) RetirarClonPlantilla(ctx context.Context, clon string) error {
	if p == nil || clon != p.nombre {
		return falloPlantilla("clon_no_admitido")
	}
	desbloquear, err := p.runtime.escritorPlantillas()
	if err != nil {
		return err
	}
	defer desbloquear()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.retirar(ctx, clon)
}
func (p *RuntimePlantillas) retirar(ctx context.Context, clon string) error {
	if clon != p.nombre || p.propiedad == nil {
		return falloPlantilla("propiedad_no_comprobable")
	}
	if _, err := p.ObservarProcedenciaFisica(ctx); err != nil {
		return err
	}
	actual, err := p.leerClon(ctx)
	if err != nil {
		return err
	}
	if actual == nil {
		if err := p.confirmarOIDRetirado(ctx, p.propiedad.OID); err != nil {
			return err
		}
		p.propiedad = nil
		return p.escribirLease(false)
	}
	if *actual != *p.propiedad || actual.Marca != p.marca {
		return falloPlantilla("clon_sustituido")
	}
	dropErr := p.tecnicoValidado(ctx, fmt.Sprintf(`DROP DATABASE "%s"`, p.nombre), func(ctx context.Context) error {
		// El hook puede haber cambiado la preimagen. Comparar después de él, bajo
		// la fase exclusiva y antes del único efecto sobre ese nombre.
		actual, err := p.leerClon(ctx)
		if err != nil {
			return err
		}
		if actual == nil || *actual != *p.propiedad || actual.Marca != p.marca {
			return falloPlantilla("clon_sustituido")
		}
		return nil
	})
	revisar, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	despues, err := p.leerClon(revisar)
	if err != nil {
		return err
	}
	if despues != nil {
		if dropErr != nil {
			return dropErr
		}
		return falloPlantilla("retirada_no_comprobable")
	}
	if err := p.confirmarOIDRetirado(revisar, p.propiedad.OID); err != nil {
		return err
	}
	p.propiedad = nil
	p.pendiente = false
	return p.escribirLease(false)
}

func oidPlantillaValido(s string) bool {
	n, err := strconv.ParseUint(s, 10, 32)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
}
func (p *RuntimePlantillas) confirmarOIDRetirado(ctx context.Context, oid string) error {
	if !oidPlantillaValido(oid) {
		return falloPlantilla("oid_no_comprobable")
	}
	sql := "SELECT count(*) FROM pg_catalog.pg_database WHERE oid=" + oid + "::oid"
	b, err := p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-v", "ON_ERROR_STOP=1", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", sql}, nil, 4096)
	if err != nil || strings.TrimSpace(string(b)) != "0" {
		return falloPlantilla("clon_renombrado")
	}
	return nil
}

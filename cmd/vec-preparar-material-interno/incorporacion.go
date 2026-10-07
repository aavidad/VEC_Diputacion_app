package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

const errConfiguracionIncorporacion = errorPropio("incorporacion-config: contenido rechazado por el cargador B2 puro")
const errMotivosIncorporacion = errorPropio("motivos B2: alguna referencia no está publicada y vigente en la autoridad nominal")

func (p preparacion) prepararIncorporacion(ctx context.Context) (bool, error) {
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
	b, ok := leerFicheroPrivado(p.incorporacionConfig, 64<<10)
	if !ok {
		return false, errConfiguracionIncorporacion
	}
	duplicadas := clavesDuplicadas(b)
	clear(b)
	if duplicadas {
		return false, errConfiguracionIncorporacion
	}
	c, raiz, err := bootstrap.LeerConfiguracionPreparacionIncorporacionB2(p.incorporacionConfig)
	if err != nil {
		return false, errConfiguracionIncorporacion
	}
	defer raiz.Close()
	if err := comprobarIdempotencia(p.idempotencia); err != nil {
		return false, err
	}
	claves, err := bootstrap.DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(p.idempotencia, p.dep.reloj())
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	if err != nil {
		return false, errIdempotencia
	}
	if len(claves) != 22 {
		return false, errDerivacion
	}
	dsn, err := leerDSN(p.dsnArchivo, p.dsnEntorno)
	if err != nil {
		return false, err
	}
	g, err := p.dep.abrirGobierno(ctx, dsn)
	if err != nil {
		if errors.Is(err, errIdentidad) {
			return false, errIdentidad
		}
		return false, errGobiernoConexion
	}
	defer g.cerrar()
	for i := range claves {
		x := &claves[i]
		op, ok := c.PersonalB2.Operaciones[x.Capacidad]
		if !ok || x.EmisorID != ct.emisor {
			return false, errEmisorDistinto
		}
		f, e := g.clavePorHuellaSecreto(ctx, x.SHA256)
		if e != nil || f.ClaveID != x.ClaveID || f.Audiencia != x.Audiencia || f.HuellaGobierno != x.HuellaGobierno ||
			f.EmisorID != x.EmisorID || !f.Desde.Equal(x.Desde) || !f.Hasta.Equal(x.Hasta) || f.Version == 0 || f.Revision == 0 ||
			!f.ActoPropio || !f.Vigente || f.Revocada || !f.PunteroVigente {
			return false, errorPropio("clave B2 " + x.Capacidad + ": esperado=publicación vigente coincidente observado=ausente o distinta")
		}
		op.Capacidad.ClaveID, op.Capacidad.Version, op.Capacidad.RevisionGobierno = f.ClaveID, f.Version, f.Revision
		op.Capacidad.File, op.Capacidad.SHA256, op.Capacidad.HuellaGobierno = x.Capacidad+".cap", x.SHA256, f.HuellaGobierno
		op.Capacidad.EmisorID, op.Capacidad.Desde, op.Capacidad.Hasta = f.EmisorID, f.Desde, f.Hasta
		c.PersonalB2.Operaciones[x.Capacidad] = op
	}
	r, err := g.raizVigente(ctx)
	if err != nil || !r.Vigente || r.ClaveID != ct.raizID || r.Version != ct.raizVersion || r.Audiencia != ct.audiencia {
		return false, errRaiz
	}
	validarMotivos := p.dep.validarMotivosIncorporacion
	if validarMotivos == nil {
		validarMotivos = bootstrap.ValidarMotivosPreparacionIncorporacionB2
	}
	if err := validarMotivos(ctx, c, raiz, p.dep.reloj()); err != nil {
		var motivo *bootstrap.ErrorMotivoPreparacionIncorporacionB2
		if errors.As(err, &motivo) && motivo != nil {
			return false, errorPropio(motivo.Error())
		}
		return false, errMotivosIncorporacion
	}
	resolverDetalle := p.dep.resolverMotivoDetalle
	if resolverDetalle == nil {
		resolverDetalle = resolverMotivoDetalleCT
	}
	motivoDetalle, err := resolverDetalle(ctx, p.inventarioCT, p.dep.reloj())
	if err != nil || c.PersonalB2.Operaciones["ct_detalle"].Motivo != motivoDetalle {
		return false, errorPropio("motivo B2 ct_detalle: esperado=resolutor RRHH vigente observado=distinto o no disponible")
	}
	temporal, rt, err := destino.crearTemporal()
	if err != nil {
		return false, err
	}
	activado := false
	defer func() {
		_ = rt.Close()
		if !activado {
			destino.retirar(temporal)
		}
	}()
	// Copiar solo las ocho rutas nominales del contrato. Los nombres de salida
	// proceden de claves cerradas, no de rutas aportadas por la configuración.
	for _, grupo := range []struct {
		prefijo string
		rutas   map[string]string
	}{{"base-", c.Pools}, {"b2-", c.PersonalB2.Pools}} {
		for nombre, ruta := range grupo.rutas {
			contenido, err := leerRelativoIncorporacion(raiz, ruta)
			if err != nil {
				return false, errConfiguracionIncorporacion
			}
			archivo := grupo.prefijo + nombre + ".dsn"
			err = escribirPrivado(rt, archivo, contenido)
			clear(contenido)
			if err != nil {
				return false, err
			}
			grupo.rutas[nombre] = archivo
		}
	}
	for i := range claves {
		secreto := claves[i].CopiarSecreto()
		err := escribirPrivado(rt, claves[i].Capacidad+".cap", secreto)
		clear(secreto)
		if err != nil {
			return false, err
		}
	}
	b, err = json.Marshal(c)
	if err != nil {
		return false, errConfiguracionIncorporacion
	}
	err = escribirPrivado(rt, "servidor.json", append(b, '\n'))
	clear(b)
	if err != nil {
		return false, err
	}
	ruta, ok := destino.rutaTemporal(temporal, rt)
	if !ok {
		return false, errActivacion
	}
	if bootstrap.ValidarMaterialPreparadoIncorporacionB2(filepath.Join(ruta, "servidor.json"), p.dep.reloj()) != nil {
		return false, errValidacion
	}
	if ctx.Err() != nil {
		return false, errCancelada
	}
	if p.dep.antesDeActivar != nil {
		if p.dep.antesDeActivar() != nil {
			return false, errCancelada
		}
	}
	if err := sincronizarDirectorio(rt); err != nil {
		return false, err
	}
	sync, err := destino.activar(temporal, p.dep.sincronizarPadre)
	if err != nil {
		return false, err
	}
	activado = true
	return sync, nil
}

// El resolutor existente reacredita el LOGIN y hace SELECT en su transacción
// SERIALIZABLE READ WRITE. No crea motivos, perfiles ni concesiones.
func resolverMotivoDetalleCT(ctx context.Context, ruta string, ahora time.Time) (core.ReferenciaEntradaCatalogo, error) {
	m, err := internactproveedores.CargarMaterial(filepath.Dir(ruta))
	if err != nil {
		return core.ReferenciaEntradaCatalogo{}, errInventarioCT
	}
	defer m.Cerrar()
	d := m.Pools["motivos_rrhh"]
	p, err := pgct.NuevoPoolResolucionMotivosRRHHPostgreSQL(ctx, d.DSN, d.Login)
	if err != nil {
		return core.ReferenciaEntradaCatalogo{}, errMotivosIncorporacion
	}
	defer p.Cerrar()
	r, err := pgct.NuevoResolutorMotivoConsultaRRHHPostgreSQL(p)
	if err != nil {
		return core.ReferenciaEntradaCatalogo{}, errMotivosIncorporacion
	}
	return r.ResolverMotivoDetalleRRHH(ctx, ahora.UTC().Truncate(time.Microsecond))
}

func leerRelativoIncorporacion(r *os.Root, nombre string) ([]byte, error) {
	if !filepath.IsLocal(nombre) {
		return nil, errConfiguracionIncorporacion
	}
	i, err := r.Lstat(nombre)
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !delUsuario(i) || i.Size() < 1 || i.Size() > 16<<10 {
		return nil, errConfiguracionIncorporacion
	}
	f, err := r.Open(nombre)
	if err != nil {
		return nil, errConfiguracionIncorporacion
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, errConfiguracionIncorporacion
	}
	return b, nil
}

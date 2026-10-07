package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	core "vec-diputacion-granada/internal/vec/domain"
)

const inventarioOHNombre = "organizacion_historica_v3.json"
const claveOHNombre = "organizacion_historica_consulta.hmac"
const errInventarioOH = errorPropio("organizacion_historica.config: esperado=inventario privado nominal admitido observado=no disponible o distinto")
const errClaveOH = errorPropio("organizacion_historica.clave: esperado=gobierno propio vigente coincidente observado=no disponible o distinto")
const errMotivoOH = errorPropio("organizacion_historica.motivo: esperado=motivo publicado vigente observado=no disponible o distinto")

type inventarioPreparadoOH struct {
	Version         int                                                    `json:"version"`
	CatalogoMotivos string                                                 `json:"catalogo_motivos"`
	MotivoConsulta  core.ReferenciaEntradaCatalogo                         `json:"motivo_consulta"`
	Capacidad       capacidadInventario                                    `json:"capacidad"`
	Contextos       map[string]internagobierno.AmbitoOrganizacionHistorica `json:"contextos"`
}

func (p preparacion) prepararOrganizacionHistorica(ctx context.Context, entrada string) (bool, error) {
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
	if !rutaCanonica(entrada) || filepath.Base(entrada) != inventarioOHNombre {
		return false, errInventarioOH
	}
	if i, err := os.Lstat(entrada); err != nil || !i.Mode().IsRegular() || !delUsuario(i) {
		return false, errInventarioOH
	}
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(filepath.Dir(entrada))
	if err != nil {
		return false, errInventarioOH
	}
	defer m.Cerrar()
	if m.CatalogoMotivos != ct.catalogo {
		return false, errMotivoOH
	}
	if err := comprobarIdempotencia(p.idempotencia); err != nil {
		return false, err
	}
	clave, err := bootstrap.DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(p.idempotencia, p.dep.reloj())
	defer clave.Borrar()
	if err != nil {
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
	capacidad, err := cotejarClaveOH(ctx, g, ct, clave)
	if err != nil {
		return false, err
	}
	validar := p.dep.validarMotivoOH
	if validar == nil {
		validar = validarMotivoOrganizacionHistorica
	}
	if err := validar(ctx, p.inventarioCT, m.MotivoConsulta, p.dep.reloj()); err != nil {
		return false, errMotivoOH
	}
	// Los contextos se conservan sin inferir asignaciones ni sustituir F1.
	// El GET existente comprueba perfil/versión/organismo/unidad vigentes.
	contextos := make(map[string]internagobierno.AmbitoOrganizacionHistorica, len(m.Contextos))
	for cuenta, ambito := range m.Contextos {
		contextos[cuenta] = ambito
	}
	preparado := inventarioPreparadoOH{m.Version, m.CatalogoMotivos, m.MotivoConsulta, capacidad, contextos}
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
	secreto := clave.CopiarSecreto()
	err = escribirPrivado(rt, claveOHNombre, secreto)
	clear(secreto)
	if err != nil {
		return false, err
	}
	b, err := json.Marshal(preparado)
	if err != nil {
		return false, errInventarioOH
	}
	err = escribirPrivado(rt, inventarioOHNombre, append(b, '\n'))
	clear(b)
	if err != nil {
		return false, err
	}
	ruta, ok := destino.rutaTemporal(temporal, rt)
	if !ok {
		return false, errActivacion
	}
	if err := validarInventarioOHPreparado(ruta, preparado, clave.Audiencia, p.dep.reloj()); err != nil {
		return false, err
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

func cotejarClaveOH(ctx context.Context, g fuenteGobierno, ct datosCT, c bootstrap.ClaveCapacidadOrganizacionHistoricaV3) (capacidadInventario, error) {
	var vacia capacidadInventario
	if c.DescriptorCapacidadOrganizacionHistoricaV3 != bootstrap.DescriptorCapacidadOrganizacionHistoricaV3Desarrollo() || c.EmisorID != ct.emisor {
		return vacia, errClaveOH
	}
	f, err := g.clavePorHuellaSecreto(ctx, c.SHA256)
	if err != nil || f.ClaveID != c.ClaveID || f.Audiencia != c.Audiencia || f.HuellaGobierno != c.HuellaGobierno || f.EmisorID != c.EmisorID || !f.Desde.Equal(c.Desde) || !f.Hasta.Equal(c.Hasta) || f.Version == 0 || f.Revision == 0 || !f.ActoPropio || !f.Vigente || f.Revocada || !f.PunteroVigente {
		return vacia, errClaveOH
	}
	r, err := g.raizVigente(ctx)
	if err != nil || !r.Vigente || r.ClaveID != ct.raizID || r.Version != ct.raizVersion || r.Audiencia != ct.audiencia {
		return vacia, errRaiz
	}
	return capacidadInventario{ClaveID: f.ClaveID, Version: f.Version, Archivo: claveOHNombre, SHA256: c.SHA256, EmisorID: f.EmisorID, Desde: f.Desde, Hasta: f.Hasta, RevisionGobierno: f.Revision, HuellaGobierno: f.HuellaGobierno}, nil
}

func validarInventarioOHPreparado(dir string, c inventarioPreparadoOH, audiencia string, ahora time.Time) error {
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(dir)
	if err != nil {
		return errValidacion
	}
	defer m.Cerrar()
	if m.Version != c.Version || m.CatalogoMotivos != c.CatalogoMotivos || m.MotivoConsulta != c.MotivoConsulta || len(m.Contextos) != len(c.Contextos) {
		return errValidacion
	}
	for cuenta, x := range c.Contextos {
		if m.Contextos[cuenta] != x {
			return errValidacion
		}
	}
	if m.Capacidad.ClaveID != c.Capacidad.ClaveID || m.Capacidad.Version != c.Capacidad.Version || m.Capacidad.Archivo != c.Capacidad.Archivo || m.Capacidad.SHA256 != c.Capacidad.SHA256 || m.Capacidad.EmisorID != c.Capacidad.EmisorID || m.Capacidad.RevisionGobierno != c.Capacidad.RevisionGobierno || m.Capacidad.HuellaGobierno != c.Capacidad.HuellaGobierno || !m.Capacidad.Desde.Equal(c.Capacidad.Desde) || !m.Capacidad.Hasta.Equal(c.Capacidad.Hasta) {
		return errValidacion
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return errValidacion
	}
	defer r.Close()
	return comprobarCapacidad(r, c.Capacidad, audiencia, ahora)
}

func validarMotivoOrganizacionHistorica(ctx context.Context, inventarioCT string, motivo core.ReferenciaEntradaCatalogo, ahora time.Time) error {
	m, err := internactproveedores.CargarMaterial(filepath.Dir(inventarioCT))
	if err != nil {
		return errMotivoOH
	}
	defer m.Cerrar()
	d := m.Pools["motivos_autorizacion"]
	return bootstrap.ValidarMotivoPreparacionOrganizacionHistorica(ctx, d.DSN, d.Login, motivo, ahora)
}

package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type fuenteOfertaCTPrueba struct {
	instantanea core.InstantaneaAutorizacion
	err         error
}

func (f *fuenteOfertaCTPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (core.InstantaneaAutorizacion, error) {
	return f.instantanea, f.err
}

func configuracionOfertaCTPrueba(dir string) config.Config {
	return config.Config{DevelopmentMaterialDir: dir,
		ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode:         config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
}

func archivoOfertaCTPrueba() archivoActivacionOfertaCT {
	perfil := func(nombre, huella, motivo string) perfilActivacionOfertaCT {
		return perfilActivacionOfertaCT{PerfilRef: "prf:ofertas:ct:" + nombre,
			Rol: referenciaRolOfertaCTDesarrollo{RolID: "ofertas_ct_" + nombre, Version: 1, HuellaSHA256: strings.Repeat(huella, 64)},
			Motivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_oferta_ct_" + nombre, CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat(motivo, 64), EntradaClave: "motivo_oferta_ct_" + nombre}}
	}
	return archivoActivacionOfertaCT{
		Esquema: esquemaActivacionOfertaCTDesarrollo,
		Vinculo: perfil("vinculo", "a", "b"), CierreBolsa: perfil("cierre", "c", "d"),
		DescarteCT:                 perfil("descarte", "e", "f"),
		FuenteAutorizacionDSNFile:  "identidad/fuente-ofertas-ct.dsn",
		MotivosAutorizacionDSNFile: "identidad/motivos-ofertas-ct.dsn",
	}
}

func TestActivacionOfertaCTPrivadaOpcionalYCerrada(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := configuracionOfertaCTPrueba(dir)
	if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); presente || err != nil {
		t.Fatalf("archivo ausente no era opcional: presente=%v err=%v", presente, err)
	}
	base := archivoOfertaCTPrueba()
	if base.Vinculo.Motivo.Validar() != nil || base.CierreBolsa.Motivo.Validar() != nil || base.DescarteCT.Motivo.Validar() != nil {
		t.Fatal("motivo de prueba inválido")
	}
	ruta := filepath.Join(dir, archivoActivacionOfertaCTDesarrollo)
	escribir := func(b []byte, modo os.FileMode) {
		t.Helper()
		if err := os.WriteFile(ruta, b, modo); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, modo); err != nil {
			t.Fatal(err)
		}
	}
	canon, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	escribir(canon, 0600)
	actual, presente, err := leerActivacionOfertaCTDesarrollo(cfg)
	if err != nil || !presente || actual != base {
		t.Fatalf("activación privada válida rechazada: presente=%v err=%v", presente, err)
	}
	for _, caso := range []struct {
		nombre string
		b      []byte
		modo   os.FileMode
	}{
		{"mundo legible", canon, 0644},
		{"campo desconocido", append(append([]byte(nil), canon[:len(canon)-1]...), []byte(`,"actor_ref":"ajeno"}`)...), 0600},
		{"campo duplicado", append(append([]byte(nil), canon[:len(canon)-1]...), []byte(`,"perfil_ref":"prf:otro"}`)...), 0600},
		{"texto posterior", append(append([]byte(nil), canon...), []byte(`{}`)...), 0600},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escribir(caso.b, caso.modo)
			if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); !presente || err == nil {
				t.Fatalf("configuración no cerrada: presente=%v err=%v", presente, err)
			}
		})
	}
	base.FuenteAutorizacionDSNFile = "../ajeno.dsn"
	b, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	escribir(b, 0600)
	if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); !presente || err == nil {
		t.Fatal("ruta privada fuera del árbol admitida")
	}
	base = archivoOfertaCTPrueba()
	base.DescarteCT.PerfilRef = base.Vinculo.PerfilRef
	b, err = json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	escribir(b, 0600)
	if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); !presente || err == nil {
		t.Fatal("mismo perfil admitido para dos efectos")
	}
	base = archivoOfertaCTPrueba()
	base.CierreBolsa.Rol.RolID = base.Vinculo.Rol.RolID
	b, err = json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	escribir(b, 0600)
	if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); !presente || err == nil {
		t.Fatal("mismo rol admitido para dos efectos")
	}
	base = archivoOfertaCTPrueba()
	base.DescarteCT.Motivo = base.CierreBolsa.Motivo
	b, err = json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	escribir(b, 0600)
	if _, presente, err := leerActivacionOfertaCTDesarrollo(cfg); !presente || err == nil {
		t.Fatal("mismo motivo admitido para dos efectos")
	}
}

func TestSeleccionOfertaCTExigeConcesionYAudienciaPropias(t *testing.T) {
	c := core.ConcesionRol{Accion: ports.AccionVincularOfertaBolsaCT,
		ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoVinculoOfertaBolsaCT,
		Finalidades: []string{ports.FinalidadVincularOfertaBolsaCT}, GarantiaMinima: core.AuthAssuranceHigh}
	if !concesionExactaSeleccionOfertaCTDesarrollo(c) {
		t.Fatal("concesión exacta rechazada")
	}
	c.Accion = ports.AccionVincularEmisionBolsa
	if concesionExactaSeleccionOfertaCTDesarrollo(c) {
		t.Fatal("concesión singular aceptada para oferta")
	}
	d := descriptorMaterialSeleccionOfertaCTDesarrollo()
	if d.Audiencia != ports.AudienciaVincularOfertaBolsaCT ||
		d.Dominio != "vec.ct.vinculo-oferta-bolsa.capacidad-v3" ||
		d.Prefijo != "clave:capacidad:ct:vinculo-oferta-bolsa:" ||
		!audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		t.Fatal("audiencia de oferta no nominal o no gobernada")
	}
	for _, adicional := range []descriptorMaterialConsumidorV3Desarrollo{
		descriptorMaterialCierreIntencionOfertaCTDesarrollo(),
		descriptorMaterialDescarteSeleccionOfertaCTDesarrollo(),
	} {
		if adicional.Audiencia == d.Audiencia || adicional.Dominio == d.Dominio || adicional.Prefijo == d.Prefijo ||
			!audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(adicional.Audiencia) {
			t.Fatal("cierre y descarte no tienen material propio")
		}
	}
	for _, politica := range []politicaPerfilOfertaCTDesarrollo{
		politicaCierreOfertaBolsaCTDesarrollo("unidad:bolsa:001", "ambito:bolsa:001"),
		politicaDescarteOfertaCTDesarrollo(),
	} {
		concesion := core.ConcesionRol{Accion: politica.accion, ModuloID: politica.modulo,
			TipoRecurso: politica.tipo, Finalidades: []string{politica.finalidad},
			GarantiaMinima: core.AuthAssuranceHigh}
		if !concesionExactaPerfilOfertaCTDesarrollo(concesion, politica) {
			t.Fatalf("concesión propia %s rechazada", politica.accion)
		}
		concesion.Accion = ports.AccionVincularOfertaBolsaCT
		if concesionExactaPerfilOfertaCTDesarrollo(concesion, politica) {
			t.Fatalf("concesión de vínculo admitida para %s", politica.accion)
		}
	}
}

func TestFuenteOfertaCTSoloLeeRolExactoVigente(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC)
	principal, perfil := "principal:ofertas:ct:001", "prf:ofertas:ct:001"
	concesion := core.ConcesionRol{Accion: ports.AccionVincularOfertaBolsaCT,
		ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoVinculoOfertaBolsaCT,
		Finalidades: []string{ports.FinalidadVincularOfertaBolsaCT}, GarantiaMinima: core.AuthAssuranceHigh}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
		"ofertas_ct_rrhh", "Ofertas CT RRHH", "asignacion_ofertas_ct",
		[]core.ConcesionRol{concesion}, []core.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: []string{"centro:ct:001"}},
			{Clave: "categoria_ref", Valores: []string{"categoria:ct:001"}},
		})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	origen := &fuenteOfertaCTPrueba{instantanea: instantanea}
	f := &fuenteRolSeleccionOfertaCTDesarrollo{fuente: origen, principalID: principal,
		personaRef: principal, perfilRef: perfil,
		politica: politicaVinculoOfertaCTDesarrollo(),
		rol: referenciaRolOfertaCTDesarrollo{RolID: instantanea.VersionRol.RolID,
			Version: instantanea.VersionRol.Version, HuellaSHA256: huella},
		reloj: relojContratacionTemporalDesarrollo{}}
	if _, err := f.ObtenerInstantaneaAutorizacion(context.Background(), principal, perfil); err != nil {
		t.Fatalf("perfil publicado exacto rechazado: %v", err)
	}
	if _, err := f.ObtenerInstantaneaAutorizacion(context.Background(), principal, "prf:otro:001"); err == nil {
		t.Fatal("otro perfil admitido")
	}
	f.rol.HuellaSHA256 = strings.Repeat("c", 64)
	if _, err := f.ObtenerInstantaneaAutorizacion(context.Background(), principal, perfil); err == nil {
		t.Fatal("versión de rol divergente admitida")
	}
	f.rol.HuellaSHA256 = huella
	origen.instantanea.VersionRol.Concesiones[0].Accion = ports.AccionVincularEmisionBolsa
	if _, err := f.ObtenerInstantaneaAutorizacion(context.Background(), principal, perfil); err == nil {
		t.Fatal("concesión singular admitida")
	}
	origen.err = vp.ErrFuenteAutorizacionNoDisponible
	if _, err := f.ObtenerInstantaneaAutorizacion(context.Background(), principal, perfil); !errors.Is(err, vp.ErrFuenteAutorizacionNoDisponible) {
		t.Fatal("caída de la fuente confundida con falta de permiso")
	}
}

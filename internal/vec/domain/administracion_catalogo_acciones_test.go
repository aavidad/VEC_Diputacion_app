package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func escenarioCatalogoAccionesAdministracion(t *testing.T) (CatalogoAccionesAdministracionV1, PropuestaPerfilAdministracionV1, time.Time) {
	t.Helper()
	desde := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	instante := desde.Add(24 * time.Hour)
	concesion := ConcesionRol{Accion: "sintetico.consultar", ModuloID: "sintetico", TipoRecurso: "expediente", Finalidades: []string{"revision"},
		GarantiaMinima: AuthAssuranceHigh, CamposPermitidos: []string{"estado"}, Obligaciones: []string{"auditar"}}
	rol := VersionRol{RolID: "revision_sintetica", Version: 1, Nombre: "revision_sintetica", Estado: EstadoVersionRolPublicada, Concesiones: []ConcesionRol{concesion}, PublicadaPor: "actor:fuente", PublicadaEn: desde}
	c := CatalogoAccionesAdministracionV1{Referencia: "catalogo:sintetico", Version: 1, FuenteRef: "fuente:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), VigenteDesde: desde, VigenteHasta: desde.Add(72 * time.Hour),
		Entradas: []EntradaAccionAdministracionV1{{Referencia: "accion:sintetica", Version: 1, FuenteRef: "fuente:modulo", FuenteVersion: 2, FuenteHuellaSHA256: strings.Repeat("b", 64), Concesion: concesion,
			DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada", VigenteDesde: desde, VigenteHasta: desde.Add(48 * time.Hour)}},
		Perfiles: []PerfilPublicadoAdministracionV1{{Rol: rol, TipoPerfil: TipoPerfilAdministracionAdministrableV1, ControlVigencia: ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "actor:fuente", ActualizadoEn: desde}}}}
	propuesto := rol
	propuesto.Version, propuesto.PublicadaEn = 2, instante
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	he, err := c.Entradas[0].HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hr, err := rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p := PropuestaPerfilAdministracionV1{CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: hc, VersionRolBaseRef: rol.Referencia(), VersionRolBaseHuellaSHA256: hr,
		RolPropuesto: propuesto, Selecciones: []SeleccionAccionAdministracionV1{{EntradaRef: c.Entradas[0].Referencia, EntradaVersion: 1, EntradaHuellaSHA256: he}}}
	// La propuesta y el catálogo no comparten slices mutables en la prueba.
	datos, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var copia PropuestaPerfilAdministracionV1
	if json.Unmarshal(datos, &copia) != nil {
		t.Fatal("clone")
	}
	return c, copia, instante
}

func TestComprobacionPerfilAdministracionConservaOrigenCompleto(t *testing.T) {
	c, p, instante := escenarioCatalogoAccionesAdministracion(t)
	d, err := ComprobarPropuestaPerfilAdministracionV1(c, p, instante)
	if err != nil || d.VersionRolRef != p.RolPropuesto.Referencia() || d.CatalogoHuellaSHA256 != p.CatalogoHuellaSHA256 || d.PropuestaHuellaSHA256 == "" {
		t.Fatalf("dictamen: %+v %v", d, err)
	}
	// Otro rol nuevo solo se admite como versión inicial sin una base ficticia.
	p.RolPropuesto.RolID, p.RolPropuesto.Version = "nuevo_sintetico", 1
	p.VersionRolBaseRef, p.VersionRolBaseHuellaSHA256 = "", ""
	if _, err := ComprobarPropuestaPerfilAdministracionV1(c, p, instante); err != nil {
		t.Fatal(err)
	}
}

func TestComprobacionPerfilAdministracionRechazaManipulaciones(t *testing.T) {
	casos := []struct {
		nombre   string
		mutar    func(*CatalogoAccionesAdministracionV1, *PropuestaPerfilAdministracionV1, *time.Time)
		esperado error
	}{
		{"accion", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].Accion = "sintetico.modificar"
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"modulo", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].ModuloID = "otro"
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"recurso", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].TipoRecurso = "persona"
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"finalidad", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].Finalidades = []string{"otra"}
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"campos", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].CamposPermitidos = append(p.RolPropuesto.Concesiones[0].CamposPermitidos, "dato_personal")
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"garantia", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].GarantiaMinima = AuthAssuranceLow
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"obligacion", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].Obligaciones = nil
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"huella catalogo", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.CatalogoHuellaSHA256 = strings.Repeat("e", 64)
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"huella entrada", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.Selecciones[0].EntradaHuellaSHA256 = strings.Repeat("e", 64)
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"huella base", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.VersionRolBaseHuellaSHA256 = strings.Repeat("e", 64)
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"version catalogo", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.CatalogoVersion++
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"referencia catalogo", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.CatalogoRef = "catalogo:otro"
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"referencia entrada", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.Selecciones[0].EntradaRef = "accion:otra"
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"referencia base", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.VersionRolBaseRef = "rol:otro:v1"
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"version entrada", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.Selecciones[0].EntradaVersion++
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"salto version", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Version++
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"caducado", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			*instante = c.VigenteHasta
		}, ErrCatalogoAccionesAdministracionNoVigente},
		{"entrada caducada", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			*instante = c.Entradas[0].VigenteHasta
		}, ErrPermisoPerfilAdministracionNoCoincide},
		{"catalogo futuro", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			*instante = c.VigenteDesde.Add(-time.Microsecond)
		}, ErrCatalogoAccionesAdministracionNoVigente},
		{"precision no canonica", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			*instante = instante.Add(time.Nanosecond)
		}, ErrCatalogoAccionesAdministracionNoVigente},
		{"publicacion futura", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			p.RolPropuesto.PublicadaEn = instante.Add(time.Microsecond)
		}, ErrPropuestaPerfilAdministracionInvalida},
		{"base futura", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			c.Perfiles[0].ControlVigencia.ActualizadoEn = instante.Add(time.Microsecond)
			p.CatalogoHuellaSHA256, _ = c.HuellaSHA256()
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"fijo nueva version", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Perfiles[0].TipoPerfil = TipoPerfilAdministracionFijoSistemaV1
			p.CatalogoHuellaSHA256, _ = c.HuellaSHA256()
		}, ErrPerfilAdministracionFijo},
		{"fijo falsa alta", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Perfiles[0].TipoPerfil = TipoPerfilAdministracionFijoSistemaV1
			p.CatalogoHuellaSHA256, _ = c.HuellaSHA256()
			p.RolPropuesto.Version = 1
			p.VersionRolBaseRef = ""
			p.VersionRolBaseHuellaSHA256 = ""
		}, ErrPerfilAdministracionFijo},
		{"tipo ausente", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Perfiles[0].TipoPerfil = ""
		}, ErrCatalogoAccionesAdministracionInvalido},
		{"rol retirado", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, instante *time.Time) {
			c.Perfiles[0].ControlVigencia.Estado = EstadoControlVigenciaVersionRolRetirada
			c.Perfiles[0].ControlVigencia.ActoRef = "acto:retirada"
			c.Perfiles[0].ControlVigencia.MotivoCodigo = "retirada"
			p.CatalogoHuellaSHA256, _ = c.HuellaSHA256()
		}, ErrOrigenPerfilAdministracionNoCoincide},
		{"comodin", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].Accion = "sintetico.*"
		}, ErrPropuestaPerfilAdministracionInvalida},
		{"lista duplicada", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.RolPropuesto.Concesiones[0].CamposPermitidos = []string{"estado", "estado"}
		}, ErrPropuestaPerfilAdministracionInvalida},
		{"lista excesiva", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			p.Selecciones = make([]SeleccionAccionAdministracionV1, 513)
		}, ErrPropuestaPerfilAdministracionInvalida},
		{"duplicado catalogo", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Entradas = append(c.Entradas, c.Entradas[0])
		}, ErrCatalogoAccionesAdministracionInvalido},
		{"dimension ausente", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Entradas[0].DimensionesAmbito = nil
		}, ErrCatalogoAccionesAdministracionInvalido},
		{"dimension global", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Entradas[0].DimensionesAmbito = []string{"global"}
		}, ErrCatalogoAccionesAdministracionInvalido},
		{"dimension incompatible con asignacion", func(c *CatalogoAccionesAdministracionV1, p *PropuestaPerfilAdministracionV1, _ *time.Time) {
			c.Entradas[0].DimensionesAmbito = []string{strings.Repeat("a", 129)}
		}, ErrCatalogoAccionesAdministracionInvalido},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, p, instante := escenarioCatalogoAccionesAdministracion(t)
			caso.mutar(&c, &p, &instante)
			d, err := ComprobarPropuestaPerfilAdministracionV1(c, p, instante)
			if !errors.Is(err, caso.esperado) || d != (DictamenPerfilAdministracionV1{}) {
				t.Fatalf("dictamen=%+v error=%v esperado=%v", d, err, caso.esperado)
			}
		})
	}
}

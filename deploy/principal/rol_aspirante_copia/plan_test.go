package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

var ahoraPrueba = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func leerFuente(t *testing.T, nombre string) (fuentePaquete, json.RawMessage) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "fuentes", nombre))
	if err != nil {
		t.Fatal(err)
	}
	var f fuentePaquete
	if err := decodificarExacto(b, &f); err != nil {
		t.Fatal(err)
	}
	return f, b
}

// rolBase reproduce la v1 publicada del aspirante: siete concesiones propias.
func rolBase() dominio.PerfilPublicadoAdministracionV1 {
	rol := dominio.VersionRol{RolID: dominio.RolIDVersionarBolsaExterna, Version: 1,
		Nombre: "areaPersonal.miBolsa.rolPortal", Estado: dominio.EstadoVersionRolPublicada,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: ahoraPrueba.Add(-48 * time.Hour)}
	for _, accion := range []string{"bolsa.participaciones_propias.consultar", "bolsa.historial_propio.consultar",
		"bolsa.participaciones_propias.solicitar_pausa", "bolsa.participaciones_propias.solicitar_reactivacion",
		"bolsa.participaciones_propias.responder_llamamiento", "bolsa.participaciones_propias.manifestar_disposicion",
		"bolsa.participaciones_propias.confirmar_contacto"} {
		recurso, finalidad := "participaciones_candidato", "gestion_participaciones_propias"
		var campos []string
		switch accion {
		case "bolsa.historial_propio.consultar":
			finalidad = "consulta_historial_propio"
			campos = []string{"contratos_propios", "llamamientos_propios", "renuncias_propias"}
		case "bolsa.participaciones_propias.consultar":
			finalidad = "consulta_participaciones_propias"
			campos = []string{"participaciones_candidato_minimizadas"}
		case "bolsa.participaciones_propias.manifestar_disposicion":
			recurso = "oferta_bolsa"
		}
		rol.Concesiones = append(rol.Concesiones, dominio.ConcesionRol{Accion: accion, ModuloID: "bolsa",
			TipoRecurso: recurso, Finalidades: []string{finalidad}, GarantiaMinima: dominio.AuthAssuranceHigh,
			CamposPermitidos: campos})
	}
	return dominio.PerfilPublicadoAdministracionV1{Rol: rol, TipoPerfil: dominio.TipoPerfilAdministracionAdministrableV1,
		ControlVigencia: dominio.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: dominio.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "per_fuente",
			ActualizadoEn: ahoraPrueba.Add(-48 * time.Hour)}}
}

// cabezaV6 publica un catálogo con las fuentes v1 de inscripción y
// preferencias y el censo de un solo rol, como la cabeza de la copia.
func cabezaV6(t *testing.T, perfil dominio.PerfilPublicadoAdministracionV1) (string, string, string) {
	t.Helper()
	inscripcion, _ := leerFuente(t, "../inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json")
	preferencias, _ := leerFuente(t, "bolsa_preferencias_bolsas_propias_v1.json")
	p := paquete{Esquema: esquemaPaquete, Referencia: "paquete:prueba", Version: 6,
		Fuentes:  []fuentePaquete{preferencias, inscripcion},
		Perfiles: []dominio.PerfilPublicadoAdministracionV1{perfil}}
	canonPaquete, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	c := dominio.CatalogoAccionesAdministracionV1{Referencia: "catalogo:prueba", Version: 6,
		FuenteRef: p.Referencia, FuenteVersion: 6, FuenteHuellaSHA256: huellaBytes(canonPaquete),
		VigenteDesde: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		VigenteHasta: time.Date(2027, 11, 1, 0, 0, 0, 0, time.UTC), Perfiles: p.Perfiles}
	for _, f := range p.Fuentes {
		c.Entradas = append(c.Entradas, f.Entradas...)
	}
	canon, err := json.Marshal(c)
	if err != nil || c.Validar() != nil {
		t.Fatalf("catálogo de prueba inválido: %v", err)
	}
	return string(canonPaquete), string(canon), huellaBytes(canon)
}

func perfilCrudo(t *testing.T, p dominio.PerfilPublicadoAdministracionV1) perfilActual {
	t.Helper()
	rol, err1 := json.Marshal(p.Rol)
	control, err2 := json.Marshal(p.ControlVigencia)
	if err1 != nil || err2 != nil {
		t.Fatal("perfil no serializable")
	}
	return perfilActual{Rol: rol, Control: control}
}

func entradaCatalogoPrueba(t *testing.T, perfil dominio.PerfilPublicadoAdministracionV1) entradaCatalogo {
	t.Helper()
	paqueteCanon, canon, huella := cabezaV6(t, perfil)
	_, inscripcion := leerFuente(t, "inscripcion_bolsa_actos_v2.json")
	_, preferencias := leerFuente(t, "bolsa_preferencias_bolsas_propias_v2.json")
	return entradaCatalogo{PaqueteCanon: paqueteCanon, CatalogoCanon: canon, CatalogoSHA256: huella,
		Fuentes: []json.RawMessage{inscripcion, preferencias}, Perfiles: []perfilActual{perfilCrudo(t, perfil)},
		OperacionRef: "caa_" + strings.Repeat("a", 32), AprobacionRef: "aprobacion:prueba",
		AprobacionSHA256: strings.Repeat("b", 64), Entorno: "desarrollo",
		PreparadoEn: ahoraPrueba, CaducaEn: ahoraPrueba.Add(22 * time.Hour)}
}

func asignacionPara(t *testing.T, rol dominio.VersionRol, version int) asignacionActual {
	t.Helper()
	a := dominio.AsignacionPerfil{AsignacionID: "candidato_0123456789abcdef", Version: version,
		PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef", PrincipalID: "per_0123456789abcdef0123456789abcdef",
		VersionRolRef: rol.Referencia(), Estado: dominio.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominio.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{"can_abcdefghijklmnopqrstuv"}}},
		VigenteDesde: ahoraPrueba.Add(-72 * time.Hour), VigenteHasta: ahoraPrueba.Add(180 * 24 * time.Hour),
		EmitidaPor: "desarrollo:portal", EmitidaEn: ahoraPrueba.Add(-72 * time.Hour)}
	h, err := a.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := json.Marshal(a)
	return asignacionActual{AsignacionRef: a.Referencia(), HuellaSHA256: h, Documento: doc}
}

func propuestaPara(t *testing.T, modo, catalogo string, base dominio.PerfilPublicadoAdministracionV1, asignacion int) (salidaPropuesta, error) {
	t.Helper()
	crudo := perfilCrudo(t, base)
	hr, _ := base.Rol.HuellaSHA256()
	hc, _ := base.ControlVigencia.HuellaSHA256()
	return prepararPropuesta(entradaPropuesta{Modo: modo, OperacionRef: "propuesta_admin:" + strings.Repeat("c", 32),
		CatalogoCanon: catalogo, Rol: crudo.Rol, RolHuellaSHA256: hr, Control: crudo.Control, ControlHuella: hc,
		Asignaciones: []asignacionActual{asignacionPara(t, base.Rol, asignacion)},
		Motivo: dominio.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("e", 32)},
		Ahora: ahoraPrueba})
}

func TestCatalogoSustituyeFuentesYConservaCenso(t *testing.T) {
	base := rolBase()
	s, err := prepararCatalogo(entradaCatalogoPrueba(t, base))
	if err != nil {
		t.Fatal(err)
	}
	var plan planAdmision
	if err := decodificarExacto([]byte(s.Plan), &plan); err != nil || huellaBytes([]byte(s.Plan)) != s.PlanSHA256 {
		t.Fatalf("plan ilegible o huella distinta: %v", err)
	}
	if plan.CatalogoVersion != "7" || plan.EsperadoVersion != "6" || plan.PaqueteVersion != "7" ||
		s.Entradas != 10 || s.Perfiles != 1 || len(s.Sustituidas) != 2 {
		t.Fatalf("catálogo inesperado: %+v", s)
	}
	var c dominio.CatalogoAccionesAdministracionV1
	if err := decodificarExacto([]byte(plan.CatalogoCanon), &c); err != nil {
		t.Fatal(err)
	}
	// Las fuentes v2 ocupan el sitio de las v1, en el mismo orden.
	if c.Entradas[0].Referencia != "accion:bolsa.preferencias_bolsas.consultar_propias" || c.Entradas[0].Version != 2 ||
		c.Entradas[2].Referencia != "accion:bolsa.inscripcion.convocatorias.listar" || c.Entradas[2].Version != 2 {
		t.Fatal("las fuentes nuevas no sustituyen a las anteriores en su sitio")
	}
}

func TestCatalogoRechazaFuenteSinAnteriorYCensoDistinto(t *testing.T) {
	base := rolBase()
	e := entradaCatalogoPrueba(t, base)
	_, otra := leerFuente(t, "bolsa_alegaciones_propias_v1.json")
	e.Fuentes = append(e.Fuentes, otra)
	if _, err := prepararCatalogo(e); err == nil {
		t.Fatal("una fuente que no sustituye a otra debe rechazarse")
	}
	e = entradaCatalogoPrueba(t, base)
	e.Perfiles = nil
	if _, err := prepararCatalogo(e); err == nil {
		t.Fatal("un censo distinto exige decidir el tipo del rol")
	}
	e = entradaCatalogoPrueba(t, base)
	e.CatalogoSHA256 = strings.Repeat("0", 64)
	if _, err := prepararCatalogo(e); err == nil {
		t.Fatal("una cabeza con otra huella debe rechazarse")
	}
}

func TestPropuestasLlevanElRolDeV1aV3(t *testing.T) {
	base := rolBase()
	s, err := prepararCatalogo(entradaCatalogoPrueba(t, base))
	if err != nil {
		t.Fatal(err)
	}
	var plan planAdmision
	_ = json.Unmarshal([]byte(s.Plan), &plan)
	if _, err := propuestaPara(t, "preferencias", plan.CatalogoCanon, base, 1); err != nil {
		// El dominio también admite preferencias primero; aquí solo se exige que no falle por el catálogo.
		t.Fatalf("preferencias sobre la v1: %v", err)
	}
	primera, err := propuestaPara(t, "inscripcion", plan.CatalogoCanon, base, 1)
	if err != nil || primera.Concesiones != 12 || len(primera.Cuerpo.Selecciones) != 5 ||
		primera.VersionRolObjetivo != "rol:"+dominio.RolIDVersionarBolsaExterna+":v2" || primera.Cuerpo.HuellaPlanEsperada == "" {
		t.Fatalf("propuesta de inscripción inesperada: %+v %v", primera, err)
	}
	// La v2 publicada por el cierre: mismas concesiones que el plan.
	var c dominio.CatalogoAccionesAdministracionV1
	_ = json.Unmarshal([]byte(plan.CatalogoCanon), &c)
	q := primera.Cuerpo
	intencion := dominio.IntencionVersionarRolBolsaExterna{CatalogoRef: q.CatalogoRef, CatalogoVersion: q.CatalogoVersion,
		CatalogoHuellaSHA256: q.CatalogoHuellaSHA256, BaseRef: q.BaseRef, BaseHuellaSHA256: q.BaseHuellaSHA256,
		ControlRevision: q.ControlRevision, ControlHuellaSHA256: q.ControlHuellaSHA256,
		Selecciones: q.Selecciones, Asignaciones: q.Asignaciones, Motivo: q.Motivo}
	completo, err := dominio.PrepararPlanVersionarRolBolsaExterna(c, base, intencion, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	v2 := base
	v2.Rol.Version, v2.Rol.Concesiones = 2, completo.DefinicionNueva.Concesiones
	v2.Rol.PublicadaEn = ahoraPrueba.Add(-time.Minute)
	v2.ControlVigencia.VersionRolRef, v2.ControlVigencia.ActualizadoEn = v2.Rol.Referencia(), ahoraPrueba.Add(-time.Minute)
	if _, err := propuestaPara(t, "inscripcion", plan.CatalogoCanon, v2, 2); err == nil {
		t.Fatal("la inscripción no puede repetirse sobre la v2")
	}
	segunda, err := propuestaPara(t, "preferencias", plan.CatalogoCanon, v2, 2)
	if err != nil || segunda.Concesiones != 14 || len(segunda.Cuerpo.Selecciones) != 2 ||
		segunda.VersionRolObjetivo != "rol:"+dominio.RolIDVersionarBolsaExterna+":v3" {
		t.Fatalf("propuesta de preferencias inesperada: %+v %v", segunda, err)
	}
	// Un control posterior al instante de la propuesta se rechaza.
	v2.ControlVigencia.ActualizadoEn = ahoraPrueba.Add(time.Second)
	if _, err := propuestaPara(t, "preferencias", plan.CatalogoCanon, v2, 2); err == nil {
		t.Fatal("el instante de la propuesta no puede ser anterior al control")
	}
}

func TestEntradaEstrictaYOrdenDesconocida(t *testing.T) {
	if _, err := ejecutar("otra", strings.NewReader("{}")); err == nil {
		t.Fatal("orden desconocida aceptada")
	}
	if _, err := ejecutar("propuesta", strings.NewReader(`{"modo":"inscripcion","sobrante":1}`)); err == nil {
		t.Fatal("campo desconocido aceptado")
	}
	if _, err := ejecutar("propuesta", strings.NewReader(`{"modo":"todo"}`)); err == nil {
		t.Fatal("modo que mezcla actos aceptado")
	}
}

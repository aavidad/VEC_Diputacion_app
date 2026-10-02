package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type fuenteInstantaneaLecturaPrueba struct {
	dato            domain.InstantaneaAutorizacion
	err             error
	llamadas        int
	persona, perfil string
}

func (f *fuenteInstantaneaLecturaPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, persona, perfil string) (domain.InstantaneaAutorizacion, error) {
	f.llamadas++
	f.persona = persona
	f.perfil = perfil
	return f.dato, f.err
}

type emisorLecturaPrueba struct {
	t        *testing.T
	ahora    time.Time
	efecto   Efecto
	llamadas int
}

func (e *emisorLecturaPrueba) EmitirAdministracionPerfiles(_ context.Context, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles, _ domain.InstantaneaAutorizacion, ef Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	if v.ValidarPara(a) != nil {
		e.t.Fatal("evidencia perdida")
	}
	e.efecto = ef
	e.efecto.Material = append([]byte(nil), ef.Material...)
	return materialSintetico(e.t, ef, a, e.ahora), nil
}

type txLecturaPrueba struct {
	*txFalsa
	consulta   string
	argumentos []any
}

func (t *txLecturaPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	t.consultas++
	t.consulta = consulta
	t.argumentos = append([]any(nil), args...)
	return t.fila
}

type poolLecturaPrueba struct {
	tx        *txLecturaPrueba
	opciones  pgx.TxOptions
	comienzos int
}

func (p *poolLecturaPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	p.opciones = o
	return p.tx, nil
}
func (p *poolLecturaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("lectura directa fuera de fachada")
}

func escenarioLectura(t *testing.T, b []byte) (*FuenteLecturas, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, *fuenteInstantaneaLecturaPrueba, *emisorLecturaPrueba, *poolLecturaPrueba) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	rol := domain.VersionRol{RolID: "admin_perfiles", Version: 1, Nombre: "Administrador", Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{{Accion: "administracion.perfiles.capacidades", ModuloID: "vec", TipoRecurso: "persona", Finalidades: []string{"administracion_perfiles"}, GarantiaMinima: domain.AuthAssuranceHigh}}, PublicadaPor: "responsable", PublicadaEn: ahora.Add(-24 * time.Hour)}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	i := domain.InstantaneaAutorizacion{AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "admin_lecturas", Version: 1, PrincipalID: r.Contexto.PersonaRef, PerfilActivoRef: r.Contexto.PerfilActivoRef, VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "responsable", EmitidaEn: ahora.Add(-2 * time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	if i.Validar() != nil {
		t.Fatal("instantanea de prueba invalida")
	}
	fuente := &fuenteInstantaneaLecturaPrueba{dato: i}
	emisor := &emisorLecturaPrueba{t: t, ahora: ahora}
	pool := &poolLecturaPrueba{tx: &txLecturaPrueba{txFalsa: &txFalsa{fila: filaFalsa{dato: b}}}}
	f, err := NuevaFuenteLecturas(&Autoridad{pool: pool, emisor: emisor, reloj: relojFijo(ahora)}, fuente)
	if err != nil {
		t.Fatal(err)
	}
	return f, r.Contexto, domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: r, Vinculo: v}, fuente, emisor, pool
}

func TestLecturaConsumeV3NominalYFiltrosSinSQLDirecto(t *testing.T) {
	f, a, v, fuente, e, p := escenarioLectura(t, []byte(`{"personas":[],"siguiente_cursor":"cursor-siguiente"}`))
	x, err := f.BuscarPersonas(context.Background(), a, v, "Ana", "cursor-anterior")
	if err != nil {
		t.Fatal(err)
	}
	if x.Personas == nil || fuente.llamadas != 1 || fuente.persona != a.PersonaRef || fuente.perfil != a.PerfilActivoRef || e.llamadas != 1 || p.comienzos != 1 || p.tx.commits != 1 || p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("lectura no autorizada/atomica")
	}
	if p.tx.consulta != personasLecturaSQL || len(p.tx.argumentos) != 11 || e.efecto.Accion != "administracion.perfiles.consultar" {
		t.Fatal("fachada no nominal")
	}
	var material lecturaJSON
	if json.Unmarshal(e.efecto.Material, &material) != nil || material.Busqueda != "Ana" || material.Cursor != "cursor-anterior" || material.ActorPersonaRef != a.PersonaRef || material.ActorPerfilRef != a.PerfilActivoRef || material.Limite != 50 {
		t.Fatal("material no ligado")
	}
}

func TestLecturaRechazaEvidenciaAjenaAntesDeFuente(t *testing.T) {
	f, a, v, fuente, e, p := escenarioLectura(t, []byte(`{"roles":[]}`))
	a.PersonaRef = "per_" + strings.Repeat("z", 22)
	_, err := f.ListarRoles(context.Background(), a, v)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || fuente.llamadas != 0 || e.llamadas != 0 || p.comienzos != 0 {
		t.Fatal("evidencia ajena alcanzo lectura")
	}
}

func TestLecturaRechazaInstantaneaAjenaORevocada(t *testing.T) {
	for _, caso := range []string{"persona", "perfil", "caducada", "rol_retirado"} {
		t.Run(caso, func(t *testing.T) {
			f, a, v, fuente, e, p := escenarioLectura(t, []byte(`{"roles":[]}`))
			switch caso {
			case "persona":
				fuente.dato.AsignacionPerfil.PrincipalID = "per_" + strings.Repeat("c", 22)
			case "perfil":
				fuente.dato.AsignacionPerfil.PerfilActivoRef = "prf_" + strings.Repeat("c", 22)
			case "caducada":
				fuente.dato.AsignacionPerfil.VigenteHasta = e.ahora
			case "rol_retirado":
				fuente.dato.ControlVigenciaVersionRol.Estado = domain.EstadoControlVigenciaVersionRolRetirada
				fuente.dato.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
				fuente.dato.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
			}
			_, err := f.ListarRoles(context.Background(), a, v)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || e.llamadas != 0 || p.comienzos != 0 {
				t.Fatalf("instantanea ajena: %v", err)
			}
		})
	}
}

func TestLecturaNoDevuelveDatosAntesDeCommitNiRespuestaAjena(t *testing.T) {
	for _, caso := range []string{"identidad", "campo_extra", "commit"} {
		t.Run(caso, func(t *testing.T) {
			b := []byte(`{"version":"1","actor_persona_ref":"per_` + strings.Repeat("a", 22) + `","acciones":[]}`)
			if caso == "identidad" {
				b = []byte(`{"version":"1","actor_persona_ref":"per_` + strings.Repeat("c", 22) + `","acciones":[]}`)
			}
			if caso == "campo_extra" {
				b = []byte(`{"version":"1","actor_persona_ref":"per_` + strings.Repeat("a", 22) + `","acciones":[],"dato_privado":"no sale"}`)
			}
			f, a, v, _, _, p := escenarioLectura(t, b)
			if caso == "commit" {
				p.tx.falloCommit = errors.New("credencial privada")
			}
			x, err := f.Capacidades(context.Background(), a, v)
			if err == nil || x.ActorPersonaRef != "" || x.Acciones != nil {
				t.Fatal("respuesta parcial expuesta")
			}
			if caso != "commit" && p.tx.commits != 0 {
				t.Fatal("respuesta ajena confirmada")
			}
			if strings.Contains(err.Error(), "privada") {
				t.Fatal("error no sanitizado")
			}
		})
	}
}

func TestLecturasLimitesYNulos(t *testing.T) {
	var f *FuenteLecturas
	if _, err := f.BuscarPersonas(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, "x", ""); err == nil {
		t.Fatal("busqueda demasiado corta")
	}
	if _, err := f.ConsultarPersona(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, "per_invalida"); err == nil {
		t.Fatal("referencia invalida")
	}
	if _, err := NuevaFuenteLecturas(nil, nil); err == nil {
		t.Fatal("constructor abierto")
	}
	for _, b := range []string{`{"personas":null}`, `{"personas":[],"siguiente_cursor":"repetido"}`} {
		f, a, v, _, _, p := escenarioLectura(t, []byte(b))
		if _, err := f.BuscarPersonas(context.Background(), a, v, "Ana", "repetido"); err == nil || p.tx.commits != 0 {
			t.Fatal("pagina incoherente confirmada")
		}
	}
}

func TestFichaNoCalculaHuellaNiAceptaOtroObjetivo(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	persona := "per_" + strings.Repeat("a", 22)
	objetivo := api.Objetivo{CuentaRef: "cta_" + strings.Repeat("c", 22), CuentaVersion: 1, PersonaRef: persona, PersonaVersion: 1, PerfilRef: "prf_" + strings.Repeat("d", 22), VinculoRef: "vca_" + strings.Repeat("e", 22), HuellaSHA256: strings.Repeat("f", 64), ProcedenciaRef: "procedencia:nominal", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("b", 64), VigenteHasta: ahora.Add(time.Hour)}
	motivos := []api.MotivoLectura{{Motivo: api.Motivo{CatalogoID: "admin.motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "alta"}, Etiqueta: "Alta"}}
	x := api.FichaPersona{PersonaRef: persona, Nombre: "Ana", Perfiles: []api.Perfil{}, Historia: []api.Historia{}, ActosDisponibles: []api.ActoDisponible{{Operacion: "otorgar", RolVersionRef: "rol:cronos_rrhh:v1", Objetivo: objetivo, Motivos: motivos}}}
	if !validarFichaLectura(x, persona, ahora) {
		t.Fatal("ficha valida rechazada")
	}
	if x.ActosDisponibles[0].Objetivo.HuellaSHA256 != strings.Repeat("f", 64) {
		t.Fatal("huella sustituida")
	}
	x.ActosDisponibles[0].Objetivo.PersonaRef = "per_" + strings.Repeat("z", 22)
	if validarFichaLectura(x, persona, ahora) {
		t.Fatal("objetivo de otra persona aceptado")
	}
}

func TestSieteLecturasUsanSuFachadaYAccionExistente(t *testing.T) {
	persona := "per_" + strings.Repeat("a", 22)
	propuesta := "propuesta_admin:" + strings.Repeat("b", 32)
	recibo := "recibo_admin:" + strings.Repeat("c", 32)
	propuestaJSON := `{"propuesta_ref":"` + propuesta + `","proponente_persona_ref":"per_` + strings.Repeat("d", 22) + `","objetivo_persona_ref":"` + persona + `","objetivo_nombre":"Ana","rol_version_ref":"rol:cronos_rrhh:v1","operacion":"otorgar","huella_sha256":"` + strings.Repeat("e", 64) + `","caduca_en":"2026-10-03T12:00:00Z","puede_cerrar":false,"motivos_cierre":[]}`
	reciboJSON := `{"operacion_ref":"acto_admin:` + strings.Repeat("d", 32) + `","acto_ref":"acto_admin:` + strings.Repeat("d", 32) + `","recibo_ref":"` + recibo + `","auditoria_ref":"auditoria:nominal","objetivo_persona_ref":"` + persona + `","perfil_ref":"prf_` + strings.Repeat("f", 22) + `","vinculo_ref":"vca_` + strings.Repeat("e", 22) + `","estado_posterior":"activo","version_posterior":1,"huella_antes_sha256":"` + strings.Repeat("a", 64) + `","huella_despues_sha256":"` + strings.Repeat("b", 64) + `","confirmado_en":"2026-10-02T12:00:00Z"}`
	for _, caso := range []struct {
		nombre, bruto, sql, accion string
		llamada                    func(*FuenteLecturas, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) error
	}{
		{"capacidades", `{"version":"1","actor_persona_ref":"` + persona + `","acciones":[]}`, capacidadesLecturaSQL, "administracion.perfiles.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.Capacidades(context.Background(), a, v)
			return err
		}},
		{"buscar_personas", `{"personas":[]}`, personasLecturaSQL, "administracion.perfiles.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.BuscarPersonas(context.Background(), a, v, "Ana", "")
			return err
		}},
		{"consultar_persona", `{"persona_ref":"` + persona + `","nombre":"Ana","unidad_nombre":"","perfiles":[],"actos_disponibles":[],"historia":[]}`, personaLecturaSQL, "administracion.perfiles.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.ConsultarPersona(context.Background(), a, v, persona)
			return err
		}},
		{"listar_roles", `{"roles":[]}`, rolesLecturaSQL, "administracion.perfiles.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.ListarRoles(context.Background(), a, v)
			return err
		}},
		{"listar_propuestas", `{"propuestas":[]}`, propuestasLecturaSQL, "administracion.perfiles.historial.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.ListarPropuestas(context.Background(), a, v)
			return err
		}},
		{"consultar_propuesta", propuestaJSON, propuestaLecturaSQL, "administracion.perfiles.historial.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.ConsultarPropuesta(context.Background(), a, v, propuesta)
			return err
		}},
		{"consultar_recibo", reciboJSON, reciboLecturaSQL, "administracion.perfiles.recibo.consultar", func(f *FuenteLecturas, a domain.ContextoActor, v domain.EvidenciaSesionAdministracionPerfiles) error {
			_, err := f.ConsultarRecibo(context.Background(), a, v, recibo)
			return err
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f, a, v, _, e, p := escenarioLectura(t, []byte(caso.bruto))
			if err := caso.llamada(f, a, v); err != nil {
				t.Fatal(err)
			}
			if p.tx.consulta != caso.sql || e.efecto.Accion != caso.accion || e.efecto.Audiencia != "vec_autorizacion.administracion_perfiles.lectura."+caso.nombre+".v1" || p.tx.commits != 1 {
				t.Fatal("lectura no nominal")
			}
		})
	}
}
